# SQL Security

- Values always go in as parameters: `db.Raw("... WHERE id = ?", v)` or GORM builders. Never `fmt.Sprintf` user input into SQL. A `forbidigo` rule in `core/.golangci.yml` flags `fmt.Sprintf` that builds SQL keywords; `gosec` also runs.
- Parameters cannot carry identifiers. Render every schema, table and column name through `core/src/sqlident`, quoting each raw component separately. Checking existence with `StorageUnitExists` or other metadata is not a substitute for quoting where the SQL is built.
- GORM plugins set their quote style in the constructor: `sqlident.DoubleQuote` (Postgres family, SQLite, DuckDB), `sqlident.Backtick` (MySQL family), `sqlident.Bracket` for bracket-quoting databases. Use a strict style (`BacktickStrict` for ClickHouse) only when the database rejects embedded delimiters; strict styles return an error instead of emitting SQL.
- Do not pass a pre-joined `schema.table` to `sqlident.New`, and do not add connector-specific full-name builders.
- Import-capable connectors need the identifier proof described in `.agents/skills/new-plugin/contract.md`.
- Server mode (not local/CLI) blocks file access through drivers. DuckDB appends `restrictedDSNOptions` and verifies the settings after connecting (`core/src/plugins/duckdb/db.go`), and rejects `?` or `#` in the path in every mode, since either would drop those options. MySQL parses the effective DSN and rejects `allowAllFiles` (`core/src/plugins/mysql/db.go`). Keep both checks when changing DSN building.

```go
name, err := sqlident.New(schemaName, tableName)
if err != nil {
    return false, err
}
qualified, err := name.SQL(p.IdentifierQuoteStyle())
if err != nil {
    return false, err
}
rows, err := db.Raw("SELECT * FROM "+qualified+" WHERE status = ?", status).Rows()
if err != nil {
    return false, err
}
defer rows.Close()
```
