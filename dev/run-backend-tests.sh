#!/usr/bin/env bash
#
# Copyright 2026 Clidey, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

set -euo pipefail

# Unified backend test runner for CE, EE, and integration.
# - Uses a repo-local GOCACHE to avoid sandboxed cache permission issues
# - Runs CE and EE unit tests with explicit per-edition modes
# - Runs CE and EE integration suites against their own docker-compose stacks
# - MODE values:
#     all (default) | unit | ce-unit | ee-unit | integration |
#     ce-integration | ee-integration | ssl

ROOT_DIR="$(cd -- "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
GOCACHE_DIR="$ROOT_DIR/core/.gocache"
EE_COMPOSE_PROJECT="whodb-ee-tests"

export GOCACHE="$GOCACHE_DIR"
MODE="${1:-all}"

has_ee() {
	[ -f "$ROOT_DIR/ee/go.mod" ]
}

usage() {
	cat <<EOF
Usage: $(basename "$0") [MODE]

Modes:
  all             Run CE + EE unit tests, then CE + EE integration tests
  unit            Run CE + EE unit tests
  ce-unit         Run CE unit tests only
  ee-unit         Run EE unit tests only
  integration     Run CE + EE integration tests
  ce-integration  Run CE integration tests only
  race            Run CE unit tests with -race -count=10
  ee-integration  Run EE integration tests only
  ssl             Run CE SSL integration tests only
EOF
}

run_hermetic_go_test() {
	local workdir="$1"
	shift

	(
		set -euo pipefail

		local test_home
		local original_home
		test_home="$(mktemp -d "${TMPDIR:-/tmp}/whodb-backend-home.XXXXXX")"
		original_home="${HOME:-}"

		cleanup() {
			rm -rf "$test_home"
		}
		trap cleanup EXIT

		while IFS='=' read -r name _; do
			if [[ "$name" == WHODB_* ]]; then
				unset "$name"
			fi
		done < <(env)

		export HOME="$test_home"
		export XDG_DATA_HOME="$test_home/.local/share"
		export XDG_CONFIG_HOME="$test_home/.config"
		export XDG_CACHE_HOME="$test_home/.cache"

		mkdir -p "$XDG_DATA_HOME" "$XDG_CONFIG_HOME" "$XDG_CACHE_HOME"

		if [ -n "$original_home" ] && [ -d "$original_home/Library/Caches/baml" ]; then
			mkdir -p "$HOME/Library/Caches"
			ln -s "$original_home/Library/Caches/baml" "$HOME/Library/Caches/baml"
		fi

		# Some unit tests rely on the Ollama endpoint being fixed at process start,
		# because the env package snapshots these vars during package initialization.
		export WHODB_OLLAMA_HOST="ollama.test"
		export WHODB_OLLAMA_PORT="11434"

		cd "$workdir"
		"$@"
	)
}

run_ce_unit() {
	echo "→ Running CE backend tests"
	run_hermetic_go_test "$ROOT_DIR/core" go test -race ./src/... ./graph/...
}

run_ee_unit() {
	if ! has_ee; then
		echo "ℹ️  EE module not present, skipping EE backend tests"
		return 0
	fi

	echo "→ Running EE backend tests"
	run_hermetic_go_test "$ROOT_DIR/ee" go test ./core/...
}

run_unit() {
	run_ce_unit
	run_ee_unit
}

run_ce_integration() {
	(
		set -euo pipefail
		echo "→ Running CE integration backend tests (docker-compose services required)"
		COMPOSE_FILE="$ROOT_DIR/dev/docker-compose.yml"
		MANAGE_COMPOSE="${WHODB_MANAGE_COMPOSE:-1}"
		COMPOSE_STARTED=0
		RUNNING_SERVICE_COUNT=0

		cleanup() {
			if [ "$MANAGE_COMPOSE" = "1" ] && [ "$COMPOSE_STARTED" -eq 1 ]; then
				echo "→ Tearing down CE integration docker-compose stack"
				docker compose -f "$COMPOSE_FILE" down --volumes --remove-orphans
			fi
		}
		trap cleanup EXIT

		if [ "$MANAGE_COMPOSE" = "1" ]; then
			RUNNING_SERVICE_COUNT="$(
				docker compose -f "$COMPOSE_FILE" ps -q \
					e2e_postgres e2e_mysql e2e_mariadb e2e_mysql_842 \
					e2e_mongo e2e_clickhouse e2e_redis e2e_elasticsearch |
					grep -c . || true
			)"
			if [ "$RUNNING_SERVICE_COUNT" -eq 8 ]; then
				echo "ℹ️  Reusing existing CE docker-compose stack"
			else
				echo "🐳 Starting CE integration docker-compose stack"
				docker compose -f "$COMPOSE_FILE" up -d
				COMPOSE_STARTED=1
			fi
		else
			echo "ℹ️  WHODB_MANAGE_COMPOSE=0, assuming CE services are already running"
		fi

		cd "$ROOT_DIR/core"
		# If we started compose ourselves, don't start again inside tests.
		START_FLAG="${WHODB_START_COMPOSE:-}"
		if [ "$COMPOSE_STARTED" -eq 1 ]; then
			START_FLAG="0"
		fi
		WHODB_START_COMPOSE="${START_FLAG:-0}" go test -tags integration \
			./src/plugins/postgres \
			./src/plugins/mysql \
			./src/plugins/clickhouse
		WHODB_START_COMPOSE="${START_FLAG:-0}" go test -tags integration ./test/integration/...
	)
}

