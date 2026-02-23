# Testing Guide

This document covers all testing infrastructure for WhoDB: frontend Playwright E2E tests, Go backend tests, and CLI tests.

## Quick Reference

```bash
# Frontend Playwright E2E (CE)
cd frontend && pnpm e2e:ce:headless          # All CE databases, headless
cd frontend && pnpm e2e:ce                   # Interactive mode (headed)

# Frontend Playwright E2E (EE) - requires ee/ submodule
cd frontend && pnpm e2e:ee:headless          # EE databases only
cd frontend && pnpm e2e:all:headless         # CE + EE combined

# Backend Go tests
bash dev/run-backend-tests.sh all            # Unit + integration
bash dev/run-backend-tests.sh unit           # Unit tests only
bash dev/run-backend-tests.sh integration    # Integration tests only

# CLI tests
bash dev/run-cli-tests.sh                    # All CLI tests
bash dev/run-cli-tests.sh --skip-postgres    # Skip PostgreSQL E2E
```

---

## Frontend Playwright E2E Tests

### Architecture

```
frontend/e2e/
├── tests/
│   ├── features/              # 29 feature test files (.spec.mjs)
│   └── postgres-screenshots.spec.mjs  # Screenshot generation (run separately)
├── fixtures/databases/        # Database configuration JSON files
├── support/
│   ├── test-fixture.mjs       # Test fixture (whodb helper, forEachDatabase, coverage)
│   ├── whodb.mjs              # WhoDB helper class (all page commands)
│   ├── database-config.mjs    # Database config loader
│   ├── global-setup.mjs       # Global setup (health checks)
│   ├── helpers/
│   │   ├── animation.mjs      # Browser state helpers
│   │   └── fixture-validator.mjs  # Feature validation
│   └── categories/            # Database category helpers
│       ├── sql.mjs            # SQL database helpers
│       ├── document.mjs       # MongoDB/Elasticsearch helpers
│       ├── keyvalue.mjs       # Redis helpers
│       └── index.mjs
├── reports/
│   ├── blobs/                 # Blob reports per database (merged after all runs)
│   ├── html/                  # Merged HTML report (all databases combined)
│   └── test-results/          # Test artifacts (screenshots, traces, videos)
└── logs/                      # Per-database execution logs
```

### Test Categories

Tests are organized by database category:

| Category | Databases |
|----------|-----------|
| `sql` | PostgreSQL, MySQL, MySQL8, MariaDB, SQLite, ClickHouse |
| `document` | MongoDB, Elasticsearch |
| `keyvalue` | Redis |

### Database Fixtures

Located in `frontend/e2e/fixtures/databases/`:
- `postgres.json`, `mysql.json`, `mysql8.json`, `mariadb.json`, `sqlite.json`
- `mongodb.json`, `elasticsearch.json`, `clickhouse.json`, `redis.json`

Each fixture defines: connection details, expected schemas, test data, and feature flags.

### NPM Scripts

#### CE Tests (Community Edition)

```bash
# Interactive mode (opens browser UI)
pnpm e2e:ce                             # All CE databases
pnpm e2e:db <database>                  # Single database (e.g., postgres, mysql)
pnpm e2e:feature <feature>              # Single feature file

# Headless mode (CI/automated)
pnpm e2e:ce:headless                    # All CE databases
pnpm e2e:db:headless <database>         # Single database
pnpm e2e:feature:headless <feature>     # Single feature

# Debug mode
pnpm e2e:ce:headless:debug              # With Playwright debug logging
```

#### EE Tests (Enterprise Edition)

Requires the `ee/` submodule.

```bash
# Interactive mode
pnpm e2e:ee                             # EE databases only
pnpm e2e:all                            # CE + EE combined
pnpm e2e:ee:db <database>               # Single EE database
pnpm e2e:ee:feature <feature>           # Single EE feature

# Headless mode
pnpm e2e:ee:headless                    # EE databases only
pnpm e2e:all:headless                   # CE + EE combined
pnpm e2e:ee:db:headless <database>
pnpm e2e:ee:feature:headless <feature>
```

### Feature Test Files

Located in `frontend/e2e/tests/features/`:

