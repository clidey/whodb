# Plugin Architecture Guide

The plugin architecture avoids hardcoded database type checks. All database-specific logic lives in plugins.

## Core Principle: No Switch Statements

```go
// WRONG - Don't do this:
func GetConstraints(dbType string, ...) {
    switch dbType {
    case "Postgres":
        // PostgreSQL logic
    case "MySQL":
        // MySQL logic
    }
}

// CORRECT - Add to PluginFunctions interface:
GetColumnConstraints(config *PluginConfig, schema string, storageUnit string) (map[string]map[string]any, error)

// Then implement in each plugin:
func (p *PostgresPlugin) GetColumnConstraints(...) { /* PostgreSQL-specific */ }
func (p *MySQLPlugin) GetColumnConstraints(...) { /* MySQL-specific */ }
```

## Plugin Hierarchy

```
BasePlugin (engine/base_plugin.go — defaults for all 30 PluginFunctions methods)
├── GormPlugin (gorm/plugin.go — SQL-specific, embeds engine.Plugin)
│   ├── PostgresPlugin      (all 6 capabilities)
│   ├── MySQLPlugin         (no SupportsSchema)
│   ├── Sqlite3Plugin       (no SupportsSchema, no SupportsDatabaseSwitch)
│   ├── ClickHousePlugin    (no SupportsSchema)
│   ├── MSSQLPlugin (EE)    (all 6 capabilities)
│   └── OraclePlugin (EE)   (all 6 capabilities)
├── MongoDBPlugin           (embeds BasePlugin — Chat, Scratchpad, DatabaseSwitch)
├── RedisPlugin             (embeds BasePlugin — Chat, Scratchpad, DatabaseSwitch)
├── ElasticSearchPlugin     (embeds BasePlugin — Chat, Scratchpad)
└── DynamoDBPlugin (EE)     (embeds BasePlugin — Chat, Scratchpad, DatabaseSwitch)
```

### BasePlugin

`engine.BasePlugin` provides default implementations for all `PluginFunctions` methods:
- User-facing operations (RawExecute, Chat, GetGraph) → `errors.ErrUnsupported`
- Internal operations (GetColumnConstraints, NullifyFKColumn) → empty/nil
- WithTransaction → direct execution (no wrapping)
- FormatValue → `fmt.Sprintf("%v", val)`

Non-SQL plugins embed `BasePlugin` and override only the methods they implement. No stubs needed.

### GormPlugin

SQL plugins extend `GormPlugin` which implements all `PluginFunctions` via GORM. It also defines `GormPluginFunctions` with SQL-specific hook methods. Key hooks that plugins override:
- `IsArrayType()` — Postgres overrides for underscore-prefixed array types
- `ResolveGraphSchema()` — ClickHouse overrides to use database name
- `ShouldCheckRowsAffected()` — ClickHouse overrides to return false
- `ConvertStringValue()` — table-driven type conversion with per-plugin overrides
- `GetPrimaryKeyColumns()` — SQLite overrides for PRAGMA-based detection

## Capabilities System

Each plugin declares its capabilities in `GetDatabaseMetadata()`:

```go
type Capabilities struct {
    SupportsScratchpad     bool  // RawExecute / Scratchpad tab
    SupportsChat           bool  // AI Chat tab
    SupportsGraph          bool  // Graph visualization tab
    SupportsSchema         bool  // Schema dropdown in sidebar
    SupportsDatabaseSwitch bool  // Database dropdown in sidebar
    SupportsModifiers      bool  // PRIMARY KEY, NOT NULL modifiers in create form
}
```

The frontend reads capabilities from the `DatabaseMetadata` GraphQL query (cached in Redux store) and uses them to show/hide UI elements. This is the **single source of truth** — the frontend falls back to hardcoded lists only before capabilities are loaded (login screen).

### Adding a New Capability

1. Add field to `engine.Capabilities` struct in `core/src/engine/metadata.go`
2. Set it in each plugin's `GetDatabaseMetadata()`
3. Expose in GraphQL schema (`core/graph/schema.graphqls` — `Capabilities` type)
4. Read in frontend via `database-features.ts` helper functions
5. The frontend reads from `reduxStore.getState().databaseMetadata.capabilities`

## Chat System

### How Chat Works

1. Plugin's `Chat()` method builds schema context (table names + columns)
2. Calls BAML function with `database_type` to generate queries in native syntax
3. BAML returns `ChatResponse[]` — messages + queries with operation types
4. Non-mutation queries (GET) execute immediately via `plugin.RawExecute()`
5. Mutation queries require user confirmation before execution

### BAML Prompts

- `core/baml_src/sql_chat.baml` — SQL-specific prompt (`GenerateSQLQuery`)
- `core/baml_src/db_chat.baml` — Database-agnostic prompt (`GenerateDBQuery`) with conditional blocks for SQL, MongoDB, Elasticsearch, Redis
- `ee/baml_src/sql_chat_ee.baml` — EE SQL prompt with chart support (`GenerateSQLQueryEE`)

### Chat by Database Type

| Database | Chat() calls | BAML function | Charts (EE) |
|---|---|---|---|
| SQL plugins (CE) | `common.SQLChatBAML` | `GenerateSQLQuery` | No |
| SQL plugins (EE) | `eecommon.SQLChatBAML` | `GenerateSQLQueryEE` | Yes |
| DynamoDB (EE) | `eecommon.SQLChatBAML` | `GenerateSQLQueryEE` | Yes |
| MongoDB | `common.DBChatBAML` | `GenerateDBQuery` | No |
| Elasticsearch | `common.DBChatBAML` | `GenerateDBQuery` | No |
| Redis | `common.DBChatBAML` | `GenerateDBQuery` | No |

