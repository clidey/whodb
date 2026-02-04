# Testing Guide

This document covers all testing infrastructure for WhoDB: frontend Cypress E2E tests, Go backend tests, and CLI tests.

## Quick Reference

```bash
# Frontend Cypress (CE)
cd frontend && pnpm cypress:ce:headless        # All CE databases, headless
cd frontend && pnpm cypress:ce                  # Interactive mode

# Frontend Cypress (EE) - requires ee/ submodule
cd frontend && pnpm cypress:ee:headless        # EE databases only
cd frontend && pnpm cypress:all:headless       # CE + EE combined

# Backend Go tests
bash dev/run-backend-tests.sh all              # Unit + integration
bash dev/run-backend-tests.sh unit             # Unit tests only
bash dev/run-backend-tests.sh integration      # Integration tests only

# CLI tests
bash dev/run-cli-tests.sh                      # All CLI tests
bash dev/run-cli-tests.sh --skip-postgres      # Skip PostgreSQL E2E
```

---

## Frontend Cypress Tests

### Architecture

```
frontend/cypress/
├── e2e/features/           # 28 feature test files (.cy.js)
├── fixtures/databases/     # Database configuration JSON files
├── support/
│   ├── e2e.js              # Global setup (exceptions, animations, coverage)
│   ├── commands.js         # Custom Cypress commands
│   ├── test-runner.js      # Database iteration helper
│   ├── fixture-validator.js
│   └── categories/         # Database category helpers
│       ├── sql.js          # SQL database helpers
│       ├── document.js     # MongoDB/Elasticsearch helpers
│       ├── keyvalue.js     # Redis helpers
│       └── index.js
└── logs/                   # Test execution logs
```

### Test Categories

Tests are organized by database category:

| Category | Databases |
|----------|-----------|
| `sql` | PostgreSQL, MySQL, MySQL8, MariaDB, SQLite, ClickHouse |
| `document` | MongoDB, Elasticsearch |
| `keyvalue` | Redis |

### Database Fixtures

Located in `frontend/cypress/fixtures/databases/`:
- `postgres.json`, `mysql.json`, `mysql8.json`, `mariadb.json`, `sqlite.json`
- `mongodb.json`, `elasticsearch.json`, `clickhouse.json`, `redis.json`

Each fixture defines: connection details, expected schemas, test data, and feature flags.

### NPM Scripts

#### CE Tests (Community Edition)

```bash
# Interactive mode (opens Cypress UI)
pnpm cypress:ce                        # All CE databases
pnpm cypress:db <database>             # Single database (e.g., postgres, mysql)
pnpm cypress:feature <feature>         # Single feature file

# Headless mode (CI/automated)
pnpm cypress:ce:headless               # All CE databases
pnpm cypress:db:headless <database>    # Single database
pnpm cypress:feature:headless <feature> # Single feature

# Debug mode
pnpm cypress:ce:headless:debug         # With debug logging
```

#### EE Tests (Enterprise Edition)

Requires the `ee/` submodule.

```bash
# Interactive mode
pnpm cypress:ee                        # EE databases only
pnpm cypress:all                       # CE + EE combined
pnpm cypress:ee:db <database>          # Single EE database
pnpm cypress:ee:feature <feature>      # Single EE feature

# Headless mode
pnpm cypress:ee:headless               # EE databases only
pnpm cypress:all:headless              # CE + EE combined
pnpm cypress:ee:db:headless <database>
pnpm cypress:ee:feature:headless <feature>
```

### Feature Test Files

Located in `frontend/cypress/e2e/features/`:

| File | Tests |
|------|-------|
| `login.cy.js` | Authentication flow |
| `crud.cy.js` | Create, Read, Update, Delete operations |
| `query-history.cy.js` | Query history tracking |
| `scratchpad.cy.js` | SQL/query editor |
| `explore.cy.js` | Database exploration |
| `graph.cy.js` | Entity relationship graphs |
| `export.cy.js` | Data export functionality |
| `pagination.cy.js` | Pagination controls |
| `sorting.cy.js` | Column sorting |
| `search.cy.js` | Search functionality |
| `where-conditions.cy.js` | SQL WHERE clause building |
| `schema-management.cy.js` | Schema operations |
| `data-types.cy.js` | Data type handling |
| `type-casting.cy.js` | Data type conversion |
| `mock-data.cy.js` | Mock data generation |
| `error-handling.cy.js` | Error scenarios |
| `loading-states.cy.js` | Loading state display |
| `keyboard-shortcuts.cy.js` | Keyboard navigation |
| `sidebar.cy.js` | Sidebar navigation |
| `tables-list.cy.js` | Table listing |
| `data-view.cy.js` | Data view UI |
| `profiles.cy.js` | User profiles |
| `settings.cy.js` | User settings |
| `storage.cy.js` | Browser storage/persistence |
| `tour.cy.js` | User onboarding tour |
| `chat.cy.js` | AI chat functionality |
| `key-types.cy.js` | Redis key types |
| `postgres-screenshots.cy.js` | Screenshot generation |

