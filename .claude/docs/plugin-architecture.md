# Plugin Architecture Guide

The public WhoDB API is now source-first, but the execution layer is still
plugin-driven. `core/src/source/` and `core/src/sourcecatalog/` define the
public `Source*` contract, while database-specific execution logic still lives
in plugins under `core/src/plugins/`.

## Core Principle: No Switch Statements

```go
// WRONG - Don't do this:
func GetConstraints(sourceType string, ...) {
    switch sourceType {
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

## Adding New Functionality

If the change is public/source-facing, update the source layer as well as the
plugin layer:

1. Add or adapt the execution method on `PluginFunctions` in `core/src/engine/plugin.go` if the capability needs new plugin behavior
2. Update the source adapter in `core/src/source/adapters/` if the public `Source*` API needs to expose that behavior
3. Update `core/src/sourcecatalog/catalog.go` if the source contract, surfaces, or object actions/views change
4. Provide default behavior in shared plugin code (`GormPlugin` in `core/src/plugins/gorm/plugin.go`) where appropriate
5. Override in specific plugins as needed

If the change is purely plugin-internal, you may only need steps 1, 4, and 5.

## Alias Databases vs Wrapper Plugins

Database catalog aliases are only for cases where the runtime behavior is
genuinely identical and only the product metadata changes (label, default port,
TLS defaults, managed-service flags, source traits, etc.).

Connection defaults such as ports belong in the shared database/source catalog
metadata (`dbcatalog` `Extra["Port"]`, which flows into
`SourceType.ConnectionFields`), not in per-plugin registries.

If an alias needs even one runtime override, promote it to a thin first-class
plugin wrapper instead of adding alias-specific branches in shared code. Common
promotion triggers:

- different introspection SQL or system catalog support
- different namespace or schema handling
- different auth/DSN behavior
- different mutation semantics
- different graph or metadata queries

Examples in the repo:
- `CockroachDB` is a PostgreSQL-derived wrapper plugin because it needs catalog
  query overrides
- `MariaDB` and `TiDB` are MySQL-derived wrapper plugin types
- `QuestDB` is treated as a PostgreSQL-derived wrapper because it is schema-less
  in our product model and cannot use the default PostgreSQL table-info query

Do not solve alias incompatibilities with `if dbType == ...` checks in shared
code.

## Request Context and Cancellation

Every request-scoped plugin operation must use the context carried by `*engine.PluginConfig`. Do not use `context.Background()` for query execution, metadata fetches, SDK calls, or health checks that are part of a user request.

- GORM-based SQL plugins should inherit cancellation and timeout behavior through `plugins.WithConnection()`, `connection_pool.go`, `connection_cache.go`, and `GormPlugin`
- Direct-driver plugins must use `config.OperationContext()` for request-scoped SDK calls
- Use `config.OperationContextWithTimeout(...)` when the plugin needs an explicit upper bound for a long-running operation
- Reserve `context.Background()` for non-request cleanup work such as cache eviction or best-effort disconnects
- If a plugin talks to a driver directly, add a small local helper so future methods inherit the same context behavior instead of repeating it by hand

```go
ctx, cancel := config.OperationContextWithTimeout(30 * time.Second)
defer cancel()

