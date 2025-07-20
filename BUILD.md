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