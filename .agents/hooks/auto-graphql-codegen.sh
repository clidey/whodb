#!/bin/bash
# After editing a .graphqls file, runs backend and frontend codegen.
# Reads tool_input.file_path from stdin JSON.

file_path=$(cat | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('tool_input',{}).get('file_path',''))" 2>/dev/null)

if [[ -z "$file_path" ]] || [[ "$file_path" != *.graphqls ]]; then
    exit 0
fi

repo_root=$(git rev-parse --show-toplevel 2>/dev/null)

if [[ "$file_path" == *ee/* ]]; then
    cd "$repo_root/ee" && go generate . 2>/dev/null
    cd "$repo_root/ee/frontend" && pnpm run generate 2>/dev/null
else
    cd "$repo_root/core" && go generate . 2>/dev/null
    cd "$repo_root/frontend" && pnpm run generate 2>/dev/null
fi
