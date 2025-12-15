# Cypress E2E Testing Guide

This guide explains the feature-based E2E testing architecture for WhoDB.

## Directory Structure

```
cypress/
├── fixtures/databases/           # Database configurations
│   ├── postgres.json
│   ├── mysql.json
│   ├── mysql8.json
│   ├── mariadb.json
│   ├── sqlite.json
│   ├── mongodb.json
│   ├── redis.json
│   ├── elasticsearch.json
│   └── clickhouse.json
├── support/
│   ├── commands.js               # Cypress custom commands
│   ├── disable-animations.css    # CSS to disable animations in tests
│   ├── e2e.js                    # E2E support file
│   ├── test-runner.js            # Database iteration helper
│   └── categories/
│       ├── index.js              # Category exports
│       ├── sql.js                # SQL database helpers
│       ├── document.js           # MongoDB/Elasticsearch helpers
│       └── keyvalue.js           # Redis helpers
└── e2e/
    └── features/                 # Feature-based tests
        ├── tables-list.cy.js
        ├── explore.cy.js
        ├── data-view.cy.js
        ├── pagination.cy.js
        ├── sorting.cy.js
        ├── crud.cy.js
        ├── where-conditions.cy.js
        ├── search.cy.js
        ├── graph.cy.js
        ├── scratchpad.cy.js
        ├── query-history.cy.js
        ├── chat.cy.js
        ├── export.cy.js
        ├── mock-data.cy.js
        ├── type-casting.cy.js
        ├── settings.cy.js
        └── keyboard-shortcuts.cy.js
```

## Running Tests

### Interactive Mode (Cypress UI)

```bash
# All databases
pnpm cypress:ce

# Specific database
pnpm cypress:db postgres
pnpm cypress:db mysql
pnpm cypress:db mysql8
pnpm cypress:db mariadb
pnpm cypress:db sqlite
pnpm cypress:db mongodb
pnpm cypress:db redis
pnpm cypress:db elasticsearch
pnpm cypress:db clickhouse
```

### Headless Mode (CI)

```bash
# All databases sequentially
pnpm cypress:ce:headless

# Specific database headless
pnpm cypress:db:headless postgres
pnpm cypress:db:headless mysql
```

### Direct Script Usage

```bash
# Interactive: ./run-cypress.sh [headless] [database]
../dev/run-cypress.sh false postgres

# Headless: ./run-cypress.sh [headless] [database]
../dev/run-cypress.sh true all
```

## Adding a New Feature Test

### Step 1: Create the Test File

Create `cypress/e2e/features/my-feature.cy.js`:

```javascript
import { forEachDatabase, hasFeature } from '../../support/test-runner';

describe('My Feature', () => {

    // Run for SQL databases only
    forEachDatabase('sql', (db) => {
        it('does something', () => {
            // db contains all config from fixtures
            cy.data('users');
        });
    });

    // Run for document databases (MongoDB, Elasticsearch)
    forEachDatabase('document', (db) => {
        it('handles documents', () => {
            // Document-specific logic
        });
    });

    // Run for key-value databases (Redis)
    forEachDatabase('keyvalue', (db) => {
        it('handles keys', () => {
            // Redis-specific logic
        });
    });

    // Run for ALL databases
    forEachDatabase('all', (db) => {
        it('works universally', () => {
            // Check db.category for type-specific logic
        });
    });

});
```

### Step 2: Use Feature Flags for Conditional Tests

```javascript
forEachDatabase('sql', (db) => {
    // Skip if feature not supported
    if (!hasFeature(db, 'scratchpad')) {
        return;
    }

    it('uses scratchpad', () => {
        cy.goto('scratchpad');
    });
});
```

### Step 3: Access Database Configuration

The `db` object contains everything from the fixture file:

```javascript
forEachDatabase('sql', (db) => {
    it('uses config', () => {
        // Connection info
        const host = db.connection.host;

        // Expected tables
        expect(db.expectedTables).to.include('users');

        // Table configuration
        const usersConfig = db.tables.users;
        const columns = usersConfig.columns;
        const testData = usersConfig.testData.initial;

        // SQL queries (database-specific syntax)
        const query = db.sql.selectAllUsers;

        // Error patterns
        const errorPattern = db.sql.errorPatterns.tableNotFound;

        // Feature flags
        const hasGraph = db.features.graph;
    });
});
```

### Step 4: Add Custom Data to Fixtures (Optional)

Add to `fixtures/databases/postgres.json`:

```json
{
  "myFeature": {
    "expectedValue": "something",
    "testInput": "data"
  }
}
```

Access in test:

```javascript
const value = db.myFeature?.expectedValue;
```

## Database Fixture Schema

Each fixture file follows this structure:

```json
{
  "type": "Postgres",
  "category": "sql",
  "connection": {
    "host": "localhost",
    "dockerHost": "e2e_postgres",
    "port": 5432,
    "user": "user",
    "password": "password",
    "database": "test_db",
    "advanced": { "Port": "3307" }
  },
  "schema": "test_schema",
  "indexRefreshDelay": 1500,
  "includesSystemViews": true,
  "features": {
    "graph": true,
    "export": true,
    "scratchpad": true,
    "mockData": true,
    "chat": true,
    "whereConditions": true,
    "queryHistory": true
  },
  "expectedTables": ["users", "orders", "..."],
  "tables": {
    "users": {
      "columns": {
        "id": "integer",
        "username": "character varying"
      },
      "expectedColumns": ["", "id", "username", "..."],
      "testData": {
        "initial": [["", "1", "john_doe", "..."]],
        "newRow": { "id": "5", "username": "alice" }
      },
      "metadata": {
        "type": "BASE TABLE",
        "hasSize": true,
        "hasCount": true
      }
    }
  },
  "graph": {
    "expectedNodes": {
      "users": ["orders"],
      "orders": ["order_items"]
    }
  },
  "sql": {
    "schemaPrefix": "test_schema.",
    "selectAllUsers": "SELECT * FROM test_schema.users;",
    "errorPatterns": {
      "tableNotFound": "relation does not exist"
    }
  }
}
```