### RawExecute by Database Type

Each database parses queries in its native format:
- **SQL plugins**: Standard SQL via GORM
- **MongoDB**: `db.collection.operation(args)` shell syntax
- **Elasticsearch**: `INDEX_NAME | QUERY_JSON` format
- **Redis**: Standard Redis commands (`HGETALL key`, `KEYS pattern`, etc.)
- **DynamoDB**: PartiQL via `ExecuteStatement`

## Adding New Functionality

1. Add method to `PluginFunctions` interface in `core/src/engine/plugin.go`
2. Add default implementation in `BasePlugin` (`core/src/engine/base_plugin.go`)
3. Override in `GormPlugin` for SQL behavior (`core/src/plugins/gorm/plugin.go`)
4. Override in specific plugins as needed
5. Add capability flag if the feature is optional
6. Frontend reads capability to show/hide UI

## Plugin File Organization

SQL-based plugins follow this structure (see `core/src/plugins/postgres/` as reference):
- `db.go` - Connection creation (implements DB method)
- `postgres.go` (or `mysql.go`, etc.) - Plugin struct, NewXxxPlugin(), database-specific queries
- `types.go` - Type definitions, alias map, capabilities, and GetDatabaseMetadata()
- `constraints.go` - Column constraint detection (optional override)
- `chat.go` - Chat implementation (optional — BasePlugin returns ErrUnsupported)
- `raw_execute.go` - RawExecute for native query languages (NoSQL plugins)

GormPlugin base class (`core/src/plugins/gorm/`) provides:
- `plugin.go` - GormPluginFunctions interface + default implementations
- `sqlbuilder.go` - SQL query building + `RecordsToColumnDefs` helper
- `errors.go` - ErrorHandler for user-friendly error messages
- `add.go`, `update.go`, `delete.go` - CRUD operations
- `chat.go` - SQL chat via BAML
- `utils.go` - Table-driven `ConvertStringValue` type conversion

Shared utilities:
- `plugins/connection_cache.go` - GORM connection caching with TTL + LRU eviction
- `plugins/connection_pool.go` - Pool config, `WithConnection()` lifecycle, SSL cache
- `plugins/table_dependencies.go` - FK-aware recursive table clearing
- `plugins/graphutil/graphutil.go` - Shared FK inference for graph visualization

## Adding a New Database

1. Create plugin directory in `core/src/plugins/` (CE) or `ee/core/src/plugins/` (EE)
2. Define plugin struct embedding `engine.BasePlugin` (or `GormPlugin` for SQL)
3. Override methods the database supports — BasePlugin defaults handle the rest
4. Set capabilities in `GetDatabaseMetadata()`
5. Register in `core/src/src.go` via `MainEngine.RegistryPlugin(yourplugin.NewYourPlugin())`
6. For EE: Register in `ee/core/src/plugins/init.go`
7. Frontend automatically adapts based on capabilities — no hardcoded type lists needed

## Database Metadata (types.go)

Each plugin provides metadata for frontend UI via `GetDatabaseMetadata()`. This is the **single source of truth** for:
- Valid operators (=, >=, LIKE, etc.)
- Type definitions (VARCHAR, INTEGER, etc.) with UI hints (hasLength, hasPrecision)
- Alias maps (INT → INTEGER, BOOL → BOOLEAN)
- Capabilities (which UI features to show)

The frontend fetches this via GraphQL `DatabaseMetadata` query on login. **No fallbacks** — if backend doesn't provide it, the UI will be broken.

### types.go Structure

```go
func (p *PostgresPlugin) GetDatabaseMetadata() *engine.DatabaseMetadata {
    return &engine.DatabaseMetadata{
        DatabaseType:    engine.DatabaseType_Postgres,
        TypeDefinitions: TypeDefinitions,
        Operators:       operators,
        AliasMap:        AliasMap,
        Capabilities: engine.Capabilities{
            SupportsScratchpad:     true,
            SupportsChat:           true,
            SupportsGraph:          true,
            SupportsSchema:         true,
            SupportsDatabaseSwitch: true,
            SupportsModifiers:      true,
        },
    }
}
```

### Type Validation

Column type validation uses `engine.ValidateColumnType()` which checks against TypeDefinitions. Types not in TypeDefinitions will be rejected when adding columns.

## Frontend Capability Detection

`frontend/src/utils/database-features.ts` provides capability-checking functions:
- `databaseSupportsChat()` — reads `capabilities.supportsChat` from Redux store
- `databaseSupportsScratchpad()` — reads `capabilities.supportsScratchpad`
- `databaseSupportsSchema()` — reads `capabilities.supportsSchema`
- `databaseSupportsDatabaseSwitching()` — reads `capabilities.supportsDatabaseSwitch`

Each function falls back to hardcoded lists if capabilities haven't loaded yet (pre-login). After login, the backend is the single source of truth.

## Quirks to Know

- SQLite doesn't use schemas - `FormTableName()` returns just table name
- PostgreSQL array types display with underscore prefix (`_text`)
- Redis `RawExecute` parses Redis command strings (HGETALL, KEYS, etc.)
- MongoDB `RawExecute` parses shell-style commands (db.collection.find(...))
- Elasticsearch `RawExecute` expects `INDEX_NAME | QUERY_JSON` format

## EE Compatibility

- EE uses `SetEEInitializer()` pattern to register plugins without modifying CE
- EE plugins in `ee/core/src/plugins/` automatically inherit interface methods
- EE overrides Chat for SQL plugins via `ee/core/src/plugins/gorm/chat.go` (adds chart support)
- NoSQL plugins use CE chat path (no EE override needed — charts require tabular data)
- Plugin architecture ensures clean CE/EE separation
