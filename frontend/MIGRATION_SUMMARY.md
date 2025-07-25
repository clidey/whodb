# GraphQL CE/EE Migration Summary

## Changes Made

### 1. Updated Codegen Configurations
- **CE Config** (`codegen.ce.yml`): Outputs to `src/generated/graphql.tsx`
- **EE Config** (`codegen.ee.yml`): Outputs to `../ee/frontend/src/generated/graphql.tsx`

### 2. Updated .gitignore
- Removed ignore for `src/generated/graphql.tsx`
- CE GraphQL types are now tracked in git
- EE GraphQL types remain in the private EE module

### 3. Created Smart Import System
- Added `@graphql` alias in `vite.config.ts` that dynamically resolves based on `VITE_BUILD_EDITION`
- Added `@graphql` path mapping in `tsconfig.json` for IDE support
- Created `tsconfig.ee.json` for EE-specific TypeScript configuration

### 4. Updated All GraphQL Imports
- Migrated all files from direct imports to use `@graphql` alias
- Updated 17 files total

### 5. Created Documentation
- `GRAPHQL_SETUP.md`: Complete guide for the new setup
- Migration script: `scripts/migrate-graphql-imports.js`

## How It Works

When you import from `@graphql`:
- **In CE mode**: Resolves to `src/generated/graphql.tsx`
- **In EE mode**: Resolves to `../ee/frontend/src/generated/graphql.tsx`

The resolution happens at build time based on the `VITE_BUILD_EDITION` environment variable.

## Benefits

1. **No Conflicts**: CE and EE types are completely separate
2. **Public CE Types**: CE GraphQL types can be checked into the public repository
3. **Private EE Types**: EE types remain in the private module
4. **Single Import Style**: Developers use `@graphql` regardless of edition
5. **Type Safety**: TypeScript properly resolves types for each edition

## Next Steps

1. Generate CE types: `npm run generate:ce`
2. Generate EE types: `npm run generate:ee` (when needed)
3. Commit the CE GraphQL types to git
4. Use `@graphql` for all future GraphQL imports