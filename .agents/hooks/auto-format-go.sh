#!/bin/bash
# Auto-formats Go files after edit/write.
# Reads tool_input.file_path from stdin JSON.

file_path=$(cat | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('tool_input',{}).get('file_path',''))" 2>/dev/null)

# Reject paths with newlines or empty paths (prevent injection)
if [[ -z "$file_path" ]] || [[ "$file_path" == *$'\n'* ]]; then
    exit 0
fi

if [[ "$file_path" == *.go ]] && [[ -f "$file_path" ]]; then
    gofmt -w -- "$file_path" 2>/dev/null
fi
