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

# Playwright E2E test runner for WhoDB
# Replaces run-cypress.sh with the same interface
#
# Usage:
#   ./run-e2e.sh [headless] [database] [spec]
#
# Arguments:
#   headless - 'true' or 'false' (default: true)
#   database - specific database to test, or 'all' (default: all)
#   spec     - specific spec file to run (default: all features)
#
# Environment variables for customization:
#   WHODB_DATABASES      - space-separated list of databases to test
#   WHODB_DB_CATEGORIES  - colon-separated db:category pairs (e.g., "postgres:sql mysql:sql")
#   WHODB_VITE_EDITION   - vite build edition (empty for CE)
#   WHODB_SETUP_MODE     - mode to pass to setup-e2e.sh (default: ce)
#   WHODB_EDITION_LABEL  - label for output (default: CE)
#   WHODB_EXTRA_WAIT     - set to 'true' for extra service wait time
#   CDP_ENDPOINT         - if set, connects to Gateway CEF browser instead of launching Chromium
#
# Examples:
#   ./run-e2e.sh true postgres tables-list    # Headless, postgres only, tables-list spec
#   ./run-e2e.sh false all                    # GUI mode (--headed), all databases
#
# Architecture:
#   - Headless mode: Loops through databases sequentially for better isolation/logging
#   - GUI mode: Single Playwright session with --headed flag

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
VITE_EDITION="${WHODB_VITE_EDITION:-}"
SETUP_MODE="${WHODB_SETUP_MODE:-ce}"
EDITION_LABEL="${WHODB_EDITION_LABEL:-CE}"
EXTRA_WAIT="${WHODB_EXTRA_WAIT:-false}"

# Convert space-separated string to array
read -ra DATABASES <<< "$DATABASES_STR"

# Lookup function for category
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

echo "🚀 Running Playwright E2E tests ($EDITION_LABEL)"
echo "   Headless: $HEADLESS"
echo "   Target DB: $TARGET_DB"
echo "   Log Level: $WHODB_LOG_LEVEL"
if [ -n "$SPEC_FILE" ]; then
    echo "   Spec: $SPEC_FILE"
fi
if [ -n "$CDP_ENDPOINT" ]; then
    echo "   Mode: Gateway CDP ($CDP_ENDPOINT)"
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

# Create logs directory
mkdir -p e2e/logs

