---
paths:
  - "frontend/**/*.ts"
  - "frontend/**/*.tsx"
  - "frontend/**/*.graphql"
---

# React Frontend Rules

## Verification (run after changes)
```bash
cd frontend && pnpm lint && pnpm typecheck && pnpm run build:ce
```

## UI Component Library: @clidey/ux
- Use `@clidey/ux` components before writing custom ones. Read the installed types in `frontend/node_modules/@clidey/ux/dist/`; variants live in each component's CVA `variants` object.
- Wrap interactive children in ux triggers (`DropdownMenuTrigger`, `TooltipTrigger`, `PopoverTrigger`, `DialogTrigger`, ...) **without** `asChild`; the library infers it for `Button`, `Badge`, `SidebarMenuButton` and intrinsic `button`/`a`/`span`/`div`, so one DOM button is rendered. Pass `asChild` only for non-ux elements that forward refs (react-router `Link`, your own `forwardRef` components). Never slot a fragment, text, array, conditional, or a component on the deny list in `frontend/scripts/trigger-children.json`; `pnpm lint` enforces this.
- New shared UI patterns go in `frontend/src/components/`; mention when one may belong upstream in `@clidey/ux`.

## GraphQL
- Define operations in `.graphql` files under `frontend/src/`; the agent hook regenerates after operation and schema edits (`pnpm run generate` otherwise). `pnpm lint` rejects inline `gql`, imports that bypass the `@graphql` alias, ux internal paths, and `tailwind-merge`/`clsx`/`sonner` in place of ux's `cn()`/`toast()`.

## Documentation
- Exported functions, components and types need JSDoc. No lint rule checks this.

## Keyboard Shortcuts
- Centralized in `frontend/src/utils/shortcuts.ts`
- Use `SHORTCUTS.*` for definitions, `matchesShortcut()` for event handling, `SHORTCUTS.*.displayKeys` for display
- Platform-variant shortcuts use `resolveShortcut()`
- Wails accelerators in `desktop-common/app.go` must be updated separately

## Testing
- Keep every `data-testid` when extracting or refactoring components; E2E selectors depend on them. Check `grep -r "data-testid" frontend/e2e/` before changing markup.

## UI Change Verification
After any visible UI change:
1. Start dev server: `cd frontend && pnpm start`
2. Check the golden path in a browser
3. Check edge cases (empty states, loading, errors)
4. Verify both light and dark themes

## Tooling
- PNPM, not NPM. `pnpx`, not `npx`
