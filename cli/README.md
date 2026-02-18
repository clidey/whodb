# WhoDB CLI

An interactive, production-ready command-line interface for WhoDB with a Claude Code-like experience.

## Features

- **Interactive TUI** - Terminal UI built with Bubble Tea
- **Multi-Database Support** - PostgreSQL, MySQL/MariaDB, SQLite, MongoDB, Redis, ClickHouse, ElasticSearch
- **Table Browser** - Navigate schemas and tables with visual grid layout
- **WHERE Builder** - Build AND-based filters for table browsing
- **SQL Editor** - Multi-line editor with schema-aware autocomplete
- **AI Chat** - Optional AI-assisted querying with consent gate (requires configured provider)
- **Responsive Data Viewer** - Paginated results with horizontal column scrolling
- **Column Selection** - Choose which columns are visible in results
- **Export Capabilities** - Export to CSV and Excel formats
- **Query History** - Persistent history with re-execution
- **Shell Completion** - Bash/Zsh/Fish install plus PowerShell script generation
- **Programmatic Mode** - JSON/CSV/plain output for scripting and automation
- **MCP Server** - Model Context Protocol server for AI assistants (Claude, Cursor, etc.)

## Installation

### Native Install (Recommended)

**macOS / Linux:**

```bash
curl -fsSL https://raw.githubusercontent.com/clidey/whodb/main/cli/install/install.sh | bash
```

**Windows (PowerShell):**

```powershell
irm https://raw.githubusercontent.com/clidey/whodb/main/cli/install/install.ps1 | iex
```

The native installer:
- Detects your OS and architecture
- Downloads the correct binary from GitHub releases
- Installs to `~/.local/bin` (macOS/Linux) or `%LOCALAPPDATA%\WhoDB\bin` (Windows)
- Adds to PATH if needed

To install a specific version:

```bash
# macOS/Linux
curl -fsSL https://raw.githubusercontent.com/clidey/whodb/main/cli/install/install.sh | bash -s v0.62.0

# Windows
$env:WHODB_VERSION = "v0.62.0"; irm https://raw.githubusercontent.com/clidey/whodb/main/cli/install/install.ps1 | iex
```

### Homebrew (macOS/Linux)

```bash
brew install whodb-cli
```

### npm

```bash
npm install -g @clidey/whodb-cli
```

Or with npx (no install):

```bash
npx @clidey/whodb-cli
```

### From Source

Requires Go 1.21+:

```bash
git clone https://github.com/clidey/whodb.git
cd whodb/cli
go build -o whodb-cli .
```

Or using the Makefile:

```bash
cd cli
make build
make install  # installs to /usr/local/bin
```

### Using Docker

```bash
# Build the Docker image (from repo root)
docker build -t whodb-cli:latest -f cli/Dockerfile .

# Or pull pre-built
docker pull clidey/whodb-cli:latest
```

### Verify Installation

```bash
whodb-cli --version
whodb-cli --help
```

## Quick Start

### 1. Connect to a Database

If you omit required flags, the interactive connection form opens:

```bash
whodb-cli connect
```

#### PostgreSQL

```bash
whodb-cli connect \
  --type postgres \
  --host localhost \
  --port 5432 \
  --user postgres \
  --database mydb \
  --name my-postgres
```

#### PostgreSQL (non-interactive password)

```bash
printf "%s\n" "$PGPASSWORD" | whodb-cli connect \
  --type postgres \
  --host localhost \
  --port 5432 \
  --user postgres \
  --database mydb \
  --name my-postgres \
  --password
```

#### MySQL

```bash
whodb-cli connect \
  --type mysql \
  --host localhost \
  --port 3306 \
  --user root \
  --database mydb \
  --name my-mysql
```

#### SQLite

```bash
whodb-cli connect \
  --type sqlite \
  --user sqlite \
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
  --database mydb \
  --name my-mongo
```

### 1b. Use Environment Profiles

