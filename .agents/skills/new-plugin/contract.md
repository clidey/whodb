# Source Contract Reference

Rules that the code does not spell out in one place. Field and type definitions
live in `core/src/source/types.go`, `interfaces.go` and `contract.go`.

## The contract is the operation policy

- Declare each capability once in the family spec or `TypeSpec`: `Surfaces`, `RootActions`, `BrowsePath`, `ObjectTypes[].Actions` and `ObjectTypes[].Views`.
- The adapter (`core/src/source/adapters/database_connector.go`) and the resolvers reject operations the contract does not declare. Listing children needs `Browse`; row reads and exports need `ViewRows`; content needs `ViewContent`; column and constraint metadata needs `Inspect`; writes need the matching data action; imports need `ImportData`; mock data needs `GenerateMockData`; creating objects needs `CreateChild` on the root or the parent. Graphs need `SurfaceGraph` and `ViewGraph` on the graph scope object.
- Execution behavior goes in `TypeTraits.Query` (`SupportsScripts`, `SupportsStreaming`, `SupportsMultiStatement`).
- Metadata reliability goes in `TypeTraits.Metadata`. Hide system schemas, collections or keys with `HiddenObjectNames` and `HiddenObjectPrefixes`, never by filtering in shared code. Hidden rules must reference declared object kinds.
- The frontend derives capability flags from the contract (`frontend/src/hooks/useSourceContract.ts`). Never branch on a source name in the backend or the frontend.

## Alias or wrapper

- **Alias**: runtime behavior is identical and only metadata differs (label, port, TLS defaults). Add a catalog entry with a new `ID` and the existing `PluginType`, as Valkey does with Redis. The family spec is looked up by ID first, then by plugin type, so an alias inherits its plugin's contract unless it has its own.
- **Wrapper**: needed as soon as one runtime behavior differs (introspection SQL, schema handling, auth or DSN, mutation semantics, graph queries). Write a thin plugin that embeds the base plugin. Examples: CockroachDB, QuestDB and YugabyteDB in `core/src/plugins/postgres/`, TiDB in `core/src/plugins/mysql/`.
- Never add `if dbType == ...` to shared code to patch an alias.
- Connection defaults such as ports go in `dbcatalog` `Extra`, not in plugin code.
- Discovery prefills may only target declared connection fields or the reserved SSL keys. `source.ValidateConnectionContract` enforces this.

## Table size attributes

`GetTableNameAndAttributes` and custom `GetStorageUnits` implementations:
- Emit raw byte counts as numeric strings. The frontend formats them (`formatBytes`). No `pg_size_pretty`, rounding or unit suffixes in SQL or Go.
- Use only the keys `"Total Size"` (data plus indexes) and `"Data Size"`.
- Scan sizes into `sql.NullInt64` and add the attribute only when valid.
- Counts that are not bytes use a different key, such as `"Entries"`.

## Identifier safety

- SQL plugins call `ConfigureIdentifierQuoting` with the database's quote style and render every raw schema, table and column name through `core/src/sqlident`. Do not add connector-specific full-name builders. See `.agents/docs/sql-security.md`.
- A source that declares `ImportData` needs a live overwrite test: a uniquely named target plus an unrelated guard table, a target name containing the database's closing delimiter and an SQL suffix (or proof the database rejects it), the production `importer.Execute` path with `ModeOverwrite`, replacement rows verified, guard unchanged, cleanup. Then list the plugin type in `coveredPluginTypes` in `core/src/sourcecatalog/identifier_coverage_test.go`; the test fails for import-capable connectors without an entry.

## SSL

- Declare modes on the catalog entry. Map native names to the unified values with `source.SSLModeInfo.Aliases` (for example, Postgres `verify-full` maps to `verify-identity`). Aliases of an existing database reuse its modes (`sslModesFor(engine.DatabaseType_Redis)` for Valkey).
- Parse options with `ssl.ParseSSLConfig` and build TLS with `ssl.BuildTLSConfig` (`core/src/common/ssl`). Certificate file paths are honored only for admin-configured profiles; login-form input must use the content keys.
- Where possible, `GetSSLStatus` should report the live connection state (Postgres queries `pg_stat_ssl`) and cache it with `plugins.GetCachedSSLStatus`.
- SSL e2e services use the `ssl` compose profile in `dev/docker-compose.yml`; dev certificates live in `dev/certs/{ca,server,client}/<db>/`.
