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
MAX_WAIT="${MAX_WAIT:-120}"

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

# Wait for backend
echo -n "⏳ Waiting for backend (${BACKEND_HOST}:${BACKEND_PORT})..."
COUNTER=0
while [ $COUNTER -lt $MAX_WAIT ]; do
    if check_port "$BACKEND_HOST" "$BACKEND_PORT"; then
        echo " ✅ Ready!"
        break
    fi
    echo -n "."
    sleep 1
    COUNTER=$((COUNTER + 1))
done

if [ $COUNTER -eq $MAX_WAIT ]; then
    echo " ❌ Timeout!"
    echo "Backend failed to start within ${MAX_WAIT} seconds"
    exit 1
fi

# Extract host and port from frontend URL
FRONTEND_HOST=$(echo "$FRONTEND_URL" | sed -e 's|^[^/]*//||' -e 's|:.*||')
FRONTEND_PORT=$(echo "$FRONTEND_URL" | sed -e 's|.*:||' -e 's|/.*||')
FRONTEND_HOST=${FRONTEND_HOST:-localhost}
FRONTEND_PORT=${FRONTEND_PORT:-3000}

# Wait for frontend
echo -n "⏳ Waiting for frontend (${FRONTEND_HOST}:${FRONTEND_PORT})..."
COUNTER=0
while [ $COUNTER -lt $MAX_WAIT ]; do
    if check_port "$FRONTEND_HOST" "$FRONTEND_PORT"; then
        echo " ✅ Ready!"
        break
    fi
    echo -n "."
    sleep 1
    COUNTER=$((COUNTER + 1))
done

if [ $COUNTER -eq $MAX_WAIT ]; then
    echo " ❌ Timeout!"
    echo "Frontend failed to start within ${MAX_WAIT} seconds"
    exit 1
fi

# Additional wait for stabilization
echo "⏳ Waiting for services to stabilize..."
sleep 3

echo "🎉 All services are ready!"