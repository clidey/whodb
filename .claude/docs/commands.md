# Development Commands Reference

## Backend

```bash
# Run backend
cd core && go run .

# Build CE binary
cd core && go build -o whodb .

# Run tests
cd core && go test ./...
```

## Frontend

```bash
# Run frontend (separate terminal)
cd frontend && pnpm start

# E2E tests (all CE databases)
cd frontend && pnpm run cypress:ce

# E2E test single database
cd frontend && pnpm run cypress:db postgres
# Available: postgres, mysql, mysql8, mariadb, sqlite, mongodb, redis, elasticsearch, clickhouse

# Type check
cd frontend && pnpm run build
```

## CLI

```bash
# Build CLI
cd cli && go build -o whodb-cli .

# Run interactive mode
cd cli && go run .

# Run CLI tests
cd cli && go test ./...
```

## GraphQL Workflow

When modifying GraphQL:

1. Edit schema: `core/graph/schema.graphqls`
2. Run backend code generation: `cd core && go generate ./...`
3. Start backend server
4. Run frontend code generation: `cd frontend && pnpm run generate`
5. Import generated hooks from `@graphql` alias

```typescript
// Always use generated hooks, never inline gql strings
import { useMyQuery } from '@graphql';
```
