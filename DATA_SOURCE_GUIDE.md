# The Ultimate WhoDB Data Source Implementation Guide

This document is the **strict, authoritative protocol** for adding new data sources to WhoDB. The entire project relies on the architectural consistency defined here. **Do not deviate from these patterns.** Whether you are an AI assistant (Claude, Codex, Gemini) or a human developer, you must follow this guide exhaustively.

---

## How to Use This Guide

WhoDB supports **two implementation paths** for new data sources:

| Path | When to Use | Base Layer |
|------|-------------|------------|
| **A: Database Plugin** | SQL/NoSQL databases that speak a wire protocol | `engine.PluginFunctions` + `DatabaseConnector` adapter |
| **B: Source-First Connector** | Non-database sources (S3, APIs, queues, graph DBs, anything else) | `source.SourceConnector` + `source.SourceSession` interfaces directly |

Both paths share the same catalog registration, frontend integration, and GraphQL exposure. The only difference is how the backend session is implemented.

Read the **Core Architecture** section first, then follow the path that matches your source.

---

## Core Architecture

### Mental Model

```
┌─────────────┐     ┌──────────────┐     ┌─────────────────┐     ┌────────────┐
│  Frontend    │────▶│  GraphQL API │────▶│  Source Registry │────▶│  Driver    │
│  (catalog    │     │  (generic,   │     │  (TypeSpec +     │     │  (opens    │
│   driven)    │     │   source-    │     │   SourceConnector│     │   sessions)│
│              │     │   agnostic)  │     │   per DriverID)  │     │            │
└─────────────┘     └──────────────┘     └─────────────────┘     └────────────┘
                                                                       │
                                              ┌────────────────────────┤
                                              ▼                        ▼
                                    ┌──────────────────┐    ┌──────────────────┐
                                    │ Path A:           │    │ Path B:           │
                                    │ DatabaseConnector │    │ Custom Connector  │
                                    │ (DriverID =       │    │ (DriverID =       │
                                    │  "database")      │    │  your custom ID)  │
                                    │ adapts legacy     │    │ implements source  │
                                    │ PluginFunctions   │    │ interfaces directly│
                                    └──────────────────┘    └──────────────────┘
```

### Key Principles

1. **Source-First GraphQL API** — You do NOT add custom GraphQL queries or mutations for new data sources. The public API (`SourceTypes`, `SourceObjects`, `SourceRows`, `RunSourceQuery`, `SourceGraph`, etc.) is generic and works for all source types automatically.
2. **Plugin Self-Registration** — Plugins register themselves via `init()` functions. The frontend and API automatically adapt based on the plugin's declared catalog entries.
3. **CE vs. EE Strict Boundary** — CE (Community Edition) knows *nothing* about EE (Enterprise Edition). CE code must never contain `ee/` imports, references, or `if isEE` logic. EE extends CE purely through registries at boot time. Edition is controlled by which entry point is compiled (`core/cmd/whodb/main.go` for CE, `ee/cmd/whodb/main.go` for EE), not build tags.
4. **No Defensive Code** — Do not write fallback logic unless explicitly requested.
5. **No SQL Injection** — Use parameterized queries or `GetPlaceholder(index)`. Never use `fmt.Sprintf` for user-supplied SQL variables.
6. **Localization** — All user-facing strings must use `t()` with YAML keys. No hardcoded UI text.

---

## Reference: Source-First Type System

Before implementing anything, understand these types. Every value listed below is the **complete** set — do not invent new values.

### `source.Category` — Broad source classification
| Value | Use For |
|-------|---------|
| `CategoryDatabase` | SQL/NoSQL databases |
| `CategoryCache` | Redis, Memcached |
| `CategorySearch` | ElasticSearch, OpenSearch |
| `CategoryObjectStore` | S3, GCS, Azure Blob |
| `CategoryFileStore` | Filesystem-like sources |

### `source.Model` — Data organization model
| Value | Use For |
|-------|---------|
| `ModelRelational` | Tables with rows and columns |
| `ModelDocument` | JSON documents / collections |
| `ModelKeyValue` | Key-value stores |
| `ModelSearch` | Full-text search indexes |
| `ModelGraph` | Graph databases (nodes + edges) |
| `ModelObject` | Object/blob storage |

### `source.Surface` — UI experiences enabled
| Value | Description |
|-------|-------------|
| `SurfaceBrowser` | Object tree navigation (always included) |
| `SurfaceQuery` | SQL/query editor |
| `SurfaceChat` | AI chat interface |
| `SurfaceGraph` | Relationship visualization |

### `source.ObjectKind` — Browseable entity types
| Value | Use For |
|-------|---------|
| `ObjectKindDatabase` | Top-level container |
| `ObjectKindSchema` | Namespace (Postgres schemas, Oracle schemas) |
| `ObjectKindTable` | Tabular data |
| `ObjectKindView` | Read-only table/view |
| `ObjectKindCollection` | Document collection (MongoDB) |
| `ObjectKindIndex` | Search index (Elasticsearch) |
| `ObjectKindKey` | Cache key (Redis) |
| `ObjectKindItem` | Item (Memcached) |
| `ObjectKindFunction` | Function |
| `ObjectKindProcedure` | Stored procedure |
| `ObjectKindTrigger` | Trigger |
| `ObjectKindSequence` | Sequence |

### `source.Action` — Operations per object kind
| Value | Description |
|-------|-------------|
| `ActionBrowse` | Navigate into / list children |
| `ActionInspect` | View metadata |
| `ActionViewRows` | View tabular data |
| `ActionViewContent` | View blob/text content |
| `ActionViewDefinition` | View DDL/source definition |
| `ActionCreateChild` | Create child objects |
| `ActionDelete` | Delete object |
| `ActionInsertData` | Insert rows/documents |
| `ActionUpdateData` | Update rows/documents |
| `ActionDeleteData` | Delete rows/documents |
| `ActionImportData` | Bulk import |
| `ActionGenerateMockData` | Generate synthetic data |
| `ActionExecute` | Execute an action |
| `ActionViewGraph` | Visualize as graph |

