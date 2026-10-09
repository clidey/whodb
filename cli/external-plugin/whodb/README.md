# WhoDB Plugin for Claude Code

WhoDB platform and database-only MCP tools for Claude Code. Work across platform hosts and projects, or query databases directly.

## Installation Methods

This plugin supports multiple installation methods. Choose the one that works best for you:

### Method 1: npm (Recommended - No pre-install needed)

Run `whodb setup` to enable platform tools, database-only tools, or both. Keep one MCP entry at `whodb mcp serve`; agents can guide setup with `whodb setup inspect`, `validate`, `apply`, and `verify`. Sign in to platform hosts through `whodb login --host <url>` and target workspaces per call. Docker examples below use an explicit database-only override; saved setup applies to the configuration directory of the running CLI.

The default configuration uses npx to auto-download and run the MCP server:

```json
{
  "whodb": {
    "command": "npx",
    "args": ["-y", "whodb", "mcp", "serve"]
  }
}
```

### Method 2: Docker (Database-only MCP)

For database-only MCP access through Docker, update your Claude settings (`.claude/settings.local.json`):

```json
{
  "mcpServers": {
    "whodb": {
      "command": "docker",
      "args": ["run", "-i", "--rm", "--network", "host", "clidey/whodb-cli", "mcp", "serve", "--database"]
    }
  }
}
```

### Method 3: Local Binary (Best performance)

Install the CLI once, then it runs instantly:

