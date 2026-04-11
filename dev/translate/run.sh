#!/usr/bin/env bash
#
# Translation pipeline: detect drift → translate → apply.
#
# Usage:
#   ./run.sh                       # all languages
#   ./run.sh fr_FR de_DE           # specific languages only
#
set -euo pipefail
cd "$(dirname "$0")"

# Forward locale args to both scripts
LOCALE_ARGS=""
if [ $# -gt 0 ]; then
    LOCALE_ARGS=$(IFS=,; echo "$*")
fi

echo "=== Step 1: Detecting translation drift ==="
echo ""
if [ -n "$LOCALE_ARGS" ]; then
    uv run python detect.py -l "$LOCALE_ARGS"
else
    uv run python detect.py
fi

echo ""
echo "=== Step 2: Translating and applying ==="
echo ""
if [ $# -gt 0 ]; then
    node translate.mjs "$@"
else
    node translate.mjs
fi
