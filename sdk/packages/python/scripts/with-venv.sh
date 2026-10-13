#!/usr/bin/env bash
# with-venv.sh — runs a command inside the package's local .venv, creating it
# and installing the package (editable, with dev extras) on first use. Lets
# `pnpm test` / `pnpm lint` / the conformance runner work on machines whose
# system python3 lacks httpx or ruff. Honors an already-activated venv
# (VIRTUAL_ENV) and CI, which installs the package itself.
set -euo pipefail

package_dir="$(cd "$(dirname "$0")/.." && pwd)"

if [[ -n "${VIRTUAL_ENV:-}" || -n "${CI:-}" ]]; then
  exec "$@"
fi

venv="$package_dir/.venv"
stamp="$venv/.whodb-installed"
if [[ ! -x "$venv/bin/python" ]]; then
  python3 -m venv "$venv"
fi
if [[ ! -f "$stamp" || "$package_dir/pyproject.toml" -nt "$stamp" ]]; then
  "$venv/bin/pip" install --quiet --upgrade pip
  "$venv/bin/pip" install --quiet -e "$package_dir[dev]"
  touch "$stamp"
fi

export PATH="$venv/bin:$PATH"
exec "$@"
