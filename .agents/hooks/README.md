# Agent Hooks

Hooks run after tool actions (edit, write, apply_patch) and help keep code
consistent. They receive a JSON payload on stdin containing the tool input.

## Hook Scripts

| Script | Trigger | What It Does |
|--------|---------|--------------|
| `auto-format-go.sh` | `.go` files edited | Runs `gofmt -w` then `golangci-lint run --fix` |
| `auto-lint-ts.sh` | `.ts`/`.tsx` files edited | Runs `oxlint --fix` |
| `auto-graphql-codegen.sh` | `.graphqls` files edited | Runs `go generate` in `core/` and `pnpm generate` in `frontend/` |
| `verify-build.sh` | Agent stop (turn end) | Compiles/type-checks only the areas changed this turn (CE/EE Go via `go build`, CE/EE frontend via `pnpm typecheck`) and blocks the turn (exit 2) if any fail. Debounces repeated identical errors to avoid loops. |
| `session-context.sh` | Session start | Injects branch name and uncommitted file count |

Translation is intentionally not an automatic hook. Agents should add or update
`en_US` strings only; run `dev/translate` tooling manually when translations are
needed.

## Supporting Files

- `changed-files.py` — parses the JSON payload to extract file paths from edit/write/apply_patch hooks

## Platform Compatibility

These hooks use the Codex/Claude Code hook format (JSON on stdin). Pi uses an
extension-based hook system instead — wrap these scripts in a Pi extension for
equivalent functionality.

## Adding a New Hook

1. Write the script in this directory (root `.agents/hooks/`).
2. Add a forwarding shim at `ee/.agents/hooks/<script-name>` so the hook still
   resolves when the agent's cwd is inside the `ee/` submodule (its own git
   repo, so `git rev-parse --show-toplevel` resolves to `ee/` rather than the
   monorepo root). Match the existing shim pattern exactly:
   ```bash
   #!/usr/bin/env bash
   exec "$(cd "$(dirname "$0")/../../.." && pwd)/.agents/hooks/<script-name>" "$@"
   ```
   Then `chmod +x` it.
3. Register the command in both `.claude/settings.json` and `.codex/hooks.json`
   using `bash "$(git rev-parse --show-toplevel)/.agents/hooks/<script-name>"`
   under the appropriate event (`SessionStart`, `PostToolUse`, `Stop`, etc.).
   Both files must be updated — neither tool reads the other's config.
4. Update the table above with the script's trigger and behavior.
