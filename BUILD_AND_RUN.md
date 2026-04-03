# WhoDB Build and Run Guide

## Quick Start

### Run
```bash
# Community Edition
cd core
go run ./cmd/whodb

# Enterprise Edition (from ee directory)
cd ee && go run ./cmd/whodb
```

## GraphQL Generation

### Backend
```bash
# Community Edition
cd core
go generate ./...

# Enterprise Edition (from ee directory)
cd ee && go generate .
```

### Frontend
```bash
# Start backend first
cd core && go run ./cmd/whodb                     # CE
cd ee && go run ./cmd/whodb        # EE

# Generate types
cd frontend
pnpm run generate      # CE
pnpm run generate:ee   # EE
```


## Dependency Management

### Running go mod tidy

Due to the dual-module workspace architecture, `go mod tidy` must be run with the appropriate workspace context.

#### Community Edition
```bash
cd core && go mod tidy
```

#### Enterprise Edition
```bash
cd core && go mod tidy
cd ee && go mod tidy

# Or if already in ee directory
go mod tidy
```

**Note:** Always use the GOWORK environment variable to ensure local module dependencies are resolved correctly.

## Development

### Backend
```bash
# Community Edition
cd core
go run ./cmd/whodb

# Enterprise Edition (from ee directory)
cd ee && go run ./cmd/whodb
```

### Frontend
```bash
cd frontend
pnpm install
pnpm start        # CE
pnpm start:ee     # EE
```