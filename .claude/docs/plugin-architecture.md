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

## Adding New Functionality

1. Add method to `PluginFunctions` interface in `core/src/engine/plugin.go`
2. Provide default implementation in base plugin (`GormPlugin` in `core/src/plugins/gorm/plugin.go`)
3. Override in specific plugins as needed
4. NoSQL plugins should return appropriate errors for SQL-specific features

## Plugin File Organization

SQL-based plugins follow this structure (see `core/src/plugins/postgres/` as reference):
- `db.go` - Connection creation (implements DB method)
- `postgres.go` (or `mysql.go`, etc.) - Plugin struct, NewXxxPlugin(), database-specific queries
- `constraints.go` - Column constraint detection (optional override)

GormPlugin base class (`core/src/plugins/gorm/`) provides:
- `plugin.go` - 40+ default method implementations
- `sqlbuilder.go` - SQL query building
- `errors.go` - ErrorHandler for user-friendly error messages
- `add.go`, `update.go`, `delete.go` - CRUD operations

## Adding a New Database

1. Create plugin directory in `core/src/plugins/` (CE) or `ee/core/src/plugins/` (EE)
2. Implement `PluginFunctions` interface (extend GormPlugin for SQL databases)
3. Register in `core/src/src.go` via `MainEngine.RegistryPlugin(yourplugin.NewYourPlugin())`
4. For EE: Register in `ee/core/src/plugins/init.go`

## Key Methods to Override for SQL Plugins

```go
// Most SQL plugins override these:
GetAllSchemasQuery() string           // information_schema query for schemas
GetSchemaTableQuery() string          // Query for columns in a table
FormTableName(schema, table) string   // "schema"."table" vs `schema`.`table`
GetPlaceholder(index int) string      // $1 for Postgres, ? for MySQL
DB(config) (*gorm.DB, error)          // Connection with driver-specific config
```

## Quirks to Know

- SQLite doesn't use schemas - `FormTableName()` returns just table name
- PostgreSQL array types display with underscore prefix (`_text`)
- MySQL `GetDatabases()` returns `ErrUnsupported`
- Redis iterates through 16 database slots to discover databases

## EE Compatibility

- EE uses `SetEEInitializer()` pattern to register plugins without modifying CE
- EE plugins in `ee/core/src/plugins/` automatically inherit interface methods
- Plugin architecture ensures clean CE/EE separation
