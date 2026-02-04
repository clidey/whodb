#!/bin/bash
#
# Copyright 2026 Clidey, Inc.
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

# Get the script directory (so it works from any location)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

echo "Setting up CLI E2E test environment..."
echo "Working from project root: $PROJECT_ROOT"

# Run cleanup first to ensure clean state
echo "Running cleanup first..."
bash "$SCRIPT_DIR/cleanup-cli-e2e.sh" 2>/dev/null || true

# Start PostgreSQL
echo "Starting PostgreSQL..."
cd "$SCRIPT_DIR"
docker-compose -f docker-compose.yml up -d e2e_postgres

# Wait for PostgreSQL to be ready
echo "Waiting for PostgreSQL..."
COUNTER=0
MAX_WAIT=60
while [ $COUNTER -lt $MAX_WAIT ]; do
    if nc -z localhost 5432 2>/dev/null; then
        echo "PostgreSQL is ready"
        break
    fi
    sleep 1
    COUNTER=$((COUNTER + 1))
done

if [ $COUNTER -ge $MAX_WAIT ]; then
    echo "PostgreSQL failed to start within ${MAX_WAIT}s"
    exit 1
fi

# Build CLI binary
echo "Building CLI..."
cd "$PROJECT_ROOT/cli"
go build -o whodb-cli .
echo "CLI built at cli/whodb-cli"

echo "CLI E2E setup complete!"
