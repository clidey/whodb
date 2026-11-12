---
name: new-plugin
description: Add a new data source to WhoDB CE, either a database plugin or a source-first connector
---

# Add a Data Source

Pick a path:
- **Database plugin**: a SQL or NoSQL database with a Go driver. Implement `engine.PluginFunctions`; the shared `database` driver adapts it to the source API.
- **Source-first connector**: anything else (object stores, APIs). Implement `source.SourceConnector` and `source.SourceSession` directly.

Never add per-source GraphQL; the generic `Source*` API serves every type. Read `contract.md` in this skill before declaring capabilities, SSL or aliases.

## Database plugin

1. **Type ID.** Add `DatabaseType_<Name> = "<Name>"` to the const block in `core/src/engine/engine.go`. The string is the source ID everywhere: catalog, icon key, `WHODB_<NAME>` env profiles.
2. **Plugin package** in `core/src/plugins/<name>/`; `core/src/plugins/postgres/` is the reference.
   - SQL: embed `gorm_plugin.GormPlugin`. The constructor calls `ConfigureIdentifierQuoting(...)`, sets `Type`, `PluginFunctions = plugin` and `GormPluginFunctions = plugin`; `init()` calls `engine.RegisterPlugin(...)`.
   - `GormPlugin` has no default for `DB`, `GetTableInfoQuery`, `GetTableNameAndAttributes`, `GetStorageUnitExistsQuery`, `GetPrimaryKeyColQuery`, `GetAllSchemasQuery`, `GetCreateTableQuery`, `GetSupportedOperators`, `GetGraphQueryDB`, `GetDatabases`, `RawExecute` or `GetSSLStatus`. Everything else has a default; override only what differs.
   - Non-SQL: embed `engine.BasePlugin`; unsupported user operations return `errors.ErrUnsupported`. See the mongodb, redis, elasticsearch and memcached plugins.
   - Do DB work inside `plugins.WithConnection(config, p.DB, ...)`; direct driver calls use `config.OperationContext()`.
   - Open network connections through `sourcedial`: when `sourcedial.Dialer()` is non-nil, pass it to the driver's dial hook (pgx `DialFunc`, go-redis `Dialer`, mongo `SetDialer` with `sourcedial.ContextDialer`); when nil, leave the driver's dialing alone. If the hook replaces the driver's TLS setup, use `sourcedial.DialTLS`. Add the connector to `core/src/plugins/sourcedial_test.go`.
   - Wire-compatible with an existing plugin? Add a catalog alias instead of a plugin (see `contract.md`).
3. **Register.** Blank-import the package in `core/cmd/whodb/main.go`. A CGO plugin that cannot build on arm/riscv64 registers through a build-tagged file in `core/src/` instead (see `src_duckdb.go`).
4. **Connection form.** Add a `ConnectableDatabase` to `catalog` in `core/src/dbcatalog/catalog.go`: `ID`, `Label`, `PluginType` (the plugin that runs it), `Extra` (advanced fields and defaults such as `"Port"`), `Fields`, `RequiredFields`, `SSLModes` (extend `sslModesFor` in `ssl_modes.go`).
5. **Contract.** Add a `FamilySpec` to `familySpecs` in `core/src/sourcecatalog/catalog.go`. Build object types with the helpers in that file (`metadataObjectType`, `tabularObjectType`, `tabularReadOnlyObjectType`, `documentObjectType`, `keyValueObjectType`, `keyValueExistingMutableObjectType`); never hand-build `source.ObjectType`. A catalog entry with no family spec for its ID or plugin type is skipped silently (`core/src/sources/database/register.go`).
6. **Session metadata.** Put type definitions, operators and the alias map in `core/src/sourcecatalog/specs/` (`sql.go` or `nonrelational.go`) and register them in `registerSessionMetadata()` in `core/src/sourcecatalog/metadata.go`. Plugins may import specs for type normalization but never register metadata from `init()`.
7. **Frontend.** Add an SVG to `ceLogos` in `frontend/src/components/icons.tsx`, keyed by the exact source ID (lookup falls back to the plugin type). Forms, SSL options and capability flags render from the backend contract. Add a `customFormRenderer` through `registerSourceTypeOverrides()` (`frontend/src/config/source-registry.ts`) only when the field-based form cannot work.

## Source-first connector

1. Package in `core/src/sources/<name>/`. Implement `source.SourceConnector` (`Open`) and a session that implements `source.SourceSession` plus only the capability interfaces it supports (`core/src/source/interfaces.go`). Resolvers type-assert those interfaces.
2. In `init()`, call `source.RegisterDriver(driverID, connector)` and `source.RegisterType(source.TypeSpec{DriverID: driverID, ...})` with `ConnectionFields`, `Contract`, `Traits` and `SSLModes`.
3. Blank-import the package in `core/cmd/whodb/main.go` and add the icon as in step 7.

## Tests

- Package tests: connection parsing, raw query execution, schema introspection.
- `core/src/sourcecatalog/catalog_test.go` validates the connection, session-metadata, object-creation and object-metadata contracts of every catalog type.
- Import-capable sources need the identifier proof in `contract.md`.
- E2E: add `frontend/e2e/fixtures/databases/<name>.json`, a service and seed data in `dev/docker-compose.yml` and `dev/sample-data/`, and the name in `DEFAULT_DATABASES` and `DEFAULT_CATEGORIES` in `dev/run-e2e.sh` and the `ALL` list in `.github/workflows/e2e-ce.yml`.

## Verify

```bash
cd core && go build ./cmd/whodb && go test ./src/sourcecatalog/... && ./lint.sh
cd frontend && pnpm run build:ce && pnpm e2e:db:headless <name>
```
