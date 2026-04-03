# WhoDB Development Guide

WhoDB is a database management tool. The `core/` directory contains the backend and `frontend/` contains the React UI.

If the `ee/` directory is present, read `ee/CLAUDE.md` for additional context. Do not add any code, comments, or references to `ee/` in the CE codebase.

## Non-Negotiable Rules

1. **Analyze before coding** - Read relevant files and understand patterns before writing code. Always check to see if something existing was done, or if an existing pattern can be reused or adapted.
2. **GraphQL-first** - All new API functionality via GraphQL. Never add HTTP resolvers unless explicitly needed (e.g., file downloads)
3. **No SQL injection** - Never use `fmt.Sprintf` with user input for SQL. Use parameterized queries or GORM builders. See `.claude/docs/sql-security.md`
4. **Plugin architecture** - Never use `switch dbType` or `if dbType ==` in shared code. All database-specific logic goes in plugins. See `.claude/docs/plugin-architecture.md`
5. **Documentation requirements** - All exported Go functions/types need doc comments. All exported TypeScript functions/components need JSDoc. See `.claude/docs/documentation.md`
6. **Localization requirements** - All user-facing strings must use `t()` with YAML keys. No fallback strings. No hardcoded UI text. See `.claude/docs/localization.md`
7. **Verify before completing** - After finishing any task, verify: (1) type checks pass (`pnpm run typecheck` for frontend, `go build` for backend), (2) no linting errors, (3) all added code is actually used (no dead code). See `.claude/docs/verification.md`
8. **Fallback clarification** - Do not include fallback logic UNLESS you were asked to. If you think the project could benefit from fallback logic, first ask and clarify
9. **Show proof** - When making a claim about how something outside of our codebase works, for example a 3rd party library or function, always provide official documentation or the actual code to back that up. Check online if you have to.
10. **No defensive code** - Do not program defensively. If there is an edge or use case that you think needs to be handled, first ask.

## Project Structure

```
core/                   # Backend (Go)
  cmd/whodb/main.go     # Entry point — imports plugins and creates GraphQL schema
  src/app/app.go        # AppConfig + Run() — shared server logic
  src/src.go            # Engine initialization (collects from global plugin registry)
  src/engine/registry.go # Global plugin registry (plugins self-register via init())
  src/engine/plugin.go  # PluginFunctions interface
  src/env/              # Environment variable declarations (pure, no log dependency)
  src/envconfig/        # Config-loading functions that need both env and log
  src/plugins/          # Database connectors (each has init() calling engine.RegisterPlugin)
  graph/schema.graphqls # GraphQL schema
  graph/*.resolvers.go  # GraphQL resolvers

frontend/               # React/TypeScript
  src/index.tsx        # Entry point
  src/store/           # Redux Toolkit state
  src/generated/       # GraphQL codegen output (@graphql alias)

cli/                    # Interactive TUI CLI (Bubble Tea)
desktop-ce/             # Desktop app (Wails)
desktop-common/         # Shared desktop code

.github/workflows/      # CI/CD pipelines (release, build, deploy)
```

Additional docs: `.claude/docs/cli.md` (CLI), `.claude/docs/desktop.md` (desktop), `.claude/docs/ci-cd.md` (GitHub Actions), `.claude/docs/testing.md` (testing).

## Testing

See `.claude/docs/testing.md` for comprehensive testing documentation including:
- Frontend Playwright E2E tests
- Docker container setup for test databases
- Go backend unit and integration tests
- CLI tests

Quick reference:
```bash
# Frontend Playwright E2E
cd frontend && pnpm e2e:ce:headless         # Headless (all databases)
cd frontend && pnpm e2e:ce                  # Interactive (headed)

# Backend Go tests
bash dev/run-backend-tests.sh all           # Unit + integration

# CLI tests
bash dev/run-cli-tests.sh                   # All CLI tests
```

## When Working on Backend (Go)

- Use `any` instead of `interface{}` (Go 1.18+)
- Use `plugins.WithConnection()` for all database operations - handles connection lifecycle
- SQL plugins should extend `GormPlugin` base class (`core/src/plugins/gorm/plugin.go`)
- When adding plugin functionality: add to `PluginFunctions` interface, implement in each plugin
- Use `ErrorHandler` (`core/src/plugins/gorm/errors.go`) for user-friendly error messages
- Never log sensitive data (passwords, API keys, tokens, connection strings)
- `env` package is for pure env var declarations only (no `log` import). Functions that parse env vars and need `log` for error reporting go in `envconfig`
- Delete build binaries after testing (`go build` artifacts)

## When Working on Frontend (TypeScript)

- Use PNPM, not NPM. Use pnpx, not npx
- Define GraphQL operations in `.graphql` files, then run `pnpm run generate`
- Import generated hooks from `@graphql` alias - never use inline `gql` strings
- **Keyboard shortcuts** are centralized in `frontend/src/utils/shortcuts.ts`. Never hardcode shortcut keys inline — use `SHORTCUTS.*` for definitions, `matchesShortcut()` for event handling, and `SHORTCUTS.*.displayKeys` for UI display. Platform-variant shortcuts (nav numbers) use `resolveShortcut()`. Some shortcuts also have Wails accelerators in `desktop-common/app.go` that must be updated separately

## When Updating Dependencies

Use `core/go.mod` as the reference point for dependency versions.

## Commands Quick Reference

See `.claude/docs/commands.md` for full reference.

```bash
# Backend: cd core && go run ./cmd/whodb
# Frontend: cd frontend && pnpm start
# CLI: cd cli && go run .
```

## Architecture

- **Plugin self-registration** — each plugin has `init() { engine.RegisterPlugin(...) }`. The entry point's blank imports control which plugins are registered
- **AppConfig DI** — `core/src/app/app.go` defines `AppConfig` (schema, HTTP handlers). The entry point calls `app.Run(config, staticFiles)`
- **Frontend registries** — components (`registerComponent`), database types (`registerDatabaseTypes`), icons (`registerIcons`), and functions (`registerDatabaseFunctions`) can be registered at boot. The frontend renders from registries — if something isn't registered, it's not shown
- **Import cycle note** — `src` → `router` → `graph` → `src` cycle exists. `Run()` lives in `src/app/` (not `src/`) to avoid it. Never add router/graph imports to `src/`

## Development Principles

- Clean, readable code over clever code
- Only add what is required - no overengineering
- Do not modify existing functionality without justification
- Do not rename variables/files unless necessary
- Remove unused code - no leftovers
- Only comment edge cases and complex logic, not obvious code
- Ask questions to understand requirements fully
- Use subagents to accomplish tasks faster
- Maintain professional, neutral tone without excessive enthusiasm
- When you finish a task, go back and check your work. Check that it is correct and that it is not over-engineered
