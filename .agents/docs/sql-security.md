# SQL Security Guidelines

**CRITICAL**: Preventing SQL injection is non-negotiable. Never use string formatting for SQL queries with user input.

## Parameterized Queries

```go
// WRONG - SQL Injection vulnerability:
query := fmt.Sprintf("SELECT * FROM %s WHERE id = '%s'", table, userInput)
db.Raw(query).Scan(&result)

// CORRECT - Use prepared statements:
db.Raw("SELECT * FROM users WHERE id = ?", userInput).Scan(&result)

// CORRECT - Use GORM builder:
db.Table("users").Where("id = ?", userInput).Find(&result)
```

## Identifier Quoting

SQL parameters protect values only. They cannot represent database object names.
Every schema, table, and column name that reaches SQL must be rendered as an
identifier with `core/src/sqlident`; validation through metadata APIs such as
`StorageUnitExists` is not a substitute for quoting at the SQL sink.

GORM-backed plugins must configure their database's quote style in the plugin
constructor:

```go
p.ConfigureIdentifierQuoting(sqlident.DoubleQuote) // PostgreSQL, SQLite, Snowflake
p.ConfigureIdentifierQuoting(sqlident.Backtick)    // MySQL-family databases
p.ConfigureIdentifierQuoting(sqlident.Bracket)     // SQL Server
```

Use `sqlident.New` when database-specific code must construct a raw qualified
name:

```go
name, err := sqlident.New(schemaName, tableName)
if err != nil {
    return false, err
}
qualifiedTable, err := name.SQL(p.IdentifierQuoteStyle())
if err != nil {
    return false, err
}
if err := db.Exec("TRUNCATE TABLE " + qualifiedTable).Error; err != nil {
    return false, err
}
```

`sqlident.New` receives raw components and quotes each component separately. Do
not pass a pre-joined `schema.table` string, add connector-specific full-name
builders, or concatenate raw identifiers. Use a strict quote style only when the
database rejects embedded delimiter characters; strict styles must return the
validation error instead of emitting SQL.

JDBC bridge dialects declare `IdentifierQuote` with `bridge.QuoteDouble`,
`bridge.QuoteBacktick`, or `bridge.QuoteBracket`. The shared bridge plugin then
uses `sqlident` for qualified names and applies the same delimiter rules to
columns.

## WithConnection Pattern

Always use `plugins.WithConnection()` for database operations:

```go
_, err := plugins.WithConnection(config, p.DB, func(db *gorm.DB) (bool, error) {
    rows, err := db.Raw("SELECT * FROM users WHERE status = ?", "active").Rows()
    // ... process rows
    return true, nil
})
```

## GORM Query Patterns

- Use `db.Raw()` with placeholders for values in complex queries
- Use GORM's query builder for simple operations
- Always close rows when done:

```go
rows, err := db.Raw("SELECT * FROM table WHERE col = ?", value).Rows()
defer rows.Close()
```
