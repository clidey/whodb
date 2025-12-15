# Verification Checklist

Before marking any task as complete, run through this verification checklist.

## Frontend (TypeScript/React)

### Type Checking
```bash
cd frontend && pnpm run typecheck
```

### Build Verification
```bash
cd frontend && pnpm run build:ce
```

### Dead Code Check
After adding new code, verify it's actually used:
- Search for function/component names to confirm they're imported and called
- Check that new exports are imported somewhere
- Remove any unused imports, variables, or functions

## Backend (Go)

### Build Verification
```bash
cd core && go build .
```

### Vet Check
```bash
cd core && go vet ./...
```

### Dead Code Check
- Verify exported functions are called from somewhere
- Check that new types are actually used
- Remove unused imports (Go compiler will catch these)

## Common Issues to Catch

1. **Unused imports** - Both Go and TypeScript will flag these
2. **Unused variables** - Especially after refactoring
3. **Orphaned helper functions** - Functions added but never called
4. **Stale utility code** - Code added for a purpose that changed
5. **Commented-out code** - Remove instead of leaving commented

## Quick Verification Commands

```bash
# Frontend full check
cd frontend && pnpm run typecheck && pnpm run build:ce

# Backend full check
cd core && go build . && go vet ./...

# Search for unused exports (manual)
# Use grep to search for function/type names and verify usage
```
