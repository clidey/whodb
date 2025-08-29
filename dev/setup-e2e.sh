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

# Build test binary with coverage
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


# Setup SQLite
echo "🔧 Setting up SQLite E2E database..."

# Create tmp directory if it doesn't exist
mkdir -p "$PROJECT_ROOT/core/tmp"

# Generate the database
sqlite3 "$PROJECT_ROOT/core/tmp/e2e_test.db" < "$SCRIPT_DIR/sample-data/sqlite3/data.sql"

# Set proper permissions
chmod 644 "$PROJECT_ROOT/core/tmp/e2e_test.db"

echo "✅ SQLite E2E database ready at core/tmp/e2e_test.db"

# Start CE database services
echo "🐳 Starting CE database services..."
cd "$SCRIPT_DIR"
docker-compose -f docker-compose.e2e.yaml up -d

# Wait for services to be ready
echo "⏳ Waiting for services to be ready..."
sleep 15

# Check if CE services are healthy
echo "🔍 Checking CE service health..."
for service in e2e_postgres e2e_mysql e2e_mariadb e2e_mongo e2e_clickhouse e2e_redis e2e_elasticsearch; do
    if docker ps --filter "name=$service" --filter "status=running" | grep -q $service; then
        echo "✅ $service is running"
    else
        echo "⚠️ $service may not be running (some services are optional)"
    fi
done

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

# Start the CE test server with coverage
echo "🚀 Starting CE test server with coverage..."
cd "$PROJECT_ROOT/core"
ENVIRONMENT=dev ./server.test -test.run=^TestMain$ -test.coverprofile=coverage.out &
TEST_SERVER_PID=$!

# Save PID for cleanup
echo $TEST_SERVER_PID > "$PROJECT_ROOT/core/tmp/test-server.pid"

# Wait for server to be ready with health check
echo "⏳ Waiting for test server to be ready..."
if [ "$EDITION" = "ee" ]; then
    MAX_WAIT=60  # More time for EE server startup
else
    MAX_WAIT=45  # More time than before for CE too
fi
COUNTER=0
while [ $COUNTER -lt $MAX_WAIT ]; do
    # Check if port 8080 is listening
    if nc -z localhost 8080 2>/dev/null; then
        echo "✅ Test server is ready and listening on port 8080 (PID: $TEST_SERVER_PID)"
        break
    fi
    echo -n "."
    sleep 1
    COUNTER=$((COUNTER + 1))
done

if [ $COUNTER -eq $MAX_WAIT ]; then
    echo "❌ Test server failed to become ready within ${MAX_WAIT} seconds"
    if ps -p $TEST_SERVER_PID > /dev/null; then
        echo "Server process is running but not responding. Check logs for errors."
        kill $TEST_SERVER_PID
    fi
    exit 1
fi

echo "🎉 $EDITION E2E backend environment setup complete!"
echo "ℹ️  Frontend will be started by the test script"