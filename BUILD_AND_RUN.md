# WhoDB Build and Run Guide

## Quick Start

### Run
```bash
# Community Edition
cd core
go run .

# Enterprise Edition (from ee directory)
cd ee && go run -tags ee ../core
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
cd core && go run .                     # CE
cd ee && go run -tags ee ../core        # EE

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
# From project root
cd core && go mod edit -replace github.com/clidey/whodb/ee=../ee-stub && go mod tidy && go mod edit -dropreplace github.com/clidey/whodb/ee -droprequire github.com/clidey/whodb/ee
cd ee-stub && go mod tidy

# Or if already in core directory
go mod edit -replace github.com/clidey/whodb/ee=../ee-stub && go mod tidy && go mod edit -dropreplace github.com/clidey/whodb/ee -droprequire github.com/clidey/whodb/ee
```

#### Enterprise Edition
```bash
# From project root
cd core && go mod edit -replace github.com/clidey/whodb/ee=../ee && go mod tidy && go mod edit -dropreplace github.com/clidey/whodb/ee -droprequire github.com/clidey/whodb/ee
cd ee && go mod tidy

# Or if already in core directory
go mod edit -replace github.com/clidey/whodb/ee=../ee && go mod tidy && go mod edit -dropreplace github.com/clidey/whodb/ee -droprequire github.com/clidey/whodb/ee

# Or if already in ee directory
go mod tidy
```

**Note:** Always use the GOWORK environment variable to ensure local module dependencies are resolved correctly.

## Development

### Backend
```bash
# Community Edition
cd core
go run .

# Enterprise Edition (from ee directory)
cd ee && go run -tags ee ../core
```

### Frontend
```bash
cd frontend
pnpm install
pnpm start        # CE
pnpm start:ee     # EE
```