### `source.DataShape` — How data is rendered
| Value | Use For |
|-------|---------|
| `DataShapeTabular` | Row/column grid |
| `DataShapeDocument` | JSON/document viewer |
| `DataShapeContent` | Blob/text content |
| `DataShapeGraph` | Graph visualization |
| `DataShapeMetadata` | Metadata-only display |

### `source.View` — Available rendering views
`ViewGrid`, `ViewJSON`, `ViewText`, `ViewSQL`, `ViewBinary`, `ViewMetadata`, `ViewGraph`

### `source.ConnectionTransport` — How the source is reached
| Value | Use For |
|-------|---------|
| `ConnectionTransportNetwork` | TCP/HTTP to remote host |
| `ConnectionTransportFile` | Local file (SQLite, DuckDB) |
| `ConnectionTransportBridge` | Bridge/sidecar transport (EE) |

### `ConnectionTraits.SupportsCustomCAContent` — SSL CA content upload
A `bool` on `ConnectionTraits`. When `true` (the default for most network sources), the frontend SSL config allows the user to paste/upload a custom CA certificate. Set to `false` for sources where the system CA bundle is always used (e.g., MSSQL, Oracle with system-CA-only mode). See the EE `relationalDatabaseSchemaSystemCAFamily()` and `relationalSchemaSystemCAFamily()` helpers.

### `source.HostInputMode` — Hostname field behavior
`HostInputModeNone`, `HostInputModeHostname`, `HostInputModeHostnameOrURL`

### `source.HostInputURLParser` — URL auto-parsing
`HostInputURLParserNone`, `HostInputURLParserPostgres`, `HostInputURLParserMongoSRV`

### `source.ProfileLabelStrategy` — How saved profiles are labeled
`ProfileLabelStrategyDefault`, `ProfileLabelStrategyHostname`, `ProfileLabelStrategyDatabase`

### `source.SchemaFidelity` — Metadata precision
| Value | Use For |
|-------|---------|
| `SchemaFidelityExact` | Schema from system tables (SQL databases) |
| `SchemaFidelitySampled` | Schema inferred from data samples (MongoDB, Elasticsearch) |

### `source.QueryExplainMode` — Query plan inspection
`QueryExplainModeNone`, `QueryExplainModeExplain`, `QueryExplainModeExplainAnalyze`, `QueryExplainModeExplainPipeline`

---

## Reference: Source Session Interfaces

GraphQL resolvers type-assert your session to check what it can do. Implement only the interfaces that apply to your source. At minimum, implement `SourceSession`.

### Type Ownership

The `source` package owns all shared data types (`Column`, `Record`, `RowsResult`, `GraphUnit`, `ChatMessage`, `SSLStatus`, `TypeDefinition`, etc.). The `engine` package re-exports them as type aliases (in `core/src/engine/aliases.go`) so that existing database plugins can continue using `engine.Column` etc. without changes. When writing new code, prefer importing from `source` directly.

```go
// Required — every session must implement this
type SourceSession interface {
    Metadata(ctx context.Context) (*SessionMetadata, error)
}

// Object browsing (almost always needed)
type SourceBrowser interface {
    ListObjects(ctx context.Context, parent *ObjectRef, kinds []ObjectKind) ([]Object, error)
    GetObject(ctx context.Context, ref ObjectRef) (*Object, error)
}

// Reading tabular data
type TabularReader interface {
    ReadRows(ctx context.Context, ref ObjectRef, where *query.WhereCondition, sort []*query.SortCondition, pageSize int, pageOffset int) (*RowsResult, error)
    Columns(ctx context.Context, ref ObjectRef) ([]Column, error)
    ColumnsBatch(ctx context.Context, refs []ObjectRef) ([]ObjectColumns, error)
}

// Per-column constraint metadata (uniqueness, defaults, checks)
type ColumnConstraintReader interface {
    ColumnConstraints(ctx context.Context, ref ObjectRef) (map[string]map[string]any, error)
}

// Reading blob/text content
type ContentReader interface {
    ReadContent(ctx context.Context, ref ObjectRef) (*ContentResult, error)
}

// Connectivity check
type AvailabilityChecker interface {
    IsAvailable(ctx context.Context) bool
}

// Query execution
type QueryRunner interface {
    RunQuery(ctx context.Context, query string, params ...any) (*RowsResult, error)
}

// Streaming query execution (row-by-row output)
type StreamQueryRunner interface {
    RunQueryStream(ctx context.Context, query string, writer QueryStreamWriter, params ...any) error
}
type QueryStreamWriter interface {
    WriteColumns(columns []Column) error
    WriteRow(row []string) error
}

// Multi-statement scripts
type ScriptRunner interface {
    RunScript(ctx context.Context, script string, multiStatement bool, params ...any) (*RowsResult, error)
}

// Graph visualization
type GraphReader interface {
    ReadGraph(ctx context.Context, ref *ObjectRef) ([]GraphUnit, error)
}

// AI chat
type SourceAssistant interface {
    Reply(ctx context.Context, ref *ObjectRef, previousConversation string, query string) ([]*ChatMessage, error)
}

// CRUD mutations
type ObjectManager interface {
    CreateObject(ctx context.Context, parent *ObjectRef, name string, fields []Record) (bool, error)
    UpdateObject(ctx context.Context, ref ObjectRef, values map[string]string, updatedColumns []string) (bool, error)
    AddRow(ctx context.Context, ref ObjectRef, values []Record) (bool, error)
    DeleteRow(ctx context.Context, ref ObjectRef, values map[string]string) (bool, error)
}

// Dynamic connection field options
type ConnectionFieldOptionsReader interface {
    ConnectionFieldOptions(ctx context.Context, fieldKey string, values map[string]string) ([]string, error)
}

// Data export
type TabularExporter interface {
    ExportRows(ctx context.Context, ref ObjectRef, writer func([]string) error, selectedRows []map[string]any) error
}
type NDJSONExporter interface {
    ExportRowsNDJSON(ctx context.Context, ref ObjectRef, writer func(string) error, selectedRows []map[string]any) error
}

// SSL/TLS status
type SecurityReader interface {
    SSLStatus(ctx context.Context) (*SSLStatus, error)
}

// Data import (returns ImportResult with RowsImported count)
type DataImporter interface {
    ImportData(ctx context.Context, ref ObjectRef, request ImportRequest) (*ImportResult, error)
}

// Mock data generation
type MockDataManager interface {
    GenerateMockData(ctx context.Context, ref ObjectRef, rowCount int, fkDensityRatio int, overwriteExisting bool) (*MockDataGenerationResult, error)
    AnalyzeMockDataDependencies(ctx context.Context, ref ObjectRef, rowCount int, fkDensityRatio int) (*MockDataDependencyAnalysis, error)
}

// Query suggestions for the UI
type QuerySuggester interface {
    QuerySuggestions(ctx context.Context, ref *ObjectRef) ([]QuerySuggestion, error)
}

// Lifecycle management (optional — implement on your SourceConnector)
type SessionInvalidator interface {
    Invalidate(ctx context.Context, spec TypeSpec, credentials *Credentials) error
}
type DriverShutdowner interface {
    Shutdown(ctx context.Context) error
}
```