| File | Tests |
|------|-------|
| `login.spec.mjs` | Authentication flow |
| `crud.spec.mjs` | Create, Read, Update, Delete operations |
| `query-history.spec.mjs` | Query history tracking |
| `scratchpad.spec.mjs` | SQL/query editor |
| `explore.spec.mjs` | Database exploration |
| `graph.spec.mjs` | Entity relationship graphs |
| `export.spec.mjs` | Data export functionality |
| `pagination.spec.mjs` | Pagination controls |
| `sorting.spec.mjs` | Column sorting |
| `search.spec.mjs` | Search functionality |
| `where-conditions.spec.mjs` | SQL WHERE clause building |
| `schema-management.spec.mjs` | Schema operations |
| `data-types.spec.mjs` | Data type handling |
| `type-casting.spec.mjs` | Data type conversion |
| `mock-data.spec.mjs` | Mock data generation |
| `error-handling.spec.mjs` | Error scenarios |
| `loading-states.spec.mjs` | Loading state display |
| `keyboard-shortcuts.spec.mjs` | Keyboard navigation |
| `sidebar.spec.mjs` | Sidebar navigation |
| `tables-list.spec.mjs` | Table listing |
| `data-view.spec.mjs` | Data view UI |
| `profiles.spec.mjs` | User profiles |
| `settings.spec.mjs` | User settings |
| `storage.spec.mjs` | Browser storage/persistence |
| `tour.spec.mjs` | User onboarding tour |
| `chat.spec.mjs` | AI chat functionality |
| `key-types.spec.mjs` | Redis key types |
| `ssl-config.spec.mjs` | SSL configuration |
| `ssl-modes.spec.mjs` | SSL modes |

### Screenshot Tests

`postgres-screenshots.spec.mjs` is excluded from the normal test suite (`testIgnore` in config). Run it separately:

```bash
cd frontend && DATABASE=postgres pnpm exec playwright test \
  --config=e2e/playwright.config.mjs \
  --project=standalone \
  tests/postgres-screenshots.spec.mjs
```

### Playwright Configuration

Key settings in `frontend/e2e/playwright.config.mjs`:
- **Viewport**: 1920×1080
- **Base URL**: `http://localhost:3000`
- **Workers**: 1 (sequential — databases share backend state)
- **Retries**: 1 locally, 2 in CI
- **Reporter**: Blob (merged into HTML after all database runs)
- **Traces/Videos**: Retained on failure

### Test Execution Model

The run script (`dev/run-e2e.sh`) executes databases **sequentially**:
1. Each database gets its own `playwright test` invocation with `DATABASE=<name>`
2. Each invocation writes a blob report to `e2e/reports/blobs/`
3. After all databases finish, blobs are merged into a single HTML report
4. Output is piped through `tee` to both terminal and `e2e/logs/<database>.log`

### Coverage

Frontend code coverage is collected via Istanbul instrumentation:
1. `vite-plugin-istanbul` instruments source code when `NODE_ENV=test`
2. The test fixture collects `window.__coverage__` after each test
3. Coverage data is written to `.nyc_output/coverage-<testId>.json`
4. `nyc report` merges and generates reports

```bash
cd frontend && pnpm view:coverage           # Text summary
cd frontend && pnpm view:coverage:frontend  # HTML report
cd frontend && pnpm coverage:clean          # Clear coverage data
```

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

#### Main Runner: `dev/run-e2e.sh`

```bash
./dev/run-e2e.sh [headless] [database] [spec]

# Examples:
./dev/run-e2e.sh                          # Interactive, all databases
./dev/run-e2e.sh true                     # Headless, all databases
./dev/run-e2e.sh true postgres            # Headless, PostgreSQL only
./dev/run-e2e.sh true postgres crud       # Specific spec file
```

#### Environment Variables

| Variable | Description |
|----------|-------------|
| `WHODB_DATABASES` | Space-separated list of databases to test |
| `WHODB_DB_CATEGORIES` | Colon-separated db:category pairs |
| `WHODB_VITE_EDITION` | Build edition (empty=CE, 'ee'=EE) |
| `WHODB_SETUP_MODE` | Setup mode ('ce' or 'ee') |
| `WHODB_LOG_LEVEL` | Log level (default: 'error') |
| `DATABASE` | Target database (set by run script per invocation) |
| `CATEGORY` | Target category (set by run script per invocation) |

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

Coverage is collected via Istanbul instrumentation during Playwright E2E runs:

```bash
cd frontend && pnpm view:coverage           # Text summary
cd frontend && pnpm view:coverage:frontend  # HTML report
cd frontend && pnpm coverage:clean          # Clear coverage data
```

### Backend Coverage

```bash
# Generate coverage
cd core && go test -coverprofile=coverage.out ./...

# View coverage
go tool cover -html=coverage.out
```

---

## E2E Test Authoring Rules

### Database-Specific vs Global Features

Not all features apply to all databases. Before writing a test, check whether the feature is database-specific:

