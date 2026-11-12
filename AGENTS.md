# WhoDB

WhoDB is a source-first data management tool. The public GraphQL API and frontend
are built around `SourceType`, `SourceContract`, `SourceObject`, `SourceObjectRef`
and `SourceSessionMetadata`; execution runs mostly through database plugins in
`core/src/plugins/`. If `ee/` is present, also read `ee/AGENTS.md`. CE code never
references `ee/`.

## Team Rules

These apply in every clidey repository, for people and agents alike.

- **Commits and pushes:** never commit or push without the user explicitly asking in that turn ("please commit this for me" or unambiguously equivalent). "Implement it" or "merge it into main locally" means change the files, not commit. If a commit seems necessary, stop and ask.
- **No AI attribution:** commit messages and PR descriptions carry no AI attribution: no `Co-Authored-By:` trailers for Claude, Codex, or any other agent, and no "Generated with" lines.
- **Untracked files:** never delete untracked or gitignored files without asking first; they often hold secrets or local state.
- **Agent replies:** lead with the answer or result. No preamble, no restating the question, no recap unless asked. Keep it short and explain only what isn't obvious; if you need something from the user, ask in one line.
- **Clarify first:** before starting, ask about anything where a different answer would change the work: scope, behaviour, which repo or environment, or anything hard to undo. Ask all your questions at once, then wait. Don't ask what the code or docs already answer; for small choices that are easy to change later, pick one and say which in your reply.

## Rules

1. New API surface goes through GraphQL (`core/graph/schema.graphqls`); plain HTTP only where GraphQL can't work, such as file downloads. Use the source-first queries (`SourceTypes`, `SourceObjects`, `SourceRows`, `RunSourceQuery`, ...); never add new `Database*` queries.
2. Never build SQL from strings with values in them. Identifiers go through `sqlident`; see `.agents/docs/sql-security.md`.
3. Database-specific logic lives in plugins: no `switch`/`if` on database type in shared code.
4. Exported Go and TypeScript symbols get doc comments (not lint-enforced).
5. User-facing text uses `t()` with YAML keys, no fallback strings; add `en_US` keys only.
6. Back claims about third-party behavior with docs or source code (`research-proof` skill).
7. "EE agent" means the in-app WhoDB AI agent, not coding agents.

## How we work

- No fallback logic or defensive handling for unrequested edge cases; ask first.
- Remove only the dead code your change created; mention other dead code.
- CSS: state the selector, property and value before editing; after two failed tries, ask.

## Where guidance lives

- `.agents/rules/<area>.md`: area rules (go-backend, react-frontend, graphql, localization, e2e-tests, plugin-architecture, analytics, cli, desktop, ci-cd). Claude Code loads them for matching files; in Codex, read the one for your area before editing.
- `.agents/skills/<name>/SKILL.md`: procedures. `new-plugin`, `new-graphql-field`, `new-frontend-page`, `add-translation`, `cli-feature`, `desktop-feature`, `platform-constrained-handler`, `review-checklist` (run before finishing), `research-proof`, `task-handoff`.
- `.agents/README.md`: index of all of the above.

## Commands

```bash
cd core && go run ./cmd/whodb          # backend; frontend: cd frontend && pnpm start; CLI: cd cli && go run .
cd core && ./lint.sh                   # builds the custom golangci binary, then lints
cd frontend && pnpm run build:ce       # typecheck + build; pnpm lint; pnpm generate (reads the schema file, no backend needed)
cd frontend && pnpm e2e:ce:headless    # Playwright E2E
bash dev/run-backend-tests.sh all      # Go unit + integration
bash dev/run-cli-tests.sh              # CLI tests (lint: cd cli && make lint)
```

## Architecture notes

- Plugins self-register in `init()` via `engine.RegisterPlugin`; the entry point's blank imports choose which register.
- Import cycle `src` → `router` → `graph` → `src`: `Run()` lives in `core/src/app/`; never import router or graph from `core/src/`.
- CE HTTP routes are set up in `core/graph` (`SetupHTTPServer`); add-ons register theirs with `graph.RegisterHTTPRoutes`.
- The frontend renders from registries (`registerComponent`, `registerSourceTypeOverrides`, `registerIcons`, `registerSourceUtilities`); unregistered things don't show.
- `sdk/` is its own pnpm project; its wire cores are generated from `sdk/spec/`.
