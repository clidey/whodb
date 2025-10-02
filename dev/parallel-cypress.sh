#!/bin/bash
#
# Simple parallel Cypress - one process per test file
#

set -e

EDITION="${1:-ce}"
HEADLESS="${2:-true}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

echo "🚀 Running Cypress tests in parallel (1 process per test)"
echo "   Edition: $EDITION"

# Setup backend
echo "⚙️ Setting up test environment..."
bash "$SCRIPT_DIR/setup-e2e.sh" "$EDITION"

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
rm -f cypress/logs/*.log cypress/logs/*.json 2>/dev/null || true

# Get all test files
TESTS=(cypress/e2e/*.cy.js)
if [ "$EDITION" = "ee" ] && [ -d "../ee/frontend/cypress/e2e" ]; then
    EE_TESTS=(../ee/frontend/cypress/e2e/*.cy.js)
    TESTS=("${TESTS[@]}" "${EE_TESTS[@]}")
fi

echo "📋 Running ${#TESTS[@]} tests in parallel..."

# Run all tests in parallel - one process each with stagger for stability
PIDS=()
COUNTER=0

# Calculate stagger time based on number of tests
# For 9 tests: ~0.3s each, for 12 tests: ~0.25s each
# This spreads starts over 2-3 seconds total
if [ ${#TESTS[@]} -le 6 ]; then
    STAGGER_TIME=0.5
elif [ ${#TESTS[@]} -le 10 ]; then
    STAGGER_TIME=0.3
else
    STAGGER_TIME=0.25
fi

echo "📊 Staggering test starts by ${STAGGER_TIME}s to ensure stable connections"

for test in "${TESTS[@]}"; do
    if [ -f "$test" ]; then
        TEST_NAME=$(basename "$test" .cy.js)
        echo "🚀 Starting: $TEST_NAME"

        if [ "$HEADLESS" = "true" ]; then
            # Set retry environment variables for more resilient connections
            CYPRESS_retries__runMode=2 \
            CYPRESS_retries__openMode=0 \
            NODE_ENV=test npx cypress run --spec "$test" --browser chromium \
                --reporter json --reporter-options "output=cypress/logs/results-$TEST_NAME.json" \
                > "cypress/logs/$TEST_NAME.log" 2>&1 &
            PIDS+=($!)

            # Stagger all test starts to avoid connection storms
            COUNTER=$((COUNTER + 1))
            if [ $COUNTER -lt ${#TESTS[@]} ]; then
                sleep $STAGGER_TIME
            fi
        else
            echo "⚠️ Parallel mode requires headless"
            exit 1
        fi
    fi
done

# Wait for all tests and collect results
echo "⏳ Waiting for all tests to complete..."
FAILED_TESTS=()

for i in "${!PIDS[@]}"; do
    PID=${PIDS[$i]}
    TEST=${TESTS[$i]}
    TEST_NAME=$(basename "$TEST" .cy.js)

    if wait $PID; then
        echo "✅ $TEST_NAME passed"
    else
        echo "❌ $TEST_NAME failed"
        FAILED_TESTS+=("$TEST_NAME")
    fi
done

# Cleanup
echo "🧹 Cleaning up..."
kill $FRONTEND_PID 2>/dev/null || true
bash "$SCRIPT_DIR/cleanup-e2e.sh" "$EDITION"

# Report results
echo ""
echo "📊 Test Results:"
echo "   Total: ${#TESTS[@]} tests"
echo "   Failed: ${#FAILED_TESTS[@]} tests"

if [ ${#FAILED_TESTS[@]} -gt 0 ]; then
    echo ""
    echo "❌ Failed tests:"
    for test in "${FAILED_TESTS[@]}"; do
        echo "   - $test"
        echo "     Log: cypress/logs/$test.log"
    done
    exit 1
else
    echo "✅ All tests passed!"
    echo "📁 Logs available in: frontend/cypress/logs/"
    exit 0
fi