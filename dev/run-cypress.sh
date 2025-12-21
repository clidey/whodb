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

# Sequential Cypress test runner for Community Edition (CE)
# Runs tests for multiple databases sequentially with a single backend+frontend
#
# Usage:
#   ./run-cypress.sh [headless] [database] [spec]
#
# Arguments:
#   headless - 'true' or 'false' (default: true)
#   database - specific database to test, or 'all' (default: all)
#   spec     - specific spec file to run (default: all features)
#
# Examples:
#   ./run-cypress.sh true postgres data-types    # Headless, postgres only, data-types spec
#
# Architecture:
#   Single shared stack for all database tests:
#   - backend:8080 ← frontend:3000 ← cypress (sequential per database)
#
# Coverage:
#   Backend writes coverage.out when terminated with SIGTERM.

set -e

# ============================================================================
# CONFIGURATION - Modify these values as needed
# ============================================================================
# Backend log level: "debug", "info", "warning", "error", "none"
export WHODB_LOG_LEVEL="${WHODB_LOG_LEVEL:-error}"
# ============================================================================

HEADLESS="${1:-true}"
TARGET_DB="${2:-all}"
SPEC_FILE="${3:-}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# CE database configurations
DATABASES=(postgres mysql mysql8 mariadb sqlite mongodb redis elasticsearch clickhouse)

# Map database to category for logging
declare -A DB_CATEGORIES=(
    [postgres]="sql"
    [mysql]="sql"
    [mysql8]="sql"
    [mariadb]="sql"
    [sqlite]="sql"
    [mongodb]="document"
    [redis]="keyvalue"
    [elasticsearch]="document"
    [clickhouse]="sql"
)

echo "🚀 Running Cypress tests sequentially (CE)"
echo "   Headless: $HEADLESS"
echo "   Target DB: $TARGET_DB"
echo "   Log Level: $WHODB_LOG_LEVEL"
if [ -n "$SPEC_FILE" ]; then
    echo "   Spec: $SPEC_FILE"
fi

# Filter databases if specific one requested
if [ "$TARGET_DB" != "all" ]; then
    FOUND=false
    for db in "${DATABASES[@]}"; do
        if [ "$db" = "$TARGET_DB" ]; then
            FOUND=true
            break
        fi
    done
    if [ "$FOUND" = "false" ]; then
        echo "❌ Unknown database: $TARGET_DB"
        echo "   Available: ${DATABASES[*]}"
        exit 1
    fi
    DATABASES=("$TARGET_DB")
fi

echo "   Databases: ${DATABASES[*]}"

# Setup environment (databases + build binary + start backend)
echo "⚙️ Setting up test environment..."
bash "$SCRIPT_DIR/setup-e2e.sh" "ce" "$TARGET_DB"

cd "$PROJECT_ROOT/frontend"

# Create logs directory if it doesn't exist
mkdir -p cypress/logs