### Cypress Configuration

Key settings in `frontend/cypress.config.js`:
- **Viewport**: 1920×1080
- **Base URL**: `http://localhost:3000`
- **Test Isolation**: Enabled
- **Retries**: 2 in run mode, 0 in open mode
- **Video**: Disabled (performance)
- **Screenshots**: On failure only

---

## Docker Test Infrastructure

### Docker Compose

Location: `dev/docker-compose.yml`

#### CE Database Services

| Service | Port | Description |
|---------|------|-------------|
| `e2e_postgres` | 5432 | PostgreSQL |
| `e2e_mysql` | 3306 | MySQL |
| `e2e_mysql_842` | 3308 | MySQL 8.4.2 |
| `e2e_mariadb` | 3307 | MariaDB |
| `e2e_mongo` | 27017 | MongoDB |
| `e2e_redis` | 6379 | Redis |
| `e2e_elasticsearch` | 9200 | Elasticsearch |
| `e2e_clickhouse` | 8123 | ClickHouse |

SQLite uses a local file (`core/tmp/e2e_test.db`), no container needed.

#### EE Database Services

Location: `ee/dev/docker-compose.yml`

### Sample Data Scripts

Located in `dev/sample-data/`:
- `postgres/data.sql` - PostgreSQL initialization
- `mysql/data.sql` - MySQL initialization
- `mongo/data.js` - MongoDB initialization
- `elasticsearch/upload.sh` - Elasticsearch data upload
- `redis/init.sh` - Redis initialization

### E2E Orchestration Scripts

#### Main Runner: `dev/run-cypress.sh`

```bash
./dev/run-cypress.sh [headless] [database] [spec]

# Examples:
./dev/run-cypress.sh                    # Interactive, all databases
./dev/run-cypress.sh headless           # Headless, all databases
./dev/run-cypress.sh headless postgres  # Headless, PostgreSQL only
./dev/run-cypress.sh headless postgres crud  # Specific spec file
```

#### Environment Variables

| Variable | Description |
|----------|-------------|
| `WHODB_DATABASES` | Space-separated list of databases to test |
| `WHODB_DB_CATEGORIES` | Colon-separated db:category pairs |
| `WHODB_CYPRESS_DIRS` | Custom Cypress directories |
| `WHODB_VITE_EDITION` | Build edition (empty=CE, 'ee'=EE) |
| `WHODB_SETUP_MODE` | Setup mode ('ce' or 'ee') |
| `WHODB_LOG_LEVEL` | Log level (default: 'error') |

#### Setup/Cleanup Scripts

```bash
# Setup E2E environment
bash dev/setup-e2e.sh

# Cleanup after tests
bash dev/cleanup-e2e.sh
```

Features:
- Smart test binary caching (detects source changes)
- Parallel port waiting for services
- Docker volume pruning on cleanup

---

## Go Backend Tests

### Architecture

```
core/
├── server_test.go                      # Server startup tests
├── graph/
│   ├── graphql_queries_test.go         # GraphQL query tests
│   ├── resolver_test.go                # Resolver tests
│   ├── graphql_integration_test.go     # Integration tests
│   ├── resolver_mutation_test.go       # Mutation tests
│   ├── http_resolvers_test.go          # HTTP resolver tests
│   └── mockdata_resolver_test.go       # Mock data tests
├── src/
│   ├── settings/*_test.go              # Settings tests
│   ├── mockdata/generator_test.go      # Generator tests
│   ├── llm/llm_test.go                 # LLM tests
│   ├── auth/*_test.go                  # Auth tests
│   └── plugins/
│       ├── common_test.go              # Common plugin tests
│       └── gorm/constraint_helpers_test.go
└── test/integration/
    ├── main_test.go                    # Integration setup
    ├── roundtrip_test.go               # Basic roundtrip
    └── full_roundtrip_test.go          # Full roundtrip
```

### Running Backend Tests

