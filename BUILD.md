# Building WhoDB

This guide explains how to build WhoDB in both Community Edition (CE) and Enterprise Edition (EE) modes.

## Prerequisites

- Go 1.21 or higher
- Node.js 18 or higher
- pnpm package manager

## Community Edition (CE)

The Community Edition is the default open-source version with all core features.

### Quick Build

```bash
# Build backend
cd core
go build -o whodb

# Build frontend
cd ../frontend
pnpm install
pnpm run build
```

### Using Build Script

```bash
./build.sh
```

### Development Mode

```bash
# Terminal 1 - Frontend
cd frontend
pnpm install
pnpm run dev

# Terminal 2 - Backend
cd core
go run .
```

## Enterprise Edition (EE)

The Enterprise Edition includes additional database support (MSSQL, Oracle, DynamoDB) and advanced features (charts, themes, query analyzer).

### Prerequisites

- Access to the `ee` directory containing Enterprise modules
- All CE prerequisites

### Quick Build

```bash
# Build backend with EE features
cd core
go build -tags ee -o whodb-ee

# Build frontend with EE features
cd ../frontend
pnpm install
VITE_BUILD_EDITION=ee pnpm run build
```

### Using Build Script

```bash
./build.sh --ee
```

The script will automatically validate that EE modules are available before building.

### Development Mode

```bash
# Terminal 1 - Frontend with EE
cd frontend
pnpm install
VITE_BUILD_EDITION=ee pnpm run dev

# Terminal 2 - Backend with EE
cd core
go run -tags ee .
```

## Docker Images

### Community Edition

```bash
docker build -f core/Dockerfile -t whodb:latest .
```

### Enterprise Edition

```bash
docker build -f core/Dockerfile.ee -t whodb:ee .
```

## GraphQL Generation

### Schema Structure

- **Core schema**: `core/graph/schema.graphqls` - Contains base types
- **EE extension**: `ee/core/graph/schema.extension.graphqls` - Extends enums and types
- **Merged schema**: `core/graph/schema.merged.graphqls` - Combined schema (generated)

### Backend GraphQL Generation

#### Community Edition
```bash
# Generate GraphQL code for CE
./scripts/generate-graphql.sh community

# Or manually:
cd core
go run github.com/99designs/gqlgen generate
```

#### Enterprise Edition
```bash
# Generate GraphQL code for EE (includes schema merging)
./scripts/generate-graphql.sh ee

# Or manually:
./scripts/merge-schema.sh ee
cd core
go run github.com/99designs/gqlgen generate --config gqlgen.ee.yml
```

The EE generation process:
1. Merges `schema.graphqls` with `schema.extension.graphqls`
2. Adds enterprise database types (MSSQL, Oracle, DynamoDB) to the DatabaseType enum
3. Generates Go code using the merged schema

### Frontend GraphQL Generation

The frontend generates TypeScript types from the running backend's GraphQL endpoint.

#### Prerequisites
- Backend must be running on `http://localhost:8080`
- The backend version (CE or EE) determines the generated types

#### Generate Frontend Types

```bash
cd frontend

# For CE types: Run CE backend first
npm run generate

# For EE types: Run EE backend first
npm run generate
```

The `codegen.yml` configuration fetches the schema from the running backend.

### Complete EE Workflow

To generate both backend and frontend with EE extensions:

```bash
# 1. Generate and build EE backend
./build.sh --ee

# 2. Start the EE backend
cd core
./whodb-ee &

# 3. Wait for backend to start
sleep 5

# 4. Generate frontend types with EE schema
cd ../frontend
npm run generate

# 5. Build frontend with EE support
VITE_BUILD_EDITION=ee npm run build
```

## Build Validation

Before building EE, you can validate that all required modules are present:

```bash
./scripts/validate-ee.sh
```

## Environment Variables

### Frontend

- `VITE_BUILD_EDITION=ee` - Enable Enterprise Edition features in frontend build

### Backend

- Use `-tags ee` build tag to include Enterprise plugins

## Features by Edition

### Community Edition
- MySQL, PostgreSQL, SQLite3, MongoDB, Redis, ClickHouse, Elasticsearch
- Full CRUD operations
- Query execution
- Schema visualization
- Chat interface
- Analytics (PostHog)

### Enterprise Edition (Additional)
- MSSQL, Oracle, DynamoDB support
- Advanced charts (line, pie)
- Query execution plan analyzer
- Custom themes
- No analytics tracking

## Troubleshooting

### EE Build Fails

If the EE build fails with "ee directory not found":

1. Ensure you have access to Enterprise modules
2. The `ee` directory should be in the project root
3. Run `./scripts/validate-ee.sh` to check all requirements

### Frontend EE Components Not Loading

1. Ensure `VITE_BUILD_EDITION=ee` is set when building/running frontend
2. Check browser console for module loading errors
3. Verify the `ee/frontend` directory structure is complete

### Backend EE Plugins Not Available

1. Ensure you're using `-tags ee` when building or running
2. Check that `ee/core/src/plugins` contains all EE plugins
3. Verify `ee/go.mod` exists and is valid