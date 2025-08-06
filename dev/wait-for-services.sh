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

# Function to check if a URL is responding
check_url() {
    local url=$1
    curl -s -o /dev/null -w "%{http_code}" "$url" 2>/dev/null || echo "000"
}

# Wait for backend
echo -n "⏳ Waiting for backend..."
COUNTER=0
while [ $COUNTER -lt $MAX_WAIT ]; do
    STATUS=$(check_url "$BACKEND_URL/graphql")
    if [ "$STATUS" = "200" ] || [ "$STATUS" = "405" ] || [ "$STATUS" = "400" ]; then
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

# Wait for frontend
echo -n "⏳ Waiting for frontend..."
COUNTER=0
while [ $COUNTER -lt $MAX_WAIT ]; do
    STATUS=$(check_url "$FRONTEND_URL")
    if [ "$STATUS" = "200" ] || [ "$STATUS" = "304" ]; then
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