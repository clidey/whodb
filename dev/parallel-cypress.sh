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

set -e

EDITION="${1:-ce}"
HEADLESS="${2:-true}"
TARGET_DB="${3:-all}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Database configurations
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

echo "🚀 Running Cypress tests in parallel (1 process per database)"
echo "   Edition: $EDITION"
echo "   Headless: $HEADLESS"
echo "   Target DB: $TARGET_DB"

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

# Setup backend (pass TARGET_DB to only start required containers)
echo "⚙️ Setting up test environment..."
bash "$SCRIPT_DIR/setup-e2e.sh" "$EDITION" "$TARGET_DB"

# Start frontend
echo "🌐 Starting frontend..."
cd "$PROJECT_ROOT/frontend"
if [ "$EDITION" = "ee" ]; then
    VITE_BUILD_EDITION=ee NODE_ENV=test vite --port 3000 --clearScreen false --logLevel error &
else
    NODE_ENV=test vite --port 3000 --clearScreen false --logLevel error &
fi
FRONTEND_PID=$!

# Wait for services
echo "⏳ Waiting for services..."
MAX_WAIT=60 bash "$SCRIPT_DIR/wait-for-services.sh"

# Brief wait to ensure backend is fully ready to handle connections
echo "⏳ Giving services a moment to stabilize..."
sleep 2

cd "$PROJECT_ROOT/frontend"

# Create logs directory if it doesn't exist
mkdir -p cypress/logs

# Clean previous logs
rm -f cypress/logs/*.log 2>/dev/null || true

echo "📋 Running ${#DATABASES[@]} database tests in parallel..."

# Calculate stagger time based on number of databases
if [ ${#DATABASES[@]} -le 4 ]; then
    STAGGER_TIME=0.5
elif [ ${#DATABASES[@]} -le 7 ]; then
    STAGGER_TIME=0.4
else
    STAGGER_TIME=0.3
fi

echo "📊 Staggering test starts by ${STAGGER_TIME}s to ensure stable connections"

# Run each database in parallel
PIDS=()
COUNTER=0

for db in "${DATABASES[@]}"; do
    echo "🚀 Starting: $db (${DB_CATEGORIES[$db]})"

    if [ "$HEADLESS" = "true" ]; then
        # Run all feature tests for this database
        CYPRESS_database="$db" \
        CYPRESS_category="${DB_CATEGORIES[$db]}" \
        CYPRESS_isDocker="true" \
        CYPRESS_retries__runMode=2 \
        CYPRESS_retries__openMode=0 \
        NODE_ENV=test npx cypress run \
            --spec "cypress/e2e/features/**/*.cy.js" \
            --browser chromium \
            > "cypress/logs/$db.log" 2>&1 &

        PIDS+=($!)

        # Stagger all test starts to avoid connection storms
        COUNTER=$((COUNTER + 1))
        if [ $COUNTER -lt ${#DATABASES[@]} ]; then
            sleep $STAGGER_TIME
        fi
    else
        echo "⚠️ Parallel mode requires headless"
        exit 1
    fi
done

# Wait for all tests and collect results
echo "⏳ Waiting for all database tests to complete..."
FAILED_DBS=()

for i in "${!PIDS[@]}"; do
    PID=${PIDS[$i]}
    DB=${DATABASES[$i]}

    if wait $PID; then
        echo "✅ $DB passed"
    else
        echo "❌ $DB failed"
        FAILED_DBS+=("$DB")
    fi
done

# Cleanup
echo "🧹 Cleaning up..."
kill $FRONTEND_PID 2>/dev/null || true
bash "$SCRIPT_DIR/cleanup-e2e.sh" "$EDITION"

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

        LOG_FILE="cypress/logs/$db.log"
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
