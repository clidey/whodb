---
name: review-checklist
description: Pre-commit/pre-PR review checklist for verifying changes are complete and correct
---

# Review Checklist

Run this checklist before marking work complete or creating a PR. The agent Stop
hooks already build `core/` and `cli/`, typecheck the frontend, check that literal
`t('key')` calls have an `en_US` entry, and run the agent-doc checker for changed
areas. Lint (which covers SQL `fmt.Sprintf`, database-type constants in shared
code and `interface{}`), the frontend production build and tests still need running.

## 1. Lint
```bash
# Backend (if core/ Go files changed)
cd core && ./lint.sh

# CLI (if cli/ Go files changed)
cd cli && make lint

# Frontend (if TS/TSX files changed)
cd frontend && pnpm lint
```

## 2. Frontend Production Build
```bash
cd frontend && pnpm run build:ce   # if TS/TSX files changed
```

## 3. Dead Code
- All new exports are imported somewhere and all new functions and types are used (search for their names)
- Removed code doesn't leave orphaned imports, variables or helpers
- No commented-out code, and no code left over from an abandoned approach

## 4. Security
- No sensitive data logged (passwords, tokens, connection strings)
- No hardcoded credentials

## 5. Localization (if UI changed)
- All user-facing strings use `t('key')`
- No hardcoded English text
- New or updated translation keys are limited to `en_US` unless the user explicitly asked for other languages

## 6. Testing
- `data-testid` attributes preserved on refactored elements
- New UI features have corresponding E2E test coverage
- Run relevant tests:
  ```bash
  cd core && go test ./...                          # or: bash dev/run-backend-tests.sh all
  bash dev/run-cli-tests.sh                         # if cli/ changed
  cd frontend && pnpm e2e:db:headless <affected-database>
  ```

## 7. Architecture
- No CE code that depends on or names another edition
- GraphQL uses source-first types (not `Database*`)
- Plugin changes go through `PluginFunctions` interface
- Exported Go and TypeScript symbols have doc comments (no linter checks this)

## 8. Cleanup
- No build binaries left behind
- No TODO comments (unless tracking a known issue)
- Diff is minimal — only lines required by the request
