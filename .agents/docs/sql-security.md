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

## Identifier Escaping

For identifiers (table/column names) that can't use placeholders, use the plugin's `EscapeIdentifier()` method:

```go
// For SQLite PRAGMA (which doesn't support placeholders):
escapedTable := p.EscapeIdentifier(tableName)
query := fmt.Sprintf("PRAGMA table_info(%s)", escapedTable)
```

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

- Use `db.Raw()` with placeholders for complex queries
- Use GORM's query builder for simple operations
- Always close rows when done:

```go
rows, err := db.Raw("SELECT * FROM table WHERE col = ?", value).Rows()
defer rows.Close()
```
