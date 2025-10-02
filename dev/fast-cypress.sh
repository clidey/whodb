#!/bin/bash
#
# Copyright 2025 Clidey, Inc.
#
# Fast-track Cypress runner for testing specific databases only
# Usage: ./fast-cypress.sh [ce|ee] [database1,database2,...] [headless]
# Example: ./fast-cypress.sh ce postgres,mysql true
#          ./fast-cypress.sh ee redis false
#

set -e

EDITION="${1:-ce}"
DATABASES="${2:-all}"
HEADLESS="${3:-false}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

echo "🚀 Fast-track Cypress E2E Testing"
echo "   Edition: $EDITION"
echo "   Databases: $DATABASES"
echo "   Headless: $HEADLESS"

# Cleanup function
cleanup() {
    echo "🧹 Cleaning up test environment..."
    pkill -TERM -f 'vite --port 3000' 2>/dev/null || true
    sleep 1

    # Only cleanup specified databases
    if [ "$DATABASES" != "all" ]; then
        for db in $(echo $DATABASES | tr ',' ' '); do
            case $db in
                postgres) docker stop e2e_postgres 2>/dev/null || true ;;
                mysql) docker stop e2e_mysql 2>/dev/null || true ;;
                mysql8) docker stop e2e_mysql_842 2>/dev/null || true ;;
                mariadb) docker stop e2e_mariadb 2>/dev/null || true ;;
                mongo) docker stop e2e_mongo 2>/dev/null || true ;;
                clickhouse) docker stop e2e_clickhouse clickhouse-init 2>/dev/null || true ;;
                redis) docker stop e2e_redis redis-init 2>/dev/null || true ;;
                elasticsearch) docker stop e2e_elasticsearch elasticsearch-init 2>/dev/null || true ;;
            esac
        done
    else
        bash "$SCRIPT_DIR/cleanup-e2e.sh" "$EDITION"
    fi

    # Kill test server
    if [ -f "$PROJECT_ROOT/core/tmp/test-server.pid" ]; then
        kill $(cat "$PROJECT_ROOT/core/tmp/test-server.pid") 2>/dev/null || true
        rm -f "$PROJECT_ROOT/core/tmp/test-server.pid"
    fi

    echo "✅ Cleanup complete"
}

trap 'cleanup; exit ${EXIT_CODE:-0}' EXIT INT TERM

# Start only specified databases
if [ "$DATABASES" != "all" ]; then
    echo "🐳 Starting only specified databases: $DATABASES"
    cd "$SCRIPT_DIR"

    SERVICES_TO_START=""
    CYPRESS_SPECS=""

    for db in $(echo $DATABASES | tr ',' ' '); do
        case $db in
            sqlite)
                # SQLite is always available
                CYPRESS_SPECS="$CYPRESS_SPECS,cypress/e2e/sqlite.cy.js"
                ;;
            postgres)
                SERVICES_TO_START="$SERVICES_TO_START e2e_postgres"
                CYPRESS_SPECS="$CYPRESS_SPECS,cypress/e2e/postgres.cy.js"
                ;;
            mysql)
                SERVICES_TO_START="$SERVICES_TO_START e2e_mysql"
                CYPRESS_SPECS="$CYPRESS_SPECS,cypress/e2e/mysql.cy.js"
                ;;
            mysql8)
                SERVICES_TO_START="$SERVICES_TO_START e2e_mysql_842"
                CYPRESS_SPECS="$CYPRESS_SPECS,cypress/e2e/mysql8.cy.js"
                ;;
            mariadb)
                SERVICES_TO_START="$SERVICES_TO_START e2e_mariadb"
                CYPRESS_SPECS="$CYPRESS_SPECS,cypress/e2e/mariadb.cy.js"
                ;;
            mongo)
                SERVICES_TO_START="$SERVICES_TO_START e2e_mongo"
                CYPRESS_SPECS="$CYPRESS_SPECS,cypress/e2e/mongo.cy.js"
                ;;
            clickhouse)
                SERVICES_TO_START="$SERVICES_TO_START e2e_clickhouse clickhouse-init"
                CYPRESS_SPECS="$CYPRESS_SPECS,cypress/e2e/clickhouse.cy.js"
                ;;
            redis)
                SERVICES_TO_START="$SERVICES_TO_START e2e_redis redis-init"
                CYPRESS_SPECS="$CYPRESS_SPECS,cypress/e2e/redis.cy.js"
                ;;
            elasticsearch)
                SERVICES_TO_START="$SERVICES_TO_START e2e_elasticsearch elasticsearch-init"
                CYPRESS_SPECS="$CYPRESS_SPECS,cypress/e2e/elasticsearch.cy.js"
                ;;
            *)
                echo "⚠️ Unknown database: $db"
                ;;
        esac
    done

    # Remove leading comma
    CYPRESS_SPECS="${CYPRESS_SPECS#,}"

    if [ -n "$SERVICES_TO_START" ]; then
        docker-compose -f docker-compose.e2e.yaml up -d $SERVICES_TO_START

        # Quick parallel wait for started services
        echo "⏳ Waiting for database services..."
        for service in $SERVICES_TO_START; do
            case $service in
                e2e_postgres) (while ! nc -z localhost 5432 2>/dev/null; do sleep 0.5; done && echo "✅ PostgreSQL ready") &;;
                e2e_mysql) (while ! nc -z localhost 3306 2>/dev/null; do sleep 0.5; done && echo "✅ MySQL ready") &;;
                e2e_mysql_842) (while ! nc -z localhost 3308 2>/dev/null; do sleep 0.5; done && echo "✅ MySQL8 ready") &;;
                e2e_mariadb) (while ! nc -z localhost 3307 2>/dev/null; do sleep 0.5; done && echo "✅ MariaDB ready") &;;
                e2e_mongo) (while ! nc -z localhost 27017 2>/dev/null; do sleep 0.5; done && echo "✅ MongoDB ready") &;;
                e2e_clickhouse) (while ! nc -z localhost 8123 2>/dev/null; do sleep 0.5; done && echo "✅ ClickHouse ready") &;;
                e2e_redis) (while ! nc -z localhost 6379 2>/dev/null; do sleep 0.5; done && echo "✅ Redis ready") &;;
                e2e_elasticsearch) (while ! nc -z localhost 9200 2>/dev/null; do sleep 0.5; done && echo "✅ ElasticSearch ready") &;;
            esac
        done
        wait
    fi
