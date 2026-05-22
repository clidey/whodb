# WhoDB Development Guide

WhoDB is a source-first data management tool. The public GraphQL API and frontend
contract are built around `SourceType`, `SourceContract`, `SourceObject`,
`SourceObjectRef`, and `SourceSessionMetadata`. The current execution layer is
still powered mainly by database plugins under `core/src/plugins/`.

`AGENTS.md` is the canonical agent instruction file for this repository. Tool-
specific files such as `CLAUDE.md` should import or point to this file instead
of duplicating these instructions.

If the `ee/` directory is present, read `ee/AGENTS.md` for additional context. Do not add any code, comments, or references to `ee/` in the CE codebase.

## Non-Negotiable Rules

1. **Analyze before coding** - Read relevant files and understand patterns before writing code. State assumptions explicitly, ask when requirements are ambiguous, and always check whether an existing pattern can be reused or adapted.
2. **GraphQL-first** - All new API functionality via GraphQL. Never add HTTP resolvers unless explicitly needed (e.g., file downloads)
3. **No SQL injection** - Never use `fmt.Sprintf` with user input for SQL. Use parameterized queries or GORM builders. See `.agents/docs/sql-security.md`
4. **Plugin architecture** - Never use `switch dbType` or `if dbType ==` in shared code. All database-specific logic goes in plugins. See `.agents/docs/plugin-architecture.md`
5. **Documentation requirements** - All exported Go functions/types need doc comments. All exported TypeScript functions/components need JSDoc. See `.agents/docs/documentation.md`
6. **Localization requirements** - All user-facing strings must use `t()` with YAML keys. No fallback strings. No hardcoded UI text. See `.agents/docs/localization.md`
7. **Verify before completing** - For non-trivial tasks, define success criteria before editing. After finishing, verify: (1) type checks pass (`pnpm run build:ce` for frontend, `go build ./cmd/whodb` for backend), (2) no linting errors, (3) all added code is actually used (no dead code). See `.agents/docs/verification.md`
8. **Fallback clarification** - Do not include fallback logic UNLESS you were asked to. If you think the project could benefit from fallback logic, first ask and clarify
9. **Show proof** - When making a claim about how something outside of our codebase works, for example a 3rd party library or function, always provide official documentation or the actual code to back that up. Check online if you have to.
10. **No defensive code** - Do not program defensively. If there is an edge or use case that you think needs to be handled, first ask.
11. **Surgical changes** - Touch only the files and lines required by the request. Do not refactor, reformat, rename, or "improve" adjacent code unless the task requires it.
12. **Simplicity first** - Solve the requested problem with the smallest clear implementation. Do not add speculative abstractions, configurability, or features.
13. **Own your cleanup** - Remove imports, variables, functions, files, and generated artifacts made unused by your own changes. Mention unrelated dead code or suspicious behavior instead of deleting it.

## Execution Workflow

For non-trivial tasks, use a short goal-driven loop:

1. Identify the expected behavior or failure path.
2. Choose the smallest change that satisfies the request.
3. Add or update focused tests when the behavior is testable and the risk justifies it.
4. Run the relevant verification commands and inspect the diff before finishing.

If the task is unclear or has multiple valid interpretations, stop and ask instead of silently choosing. If the implementation grows beyond the request, pause and simplify before continuing.

## Agent Operating Model

- Treat `AGENTS.md` as the shared source of truth for Codex, Claude Code via import, opencode, Pi, and other compatible coding agents.
- Keep always-loaded instructions concise. Move detailed workflows, checklists, and runbooks into linked docs rather than duplicating them in tool-specific files.
- Use planning mode for multi-file, risky, architectural, or ambiguous changes. Skip formal plans for obvious single-purpose edits.
- Use separate agents only for bounded sidecar work such as codebase exploration, review, test triage, or documentation lookup. Do not delegate blocking implementation work when the main session is waiting on it.
- Do not run parallel implementation sessions against the same files. Use separate worktrees or explicitly disjoint file ownership for parallel work.
- Ask review-oriented agents to find correctness, regression, security, and missing-test issues first; summaries are secondary.

## Project Structure

