#!/bin/bash
# After editing a GraphQL schema (.graphqls) or frontend operation (.graphql)
# file, runs the backend and frontend codegen that read it. EE codegen merges the
# CE schema and CE operations, so CE edits regenerate EE too when it is present.
# Codegen failures are reported back to the agent.

hook_dir="$(cd -- "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# WHODB_ROOT is set by the forwarding shims used when the session runs inside a
# submodule, where git rev-parse would resolve to the submodule instead.
repo_root="${WHODB_ROOT:-$(git rev-parse --show-toplevel 2>/dev/null)}"
needs_ce_go=0
needs_ee_go=0
needs_ce_ts=0
needs_ee_ts=0

if [[ -z "$repo_root" ]]; then
    exit 0
fi
repo_root="$(cd -- "$repo_root" && pwd -P)"

while IFS= read -r file_path; do
    # Relative paths are relative to the session's directory, which may be ee/.
    rel_path="$file_path"
    if [[ -f "$file_path" ]]; then
        abs_path="$(cd -- "$(dirname "$file_path")" && pwd -P)/$(basename "$file_path")"
        rel_path="${abs_path#"$repo_root"/}"
    fi

    case "$rel_path" in
        ee/*.graphqls)
            needs_ee_go=1 needs_ee_ts=1 ;;
        ee/frontend/src/*.graphql)
            needs_ee_ts=1 ;;
        *.graphqls)
            needs_ce_go=1 needs_ce_ts=1
            [[ -d "$repo_root/ee" ]] && needs_ee_go=1 needs_ee_ts=1 ;;
        frontend/src/*.graphql)
            needs_ce_ts=1
            [[ -d "$repo_root/ee" ]] && needs_ee_ts=1 ;;
    esac
done < <(python3 "$hook_dir/changed-files.py")

errors=""

run_codegen() {
    local label="$1" dir="$2"
    shift 2
    local out
    if ! out="$(cd "$repo_root/$dir" && "$@" 2>&1)"; then
        errors+="### ${label} failed (cd ${dir} && $*)"$'\n'
        errors+="$(printf '%s\n' "$out" | tail -30)"$'\n\n'
    fi
}

# EE backend codegen binds to the CE models, so CE backend runs first.
[[ "$needs_ce_go" -eq 1 ]] && run_codegen "CE backend codegen" core go generate .
[[ "$needs_ee_go" -eq 1 ]] && run_codegen "EE backend codegen" ee go generate .
[[ "$needs_ce_ts" -eq 1 ]] && run_codegen "CE frontend codegen" frontend pnpm run generate
[[ "$needs_ee_ts" -eq 1 ]] && run_codegen "EE frontend codegen" ee/frontend pnpm run generate

[[ -z "$errors" ]] && exit 0

{
    echo "GraphQL codegen failed; generated code is stale until this is fixed:"
    echo
    printf '%s' "$errors"
} >&2
exit 2
