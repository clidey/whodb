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

# Get the script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
CLI_DIR="$PROJECT_ROOT/cli"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Track results
UNIT_RESULT=0
SQLITE_RESULT=0
CLI_RESULT=0
POSTGRES_RESULT=0

print_header() {
    echo ""
    echo "========================================"
    echo "$1"
    echo "========================================"
}

print_result() {
    if [ $2 -eq 0 ]; then
        echo -e "${GREEN}✓ $1 PASSED${NC}"
    else
        echo -e "${RED}✗ $1 FAILED${NC}"
    fi
}

# Parse arguments
RUN_POSTGRES=true
VERBOSE=""
for arg in "$@"; do
    case $arg in
        --skip-postgres)
            RUN_POSTGRES=false
            ;;
        -v|--verbose)
            VERBOSE="-v"
            ;;
        -h|--help)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Run all CLI tests (unit, SQLite E2E, CLI E2E, PostgreSQL E2E)"
            echo ""
            echo "Options:"
            echo "  --skip-postgres  Skip PostgreSQL E2E tests"
            echo "  -v, --verbose    Show verbose test output"
            echo "  -h, --help       Show this help message"
            exit 0
            ;;
    esac
done

cd "$CLI_DIR"

# Clear test cache to ensure all tests actually run
echo "Clearing test cache..."
go clean -testcache

# 1. Unit Tests
print_header "Running Unit Tests"
if go test $VERBOSE ./internal/... 2>&1; then
    UNIT_RESULT=0
else
    UNIT_RESULT=1
fi

# 2. SQLite E2E Tests (no binary needed)
print_header "Running SQLite E2E Tests"
if go test $VERBOSE ./e2e/... -run "^TestEndToEnd" 2>&1; then
    SQLITE_RESULT=0
else
    SQLITE_RESULT=1
fi

# 3. CLI E2E Tests (needs binary)
print_header "Building CLI Binary"
go build -o whodb-cli .
echo "CLI built at cli/whodb-cli"

print_header "Running CLI E2E Tests"
if go test -tags=e2e_cli $VERBOSE ./e2e/... -run "^TestCLI_" 2>&1; then
    CLI_RESULT=0
else
    CLI_RESULT=1
fi

# Clean up binary
rm -f whodb-cli

# 4. PostgreSQL E2E Tests (optional, requires Docker)
if [ "$RUN_POSTGRES" = true ]; then
    print_header "Running PostgreSQL E2E Tests"

    # Check if Docker is available
    if ! command -v docker &> /dev/null; then
        echo -e "${YELLOW}⚠ Docker not found, skipping PostgreSQL tests${NC}"
        POSTGRES_RESULT=-1
    else
        # Setup PostgreSQL
        echo "Starting PostgreSQL..."
        cd "$SCRIPT_DIR"
        bash cleanup-cli-e2e.sh 2>/dev/null || true
        docker-compose -f docker-compose.yml up -d e2e_postgres

        # Wait for PostgreSQL
        echo "Waiting for PostgreSQL..."
        COUNTER=0
        while [ $COUNTER -lt 60 ]; do
            if nc -z localhost 5432 2>/dev/null; then
                echo "PostgreSQL is ready"
                break
            fi
            sleep 1
            COUNTER=$((COUNTER + 1))
        done

        if [ $COUNTER -ge 60 ]; then
            echo -e "${YELLOW}⚠ PostgreSQL did not start in time, skipping tests${NC}"
            POSTGRES_RESULT=-1
        else
            # Build CLI and run tests
            cd "$CLI_DIR"
            go build -o whodb-cli .

            if go test -tags=e2e_postgres $VERBOSE ./e2e/... -run "^TestPostgres_" 2>&1; then
                POSTGRES_RESULT=0
            else
                POSTGRES_RESULT=1
            fi

            rm -f whodb-cli
        fi

        # Cleanup PostgreSQL
        cd "$SCRIPT_DIR"
        bash cleanup-cli-e2e.sh 2>/dev/null || true
    fi
fi

# Summary
print_header "Test Results Summary"
print_result "Unit Tests" $UNIT_RESULT
print_result "SQLite E2E Tests" $SQLITE_RESULT
print_result "CLI E2E Tests" $CLI_RESULT

if [ "$RUN_POSTGRES" = true ]; then
    if [ $POSTGRES_RESULT -eq -1 ]; then
        echo -e "${YELLOW}⚠ PostgreSQL E2E Tests SKIPPED${NC}"
    else
        print_result "PostgreSQL E2E Tests" $POSTGRES_RESULT
    fi
else
    echo -e "${YELLOW}⚠ PostgreSQL E2E Tests SKIPPED (--skip-postgres)${NC}"
fi

# Exit with failure if any test failed
if [ $UNIT_RESULT -ne 0 ] || [ $SQLITE_RESULT -ne 0 ] || [ $CLI_RESULT -ne 0 ]; then
    exit 1
fi

if [ "$RUN_POSTGRES" = true ] && [ $POSTGRES_RESULT -eq 1 ]; then
    exit 1
fi

echo ""
echo -e "${GREEN}All tests passed!${NC}"
exit 0
