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
- Values are parameters; identifiers go through `core/src/sqlident` after `GormPlugin.ConfigureIdentifierQuoting(...)`. Never concatenate an identifier or treat `StorageUnitExists` as the safety boundary. See `.agents/docs/sql-security.md`.

## Plugin Patterns
- Use `plugins.WithConnection(config, p.DB, func(db *gorm.DB) (...) { ... })` for all DB operations
- Extend `GormPlugin` for SQL databases — override only what differs
- Use `config.OperationContext()` or `config.OperationContextWithTimeout(...)` for request-scoped work — never `context.Background()` for user requests
- Self-register in `init()` via `engine.RegisterPlugin(...)` — add blank import in entry point

## Code Style
- `env` package: pure declarations only (no `log` import). Parsing + error reporting → `envconfig`
- Delete build binaries after testing
- Never log passwords, API keys, tokens, or connection strings
- The server binds all interfaces (`core/src/app/app.go`) and `core/Dockerfile` runs as root, on purpose: a loopback default and a non-root image were tried and dropped because they break existing access and bind-mount ownership. Change either only with a migration plan.

## Documentation
- Exported functions and types need a doc comment that starts with the name. The `revive` `exported` check is disabled, so review this by hand.
- Comments explain why: edge cases, database quirks, security or performance trade-offs. Don't restate the code or annotate each item in a data list.
