# Agent Hooks

Hooks run after tool actions (edit, write, apply_patch) and help keep code
consistent. They receive a JSON payload on stdin containing the tool input.

## Hook Scripts

| Script | Trigger | What It Does |
|--------|---------|--------------|
| `auto-format-go.sh` | `.go` files edited | Runs `gofmt -w` then `golangci-lint run --fix` |
| `auto-lint-ts.sh` | `.ts`/`.tsx` files edited | Runs `oxlint --fix` |
| `auto-graphql-codegen.sh` | `.graphqls` files edited | Runs `go generate` in `core/` and `pnpm generate` in `frontend/` |
| `auto-translate.sh` | `locales/*.yaml` files edited | Runs translation drift detection + auto-translation |
| `session-context.sh` | Session start | Injects branch name and uncommitted file count |

## Supporting Files

- `changed-files.py` — parses the JSON payload to extract file paths from edit/write/apply_patch hooks

## Platform Compatibility

These hooks use the Codex/Claude Code hook format (JSON on stdin). Pi uses an
extension-based hook system instead — wrap these scripts in a Pi extension for
equivalent functionality.