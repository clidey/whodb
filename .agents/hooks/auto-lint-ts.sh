#!/bin/bash
# After editing a .ts/.tsx file, runs eslint --fix on it.
# Reads tool_input.file_path from stdin JSON.

file_path=$(cat | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('tool_input',{}).get('file_path',''))" 2>/dev/null)

if [[ -z "$file_path" ]] || [[ "$file_path" == *$'\n'* ]]; then
    exit 0
fi

if [[ "$file_path" == *.ts ]] || [[ "$file_path" == *.tsx ]]; then
    if [[ -f "$file_path" ]]; then
        repo_root=$(git rev-parse --show-toplevel 2>/dev/null)
        cd "$repo_root/frontend" && pnpm exec eslint --fix -- "$file_path" 2>/dev/null
    fi
fi
