#!/bin/bash
# Injects branch name and uncommitted file count at session start.
# Output is added as context for the agent.

branch=$(git -C "${WHODB_ROOT:-$(git rev-parse --show-toplevel)}" rev-parse --abbrev-ref HEAD 2>/dev/null)
dirty=$(git -C "${WHODB_ROOT:-$(git rev-parse --show-toplevel)}" status --porcelain 2>/dev/null | wc -l | tr -d ' ')

echo "Branch: ${branch:-unknown}, Uncommitted files: ${dirty}"
