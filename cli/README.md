# WhoDB CLI

An interactive, production-ready command-line interface for WhoDB with a Claude Code-like experience.

## Features

- **Interactive TUI** - Beautiful terminal-based user interface built with Bubble Tea
- **Multi-Database Support** - PostgreSQL, MySQL, SQLite, MongoDB, Redis, ClickHouse, and more
- **Table Browser** - Navigate schemas and tables with visual grid layout
- **Visual Query Builder** - Build WHERE conditions interactively
- **SQL Editor** - Syntax highlighting with autocomplete
- **AI Assistant** - Natural language to SQL conversion (requires AI backend)
- **Responsive Data Viewer** - Paginated results with horizontal scrolling
- **Export Capabilities** - Export to CSV and Excel formats
- **Query History** - Persistent history with re-execution
- **Shell Completion** - Bash, Zsh, and Fish autocompletion
- **Programmatic Mode** - JSON/CSV output for scripting and automation
- **MCP Server** - Model Context Protocol server for AI assistants (Claude, Cursor, etc.)

## Installation

### From Source

```bash
cd cli
go build -o whodb-cli .
```

### Using Docker

```bash
# Build the Docker image (from repo root)
docker build -t whodb-cli:latest -f cli/Dockerfile .
```

### Verify Installation

```bash
whodb-cli --help
```

## Quick Start

### 1. Connect to a Database

#### PostgreSQL

```bash
whodb-cli connect \
  --type postgres \
  --host localhost \
  --port 5432 \
  --user postgres \
  --password mypassword \
  --database mydb \
  --name my-postgres
```

#### MySQL

```bash
whodb-cli connect \
  --type mysql \
  --host localhost \
  --port 3306 \
  --user root \
  --password mypassword \
  --database mydb \
  --name my-mysql
```

#### SQLite

```bash
whodb-cli connect \
  --type sqlite \
  --database /path/to/database.db \
  --name my-sqlite
```

#### MongoDB

```bash
whodb-cli connect \
  --type mongodb \
  --host localhost \
  --port 27017 \
  --user admin \
  --password mypassword \
  --database mydb \
  --name my-mongo
```

### 2. Start Interactive Mode

```bash
# Start the TUI (default behavior)
whodb-cli
```

### 3. Execute a Quick Query

```bash
whodb-cli query "SELECT * FROM users LIMIT 10" --connection my-postgres
```

## Commands

### Root Command (Interactive Mode)

Running `whodb-cli` without arguments starts the interactive TUI.

```bash
whodb-cli [flags]
```

Flags:

- `--config`: Config file path (default: `~/.whodb-cli/config.yaml`)
- `--debug`: Enable debug mode

### connect

Connect to a database and optionally save the connection.

```bash
whodb-cli connect [flags]
```

Flags:

- `--type`: Database type (required) - postgres, mysql, sqlite, mongodb, redis, clickhouse
- `--host`: Database host (default: localhost)
- `--port`: Database port (default varies by type)
- `--user`: Username
- `--password`: Password
- `--database`: Database name
- `--name`: Connection name (saves for later use)

### query

Execute a SQL query directly.

```bash
whodb-cli query "SQL" [flags]
```

Flags:

- `--connection`: Connection name to use

### completion

Generate or install shell completion scripts.

```bash
# Show help
whodb-cli completion

# Print completion script to stdout
whodb-cli completion bash
whodb-cli completion zsh
whodb-cli completion fish

# Install completion (auto-detects shell)
whodb-cli completion install

# Install for specific shell
whodb-cli completion install bash

# Uninstall completion
whodb-cli completion uninstall
```

Install paths (rc files updated automatically):

- Bash: `~/.local/share/bash-completion/completions/whodb-cli`
- Zsh: `~/.zsh/completions/_whodb-cli`
- Fish: `~/.config/fish/completions/whodb-cli.fish`

## Programmatic Commands

These commands output structured data for scripting, automation, and AI integration.

### schemas

List database schemas.

```bash
whodb-cli schemas --connection my-postgres --format json
```

Flags:
- `--connection, -c`: Connection name (required if multiple connections exist)
- `--format, -f`: Output format: `json`, `table`, `plain` (default: table)

### tables

List tables in a schema.

```bash
whodb-cli tables --connection my-postgres --schema public --format json
```

Flags:
- `--connection, -c`: Connection name
- `--schema, -s`: Schema name (default varies by database)
- `--format, -f`: Output format: `json`, `table`, `plain`

### columns

Describe table columns.

```bash
whodb-cli columns --connection my-postgres --table users --format json
```

Flags:
- `--connection, -c`: Connection name
- `--table, -t`: Table name (required)
- `--schema, -s`: Schema name
- `--format, -f`: Output format: `json`, `table`, `plain`

### connections

Manage saved connections.

```bash
# List connections
whodb-cli connections list --format json

# Test a connection
whodb-cli connections test my-postgres

# Add a connection
whodb-cli connections add --name prod --type postgres --host db.example.com --port 5432 --user app --database mydb

# Remove a connection
whodb-cli connections remove prod
```

### export

Export table data to file.

