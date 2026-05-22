---
paths:
  - "frontend/**/*.ts"
  - "frontend/**/*.tsx"
  - "frontend/**/*.graphql"
---

# React Frontend Rules

## Verification (run after changes)
```bash
cd frontend && pnpm run build:ce
```

## GraphQL
- Define operations in `.graphql` files, then `pnpm run generate`
- Import generated hooks from `@graphql` alias — never inline `gql` strings

## Localization
- All user-facing strings use `t('key')` from `useTranslation('component-path')`
- No fallback strings — `t('key', 'fallback')` is a compile error
- Check `common.yaml` first before adding keys — shared terms live there
- After adding keys, run: `cd dev/translate && python3 detect.py && node translate.mjs`

## Keyboard Shortcuts
- Centralized in `frontend/src/utils/shortcuts.ts`
- Use `SHORTCUTS.*` for definitions, `matchesShortcut()` for event handling, `SHORTCUTS.*.displayKeys` for display
- Platform-variant shortcuts use `resolveShortcut()`
- Wails accelerators in `desktop-common/app.go` must be updated separately

## Testing
- Preserve all `data-testid` attributes when refactoring — E2E tests depend on them
- Search `grep -r "data-testid" frontend/e2e/` to see what selectors tests use

## Tooling
- PNPM, not NPM. `pnpx`, not `npx`
