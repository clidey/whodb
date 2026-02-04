# Development Commands Reference

## Backend

```bash
# Run backend
cd core && go run .

# Build CE binary
cd core && go build -o whodb .

# Run tests (see testing.md for full guide)
cd core && go test ./...
bash dev/run-backend-tests.sh all    # Unit + integration
```

## Frontend

```bash
# Run frontend (separate terminal)
cd frontend && pnpm start

# E2E tests - see testing.md for full guide
cd frontend && pnpm cypress:ce           # Interactive, all CE databases
cd frontend && pnpm cypress:ce:headless  # Headless, all CE databases
cd frontend && pnpm cypress:db postgres  # Single database
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

# Run CLI tests (see testing.md for full guide)
bash dev/run-cli-tests.sh            # All CLI tests
cd cli && go test ./...              # Unit tests only
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
