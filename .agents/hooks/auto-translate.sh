#!/bin/bash
# After editing a locale YAML file, runs drift detection + translation.
# Reads tool_input.file_path from stdin JSON.

file_path=$(cat | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('tool_input',{}).get('file_path',''))" 2>/dev/null)

if [[ -z "$file_path" ]]; then
    exit 0
fi

if [[ "$file_path" == */locales/*.yaml ]] || [[ "$file_path" == */locales/*.yml ]]; then
    repo_root=$(git rev-parse --show-toplevel 2>/dev/null)
    cd "$repo_root/dev/translate" && python3 detect.py 2>/dev/null && node translate.mjs 2>/dev/null
fi
