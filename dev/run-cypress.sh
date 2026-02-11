#!/bin/bash
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

# Cypress test runner for WhoDB
# Runs tests with a single backend+frontend
#
# Usage:
#   ./run-cypress.sh [headless] [database] [spec]
#
# Arguments:
#   headless - 'true' or 'false' (default: true)
#   database - specific database to test, or 'all' (default: all)
#   spec     - specific spec file to run (default: all features)
#
# Environment variables for customization:
#   WHODB_DATABASES      - space-separated list of databases to test
#   WHODB_DB_CATEGORIES  - colon-separated db:category pairs (e.g., "postgres:sql mysql:sql")
#   WHODB_CYPRESS_DIRS   - colon-separated db:dir pairs for non-default cypress dirs
#   WHODB_VITE_EDITION   - vite build edition (empty for CE)
#   WHODB_SETUP_MODE     - mode to pass to setup-e2e.sh (default: ce)
#   WHODB_EDITION_LABEL  - label for output (default: CE)
#   WHODB_EXTRA_WAIT     - set to 'true' for extra service wait time
#
# Examples:
#   ./run-cypress.sh true postgres data-types    # Headless, postgres only, data-types spec
#   ./run-cypress.sh false all                   # GUI mode, all databases in single session
#
# Architecture:
#   - Headless mode: Loops through databases sequentially for better isolation/logging
#   - GUI mode: Single Cypress session, forEachDatabase handles iteration internally

set -e

# ============================================================================
# CONFIGURATION
# ============================================================================
export WHODB_LOG_LEVEL="${WHODB_LOG_LEVEL:-error}"
# ============================================================================

HEADLESS="${1:-true}"
TARGET_DB="${2:-all}"
SPEC_FILE="${3:-}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Default CE database configurations (can be overridden via env vars)
DEFAULT_DATABASES="postgres mysql mysql8 mariadb sqlite mongodb redis elasticsearch clickhouse"
DEFAULT_CATEGORIES="postgres:sql mysql:sql mysql8:sql mariadb:sql sqlite:sql mongodb:document redis:keyvalue elasticsearch:document clickhouse:sql"

# Use env vars or defaults
DATABASES_STR="${WHODB_DATABASES:-$DEFAULT_DATABASES}"
CATEGORIES_STR="${WHODB_DB_CATEGORIES:-$DEFAULT_CATEGORIES}"
CYPRESS_DIRS_STR="${WHODB_CYPRESS_DIRS:-}"
VITE_EDITION="${WHODB_VITE_EDITION:-}"
SETUP_MODE="${WHODB_SETUP_MODE:-ce}"
EDITION_LABEL="${WHODB_EDITION_LABEL:-CE}"
EXTRA_WAIT="${WHODB_EXTRA_WAIT:-false}"

# Export spec file so setup scripts can decide whether SSL is needed
export WHODB_SPEC_FILE="${WHODB_SPEC_FILE:-$SPEC_FILE}"

# Convert space-separated string to array
read -ra DATABASES <<< "$DATABASES_STR"

# Lookup functions for category and cypress dir (Bash 3.2 compatible)
get_category() {
    local lookup_db="$1"
    for pair in $CATEGORIES_STR; do
        local db="${pair%%:*}"
        local cat="${pair#*:}"
        if [ "$db" = "$lookup_db" ]; then
            echo "$cat"
            return
        fi
    done
    echo "unknown"
}

get_cypress_dir() {
    local lookup_db="$1"
    if [ -z "$CYPRESS_DIRS_STR" ]; then
        echo "$PROJECT_ROOT/frontend"
        return
    fi
    for pair in $CYPRESS_DIRS_STR; do
        local db="${pair%%:*}"
        local dir="${pair#*:}"
        if [ "$db" = "$lookup_db" ]; then
            echo "$dir"
            return
        fi
    done
    echo "$PROJECT_ROOT/frontend"
}

echo "🚀 Running Cypress tests ($EDITION_LABEL)"
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
bash "$SCRIPT_DIR/setup-e2e.sh" "$SETUP_MODE" "$TARGET_DB"

cd "$PROJECT_ROOT/frontend"

# Create logs directory if it doesn't exist
mkdir -p cypress/logs