Commands that accept `--connection` can also use environment profiles, for example `WHODB_POSTGRES='[{"alias":"prod","host":"localhost","user":"user","password":"pass","database":"mydb","port":"5432"}]'` or `WHODB_MYSQL_1='{"alias":"dev","host":"localhost","user":"user","password":"pass","database":"devdb","port":"3306"}'`. Each object supports `alias` (connection name), `host`, `user`, `password`, `database`, `port`, and optional `config` for advanced settings. `port` stays at the root level; the CLI also forwards it as the `Port` advanced key when building plugin credentials, so you do not need to include `Port` in `config`. Advanced `config` keys are plugin-specific; see `core/src/plugins/*/db.go` for the keys that are read.

```bash
# Array format (multiple profiles for a database type)
export WHODB_POSTGRES='[{"alias":"prod","host":"localhost","user":"user","password":"pass","database":"mydb","port":"5432"}]'

# Numbered format (one profile per variable)
export WHODB_MYSQL_1='{"alias":"dev","host":"localhost","user":"user","password":"pass","database":"devdb","port":"3306"}'
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

- `--debug`: Enable debug mode
- `--no-color`: Disable colored output

### connect

Connect to a database and optionally save the connection. If required flags are missing, the interactive connection form opens.

```bash
whodb-cli connect [flags]
```

Flags:

- `--type`: Database type (postgres, mysql, sqlite, mongodb, redis, clickhouse, elasticsearch, mariadb)
- `--host`: Database host (default: localhost)
- `--port`: Database port (default varies by type)
- `--user`: Username
- `--database`: Database name
- `--schema`: Preferred schema (optional)
- `--name`: Connection name (saves for later use)
- `--password`: Read password from stdin when not using a TTY (pipe a single line)

On a TTY, you will be prompted for the password with input hidden.

### query

Execute a SQL query directly. Use `-` to read SQL from stdin.

```bash
whodb-cli query "SQL" [flags]
```

Flags:

- `--connection, -c`: Connection name to use (optional; if omitted, the first available connection is used)
- `--format, -f`: Output format: `auto`, `table`, `plain`, `json`, `csv`
- `--quiet, -q`: Suppress informational messages

`auto` uses table output for terminals and plain output for pipes.

### completion

Generate or install shell completion scripts.

```bash
# Show help
whodb-cli completion

# Print completion script to stdout
whodb-cli completion bash
whodb-cli completion zsh
whodb-cli completion fish
whodb-cli completion powershell

# Install completion (auto-detects shell)
whodb-cli completion install

# Install for specific shell
whodb-cli completion install bash

# Uninstall completion
whodb-cli completion uninstall
```

Install paths (bash/zsh rc files updated automatically):

- Bash: `~/.local/share/bash-completion/completions/whodb-cli`
- Zsh: `~/.zsh/completions/_whodb-cli`
- Fish: `~/.config/fish/completions/whodb-cli.fish`
- PowerShell: Manual install (see `whodb-cli completion powershell`)

## Programmatic Commands

These commands output structured data for scripting, automation, and AI integration.

### schemas

List database schemas.

```bash
whodb-cli schemas --connection my-postgres --format json
```

Flags:
- `--connection, -c`: Connection name (optional; if omitted, the first available connection is used)
- `--format, -f`: Output format: `auto`, `table`, `plain`, `json`, `csv`
- `--quiet, -q`: Suppress informational messages

### tables

List tables in a schema.

```bash
whodb-cli tables --connection my-postgres --schema public --format json
```

Flags:
- `--connection, -c`: Connection name (optional; if omitted, the first available connection is used)
- `--schema, -s`: Schema name (default varies by database)
- `--format, -f`: Output format: `auto`, `table`, `plain`, `json`, `csv`
- `--quiet, -q`: Suppress informational messages

### columns

Describe table columns.

```bash
whodb-cli columns --connection my-postgres --table users --format json
```

Flags:
- `--connection, -c`: Connection name (optional; if omitted, the first available connection is used)
- `--table, -t`: Table name (required)
- `--schema, -s`: Schema name
- `--format, -f`: Output format: `auto`, `table`, `plain`, `json`, `csv`
- `--quiet, -q`: Suppress informational messages

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

Flags (applies to all subcommands):
- `--format, -f`: Output format: `auto`, `table`, `plain`, `json`, `csv`
- `--quiet, -q`: Suppress informational messages

### export

Export table data or query results to file.

```bash
# Export to CSV
whodb-cli export --connection my-postgres --table users --format csv --output users.csv

