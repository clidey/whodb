---
name: review-checklist
description: Pre-commit/pre-PR review checklist for verifying changes are complete and correct
---

# Review Checklist

Run this checklist before marking work complete or creating a PR.

## 1. Build Verification
```bash
# Backend (if Go files changed)
cd core && go build ./cmd/whodb && go vet ./...

# Frontend (if TS/TSX files changed)
cd frontend && pnpm run build:ce

# EE backend (if ee/ Go files changed)
cd ee && go build ./cmd/whodb

# EE frontend (if ee/ frontend files changed)
cd frontend && pnpm exec vite build --config ../ee/frontend/vite.ee.config.mts
```

## 2. Dead Code
- All new exports are imported somewhere
- All new functions are called
- All new types are used
- Removed code doesn't leave orphaned imports

## 3. Security
- No `fmt.Sprintf` with user input for SQL
- No sensitive data logged (passwords, tokens, connection strings)
- No hardcoded credentials

## 4. Localization (if UI changed)
- All user-facing strings use `t('key')`
- No hardcoded English text
- Translation keys added to appropriate YAML file
- Ran: `cd dev/translate && python3 detect.py && node translate.mjs`

## 5. Testing
- `data-testid` attributes preserved on refactored elements
- New UI features have corresponding E2E test coverage
- Run relevant tests:
  ```bash
  cd frontend && pnpm e2e:db:headless <affected-database>
  ```

## 6. Architecture
- No `switch dbType` or `if dbType ==` in shared code
- No CE code referencing `ee/`
- GraphQL uses source-first types (not `Database*`)
- Plugin changes go through `PluginFunctions` interface

## 7. Cleanup
- No build binaries left behind
- No commented-out code
- No TODO comments (unless tracking a known issue)
- Diff is minimal — only lines required by the request
