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

# Cleanup SQLite and tmp directory (but preserve hash file for caching)
echo "🧹 Cleaning up tmp directory..."
if [ -d "$PROJECT_ROOT/core/tmp" ]; then
    # Clean specific files but preserve the hash file
    rm -f "$PROJECT_ROOT/core/tmp/e2e_test.db"
    rm -f "$PROJECT_ROOT/core/tmp/test-server.pid"
    # Only delete other files, keep .test-binary-hash
    find "$PROJECT_ROOT/core/tmp" -type f ! -name '.test-binary-hash' -delete 2>/dev/null || true
    echo "✅ tmp directory cleaned (preserved hash cache)"
else
    echo "ℹ️ No tmp directory to clean up"
fi

# Keep test binary for caching (comment out deletion)
# The binary will be rebuilt only when source changes are detected
if [ -f "$PROJECT_ROOT/core/server.test" ]; then
    echo "ℹ️  Test binary preserved for caching: server.test"
    echo "   Size: $(du -h "$PROJECT_ROOT/core/server.test" | cut -f1)"
    echo "   To force rebuild, delete: rm $PROJECT_ROOT/core/server.test"
fi

# Coverage is already written by the test server to coverage.out
# Just report that it's available
if [ -f "$PROJECT_ROOT/core/coverage.out" ]; then
    echo "✅ Backend coverage saved to core/coverage.out"
fi

# Clean up frontend coverage artifacts for next run
echo "🧹 Cleaning frontend coverage artifacts..."
if [ -d "$PROJECT_ROOT/frontend/.nyc_output" ]; then
    # Save the coverage report before cleaning if it exists
    if [ -f "$PROJECT_ROOT/frontend/.nyc_output/out.json" ]; then
        echo "✅ Frontend coverage data preserved in .nyc_output/out.json"
    fi
fi
# Note: We keep the coverage artifacts for review but they will be cleared on next setup

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

# First, force remove all containers (even if they're not running)
docker-compose -f docker-compose.e2e.yaml rm -f -s -v 2>/dev/null || true

# Then do a full teardown with volumes
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