# Export to Excel
whodb-cli export --connection my-postgres --table orders --format excel --output orders.xlsx

# Export query results
whodb-cli export --connection my-postgres --query "SELECT * FROM users" --output users.csv
```

Flags:
- `--connection, -c`: Connection name (optional; if omitted, the first available connection is used)
- `--table, -t`: Table name (required unless using `--query`)
- `--query, -Q`: SQL query to export results from (use instead of `--table`)
- `--schema, -s`: Schema name
- `--format, -f`: Export format: `csv` or `excel` (auto-detected from filename if omitted)
- `--output, -o`: Output file path (required)
- `--delimiter, -d`: CSV delimiter (default: comma)
- `--quiet, -q`: Suppress informational messages

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
- `--limit, -l`: Limit number of results (0 = no limit)
- `--format, -f`: Output format: `auto`, `table`, `plain`, `json`, `csv`
- `--quiet, -q`: Suppress informational messages

## MCP Server

WhoDB can run as an MCP (Model Context Protocol) server, enabling AI assistants like Claude, Cursor, and others to query your databases.

### Start the MCP Server

```bash
# Default: stdio transport (for Claude Desktop, Claude Code, etc.)
whodb-cli mcp serve

# HTTP transport (for cloud deployments, Docker, Kubernetes)
whodb-cli mcp serve --transport=http --port=3000
```

This starts an MCP server that exposes these tools:

| Tool | Description |
|------|-------------|
| `whodb_connections` | List available database connections |
| `whodb_schemas` | List schemas in a database (set `include_tables` for tables too) |
| `whodb_tables` | List tables in a schema (set `include_columns` for column details too) |
| `whodb_columns` | Describe table columns |
| `whodb_query` | Execute SQL queries (results include `column_types`) |
| `whodb_confirm` | Confirm pending write operations (only when confirm-writes is enabled) |
| `whodb_pending` | List pending write confirmations (only when confirm-writes is enabled) |

Write operations require confirmation by default. Use `--allow-write` to disable confirmations, or `--read-only` to block writes entirely.

### Transport Modes

**stdio (default)** - For local CLI integration with Claude Desktop, Claude Code, etc.

```bash
whodb-cli mcp serve
```

**HTTP** - For cloud deployments, Docker, Kubernetes, or shared access.

```bash
whodb-cli mcp serve --transport=http --host=0.0.0.0 --port=8080
```

HTTP mode exposes:
- `/mcp` - MCP endpoint (streaming HTTP)
- `/health` - Health check endpoint

### Security Modes

| Mode | Flag | Description |
|------|------|-------------|
| Confirm-writes | *(default)* | Write operations require user approval |
| Safe mode | `--safe-mode` | Read-only + strict security (for demos/playgrounds) |
| Read-only | `--read-only` | Blocks all write operations |
| Allow-write | `--allow-write` | Full write access without confirmation |

### MCP Flags

**Security:**
- `--safe-mode`: Read-only + strict security (for demos/playgrounds)
- `--read-only`: Block all write operations
- `--allow-write`: Allow writes without confirmation (use with caution)
- `--allow-drop`: Allow DROP/TRUNCATE when running with `--allow-write`
- `--security`: Validation level (`strict`, `standard`, `minimal`)

**Query Limits:**
- `--timeout`: Query timeout (default 30s)
- `--max-rows`: Limit rows returned per query (0 = unlimited)
- `--allow-multi-statement`: Allow multiple SQL statements in one query

**Transport:**
- `--transport`: `stdio` (default) or `http`
- `--host`: Bind address (default: localhost)
- `--port`: Listen port (default: 3000)

**Connection Scoping:**
- `--allowed-connections`: Comma-separated list of connections to allow (restricts access)
- `--default-connection`: Default connection when not specified (does not restrict access)

```bash
# Restrict AI to specific connections only
whodb-cli mcp serve --allowed-connections prod,staging

# Set default without restricting access
whodb-cli mcp serve --default-connection prod

