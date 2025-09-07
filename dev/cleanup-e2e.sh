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

# Get edition from parameter (default to CE)
EDITION="${1:-ce}"

echo "🧹 Cleaning up $EDITION E2E environment..."

# Get the script directory (so it works from any location)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

echo "📁 Working from project root: $PROJECT_ROOT"

# Cleanup SQLite and tmp directory
echo "🧹 Cleaning up tmp directory..."
if [ -d "$PROJECT_ROOT/core/tmp" ]; then
    rm -rf "$PROJECT_ROOT/core/tmp"
    echo "✅ tmp directory cleaned up"
else
    echo "ℹ️ No tmp directory to clean up"
fi

# Clean up test binary
if [ -f "$PROJECT_ROOT/core/server.test" ]; then
    rm "$PROJECT_ROOT/core/server.test"
    echo "✅ Test binary cleaned up"
fi

# Coverage is already written by the test server to coverage.out
# Just report that it's available
if [ -f "$PROJECT_ROOT/core/coverage.out" ]; then
    echo "✅ Backend coverage saved to core/coverage.out"
fi

# If EE mode, run EE-specific cleanup first (if it exists)
if [ "$EDITION" = "ee" ]; then
    EE_CLEANUP_SCRIPT="$PROJECT_ROOT/ee/dev/cleanup-ee-databases.sh"
    if [ -f "$EE_CLEANUP_SCRIPT" ]; then
        echo "🧹 Running EE-specific cleanup..."
        bash "$EE_CLEANUP_SCRIPT"
    fi
fi

# Stop and remove CE Docker services
echo "🐳 Stopping CE database services..."
cd "$SCRIPT_DIR"
# Use --volumes to ensure volumes are removed, and --remove-orphans for cleanup
# The --timeout 0 forces immediate stop without graceful shutdown
docker-compose -f docker-compose.e2e.yaml down --volumes --remove-orphans --timeout 0

# Force prune any dangling volumes to ensure complete cleanup
echo "🔄 Pruning any dangling volumes..."
docker volume prune -f

# Stop the test server if it's running
echo "🛑 Stopping test server..."

# Try to read PID from file first
if [ -f "$PROJECT_ROOT/core/tmp/test-server.pid" ]; then
    TEST_SERVER_PID=$(cat "$PROJECT_ROOT/core/tmp/test-server.pid")
    if ps -p $TEST_SERVER_PID > /dev/null 2>&1; then
        kill $TEST_SERVER_PID
        # Wait for process to finish and write coverage
        sleep 2
        echo "✅ Test server stopped (PID: $TEST_SERVER_PID)"
    fi
    rm -f "$PROJECT_ROOT/core/tmp/test-server.pid"
elif [ -n "$TEST_SERVER_PID" ] && ps -p $TEST_SERVER_PID > /dev/null 2>&1; then
    kill $TEST_SERVER_PID
    echo "✅ Test server stopped (PID: $TEST_SERVER_PID)"
else
    # Try to find and kill any running server.test processes
    PIDS=$(pgrep -f "server.test" 2>/dev/null || true)
    if [ -n "$PIDS" ]; then
        echo "🔄 Found running server.test processes, stopping them..."
        echo $PIDS | xargs kill
        echo "✅ All server.test processes stopped"
    else
        echo "ℹ️ No test server processes found"
    fi
fi



# Kill anything still on port 3000 (just in case)
echo "🔍 Ensuring port 3000 is free..."
lsof -ti:3000 | xargs kill -9 2>/dev/null || true

# Kill anything still on port 8080 (just in case)
echo "🔍 Ensuring port 8080 is free..."
lsof -ti:8080 | xargs kill -9 2>/dev/null || true

echo "✅ $EDITION E2E environment cleanup complete!"