#!/usr/bin/env bash
#
# Copyright 2026 Clidey, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

set -euo pipefail

ROOT_DIR="$(cd -- "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

FAILURES=0

ok() {
  printf 'OK: %s\n' "$1"
}

fail() {
  printf 'FAIL: %s\n' "$1" >&2
  FAILURES=$((FAILURES + 1))
}

require_command() {
  if command -v "$1" >/dev/null 2>&1; then
    ok "found command: $1"
  else
    fail "missing required command: $1"
  fi
}

require_file() {
  if [ -f "$1" ]; then
    ok "required file exists: $1"
  else
    fail "required file missing: $1"
  fi
}

require_dir() {
  if [ -d "$1" ]; then
    ok "required directory exists: $1"
  else
    fail "required directory missing: $1"
  fi
}

require_file_contains() {
  local file="$1"
  local pattern="$2"
  local description="$3"

  if [ ! -f "$file" ]; then
    fail "$description; file missing: $file"
    return
  fi

  if grep -Eq "$pattern" "$file"; then
    ok "$description"
  else
    fail "$description"
  fi
}

# .claude/rules and .claude/skills hold only symlinks into .agents/ (Claude Code
# reads rules and skills there); real files belong in .agents/.
check_legacy_dir_has_no_files() {
  local dir="$1"

  if [ ! -d "$dir" ]; then
    ok "legacy directory absent: $dir"
    return
  fi

  if find "$dir" -type f | grep -q .; then
    fail "legacy directory contains files: $dir"
  else
    ok "legacy directory has no files: $dir"
  fi
}

check_stale_paths() {
  local output
  local pattern

  pattern='(^|[^~[:alnum:]_/.-])((\./)?\.claude/docs|(\./)?docs/BILLING\.md)'

  # Only this repository's files (tracked, plus untracked and not ignored);
  # submodule contents (mode 160000 entries) are checked by their own
  # repositories.
  if output="$({ git ls-files -s | awk '$1 != "160000" { sub(/^[^\t]*\t/, ""); print }'; git ls-files --others --exclude-standard; } \
    | grep -vE '(^|/)(node_modules|dist|build|coverage)/|^frontend/src/generated/|(^|/)\.agents/skills/|^dev/check-agent-instructions\.sh$|^\.gitignore$' \
    | tr '\n' '\0' \
    | xargs -0 rg --hidden --line-number --color never --no-messages "$pattern" 2>/dev/null)"; then
    printf '%s\n' "$output" >&2
    fail "stale legacy agent paths found"
  else
    ok "no stale legacy agent paths found"
  fi
}

check_trailing_whitespace() {
  local output
  local targets

  targets=(
    AGENTS.md
    CLAUDE.md
    .agents
    dev/check-agent-instructions.sh
  )

  if output="$(rg --hidden --line-number --color never '[[:blank:]]+$' "${targets[@]}" \
    --glob '!**/.agents/skills/**' 2>/dev/null)"; then
    printf '%s\n' "$output" >&2
    fail "trailing whitespace found in agent instruction files"
  else
    ok "no trailing whitespace in agent instruction files"
  fi
}

require_command rg

require_file AGENTS.md
require_file CLAUDE.md
require_file .agents/README.md
require_dir .agents/rules
require_dir .agents/skills
require_file .agents/skills/task-handoff/SKILL.md
require_file .agents/skills/research-proof/SKILL.md
require_file .agents/skills/review-checklist/SKILL.md

require_file_contains AGENTS.md '\.agents/README\.md' 'AGENTS.md links the shared agent index'
require_file_contains .agents/README.md '\.agents/rules/' '.agents/README.md links agent rules'
require_file_contains .agents/README.md '\.agents/skills/' '.agents/README.md links agent skills'
require_file_contains .agents/README.md '\.agents/skills/task-handoff/SKILL\.md' '.agents/README.md links task handoff skill'
require_file_contains .agents/README.md '\.agents/skills/research-proof/SKILL\.md' '.agents/README.md links research proof skill'

check_legacy_dir_has_no_files .claude/docs
check_legacy_dir_has_no_files .claude/rules
check_legacy_dir_has_no_files .claude/skills

check_stale_paths
check_trailing_whitespace

# Every repository path named in agent guidance must exist.
if python3 dev/check-agent-doc-paths.py --root . --exclude-skill impeccable AGENTS.md CLAUDE.md .agents; then
  ok "agent guidance names only existing paths"
else
  fail "agent guidance names paths that don't exist"
fi

if [ "$FAILURES" -eq 0 ]; then
  printf 'Agent instruction checks passed.\n'
else
  printf 'Agent instruction checks failed: %d\n' "$FAILURES" >&2
  exit 1
fi