```bash
# Main test runner
bash dev/run-backend-tests.sh all          # Unit + integration (default)
bash dev/run-backend-tests.sh unit         # Unit tests only
bash dev/run-backend-tests.sh integration  # Integration tests only

# Direct Go commands
cd core && go test ./...                   # All unit tests
cd core && go test ./graph/...             # GraphQL tests only
cd core && go test ./src/auth/...          # Auth tests only
```

### Integration Tests

Integration tests require Docker services running:

```bash
# Start services manually
docker compose -f dev/docker-compose.yml up -d

# Run integration tests
cd core && go test ./test/integration/...

# Or use the script (manages Docker automatically)
bash dev/run-backend-tests.sh integration
```

### EE Backend Tests

If `ee/` submodule is available:

```bash
# Unit tests with EE tag
cd core && go test -tags ee ./...

# The run-backend-tests.sh script handles this automatically
bash dev/run-backend-tests.sh all
```

---

## CLI Tests

### Architecture

```
cli/
├── cmd/
│   ├── cmd_test.go                     # Command tests
│   ├── mcp_test.go                     # MCP integration
│   ├── programmatic_commands_test.go   # Programmatic commands
│   └── test_env_test.go                # Test environment
├── internal/
│   ├── config/config_test.go           # Configuration
│   ├── database/
│   │   ├── manager_test.go             # Database manager
│   │   ├── integration_test.go         # Integration tests
│   │   └── test_env_test.go            # Test environment
│   └── tui/
│       ├── model_test.go               # TUI model
│       ├── history_view_test.go        # History view
│       ├── where_view_test.go          # WHERE clause view
│       ├── columns_view_test.go        # Columns view
│       ├── results_view_test.go        # Results view
│       ├── browser_view_test.go        # Browser view
│       ├── schema_view_test.go         # Schema view
│       ├── chat_view_test.go           # Chat view
│       ├── connection_view_test.go     # Connection view
│       ├── export_view_test.go         # Export view
│       ├── editor_view_test.go         # Editor view
│       └── test_env_test.go            # Test environment
└── e2e/                                # E2E test setup
```

### Running CLI Tests

```bash
# Main test runner (recommended)
bash dev/run-cli-tests.sh                  # All tests
bash dev/run-cli-tests.sh --skip-postgres  # Skip PostgreSQL E2E
bash dev/run-cli-tests.sh -v               # Verbose output

# Direct Go commands
cd cli && go test ./internal/...           # Internal tests only
cd cli && go test ./cmd/...                # Command tests only
```

### CLI E2E Tests

```bash
# Full E2E workflow (setup, test, cleanup)
bash dev/run-cli-e2e.sh

# Manual control
bash dev/setup-cli-e2e.sh     # Setup PostgreSQL container
bash dev/cleanup-cli-e2e.sh   # Cleanup container
```

### Test Types

| Type | Command | Description |
|------|---------|-------------|
| Unit | `go test ./internal/...` | Internal package tests |
| SQLite E2E | `go test ./e2e/... -run "^TestEndToEnd"` | SQLite integration |
| CLI E2E | `go test -tags=e2e_cli ./e2e/... -run "^TestCLI_"` | Full CLI E2E |
| Postgres E2E | `go test -tags=e2e_postgres ./e2e/... -run "^TestPostgres_"` | PostgreSQL E2E |

---

## Coverage

### Frontend Coverage

```bash
# View frontend coverage report
cd frontend && pnpm view:coverage:frontend
```

Coverage is collected via `@cypress/code-coverage` during Cypress runs.

### Backend Coverage

```bash
# Generate coverage
cd core && go test -coverprofile=coverage.out ./...

# View coverage
go tool cover -html=coverage.out
```

---

## Troubleshooting

### Common Issues

**Cypress tests fail to start:**
- Ensure Docker services are running: `docker compose -f dev/docker-compose.yml ps`
- Check frontend is built: `cd frontend && pnpm build`
- Verify ports are free: 3000 (frontend), 8080 (backend)

**Database connection errors:**
- Wait for services to be healthy: `bash dev/wait-for-services.sh`
- Check Docker logs: `docker compose -f dev/docker-compose.yml logs <service>`

**Test binary caching issues:**
- Clear cache: `rm -rf core/tmp/server.test*`
- Force rebuild: Delete `core/tmp/.source_hash`

**EE tests not found:**
- Ensure `ee/` submodule is cloned: `git submodule update --init`

### Debug Mode

```bash
# Cypress debug logging
pnpm cypress:ce:headless:debug

# Go test verbose
go test -v ./...

# CLI test verbose
bash dev/run-cli-tests.sh -v
```
