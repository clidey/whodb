#!/bin/bash
# After editing a .ts/.tsx file, runs eslint --fix on it.

hook_dir="$(cd -- "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(git rev-parse --show-toplevel 2>/dev/null)"

resolve_file() {
    local file_path="$1"

    if [[ -f "$file_path" ]]; then
        cd -- "$(dirname "$file_path")" && printf '%s/%s\n' "$(pwd -P)" "$(basename "$file_path")"
    elif [[ -n "$repo_root" ]] && [[ -f "$repo_root/$file_path" ]]; then
        cd -- "$(dirname "$repo_root/$file_path")" && printf '%s/%s\n' "$(pwd -P)" "$(basename "$file_path")"
    fi
}

python3 "$hook_dir/changed-files.py" | while IFS= read -r file_path; do
    if [[ "$file_path" == *.ts ]] || [[ "$file_path" == *.tsx ]]; then
        resolved_path="$(resolve_file "$file_path")"
        if [[ -z "$resolved_path" ]] || [[ -z "$repo_root" ]]; then
            continue
        fi

        if [[ "$resolved_path" == "$repo_root/ee/frontend/"* ]]; then
            (cd "$repo_root/ee/frontend" && pnpm exec eslint --fix -- "$resolved_path" 2>/dev/null)
        elif [[ "$resolved_path" == "$repo_root/frontend/"* ]]; then
            (cd "$repo_root/frontend" && pnpm exec eslint --fix -- "$resolved_path" 2>/dev/null)
        fi
    fi
done
