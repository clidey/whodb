---
paths:
  - "cli/**"
---

# CLI Rules

## Architecture
- Database commands and the TUI call core plugins directly through `cli/internal/database` (`database.Manager`), not GraphQL. Hosted platform commands are the exception: they use the hosted GraphQL API through `cli/internal/platform`.
- `main.go` only passes identity and bootstrap to the shared runner in `cli/app`. Keep command and TUI logic out of it.
- Source type lists, completions and SSL modes come from the core source catalog through `cli/internal/sourcetypes`. Import, mock data, ERD, suggestions and diff reuse core logic; don't keep CLI-local copies.
- Shared CLI code stays edition-neutral: no edition-name branches. Extend through bootstrap imports, identity config or hooks.

## Identity
- Command name, display name, home dir, keyring service and update-check URLs come from `cli/pkg/identity`.
- Cobra `Use`, `Short`, `Long` and `Example` text is rewritten at startup by `identity.ReplaceText` (`cli/cmd/runtime.go`), so writing `whodb` there is fine. Other runtime strings and paths must use identity helpers, for example `identity.HomePath(...)` rather than `~/.whodb-cli`.

## Output and state
- Machine-readable output (`--format json|ndjson|csv`) keeps a stable shape with no spinner or progress noise; use `cli/pkg/output`.
- CLI settings live in the `cli` section of the shared `config.json`; passwords go in the OS keyring. Workspace restore saves only lightweight UI state, never query results.
- Hosted platform commands use only the operations and fields the host's platform manifest advertises. When the server rejects an unknown operation, the CLI refreshes the manifest and retries once.

## TUI
- Don't wrap content that mixes styled text with a Bubbles viewport `View()` in `lipgloss.NewStyle().Padding(...)`: lipgloss counts escape sequences and pads the viewport lines wrongly. Prefix each line by hand, as `renderForm()` in `cli/internal/tui/connection_view.go` does.

## Verify
```bash
(cd cli && make lint && go test ./...)
bash dev/run-cli-tests.sh            # from repo root: unit + E2E; --skip-postgres, -v
```
