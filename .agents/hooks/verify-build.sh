#!/usr/bin/env bash
# Stop hook: type-checks / compiles only the areas changed this turn and surfaces
# errors back to the agent. The auto-format/lint PostToolUse hooks silently
# auto-fix style but swallow compile and type errors; this catches those before
# the turn ends. Works for both Claude Code (.claude/settings.json) and Codex
# (.codex/hooks.json), and for CE plus the EE submodule.

set -uo pipefail

repo="$(git rev-parse --show-toplevel 2>/dev/null)" || exit 0
[[ -z "$repo" ]] && exit 0

payload="$(cat 2>/dev/null || true)"

# Loop guard: if the agent is already continuing because of a previous Stop-hook
# block, don't block again on this pass.
if printf '%s' "$payload" | grep -q '"stop_hook_active"[[:space:]]*:[[:space:]]*true'; then
	exit 0
fi

# git status for a working tree: tracked modifications + staged + untracked.
changed_in() {
	(cd "$1" 2>/dev/null && {
		git diff --name-only
		git diff --cached --name-only
		git ls-files --others --exclude-standard
	} 2>/dev/null)
}

main_changes="$(changed_in "$repo")"
ee_changes=""
[[ -d "$repo/ee" ]] && ee_changes="$(changed_in "$repo/ee")"

need_core_go=0 need_ce_ts=0 need_ee_go=0 need_ee_ts=0
grep -qE '^core/.*\.go$'          <<<"$main_changes" && need_core_go=1
grep -qE '^frontend/.*\.(ts|tsx)$' <<<"$main_changes" && need_ce_ts=1
grep -qE '\.go$'                  <<<"$ee_changes"   && need_ee_go=1
grep -qE '^frontend/.*\.(ts|tsx)$' <<<"$ee_changes"  && need_ee_ts=1

errors=""

run_check() {
	local label="$1" dir="$2"
	shift 2
	local out
	if ! out="$(cd "$repo/$dir" && "$@" 2>&1)"; then
		errors+="### ${label} failed (cd ${dir} && $*)"$'\n'
		errors+="$(printf '%s\n' "$out" | tail -40)"$'\n\n'
	fi
}

if [[ "$need_core_go" == 1 ]] && command -v go >/dev/null 2>&1; then
	run_check "CE Go build" core go build ./...
fi
if [[ "$need_ee_go" == 1 ]] && command -v go >/dev/null 2>&1; then
	run_check "EE Go build" ee go build ./...
fi
if [[ "$need_ce_ts" == 1 ]] && command -v pnpm >/dev/null 2>&1; then
	run_check "CE typecheck" frontend pnpm typecheck
fi
if [[ "$need_ee_ts" == 1 ]] && command -v pnpm >/dev/null 2>&1; then
	run_check "EE typecheck" ee/frontend pnpm typecheck
fi

[[ -z "$errors" ]] && exit 0

# Debounce so a genuinely unfixable error can't trap the agent in a loop on
# harnesses that don't send stop_hook_active: if the identical error set was
# reported on the immediately preceding run, let the agent stop.
fp_file="$repo/.git/.whodb-verify-build-fp"
fp="$(printf '%s' "$errors" | shasum 2>/dev/null | awk '{print $1}')"
if [[ -n "$fp" && -f "$fp_file" && "$(cat "$fp_file" 2>/dev/null)" == "$fp" ]]; then
	rm -f "$fp_file"
	exit 0
fi
[[ -n "$fp" ]] && printf '%s' "$fp" >"$fp_file"

{
	echo "Build/type-check errors in code changed this turn — fix before finishing:"
	echo
	printf '%s' "$errors"
} >&2
exit 2