# Clean previous test artifacts (once at suite start, not per-database)
rm -f cypress/logs/*.log 2>/dev/null || true
rm -rf cypress/screenshots/* 2>/dev/null || true
rm -rf cypress/videos/* 2>/dev/null || true

# Start frontend dev server
echo "🌐 Starting frontend dev server..."
NODE_ENV=test pnpm exec vite --port 3000 --clearScreen false --logLevel error > cypress/logs/frontend.log 2>&1 &
FRONTEND_PID=$!

# Wait for frontend to be ready
echo "⏳ Waiting for frontend to be ready..."
COUNTER=0
while [ $COUNTER -lt 30 ]; do
    if nc -z localhost 3000 2>/dev/null; then
        echo "✅ Frontend is ready on port 3000"
        break
    fi
    sleep 0.5
    COUNTER=$((COUNTER + 1))
done

if ! nc -z localhost 3000 2>/dev/null; then
    echo "❌ Frontend failed to start"
    kill $FRONTEND_PID 2>/dev/null || true
    exit 1
fi

echo "📋 Running ${#DATABASES[@]} database tests sequentially..."

# Track results
FAILED_DBS=()

# Run tests for each database sequentially
for db in "${DATABASES[@]}"; do
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "🧪 Testing: $db (${DB_CATEGORIES[$db]})"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

    # Build spec pattern
    if [ -n "$SPEC_FILE" ]; then
        if [[ "$SPEC_FILE" == *.cy.* ]]; then
            SPEC_PATTERN="cypress/e2e/features/$SPEC_FILE"
        else
            SPEC_PATTERN="cypress/e2e/features/$SPEC_FILE.cy.js"
        fi
    else
        SPEC_PATTERN="cypress/e2e/features/**/*.cy.js"
    fi

    # Run Cypress test
    if [ "$HEADLESS" = "true" ]; then
        CYPRESS_database="$db" \
        CYPRESS_category="${DB_CATEGORIES[$db]}" \
        CYPRESS_retries__runMode=2 \
        CYPRESS_retries__openMode=0 \
        NODE_ENV=test pnpx cypress run \
            --spec "$SPEC_PATTERN" \
            --browser electron \
            2>&1 | tee "cypress/logs/$db.log"
        RESULT=${PIPESTATUS[0]}
    else
        CYPRESS_database="$db" \
        CYPRESS_category="${DB_CATEGORIES[$db]}" \
        NODE_ENV=test pnpx cypress open \
            --e2e \
            --browser electron
        RESULT=$?
    fi

    if [ $RESULT -eq 0 ]; then
        echo "✅ $db passed"
    else
        echo "❌ $db failed"
        FAILED_DBS+=("$db")
    fi
done

# Cleanup
echo ""
echo "🧹 Cleaning up..."

# Kill frontend
kill $FRONTEND_PID 2>/dev/null || true

# Run standard cleanup (stops backend, docker containers)
bash "$SCRIPT_DIR/cleanup-e2e.sh" "ce"

# Report results
echo ""
echo "📊 Test Results:"
echo "   Total: ${#DATABASES[@]} databases"
echo "   Failed: ${#FAILED_DBS[@]} databases"

if [ ${#FAILED_DBS[@]} -gt 0 ]; then
    echo ""
    echo "❌ Failed Tests:"
    for db in "${FAILED_DBS[@]}"; do
        echo ""
        echo "   [$db] (${DB_CATEGORIES[$db]})"

        LOG_FILE="$PROJECT_ROOT/frontend/cypress/logs/$db.log"
        if [ -f "$LOG_FILE" ]; then
            # Parse the Cypress summary table to show failed specs
            FAILED_SPECS=$(grep -E "│\s*✖" "$LOG_FILE" 2>/dev/null | sed 's/.*✖\s*//' | sed 's/│.*//' | while read -r line; do
                # Extract spec name and test counts
                SPEC=$(echo "$line" | awk '{print $1}')
                TOTAL=$(echo "$line" | awk '{print $3}')
                PASSED=$(echo "$line" | awk '{print $4}')
                FAILED=$(echo "$line" | awk '{print $5}')
                echo "     - $SPEC ($PASSED/$TOTAL passed, $FAILED failed)"
            done)

            if [ -n "$FAILED_SPECS" ]; then
                echo "$FAILED_SPECS"
            else
                # Fallback: show summary line
                SUMMARY=$(grep -E "✖.*failed" "$LOG_FILE" 2>/dev/null | tail -1)
                if [ -n "$SUMMARY" ]; then
                    echo "     $SUMMARY"
                else
                    echo "     (Could not parse test results)"
                fi
            fi
        else
            echo "     (Log file not found)"
        fi
        echo "     Log: $LOG_FILE"
    done
    exit 1
else
    echo "✅ All database tests passed!"
    echo "📁 Logs available in: frontend/cypress/logs/"
    exit 0
fi