---

## Reference: ObjectType Builder Helpers

When defining `ObjectTypes` in your `FamilySpec`, use these helpers from `sourcecatalog`. Do NOT construct `source.ObjectType` structs manually.

```go
// Container objects (Database, Schema, Keyspace, etc.)
// createChild=true adds ActionCreateChild
metadataObjectType(kind, singularLabel, pluralLabel, createChild bool)

// Fully mutable tabular objects (Tables)
// Actions: Inspect, ViewRows, InsertData, UpdateData, DeleteData, ImportData, GenerateMockData
tabularObjectType(kind, singularLabel, pluralLabel)

// Read-only tabular objects (Views)
// Actions: Inspect, ViewRows, ViewDefinition
tabularReadOnlyObjectType(kind, singularLabel, pluralLabel)

// Document objects (Collections, Indexes)
// Actions: Inspect, ViewRows, InsertData, UpdateData, DeleteData, [extraActions...]
documentObjectType(kind, singularLabel, pluralLabel, extraActions ...source.Action)

// Fully mutable key-value objects (Redis keys)
// Actions: Inspect, ViewRows, InsertData, UpdateData, DeleteData
keyValueObjectType(kind, singularLabel, pluralLabel)

// Existing-only mutable objects (Memcached items — no insert, only update/delete)
// Actions: Inspect, ViewRows, UpdateData, DeleteData
keyValueExistingMutableObjectType(kind, singularLabel, pluralLabel)
```

### Trait Builder Helpers

```go
// Network source with exact schema
networkTraits(hostInputMode, urlParser) source.TypeTraits

// Network source with sampled schema (MongoDB, Elasticsearch)
sampledNetworkTraits(hostInputMode, urlParser, profileLabelStrategy) source.TypeTraits

// File-based source (SQLite, DuckDB)
fileTraits(profileLabelStrategy) source.TypeTraits

// Bridge/sidecar source (EE only)
bridgeTraits() source.TypeTraits  // EE only, in ee/core/src/sourcecatalog/register.go
```

---

## Path A: Adding a Database Plugin

Use this path for SQL/NoSQL databases. This is the most common path.

### Phase 1: Define the Engine Constant

Every data source needs a unique `DatabaseType` identifier.

- **CE sources**: Add to `core/src/engine/engine.go`
- **EE sources**: Add to `ee/core/src/engine/types.go`

```go
const DatabaseType_MyNewDB DatabaseType = "MyNewDB"
```

The string value must be globally unique and becomes the source's ID everywhere.

### Phase 2: Implement the Plugin

Create your plugin package in `core/src/plugins/<name>/` (CE) or `ee/core/src/plugins/<name>/` (EE).

#### For SQL databases: Embed `GormPlugin`

```go
package mynewdb

import (
    "github.com/clidey/whodb/core/src/engine"
    gorm_plugin "github.com/clidey/whodb/core/src/plugins/gorm"
)

type MyNewDBPlugin struct {
    gorm_plugin.GormPlugin
}

func init() {
    engine.RegisterPlugin(NewMyNewDBPlugin())
}

func NewMyNewDBPlugin() *engine.Plugin {
    plugin := &MyNewDBPlugin{}
    plugin.Type = engine.DatabaseType_MyNewDB
    plugin.PluginFunctions = plugin
    plugin.GormPluginFunctions = plugin
    return &plugin.Plugin
}
```

**You MUST implement these `GormPluginFunctions` methods** (the base provides defaults for most `PluginFunctions` methods, but these have no default):

