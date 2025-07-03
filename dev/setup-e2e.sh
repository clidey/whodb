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

echo "🚀 Setting up complete E2E environment..."

# Get the script directory (so it works from any location)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

echo "📁 Working from project root: $PROJECT_ROOT"

# Setup SQLite
echo "🔧 Setting up SQLite E2E database..."

# Create tmp directory if it doesn't exist
mkdir -p "$PROJECT_ROOT/core/tmp"

# Generate the database
sqlite3 "$PROJECT_ROOT/core/tmp/e2e_test.db" < "$SCRIPT_DIR/sample-data/sqlite3/data.sql"

# Set proper permissions
chmod 644 "$PROJECT_ROOT/core/tmp/e2e_test.db"

echo "✅ SQLite E2E database ready at core/tmp/e2e_test.db"

# Start other database services
echo "🐳 Starting database services..."
cd "$SCRIPT_DIR"
docker-compose -f docker-compose.e2e.yaml up -d

# Wait for services to be ready
echo "⏳ Waiting for services to be ready..."
sleep 10

# Check if services are healthy
echo "🔍 Checking service health..."
for service in e2e_postgres e2e_mysql e2e_mariadb e2e_mongo e2e_clickhouse; do
    if docker ps --filter "name=$service" --filter "status=running" | grep -q $service; then
        echo "✅ $service is running"
    else
        echo "❌ $service failed to start"
    fi
done

echo "🎉 E2E environment setup complete!"