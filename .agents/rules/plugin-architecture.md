---
paths:
  - "core/src/engine/**"
  - "core/src/plugins/**"
  - "core/src/source/**"
  - "core/src/sources/**"
  - "core/src/sourcecatalog/**"
  - "core/src/dbcatalog/**"
---

# Plugin and Source Architecture Rules

The public API is source-first (`core/src/source`, `core/src/sourcecatalog`); execution still runs in plugins under `core/src/plugins/`. To add a data source, use the `new-plugin` skill.

## No database switches in shared code
- Never `switch dbType` or `if dbType ==` outside a plugin. Per-type declarations belong in `dbcatalog` and `sourcecatalog`; per-type behavior belongs in a plugin.
- New behavior: add a method to `PluginFunctions` (`core/src/engine/plugin.go`), give it a default in `GormPlugin` or `engine.BasePlugin`, and override it in the plugins that differ.
- If the behavior is public, also expose it through the adapter in `core/src/source/adapters/` and declare it in the contract in `core/src/sourcecatalog/catalog.go`. The adapter and resolvers reject operations the contract does not declare.
- Execution support (`TypeTraits.Query`) and metadata fidelity and hidden objects (`TypeTraits.Metadata`) are declared in the catalog, never inferred from the database type.
- Wire-compatible databases are catalog aliases; any runtime difference needs a thin wrapper plugin (see `.agents/skills/new-plugin/contract.md`).

## Session metadata
- Type definitions, operators and alias maps live in side-effect-free `core/src/sourcecatalog/specs/` and are registered centrally in `core/src/sourcecatalog/metadata.go`. Plugin `init()` only calls `engine.RegisterPlugin`; plugins import specs for normalization.
- A source with the query surface must resolve session metadata with operators, and every alias target must be a declared type; `source.ValidateSessionMetadataContract` enforces this in the catalog tests. When definitions exist, `engine.ValidateColumnType` rejects new column types that are not in them.

## Request context
- Request-scoped work uses the context on `*engine.PluginConfig`. GORM plugins inherit it through `plugins.WithConnection`. Direct-driver plugins call `config.OperationContext()` or `config.OperationContextWithTimeout(...)`, ideally through one local helper so new methods inherit it.
- `context.Background()` is only for cleanup that is not part of a request, such as cache eviction.

## Known quirks
- SQLite has no schemas; its `FormTableName` returns the bare table name.
- Postgres array column types start with `_` (`_text`).
- Redis lists databases by probing slots 0-15.
