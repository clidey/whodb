#!/usr/bin/env bash
# pack_test.sh — consumption test for the BUILT ARTIFACT: builds the wheel
# exactly as the release job does, installs it into a clean venv, and imports
# the public surface. Catches packaging problems (missing modules, broken
# metadata) that source-tree tests cannot see.
set -euo pipefail

package_dir="$(cd "$(dirname "$0")/.." && pwd)"
scratch="$(mktemp -d)"
trap 'rm -rf "$scratch"' EXIT

python3 -m venv "$scratch/venv"
"$scratch/venv/bin/pip" install --quiet --upgrade build
(cd "$package_dir" && "$scratch/venv/bin/python" -m build --wheel --outdir "$scratch/dist" >/dev/null)
"$scratch/venv/bin/pip" install --quiet "$scratch"/dist/*.whl

"$scratch/venv/bin/python" - <<'PY'
import whodb
from whodb import WhoDB, AsyncWhoDB
from whodb._errors import WhoDBError, AuthError, WhoDBVersionError
from whodb._transport_ipc import IpcTransport

for symbol in (WhoDB, AsyncWhoDB, WhoDBError, AuthError, WhoDBVersionError, IpcTransport):
    assert callable(symbol), f"missing public symbol: {symbol}"
print("wheel import ok")
PY

echo "pack_test: PASS"
