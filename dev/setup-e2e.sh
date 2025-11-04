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

# Get edition from parameter (default to CE)
EDITION="${1:-ce}"

# Check if this is EE-only mode (passed from run-cypress.sh)
if [ "$EDITION" = "ee-only" ]; then
    SKIP_CE_DATABASES="true"
    EDITION="ee"  # Use ee for everything else
else
    SKIP_CE_DATABASES="false"
fi

# Get the script directory (so it works from any location)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

echo "📁 Working from project root: $PROJECT_ROOT"
echo "🔧 Setting up $EDITION E2E environment..."


# Run cleanup first to ensure clean state
echo "🧹 Running cleanup first..."
if [ -f "$SCRIPT_DIR/cleanup-e2e.sh" ]; then
    bash "$SCRIPT_DIR/cleanup-e2e.sh"
else
    echo "⚠️ cleanup-e2e.sh not found, continuing without cleanup"
fi

# Build test binary with coverage (with smart caching)
BINARY_PATH="$PROJECT_ROOT/core/server.test"
HASH_FILE="$PROJECT_ROOT/core/tmp/.test-binary-hash"
mkdir -p "$PROJECT_ROOT/core/tmp"

# Allow force rebuild via environment variable
if [ "${FORCE_REBUILD:-false}" = "true" ]; then
    echo "🔨 Force rebuild requested (FORCE_REBUILD=true)"
    rm -f "$BINARY_PATH" "$HASH_FILE"
fi

# Calculate hash of source files
calculate_source_hash() {
    if [ "$EDITION" = "ee" ]; then
        find "$PROJECT_ROOT/core" "$PROJECT_ROOT/ee" -name "*.go" -type f -exec md5sum {} \; | sort | md5sum | cut -d' ' -f1
    else
        find "$PROJECT_ROOT/core" -name "*.go" -type f -exec md5sum {} \; | sort | md5sum | cut -d' ' -f1
    fi
}

CURRENT_HASH=$(calculate_source_hash)
NEEDS_REBUILD=true

# Check if we can skip rebuild
if [ -f "$BINARY_PATH" ] && [ -f "$HASH_FILE" ]; then
    STORED_HASH=$(cat "$HASH_FILE")
    if [ "$CURRENT_HASH" = "$STORED_HASH" ]; then
        echo "✅ Using cached test binary - NO REBUILD NEEDED"
        echo "   Previous hash: ${STORED_HASH:0:8}..."
        echo "   Current hash:  ${CURRENT_HASH:0:8}... (matches)"
        echo "   Binary path:   $BINARY_PATH"
        echo "   Last modified: $(date -r "$BINARY_PATH" '+%Y-%m-%d %H:%M:%S')"
        NEEDS_REBUILD=false
    else
        echo "🔄 Source files changed - REBUILD REQUIRED"
        echo "   Previous hash: ${STORED_HASH:0:8}..."
        echo "   Current hash:  ${CURRENT_HASH:0:8}... (different)"
    fi
else
    if [ ! -f "$BINARY_PATH" ]; then
        echo "🔨 Test binary not found - BUILD REQUIRED"
    else
        echo "🔨 Hash file missing - BUILD REQUIRED"
    fi
fi

if [ "$NEEDS_REBUILD" = "true" ]; then
    if [ "$EDITION" = "ee" ]; then
        # Check if EE directory exists
        if [ ! -d "$PROJECT_ROOT/ee" ]; then
            echo "❌ EE directory not found. Cannot run EE tests."
            exit 1
        fi
        echo "🔧 Building EE test binary with coverage..."
        cd "$PROJECT_ROOT/core"
        GOWORK="$PROJECT_ROOT/go.work.ee" go test -tags ee -coverpkg=./...,../ee/... -c -o server.test
        echo "✅ EE test binary built successfully"
    else
        echo "🔧 Building CE test binary with coverage..."
        cd "$PROJECT_ROOT/core"
        go test -coverpkg=./... -c -o server.test
        echo "✅ CE test binary built successfully"
    fi
    # Store hash for next run
    echo "$CURRENT_HASH" > "$HASH_FILE"
fi


# Setup SQLite (with smart initialization check)
SQLITE_DB="$PROJECT_ROOT/core/tmp/e2e_test.db"
SQLITE_NEEDS_INIT=true

# Check if SQLite database exists and has the expected tables
if [ -f "$SQLITE_DB" ]; then
    # Check if database has expected structure (check for a key table)
    if sqlite3 "$SQLITE_DB" "SELECT name FROM sqlite_master WHERE type='table' AND name='users';" 2>/dev/null | grep -q users; then
        echo "✅ SQLite E2E database already initialized, skipping setup"
        SQLITE_NEEDS_INIT=false
    else
        echo "⚠️ SQLite database exists but is incomplete, reinitializing..."
        rm -f "$SQLITE_DB"
    fi
fi

if [ "$SQLITE_NEEDS_INIT" = "true" ]; then
    echo "🔧 Setting up SQLite E2E database..."

    # Create tmp directory if it doesn't exist
    mkdir -p "$PROJECT_ROOT/core/tmp"

    # Generate the database
    sqlite3 "$SQLITE_DB" < "$SCRIPT_DIR/sample-data/sqlite3/data.sql"

    # Set proper permissions
    chmod 644 "$SQLITE_DB"

    echo "✅ SQLite E2E database ready at core/tmp/e2e_test.db"
fi

