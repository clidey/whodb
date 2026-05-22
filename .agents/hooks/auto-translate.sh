#!/bin/bash
# After editing a locale YAML file, runs drift detection + translation.

hook_dir="$(cd -- "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(git rev-parse --show-toplevel 2>/dev/null)"
needs_translation=0

if [[ -z "$repo_root" ]]; then
    exit 0
fi

while IFS= read -r file_path; do
    if [[ "$file_path" == */locales/*.yaml ]] || [[ "$file_path" == */locales/*.yml ]]; then
        needs_translation=1
    fi
done < <(python3 "$hook_dir/changed-files.py")

if [[ "$needs_translation" -eq 1 ]]; then
    cd "$repo_root/dev/translate" && python3 detect.py 2>/dev/null && node translate.mjs 2>/dev/null
fi
