# WhoDB CLI Architecture

This document provides an overview of the WhoDB CLI architecture and design decisions.

## Overview

WhoDB CLI is a production-ready, interactive command-line interface built with Go that provides a Claude Code-like
experience for database management. It leverages WhoDB's existing plugin architecture to support all database types.

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                        WhoDB CLI                             │
├─────────────────────────────────────────────────────────────┤
│                         cmd/                                 │
│  ┌──────────┐  ┌──────────┐  ┌───────────┐  ┌────────────┐│
│  │  root    │  │ connect  │  │  query    │  │completion  ││
│  └──────────┘  └──────────┘  └───────────┘  └────────────┘│
├─────────────────────────────────────────────────────────────┤
│                      internal/                               │
│  ┌──────────────────────────────────────────────────────┐  │
│  │                    tui/                               │  │
│  │  ┌────────────┐  ┌─────────────┐  ┌──────────────┐  │  │
│  │  │MainModel   │  │ConnectionView│  │BrowserView   │  │  │
│  │  └────────────┘  └─────────────┘  └──────────────┘  │  │
│  │  ┌────────────┐  ┌─────────────┐  ┌──────────────┐  │  │
│  │  │EditorView  │  │ResultsView  │  │HistoryView   │  │  │
│  │  └────────────┘  └─────────────┘  └──────────────┘  │  │
│  │  ┌────────────┐  ┌─────────────┐  ┌──────────────┐  │  │
│  │  │ChatView    │  │ExportView   │  │ WhereView    │  │  │
│  │  └────────────┘  └─────────────┘  └──────────────┘  │  │
│  └──────────────────────────────────────────────────────┘  │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐    │
│  │  config/     │  │  database/   │  │  history/    │    │
│  │  - Config    │  │  - Manager   │  │  - Manager   │    │
│  │  - Connection│  │  - Engine    │  │  - Entry     │    │
│  └──────────────┘  └──────────────┘  └──────────────┘    │
├─────────────────────────────────────────────────────────────┤
│                       pkg/                                   │
│  ┌──────────────┐                                          │
│  │  styles/     │                                          │
│  │  - Colors    │                                          │
│  │  - Styles    │                                          │
│  └──────────────┘                                          │
├─────────────────────────────────────────────────────────────┤
│                   WhoDB Core Engine                          │
│  ┌──────────────────────────────────────────────────────┐  │
│  │                    Plugin System                      │  │
│  │  PostgreSQL, MySQL, SQLite, MongoDB, Redis, etc.     │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

## Key Components

### 1. Command Layer (`cmd/`)

The command layer uses Cobra for CLI command handling:

- **root.go**: Main entry point, starts TUI, configuration initialization
- **connect.go**: Database connection command
- **query.go**: Direct SQL query execution
- **completion.go**: Shell completion (bash, zsh, fish)

### 2. TUI Layer (`internal/tui/`)

Built with Bubble Tea (Elm architecture), the TUI provides an interactive experience:

#### MainModel

- Central state management
- View mode switching
- Event routing to sub-views
- Window size management

#### Views

Each view implements the Bubble Tea model interface:

- **ConnectionView**: List and select database connections
- **BrowserView**: Navigate schemas and tables in grid layout
- **EditorView**: SQL editor with syntax highlighting and autocomplete
- **ResultsView**: Display query results in responsive, paginated tables
- **HistoryView**: Browse and re-execute past queries
- **ChatView**: AI assistant for natural language queries
- **ExportView**: Export data to CSV or Excel
- **WhereView**: Visual query builder for WHERE conditions

### 3. Business Logic Layer (`internal/`)

#### Config Package

- **Config**: Application configuration
- **Connection**: Database connection details
- Persistence to `~/.whodb-cli/config.yaml`

#### Database Package

- **Manager**: Wraps WhoDB engine
- Direct plugin access for all operations
- Connection lifecycle management
- Export functionality (CSV, Excel)

#### History Package

- **Manager**: Query history management
- Persistence to `~/.whodb-cli/history.json`
- Success/failure tracking

### 4. Styling Layer (`pkg/styles/`)

Centralized styling using Lipgloss:

- Color palette (Primary, Secondary, Success, Error, etc.)
- Reusable styles (Box, Title, Table, etc.)
- Syntax highlighting colors
- Helper functions for consistent rendering

## Data Flow

### Connection Flow

