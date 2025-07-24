# shadcn/ui Dependencies

To complete the migration to shadcn/ui, you need to install the following dependencies:

## Required Dependencies

```bash
pnpm add @radix-ui/react-icons @radix-ui/react-label @radix-ui/react-slot @radix-ui/react-dropdown-menu @radix-ui/react-select @radix-ui/react-switch @radix-ui/react-checkbox @radix-ui/react-toast @radix-ui/react-tooltip @radix-ui/react-progress @radix-ui/react-avatar class-variance-authority clsx lucide-react
```

## What was done

1. Created a new `ux` folder with all shadcn/ui components
2. Migrated all existing components to shadcn/ui equivalents:
   - Button → Button (with variants)
   - Card → Card (with sub-components)
   - Dropdown → Select and DropdownMenu
   - Input → Input, Label, Switch, Checkbox
   - Table → Table (with sub-components)
   - Loading → Skeleton and Spinner
   - Search → SearchInput
   - Breadcrumbs → Breadcrumb (with sub-components)
   - Notifications → Toast (with sub-components)
   - Additional components: Tooltip, Alert, Progress, Badge, Avatar

3. Added shadcn/ui CSS variables to `index.css`
4. Created `components.json` configuration file
5. Created `index.ts` for easy imports

## Next Steps

After installing dependencies:
1. Update all imports in the codebase to use the new components from `@/ux`
2. Test all components to ensure they work correctly
3. Adjust any custom styling as needed