| Method | Purpose |
|--------|---------|
| `DB(config *engine.PluginConfig) (*gorm.DB, error)` | Open a GORM connection |
| `GetPlaceholder(index int) string` | SQL parameter placeholder (e.g., `$1` for Postgres, `?` for MySQL) |
| `GetTableInfoQuery() string` | SQL query listing tables with metadata for a schema |
| `GetStorageUnitExistsQuery() string` | SQL query checking if a table exists |
| `GetPrimaryKeyColQuery() string` | SQL query finding primary key columns |
| `GetAllSchemasQuery() string` | SQL query listing all schemas |
| `GetCreateTableQuery(db, schema, storageUnit, columns) string` | DDL for CREATE TABLE |
| `GetSupportedOperators() map[string]string` | Map of display→SQL operators |
| `GetGraphQueryDB(db, schema) *gorm.DB` | Query builder for graph/FK relationships |
| `GetTableNameAndAttributes(rows) (string, []Record)` | Parse table info query results |
| `ParseConnectionConfig(config) (*ConnectionInput, error)` | Parse credentials into connection params |

**Size attribute conventions** — `GetTableNameAndAttributes` (and any custom `GetStorageUnits` implementation) must follow these rules for size-related attributes:

| Rule | Detail |
|------|--------|
| **Emit raw bytes** | Size values MUST be numeric strings representing bytes (e.g., `"1048576"` not `"1 MB"`). The frontend owns formatting via `formatBytes()`. |
| **Standard key names** | Use `"Total Size"` (data + indexes) and/or `"Data Size"` (data only). Do NOT invent custom names like `"Size (KB)"` or `"Storage Size"`. |
| **Use `sql.NullInt64`** | Scan size columns as `sql.NullInt64`. Only append the attribute when `.Valid` is true — this gracefully handles tables without allocation data. |
| **No backend formatting** | Never use `pg_size_pretty()`, `formatReadableSize()`, `ROUND(... / 1024)`, or `fmt.Sprintf("%.2f MB", ...)` in size queries. |
| **Non-byte counts** | For counts that are NOT bytes (e.g., Redis element count), use a distinct key like `"Entries"` or `"Item Count"` — these must NOT be named `"Size"`. |

Example (`GetTableNameAndAttributes`):
```go
func (p *MyPlugin) GetTableNameAndAttributes(rows *sql.Rows) (string, []engine.Record) {
    var tableName, tableType string
    var totalSize, dataSize sql.NullInt64
    if err := rows.Scan(&tableName, &tableType, &totalSize, &dataSize); err != nil {
        return "", nil
    }
    attributes := []engine.Record{{Key: "Type", Value: tableType}}
    if totalSize.Valid {
        attributes = append(attributes, engine.Record{Key: "Total Size", Value: fmt.Sprintf("%d", totalSize.Int64)})
    }
    if dataSize.Valid {
        attributes = append(attributes, engine.Record{Key: "Data Size", Value: fmt.Sprintf("%d", dataSize.Int64)})
    }
    return tableName, attributes
}
```

**You SHOULD override these for correctness** (GormPlugin provides defaults):

| Method | Default | Override When |
|--------|---------|---------------|
| `NormalizeType(typeName) string` | Strips length, uppercases | Your DB has type aliases (e.g., `INT4`→`INTEGER`) |
| `ConvertStringValue(value, columnType, isNullable) (any, error)` | Generic conversion | Your DB has custom types |
| `GetColumnConstraints(config, schema, storageUnit) (map, error)` | Empty map | Your DB has column constraints |
| `GetForeignKeyRelationships(config, schema, storageUnit) (map, error)` | Empty map | Your DB has foreign keys |
| `GetRowsOrderBy(db, schema, storageUnit) string` | Empty string | Your DB needs default ordering |
| `MarkGeneratedColumns(config, schema, storageUnit, columns) error` | No-op | Your DB has auto-increment/computed columns |
| `GetColumnCodec(columnType) ColumnCodec` | nil (use default) | Your DB has custom scan/format for columns |
| `IsGeometryType(columnType) bool` | false | Your DB has spatial types |
| `IsArrayType(columnType) bool` | false | Your DB has array types |
| `ShouldCheckRowsAffected() bool` | true | Your DB driver doesn't report affected rows |
| `GetMaxBulkInsertParameters() int` | 65535 | Your DB has a lower param limit |
| `GetLastInsertID(db) (int64, error)` | 0, nil | Your DB supports returning last insert ID |

**The base `GormPlugin` provides these `PluginFunctions` implementations for free**:
`GetStorageUnits`, `StorageUnitExists`, `GetAllSchemas`, `GetRows`, `GetRowCount`, `GetColumnsForTable`, `WithTransaction`, `MarkGeneratedColumns`, `GetForeignKeyRelationships`

**You MUST implement these `PluginFunctions` methods yourself** (no GormPlugin default):

| Method | Purpose |
|--------|---------|
| `GetDatabases(config) ([]string, error)` | List available databases |
| `IsAvailable(ctx, config) bool` | Verify connectivity |
| `AddStorageUnit(config, schema, storageUnit, fields) (bool, error)` | CREATE TABLE |
| `UpdateStorageUnit(config, schema, storageUnit, values, updatedColumns) (bool, error)` | UPDATE row |
| `AddRow(config, schema, storageUnit, values) (bool, error)` | INSERT row |
| `AddRowReturningID(config, schema, storageUnit, values) (int64, error)` | INSERT returning ID |
| `BulkAddRows(config, schema, storageUnit, rows) (bool, error)` | Bulk INSERT |
| `DeleteRow(config, schema, storageUnit, values) (bool, error)` | DELETE row |
| `GetGraph(config, schema) ([]GraphUnit, error)` | FK relationship graph |
| `RawExecute(config, query, params...) (*GetRowsResult, error)` | Execute raw SQL |
| `Chat(config, schema, previousConversation, query) ([]*ChatMessage, error)` | AI chat |
| `ExportData(config, schema, storageUnit, writer, selectedRows) error` | Export rows |
| `FormatValue(val any) string` | Format a value for display |
| `GetSSLStatus(config) (*SSLStatus, error)` | SSL/TLS connection status |
| `ClearTableData(config, schema, storageUnit) (bool, error)` | Truncate table |
| `NullifyFKColumn(config, schema, storageUnit, column) error` | Nullify FK column |

