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

# Get the script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

echo "📸 WhoDB Screenshot Generation Environment"
echo "=========================================="
echo ""

# Cleanup function
cleanup() {
    echo ""
    echo "🧹 Cleaning up screenshot environment..."

    # Stop test server
    if [ -f "$PROJECT_ROOT/core/tmp/screenshot-server.pid" ]; then
        TEST_SERVER_PID=$(cat "$PROJECT_ROOT/core/tmp/screenshot-server.pid")
        if ps -p $TEST_SERVER_PID > /dev/null 2>&1; then
            echo "   Stopping test server (PID: $TEST_SERVER_PID)..."
            kill $TEST_SERVER_PID || true
            sleep 2
            # Force kill if still running
            if ps -p $TEST_SERVER_PID > /dev/null 2>&1; then
                kill -9 $TEST_SERVER_PID || true
            fi
        fi
        rm -f "$PROJECT_ROOT/core/tmp/screenshot-server.pid"
    fi

    # Stop frontend dev server
    echo "   Stopping frontend dev server..."
    pkill -f "vite --port 3000" || true

    # Stop Docker services
    echo "   Stopping Docker services..."
    cd "$SCRIPT_DIR"
    docker compose -f docker-compose.screenshot.yaml down || true

    echo "✅ Cleanup complete"
}

# Register cleanup on exit
trap cleanup EXIT

# Step 1: Build test binary
echo "🔨 Building test binary..."
cd "$PROJECT_ROOT/core"

# Always rebuild the test binary to ensure the latest schema changes are included
BINARY_PATH="$PROJECT_ROOT/core/server.test"
echo "   Building test binary..."
go test -coverpkg=./... -c -o server.test ./cmd/whodb
echo "   ✅ Test binary built"

# Step 2: Setup SQLite database
echo ""
echo "🗄️  Setting up SQLite database..."
SQLITE_DB="$PROJECT_ROOT/core/tmp/e2e_test.db"
mkdir -p "$PROJECT_ROOT/core/tmp"

if [ -f "$SQLITE_DB" ]; then
    if sqlite3 "$SQLITE_DB" "SELECT name FROM sqlite_master WHERE type='table' AND name='users';" 2>/dev/null | grep -q users; then
        echo "   SQLite database already initialized"
    else
        echo "   Reinitializing SQLite database..."
        rm -f "$SQLITE_DB"
        sqlite3 "$SQLITE_DB" < "$SCRIPT_DIR/sample-data/sqlite3/data.sql"
        chmod 644 "$SQLITE_DB"
    fi
else
    echo "   Creating SQLite database..."
    sqlite3 "$SQLITE_DB" < "$SCRIPT_DIR/sample-data/sqlite3/data.sql"
    chmod 644 "$SQLITE_DB"
fi
echo "   ✅ SQLite database ready"

# Step 3: Start Docker services
echo ""
echo "🐳 Starting Docker services for screenshots..."
cd "$SCRIPT_DIR"
docker compose -f docker-compose.screenshot.yaml up -d --remove-orphans

# Wait for PostgreSQL to be ready
echo "   Waiting for PostgreSQL to be ready..."
COUNTER=0
MAX_WAIT=60
while [ $COUNTER -lt $MAX_WAIT ]; do
    if nc -z localhost 5432 2>/dev/null; then
        echo "   ✅ PostgreSQL is ready"
        break
    fi
    sleep 1
    COUNTER=$((COUNTER + 1))
done

if [ $COUNTER -ge $MAX_WAIT ]; then
    echo "   ❌ PostgreSQL failed to start within ${MAX_WAIT} seconds"
    exit 1
fi

# Additional wait for PostgreSQL to fully initialize
sleep 5

# Step 4: Start backend server
echo ""
echo "🚀 Starting backend test server..."
cd "$PROJECT_ROOT/core"
ENVIRONMENT=dev WHODB_DISABLE_MOCK_DATA_GENERATION='' \
    ./server.test -test.run=^TestMain$ &
TEST_SERVER_PID=$!
echo $TEST_SERVER_PID > "$PROJECT_ROOT/core/tmp/screenshot-server.pid"

# Wait for server to be ready
echo "   Waiting for backend server..."
COUNTER=0
MAX_WAIT=30
while [ $COUNTER -lt $MAX_WAIT ]; do
    if nc -z localhost 8080 2>/dev/null; then
        echo "   ✅ Backend server is ready (PID: $TEST_SERVER_PID)"
        break
    fi
    sleep 0.5
    COUNTER=$((COUNTER + 1))
done

if [ $COUNTER -ge $MAX_WAIT ]; then
    echo "   ❌ Backend server failed to start within ${MAX_WAIT} seconds"
    exit 1
fi

# Step 5: Start frontend dev server
echo ""
echo "🎨 Starting frontend dev server..."
cd "$PROJECT_ROOT/frontend"
VITE_E2E_TEST=true NODE_ENV=test vite --port 3000 > /dev/null 2>&1 &
FRONTEND_PID=$!

# Wait for frontend to be ready
echo "   Waiting for frontend server..."
COUNTER=0
MAX_WAIT=60
while [ $COUNTER -lt $MAX_WAIT ]; do
    if nc -z localhost 3000 2>/dev/null; then
        echo "   ✅ Frontend server is ready (PID: $FRONTEND_PID)"
        break
    fi
    sleep 1
    COUNTER=$((COUNTER + 1))
done

if [ $COUNTER -ge $MAX_WAIT ]; then
    echo "   ❌ Frontend server failed to start within ${MAX_WAIT} seconds"
    exit 1
fi

# Additional wait for frontend to fully initialize
sleep 3

# Step 6: Run screenshot tests
echo ""
echo "📸 Running screenshot tests..."
echo ""
cd "$PROJECT_ROOT/frontend"

echo "   Test spec: tests/postgres-screenshots.spec.mjs"

cd "$PROJECT_ROOT/frontend/e2e"
NODE_ENV=test pnpm exec playwright test \
    --project=standalone \
    tests/postgres-screenshots.spec.mjs

PW_EXIT_CODE=$?

# Step 7: Display results
echo ""
echo "=========================================="
if [ $PW_EXIT_CODE -eq 0 ]; then
    echo "✅ Screenshot generation completed successfully"
    echo ""
    echo "📁 Screenshots saved to:"
    echo "   $PROJECT_ROOT/frontend/e2e/screenshots/postgres/"
    echo ""
    echo "💡 Tip: You can find all screenshots organized by test number and name"
else
    echo "❌ Screenshot generation failed with exit code: $PW_EXIT_CODE"
    echo ""
    echo "Check the Playwright output above for details"
fi
echo "=========================================="
echo ""

exit $PW_EXIT_CODE
