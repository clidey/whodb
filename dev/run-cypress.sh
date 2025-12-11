#!/bin/bash
#
# Copyright 2025 Clidey, Inc.
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

# Cypress test runner for Community Edition (CE)
#
# Usage:
#   ./run-cypress.sh [headless] [database]
#
# Arguments:
#   headless - 'true' or 'false' (default: false)
#   database - specific database to test, or 'all' (default: all)
#
# Examples:
#   ./run-cypress.sh false postgres   # Open Cypress UI for postgres only
#   ./run-cypress.sh true mysql       # Headless run for mysql only
#   ./run-cypress.sh false all        # Open Cypress UI for all CE databases

set -e

# Parse arguments
HEADLESS="${1:-false}"
TARGET_DB="${2:-all}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Available CE databases
CE_DBS="postgres mysql mysql8 mariadb sqlite mongodb redis elasticsearch clickhouse"

# Validate database if specified
if [ "$TARGET_DB" != "all" ]; then
    FOUND=false
    for db in $CE_DBS; do
        if [ "$db" = "$TARGET_DB" ]; then
            FOUND=true
            break
        fi
    done
    if [ "$FOUND" = "false" ]; then
        echo "❌ Unknown database: $TARGET_DB"
        echo "   Available CE databases: $CE_DBS"
        exit 1
    fi
fi

echo "🚀 Cypress Test Runner (CE)"
echo "   Headless: $HEADLESS"
echo "   Database: $TARGET_DB"

# Cleanup function
cleanup() {
    echo "🧹 Cleaning up test environment..."

    # Kill frontend gracefully first
    echo "   Stopping frontend..."
    pkill -TERM -f 'vite --port 3000' 2>/dev/null || true
    sleep 1

    # Run cleanup script
    echo "   Running cleanup script (preserving test binary cache)..."
    bash "$SCRIPT_DIR/cleanup-e2e.sh" "ce"

    # Kill anything still on port 3000 (just in case)
    echo "   Ensuring port 3000 is free..."
    if lsof -ti:3000 >/dev/null 2>&1; then
        lsof -ti:3000 | xargs kill -9 2>/dev/null || true
    fi

    echo "✅ Cleanup complete"
}

# Set trap to cleanup on any exit
trap 'cleanup; exit ${EXIT_CODE:-0}' EXIT INT TERM

# Main test execution with error handling
run_tests() {
    # Setup backend with database parameters
    echo "🚀 Setting up CE test environment..."
    bash "$SCRIPT_DIR/setup-e2e.sh" "ce" "$TARGET_DB" || { echo "❌ Backend setup failed"; return 1; }

    # Start frontend
    echo "🚀 Starting frontend (CE mode)..."
    cd "$PROJECT_ROOT/frontend"

    # Vite optimizations for faster startup
    VITE_FLAGS="--port 3000 --clearScreen false --logLevel warn"
    NODE_ENV=test vite $VITE_FLAGS &
    FRONTEND_PID=$!

    # Wait for services
    echo "⏳ Waiting for services..."
    MAX_WAIT=60 bash "$SCRIPT_DIR/wait-for-services.sh" || { echo "❌ Services failed to start"; return 1; }

    # Configure Cypress
    echo "🧪 Running Cypress tests (CE)..."
    cd "$PROJECT_ROOT/frontend"

    # Build spec pattern for feature-based tests
    CE_FEATURE_SPEC="cypress/e2e/features/**/*.cy.{js,jsx,ts,tsx}"
    CYPRESS_CONFIG="{\"specPattern\":\"$CE_FEATURE_SPEC\"}"

    # Detect available browser
    if command -v chromium >/dev/null 2>&1 || command -v chromium-browser >/dev/null 2>&1; then
        BROWSER="chromium"
    elif command -v google-chrome >/dev/null 2>&1 || command -v google-chrome-stable >/dev/null 2>&1; then
        BROWSER="chrome"
    else
        BROWSER=""
    fi

    # Build environment variables for database targeting
    ENV_VARS=""
    if [ "$TARGET_DB" != "all" ]; then
        ENV_VARS="CYPRESS_database=$TARGET_DB"
    fi

    # Run Cypress
    if [ "$HEADLESS" = "true" ]; then
        if [ -n "$BROWSER" ]; then
            BROWSER_ARG="--browser $BROWSER"
        else
            BROWSER_ARG=""
        fi

        env $ENV_VARS NODE_ENV=test pnpx cypress run $BROWSER_ARG --config "$CYPRESS_CONFIG" || { echo "❌ Cypress tests failed"; return 1; }
    else
        env $ENV_VARS NODE_ENV=test pnpx cypress open --config "$CYPRESS_CONFIG" || { echo "❌ Cypress failed to open"; return 1; }
    fi

    echo "✅ Test run complete"
    return 0
}

# Execute tests and capture exit code
run_tests
EXIT_CODE=$?

# The trap will handle cleanup and exit with the correct code