- **Sidebar terminology** (`databaseSchemaTerminology` setting): Only affects databases where `sidebar.showsDatabaseDropdown: true` AND `sidebar.showsSchemaDropdown: false` (MySQL, MariaDB, ClickHouse, MongoDB, Redis). Has NO effect on Postgres (which always shows "Database" + "Schema" as separate dropdowns). Filter with: `if (!db.sidebar?.showsDatabaseDropdown || db.sidebar?.showsSchemaDropdown !== false) return;`
- **Schema dropdown** (`sidebar-schema`): Only exists for databases where `sidebar.showsSchemaDropdown: true` (Postgres). Other databases don't have it.
- **Database dropdown** (`sidebar-database`): Only for databases where `sidebar.showsDatabaseDropdown: true`.

General rule: check the fixture's `sidebar` config to determine which UI elements exist for that database type.

### Sidebar Test IDs

The sidebar has distinct test IDs for **labels** (headings) vs **values** (selected items):

| Test ID | Element | Contains |
|---------|---------|----------|
| `sidebar-database-label` | `<h2>` heading | Label text: "Database" or "Schema" (based on terminology setting) |
| `sidebar-database` | `SearchSelect` button | Selected value: e.g., "test_db" |
| `sidebar-schema` | `SearchSelect` button | Selected value: e.g., "test_schema" |
| `sidebar-profile` | Profile section | Connection info |

### Redux State: Never Write to localStorage Directly

Redux-persist rehydrates from localStorage **asynchronously** on page load. Writing directly to `localStorage` and then navigating creates a race condition — the component may mount before rehydration completes, using default values instead.

**Wrong** (flaky):
```js
await page.evaluate(() => {
    const settings = JSON.parse(localStorage.getItem('persist:settings') || '{}');
    settings.defaultPageSize = '2';
    localStorage.setItem('persist:settings', JSON.stringify(settings));
});
await whodb.data(tableName); // May use default pageSize=100 instead of 2
```

**Right** — use the settings page UI (dispatches Redux action, immediately updates in-memory store):
```js
await whodb.goto('settings');
await page.locator('#default-page-size').click();
await page.locator('[data-value="custom"]').click();
await page.locator('input[type="number"]').clear();
await page.locator('input[type="number"]').fill('2');
await page.locator('input[type="number"]').press('Enter');
await whodb.data(tableName); // Reliably uses pageSize=2
```

**Exception**: The `whodb.login()` and `whodb.data()` helpers write `storageUnitView` to localStorage. This works because they immediately trigger a full page navigation afterward, and the value is non-critical (just controls card vs list view).

### forEachDatabase Filtering Patterns

Use fixture config properties to filter, not hardcoded type names:

```js
// Good: condition-based, works for any database with the right config
forEachDatabase('all', (db) => {
    if (!db.sidebar?.showsDatabaseDropdown || db.sidebar?.showsSchemaDropdown !== false) return;
    // Tests for databases where terminology setting is relevant
});

// Good: feature-based filtering via options
forEachDatabase('sql', (db) => { ... }, { features: ['pagination'] });

// Acceptable: type-based when testing truly database-specific behavior
forEachDatabase('sql', (db) => {
    if (db.type !== 'Postgres') return;
    // Postgres-specific tests
});
```

### WhoDB Helper Methods

Always prefer existing helper methods over raw page interactions:

| Task | Helper | Don't Do |
|------|--------|----------|
| Navigate to table data | `whodb.data(tableName)` | Manual card click |
| Change page size in table | `whodb.setTablePageSize(n)` | Direct combobox interaction |
| Submit table query | `whodb.submitTable()` | Click query button manually |
| Get table contents | `whodb.getTableData()` | Manual DOM scraping |
| Select schema | `whodb.selectSchema(v)` | Click sidebar-schema manually |
| Navigate to route | `whodb.goto('settings')` | `page.goto(url)` |

---

## Troubleshooting

### Common Issues

**E2E tests fail to start:**
- Ensure Docker services are running: `docker compose -f dev/docker-compose.yml ps`
- Check frontend is built: `cd frontend && pnpm build`
- Verify ports are free: 3000 (frontend), 8080 (backend)

**Permission errors on test artifacts:**
- Root-owned files from Docker/gateway runs: `sudo rm -rf frontend/e2e/reports/`
- The run script attempts `sudo` cleanup automatically

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
# Playwright debug logging
pnpm e2e:ce:headless:debug

# Go test verbose
go test -v ./...

# CLI test verbose
bash dev/run-cli-tests.sh -v
```
