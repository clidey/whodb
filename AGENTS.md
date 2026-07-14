# WhoDB Development Guide

WhoDB is a source-first data management tool. The public GraphQL API and frontend
contract are built around `SourceType`, `SourceContract`, `SourceObject`,
`SourceObjectRef`, and `SourceSessionMetadata`. The current execution layer is
still powered mainly by database plugins under `core/src/plugins/`.

`AGENTS.md` is the canonical agent instruction file for this repository. Tool-
specific files such as `CLAUDE.md` should import or point to this file instead
of duplicating these instructions.

If the `ee/` directory is present, read `ee/AGENTS.md` for additional context. Do not add any code, comments, or references to `ee/` in the CE codebase.

## Terminology

- **"EE agent" / "ee agent"** means the in-app WhoDB EE browser AI agent feature (under `ee/`), NOT the coding-agent (Claude/Codex) configuration or any MCP server. When in doubt, ask which is meant before acting.

## Non-Negotiable Rules

1. **GraphQL-first** - All new API functionality via GraphQL. Never add HTTP resolvers unless explicitly needed (e.g., file downloads)
2. **No SQL injection** - Never use `fmt.Sprintf` with user input for SQL. Use parameterized queries or GORM builders. See `.agents/docs/sql-security.md`
3. **Plugin architecture** - Never use `switch dbType` or `if dbType ==` in shared code. All database-specific logic goes in plugins. See `.agents/docs/plugin-architecture.md`
4. **Documentation requirements** - All exported Go functions/types need doc comments. All exported TypeScript functions/components need JSDoc. See `.agents/docs/documentation.md`
5. **Localization requirements** - All user-facing strings must use `t()` with YAML keys. No fallback strings. No hardcoded UI text. When adding or updating keys, edit `en_US` only unless the user explicitly asks for other languages. See `.agents/docs/localization.md`
6. **Verify before completing** - For non-trivial tasks, define success criteria before editing. After finishing, verify: (1) type checks pass (`pnpm run build:ce` for frontend, `go build ./cmd/whodb` for backend), (2) no linting errors, (3) all added code is actually used (no dead code). See `.agents/docs/verification.md`
7. **Show proof** - When making a claim about how something outside of our codebase works, for example a 3rd party library or function, always provide official documentation or the actual code to back that up. Check online if you have to.

## Behavioral Guidelines

These guidelines reduce common LLM coding mistakes. They bias toward caution over speed — for trivial tasks, use judgment.

### 1. Think Before Coding

**Don't assume. Don't hide confusion. Surface tradeoffs.**

Before implementing:
- State your assumptions explicitly. If uncertain, ask.
- If multiple interpretations exist, present them — don't pick silently.
- If a simpler approach exists, say so. Push back when warranted.
- If something is unclear, stop. Name what's confusing. Ask.

### 2. Simplicity First

**Minimum code that solves the problem. Nothing speculative.**

- No features beyond what was asked.
- No abstractions for single-use code.
- No "flexibility" or "configurability" that wasn't requested.
- No error handling for impossible scenarios.
- No fallback logic unless explicitly asked — if you think it's needed, ask first.
- No defensive programming — if an edge case needs handling, ask first.
- If you write 200 lines and it could be 50, rewrite it.
- Right-size architecture. Start with the simplest surgical fix. Do not propose heavyweight patterns (Temporal workflows, outbox, saga, new infra) unless the simple fix is shown to be insufficient — and explain why before building it.

Ask yourself: "Would a senior engineer say this is overcomplicated?" If yes, simplify.

### 3. Surgical Changes

**Touch only what you must. Clean up only your own mess.**

When editing existing code:
- Don't "improve" adjacent code, comments, or formatting.
- Don't refactor things that aren't broken.
- Match existing style, even if you'd do it differently.
- If you notice unrelated dead code, mention it — don't delete it.

When your changes create orphans:
- Remove imports/variables/functions that YOUR changes made unused.
- Don't remove pre-existing dead code unless asked.

The test: Every changed line should trace directly to the user's request.

### 4. Goal-Driven Execution

**Define success criteria. Loop until verified.**

