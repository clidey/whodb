# WhoDB CLI

The CLI is an interactive terminal interface for WhoDB with split-pane TUI support. Located in `cli/` (separate Go module).

## Quick Reference

```bash
# Build
cd cli && go build -o whodb-cli .

# Run interactive mode
./whodb-cli

# Connect to database
./whodb-cli connect --type postgres --host localhost --user postgres --database mydb
./whodb-cli connect --type sqlite3 --database ./app.db
./whodb-cli connect --docker  # auto-detect running Docker DB containers

# Execute query
./whodb-cli query "SELECT * FROM users" --connection my-postgres

# Import data
./whodb-cli import --connection mydb --file data.csv --table users --create-table

# Export data
./whodb-cli export --connection mydb --table users --output users.csv

# MCP server
./whodb-cli mcp serve
```

## Architecture

```
cli/
  main.go              # Entry point — imports ALL database plugins
  cmd/                 # CLI commands (Cobra)
    root.go            # Root command, starts TUI
    connect.go         # Database connection (supports --docker flag)
    query.go           # Query execution
    import.go          # CSV/Excel import
    export.go          # CSV/Excel export
    mcp.go             # MCP server command
    completion.go      # Shell completion
  internal/
    tui/               # Terminal UI (Bubble Tea)
      model.go         # Main model, view routing, layout management
      layout/          # Split-pane layout engine (container tree)
        container.go   # Binary tree containers, geometry, rendering
        presets.go     # Named layouts (Single, Explore, Query, Full)
      pane.go          # Pane interface (all views implement this)
      pane_impl.go     # Pane interface implementations for all views
      *_view.go        # Individual views (browser, editor, results, etc.)
      autocomplete.go  # Context-aware SQL autocomplete with ranking
      sqlformat.go     # SQL formatter (Ctrl+F)
      erd_view.go      # ER diagram view
      erd_layout.go    # ER diagram box-drawing layout
      explain_view.go  # EXPLAIN query plan viewer
      import_view.go   # Import wizard
      bookmarks_view.go # Saved queries
      json_viewer.go   # JSON cell pretty-printer
      cmdlog_view.go   # Command log (transparency)
    config/            # Configuration (JSON, keyring)
    database/          # DB manager (wraps core plugins)
      manager.go       # Connection, query execution, caching
      import.go        # CSV/Excel reading, type inference, batch import
    docker/            # Docker container detection
      detect.go        # `docker ps` parsing, image-to-DB-type matching
    ssh/               # SSH tunnel support
      tunnel.go        # SSH tunnel with known_hosts verification
    history/           # Query history
    baml/              # AI/BAML initialization
  pkg/
    mcp/               # MCP server implementation
    styles/            # UI styling (Lipgloss) + theme system
      styles.go        # Color variables, render helpers
      theme.go         # 8 built-in themes, SetTheme/GetTheme
    output/            # Output formatting (table/json/csv/plain)
```

## Key Design Decisions

1. **Direct Plugin Access** — Uses WhoDB's plugin system directly (not GraphQL) for lower overhead
2. **Bubble Tea (Elm Architecture)** — Predictable state, keyboard-first design
3. **Pane Interface** — All views implement `Pane` for polymorphic layout dispatch
4. **Split-Pane Layout** — Binary tree container system. Layouts: Single, Explore (Browser|Results), Query (Editor/Results), Full (Browser|Editor/Results)
5. **Modal Views** — Export, Where, Columns, History, Bookmarks, etc. overlay as full-screen and restore the previous layout on close
6. **Async Message Routing** — Completion messages (PageLoadedMsg, QueryExecutedMsg, etc.) route by TYPE to the correct view, not by the currently focused pane
7. **Theme System** — Global mutable state via `styles.SetTheme()`. 8 built-in themes. All views use package-level color variables that update on theme change.
8. **Compact Mode** — Views suppress help text when rendered inside multi-pane layouts. A global help bar shows context-sensitive shortcuts instead.

## Configuration

Config stored at `~/.whodb/config.json`:

```json
{
  "cli": {
    "connections": [...],
    "history": { "max_entries": 1000, "persist": true },
    "display": { "theme": "default", "page_size": 50 },
    "query": { "timeout_seconds": 30 },
    "saved_queries": [...],
    "read_only": false
  }
}
```

Passwords stored in OS keyring (macOS Keychain, Linux Secret Service). Falls back to config file with 0600 permissions.

## Plugin Registration

**Critical**: `cli/main.go` must explicitly import all database plugin sub-packages:
```go
_ "github.com/clidey/whodb/core/src/plugins/postgres"
_ "github.com/clidey/whodb/core/src/plugins/mysql"
// ... etc
```
The parent package `_ "github.com/clidey/whodb/core/src/plugins"` is utility code only — it does NOT register plugins. Each plugin self-registers via `init()`.

## Known Patterns

- **lipgloss.Padding bug** — Don't use `lipgloss.Padding(1,2).Render()` when content combines ANSI-styled text with viewport output. Use manual `"  "` prefix per line instead. (See `connection_view.go` renderForm)
- **SQLite empty schema** — SQLite has no schemas. `PageLoadedMsg` and action guards check `tableName != ""` (not `schema != "" && tableName != ""`).
- **Disconnect reset** — When disconnecting, reset `connectionView.connecting = false` and call `refreshList()` to avoid stale "Connecting..." state.
- **Layout suspend/restore** — Modal views call `m.suspendLayout()` before `PushView()`. `PopView()` automatically calls `m.restoreLayout()`.
- **Tab passthrough** — Views that need Tab (ERD for table cycling, Connection for form nav) are exempted from global Tab handling in `model.go`.

## Supported Databases

Postgres, MySQL, MariaDB, Sqlite3, DuckDB, MongoDB, Redis, ClickHouse, ElasticSearch, Memcached

## Testing

```bash
cd cli

# All tests
go test ./... -short

# TUI tests only
go test ./internal/tui/... -short

# Layout engine tests
go test ./internal/tui/layout/...

# Docker detection tests (mock, no Docker needed)
go test ./internal/docker/...

# SSH tunnel tests (no SSH server needed)
go test ./internal/ssh/...

# Database import tests
go test ./internal/database/... -run "TestRead|TestDetect|TestInfer|TestPreview"
```

Tests that need a database use `setupConnectedModel()` with a temp SQLite DB and `WHODB_CLI=true`.

## Detailed Documentation

- `cli/ROADMAP.md` — Feature roadmap with completed/remaining phases and shortcut reference
- `cli/README.md` — User-facing usage guide
- `cli/ARCHITECTURE.md` — Technical architecture details