# Clean previous test artifacts (once at suite start, not per-database)
rm -f cypress/logs/*.log 2>/dev/null || true
rm -rf cypress/screenshots/* 2>/dev/null || true
rm -rf cypress/videos/* 2>/dev/null || true

# Start frontend dev server
echo "🌐 Starting frontend dev server..."
# WHODB_BACKEND_PORT can be set by EE script for containerized backend (default: 8080)
BACKEND_PORT="${WHODB_BACKEND_PORT:-8080}"
if [ -n "$VITE_EDITION" ]; then
    VITE_BUILD_EDITION="$VITE_EDITION" VITE_BACKEND_PORT="$BACKEND_PORT" NODE_ENV=test pnpm exec vite --port 3000 --clearScreen false --logLevel error > cypress/logs/frontend.log 2>&1 &
else
    VITE_BACKEND_PORT="$BACKEND_PORT" NODE_ENV=test pnpm exec vite --port 3000 --clearScreen false --logLevel error > cypress/logs/frontend.log 2>&1 &
fi
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

# Extra wait for services if needed
if [ "$EXTRA_WAIT" = "true" ]; then
    echo "⏳ Waiting for services..."
    MAX_WAIT=90 bash "$SCRIPT_DIR/wait-for-services.sh"
    echo "⏳ Giving services a moment to stabilize..."
    sleep 2
fi

# Track results
FAILED_DBS=()

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

if [ "$HEADLESS" = "true" ]; then
    # Headless mode: Loop through databases for better isolation and per-database logs
    echo "📋 Running ${#DATABASES[@]} database tests sequentially..."

    for db in "${DATABASES[@]}"; do
        echo ""
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        echo "🧪 Testing: $db ($(get_category "$db"))"
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

        # Determine cypress directory (use override if set, otherwise default)
        CYPRESS_DIR="$(get_cypress_dir "$db")"

        if [ "$CYPRESS_DIR" = "$PROJECT_ROOT/frontend" ]; then
            # Default dir - use spec pattern
            (
                cd "$CYPRESS_DIR"
                CYPRESS_database="$db" \
                CYPRESS_category="$(get_category "$db")" \
                CYPRESS_retries__runMode=2 \
                CYPRESS_retries__openMode=0 \
                NODE_ENV=test pnpx cypress run \
                    --spec "$SPEC_PATTERN" \
                    --browser electron \
                    2>&1 | tee "$PROJECT_ROOT/frontend/cypress/logs/$db.log"
                exit ${PIPESTATUS[0]}
            ) && RESULT=0 || RESULT=$?
        else
            # Custom dir (e.g., EE) - build spec pattern for both CE and EE features
            if [ -n "$SPEC_FILE" ]; then
                if [[ "$SPEC_FILE" == *.cy.* ]]; then
                    # Full filename provided - check both CE and EE locations
                    EE_SPEC_PATTERN="cypress/e2e/features/$SPEC_FILE,../../frontend/cypress/e2e/features/$SPEC_FILE"
                else
                    # Short name provided - add .cy.js extension
                    EE_SPEC_PATTERN="cypress/e2e/features/$SPEC_FILE.cy.js,../../frontend/cypress/e2e/features/$SPEC_FILE.cy.js"
                fi
                (
                    cd "$CYPRESS_DIR"
                    CYPRESS_database="$db" \
                    CYPRESS_category="$(get_category "$db")" \
                    CYPRESS_retries__runMode=2 \
                    CYPRESS_retries__openMode=0 \
                    NODE_ENV=test pnpx cypress run \
                        --spec "$EE_SPEC_PATTERN" \
                        --browser electron \
                        2>&1 | tee "$PROJECT_ROOT/frontend/cypress/logs/$db.log"
                    exit ${PIPESTATUS[0]}
                ) && RESULT=0 || RESULT=$?
            else
                # No spec file - run all specs (let cypress.config.js specPattern handle it)
                (
                    cd "$CYPRESS_DIR"
                    CYPRESS_database="$db" \
                    CYPRESS_category="$(get_category "$db")" \
                    CYPRESS_retries__runMode=2 \
                    CYPRESS_retries__openMode=0 \
                    NODE_ENV=test pnpx cypress run \
                        --browser electron \
                        2>&1 | tee "$PROJECT_ROOT/frontend/cypress/logs/$db.log"
                    exit ${PIPESTATUS[0]}
                ) && RESULT=0 || RESULT=$?
            fi
        fi

        if [ $RESULT -eq 0 ]; then
            echo "✅ $db passed"
        else
            echo "❌ $db failed"
            FAILED_DBS+=("$db")
        fi
    done
else
    # GUI mode: Single Cypress session, forEachDatabase handles iteration internally
    echo "📋 Opening Cypress GUI (forEachDatabase handles database iteration)..."

    # Determine cypress directory (use first database's dir, or default)
    CYPRESS_DIR="$(get_cypress_dir "${DATABASES[0]}")"

    # Set database filter if specific database requested
    ENV_VARS=""
    if [ "$TARGET_DB" != "all" ]; then
        ENV_VARS="CYPRESS_database=$TARGET_DB CYPRESS_category=$(get_category "$TARGET_DB")"
    fi

    (
        cd "$CYPRESS_DIR"
        env $ENV_VARS NODE_ENV=test pnpx cypress open --e2e --browser electron
        exit $?
    ) && RESULT=0 || RESULT=$?

    if [ $RESULT -ne 0 ]; then
        FAILED_DBS+=("gui-session")
    fi
fi

# Cleanup
echo ""
echo "🧹 Cleaning up..."

# Kill frontend
kill $FRONTEND_PID 2>/dev/null || true

# Run standard cleanup (stops backend, docker containers)
bash "$SCRIPT_DIR/cleanup-e2e.sh" "$SETUP_MODE"

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
        echo "   [$db] ($(get_category "$db"))"

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
