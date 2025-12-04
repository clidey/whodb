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

# Unified Cypress test runner for both CE and EE
#
# Usage:
#   ./run-cypress.sh [edition] [headless] [database]
#
# Arguments:
#   edition  - 'ce', 'ee', or 'ee-only' (default: ce)
#   headless - 'true' or 'false' (default: false)
#   database - specific database to test, or 'all' (default: all)
#
# Examples:
#   ./run-cypress.sh ce false postgres   # Open Cypress UI for postgres only
#   ./run-cypress.sh ce true mysql       # Headless run for mysql only
#   ./run-cypress.sh ee false all        # Open Cypress UI for all EE databases

set -e

# Parse arguments
EDITION="${1:-ce}"  # Default to CE
HEADLESS="${2:-false}"
TARGET_DB="${3:-all}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Available databases
AVAILABLE_DBS="postgres mysql mysql8 mariadb sqlite mongodb redis elasticsearch clickhouse"

# Validate edition
if [ "$EDITION" != "ce" ] && [ "$EDITION" != "ee" ] && [ "$EDITION" != "ee-only" ]; then
    echo "❌ Invalid edition: $EDITION. Use 'ce', 'ee', or 'ee-only'"
    exit 1
fi

# Validate database if specified
if [ "$TARGET_DB" != "all" ]; then
    FOUND=false
    for db in $AVAILABLE_DBS; do
        if [ "$db" = "$TARGET_DB" ]; then
            FOUND=true
            break
        fi
    done
    if [ "$FOUND" = "false" ]; then
        echo "❌ Unknown database: $TARGET_DB"
        echo "   Available: $AVAILABLE_DBS"
        exit 1
    fi
fi

echo "🚀 Cypress Test Runner"
echo "   Edition: $EDITION"
echo "   Headless: $HEADLESS"
echo "   Database: $TARGET_DB"

# Cleanup function
cleanup() {
    echo "🧹 Cleaning up test environment..."

    # Kill frontend gracefully first
    echo "   Stopping frontend..."
    pkill -TERM -f 'vite --port 3000' 2>/dev/null || true
    sleep 1

    # Run cleanup script with edition parameter (ee-only uses ee cleanup)
    echo "   Running cleanup script (preserving test binary cache)..."
    if [ "$EDITION" = "ee-only" ]; then
        bash "$SCRIPT_DIR/cleanup-e2e.sh" "ee"
    else
        bash "$SCRIPT_DIR/cleanup-e2e.sh" "$EDITION"
    fi

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
    # Setup backend with edition parameter (pass ee-only directly)
    echo "🚀 Setting up $EDITION test environment..."
    bash "$SCRIPT_DIR/setup-e2e.sh" "$EDITION" || { echo "❌ Backend setup failed"; return 1; }

    # Start frontend with appropriate edition and optimizations
    echo "🚀 Starting frontend ($EDITION mode)..."
    cd "$PROJECT_ROOT/frontend"

    # Vite optimizations for faster startup
    VITE_FLAGS="--port 3000 --clearScreen false --logLevel warn"

    if [ "$EDITION" = "ee" ] || [ "$EDITION" = "ee-only" ]; then
        VITE_BUILD_EDITION=ee NODE_ENV=test vite $VITE_FLAGS &
    else
        NODE_ENV=test vite $VITE_FLAGS &
    fi
    FRONTEND_PID=$!

    # Wait for services
    echo "⏳ Waiting for services..."
    if [ "$EDITION" = "ee" ] || [ "$EDITION" = "ee-only" ]; then
        MAX_WAIT=90 bash "$SCRIPT_DIR/wait-for-services.sh" || { echo "❌ Services failed to start"; return 1; }
    else
        MAX_WAIT=60 bash "$SCRIPT_DIR/wait-for-services.sh" || { echo "❌ Services failed to start"; return 1; }
    fi

    # Configure Cypress based on edition
    echo "🧪 Running Cypress tests ($EDITION)..."
    cd "$PROJECT_ROOT/frontend"

    # Build spec pattern for feature-based tests
    FEATURE_SPEC="cypress/e2e/features/**/*.cy.{js,jsx,ts,tsx}"

    if [ "$EDITION" = "ee" ]; then
        # For EE, check if EE test directory exists
        if [ -d "$PROJECT_ROOT/ee/frontend/cypress/e2e" ]; then
            # Include both CE and EE tests
            CYPRESS_CONFIG="{\"specPattern\":[\"$FEATURE_SPEC\",\"../ee/frontend/cypress/e2e/**/*.cy.{js,jsx,ts,tsx}\"]}"
        else
            echo "⚠️ EE test directory not found, running CE tests only"
            CYPRESS_CONFIG="{\"specPattern\":\"$FEATURE_SPEC\"}"
        fi
    elif [ "$EDITION" = "ee-only" ]; then
        # For EE-only, only run EE tests
        if [ -d "$PROJECT_ROOT/ee/frontend/cypress/e2e" ]; then
            CYPRESS_CONFIG="{\"specPattern\":\"../ee/frontend/cypress/e2e/**/*.cy.{js,jsx,ts,tsx}\"}"
        else
            echo "❌ EE test directory not found"
            return 1
        fi
    else
        # For CE, only run CE feature tests
        CYPRESS_CONFIG="{\"specPattern\":\"$FEATURE_SPEC\"}"
    fi

    # Detect available browser
    # Check for chromium first (common on Linux/Mac), then chrome
    if command -v chromium >/dev/null 2>&1 || command -v chromium-browser >/dev/null 2>&1; then
        BROWSER="chromium"
    elif command -v google-chrome >/dev/null 2>&1 || command -v google-chrome-stable >/dev/null 2>&1; then
        BROWSER="chrome"
    else
        # Let Cypress use its default
        BROWSER=""
    fi

    # Build environment variables for database targeting
    ENV_VARS=""
    if [ "$TARGET_DB" != "all" ]; then
        ENV_VARS="CYPRESS_database=$TARGET_DB"
    fi
    ENV_VARS="$ENV_VARS CYPRESS_isDocker=true"

    # Run Cypress
    if [ "$HEADLESS" = "true" ]; then
        if [ -n "$BROWSER" ]; then
            BROWSER_ARG="--browser $BROWSER"
        else
            BROWSER_ARG=""
        fi

        if [ -n "$CYPRESS_CONFIG" ]; then
            env $ENV_VARS NODE_ENV=test npx cypress run $BROWSER_ARG --config "$CYPRESS_CONFIG" || { echo "❌ Cypress tests failed"; return 1; }
        else
            env $ENV_VARS NODE_ENV=test npx cypress run $BROWSER_ARG || { echo "❌ Cypress tests failed"; return 1; }
        fi
    else
        if [ -n "$CYPRESS_CONFIG" ]; then
            env $ENV_VARS NODE_ENV=test npx cypress open --config "$CYPRESS_CONFIG" || { echo "❌ Cypress failed to open"; return 1; }
        else
            env $ENV_VARS NODE_ENV=test npx cypress open || { echo "❌ Cypress failed to open"; return 1; }
        fi
    fi

    echo "✅ Test run complete"
    return 0
}

# Execute tests and capture exit code
run_tests
EXIT_CODE=$?

# The trap will handle cleanup and exit with the correct code
