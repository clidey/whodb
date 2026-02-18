---
name: whodb
description: Database operations including querying, schema exploration, and data analysis. Activates for tasks involving PostgreSQL, MySQL, MariaDB, SQLite, MongoDB, Redis, Elasticsearch, or ClickHouse databases.
---

# WhoDB Database Assistant

You have access to WhoDB for database operations. Use these tools and commands to help users with database tasks.

## MCP Tools (Preferred)

When the WhoDB MCP server is available, use these tools directly:

### whodb_connections
List all available database connections.
```
No parameters required.
Returns: List of connection names with type and source (env/saved).
```

### whodb_query
Execute SQL queries against a database.
```
Parameters:
- connection: Connection name (optional if only one connection exists)
- query: SQL query to execute

Example: whodb_query(connection="mydb", query="SELECT * FROM users LIMIT 10")
```

### whodb_schemas
List all schemas in a database.
```
Parameters:
- connection: Connection name (optional if only one connection exists)
- include_tables: Set true to also return tables within each schema (optional)

Example: whodb_schemas(connection="mydb")
Example: whodb_schemas(connection="mydb", include_tables=true)
```

### whodb_tables
List all tables in a schema.
```
Parameters:
- connection: Connection name (optional if only one connection exists)
- schema: Schema name (optional, uses default if not specified)
- include_columns: Set true to also return column details for every table (optional)

Example: whodb_tables(connection="mydb", schema="public")
Example: whodb_tables(connection="mydb", schema="public", include_columns=true)
```

### whodb_columns
Describe columns in a table.
```
Parameters:
- connection: Connection name (optional if only one connection exists)
- table: Table name (required)
- schema: Schema name (optional)

Example: whodb_columns(connection="mydb", table="users")
```

## CLI Commands (Fallback)

If MCP tools are unavailable, use the CLI directly via Bash:

### Query Execution
```bash
whodb-cli query "SELECT * FROM users LIMIT 10" --connection mydb --format json
```

### Schema Discovery
```bash
# List schemas
whodb-cli schemas --connection mydb --format json

# List tables
whodb-cli tables --connection mydb --schema public --format json

# Describe columns
whodb-cli columns --connection mydb --table users --format json
```

### Connection Management
```bash
# List connections
whodb-cli connections list --format json

# Test connection
whodb-cli connections test mydb

# Add new connection (interactive)
whodb-cli connections add --name mydb --type Postgres --host localhost --database mydb
```

### Data Export
```bash
# Export to CSV
whodb-cli export --connection mydb --table users --output users.csv

# Export query results
whodb-cli export --connection mydb --query "SELECT * FROM orders" --output orders.xlsx
```

## Workflow Examples

### Explore a New Database (Efficient)
1. List connections: `whodb_connections`
2. Get tables with columns in one call: `whodb_tables(connection="name", include_columns=true)`
3. Sample data: `whodb_query(connection="name", query="SELECT * FROM users LIMIT 5")`

Query results include `column_types` alongside column names, so you know the data types without a separate call.

### Explore a New Database (Multi-Schema)
1. List connections: `whodb_connections`
2. Get all schemas with tables: `whodb_schemas(connection="name", include_tables=true)`
3. Get columns for the schema you care about: `whodb_tables(connection="name", schema="public", include_columns=true)`
4. Query: `whodb_query(connection="name", query="SELECT * FROM users LIMIT 5")`

### Answer Data Questions
1. Understand the schema first - check table structure
2. Write targeted queries with appropriate filters
3. Always use LIMIT for exploratory queries
4. Present results in a clear, readable format

## Best Practices

- **Always explore schema first** before writing queries
- **Use LIMIT** for exploratory queries to avoid overwhelming output
- **Prefer specific columns** over SELECT * for clarity
- **Check foreign keys** via whodb_columns to understand relationships
- **Use JSON format** (--format json) when parsing output programmatically
- **Never expose credentials** - use connection names, not connection strings