_, err := client.ListTables(ctx, input)
```

```go
func queryWithContext(session *gocql.Session, config *engine.PluginConfig, stmt string, values ...any) *gocql.Query {
    return session.Query(stmt, values...).WithContext(config.OperationContext())
}
```

## Plugin File Organization

SQL-based plugins follow this structure (see `core/src/plugins/postgres/` as reference):
- `db.go` - Connection creation (implements DB method)
- `postgres.go` (or `mysql.go`, etc.) - Plugin struct, NewXxxPlugin(), database-specific queries
- `types.go` - Type definitions and alias map wrappers used by the plugin runtime
- `constraints.go` - Column constraint detection (optional override)

Source-owned editor/query metadata now lives in side-effect-free specs under
`core/src/sourcecatalog/specs/`. Plugins may read those specs, but they should
not register session metadata themselves.

GormPlugin base class (`core/src/plugins/gorm/`) provides:
- `plugin.go` - 40+ default method implementations
- `sqlbuilder.go` - SQL query building
- `errors.go` - ErrorHandler for user-friendly error messages
- `add.go`, `update.go`, `delete.go` - CRUD operations

## Adding a New Database

1. Create plugin directory in `core/src/plugins/`
2. Implement `PluginFunctions` interface (extend GormPlugin for SQL databases)
3. Add `init()` function calling `engine.RegisterPlugin(NewYourPlugin())` — the plugin self-registers when imported
4. Add a blank import in the entry point (`core/cmd/whodb/main.go`): `_ "github.com/clidey/whodb/core/src/plugins/yourplugin"`

## Key Methods to Override for SQL Plugins

```go
// Most SQL plugins override these:
GetAllSchemasQuery() string           // information_schema query for schemas
GetSchemaTableQuery() string          // Query for columns in a table
FormTableName(schema, table) string   // Default: "schema.table" (override for different behavior, e.g. SQLite ignores schema)
GetPlaceholder(index int) string      // $1 for Postgres, ? for MySQL
DB(config) (*gorm.DB, error)          // Connection with driver-specific config
GetLastInsertID(db *gorm.DB) (int64, error) // Default: returns 0 (override for MySQL, Postgres, SQLite)
```

## Session Metadata (types.go)

Each SQL plugin family must provide session metadata for editor/query-builder UI
through the source-owned specs in `core/src/sourcecatalog/specs/`. The shared
`core/src/sourcecatalog/metadata.go` file registers those specs centrally. This
metadata is the source of truth for:
- Valid operators (=, >=, LIKE, etc.)
- Type definitions (VARCHAR, INTEGER, etc.) with UI hints (hasLength, hasPrecision)
- Alias maps (INT → INTEGER, BOOL → BOOLEAN)

This metadata is exposed through the source-first GraphQL
`SourceSessionMetadata` query after login. **No fallbacks** - if the backend
doesn't provide it, the UI type selectors and query helpers will be broken.

Do not call `sourcecatalog.RegisterSessionMetadata(...)` from plugin `init()`
functions anymore. Keep plugin `init()` limited to runtime plugin registration
(`engine.RegisterPlugin(...)`). If a plugin needs to reuse the same alias map or
type definitions for runtime normalization, import them from
`core/src/sourcecatalog/specs/`.

Feature gating is not owned by session metadata. Public behavior
such as chat/query/graph surfaces and source object actions/views comes from the
source catalog contract in `core/src/sourcecatalog/catalog.go`.
The source contract is authoritative: `SourceContract.Surfaces`, `RootActions`,
`BrowsePath`, and `ObjectTypes.Actions` are used by the backend source adapter
to block unsupported operations and by the frontend to show or hide source
surfaces and object controls. Do not add source-name conditionals for behavior
that can be represented in the contract.

Frontend and CLI connection/presentation behavior also comes from the source
model now. Use `SourceType.Traits` for things like file-vs-network transport,
host input parsing, profile labeling, schema fidelity, and query UI options.
Do not reintroduce `DatabaseType` branches for those decisions.

Source metadata reliability also belongs in `SourceType.Traits`. Use
`TypeTraits.Metadata` to declare column, constraint, graph, and internal-object
filtering fidelity. Declare system schemas, internal collections, hidden
indices, or synthetic keys with `HiddenObjectNames` or `HiddenObjectPrefixes`
in the source catalog; the database adapter applies those rules consistently to
browse and graph metadata.

### types.go Structure

```go
package postgres

import (
    "github.com/clidey/whodb/core/src/common"
    sourcecatalogspecs "github.com/clidey/whodb/core/src/sourcecatalog/specs"
)

// AliasMap maps type aliases to canonical names (UPPERCASE keys and values)
var AliasMap = sourcecatalogspecs.PostgresAliasMap

// TypeDefinitions - canonical types shown in UI type selector
var TypeDefinitions = sourcecatalogspecs.PostgresTypeDefinitions

func NormalizeType(typeName string) string {
    return common.NormalizeTypeWithMap(typeName, AliasMap)
}
```

### Type Validation

Column type validation uses `engine.ValidateColumnType()` which checks against TypeDefinitions. Types not in TypeDefinitions will be rejected when adding columns.

## Quirks to Know

- SQLite doesn't use schemas - `FormTableName()` returns just table name
- PostgreSQL array types display with underscore prefix (`_text`)
- MySQL `GetDatabases()` returns `ErrUnsupported`
- Redis iterates through 16 database slots to discover databases

- Plugin architecture ensures clean code separation
