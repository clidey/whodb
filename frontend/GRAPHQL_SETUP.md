# GraphQL Code Generation Setup

This document explains the GraphQL code generation setup for Community Edition (CE) and Enterprise Edition (EE) builds.

## Overview

The project supports two separate GraphQL schemas:
- **CE (Community Edition)**: The open-source version with core features
- **EE (Enterprise Edition)**: The commercial version with additional features

## File Structure

- **CE GraphQL Types**: `src/generated/graphql.tsx` (tracked in git)
- **EE GraphQL Types**: `../ee/frontend/src/generated/graphql.tsx` (in private EE module)

## Configuration Files

- `codegen.ce.yml`: Generates CE GraphQL types from CE backend
- `codegen.ee.yml`: Generates EE GraphQL types from EE backend

## Usage

### Generating GraphQL Types

```bash
# Generate CE types (default)
npm run generate:ce

# Generate EE types
npm run generate:ee
```

### Building the Application

```bash
# Build CE version (default)
npm run build

# Build EE version
npm run build:ee
```

### Development Server

```bash
# Start CE development server
npm run start

# Start EE development server
npm run start:ee
```

## Import Strategy

All GraphQL imports use the `@graphql` alias which dynamically resolves to the correct location based on the `VITE_BUILD_EDITION` environment variable:

```typescript
// Always use this import style
import { DatabaseType, useGetDatabaseQuery } from '@graphql';

// Never use direct imports like these:
// import { DatabaseType } from '../generated/graphql';
// import { DatabaseType } from '@ee/generated/graphql';
```

## How It Works

1. **Vite Configuration**: The `vite.config.ts` file contains an alias that dynamically points to either CE or EE GraphQL types based on `VITE_BUILD_EDITION`

2. **TypeScript Configuration**: The `tsconfig.json` provides path mapping for the `@graphql` alias for IDE support

3. **Build Process**: 
   - CE builds use local `src/generated/graphql.tsx`
   - EE builds use `../ee/frontend/src/generated/graphql.tsx`

## Migrating Existing Imports

If you have existing direct imports to GraphQL files, run the migration script:

```bash
node scripts/migrate-graphql-imports.js
```

This will automatically update all imports to use the `@graphql` alias.

## Important Notes

1. **CE Types in Git**: The CE generated types (`src/generated/graphql.tsx`) are now tracked in git to ensure the open-source version can be built without requiring GraphQL generation

2. **EE Types Private**: The EE generated types remain in the private EE module and are not exposed in the public repository

3. **Type Safety**: Both CE and EE types should maintain the same interface structure for shared functionality to ensure seamless switching between editions

4. **Development Workflow**: 
   - When working on CE features, ensure the CE backend is running on port 8080
   - When working on EE features, ensure the EE backend is running on port 8080
   - Always regenerate types after backend GraphQL schema changes