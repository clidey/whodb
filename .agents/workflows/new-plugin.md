---
name: new-plugin
description: Step-by-step guide for adding a new database plugin (CE or EE)
---

# Add a New Database Plugin

## Prerequisites
- Decide: CE (`core/src/plugins/`) or EE (`ee/core/src/plugins/`)
- Determine: SQL-based (extend GormPlugin) or custom driver

## Steps

### 1. Create Plugin Directory
```
core/src/plugins/<name>/
├── db.go          # Connection creation (DB method)
├── <name>.go      # Plugin struct, NewXxxPlugin(), database-specific queries
├── types.go       # Type definitions, alias map (imports from sourcecatalog/specs/)
└── constraints.go # Column constraint detection (optional override)
```

### 2. Implement Plugin Struct
```go
package <name>

import (
    "github.com/clidey/whodb/core/src/engine"
    "github.com/clidey/whodb/core/src/plugins/gorm"
)

type <Name>Plugin struct {
    gorm.GormPlugin
}

func New<Name>Plugin() *engine.Plugin {
    p := &<Name>Plugin{}
    return &engine.Plugin{
        Type:            engine.DatabaseType_<Name>,
        PluginFunctions: p,
    }
}

func init() {
    engine.RegisterPlugin(New<Name>Plugin())
}
```

### 3. Add Database Type
Add constant to `core/src/engine/types.go`:
```go
DatabaseType_<Name> engine.DatabaseType = "<Name>"
```

### 4. Add Blank Import
In `core/cmd/whodb/main.go` (CE) or `ee/cmd/whodb/main.go` (EE):
```go
_ "github.com/clidey/whodb/core/src/plugins/<name>"
```

### 5. Add Source Catalog Entry
Register in `core/src/dbcatalog/register.go` (or `ee/core/src/dbcatalog/register.go`):
- Connection fields, SSL modes, default port
- Source contract (surfaces, object types, actions)
- Type traits (transport, schema fidelity, query capabilities)

### 6. Add Session Metadata Spec
Create `core/src/sourcecatalog/specs/<name>.go`:
- Type definitions (canonical types for UI type selector)
- Alias map (INT → INTEGER, etc.)
- Operator definitions

Register in `core/src/sourcecatalog/metadata.go`.

### 7. Override Key GormPlugin Methods
At minimum for SQL plugins:
- `DB(config) (*gorm.DB, error)` — connection with driver config
- `GetAllSchemasQuery() string`
- `GetSchemaTableQuery() string`
- `GetPlaceholder(index int) string` — `$1` for Postgres-like, `?` for MySQL-like

### 8. Add Frontend Icon
In `frontend/src/icons.tsx` (or `ee/frontend/src/icons.tsx`):
```typescript
registerIcons('<name>', () => import('./icons/<Name>Icon'))
```

### 9. Verification
```bash
cd core && go build ./cmd/whodb && go vet ./...
cd frontend && pnpm run build:ce
```

### 10. Tests
- Add database fixture in `frontend/e2e/fixtures/databases/<name>.json`
- Add Docker service in `dev/docker-compose.yml` with seed data
- Run: `cd frontend && pnpm e2e:db:headless <name>`

## Reference
- See `DATA_SOURCE_GUIDE.md` for full details
- See `core/src/plugins/postgres/` as reference implementation
- See `.agents/docs/plugin-architecture.md` for architecture details
