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

## UI Component Library: @clidey/ux

WhoDB uses `@clidey/ux` (Clidey's shared component library). **Always use existing components before creating custom ones.**

### Available Components (import from `@clidey/ux`)
- **Layout**: Sidebar, Card, Tabs, Accordion, Resizable panels, ScrollArea, Separator
- **Forms**: Input, SearchInput, TextArea, Select, SearchSelect, Checkbox, Switch, Label
- **Feedback**: Alert, AlertDialog, Dialog, Drawer, Sheet, Toaster/toast, Spinner, Skeleton, Progress, EmptyState
- **Navigation**: Breadcrumb, Pagination, Command palette, ContextMenu, DropdownMenu, Tooltip
- **Data**: Table, VirtualizedTableBody, Tree, StackList, Badge, Chart
- **Actions**: Button, ButtonGroup
- **Theme**: ThemeProvider, useTheme, ModeToggle

### Usage Pattern
```typescript
import { Button, Card, CardHeader, CardTitle, CardContent, Input, toast } from '@clidey/ux';
```

### Component Source (always up to date)
Before using an unfamiliar component, read its source directly:
- Installed: `frontend/node_modules/@clidey/ux/dist/` (type declarations)
- Full source: `../ux/src/components/ui/<component>.tsx` (if the repo is cloned as a sibling)
- Each component uses CVA (class-variance-authority) for variants — read the `variants` object to see all options

### Rules
- Never reimplement what `@clidey/ux` already provides (buttons, inputs, dialogs, cards, tables, etc.)
- Use `cn()` from `@clidey/ux` for class merging (Tailwind + clsx)
- Use `toast()` from `@clidey/ux` for notifications (Sonner-based)
- Import from `@clidey/ux` directly, not from internal paths
- For new shared UI patterns not in `@clidey/ux`, add to `frontend/src/components/` and mention it may be a candidate for upstream

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

## UI Change Verification
After any visible UI change:
1. Start dev server: `cd frontend && pnpm start`
2. Check the golden path in a browser
3. Check edge cases (empty states, loading, errors)
4. Verify both light and dark themes

## Tooling
- PNPM, not NPM. `pnpx`, not `npx`