```bash
# Export to CSV
whodb-cli export --connection my-postgres --table users --format csv --output users.csv

# Export to Excel
whodb-cli export --connection my-postgres --table orders --format excel --output orders.xlsx
```

Flags:
- `--connection, -c`: Connection name
- `--table, -t`: Table name (required)
- `--schema, -s`: Schema name
- `--format, -f`: Export format: `csv`, `excel`
- `--output, -o`: Output file path

### history

Access query history.

```bash
# List recent queries
whodb-cli history list --limit 20 --format json

# Search history
whodb-cli history search "SELECT.*users"

# Clear history
whodb-cli history clear
```

Flags:
- `--limit, -l`: Number of entries to show (default: 10)
- `--format, -f`: Output format: `json`, `table`, `plain`

## MCP Server

WhoDB can run as an MCP (Model Context Protocol) server, enabling AI assistants like Claude, Cursor, and others to query your databases.

### Start the MCP Server

```bash
whodb-cli mcp serve
```

This starts a stdio-based MCP server that exposes these tools:

| Tool | Description |
|------|-------------|
| `whodb_connections` | List available database connections |
| `whodb_schemas` | List schemas in a database |
| `whodb_tables` | List tables in a schema |
| `whodb_columns` | Describe table columns |
| `whodb_query` | Execute SQL queries |

### Configure Connections

The MCP server uses a hybrid credential system:

**Option 1: Environment Variables** (recommended for production)

```bash
# Format: WHODB_{NAME}_URI
export WHODB_PROD_URI="postgres://user:pass@host:5432/dbname"
export WHODB_STAGING_URI="mysql://user:pass@host:3306/dbname"
```

**Option 2: Saved Connections**

Use `whodb-cli connect --name mydb ...` to save connections that the MCP server can access.

### Claude Desktop Configuration

Add to `~/.config/claude/claude_desktop_config.json` (Linux) or `~/Library/Application Support/Claude/claude_desktop_config.json` (macOS):

```json
{
  "mcpServers": {
    "whodb": {
      "command": "/path/to/whodb-cli",
      "args": ["mcp", "serve"],
      "env": {
        "WHODB_PROD_URI": "postgres://user:pass@host:5432/db"
      }
    }
  }
}
```

### Claude Code Configuration

Create `.mcp.json` in your project root:

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

### Docker MCP Server

```bash
docker run -i --rm \
  -e WHODB_PROD_URI="postgres://user:pass@host:5432/db" \
  --network host \
  whodb-cli:latest mcp serve
```

## Interactive Mode Views

### 1. Connection View

Select and manage database connections.

| Key     | Action                       |
|---------|------------------------------|
| `↑/k`   | Move up                      |
| `↓/j`   | Move down                    |
| `Enter` | Connect to selected database |
| `n`     | New connection               |
| `d`     | Delete connection            |
| `Esc`   | Back / Cancel                |

### 2. Browser View

Navigate schemas and tables in a visual grid layout.

| Key                     | Action             |
|-------------------------|--------------------|
| `↑/k` `↓/j` `←/h` `→/l` | Navigate grid      |
| `/` or `f`              | Filter tables      |
| `s`                     | Switch schema      |
| `Enter`                 | View table data    |
| `e`                     | Open SQL editor    |
| `Ctrl+H`                | View query history |
| `a`                     | Open AI assistant  |
| `r`                     | Refresh table list |
| `Esc`                   | Disconnect         |

### 3. Editor View

Write and execute SQL queries with syntax highlighting and autocomplete.

| Key                              | Action               |
|----------------------------------|----------------------|
| `Alt+Enter` (`Option+Enter` Mac) | Execute query        |
| `Ctrl+Space`                | Trigger autocomplete |
| `Tab` / `Shift+Tab`         | Navigate suggestions |
| `Ctrl+L`                    | Clear editor         |
| `Esc`                       | Back to browser      |

Features:

- SQL syntax highlighting
- Schema-aware autocomplete (tables, columns, keywords)
- Multi-line editing
- Error display

### 4. Results View

View query results in a responsive, paginated table.

| Key         | Action              |
|-------------|---------------------|
| `↑/k` `↓/j` | Navigate rows       |
| `←/h` `→/l` | Scroll columns      |
| `n`         | Next page           |
| `p`         | Previous page       |
| `e`         | Export data         |
| `w`         | Add WHERE condition |
| `Esc`       | Back                |

Features:

- Pagination (configurable, default 50 rows)
- Column resizing
- Data export (CSV, Excel)

### 5. History View

Browse and re-execute past queries.

| Key         | Action                 |
|-------------|------------------------|
| `↑/k` `↓/j` | Navigate history       |
| `/`         | Filter history         |
| `Enter`     | Load query into editor |
| `r`         | Re-run query           |
| `c`         | Clear history          |
| `Esc`       | Back                   |

### 6. AI Chat View

Natural language database queries (requires AI backend configuration).

| Key      | Action          |
|----------|-----------------|
| `Enter`  | Send message    |
| `Ctrl+M` | Change AI model |
| `Esc`    | Back to browser |

### 7. Export View

Export data to CSV or Excel format.