### Optional Fixture Properties

| Property | Description | Used By |
|----------|-------------|---------|
| `connection.advanced` | Advanced login options (e.g., `{"Port": "3307"}`) | MariaDB (non-default port) |
| `indexRefreshDelay` | Delay in ms after add/delete for index refresh | Elasticsearch |
| `includesSystemViews` | Whether to expect `system.views` in table list | MongoDB |

## Available Helpers

### Test Runner (`support/test-runner.js`)

```javascript
import {
    forEachDatabase,      // Iterate over databases by category
    hasFeature,           // Check if db supports a feature
    getTableConfig,       // Get table config: getTableConfig(db, 'users')
    getSqlQuery,          // Get SQL query: getSqlQuery(db, 'selectAllUsers')
    getErrorPattern,      // Get error: getErrorPattern(db, 'tableNotFound')
    getDatabaseConfig,    // Get config by name: getDatabaseConfig('postgres')
    loginToDatabase,      // Login helper (auto-called by forEachDatabase)
} from '../../support/test-runner';
```

### SQL Helpers (`support/categories/sql.js`)

```javascript
import {
    verifyRow,            // Verify single row data
    verifyRows,           // Verify multiple rows
    verifyColumnTypes,    // Verify column types from explore
    verifyMetadata,       // Verify table metadata
    verifyGraph,          // Verify graph topology
    verifyScratchpadOutput,
} from '../../support/categories/sql';
```

### Document Helpers (`support/categories/document.js`)

```javascript
import {
    parseDocument,        // Parse JSON from table row
    verifyDocument,       // Verify document properties
    verifyDocumentRow,    // Verify row contains expected doc
    getDocumentId,        // Get _id from document row
    createUpdatedDocument,// Create updated doc for editing
} from '../../support/categories/document';
```

### Key-Value Helpers (`support/categories/keyvalue.js`)

```javascript
import {
    verifyHashField,      // Verify hash field value
    verifyHashFields,     // Verify hash has expected fields
    verifyMembers,        // Verify set/list members
    verifySortedSetEntries,
    verifyStringValue,    // Verify string key value
    verifyColumnsForType, // Verify columns match Redis type
    filterSessionKeys,    // Filter out session:* keys
} from '../../support/categories/keyvalue';
```

## Example: Complete Feature Test

```javascript
// cypress/e2e/features/autocomplete.cy.js

import { forEachDatabase, hasFeature, getSqlQuery } from '../../support/test-runner';

describe('SQL Autocomplete', () => {

    forEachDatabase('sql', (db) => {
        if (!hasFeature(db, 'scratchpad')) {
            return;
        }

        beforeEach(() => {
            cy.goto('scratchpad');
        });

        it('suggests table names when typing', () => {
            cy.writeCode(0, 'SELECT * FROM us');
            cy.get('.cm-tooltip-autocomplete').should('be.visible');
            cy.get('.cm-completionLabel').should('contain', 'users');
        });

        it('suggests columns for known tables', () => {
            const prefix = db.sql?.schemaPrefix || '';
            cy.writeCode(0, `SELECT user FROM ${prefix}users`);
            cy.get('.cm-completionLabel').should('contain', 'username');
        });

        it('executes completed query', () => {
            const query = getSqlQuery(db, 'selectAllUsers');
            cy.writeCode(0, query);
            cy.runCode(0);

            cy.getCellQueryOutput(0).then(({ rows }) => {
                expect(rows.length).to.be.greaterThan(0);
            });
        });
    });

});
```

## Database Categories

| Category | Databases | Typical Features |
|----------|-----------|------------------|
| `sql` | Postgres, MySQL, MySQL8, MariaDB, SQLite, ClickHouse | Tables, columns, SQL queries, graph, scratchpad |
| `document` | MongoDB, Elasticsearch | Collections/indices, JSON documents, where conditions |
| `keyvalue` | Redis | Keys, hash/list/set/zset types, no graph/scratchpad |

## Adding a New Database

1. Create `fixtures/databases/newdb.json` following the schema above
2. Add to `DATABASES` array in `dev/run-cypress.sh`
3. Add to `DB_CATEGORIES` map in `dev/run-cypress.sh`
4. Add to `databaseConfigs` in `support/test-runner.js`
5. Add Docker service to `dev/docker-compose.e2e.yaml`
6. Add npm script to `package.json` (optional)

## Troubleshooting

### Tests not running for a database

Check that:
- The database is in the `DATABASES` array in `run-cypress.sh`
- The fixture file exists in `fixtures/databases/`
- The `forEachDatabase` category matches (`sql`, `document`, `keyvalue`, or `all`)

### Feature test skipped unexpectedly

Check the `features` object in the database fixture:

```json
{
  "features": {
    "yourFeature": true
  }
}
```

### Viewing logs for failed tests

```bash
# Logs are in frontend/cypress/logs/
cat cypress/logs/postgres.log
cat cypress/logs/mysql.log
```

### Running a single test file

```bash
CYPRESS_database=postgres npx cypress run --spec "cypress/e2e/features/crud.cy.js"
```