# Start CE database services (skip if EE-only mode)
if [ "$SKIP_CE_DATABASES" = "false" ]; then
    echo "🐳 Preparing Docker services..."
    cd "$SCRIPT_DIR"

    # Pre-pull images in parallel to avoid delays during startup
    echo "📦 Ensuring Docker images are available..."
    docker-compose -f docker-compose.e2e.yaml pull --quiet 2>/dev/null || true

    echo "🚀 Starting CE database services..."
    # Start containers (docker-compose v2+ starts in parallel by default)
    docker-compose -f docker-compose.e2e.yaml up -d --remove-orphans

    # Wait for services using parallel port checks
    echo "⏳ Waiting for services to be ready..."

    # Simple function to wait for a service by checking its port
    wait_for_port() {
        local service=$1
        local port=$2
        local max_wait=${3:-60}  # Allow custom timeout, default 60s
        local counter=0

        while [ $counter -lt $max_wait ]; do
            if nc -z localhost $port 2>/dev/null; then
                echo "✅ $service is ready (port $port)"
                return 0
            fi
            sleep 1
            counter=$((counter + 1))
        done
        echo "⚠️ $service timeout after ${max_wait}s (port $port)"
        return 1
    }

    # Start all checks in parallel - simple port checks
    wait_for_port "PostgreSQL" 5432 90 &  # Heavy init script with tables
    PID_PG=$!
    wait_for_port "MySQL" 3306 90 &  # Heavy init script with tables
    PID_MYSQL=$!
    wait_for_port "MySQL8" 3308 90 &  # Heavy init script with tables
    PID_MYSQL8=$!
    wait_for_port "MariaDB" 3307 90 &  # Heavy init script with tables
    PID_MARIA=$!
    wait_for_port "MongoDB" 27017 30 &  # Light init script
    PID_MONGO=$!
    wait_for_port "ClickHouse" 8123 30 &  # Quick startup
    PID_CH=$!
    wait_for_port "Redis" 6379 20 &  # Very fast startup
    PID_REDIS=$!
    wait_for_port "ElasticSearch" 9200 60 &  # Can be slow to start
    PID_ES=$!

    # Wait for all background processes
    echo "⏳ Waiting for all services to be ready in parallel..."
    FAILED=false
    for pid in $PID_PG $PID_MYSQL $PID_MYSQL8 $PID_MARIA $PID_MONGO $PID_CH $PID_REDIS $PID_ES; do
        if ! wait $pid; then
            FAILED=true
        fi
    done

    if [ "$FAILED" = "true" ]; then
        echo "⚠️ Some services failed to start, but continuing..."
    else
        echo "✅ All services are ready!"
    fi
else
    echo "⏭️ Skipping CE database services (EE-only mode)"
fi

# If EE mode, run EE-specific setup (if it exists)
if [ "$EDITION" = "ee" ]; then
    EE_SETUP_SCRIPT="$PROJECT_ROOT/ee/dev/setup-ee-databases.sh"
    if [ -f "$EE_SETUP_SCRIPT" ]; then
        echo "🔧 Running EE-specific setup..."
        bash "$EE_SETUP_SCRIPT"
    else
        echo "⚠️ EE setup script not found, continuing with CE only"
    fi
fi

# Clean up coverage files in parallel
echo "🧹 Cleaning previous coverage artifacts..."
(
    # Backend coverage
    COVERAGE_FILE="$PROJECT_ROOT/core/coverage.out"
    [ -f "$COVERAGE_FILE" ] && rm -f "$COVERAGE_FILE"
) &
(
    # Frontend coverage
    [ -d "$PROJECT_ROOT/frontend/.nyc_output" ] && rm -rf "$PROJECT_ROOT/frontend/.nyc_output"
    [ -d "$PROJECT_ROOT/frontend/coverage" ] && rm -rf "$PROJECT_ROOT/frontend/coverage"
) &
wait
echo "✅ Coverage cleanup complete"

# Start the CE test server with coverage
echo "🚀 Starting CE test server with coverage..."
cd "$PROJECT_ROOT/core"
# Increase connection limits for parallel tests
GOMAXPROCS=4 ENVIRONMENT=dev WHODB_DISABLE_MOCK_DATA_GENERATION='orders,DEPARTMENTS' \
    ./server.test -test.run=^TestMain$ -test.coverprofile=coverage.out &
TEST_SERVER_PID=$!

# Save PID for cleanup
echo $TEST_SERVER_PID > "$PROJECT_ROOT/core/tmp/test-server.pid"

# Wait for server to be ready with active health check
echo "⏳ Waiting for test server to be ready..."
if [ "$EDITION" = "ee" ]; then
    MAX_WAIT=30  # Usually starts in 5-10s
else
    MAX_WAIT=20  # Usually starts in 3-5s
fi
COUNTER=0
while [ $COUNTER -lt $MAX_WAIT ]; do
    # Check if port 8080 is listening
    if nc -z localhost 8080 2>/dev/null; then
        echo "✅ Test server is ready and listening on port 8080 (PID: $TEST_SERVER_PID)"
        break
    fi
    # Frequent polling for quick detection
    sleep 0.2
    COUNTER=$((COUNTER + 1))
done

if [ $COUNTER -ge $MAX_WAIT ]; then
    echo "❌ Test server failed to become ready within ${MAX_WAIT} seconds"
    if ps -p $TEST_SERVER_PID > /dev/null; then
        echo "Server process is running but not responding. Check logs for errors."
        kill $TEST_SERVER_PID
    fi
    exit 1
fi

echo "🎉 $EDITION E2E backend environment setup complete!"
echo "ℹ️  Frontend will be started by the test script"