else
    # Run full setup for all databases
    CYPRESS_SPECS=""
    bash "$SCRIPT_DIR/setup-e2e.sh" "$EDITION" || { echo "❌ Backend setup failed"; exit 1; }
fi

# Setup SQLite if needed
if echo "$DATABASES" | grep -qE "(sqlite|all)"; then
    SQLITE_DB="$PROJECT_ROOT/core/tmp/e2e_test.db"
    if [ ! -f "$SQLITE_DB" ]; then
        echo "🔧 Setting up SQLite..."
        mkdir -p "$PROJECT_ROOT/core/tmp"
        sqlite3 "$SQLITE_DB" < "$SCRIPT_DIR/sample-data/sqlite3/data.sql"
        chmod 644 "$SQLITE_DB"
    fi
fi

# Start backend server (reuse cached binary if available)
echo "🚀 Starting test server..."
cd "$PROJECT_ROOT/core"
if [ ! -f "server.test" ]; then
    echo "🔧 Building test binary..."
    if [ "$EDITION" = "ee" ]; then
        GOWORK="$PROJECT_ROOT/go.work.ee" go test -tags ee -coverpkg=./...,../ee/... -c -o server.test
    else
        go test -coverpkg=./... -c -o server.test
    fi
fi

ENVIRONMENT=dev WHODB_DISABLE_MOCK_DATA_GENERATION='orders,DEPARTMENTS' ./server.test -test.run=^TestMain$ -test.coverprofile=coverage.out &
TEST_SERVER_PID=$!
echo $TEST_SERVER_PID > "$PROJECT_ROOT/core/tmp/test-server.pid"

# Wait for backend
while ! nc -z localhost 8080 2>/dev/null; do sleep 0.5; done
echo "✅ Backend ready"

# Start frontend
echo "🚀 Starting frontend..."
cd "$PROJECT_ROOT/frontend"
if [ "$EDITION" = "ee" ]; then
    VITE_BUILD_EDITION=ee NODE_ENV=test vite --port 3000 --clearScreen false --logLevel error &
else
    NODE_ENV=test vite --port 3000 --clearScreen false --logLevel error &
fi

# Wait for frontend
while ! nc -z localhost 3000 2>/dev/null; do sleep 0.5; done
echo "✅ Frontend ready"

# Run Cypress with specific specs if provided
echo "🧪 Running Cypress tests..."
cd "$PROJECT_ROOT/frontend"

if [ "$HEADLESS" = "true" ]; then
    if [ -n "$CYPRESS_SPECS" ]; then
        NODE_ENV=test npx cypress run --spec "$CYPRESS_SPECS" --browser chromium
    else
        NODE_ENV=test npx cypress run --browser chromium
    fi
else
    if [ -n "$CYPRESS_SPECS" ]; then
        NODE_ENV=test npx cypress open --config "specPattern=$CYPRESS_SPECS"
    else
        NODE_ENV=test pnpm cypress open
    fi
fi

EXIT_CODE=$?
echo "✅ Test run complete"