#!/bin/bash
# After editing a .graphqls file, runs backend and frontend codegen.

hook_dir="$(cd -- "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(git rev-parse --show-toplevel 2>/dev/null)"
needs_ce=0
needs_ee=0

if [[ -z "$repo_root" ]]; then
    exit 0
fi

while IFS= read -r file_path; do
    if [[ "$file_path" != *.graphqls ]]; then
        continue
    fi

    if [[ "$file_path" == ee/* ]] || [[ "$file_path" == */ee/* ]]; then
        needs_ee=1
    else
        needs_ce=1
    fi
done < <(python3 "$hook_dir/changed-files.py")

if [[ "$needs_ce" -eq 1 ]]; then
    cd "$repo_root/core" && go generate . 2>/dev/null
    cd "$repo_root/frontend" && pnpm run generate 2>/dev/null
fi

if [[ "$needs_ee" -eq 1 ]]; then
    cd "$repo_root/ee" && go generate . 2>/dev/null
    cd "$repo_root/ee/frontend" && pnpm run generate 2>/dev/null
fi
