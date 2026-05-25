---
name: cli-feature
description: Add a new command or TUI view to the WhoDB CLI
---

# Add a CLI Feature

## Adding a New Command

### 1. Create Command File
`cli/cmd/<name>.go`:
```go
package cmd

import "github.com/spf13/cobra"

var <name>Cmd = &cobra.Command{
    Use:   "<name>",
    Short: "Description",
    RunE: func(cmd *cobra.Command, args []string) error {
        // Implementation using database.Manager
        return nil
    },
}

func init() {
    rootCmd.AddCommand(<name>Cmd)
}
```

### 2. Key Rules
- Use `database.Manager` for all DB operations (direct plugin access, not GraphQL)
- Identity-driven text: use `pkg/identity` helpers, never hardcode `whodb-cli`
- Structured output: use `pkg/output` writers (table/json/csv/ndjson) for machine-readable commands
- Keep `main.go` thin — it calls the shared runtime, nothing else

### 3. Verification
```bash
cd cli && go build . && go vet ./...
```

---

## Adding a New TUI View

### 1. Create View File
`cli/internal/tui/<name>_view.go`:

Implement the Bubble Tea model interface:
- `Init() tea.Cmd`
- `Update(msg tea.Msg) (tea.Model, tea.Cmd)`
- `View() string`

### 2. Add to MainModel
- Add view mode to `internal/tui/model.go`
- Add navigation in `Update()`
- Use `pkg/styles` for consistent rendering

### 3. Key Patterns
- Vim-like navigation (hjkl)
- Tab for view switching
- Esc for going back
- Don't use lipgloss Padding with viewport output (use manual prefix)
- Use `database.Manager` for data access

### 4. Verification
```bash
cd cli && go test ./internal/tui/... && go build .
```

## EE CLI Notes
- EE CLI lives in `ee/cli/`, mirrors CE structure
- EE is additive — adds plugins and identity, doesn't replace shared code
- Shared CLI code in `cli/` must remain edition-neutral
- EE bootstrap registers EE plugins in `ee/cli/internal/bootstrap/bootstrap.go`
