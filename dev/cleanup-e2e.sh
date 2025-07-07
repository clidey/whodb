#!/bin/bash
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

echo "🧹 Cleaning up complete E2E environment..."

# Get the script directory (so it works from any location)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

echo "📁 Working from project root: $PROJECT_ROOT"

# Cleanup SQLite
echo "🧹 Cleaning up tmp directory..."
if [ -d "$PROJECT_ROOT/core/tmp" ]; then
    rm -rf "$PROJECT_ROOT/core/tmp"
    echo "✅ tmp directory cleaned up"
else
    echo "ℹ️ No tmp directory to clean up"
fi

# Stop and remove Docker services
echo "🐳 Stopping database services..."
cd "$SCRIPT_DIR"
docker-compose -f docker-compose.e2e.yaml down

# Stop the test server if it's running
echo "🛑 Stopping test server..."
if [ -n "$TEST_SERVER_PID" ] && ps -p $TEST_SERVER_PID > /dev/null 2>&1; then
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


# Run the existing cleanup script if it exists
if [ -f "$SCRIPT_DIR/cleanup.sh" ]; then
    echo "🗑️ Running Docker cleanup..."
    chmod +x "$SCRIPT_DIR/cleanup.sh"
    bash "$SCRIPT_DIR/cleanup.sh"
else
    echo "ℹ️ No cleanup.sh found, skipping Docker volume cleanup"
fi

echo "✅ E2E environment cleanup complete!"