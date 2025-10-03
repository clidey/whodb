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

set -e

# Default URLs
BACKEND_URL="${BACKEND_URL:-http://localhost:8080}"
FRONTEND_URL="${FRONTEND_URL:-http://localhost:3000}"
MAX_WAIT="${MAX_WAIT:-60}"

echo "⏳ Waiting for services to be ready..."
echo "   Backend:  $BACKEND_URL"
echo "   Frontend: $FRONTEND_URL"
echo "   Timeout:  ${MAX_WAIT}s"

# Function to check if a port is listening
check_port() {
    local host=$1
    local port=$2
    nc -z "$host" "$port" 2>/dev/null
}

# Extract host and port from URLs
BACKEND_HOST=$(echo "$BACKEND_URL" | sed -e 's|^[^/]*//||' -e 's|:.*||')
BACKEND_PORT=$(echo "$BACKEND_URL" | sed -e 's|.*:||' -e 's|/.*||')
BACKEND_HOST=${BACKEND_HOST:-localhost}
BACKEND_PORT=${BACKEND_PORT:-8080}

# Extract host and port from frontend URL
FRONTEND_HOST=$(echo "$FRONTEND_URL" | sed -e 's|^[^/]*//||' -e 's|:.*||')
FRONTEND_PORT=$(echo "$FRONTEND_URL" | sed -e 's|.*:||' -e 's|/.*||')
FRONTEND_HOST=${FRONTEND_HOST:-localhost}
FRONTEND_PORT=${FRONTEND_PORT:-3000}

# Function to wait for a specific port
wait_for_port_async() {
    local name=$1
    local host=$2
    local port=$3
    local max_wait=$4
    local counter=0

    echo "⏳ Waiting for $name ($host:$port)..."
    while [ $counter -lt $max_wait ]; do
        if check_port "$host" "$port"; then
            echo "✅ $name is ready!"
            return 0
        fi
        sleep 1
        counter=$((counter + 1))
    done
    echo "❌ $name timeout after ${max_wait}s"
    return 1
}

# Start parallel waits
wait_for_port_async "Backend" "$BACKEND_HOST" "$BACKEND_PORT" "$MAX_WAIT" &
PID_BACKEND=$!

wait_for_port_async "Frontend" "$FRONTEND_HOST" "$FRONTEND_PORT" "$MAX_WAIT" &
PID_FRONTEND=$!

# Wait for both processes
FAILED=false
if ! wait $PID_BACKEND; then
    echo "Backend failed to start within ${MAX_WAIT} seconds"
    FAILED=true
fi

if ! wait $PID_FRONTEND; then
    echo "Frontend failed to start within ${MAX_WAIT} seconds"
    FAILED=true
fi

if [ "$FAILED" = "true" ]; then
    exit 1
fi

echo "🎉 All services are ready!"