# Clean previous test artifacts (once at suite start, not per-database)
# Use sudo if needed - gateway/Docker runs may leave root-owned files
rm -f e2e/logs/*.log 2>/dev/null || true
rm -rf e2e/reports/* 2>/dev/null || sudo rm -rf e2e/reports/* 2>/dev/null || true
mkdir -p e2e/reports/test-results e2e/reports/blobs e2e/reports/html

# Start frontend dev server
echo "🌐 Starting frontend dev server..."
BACKEND_PORT="${WHODB_BACKEND_PORT:-8080}"
if [ -n "$VITE_EDITION" ]; then
    VITE_BUILD_EDITION="$VITE_EDITION" VITE_BACKEND_PORT="$BACKEND_PORT" NODE_ENV=test pnpm exec vite --port 3000 --clearScreen false --logLevel error > e2e/logs/frontend.log 2>&1 &
else
    VITE_BACKEND_PORT="$BACKEND_PORT" NODE_ENV=test pnpm exec vite --port 3000 --clearScreen false --logLevel error > e2e/logs/frontend.log 2>&1 &
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
    if [[ "$SPEC_FILE" == *.spec.* ]]; then
        SPEC_PATTERN="tests/features/$SPEC_FILE"
    else
        SPEC_PATTERN="tests/features/$SPEC_FILE.spec.mjs"
    fi
else
    SPEC_PATTERN=""
fi

# Determine Playwright project
PW_PROJECT="standalone"
if [ -n "$CDP_ENDPOINT" ]; then
    PW_PROJECT="gateway"
fi

# Common Playwright args
PW_CONFIG="$PROJECT_ROOT/frontend/e2e/playwright.config.mjs"
PW_ARGS="--config=$PW_CONFIG --project=$PW_PROJECT"
if [ "$HEADLESS" = "false" ]; then
    PW_ARGS="$PW_ARGS --headed"
fi

if [ "$HEADLESS" = "true" ]; then
    # Headless mode: Run all databases in parallel (1 Playwright process per database).
    # Each process gets its own browser, outputDir, and blob report — no file collisions.
    # The backend handles concurrent connections fine (9 cached connections, well under the 50 limit).
    echo "📋 Running ${#DATABASES[@]} database tests in parallel..."

    declare -A DB_PIDS

    for db in "${DATABASES[@]}"; do
        echo "🧪 Starting: $db ($(get_category "$db"))"

        (
            cd "$PROJECT_ROOT/frontend"
            DATABASE="$db" \
            CATEGORY="$(get_category "$db")" \
            pnpm exec playwright test \
                $PW_ARGS \
                $SPEC_PATTERN \
                > "$PROJECT_ROOT/frontend/e2e/logs/$db.log" 2>&1
            exit $?
        ) &
        DB_PIDS["$db"]=$!
    done

    echo ""
    echo "⏳ Waiting for all databases to complete..."

    # Track which databases have finished
    declare -A DB_DONE
    DONE_COUNT=0
    TOTAL=${#DATABASES[@]}

    while [ $DONE_COUNT -lt $TOTAL ]; do
        for db in "${DATABASES[@]}"; do
            [ -n "${DB_DONE[$db]}" ] && continue
            if ! kill -0 "${DB_PIDS[$db]}" 2>/dev/null; then
                wait "${DB_PIDS[$db]}" && DB_DONE[$db]="pass" || DB_DONE[$db]="fail"
                DONE_COUNT=$((DONE_COUNT + 1))
                [ "${DB_DONE[$db]}" = "fail" ] && FAILED_DBS+=("$db")

                # Print result permanently
                printf "\r\033[2K"
                if [ "${DB_DONE[$db]}" = "pass" ]; then
                    echo "✅ [$DONE_COUNT/$TOTAL] $db passed"
                else
                    echo "❌ [$DONE_COUNT/$TOTAL] $db failed (see e2e/logs/$db.log)"
                fi
            fi
        done

        if [ $DONE_COUNT -lt $TOTAL ]; then
            # Single-line status showing what each running db is on
            RUNNING=""
            for db in "${DATABASES[@]}"; do
                [ -n "${DB_DONE[$db]}" ] && continue
                LOG="$PROJECT_ROOT/frontend/e2e/logs/$db.log"
                SPEC=$(grep -oP '› e2e/tests/features/\K[^:]+' "$LOG" 2>/dev/null | tail -1 | sed 's/\.spec\.mjs//')
                RUNNING="$RUNNING $db(${SPEC:-…})"
            done
            printf "\r\033[2K⏳ %d/%d done |%s" "$DONE_COUNT" "$TOTAL" "$RUNNING"
            sleep 2
        fi
    done
    echo ""
else
    # GUI mode: Single Playwright session with --headed
    echo "📋 Opening Playwright in headed mode..."

    ENV_VARS=""
    if [ "$TARGET_DB" != "all" ]; then
        ENV_VARS="DATABASE=$TARGET_DB CATEGORY=$(get_category "$TARGET_DB")"
    fi

    (
        cd "$PROJECT_ROOT/frontend"
        env $ENV_VARS pnpm exec playwright test $PW_ARGS --headed
        exit $?
    ) && RESULT=0 || RESULT=$?

    if [ $RESULT -ne 0 ]; then
        FAILED_DBS+=("gui-session")
    fi
fi

# Merge blob reports from all database runs into a single HTML report
if [ -d "$PROJECT_ROOT/frontend/e2e/reports/blobs" ] && ls "$PROJECT_ROOT/frontend/e2e/reports/blobs"/*.zip 1>/dev/null 2>&1; then
    echo ""
    echo "📊 Merging test reports..."
    (
        cd "$PROJECT_ROOT/frontend"
        PLAYWRIGHT_HTML_OPEN=never \
        PLAYWRIGHT_HTML_OUTPUT_DIR=e2e/reports/html \
        pnpm exec playwright merge-reports \
            e2e/reports/blobs \
            --reporter=html \
            2>/dev/null || true
    )
    echo "📁 HTML report: frontend/e2e/reports/html/index.html"
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

        LOG_FILE="$PROJECT_ROOT/frontend/e2e/logs/$db.log"
        if [ -f "$LOG_FILE" ]; then
            # Show last few lines with failure info
            FAILURES=$(grep -E "(✘|FAILED|Error)" "$LOG_FILE" 2>/dev/null | tail -10)
            if [ -n "$FAILURES" ]; then
                echo "$FAILURES" | sed 's/^/     /'
            else
                echo "     (Could not parse test results)"
            fi
        else
            echo "     (Log file not found)"
        fi
        echo "     Log: $LOG_FILE"
    done
    exit 1
else
    echo "✅ All database tests passed!"
    echo "📁 Logs available in: frontend/e2e/logs/"
    echo "📁 HTML report: frontend/e2e/reports/html/"
    exit 0
fi