**Connection lifecycle**: Always use `plugins.WithConnection(config, p.DB, func(db *gorm.DB) ...)` for all database operations. This handles connection pooling and lifecycle.

**Error handling**: Use `ErrorHandler` from `core/src/plugins/gorm/errors.go` for user-friendly error messages. Call `p.InitPlugin()` in your constructor or first method to initialize it.

#### For non-SQL databases: Embed `BasePlugin`

For databases that don't use SQL (MongoDB, Redis, Elasticsearch, etc.), embed `engine.BasePlugin` and override only the methods you support. `BasePlugin` (`core/src/engine/base_plugin.go`) provides default implementations for all `PluginFunctions` methods — user-facing operations return `errors.ErrUnsupported`, internal operations return empty results.

```go
type MyNoSQLPlugin struct {
    engine.BasePlugin  // Provides defaults for all 27 PluginFunctions methods
}

// Override only what your source supports
func (p *MyNoSQLPlugin) GetDatabases(config *engine.PluginConfig) ([]string, error) { ... }
func (p *MyNoSQLPlugin) IsAvailable(ctx context.Context, config *engine.PluginConfig) bool { ... }
func (p *MyNoSQLPlugin) GetStorageUnits(config *engine.PluginConfig, schema string) ([]engine.StorageUnit, error) { ... }
// ... etc
```

See existing examples:
- MongoDB: `core/src/plugins/mongodb/`
- Redis: `core/src/plugins/redis/`
- Elasticsearch: `core/src/plugins/elasticsearch/`
- Memcached: `core/src/plugins/memcached/`
- DynamoDB (EE): `ee/core/src/plugins/dynamodb/`
- Cassandra (EE): `ee/core/src/plugins/cassandra/`

### Phase 3: Register the Plugin Entry Point

Add a blank import so `init()` runs:

- **CE**: `core/cmd/whodb/main.go`
- **EE**: `ee/cmd/whodb/main.go` (import BOTH CE and EE plugins)

```go
_ "github.com/clidey/whodb/core/src/plugins/mynewdb"
```

### Phase 4: Register in the Database Catalog

This controls the connection UI form and which fields are shown.

- **CE**: Add to the `catalog` slice in `core/src/dbcatalog/catalog.go`
- **EE**: Call `corecatalog.Register()` in `ee/core/src/dbcatalog/register.go`

```go
{
    ID:         engine.DatabaseType_MyNewDB,
    Label:      "My New DB",
    PluginType: engine.DatabaseType_MyNewDB,
    Extra: map[string]source.ConnectionExtraField{
        "Port": {DefaultValue: "5432"},
    },
    Fields: FieldVisibility{
        Hostname: true, Username: true, Password: true, Database: true,
    },
    RequiredFields: FieldRequirements{
        Hostname: true, Username: true, Password: true, Database: true,
    },
    SSLModes: sslModesFor(engine.DatabaseType_MyNewDB),  // Or omit if no SSL
}
```

**`ConnectableDatabase` fields reference:**

| Field | Type | Purpose |
|-------|------|---------|
| `ID` | `engine.DatabaseType` | Unique identifier — becomes the source type ID |
| `Label` | `string` | UI display name |
| `PluginType` | `engine.DatabaseType` | Which plugin handles this. Can differ from ID for aliases (e.g., Valkey→Redis) |
| `Extra` | `map[string]source.ConnectionExtraField` | Advanced field definitions and defaults (Port, Region, etc.) |
| `Fields` | `FieldVisibility` | Which standard fields are shown: `Hostname`, `Username`, `Password`, `Database`, `SearchPath` |
| `RequiredFields` | `FieldRequirements` | Which fields are mandatory: `Hostname`, `Username`, `Password`, `Database` |
| `IsAWSManaged` | `bool` | True for AWS managed services (ElastiCache, DocumentDB) |
| `SSLModes` | `[]source.SSLModeInfo` | SSL mode options. Use `sslModesFor()` helper or define custom |

`ConnectionExtraField` keeps advanced-field metadata in one place:

- `DefaultValue` sets the default UI value.
- `Kind` is optional and controls how the field renders (`Boolean`, `Password`, `FilePath`, etc.).
- `Required` is optional and marks an advanced field as mandatory in the shared connection form.
- `LabelKey` and `PlaceholderKey` are optional overrides for localization keys.

**Wire-compatible alias pattern**: If your source is wire-compatible with an existing plugin (e.g., Valkey is Redis-compatible), set `ID` to the new name but `PluginType` to the existing plugin:

```go
{
    ID:         engine.DatabaseType_Valkey,    // New unique ID
    Label:      "Valkey",
    PluginType: engine.DatabaseType_Redis,     // Reuses Redis plugin
    Extra: map[string]source.ConnectionExtraField{
        "Port": {DefaultValue: "6379"},
    },
    // ...
}
```

### Phase 5: Register the Source Family Spec

This defines browsing hierarchy, capabilities, and actions.

- **CE**: Add to the `familySpecs` map in `core/src/sourcecatalog/catalog.go`
- **EE**: Call `coresourcecatalog.RegisterFamilySpec()` in `ee/core/src/sourcecatalog/register.go`

