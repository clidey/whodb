#!/usr/bin/env bash
# Install the pre-commit git hook.
# Run once per clone: bash dev/setup-hooks.sh
set -euo pipefail
REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cp "$REPO_ROOT/.agents/hooks/pre-commit" "$REPO_ROOT/.git/hooks/pre-commit"
chmod +x "$REPO_ROOT/.git/hooks/pre-commit"
echo "hooks: pre-commit installed"
