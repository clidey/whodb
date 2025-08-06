# AGENTS.md - WhoDB Development Guide

## Build & Test Commands
```bash
# Build
./build.sh                    # CE build (frontend + backend)
./build.sh --ee               # EE build
./dev.sh                      # CE development mode
./dev.sh --ee                 # EE development mode

# Backend Tests
cd core && go test ./... -cover                    # All tests with coverage
cd core && go test ./src/plugins/postgres -v      # Single package test

# Frontend Tests  
cd frontend && pnpm run cypress:ce                # CE E2E tests
cd frontend && pnpm run cypress:ee                # EE E2E tests

# GraphQL Generation
cd frontend && pnpm run generate                  # Generate TS types (backend must be running)
cd core && go generate ./...                      # Generate Go GraphQL code
```

## Code Style Guidelines

### Go Backend
- Use Apache 2.0 license headers on all files
- Package names: lowercase, single word (e.g., `package engine`)
- Constants: `DatabaseType_Postgres` format for enums
- Interfaces: Implement `Plugin` interface for database connectors
- Error handling: Return errors, don't panic
- Imports: Standard library first, then third-party, then local

### TypeScript Frontend  
- Use Apache 2.0 license headers with `/**` comments
- Interfaces: Prefix with `I` (e.g., `IButtonProps`)
- Components: PascalCase, use FC type
- Props: Destructure in function signature
- Imports: React first, then libraries, then local (`@/`, `@ee/`, `@graphql`)
- Styling: TailwindCSS classes, use `twMerge` for conditional styles
- GraphQL: Create `.graphql` files, run `pnpm run generate`, import from `@graphql`

### General Rules
- Clean, readable code over clever solutions
- Don't modify existing functionality unless necessary  
- Use existing patterns and libraries in the codebase
- GraphQL-first API design (avoid HTTP endpoints unless required)
- Follow dual-edition architecture (CE/EE separation)