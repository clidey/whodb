#!/bin/bash
# Auto-formats Go files after edit/write/apply_patch hooks.
# Runs gofmt, then golangci-lint auto-fixes on the edited core/ packages.

hook_dir="$(cd -- "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# WHODB_ROOT is set by the forwarding shims used when the session runs inside a
# submodule, where git rev-parse would resolve to the submodule instead.
repo_root="${WHODB_ROOT:-$(git rev-parse --show-toplevel 2>/dev/null)}"

resolve_file() {
	local file_path="$1"

	if [[ -f "$file_path" ]]; then
		printf '%s\n' "$file_path"
	elif [[ -n "$repo_root" ]] && [[ -f "$repo_root/$file_path" ]]; then
		printf '%s\n' "$repo_root/$file_path"
	fi
}

go_files=()
while IFS= read -r file_path; do
	if [[ "$file_path" == *.go ]]; then
		resolved_path="$(resolve_file "$file_path")"
		if [[ -n "$resolved_path" ]]; then
			go_files+=("$resolved_path")
		fi
	fi
done < <(python3 "$hook_dir/changed-files.py")

[[ ${#go_files[@]} -eq 0 ]] && exit 0

for go_file in "${go_files[@]}"; do
	gofmt -w -- "$go_file" >/dev/null 2>&1 || true
done

# The config loads the gounslop module plugin, which only the custom binary that
# core/lint.sh builds can run. Skip until it exists rather than stall the edit
# on a build.
lint_bin="${TMPDIR:-/tmp}/whodb-custom-gcl"
core_dir="$(cd "$repo_root/core" 2>/dev/null && pwd -P)" || exit 0
[[ -x "$lint_bin" ]] || exit 0

packages=()
for go_file in "${go_files[@]}"; do
	dir="$(cd "$(dirname "$go_file")" && pwd -P)"
	if [[ "$dir" == "$core_dir" || "$dir" == "$core_dir"/* ]]; then
		packages+=(".${dir#"$core_dir"}")
	fi
done

[[ ${#packages[@]} -eq 0 ]] && exit 0

# Exit 1 means findings remain after --fix; pre-commit and CI report those. Any
# other failure means the linter did not run, which the agent should see.
out="$(cd "$core_dir" && "$lint_bin" run --fix --allow-serial-runners --config .golangci.yml "${packages[@]}" 2>&1)"
status=$?
if [[ "$status" -gt 1 ]]; then
	{
		echo "golangci-lint could not run on the edited core/ packages (exit $status):"
		printf '%s\n' "$out" | tail -20
	} >&2
	exit 2
fi

exit 0