| Key     | Action                    |
|---------|---------------------------|
| `Tab`   | Switch format (CSV/Excel) |
| `Enter` | Confirm export            |
| `Esc`   | Cancel                    |

## Configuration

### Config File Location

```
~/.whodb-cli/config.yaml
```

### Config Structure

```yaml
connections:
  - name: local-postgres
    type: postgres
    host: localhost
    port: 5432
    username: postgres
    database: mydb

  - name: prod-mysql
    type: mysql
    host: prod-server
    port: 3306
    username: app_user
    database: production

history:
  max_entries: 1000
  persist: true

display:
  theme: dark
  page_size: 50
```

### Environment Variables

Override configuration with environment variables:

```bash
export WHODB_CLI_DEBUG=true
export WHODB_CLI_CONFIG=/custom/path/config.yaml
```

## Docker Usage

### Run Interactively

```bash
docker run -it --rm whodb-cli:latest
```

### Connect to Host Database

```bash
docker run -it --rm --network host whodb-cli:latest connect \
  --type postgres \
  --host localhost \
  --user postgres \
  --database mydb
```

### Execute Query

```bash
docker run -it --rm --network host whodb-cli:latest query "SELECT version()"
```

### Persist Configuration

```bash
docker run -it --rm \
  -v ~/.whodb-cli:/root/.whodb-cli \
  --network host \
  whodb-cli:latest
```

### Environment Variables

- `TERM=xterm-256color` - Proper terminal colors (set by default)
- `WHODB_CLI_*` - Any config can be set via environment variables

## Keyboard Reference Card

### Global

| Key      | Action    |
|----------|-----------|
| `Ctrl+C` | Quit      |
| `Esc`    | Go back   |
| `?`      | Show help |

### Navigation (Vim-style)

| Key        | Action        |
|------------|---------------|
| `↑` or `k` | Up            |
| `↓` or `j` | Down          |
| `←` or `h` | Left          |
| `→` or `l` | Right         |
| `/`        | Filter/Search |

### Common Actions

| Key     | Action         |
|---------|----------------|
| `Enter` | Select/Execute |
| `r`     | Refresh/Re-run |
| `e`     | Edit/Export    |
| `n`     | New/Next       |
| `p`     | Previous       |
| `d`     | Delete         |

## Troubleshooting

### Connection Issues

**"Cannot connect to database"**

```bash
# Verify database is running
pg_isready -h localhost -p 5432  # PostgreSQL
mysql -h localhost -u root -p     # MySQL
```

**"Plugin not found"**

Supported database types: `postgres`, `mysql`, `sqlite`, `mongodb`, `redis`, `clickhouse`, `elasticsearch`, `mariadb`

### Display Issues

**"Garbled text / incorrect colors"**

```bash
# Set terminal type
export TERM=xterm-256color
```

Recommended terminals: iTerm2, Alacritty, Windows Terminal, Kitty

### Configuration Issues

**"Config not found"**

```bash
mkdir -p ~/.whodb-cli
whodb-cli connect --type postgres --host localhost --name test
```

**"Permissions error"**

```bash
chmod 700 ~/.whodb-cli
chmod 600 ~/.whodb-cli/config.yaml
```

### Debug Mode

```bash
whodb-cli --debug
```

## Architecture

```
cli/
├── cmd/                # CLI commands (Cobra)
│   ├── root.go         # Main entry, starts TUI
│   ├── connect.go      # Database connection
│   ├── query.go        # Direct query execution
│   ├── schemas.go      # List schemas
│   ├── tables.go       # List tables
│   ├── columns.go      # Describe columns
│   ├── connections.go  # Connection management
│   ├── export.go       # Data export
│   ├── history.go      # Query history
│   ├── mcp.go          # MCP server command
│   └── completion.go   # Shell completion
├── internal/
│   ├── tui/            # Terminal UI (Bubble Tea)
│   │   ├── model.go
│   │   ├── connection_view.go
│   │   ├── browser_view.go
│   │   ├── editor_view.go
│   │   ├── results_view.go
│   │   ├── history_view.go
│   │   ├── chat_view.go
│   │   ├── export_view.go
│   │   └── where_view.go
│   ├── config/         # Configuration (Viper)
│   ├── database/       # Database manager
│   └── history/        # Query history
├── pkg/
│   ├── mcp/            # MCP server implementation
│   │   ├── server.go   # Server setup
│   │   ├── tools.go    # Tool handlers
│   │   └── credentials.go # Connection resolution
│   └── styles/         # UI styling (Lipgloss)
├── skills/             # Claude Code skills
│   ├── whodb/          # Main database skill
│   ├── query-builder/  # Natural language → SQL
│   └── schema-designer/ # Schema design assistance
├── agents/             # Claude Code agents
│   ├── database-analyst.md
│   ├── query-optimizer.md
│   └── report-generator.md
└── plugin.json         # Plugin manifest
```

## Development

```bash
# Run in development mode
go run .

# Run tests
go test ./...

# Build with race detector
go build -race -o whodb-cli .

# Lint
golangci-lint run ./...
```

## License

Apache License 2.0 - See LICENSE file for details.
