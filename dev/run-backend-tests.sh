#!/usr/bin/env bash
set -euo pipefail

# Unified backend test runner for CE and EE.
# - Uses a repo-local GOCACHE to avoid sandboxed cache permission issues
# - Runs CE unit tests (excludes interactive server_test)
# - Optionally runs EE tests when the ee module is present

ROOT_DIR="$(cd -- "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
GOCACHE_DIR="$ROOT_DIR/core/.gocache"

export GOCACHE="$GOCACHE_DIR"

echo "→ Running CE backend tests"
(
  cd "$ROOT_DIR/core"
  go test ./src/... ./graph/...
)

if [ -d "$ROOT_DIR/ee" ]; then
  echo "→ Running EE backend tests (ee build tag)"
  (
    cd "$ROOT_DIR/ee"
    go test -tags ee ./core/...
  )
else
  echo "ℹ️ EE module not found; skipping EE backend tests"
fi