```go
"MyNewDB": {
    Category:       source.CategoryDatabase,
    Traits:         networkTraits(source.HostInputModeHostname, source.HostInputURLParserNone),
    Model:          source.ModelRelational,
    Surfaces:       []source.Surface{source.SurfaceBrowser, source.SurfaceQuery, source.SurfaceChat, source.SurfaceGraph},
    BrowsePath:     []source.ObjectKind{source.ObjectKindDatabase, source.ObjectKindSchema, source.ObjectKindTable},
    DefaultObject:  source.ObjectKindTable,
    GraphScopeKind: ptr(source.ObjectKindSchema),
    ObjectTypes: []source.ObjectType{
        metadataObjectType(source.ObjectKindDatabase, "Database", "Databases", true),
        metadataObjectType(source.ObjectKindSchema, "Schema", "Schemas", true),
        tabularObjectType(source.ObjectKindTable, "Table", "Tables"),
        tabularReadOnlyObjectType(source.ObjectKindView, "View", "Views"),
    },
},
```

**Choosing your `BrowsePath`** — this controls the object drill-down hierarchy:

| Pattern | Example | BrowsePath |
|---------|---------|-----------|
| Database → Schema → Table | Postgres, MSSQL | `[Database, Schema, Table]` |
| Database → Table | MySQL, ClickHouse | `[Database, Table]` |
| Schema → Table | Oracle | `[Schema, Table]` |
| Table only | SQLite, Athena | `[Table]` (add `RootActions: [Browse, CreateChild]`) |
| Database → Collection | MongoDB | `[Database, Collection]` |
| Database → Key | Redis | `[Database, Key]` |
| Keyspace → Table | Cassandra | Use ObjectKindSchema with label "Keyspace" |
| Index only | Elasticsearch | `[Index]` |

**`GraphScopeKind`**: Set to the ObjectKind that scopes relationship visualization. Use `ptr(source.ObjectKindSchema)` for schema-scoped, `ptr(source.ObjectKindDatabase)` for database-scoped, or `nil` for no graph support.

### Phase 6: Register Session Metadata

This provides type definitions, operators, and aliases for the query builder UI.

- **CE**: Add to `registerSessionMetadata()` in `core/src/sourcecatalog/metadata.go`
- **EE**: Add to `Register()` in `ee/core/src/sourcecatalog/register.go`

```go
RegisterSessionMetadata(
    string(engine.DatabaseType_MyNewDB),
    SessionMetadataFromOperatorMap(specs.MyNewDBTypeDefinitions, specs.MyNewDBSupportedOperators, specs.MyNewDBAliasMap),
)
```

**You must define these in the specs package**:
- **CE**: `core/src/sourcecatalog/specs/sql.go` (or a new file for non-SQL)
- **EE**: `ee/core/src/sourcecatalog/specs/sql.go`

```go
// Operators: map of display name → SQL operator
var MyNewDBSupportedOperators = map[string]string{
    "=": "=", "!=": "!=", ">": ">", "<": "<",
    ">=": ">=", "<=": "<=",
    "LIKE": "LIKE", "IN": "IN", "NOT IN": "NOT IN",
    "IS NULL": "IS NULL", "IS NOT NULL": "IS NOT NULL",
    "BETWEEN": "BETWEEN", "NOT BETWEEN": "NOT BETWEEN",
}

// Type aliases: map alias → canonical name
var MyNewDBAliasMap = map[string]string{
    "INT4": "INTEGER",
    "BOOL": "BOOLEAN",
}

// Type definitions for the column type selector UI
var MyNewDBTypeDefinitions = []engine.TypeDefinition{
    {ID: "INTEGER", Label: "integer", Category: engine.TypeCategoryNumeric},
    {ID: "VARCHAR", Label: "varchar", HasLength: true, DefaultLength: engine.IntPtr(255), Category: engine.TypeCategoryText},
    {ID: "BOOLEAN", Label: "boolean", Category: engine.TypeCategoryBoolean},
    {ID: "TIMESTAMP", Label: "timestamp", Category: engine.TypeCategoryDatetime},
    {ID: "JSON", Label: "json", Category: engine.TypeCategoryJSON},
}
```

### Phase 7: Frontend Integration

Because of the catalog system, the UI is mostly auto-generated. You only need icons and locale strings.

#### 7a. Frontend Source Type Discovery

The frontend discovers all source types dynamically from the backend via the `SourceTypes` GraphQL query. There is a `DatabaseType` constant in `frontend/src/config/source-types.ts` for convenience references to well-known CE types, but **it is not required for new sources to appear**. The frontend renders any source type returned by the backend catalog.

If you want to add a convenience constant (optional):

```typescript
export const DatabaseType = {
    // ... existing types
    MyNewDb: "MyNewDB",  // Must exactly match the backend Go constant string
} as const;
```

#### 7b. SVG Icon Registration

- **CE**: Add to the `ceLogos` object in `frontend/src/components/icons.tsx`
- **EE**: Add to `eeLogos` in `ee/frontend/src/icons.tsx`

```tsx
MyNewDB: <svg>...</svg>,
```

The key MUST exactly match your `DatabaseType` string (the `ID` field from the catalog).

Icon resolution order: `Icons.Logos[sourceId]` → `Icons.Logos[connector]` → empty span.

#### 7c. Localization Strings

Add to `frontend/src/locales/common.yaml` (CE) or `ee/frontend/src/locales/` (EE):

```yaml
en_US:
  myNewDBDesc: "Description for cloud provider integration"
```

#### 7d. Custom Connection Form (Rarely Needed)

Only needed if your source requires a heavily customized connection form that doesn't fit the standard field-based layout (e.g., DynamoDB's AWS region picker).

For normal database sources, do not build a bespoke form first. Prefer the
backend-declared `connectionFields` and `sslModes` contract and let the shared
frontend connection form render it. CE login and EE platform create/edit now
share the same advanced-field and SSL rendering path, so generic field-based
sources should work in both places without extra frontend code.

- **CE**: Register via `registerSourceTypeOverrides()` in `frontend/src/config/source-registry.ts`
- **EE**: Register in `ee/frontend/src/config.tsx` via `eeSourceTypeOverrides` array, which is passed to `registerSourceTypeOverrides()` in `ee/frontend/src/register.ts`

