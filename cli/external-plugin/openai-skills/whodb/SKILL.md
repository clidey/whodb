---
name: whodb
description: "Work with WhoDB platform hosts and project resources via MCP, including multiple environments, or query databases directly in database-only MCP mode."
license: Apache-2.0
metadata:
  author: clidey
  version: "0.106.0"
compatibility: "Requires Node.js (npx) or the whodb binary installed."
---

# WhoDB

Query databases, explore schemas, analyze data quality, and get optimization recommendations through the WhoDB MCP server.

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

## Database-only MCP setup

If the WhoDB MCP server is not connected, set it up:

1. Add the MCP server (npx, no install needed):
   ```bash
   codex mcp add whodb -- npx -y whodb mcp serve
   ```

2. Or if `whodb` is installed locally:
   ```bash
   codex mcp add whodb -- whodb mcp serve
   ```

3. Configure database connections via environment variables:
   ```bash
   export WHODB_POSTGRES='[{"alias":"prod","host":"localhost","user":"user","password":"pass","database":"mydb","port":"5432"}]'
   export WHODB_MYSQL_1='{"alias":"dev","host":"localhost","user":"user","password":"pass","database":"devdb","port":"3306"}'
   ```

After adding the MCP server, restart Codex.

## Available MCP Tools

### whodb_connections
List all available database connections.
```
No parameters required.
Returns: connection names with type and source (env/saved).
```

### whodb_schemas
List schemas in a database.
```
Parameters:
- connection: Connection name (optional if only one exists)
```

### whodb_tables
List tables in a schema.
```
Parameters:
- connection: Connection name (optional)
- schema: Schema name (optional, uses default)
```

### whodb_columns
Describe table columns and types.
```
Parameters:
- connection: Connection name (optional)
- table: Table name (required)
- schema: Schema name (optional)
```

### whodb_query
Execute SQL queries with security validation.
```
Parameters:
- connection: Connection name (optional)
- query: SQL query to execute
```

### whodb_explain
Run EXPLAIN for a SQL query.
```
Parameters:
- connection: Connection name (optional)
- query: SQL query to analyze
```

### whodb_erd
Load graph/relationship metadata for ER diagrams.
```
Parameters:
- connection: Connection name (optional)
- schema: Schema name (optional)
```

### whodb_diff
Compare schema metadata between two connections.
```
Parameters:
- source: Source connection name (required)
- target: Target connection name (required)
```

### whodb_audit
Run data quality audits on a table.
```
Parameters:
- connection: Connection name (optional)
- table: Table name (required)
- schema: Schema name (optional)
```

### whodb_suggestions
Load backend-generated query suggestions for a connection.
```
Parameters:
- connection: Connection name (optional)
- schema: Schema name (optional)
```

### whodb_confirm
Confirm a pending write operation (only available in confirm-writes mode, which is the default).
```
Parameters:
- token: Confirmation token from a pending write
```

### whodb_pending
List pending confirmation tokens.
```
No parameters required.
```

## Standard Workflow

### Step 1: Discover connections
```
whodb_connections → list available databases
```

### Step 2: Explore schema
```
whodb_schemas(connection="name") → list schemas
whodb_tables(connection="name", schema="public") → list tables
whodb_columns(connection="name", table="users") → column details
```

### Step 3: Query data
```
whodb_query(connection="name", query="SELECT * FROM users LIMIT 10")
```

### Step 4: Analyze (as needed)
```
whodb_explain(connection="name", query="SELECT ...") → query plan
whodb_erd(connection="name") → table relationships
whodb_audit(connection="name", table="users") → data quality
whodb_diff(source="prod", target="staging") → schema comparison
```

## Query Building

When converting natural language to SQL:

1. Always check schema first with `whodb_tables` and `whodb_columns`
2. Match entities to table names, attributes to column names
3. Identify foreign key joins from column metadata
4. Use LIMIT for exploratory queries (default: 100)

### Date handling varies by database

| Intent | PostgreSQL | MySQL | SQLite |
|--------|-----------|-------|--------|
| Last 7 days | `>= NOW() - INTERVAL '7 days'` | `>= DATE_SUB(NOW(), INTERVAL 7 DAY)` | `>= DATE('now', '-7 days')` |
| Start of month | `>= DATE_TRUNC('month', CURRENT_DATE)` | `>= DATE_FORMAT(NOW(), '%Y-%m-01')` | `>= DATE('now', 'start of month')` |

## Security Modes

The MCP server defaults to confirm-writes mode (writes require confirmation via `whodb_confirm`).

- `--read-only`: blocks all write operations
- `--safe-mode`: read-only + strict security validation
- `--allow-write`: full write access without confirmation

## Safety Rules

- Always explore schema before writing queries
- Use LIMIT for exploratory queries
- Prefer specific columns over SELECT *
- Never generate DELETE, UPDATE, or DROP unless explicitly requested
- Never expose or log credentials
- Use connection names, not connection strings
