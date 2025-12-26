#!/usr/bin/env bash
set -euo pipefail

# Unified backend test runner for CE, EE, and integration.
# - Uses a repo-local GOCACHE to avoid sandboxed cache permission issues
# - Runs CE unit tests (excludes interactive server_test)
# - Runs EE tests when the ee module is present
# - Runs live integration tests (docker-compose) by default; set MODE to limit
#   MODE values: all (default) | unit | integration

ROOT_DIR="$(cd -- "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
GOCACHE_DIR="$ROOT_DIR/core/.gocache"

export GOCACHE="$GOCACHE_DIR"
MODE="${1:-all}"

run_unit() {
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
}

run_integration() {
  (
    set -euo pipefail
    echo "→ Running integration backend tests (docker-compose services required)"
    COMPOSE_FILE="$ROOT_DIR/dev/docker-compose.e2e.yaml"
    MANAGE_COMPOSE="${WHODB_MANAGE_COMPOSE:-1}"
    COMPOSE_STARTED=0

    cleanup() {
      if [ "$MANAGE_COMPOSE" = "1" ] && [ "$COMPOSE_STARTED" -eq 1 ]; then
        echo "→ Tearing down integration docker-compose stack"
        docker compose -f "$COMPOSE_FILE" down --volumes --remove-orphans
      fi
    }
    trap cleanup EXIT

    if [ "$MANAGE_COMPOSE" = "1" ]; then
      if docker compose -f "$COMPOSE_FILE" ps -q | grep -q .; then
        echo "ℹ️  Reusing existing docker-compose stack"
      else
        echo "🐳 Starting integration docker-compose stack"
        docker compose -f "$COMPOSE_FILE" up -d
        COMPOSE_STARTED=1
      fi
    else
      echo "ℹ️  WHODB_MANAGE_COMPOSE=0, assuming services are already running"
    fi

    cd "$ROOT_DIR/core"
    # If we started compose ourselves, don't start again inside tests.
    START_FLAG="${WHODB_START_COMPOSE:-}"
    if [ "$COMPOSE_STARTED" -eq 1 ]; then
      START_FLAG="0"
    fi
    WHODB_START_COMPOSE="${START_FLAG:-0}" go test -tags integration ./test/integration/...
  )
}

case "$MODE" in
  all)
    run_unit
    run_integration
    ;;
  unit)
    run_unit
    ;;
  integration)
    run_integration
    ;;
  *)
    echo "Unknown MODE: $MODE"
    echo "Usage: $(basename "$0") [all|unit|integration]"
    exit 1
    ;;
esac
