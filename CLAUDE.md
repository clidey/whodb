# WhoDB Development Guide

WhoDB is a database management tool with dual-edition architecture: Community Edition (CE) in `/core` and Enterprise Edition (EE) in `/ee`.

**Note**: The `ee/` submodule is only available to core WhoDB developers. If `ee/` is not present, stub files provide no-op implementations so CE builds work normally. Build tags (`//go:build ee` vs `//go:build !ee`) control which code is compiled.

**EE Documentation**: All Enterprise Edition documentation, commands, and development guides are in `ee/CLAUDE.md` and `ee/.claude/docs/`. This file covers CE only.

## Non-Negotiable Rules

1. **Analyze before coding** - Read relevant files and understand patterns before writing code
2. **GraphQL-first** - All new API functionality via GraphQL. Never add HTTP resolvers unless explicitly needed (e.g., file downloads)
3. **No SQL injection** - Never use `fmt.Sprintf` with user input for SQL. Use parameterized queries or GORM builders. See `.claude/docs/sql-security.md`
4. **Plugin architecture** - Never use `switch dbType` or `if dbType ==` in shared code. All database-specific logic goes in plugins. See `.claude/docs/plugin-architecture.md`
5. **CE/EE separation** - EE code MUST stay in the ee submodule. All EE documentation is in `ee/`. No exceptions
6. **Documentation requirements** - All exported Go functions/types need doc comments. All exported TypeScript functions/components need JSDoc. See `.claude/docs/documentation.md`
7. **Localization requirements** - All user-facing strings must use `t()` with YAML keys. No fallback strings. No hardcoded UI text. See `.claude/docs/localization.md`
8. **Verify before completing** - After finishing any task, verify: (1) type checks pass (`pnpm run typecheck` for frontend, `go build` for backend), (2) no linting errors, (3) all added code is actually used (no dead code). See `.claude/docs/verification.md`

## Project Structure

```
core/                   # CE backend (Go)
  server.go             # Entry point (func main)
  src/src.go            # Engine initialization, plugin registration
  src/engine/plugin.go  # PluginFunctions interface
  src/plugins/          # Database connectors (each implements PluginFunctions)
  graph/schema.graphqls # GraphQL schema
  graph/*.resolvers.go  # GraphQL resolvers

frontend/               # React/TypeScript
  src/index.tsx        # Entry point
  src/store/           # Redux Toolkit state
  src/generated/       # GraphQL codegen output (@graphql alias)

cli/                    # Interactive TUI CLI (Bubble Tea)
desktop-ce/             # CE desktop app (Wails)
desktop-common/         # Shared desktop code

.github/workflows/      # CI/CD pipelines (release, build, deploy)
```

Additional docs: `.claude/docs/cli.md` (CLI), `.claude/docs/desktop.md` (desktop), `.claude/docs/ci-cd.md` (GitHub Actions).

## When Working on Backend (Go)

- Use `any` instead of `interface{}` (Go 1.18+)
- Use `plugins.WithConnection()` for all database operations - handles connection lifecycle
- SQL plugins should extend `GormPlugin` base class (`core/src/plugins/gorm/plugin.go`)
- When adding plugin functionality: add to `PluginFunctions` interface, implement in each plugin
- Use `ErrorHandler` (`core/src/plugins/gorm/errors.go`) for user-friendly error messages
- Never log sensitive data (passwords, API keys, tokens, connection strings)
- Delete build binaries after testing (`go build` artifacts)

## When Working on Frontend (TypeScript)

- Use PNPM, not NPM. Use pnpmx, not npx
- Define GraphQL operations in `.graphql` files, then run `pnpm run generate`
- Import generated hooks from `@graphql` alias - never use inline `gql` strings
- CE features in `frontend/src/`

## When Updating Dependencies

Use `core/go.mod` as the reference point for dependency versions.

## Commands Quick Reference

See `.claude/docs/commands.md` for full reference. EE commands are in `ee/CLAUDE.md`.

```bash
# Backend: cd core && go run .
# Frontend: cd frontend && pnpm start
# CLI: cd cli && go run .
```

## Development Principles

- Clean, readable code over clever code
- Only add what is required - no overengineering
- Do not modify existing functionality without justification
- Do not rename variables/files unless necessary
- Remove unused code - no leftovers
- Comment WHY, not WHAT - explain reasoning, edge cases, and non-obvious behavior. Never comment obvious code
- Ask questions to understand requirements fully
- Use subagents to accomplish tasks faster
- Maintain professional, neutral tone without excessive enthusiasm
- When you finish a task, go back and check your work. Check that it is correct and that it is not over-engineered