```
core/                   # Backend (Go)
  cmd/whodb/main.go     # Entry point — imports plugins and creates GraphQL schema
  src/app/app.go        # AppConfig + Run() — shared server logic
  src/src.go            # Engine initialization (collects from global plugin registry)
  src/engine/registry.go # Global plugin registry (plugins self-register via init())
  src/engine/plugin.go  # PluginFunctions interface
  src/source/           # Source-first public contract + connector/session interfaces
  src/sourcecatalog/    # Public source catalog exposed to GraphQL/frontend
  src/dbcatalog/        # Internal connectable-database catalog adapted into sourcecatalog
  src/env/              # Environment variable declarations (pure, no log dependency)
  src/envconfig/        # Config-loading functions that need both env and log
  src/plugins/          # Database connectors (each has init() calling engine.RegisterPlugin)
                        # Includes: postgres, mysql, sqlite3, mongodb, redis, elasticsearch,
                        #           clickhouse, duckdb, memcached (+ mariadb via mysql)
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

Additional agent docs: `.agents/docs/cli.md` (CLI), `.agents/docs/desktop.md` (desktop), `.agents/docs/ci-cd.md` (GitHub Actions), `.agents/docs/testing.md` (testing). For adding new data sources, follow `DATA_SOURCE_GUIDE.md` (EE-specific additions in `ee/DATA_SOURCE_GUIDE_EE.md`).

## Testing

See `.agents/docs/testing.md` for comprehensive testing documentation including:
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

## Domain-Specific Guidelines

For language-specific or domain-specific guidelines, refer to the shared rules in `.agents/rules/`:
- Go Backend (`core/`, `cli/`): `.agents/rules/go-backend.md`
- React Frontend (`frontend/`): `.agents/rules/react-frontend.md`
- GraphQL (`**/graph/`, `**/*.graphqls`): `.agents/rules/graphql.md`
- Localization (`**/locales/`): `.agents/rules/localization.md`
- E2E Tests (`frontend/e2e/`): `.agents/rules/e2e-tests.md`

## Procedures

For multi-step tasks, follow the step-by-step workflows in `.agents/workflows/`. Read the relevant file before starting:

| Task | Guide |
|------|-------|
| Add a new database plugin | `.agents/workflows/new-plugin.md` |
| Add a GraphQL query/mutation end-to-end | `.agents/workflows/new-graphql-field.md` |
| Add a new frontend page | `.agents/workflows/new-frontend-page.md` |
| Add or update translation keys | `.agents/workflows/add-translation.md` |
| Add a CLI command or TUI view | `.agents/workflows/cli-feature.md` |
| Add a platform-constrained HTTP handler | `.agents/workflows/platform-constrained-handler.md` |
| Prepare or consume a handoff | `.agents/workflows/task-handoff.md` |
| Prove claims about external behavior | `.agents/workflows/research-proof.md` |
| Pre-commit verification | `.agents/workflows/review-checklist.md` |

## When Updating Dependencies

Use `core/go.mod` as the reference point for dependency versions.

## Commands Quick Reference

See `.agents/docs/commands.md` for full reference.

```bash
# Backend: cd core && go run ./cmd/whodb
# Frontend: cd frontend && pnpm start
# CLI: cd cli && go run .
```

## Architecture

- **Plugin self-registration** — each plugin has `init() { engine.RegisterPlugin(...) }`. The entry point's blank imports control which plugins are registered
- **Source-first public API** — new public GraphQL/frontend work should use `SourceTypes`, `SourceProfiles`, `SourceFieldOptions`, `SourceSessionMetadata`, `SourceObjects`, `SourceRows`, `RunSourceQuery`, and `SourceGraph`. Do not add new public `Database*` queries or capability surfaces
- **AppConfig DI** — `core/src/app/app.go` defines `AppConfig` (schema, HTTP handlers). The entry point calls `app.Run(config, staticFiles)`
- **Frontend registries** — components (`registerComponent`), source types (`registerSourceTypeOverrides`), icons (`registerIcons`), and source utilities (`registerSourceUtilities`) can be registered at boot. The frontend renders from registries — if something isn't registered, it's not shown
- **Import cycle note** — `src` → `router` → `graph` → `src` cycle exists. `Run()` lives in `src/app/` (not `src/`) to avoid it. Never add router/graph imports to `src/`

## Development Principles

- Clean, readable code over clever code
- Keep every changed line tied directly to the user's request
- Only add what is required - no overengineering
- Prefer existing style and local helper APIs over new abstractions
- Do not modify existing functionality without justification
- Do not rename variables/files unless necessary
- Remove unused code introduced by your changes - no leftovers
- Do not delete unrelated dead code unless asked
- Only comment edge cases and complex logic, not obvious code
- Ask questions to understand requirements fully
- Use separate agents only for bounded sidecar work that will not conflict with the main implementation path
- Maintain professional, neutral tone without excessive enthusiasm
- When you finish a task, go back and check your work. Check that it is correct and that it is not over-engineered
