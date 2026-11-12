---
paths:
  - "frontend/e2e/**"
  - "dev/run-e2e.sh"
  - "dev/docker-compose.yml"
---

# E2E Test Rules

## Running Tests
```bash
cd frontend && pnpm e2e:ce:headless         # All databases, headless
cd frontend && pnpm e2e:db:headless postgres # Single database
cd frontend && pnpm e2e:feature:headless crud # Single feature
cd frontend && pnpm screenshot              # Docs screenshots (excluded from normal runs)
```

## Key Conventions
- Never write to localStorage directly for Redux state — use the settings page UI (redux-persist rehydration race)
- Use fixture config for conditional tests: `db.sidebar?.showsDatabaseDropdown`, `db.features`, etc.
- Use whodb helpers: `whodb.data()`, `whodb.setTablePageSize()`, `whodb.goto()`
- Filter by fixture capabilities, not hardcoded type names
- A spec that changes data must be listed in `MUTATING_TESTS` in `frontend/e2e/playwright.config.mjs`; those run after the read-only specs, with no retries.
- Selectors are `data-testid`s; when refactoring components keep every one the specs use.

## Sidebar Test IDs
- `sidebar-database-label` = heading text
- `sidebar-database` = selected value (e.g., "test_db")

## Terminology
- The "terminology" setting only affects DBs where `sidebar.showsSchemaDropdown: false` (MySQL, MariaDB, ClickHouse, MongoDB, Redis) — NOT Postgres

## Test Architecture
- Databases run one after another with one worker, because they share backend state. Each gets its own Playwright invocation with `DATABASE=<name>`, and the blob reports merge into one HTML report.
- Adding a database means a fixture in `frontend/e2e/fixtures/databases/`, a service in `dev/docker-compose.yml`, and entries in `DEFAULT_DATABASES` and `DEFAULT_CATEGORIES` in `dev/run-e2e.sh` and the `ALL` list in `.github/workflows/e2e-ce.yml`.
- SSL variants live under the compose `ssl` profile; SQLite uses `core/tmp/e2e_test.db` instead of a container.
