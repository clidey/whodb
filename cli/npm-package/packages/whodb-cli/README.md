# @clidey/whodb-cli

WhoDB CLI - a powerful database management tool with interactive TUI, programmatic commands, and MCP server for AI assistants.

Installing this package provides the `whodb` command. If you'd rather use a
namespaced package name, install [`@clidey/whodb`](https://www.npmjs.com/package/@clidey/whodb)
instead — both provide the same command.

## Installation

```bash
# Run directly (no install required)
npx @clidey/whodb-cli --help

# Or install globally
npm install -g @clidey/whodb-cli
```

Both install methods provide the `whodb` command.

## Features

- **Interactive TUI**: Full terminal UI for database management
- **Programmatic Commands**: Query, export, and manage databases from scripts
- **MCP Server**: Integrate with Claude Desktop, Claude Code, and other MCP clients
- **Assistant Integrations**: Install ready-to-use MCP configs for Codex, Claude Code, Cursor, VS Code, GitHub Copilot CLI, Gemini CLI, Windsurf, OpenCode, Cline, Zed, Continue, and aider

## Supported Databases

- PostgreSQL
- MySQL / MariaDB
- SQLite
- MongoDB
- Redis
- Elasticsearch
- ClickHouse

## Usage

### Interactive Mode (TUI)

```bash
npx @clidey/whodb
```

### Programmatic Commands

```bash
# Query a database
npx @clidey/whodb query "SELECT * FROM users LIMIT 10" --connection mydb

# List schemas
npx @clidey/whodb schemas --connection mydb --format json

# List tables
npx @clidey/whodb tables --connection mydb --schema public

# Describe columns
npx @clidey/whodb columns --connection mydb --table users

# Export data
npx @clidey/whodb export --connection mydb --table users --format csv --output users.csv
```

### MCP Server Mode

Start as an MCP server for AI assistant integration:

```bash
npx @clidey/whodb mcp serve
```

Write operations require confirmation by default. Use `--allow-write` to disable confirmations or `--read-only` to block writes.

Restrict AI access to specific connections with `--allowed-connections`:

```bash
npx @clidey/whodb mcp serve --allowed-connections prod,staging
```

## MCP Client Configuration (Example)

Example configuration (from `whodb mcp serve --help`):

```json
{
  "mcpServers": {
    "whodb": {
      "command": "whodb",
      "args": ["mcp", "serve"],
      "env": {
        "WHODB_POSTGRES_1": "{\"alias\":\"prod\",\"host\":\"localhost\",\"user\":\"user\",\"password\":\"pass\",\"database\":\"db\"}"
      }
    }
  }
}
```

## Assistant Integration Installer

Install native assistant configuration with the bundled CLI:

```bash
npx @clidey/whodb skills install --target cursor
npx @clidey/whodb skills install --target vscode
npx @clidey/whodb skills install --target gemini-cli
npx @clidey/whodb skills install --target claude-code --include-agents
npx @clidey/whodb skills install --target cursor --dry-run
```

Supported targets: `codex`, `claude-code`, `cursor`, `vscode`, `github-copilot`, `gemini-cli`, `windsurf`, `opencode`, `cline`, `zed`, `continue`, and `aider`.

Existing JSON and JSONC configuration files are merged in place and rewritten as formatted JSON.
Before an existing JSON or YAML assistant config is rewritten, the original file is saved beside it as `<filename>.whodb.bak`.
Use `--dry-run` to preview created or updated files and any backup paths without writing changes.

## Environment Variables

Configure database connections via environment profiles, for example `WHODB_POSTGRES='[{"alias":"prod","host":"localhost","user":"user","password":"pass","database":"db","port":"5432"}]'` or `WHODB_MYSQL_1='{"alias":"dev","host":"localhost","user":"user","password":"pass","database":"devdb","port":"3306"}'`. Each object supports `alias` (connection name), `host`, `user`, `password`, `database`, `port`, and optional `config`.

| Variable | Description |
|----------|-------------|
| `WHODB_<DBTYPE>` | JSON array of credential objects for a database type |
| `WHODB_<DBTYPE>_N` | JSON object for a single credential (numbered profiles) |

Use the `alias` field to set the connection name (e.g., `prod`). If omitted, the CLI assigns a name like `postgres-1`.

## Available MCP Tools

| Tool | Description |
|------|-------------|
| `whodb_connections` | List available database connections |
| `whodb_schemas` | List schemas in a database |
| `whodb_tables` | List tables in a schema |
| `whodb_columns` | Get column details for a table |
| `whodb_query` | Execute SQL queries |
| `whodb_confirm` | Confirm pending write operations (only when confirm-writes is enabled) |

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
