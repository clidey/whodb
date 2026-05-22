---
paths:
  - "frontend/e2e/**"
  - "ee/frontend/e2e/**"
  - "dev/run-e2e.sh"
  - "dev/docker-compose.yml"
---

# E2E Test Rules

## Running Tests
```bash
cd frontend && pnpm e2e:ce:headless         # All databases, headless
cd frontend && pnpm e2e:db:headless postgres # Single database
cd frontend && pnpm e2e:feature:headless crud # Single feature
```

## Key Conventions
- Never write to localStorage directly for Redux state — use the settings page UI (redux-persist rehydration race)
- Use fixture config for conditional tests: `db.sidebar?.showsDatabaseDropdown`, `db.features`, etc.
- Use whodb helpers: `whodb.data()`, `whodb.setTablePageSize()`, `whodb.goto()`
- Filter by fixture capabilities, not hardcoded type names

## Sidebar Test IDs
- `sidebar-database-label` = heading text
- `sidebar-database` = selected value (e.g., "test_db")

## Terminology
- The "terminology" setting only affects DBs where `sidebar.showsSchemaDropdown: false` (MySQL, MariaDB, ClickHouse, MongoDB, Redis) — NOT Postgres

## Test Architecture
- Tests run sequentially (databases share backend state)
- Each database gets its own Playwright invocation with `DATABASE=<name>`
- Blob reports merge into single HTML report after all runs
