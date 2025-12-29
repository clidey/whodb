# @clidey/whodb-cli

WhoDB CLI - a powerful database management tool with interactive TUI, programmatic commands, and MCP server for AI assistants.

## Installation

```bash
# Run directly (no install required)
npx @clidey/whodb-cli --help

# Or install globally
npm install -g @clidey/whodb-cli
```

## Features

- **Interactive TUI**: Full terminal UI for database management
- **Programmatic Commands**: Query, export, and manage databases from scripts
- **MCP Server**: Integrate with Claude Desktop, Claude Code, and other AI assistants

## Supported Databases

- PostgreSQL
- MySQL / MariaDB
- SQLite
- MongoDB
- Redis
- Elasticsearch
- ClickHouse

Enterprise Edition adds: MSSQL, Oracle, DynamoDB

## Usage

### Interactive Mode (TUI)

```bash
npx @clidey/whodb-cli
```

### Programmatic Commands

```bash
# Query a database
npx @clidey/whodb-cli query "SELECT * FROM users LIMIT 10" --connection mydb

# List schemas
npx @clidey/whodb-cli schemas --connection mydb --format json

# List tables
npx @clidey/whodb-cli tables --connection mydb --schema public

# Describe columns
npx @clidey/whodb-cli columns --connection mydb --table users

# Export data
npx @clidey/whodb-cli export --connection mydb --table users --format csv
```

### MCP Server Mode

Start as an MCP server for AI assistant integration:

```bash
npx @clidey/whodb-cli mcp serve
```

## Usage with Claude Desktop

Add to your `~/.config/claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "whodb": {
      "command": "npx",
      "args": ["-y", "@clidey/whodb-cli", "mcp", "serve"],
      "env": {
        "WHODB_DEFAULT_URI": "postgres://user:pass@localhost:5432/mydb"
      }
    }
  }
}
```

## Usage with Claude Code

```bash
claude --mcp-server "npx -y @clidey/whodb-cli mcp serve"
```

Or add to your `.mcp.json`:

```json
{
  "mcpServers": {
    "whodb": {
      "command": "npx",
      "args": ["-y", "@clidey/whodb-cli", "mcp", "serve"],
      "env": {}
    }
  }
}
```

## Environment Variables

Configure database connections via environment variables:

| Variable | Description |
|----------|-------------|
| `WHODB_DEFAULT_URI` | Default database connection string |
| `WHODB_{NAME}_URI` | Named connection (e.g., `WHODB_PROD_URI`) |

## Available MCP Tools

| Tool | Description |
|------|-------------|
| `whodb_connections` | List available database connections |
| `whodb_schemas` | List schemas in a database |
| `whodb_tables` | List tables in a schema |
| `whodb_columns` | Get column details for a table |
| `whodb_query` | Execute SQL queries |

## Platform Support

This package automatically installs the correct binary for your platform:

- macOS (Intel and Apple Silicon)
- Linux (x64, ARM64, ARMv7)
- Windows (x64, ARM64)

## Links

- [GitHub Repository](https://github.com/clidey/whodb)
- [Documentation](https://github.com/clidey/whodb#readme)

## License

Apache-2.0