# Combine: restrict to prod/staging, default to staging
whodb-cli mcp serve --allowed-connections prod,staging --default-connection staging
```

When `--allowed-connections` is set:
- `whodb_connections` only shows allowed connections
- Queries to other connections are rejected
- First allowed connection becomes the default (unless `--default-connection` is set)

### Configure Connections

The MCP server uses the same connection sources as the CLI:

**Option 1: Environment Profiles** (recommended for production)

Use env profiles like `WHODB_POSTGRES='[{"alias":"prod","host":"host","user":"user","password":"pass","database":"dbname","port":"5432"}]'` or `WHODB_MYSQL_1='{"alias":"staging","host":"host","user":"user","password":"pass","database":"dbname","port":"3306"}'`. Each object supports `alias` (connection name), `host`, `user`, `password`, `database`, `port`, and optional `config`.

Use the JSON formats shown above. `alias` sets the connection name used in MCP tools.

```bash
# Array format
export WHODB_POSTGRES='[{"alias":"prod","host":"host","user":"user","password":"pass","database":"dbname","port":"5432"}]'

# Numbered format (one profile per variable)
export WHODB_MYSQL_1='{"alias":"staging","host":"host","user":"user","password":"pass","database":"dbname","port":"3306"}'
```

If `alias` is omitted, the CLI assigns a name like `postgres-1`.
Saved connections take precedence if names collide.

**Option 2: Saved Connections**

Use `whodb-cli connect --name mydb ...` to save connections that the MCP server can access.

If a tool call omits `connection`, the MCP server uses the only available connection or returns an error if multiple are available.

### MCP Client Configuration (Example)

Example configuration (from `whodb-cli mcp serve --help`):

```json
{
  "mcpServers": {
      "whodb": {
        "command": "whodb-cli",
        "args": ["mcp", "serve"],
        "env": {
          "WHODB_POSTGRES_1": "{\"alias\":\"prod\",\"host\":\"localhost\",\"user\":\"user\",\"password\":\"pass\",\"database\":\"db\"}"
        }
      }
    }
  }
```

### Docker MCP Server

```bash
docker run -i --rm \
  -e WHODB_POSTGRES_1='{"alias":"prod","host":"host","user":"user","password":"pass","database":"db"}' \
  --network host \
  whodb-cli:latest mcp serve
