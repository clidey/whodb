# WhoDB Build and Run Guide

## Quick Start

```bash
cd core && go run ./cmd/whodb
```

## GraphQL Generation

### Backend
```bash
cd core && go generate ./...
```

### Frontend
```bash
# Start backend first
cd core && go run ./cmd/whodb

# Generate types
cd frontend && pnpm run generate
```

## Dependency Management

```bash
cd core && go mod tidy
```

## Development

### Backend
```bash
cd core && go run ./cmd/whodb
```

### Frontend
```bash
cd frontend
pnpm install
pnpm start
```
