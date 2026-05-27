# WhoDB CLI

The CLI is an interactive terminal interface for WhoDB with split-pane TUI support. It lives in `cli/` and is a separate Go module.

## Quick Reference

```bash
# Build
cd cli && go build -o whodb-cli .

# Run interactive mode
./whodb-cli

# Connect to database
./whodb-cli connect --type postgres --host localhost --user postgres --database mydb
./whodb-cli connect --type sqlite3 --database ./app.db
./whodb-cli connect --docker

# Execute query
./whodb-cli query "SELECT * FROM users" --connection my-postgres
./whodb-cli query --stream --format ndjson "SELECT * FROM audit_log"

# Cloud discovery (when provider support is enabled)
./whodb-cli cloud providers list
./whodb-cli cloud connections list
./whodb-cli connect --discovered aws-prod-us-west-2/prod-db

# Schema diff
./whodb-cli diff --from staging --to prod

# Mock data
./whodb-cli mock-data --connection mydb --table orders --rows 50 --analyze

# MCP server
./whodb-cli mcp serve
```

## Current Architecture

```
cli/
  main.go                 # CE entry point
  app/
    app.go                # Shared CLI runtime entry (bootstraps identity + startup)
  cmd/                    # Cobra command tree + TUI launcher
    root.go               # Root command, startup, config/env wiring
    runtime.go            # Applies runtime identity to command text
    connect.go            # Database connection
    query.go              # Query execution
    diff.go               # Schema diff
    cloud.go              # Cloud provider inspection and discovery
    cloud_connect.go      # Shared discovered-resource connect/save helpers
    explain.go            # EXPLAIN output
    erd.go                # Graph output
    import.go             # Shared import pipeline entry
    export.go             # Export with streaming support
    mock_data.go          # FK-aware mock data generation
    suggestions.go        # Backend-generated query suggestions
    mcp.go                # MCP server command
    completion.go         # Shell completion
  internal/
    bootstrap/
      bootstrap.go        # CLI runtime setup + CE plugin registration
    tui/                  # Bubble Tea UI
      model.go            # Main model, view routing, workspace restore
      *_view.go           # Individual panes and modal views
      row_write_view.go   # Add/edit/delete row form
      diff_view.go        # TUI schema diff view
      mock_data_view.go   # Interactive mock-data flow
    database/
      manager.go          # Direct plugin access, metadata/query helpers, streaming bridge
      crud.go             # Add/update/delete row helpers
      import.go           # Shared import wrappers
      mockdata.go         # Shared mock data wrappers and guards
    config/               # Unified config section + workspace persistence
    connectionopts/       # Shared connection option mapping (SSL, etc.)
    cloud/                # Shared cloud provider runtime adapter
    baml/                 # BAML setup
  pkg/
    identity/
      identity.go         # CLI runtime identity (command name, storage names, etc.)
    output/               # Table/json/csv/plain/ndjson output + streaming writers
    mcp/                  # MCP server implementation
    styles/               # Themes and render helpers
    version/              # Version formatting
    updatecheck/          # Release check
```

## Runtime Split

The CLI now has a dedicated shared runtime layer:

- `cli/app/app.go` is the shared process runner
- `cli/internal/bootstrap/bootstrap.go` owns CE startup/bootstrap
- `cli/pkg/identity/identity.go` owns runtime identity such as:
  - command name
  - display name
  - local config/cache directory naming
  - keyring service name
  - update-check URLs

Keep CE defaults in `cli/pkg/identity`. Do not add non-CE product-specific values there. Shared code should consume generic identity fields instead of branching on edition names.

## Key Design Decisions

1. **Direct Plugin Access** — The CLI uses core plugins directly, not GraphQL, for command/TUI execution
2. **Shared Runtime** — `main.go` should stay thin and call the shared runner in `cli/app`
3. **Identity-Driven Text and Paths** — Command text, completion install paths, BAML library storage, keyring naming, and update messaging should come from `pkg/identity`
4. **Bubble Tea + Pane Model** — All panes implement the same interface and render into shared layouts
5. **Backend-Driven Database Catalog** — Database picker/completions should come from `core/src/dbcatalog`, not a CLI-local hardcoded list
6. **Shared Feature Surfaces** — Import, mock data, ERD graph, suggestions, and diff should reuse backend/shared logic rather than reimplementing locally
7. **Structured Automation Output** — Machine-readable commands should keep stable JSON/NDJSON contracts and avoid spinner noise
8. **Workspace Restore** — Persist only lightweight restorable UI state, not heavy query payloads

## Current Capabilities

- Split-pane TUI layouts with workspace restore
- Query execution with plain/json/csv/ndjson output
- Additive streaming mode for query/export
- Backend-generated suggestions
- Shared backend import pipeline
- FK-aware mock data generation
- Add/edit/delete row flows in the TUI
- Schema diff command and TUI diff view
- ERD based on backend graph data
- SSL configuration and SSL status visibility
- Expanded MCP read tool surface
- Cloud provider discovery commands backed by the shared provider runtime
- Discovered-resource connect/save flows built on the shared cloud prefill path

## Configuration

CLI state is stored in the unified `config.json` under the `cli` section through `core/src/common/config`.

The CLI also keeps edition-local files under the identity-specific home dir. Shared code should use `cli/pkg/identity.HomePath(...)` instead of hardcoding `~/.whodb-cli`.

Passwords are stored in the OS keyring when available. The keyring service name is also identity-driven.

## Rules for Future CLI Changes

### 1. Keep entrypoints thin

`main.go` should only:
- set up identity/bootstrap
- call the shared runtime

Do not move command/TUI logic back into the binary entrypoint.

### 2. Do not hardcode command identity

Avoid new hardcoded uses of:
- `whodb-cli`
- `WhoDB CLI`
- `.whodb-cli`
- `WhoDB-CLI`
- `WHODB_CLI`

Use the shared identity helpers instead.

### 3. Prefer backend/shared sources over CLI-local copies

Use shared backend logic for:
- database catalog
- import behavior
- mock data planning/generation
- graph/ERD data
- query suggestions

### 4. Do not add edition checks to shared CLI features unless there is no better extension point

If a future edition needs different behavior, prefer:
- bootstrap/import differences
- identity config
- extension hooks

Avoid baking product-specific branching into normal command/TUI paths.

## Testing

```bash
cd cli

# All CLI tests
go test ./...

# TUI tests
go test ./internal/tui/...

# Command tests
go test ./cmd

# Shared output/runtime tests
go test ./pkg/output ./pkg/version ./pkg/identity

# Full dev script
bash ../dev/run-cli-tests.sh
```

## Related Files

- `cli/README.md` — user-facing usage guide
- `cli/ARCHITECTURE.md` — broader architecture notes
- `dev/run-cli-tests.sh` — CLI verification script