```

## Interactive Mode Views

### 1. Connection View

Select and manage database connections.

| Key     | Action                       |
|---------|------------------------------|
| `↑/k/Shift+Tab` | Move up                    |
| `↓/j/Tab`       | Move down                  |
| `Enter`         | Connect to selected database |
| `n`             | New connection             |
| `d`             | Delete connection          |
| `Esc`           | Back (form) / press twice to quit (list) |
| `Ctrl+C`        | Force quit                 |

Form mode: `Tab`/`Shift+Tab` or `↑/↓` to move fields, `←/→` to change database type, `Enter` to connect.

### 2. Browser View

Navigate schemas and tables in a visual grid layout.

| Key                     | Action             |
|-------------------------|--------------------|
| `↑/k` `↓/j` `←/h` `→/l` | Navigate grid      |
| `/` or `f`              | Filter tables      |
| `Ctrl+S`                | Select schema      |
| `Enter`                 | View table data    |
| `Ctrl+E`                | Open SQL editor    |
| `Ctrl+H`                | View query history |
| `Ctrl+A`                | Open AI chat       |
| `Ctrl+R`                | Refresh table list |
| `Tab`                   | Next view          |
| `Esc`                   | Disconnect         |
| `Ctrl+C`                | Quit               |

### 3. Editor View

Write and execute SQL queries with schema-aware autocomplete.

| Key                              | Action               |
|----------------------------------|----------------------|
| `Alt+Enter` (`Option+Enter` Mac) | Execute query        |
| `Ctrl+Space` (`Ctrl+@`)     | Trigger autocomplete |
| `↑/↓` or `Ctrl+P/N`         | Navigate suggestions |
| `Enter`                     | Accept suggestion    |
| `Ctrl+L`                    | Clear editor         |
| `Tab`                       | Next view            |
| `Esc`                       | Back to browser      |

Features:

- Schema-aware autocomplete (tables, columns, keywords, snippets)
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
| `s`         | Cycle page size     |
| `Shift+S`   | Custom page size    |
| `e`         | Export data         |
| `w`         | Add WHERE condition |
| `c`         | Select columns      |
| `Esc`       | Back                |

Features:

- Pagination (configurable, default 50 rows)
- Column selection
- Data export (CSV, Excel)

### 5. History View

Browse and re-execute past queries.

| Key         | Action                 |
|-------------|------------------------|
| `↑/k` `↓/j` | Navigate history       |
| `/`         | Filter history         |
| `Enter`     | Load query into editor |
| `r`         | Re-run query           |
| `Shift+D`   | Clear history          |
| `y`/`n`     | Confirm clear          |
| `Tab`       | Next view              |
| `Esc`       | Back                   |

### 6. AI Chat View

AI-assisted database chat (requires a configured provider and consent).

| Key           | Action                 |
|---------------|------------------------|
| `↑/↓`         | Cycle fields           |
| `←/→`         | Change selection       |
| `Ctrl+L`      | Load models            |
| `Ctrl+I`      | Focus message input    |
| `Ctrl+P/N`    | Select message         |
| `Enter`       | Confirm/send/view      |
| `Ctrl+R`      | Revoke consent         |
| `Esc`         | Back to browser        |

Consent gate: press `a` to accept or `Esc` to exit.

### 7. Export View

Export data to CSV or Excel format.

| Key         | Action                         |
|-------------|--------------------------------|
| `Tab`/`↑/↓` | Move between fields            |
| `←/→`       | Change format/delimiter/toggle |
| `Enter`     | Confirm export                 |
| `Esc`       | Cancel                         |

## Configuration

### Config File Location

WhoDB CLI stores data in the unified WhoDB config:

- macOS: `~/Library/Application Support/whodb/config.json`
- Linux: `$XDG_DATA_HOME/whodb/config.json` (default: `~/.local/share/whodb/config.json`)
- Windows: `%APPDATA%\\whodb\\config.json`

Development builds append `-dev` to the data directory name, and EE builds append `-ee`.

Query history is stored alongside the config as `history.json`. If a legacy `~/.whodb-cli/config.yaml` exists, it is migrated automatically.

### Config Structure

```json
{
  "cli": {
    "connections": [
      {
        "name": "local-postgres",
        "type": "Postgres",
        "host": "localhost",
        "port": 5432,
        "username": "postgres",
        "database": "mydb",
        "schema": "public"
      }
    ],
    "history": {
      "max_entries": 1000,
      "persist": true
    },
    "display": {
      "theme": "dark",
      "page_size": 50
    },
    "ai": {
      "consent_given": false
    },
    "query": {
      "timeout_seconds": 30
    }
  }
}
```

Passwords are stored in the OS keyring when available. If not available, they are written to `config.json` (new files are created with `0600` permissions).

### Environment Variables

```bash
export WHODB_CLI_DEBUG=true
export WHODB_CLI_NO_COLOR=true
export NO_COLOR=1
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
  -v ~/.local/share/whodb:/root/.local/share/whodb \
  --network host \
  whodb-cli:latest
```

### Environment Variables

- `TERM=xterm-256color` - Proper terminal colors (set by default)
- `WHODB_CLI_DEBUG` / `WHODB_CLI_NO_COLOR` / `NO_COLOR` - Control CLI output

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
mkdir -p ~/.local/share/whodb
whodb-cli connect --type postgres --host localhost --name test
```
Adjust the path for your OS (see Configuration).

**"Permissions error"**

```bash
chmod 700 ~/.local/share/whodb
chmod 600 ~/.local/share/whodb/config.json ~/.local/share/whodb/history.json
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
│   │   ├── where_view.go
│   │   ├── columns_view.go
│   │   ├── schema_view.go
│   │   └── messages.go
│   ├── config/         # Unified config.json + keyring storage
│   ├── database/       # Database manager
│   └── history/        # Query history
├── pkg/
│   ├── mcp/            # MCP server implementation
│   │   ├── server.go   # Server setup
│   │   ├── tools.go    # Tool handlers
│   │   └── credentials.go # Connection resolution
│   ├── output/         # Programmatic output formatting
│   ├── styles/         # UI styling (Lipgloss)
│   ├── version/        # Build/version info
│   └── crash/          # Panic handler and crash report
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
