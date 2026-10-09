---
name: whodb
description: Work with WhoDB platform hosts, organizations, projects, and resources through MCP, or use database-only MCP tools when database mode is configured.
---

# WhoDB Database Assistant

You have access to WhoDB for database operations. Use these tools and commands to help users with database tasks.

## Platform MCP (default)

`whodb mcp serve` exposes hosted platform tools. Sign in to each required host with
`whodb login --host <url>`. Start with `whodb_platform_hosts` to discover saved
hosts and accounts, then `whodb_platform_orgs` with `workspace: {host}` and
`whodb_platform_projects` with `workspace: {host, org}`.

Resolve the intended target with `whodb_platform_workspace_resolve`. Pass an
explicit `workspace: {host, org, project}` on subsequent tools when working across
UAT, production, or projects. Targets apply to one call, including parallel calls,
and never change saved defaults. Check the returned `scope`. Do not run `whodb use`
to switch another session's defaults. Omitted targets use process overrides, then
saved defaults. Read `whodb://platform/schema` for operations and payloads.

Writes return a preview bound to its host, account, organization, and project.
Confirm only after approval of that exact preview. Changing defaults does not
redirect a pending confirmation. Ask for `whodb login --host <url>` when a host
needs authentication; no separate login is needed per project.

## Guided setup

This skill is optional. WhoDB MCP supplies its own instructions, tool schemas,
and setup guidance. Prefer live MCP discovery if this skill differs from it.
Keep one entry: command `whodb`, args `["mcp", "serve"]`.

Use `whodb_mcp_setup` with `action: "inspect"` to discover settings, targets,
choices, and the settings schema. Ask the user what they want, then use
`action: "preview"` with a `patch`. Show the proposed settings and obtain approval
before `action: "apply"` with the returned `confirmation_token` and `approved: true`.
Do not send a patch with apply. Tokens expire after five minutes; only the latest
preview is valid. Restart MCP after changed settings, then use `action: "verify"`.
Verification checks active settings. Omitted patch fields are preserved.

No shell access is required for this flow. Browser sign-in and credential entry
remain user actions; never request credentials in chat. If setup tools are
filtered out, ask the user to run `whodb setup`. CLI agents can alternatively use
`whodb setup inspect`, `validate`, `apply --yes`, and `verify`.
Inspect `whodb://mcp/configuration` for active modules and restrictions.
Platform calls target `workspace`; database calls target `connection`.

## Database-only MCP

Enable the database module through `whodb setup`; both modules can run in the same `whodb mcp serve` connection. `--database` remains an advanced process override.
This connects directly to databases without a WhoDB platform server; databases
can be remote. The terminal UI is available with `whodb --tui`.

### Database tools

When database-only MCP is configured, use these tools directly:

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
whodb query "SELECT * FROM users LIMIT 10" --connection mydb --format json
```

### Schema Discovery
```bash
# List schemas
whodb schemas --connection mydb --format json

# List tables
whodb tables --connection mydb --schema public --format json

# Describe columns
whodb columns --connection mydb --table users --format json
```

### Connection Management
```bash
# List connections
whodb connections list --format json

# Test connection
whodb connections test mydb

# Add new connection (interactive)
whodb connections add --name mydb --type Postgres --host localhost --database mydb
```

### Data Export
```bash
# Export to CSV
whodb export --connection mydb --table users --output users.csv

# Export query results
whodb export --connection mydb --query "SELECT * FROM orders" --output orders.xlsx
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
