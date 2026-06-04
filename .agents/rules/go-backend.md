---
paths:
  - "core/**/*.go"
  - "cli/**/*.go"
---

# Go Backend Rules

## Verification (run after changes)
```bash
cd core && ./lint.sh && go build ./cmd/whodb
```

## SQL Security
- Never `fmt.Sprintf` with user input — use `db.Raw("... WHERE x = ?", val)` or GORM builder
- For identifiers (table/column names), use `p.EscapeIdentifier(name)`
- Always close `*sql.Rows` with `defer rows.Close()`

## Plugin Patterns
- Use `plugins.WithConnection(config, p.DB, func(db *gorm.DB) (...) { ... })` for all DB operations
- Extend `GormPlugin` for SQL databases — override only what differs
- Use `config.OperationContext()` or `config.OperationContextWithTimeout(...)` for request-scoped work — never `context.Background()` for user requests
- Self-register in `init()` via `engine.RegisterPlugin(...)` — add blank import in entry point

## Code Style
- `any` not `interface{}`
- `env` package: pure declarations only (no `log` import). Parsing + error reporting → `envconfig`
- Delete build binaries after testing
- Never log passwords, API keys, tokens, or connection strings