```
User Input → ConnectionView → Database Manager → WhoDB Engine → Plugin → Database
```

### Query Execution Flow

```
EditorView → Database Manager → Plugin.RawExecute() → ResultsView
                ↓
         History Manager
```

### Table Browsing Flow

```
BrowserView → Database Manager → Plugin.GetStorageUnits() → List
           ↓
     ResultsView (on selection)
```

## Design Decisions

### 1. Elm Architecture (Bubble Tea)

Chosen for:

- Predictable state management
- Excellent keyboard handling
- Responsive terminal UI
- Strong community support

### 2. Direct Plugin Access

Instead of GraphQL:

- Lower overhead for CLI operations
- Direct access to all plugin features
- Simpler error handling
- Better performance

### 3. View Separation

Each view is isolated:

- Independent update logic
- Focused responsibilities
- Easy to test
- Simple navigation

### 4. Persistent State

- Configuration persists across sessions
- History survives restarts
- Connection details saved securely (passwords in keychain on supported platforms)

### 5. Keyboard-First Design

All operations accessible via keyboard:

- Vim-like navigation (hjkl)
- Tab for view switching
- Esc for going back
- Consistent shortcuts across views

## Performance Considerations

### 1. Pagination

- Results are paginated by default (50 rows)
- Lazy loading for large tables
- Memory-efficient rendering

### 2. Syntax Highlighting

- Lightweight keyword-based highlighting
- No heavy parsing libraries
- Fast enough for interactive editing

### 3. Table Rendering

- Bubbletea's table component for efficiency
- Virtualization for large result sets
- Responsive column sizing

## Security

### 1. Password Storage

- Passwords not stored in plain text
- Configuration file has restricted permissions (0600)
- Option to use environment variables

### 2. SQL Injection

- All queries go through WhoDB's plugin layer
- Plugins use parameterized queries
- No direct string concatenation

### 3. Connection Validation

- Connections validated before use
- Plugin availability check
- Graceful error handling

## Extensibility

### Adding New Views

1. Create new view file in `internal/tui/`
2. Implement Bubble Tea model interface
3. Add to MainModel
4. Add navigation in Update()

### Adding New Commands

1. Create new file in `cmd/`
2. Implement cobra.Command
3. Register in root.go

### Custom Themes

1. Modify `pkg/styles/styles.go`
2. Add color scheme definitions
3. Update style definitions

## Testing Strategy

### Unit Tests

- Each package has test files
- Mock database connections
- Test configuration loading

### Integration Tests

- Test against real databases
- Verify plugin integration
- Check data accuracy

### Manual Testing

- Test on different terminal emulators
- Verify keyboard shortcuts
- Check responsive layout

## Future Enhancements

### Planned Features

1. **Schema Visualization**: ER diagrams in terminal
2. **Custom Themes**: User-defined color schemes
3. **Plugins**: Custom command extensions
4. **Macros**: Recorded command sequences
5. **Diff Mode**: Compare query results
6. **Bookmarks**: Save frequently used queries

### Performance Improvements

1. Caching for schema information
2. Background loading for large tables
3. Incremental rendering for results
4. Connection pooling

### UX Improvements

1. Tutorial mode for new users
2. Command palette (Ctrl+P)
3. Fuzzy search everywhere
4. Mouse support (optional)
5. Split-pane view

## Dependencies

### Core

- **bubbletea**: TUI framework
- **bubbles**: Pre-built TUI components
- **lipgloss**: Styling and layout
- **cobra**: CLI commands
- **viper**: Configuration management

### WhoDB

- **core/engine**: Database plugin system

## Deployment

### Binary Distribution

```bash
# Build for multiple platforms
cd cli
go build -o whodb-cli .

# Cross-compile
GOOS=linux GOARCH=amd64 go build -o whodb-cli-linux
GOOS=darwin GOARCH=arm64 go build -o whodb-cli-macos
GOOS=windows GOARCH=amd64 go build -o whodb-cli.exe
```

### Package Managers

- Homebrew formula (planned)
- apt/yum repositories (planned)
- Snap package (planned)
- Docker image (available)

## Contributing

### Code Style

- Follow Go conventions
- Run `gofmt` before committing
- Add comments for exported functions
- Write tests for new features

### Pull Requests

1. Fork the repository
2. Create feature branch
3. Implement changes
4. Add tests
5. Update documentation
6. Submit PR

## License

Apache License 2.0 - See LICENSE file for details.