1. Download from [GitHub Releases](https://github.com/clidey/whodb/releases)
2. Add to your PATH
3. Update Claude settings:

```json
{
  "mcpServers": {
    "whodb": {
      "command": "whodb",
      "args": ["mcp", "serve"]
    }
  }
}
```

### Method 4: Homebrew (macOS/Linux)

```bash
brew install whodb-cli
```

The formula is named `whodb-cli` (the `whodb` name is used by the desktop app cask), but it installs the `whodb` command.

### Method 5: Go Install (Requires Go)

```bash
go install github.com/clidey/whodb/cli@latest
```

## Supported Databases

- PostgreSQL
- MySQL / MariaDB
- SQLite
- MongoDB
- Redis
- ClickHouse
- Elasticsearch
- And more via WhoDB plugins

## Manual CLI installation

For the local binary method, install the CLI using one of these options:

### Option 1: Download Binary

Download the latest release from [GitHub Releases](https://github.com/clidey/whodb/releases) and add it to your PATH.

### Option 2: Build from Source

```bash
git clone https://github.com/clidey/whodb.git
cd whodb/cli
go build -o whodb .
sudo mv whodb /usr/local/bin/
```

### Option 3: Docker

If you prefer Docker, update your Claude Code settings to use:

```json
{
  "whodb": {
    "command": "docker",
    "args": ["run", "-i", "--rm", "--network", "host", "clidey/whodb-cli", "mcp", "serve", "--database"]
  }
}
```

## Configuration

### Database-only MCP connections

These connection settings apply to `whodb mcp serve --database`. Platform mode
uses the sources configured in the selected platform workspace.

Configure database connections using environment variables:

```bash
# Environment profiles (examples below)
export WHODB_POSTGRES='[{"alias":"prod","host":"localhost","user":"user","password":"pass","database":"mydb","port":"5432"}]'
export WHODB_MYSQL_1='{"alias":"dev","host":"localhost","user":"user","password":"pass","database":"devdb","port":"3306"}'
```

Or use the CLI to save connections:

```bash
whodb connect --type postgres --host localhost --port 5432 --user myuser --database mydb --name prod
```

## MCP Tools

This plugin provides the following tools:

| Tool | Description |
|------|-------------|
| `whodb_connections` | List available database connections |
| `whodb_schemas` | List database schemas/namespaces |
| `whodb_tables` | List tables in a schema (supports `include_columns` for inline column details) |
| `whodb_columns` | Describe table columns and types |
| `whodb_query` | Execute SQL queries (security-validated, supports parameterized queries) |
| `whodb_explain` | Run EXPLAIN for a SQL query |
| `whodb_diff` | Compare schema metadata between two connections |
| `whodb_erd` | Load graph/relationship metadata for a schema |
| `whodb_audit` | Run data quality checks on one schema or table |
| `whodb_suggestions` | Load backend-generated starter queries |
| `whodb_confirm` | Confirm pending write operations (only with confirm-writes mode) |
| `whodb_pending` | List pending write confirmations awaiting approval |

## Security Modes

By default, write operations require confirmation before executing (confirm-writes mode).

| Flag | Description |
|------|-------------|
| *(default)* | Confirm-writes: all writes require user confirmation |
| `--safe-mode` | Read-only + strict security (for demos and playgrounds) |
| `--read-only` | Read-only: SELECT, SHOW, DESCRIBE, EXPLAIN only |
| `--allow-write` | Full write access without confirmation (use with caution) |
| `--security=strict` | Blocks dangerous functions (pg_read_file, COPY, LOAD_FILE, etc.) |
| `--security=standard` | Basic validation (default) |
| `--security=minimal` | Only blocks DELETE without WHERE (when writes allowed) |

Additional options:

| Flag | Description |
|------|-------------|
| `--timeout=30s` | Query timeout duration |
| `--max-rows=100` | Limit rows returned per query (0 = unlimited) |
| `--allow-drop` | Allow DROP/TRUNCATE even with `--allow-write` |
| `--allow-multi-statement` | Allow multiple SQL statements in one query |
| `--tools=schemas,tables` | Comma-separated list of tools to enable (default: all) |
| `--disable-tools=query` | Comma-separated list of tools to disable |
| `--default-connection=prod` | Set default connection (no access restriction) |
| `--allowed-connections=prod,staging` | Restrict access to listed connections only |
| `--transport=http` | Run as HTTP service instead of stdio |
| `--host=localhost` | Bind address (HTTP mode only) |
| `--port=3000` | Listen port (HTTP mode only) |
| `--no-analytics` | Disable anonymous usage analytics |

## Skills

### whodb
Main database skill - activates for any database-related task.

### query-builder
Natural language to SQL conversion. Activates when you ask questions like:
- "Show me users who signed up last week"
- "Find orders over $100"
- "Get the top 10 customers"

### schema-designer
Database schema design assistance. Activates when you ask to:
- Create tables
- Design schemas
- Model data relationships

## Agents

### database-analyst
Deep database analysis expert. Use for:
- Schema analysis and documentation
- Data quality assessment
- Multi-step data exploration

**Invoke:** "Use the database-analyst agent to analyze my schema"

### query-optimizer
Query performance specialist. Use for:
- Slow query analysis
- Index recommendations
- Query rewriting

**Invoke:** "Use the query-optimizer agent to optimize this query"

### report-generator
Data reporting specialist. Use for:
- Formatted reports from queries
- Data summaries and trends
- Export preparation

**Invoke:** "Use the report-generator agent to create a sales report"

## Examples

```
# Explore a database
Show me the tables in my prod database

# Query data
Find all users created in the last 7 days

# Analyze performance
Use the query-optimizer agent to analyze: SELECT * FROM orders WHERE status = 'pending'

# Design schema
Help me design a schema for a blog with posts, comments, and tags
```

## Links

- [WhoDB Repository](https://github.com/clidey/whodb)
- [WhoDB Documentation](https://github.com/clidey/whodb#readme)
- [CLI Documentation](https://github.com/clidey/whodb/tree/main/cli#readme)

## License

Apache License 2.0

### MCP without skills

Skills are optional supplements. MCP initialization instructions, tool schemas,
and `whodb://mcp/configuration` provide essential guidance directly. Use
`whodb_mcp_setup` actions `inspect`, `preview` (with a settings patch), `apply`
(with the exact approved preview token and `approved: true`), and `verify`.
No shell access is required for these actions. Browser login and credential entry
remain user actions. Restart after changing settings; verification checks the
active process. The setup tool follows tool filters and can be disabled when
configuration should remain outside the agent.