run_ee_integration() {
	if ! has_ee; then
		echo "ℹ️  EE module not present, skipping EE integration tests"
		return 0
	fi

	(
		set -euo pipefail
		echo "→ Running EE integration backend tests (docker-compose services required)"
		COMPOSE_FILE="$ROOT_DIR/ee/dev/docker-compose.yml"
		MANAGE_COMPOSE="${WHODB_MANAGE_COMPOSE:-1}"
		COMPOSE_STARTED=0
		RUNNING_SERVICE_COUNT=0

		cleanup() {
			if [ "$MANAGE_COMPOSE" = "1" ] && [ "$COMPOSE_STARTED" -eq 1 ]; then
				echo "→ Tearing down EE integration docker-compose stack"
				docker compose -p "$EE_COMPOSE_PROJECT" -f "$COMPOSE_FILE" --profile ee down --volumes --remove-orphans
			fi
		}
		trap cleanup EXIT

		if [ "$MANAGE_COMPOSE" = "1" ]; then
			RUNNING_SERVICE_COUNT="$(
				docker compose -p "$EE_COMPOSE_PROJECT" -f "$COMPOSE_FILE" --profile ee ps -q \
					e2e_mssql e2e_dynamodb e2e_oracle e2e_cassandra |
					grep -c . || true
			)"
			if [ "$RUNNING_SERVICE_COUNT" -eq 4 ]; then
				echo "ℹ️  Reusing existing EE docker-compose stack"
			else
				echo "🐳 Starting EE integration docker-compose stack"
				docker compose -p "$EE_COMPOSE_PROJECT" -f "$COMPOSE_FILE" --profile ee up -d
				COMPOSE_STARTED=1
			fi
		else
			echo "ℹ️  WHODB_MANAGE_COMPOSE=0, assuming EE services are already running"
		fi

		cd "$ROOT_DIR/ee"
		START_FLAG="${WHODB_START_COMPOSE:-}"
		if [ "$COMPOSE_STARTED" -eq 1 ]; then
			START_FLAG="0"
		fi
		WHODB_START_COMPOSE="${START_FLAG:-0}" go test -tags integration \
			./core/src/plugins/cassandra \
			./core/src/plugins/dynamodb \
			./core/src/plugins/mssql \
			./core/src/plugins/oracle
		WHODB_START_COMPOSE="${START_FLAG:-0}" go test -tags integration ./test/integration/...
	)
}

run_integration() {
	run_ce_integration
	run_ee_integration
}

run_ssl() {
	(
		set -euo pipefail
		echo "→ Running SSL integration tests (docker-compose ssl profile required)"
		COMPOSE_FILE="$ROOT_DIR/dev/docker-compose.yml"
		MANAGE_COMPOSE="${WHODB_MANAGE_COMPOSE:-1}"
		COMPOSE_STARTED=0

		cleanup() {
			if [ "$MANAGE_COMPOSE" = "1" ] && [ "$COMPOSE_STARTED" -eq 1 ]; then
				echo "→ Tearing down SSL docker-compose stack"
				docker compose -f "$COMPOSE_FILE" --profile ssl down --volumes --remove-orphans
			fi
		}
		trap cleanup EXIT

		if [ "$MANAGE_COMPOSE" = "1" ]; then
			if docker compose -f "$COMPOSE_FILE" --profile ssl ps -q | grep -q .; then
				echo "ℹ️  Reusing existing SSL docker-compose stack"
			else
				echo "🐳 Starting SSL docker-compose stack (profile: ssl)"
				docker compose -f "$COMPOSE_FILE" --profile ssl up -d
				COMPOSE_STARTED=1
				echo "⏳ Waiting for SSL services to be ready..."
				sleep 10
			fi
		else
			echo "ℹ️  WHODB_MANAGE_COMPOSE=0, assuming SSL services are already running"
		fi

		cd "$ROOT_DIR/core"
		WHODB_SSL_TESTS=1 go test -tags integration -v -run "SSL" ./test/integration/...
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
ce-unit)
	run_ce_unit
	;;
race)
	echo "→ Running CE tests with race detector (10× iterations)"
	run_hermetic_go_test "$ROOT_DIR/core" go test -race -count=10 ./src/... ./graph/...
	;;
ee-unit)
	run_ee_unit
	;;
ce-integration)
	run_ce_integration
	;;
ee-integration)
	run_ee_integration
	;;
integration)
	run_integration
	;;
ssl)
	run_ssl
	;;
-h | --help | help)
	usage
	;;
*)
	echo "Unknown MODE: $MODE"
	usage
	exit 1
	;;
esac
