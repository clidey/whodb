#!/bin/bash
# Auto-formats Go files after edit/write/apply_patch hooks.

hook_dir="$(cd -- "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(git rev-parse --show-toplevel 2>/dev/null)"

resolve_file() {
    local file_path="$1"

    if [[ -f "$file_path" ]]; then
        printf '%s\n' "$file_path"
    elif [[ -n "$repo_root" ]] && [[ -f "$repo_root/$file_path" ]]; then
        printf '%s\n' "$repo_root/$file_path"
    fi
}

python3 "$hook_dir/changed-files.py" | while IFS= read -r file_path; do
    if [[ "$file_path" == *.go ]]; then
        resolved_path="$(resolve_file "$file_path")"
        if [[ -n "$resolved_path" ]]; then
            gofmt -w -- "$resolved_path" 2>/dev/null
        fi
    fi
done
