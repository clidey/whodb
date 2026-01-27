# WhoDB Plugin for Claude Code

Database management tools for Claude Code. Query databases, explore schemas, analyze data, and get optimization recommendations.

## Installation Methods

This plugin supports multiple installation methods. Choose the one that works best for you:

### Method 1: npm (Recommended - No pre-install needed)

The default configuration uses npx to auto-download and run the MCP server:

```json
{
  "whodb": {
    "command": "npx",
    "args": ["-y", "@clidey/whodb-cli", "mcp", "serve"]
  }
}
```

### Method 2: Docker (No pre-install needed)

If you prefer Docker, update your Claude settings (`.claude/settings.local.json`):

```json
{
  "mcpServers": {
    "whodb": {
      "command": "docker",
      "args": ["run", "-i", "--rm", "--network", "host", "clidey/whodb-cli", "mcp", "serve"]
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
      "command": "whodb-cli",
      "args": ["mcp", "serve"]
    }
  }
}
```

### Method 4: Homebrew (macOS/Linux)

```bash
brew install clidey/tap/whodb-cli
```

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

## Prerequisites

Install the WhoDB CLI before using this plugin:

### Option 1: Download Binary

Download the latest release from [GitHub Releases](https://github.com/clidey/whodb/releases) and add it to your PATH.

### Option 2: Build from Source

```bash
git clone https://github.com/clidey/whodb.git
cd whodb/cli
go build -o whodb-cli .
sudo mv whodb-cli /usr/local/bin/
```

### Option 3: Docker

If you prefer Docker, update your Claude Code settings to use:

```json
{
  "whodb": {
    "command": "docker",
    "args": ["run", "-i", "--rm", "--network", "host", "clidey/whodb-cli", "mcp", "serve"]
  }
}
```

## Configuration

### Database Connections

Configure database connections using environment variables:

```bash
# Environment profiles (examples below)
export WHODB_POSTGRES='[{"alias":"prod","host":"localhost","user":"user","password":"pass","database":"mydb","port":"5432"}]'
export WHODB_MYSQL_1='{"alias":"dev","host":"localhost","user":"user","password":"pass","database":"devdb","port":"3306"}'
```

Or use the CLI to save connections:

```bash
whodb-cli connect --type postgres --host localhost --port 5432 --user myuser --database mydb --name prod
```

## MCP Tools

This plugin provides the following tools:

| Tool | Description |
|------|-------------|
| `whodb_connections` | List available database connections |
| `whodb_schemas` | List schemas in a database |
| `whodb_tables` | List tables in a schema |
| `whodb_columns` | Describe table columns and types |
| `whodb_query` | Execute SQL queries |
| `whodb_confirm` | Confirm pending write operations (only when confirm-writes is enabled) |

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