```typescript
const override: SourceTypeOverride = {
    id: "MyNewDB",
    customFormRenderer: MyNewDBLoginForm,
};
```

Use `customFormRenderer` only when the standard field-based flow is genuinely
insufficient.

The frontend `decorateSourceType()` function auto-derives all capability flags (`supportsChat`, `supportsGraph`, `supportsScratchpad`, `supportsSchema`, `supportsDatabaseSwitching`, `supportsMockData`, etc.) from the backend contract. Do not set these manually — they come from `source.Contract.Surfaces`, `BrowsePath`, and `ObjectTypes`.

---

## Path B: Adding a Source-First Connector

Use this path for non-database sources (S3, APIs, queues, graph databases) that don't fit the `PluginFunctions` interface.

### Phase 1: Implement SourceConnector and SourceSession

Create your connector package in `core/src/sources/<name>/` (CE) or `ee/core/src/sources/<name>/` (EE).

```go
package s3source

import (
    "context"
    "github.com/clidey/whodb/core/src/source"
)

const driverID = "s3"

func init() {
    source.RegisterDriver(driverID, &S3Connector{})
}

type S3Connector struct{}

func (c *S3Connector) Open(ctx context.Context, spec source.TypeSpec, credentials *source.Credentials) (source.SourceSession, error) {
    // Parse credentials from spec.ConnectionFields mapping
    region := credentials.Values["Region"]
    bucket := credentials.Values["Bucket"]
    // ... create client
    return &S3Session{client: client, spec: spec}, nil
}

type S3Session struct {
    client *s3.Client
    spec   source.TypeSpec
}

// Required: session metadata
func (s *S3Session) Metadata(ctx context.Context) (*source.SessionMetadata, error) {
    return &source.SessionMetadata{
        SourceType: s.spec.ID,
    }, nil
}

// Implement source.SourceBrowser
func (s *S3Session) ListObjects(ctx context.Context, parent *source.ObjectRef, kinds []source.ObjectKind) ([]source.Object, error) {
    // List buckets or objects within a bucket
}

func (s *S3Session) GetObject(ctx context.Context, ref source.ObjectRef) (*source.Object, error) {
    // Get one object by reference
}

// Implement source.ContentReader for viewing object content
func (s *S3Session) ReadContent(ctx context.Context, ref source.ObjectRef) (*source.ContentResult, error) {
    // Download and return object content
}

// Implement source.AvailabilityChecker
func (s *S3Session) IsAvailable(ctx context.Context) bool {
    // Check connectivity
}
```

### Phase 2: Register the TypeSpec

In the same `init()` or a separate registration file, register the full `TypeSpec`:

```go
func init() {
    source.RegisterDriver(driverID, &S3Connector{})
    source.RegisterType(source.TypeSpec{
        ID:        "S3",
        Label:     "Amazon S3",
        DriverID:  driverID,           // Must match RegisterDriver ID
        Connector: "S3",               // Used for icon lookup fallback
        Category:  source.CategoryObjectStore,
        Traits: source.TypeTraits{
            Connection: source.ConnectionTraits{
                Transport:     source.ConnectionTransportNetwork,
                HostInputMode: source.HostInputModeNone,
            },
            Presentation: source.PresentationTraits{
                ProfileLabelStrategy: source.ProfileLabelStrategyDefault,
                SchemaFidelity:       source.SchemaFidelitySampled,
            },
        },
        ConnectionFields: []source.ConnectionField{
            {
                Key:             "Region",
                Kind:            source.ConnectionFieldKindText,
                Section:         source.ConnectionFieldSectionPrimary,
                Required:        true,
                LabelKey:        "region",
                PlaceholderKey:  "enterRegion",
                CredentialField: source.CredentialFieldAdvanced,
                AdvancedKey:     "Region",
            },
            {
                Key:             "Access Key",
                Kind:            source.ConnectionFieldKindText,
                Section:         source.ConnectionFieldSectionPrimary,
                Required:        true,
                LabelKey:        "accessKey",
                CredentialField: source.CredentialFieldUsername,
            },
            {
                Key:             "Secret Key",
                Kind:            source.ConnectionFieldKindPassword,
                Section:         source.ConnectionFieldSectionPrimary,
                Required:        true,
                LabelKey:        "secretKey",
                CredentialField: source.CredentialFieldPassword,
            },
        },
        Contract: source.Contract{
            Model:             source.ModelObject,
            Surfaces:          []source.Surface{source.SurfaceBrowser},
            RootActions:       []source.Action{source.ActionBrowse},
            BrowsePath:        []source.ObjectKind{source.ObjectKindDatabase, source.ObjectKindItem},
            DefaultObjectKind: source.ObjectKindItem,
            ObjectTypes: []source.ObjectType{
                {
                    Kind:          source.ObjectKindDatabase,
                    DataShape:     source.DataShapeMetadata,
                    Actions:       []source.Action{source.ActionBrowse},
                    Views:         []source.View{source.ViewMetadata},
                    SingularLabel: "Bucket",
                    PluralLabel:   "Buckets",
                },
                {
                    Kind:          source.ObjectKindItem,
                    DataShape:     source.DataShapeContent,
                    Actions:       []source.Action{source.ActionInspect, source.ActionViewContent},
                    Views:         []source.View{source.ViewText, source.ViewBinary, source.ViewMetadata},
                    SingularLabel: "Object",
                    PluralLabel:   "Objects",
                },
            },
        },
    })
}
```

### Phase 3: Entry Point Import

Add the blank import:

```go
_ "github.com/clidey/whodb/core/src/sources/s3"
```

### Phase 4: Frontend Integration

Same as Path A Phase 7 — add icon, locale strings, and optionally a custom form.

