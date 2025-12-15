# WhoDB CLI

The CLI is an interactive terminal interface for WhoDB with TUI support. It's located in `cli/` (separate Go module).

## Quick Reference

```bash
# Build
cd cli && go build -o whodb-cli .

# Run interactive mode
./whodb-cli

# Connect to database
./whodb-cli connect --type postgres --host localhost --user postgres --database mydb --name my-postgres

# Execute query
./whodb-cli query "SELECT * FROM users" --connection my-postgres

# Docker
docker build -t whodb-cli:latest -f cli/Dockerfile .
docker run -it --rm whodb-cli:latest
```

## Architecture

```
cli/
  main.go           # Entry point
  cmd/              # CLI commands (Cobra)
    root.go         # Root command, starts TUI
    connect.go      # Database connection
    query.go        # Query execution
    completion.go   # Shell completion (bash/zsh/fish)
  internal/
    tui/            # Terminal UI (Bubble Tea)
      model.go      # Main model, view routing
      *_view.go     # Individual views
    config/         # Configuration (Viper)
    database/       # DB connection handling
    history/        # Query history
  pkg/
    styles/         # UI styling (Lipgloss)
```

## Key Design Decisions

1. **Direct Plugin Access** - Uses WhoDB's plugin system directly (not GraphQL) for lower overhead
2. **Bubble Tea (Elm Architecture)** - Predictable state, excellent keyboard handling
3. **View Separation** - Each view (browser, editor, results, etc.) is isolated

## Configuration

Config stored at `~/.whodb-cli/config.yaml`:

```yaml
connections:
  - name: local-postgres
    type: postgres
    host: localhost
    port: 5432
    username: postgres
    database: mydb

history:
  max_entries: 1000
  persist: true
```

## Supported Databases

postgres, mysql, sqlite, mongodb, redis, clickhouse, elasticsearch, mariadb

## Detailed Documentation

The CLI has extensive documentation:
- `cli/README.md` - Complete usage guide with keyboard shortcuts
- `cli/ARCHITECTURE.md` - Technical architecture and design decisions
