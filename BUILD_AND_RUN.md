# WhoDB Build and Run Guide

## Quick Start

### Run
```bash
# Community Edition
cd core
go run .

# Enterprise Edition (from root)
GOWORK=$PWD/go.work.ee go run -tags ee ./core
```

## GraphQL Generation

### Backend
```bash
# Community Edition
cd core
go generate ./...

# Enterprise Edition
cd ee
GOWORK=$PWD/../go.work.ee go generate .
```

### Frontend
```bash
# Start backend first
cd core && go run .              # CE
GOWORK=$PWD/go.work.ee go run -tags ee ./core  # EE

# Generate types
cd frontend
pnpm run generate      # CE
pnpm run generate:ee   # EE
```

## Development

### Backend
```bash
# Community Edition
cd core
go run .

# Enterprise Edition (from root)
GOWORK=$PWD/go.work.ee go run -tags ee ./core
```

### Frontend
```bash
cd frontend
pnpm install
pnpm start        # CE
pnpm start:ee     # EE
```