### Key Differences from Path A

| Aspect | Path A (Database Plugin) | Path B (Source-First) |
|--------|--------------------------|----------------------|
| DriverID | `"database"` (shared adapter) | Custom (e.g., `"s3"`, `"kafka"`) |
| Session impl | `DatabaseSession` adapter wraps `PluginFunctions` | You implement `SourceSession` + interfaces directly |
| Catalog | Use `dbcatalog` + `sourcecatalog.FamilySpec` | Register `TypeSpec` directly via `source.RegisterType()` |
| Connection fields | Derived from `FieldVisibility` + `Extra` map | Defined explicitly in `TypeSpec.ConnectionFields` |
| Object hierarchy | Adapter maps `GetDatabases`/`GetAllSchemas`/`GetStorageUnits` to `BrowsePath` | You control `ListObjects` directly |

---

## Phase: SSL Mode Registration (Both Paths)

If your source supports SSL/TLS, define SSL modes.

### Using existing SSL mode helpers

In `core/src/dbcatalog/ssl_modes.go`, add your database to `sslModesFor()`, or define modes inline:

```go
SSLModes: []source.SSLModeInfo{
    {Value: "disable", Label: "Disabled", Description: "No SSL"},
    {Value: "require", Label: "Required", Description: "SSL required, no verification"},
    {Value: "verify-full", Label: "Verify Full", Description: "SSL with full certificate verification"},
},
```

### For Path B sources

Include `SSLModes` directly in your `TypeSpec`:

```go
source.RegisterType(source.TypeSpec{
    // ...
    SSLModes: []source.SSLModeInfo{
        {Value: "disable", Label: "Disabled"},
        {Value: "enable", Label: "Enabled"},
    },
})
```

---

## EE-Only Sources

For EE-only data sources, see `ee/DATA_SOURCE_GUIDE_EE.md`. It covers EE engine types, catalog registration, source family specs, session metadata, and frontend icons.

---

## Verification & Testing

A task is **NOT** complete until all of these pass.

### 1. Type Generation

```bash
cd core && go generate .
cd frontend && pnpm run generate
# If EE:
cd ee && go generate .
cd ee/frontend && pnpm run generate
```

### 2. Build Checks

```bash
# Backend
cd core && go build ./cmd/whodb      # CE
cd ee && go build ./cmd/whodb        # EE

# Frontend
cd frontend && pnpm run build:ce     # CE
```

### 3. Automated Tests

Add tests in your plugin package (`plugin_test.go`). At minimum, test:
- Connection parsing
- Raw query execution
- Schema introspection

```bash
bash dev/run-backend-tests.sh all
```

### 4. Linting

Fix all linter errors before considering the job done. Do not suppress warnings.

---

## Registration Pipeline Summary

Understanding the full registration flow helps debug issues:

### Path A (Database Plugin):

```
1. Plugin init() → engine.RegisterPlugin(plugin)                    [plugin available in engine registry]
2. dbcatalog entry → ConnectableDatabase{ID, PluginType, Fields}    [connection form defined]
3. sourcecatalog FamilySpec → familySpecs["MyNewDB"]                [capabilities defined]
4. sourcecatalog metadata → RegisterSessionMetadata(...)            [query builder metadata]
5. core/src/sources/database/register.go init():
   - Iterates dbcatalog.All()
   - Calls sourcecatalog.BuildTypeSpec(entry) for each
   - Calls source.RegisterType(spec) for each                      [TypeSpec in global registry]
6. source/adapters/database_connector.go init():
   - Registers DatabaseConnector as driver "database"               [driver ready]
7. GraphQL resolver:
   - getSourceSessionForContext(ctx)
   - source.Open(spec, credentials) → DatabaseConnector.Open()
   - Returns DatabaseSession wrapping the engine.Plugin
```

### Path B (Source-First Connector):

```
1. Connector init():
   - source.RegisterDriver(driverID, connector)                    [driver registered]
   - source.RegisterType(TypeSpec{DriverID: driverID, ...})        [TypeSpec in global registry]
2. GraphQL resolver:
   - getSourceSessionForContext(ctx)
   - source.Open(spec, credentials) → YourConnector.Open()
   - Returns your SourceSession implementation
```

---

## Final Checklist

### For Path A (Database Plugin):

- [ ] Engine constant defined (`engine.go` or `ee/core/src/engine/types.go`)
- [ ] Plugin package created with `init()` calling `engine.RegisterPlugin()`
- [ ] All required `PluginFunctions` / `GormPluginFunctions` methods implemented
- [ ] Blank import added to entry point (`main.go`)
- [ ] `ConnectableDatabase` entry in dbcatalog (with SSL modes if applicable)
- [ ] `FamilySpec` in sourcecatalog
- [ ] Session metadata registered (operators, type definitions, alias map)
- [ ] Frontend icon added (keyed by exact source ID string)
- [ ] Localization strings added
- [ ] `go generate`, `pnpm run generate` — both pass
- [ ] `go build`, `pnpm run build:ce` — zero errors
- [ ] Tests added and passing

### For Path B (Source-First Connector):

- [ ] Connector package created with `init()` calling `source.RegisterDriver()` and `source.RegisterType()`
- [ ] `SourceSession` implemented + relevant capability interfaces
- [ ] Blank import added to entry point (`main.go`)
- [ ] `TypeSpec` has correct `DriverID`, `ConnectionFields`, `Contract`, `Traits`
- [ ] Frontend icon added
- [ ] Localization strings added
- [ ] `go generate`, `pnpm run generate` — both pass
- [ ] `go build`, `pnpm run build:ce` — zero errors
- [ ] Tests added and passing

### For EE sources (additional):

See `ee/DATA_SOURCE_GUIDE_EE.md` for the full EE checklist.
