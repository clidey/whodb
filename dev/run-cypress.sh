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

set -e

# Parse arguments
EDITION="${1:-ce}"  # Default to CE
HEADLESS="${2:-false}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Validate edition
if [ "$EDITION" != "ce" ] && [ "$EDITION" != "ee" ]; then
    echo "❌ Invalid edition: $EDITION. Use 'ce' or 'ee'"
    exit 1
fi

# Cleanup function
cleanup() {
    echo "🧹 Cleaning up test environment..."
    
    # Kill frontend gracefully first
    echo "   Stopping frontend..."
    pkill -TERM -f 'vite --port 3000' 2>/dev/null || true
    sleep 1
    
    # Run cleanup script with edition parameter
    echo "   Running cleanup script..."
    bash "$SCRIPT_DIR/cleanup-e2e.sh" "$EDITION"
    
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
    # Setup backend with edition parameter
    echo "🚀 Setting up $EDITION test environment..."
    bash "$SCRIPT_DIR/setup-e2e.sh" "$EDITION" || { echo "❌ Backend setup failed"; return 1; }

    # Start frontend with appropriate edition
    echo "🚀 Starting frontend ($EDITION mode)..."
    cd "$PROJECT_ROOT/frontend"
    
    if [ "$EDITION" = "ee" ]; then
        VITE_BUILD_EDITION=ee NODE_ENV=test vite --port 3000 &
    else
        NODE_ENV=test vite --port 3000 &
    fi
    FRONTEND_PID=$!

    # Wait for services
    echo "⏳ Waiting for services..."
    if [ "$EDITION" = "ee" ]; then
        MAX_WAIT=180 bash "$SCRIPT_DIR/wait-for-services.sh" || { echo "❌ Services failed to start"; return 1; }
    else
        MAX_WAIT=150 bash "$SCRIPT_DIR/wait-for-services.sh" || { echo "❌ Services failed to start"; return 1; }
    fi

    # Configure Cypress based on edition
    echo "🧪 Running Cypress tests ($EDITION)..."
    cd "$PROJECT_ROOT/frontend"
    
    if [ "$EDITION" = "ee" ]; then
        # For EE, check if EE test directory exists
        if [ -d "$PROJECT_ROOT/ee/frontend/cypress/e2e" ]; then
            # Include both CE and EE tests
            CYPRESS_CONFIG="{\"specPattern\":[\"cypress/e2e/**/*.cy.{js,jsx,ts,tsx}\",\"../ee/frontend/cypress/e2e/**/*.cy.{js,jsx,ts,tsx}\"]}"
        else
            echo "⚠️ EE test directory not found, running CE tests only"
            CYPRESS_CONFIG=""
        fi
    else
        # For CE, only run CE tests
        CYPRESS_CONFIG=""
    fi
    
    # Run Cypress
    if [ "$HEADLESS" = "true" ]; then
        if [ -n "$CYPRESS_CONFIG" ]; then
            NODE_ENV=test npx cypress run --browser chrome --config "$CYPRESS_CONFIG" || { echo "❌ Cypress tests failed"; return 1; }
        else
            NODE_ENV=test npx cypress run --browser chrome || { echo "❌ Cypress tests failed"; return 1; }
        fi
    else
        if [ -n "$CYPRESS_CONFIG" ]; then
            NODE_ENV=test npx cypress open --config "$CYPRESS_CONFIG" || { echo "❌ Cypress failed to open"; return 1; }
        else
            NODE_ENV=test pnpm cypress open || { echo "❌ Cypress failed to open"; return 1; }
        fi
    fi

    echo "✅ Test run complete"
    return 0
}

# Execute tests and capture exit code
run_tests
EXIT_CODE=$?

# The trap will handle cleanup and exit with the correct code