Transform tasks into verifiable goals:
- "Add validation" → "Write tests for invalid inputs, then make them pass"
- "Fix the bug" → "Write a test that reproduces it, then make it pass"
- "Refactor X" → "Ensure tests pass before and after"

For debugging, confirm the root cause with evidence (a log, trace, or minimal repro) before changing code. Don't fix-and-see; if the first fix doesn't hold, re-diagnose rather than trying another speculative fix.

For multi-step tasks, state a brief plan:
```
1. [Step] → verify: [check]
2. [Step] → verify: [check]
3. [Step] → verify: [check]
```

Strong success criteria let you loop independently. Weak criteria ("make it work") require constant clarification.

### 5. CSS and Styling Changes

**Diagnose before editing. One targeted change at a time.**

- For a styling fix, state the specific approach (which selector/property/value and why) before editing.
- Make one targeted change at a time; don't broadly rewrite styles speculatively.
- If a styling fix fails twice, stop and ask for guidance (or a reference screenshot / exact value) rather than continuing to guess.

## Execution Workflow

For non-trivial tasks, use a short goal-driven loop:

1. Identify the expected behavior or failure path.
2. Choose the smallest change that satisfies the request.
3. Add or update focused tests when the behavior is testable and the risk justifies it.
4. Run the relevant verification commands and inspect the diff before finishing.

## Agent Operating Model

- Treat `AGENTS.md` as the shared source of truth for Codex, Claude Code via import, opencode, Pi, and other compatible coding agents.
- Use `.agents/README.md` as the index for deeper agent guidance, and read only the relevant linked file before editing.
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

frontend/               # React/TypeScript (uses @clidey/ux component library)
  src/index.tsx        # Entry point
  src/store/           # Redux Toolkit state
  src/generated/       # GraphQL codegen output (@graphql alias)

cli/                    # Interactive TUI CLI (Bubble Tea)
desktop-ce/             # Desktop app (Wails)
desktop-common/         # Shared desktop code

.github/workflows/      # CI/CD pipelines (release, build, deploy)
```

Additional agent docs: `.agents/docs/cli.md` (CLI), `.agents/docs/desktop.md` (desktop), `.agents/docs/ci-cd.md` (GitHub Actions), `.agents/docs/testing.md` (testing), `.agents/docs/analytics.md` (PostHog analytics contract). For adding new data sources, follow `DATA_SOURCE_GUIDE.md` (EE-specific additions in `ee/DATA_SOURCE_GUIDE_EE.md`).

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
- Desktop (`desktop-ce/`, `desktop-common/`): `.agents/rules/desktop.md`
- CI/CD (`.github/`): `.agents/rules/ci-cd.md`

## Procedures

For multi-step tasks, follow the step-by-step workflows in `.agents/workflows/`. Read the relevant file before starting:

| Task | Guide |
|------|-------|
| Add a new database plugin | `.agents/workflows/new-plugin.md` |
| Add a GraphQL query/mutation end-to-end | `.agents/workflows/new-graphql-field.md` |
| Add a new frontend page | `.agents/workflows/new-frontend-page.md` |
| Add or update translation keys | `.agents/workflows/add-translation.md` |
| Add a CLI command or TUI view | `.agents/workflows/cli-feature.md` |
| Add or modify desktop app functionality | `.agents/workflows/desktop-feature.md` |
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
- **HTTP route extensions** — CE-owned HTTP routes live in `core/graph/SetupHTTPServer`. Edition or platform add-ons register their own non-GraphQL routes with `graph.RegisterHTTPRoutes`; do not add add-on route names or unsupported stubs to CE.
- **Frontend registries** — components (`registerComponent`), source types (`registerSourceTypeOverrides`), icons (`registerIcons`), and source utilities (`registerSourceUtilities`) can be registered at boot. The frontend renders from registries — if something isn't registered, it's not shown
- **Import cycle note** — `src` → `router` → `graph` → `src` cycle exists. `Run()` lives in `src/app/` (not `src/`) to avoid it. Never add router/graph imports to `src/`

## Development Principles

- Clean, readable code over clever code
- Prefer existing style and local helper APIs over new abstractions
- Only comment edge cases and complex logic, not obvious code
- Maintain professional, neutral tone without excessive enthusiasm
