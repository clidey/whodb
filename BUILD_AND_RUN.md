# WhoDB Build and Run Guide

This guide provides all build and run instructions for WhoDB Community Edition (CE) and Enterprise Edition (EE).

## Prerequisites

### Required Tools
- **Go** 1.21 or higher - [Download](https://golang.org/dl/)
- **Node.js** 18 or higher - [Download](https://nodejs.org/)
- **pnpm** - Install with `npm install -g pnpm`
- **Git** (for version info in builds)

### Optional Tools
- **Docker** (for containerized builds and deployment)
- **Ollama** (for AI chat features) - [Download](https://ollama.com/)

## Quick Start

### Running WhoDB (Easiest)

#### Production Mode
```bash
# Community Edition
./run.sh

# Enterprise Edition
./run.sh --ee
```
Access at: http://localhost:8080

#### Development Mode (with hot-reload)
```bash
# Community Edition
./dev.sh

# Enterprise Edition
./dev.sh --ee
```
Backend: http://localhost:8080, Frontend: http://localhost:1234

### Docker Quick Start
```bash
# Using Docker
docker run -it -p 8080:8080 clidey/whodb

# Using Docker Compose
docker-compose up
```

## Building for Production

### Quick Build Commands

```bash
# Build everything - Community Edition
./build.sh

# Build everything - Enterprise Edition
./build.sh --ee

# Build specific components
./build.sh --backend-only
./build.sh --frontend-only

# Clean build (removes artifacts first)
./build.sh --clean
```

### Build Output Locations
- **CE Backend**: `whodb` (in project root)
- **EE Backend**: `whodb-ee` (in project root)
- **Frontend**: `frontend/build/`

## GraphQL Code Generation

GraphQL code generation is required when modifying GraphQL schemas or queries.

### Backend GraphQL Generation

#### Community Edition
```bash
cd core
go generate ./...
```

#### Enterprise Edition

The EE edition uses native GraphQL schema extension support. The schema extension file at `ee/core/graph/schema.extension.graphqls` extends the base schema with enterprise-specific types.

**From EE directory:**
```bash
cd ee
GOWORK=$PWD/../go.work.ee go generate .
```

**From root directory:**
```bash
GOWORK=$PWD/go.work.ee go generate ./ee/...
```

**Note:** The generation now uses gqlgen directly with native schema extension support instead of manual merging scripts. Any changes to `ee/core/graph/schema.extension.graphqls` will be automatically picked up during generation.

### Frontend GraphQL Generation

Frontend GraphQL types must be generated from a running backend.

#### Prerequisites
1. Backend must be running on http://localhost:8080
2. Backend must have introspection enabled (development mode)

#### Generate Frontend Types

**Community Edition:**
```bash
# Terminal 1: Start backend with introspection
cd core
go run .

# Terminal 2: Generate frontend types
cd frontend
pnpm run generate
```

**Enterprise Edition:**
```bash
# Terminal 1: Start EE backend with introspection
cd core
GOWORK=$PWD/../go.work.ee go run -tags ee .

# Terminal 2: Generate frontend types
cd frontend
pnpm run generate:ee
```

### Important GraphQL Notes
- Frontend types location: `frontend/src/generated/graphql.tsx` (CE) or `ee/frontend/src/generated/graphql.tsx` (EE)
- Always use `@graphql` import alias in code
- Regenerate types after any backend schema changes

## Development Setup

### Manual Backend Development

#### Community Edition
```bash
cd core
go run .
```

#### Enterprise Edition
```bash
cd core
GOWORK=$PWD/../go.work.ee go run -tags ee .
```

### Manual Frontend Development

#### First Time Setup
```bash
cd frontend
pnpm install
```

#### Development Server
```bash
# Community Edition
pnpm start

# Enterprise Edition
pnpm start:ee
```

## Testing

### Backend Tests
```bash
# Community Edition
cd core
go test ./... -cover

# Enterprise Edition
cd core
GOWORK=$PWD/../go.work.ee go test -tags ee ./... -cover
```

### Frontend E2E Tests

The E2E test commands have been consolidated for better usability. They automatically handle:
- Setting up the test environment (backend server, frontend server, Docker services)
- Running Cypress tests
- Cleaning up all processes and containers when done

#### Running E2E Tests

```bash
cd frontend

# Community Edition - Interactive Cypress UI
pnpm run cypress:ce

# Community Edition - Headless (for CI/automation)
pnpm run cypress:ce:headless

# Enterprise Edition - Interactive Cypress UI
pnpm run cypress:ee

# Enterprise Edition - Headless (for CI/automation)
pnpm run cypress:ee:headless
```

#### What These Commands Do

1. **Setup Phase** (`_test:*:setup` - runs automatically):
   - Cleans up any previous test environment
   - Builds and starts the backend test server with coverage
   - Sets up test databases (SQLite for CE, additional databases for EE)
   - Starts Docker containers for test databases
   - Launches the frontend Vite dev server on port 3000

2. **Test Phase**:
   - Opens Cypress UI (interactive) or runs tests headlessly
   - Tests run against the local environment on port 3000

3. **Cleanup Phase** (`_test:*:cleanup` - runs automatically):
   - Stops the frontend Vite server
   - Stops the backend test server
   - Removes Docker containers and volumes
   - Cleans up temporary test files

#### Auxiliary Test Commands

These commands are prefixed with `_` and are used internally by the main test commands. They are not recommended for direct use:

```bash
# Individual setup/cleanup commands (not recommended for direct use)
pnpm run _test:ce:setup     # Sets up CE test environment
pnpm run _test:ce:cleanup    # Cleans up CE test environment
pnpm run _test:ee:setup     # Sets up EE test environment
pnpm run _test:ee:cleanup    # Cleans up EE test environment

# Individual Cypress commands (not recommended for direct use)
pnpm run _cypress:ce:open   # Opens Cypress UI without setup
pnpm run _cypress:ce:run    # Runs Cypress headlessly without setup
```

## Docker Builds

### Build Docker Images

#### Single Architecture Build
```bash
# Community Edition
docker build -f core/Dockerfile -t whodb:ce .

# Enterprise Edition (requires EE access)
docker build -f core/Dockerfile.ee -t whodb:ee .
```

#### Multi-Architecture Build
```bash
# Setup buildx (one time)
docker buildx create --use

# Build and push CE
docker buildx build --platform linux/amd64,linux/arm64 \
  -t your-registry/whodb:ce \
  -f core/Dockerfile . --push

# Build and push EE
docker buildx build --platform linux/amd64,linux/arm64 \
  -t your-registry/whodb:ee \
  -f core/Dockerfile.ee . --push
```

### Docker Compose Configuration
```yaml
version: "3.8"
services:
  whodb:
    image: clidey/whodb
    environment:
      # Optional AI configuration
      - OLLAMA_URL=http://localhost:11434
      - ANTHROPIC_API_KEY=${ANTHROPIC_API_KEY}
      - OPENAI_API_KEY=${OPENAI_API_KEY}
    ports:
      - "8080:8080"
    volumes:
      # Optional for SQLite databases
      - ./data:/data
```

## Environment Variables

### Backend
- `PORT=8080` - Change default port
- `OLLAMA_URL` - Ollama server URL
- `ANTHROPIC_API_KEY` - Claude API key
- `OPENAI_API_KEY` - OpenAI API key

### Frontend Build
- `VITE_API_URL` - Custom API endpoint
- `VITE_BUILD_EDITION` - Set to 'ee' for Enterprise
- `VITE_DEFAULT_THEME` - Set default theme

## Troubleshooting

### Common Issues

#### "pnpm is not installed"
```bash
npm install -g pnpm
```

#### "EE directory not found"
- Ensure you have access to EE modules
- The `ee` directory must exist in the project root

#### GraphQL Generation Fails
```bash
# Ensure all dependencies are downloaded
cd core && go mod download
cd ../ee && go mod download

# For EE generation, use one of the options above
```

#### EE Runtime Error: "nil pointer dereference" on Login
This happens when EE plugins aren't properly registered. Ensure:
1. You have the `server_ee.go` file in the core directory with:
   ```go
   //go:build ee
   
   package main
   
   import (
       _ "github.com/clidey/whodb/ee/core/src/plugins"
   )
   ```
2. You're running with the `-tags ee` flag
3. The GOWORK environment variable points to `go.work.ee`

#### Frontend Build Issues
```bash
# Clean and reinstall
cd frontend
rm -rf node_modules pnpm-lock.yaml
pnpm install

# Check TypeScript errors
pnpm exec tsc --noEmit
```

#### Port Already in Use
```bash
# Find process using port 8080
lsof -i :8080
# Kill the process
kill -9 <PID>
```

### Clean Build
If experiencing persistent issues:
```bash
# Full clean build
./build.sh --clean --ee

# Manual cleanup
rm -rf frontend/node_modules frontend/build core/build
rm -f whodb whodb-ee
```

## Version Information

Check version of built binaries:
```bash
./whodb --version
./whodb-ee --version
```

## Additional Resources

- [Architecture Documentation](./docs/ARCHITECTURE.md)
- [CLAUDE.md](./CLAUDE.md) - AI assistant instructions
- [Frontend Development](./frontend/README.md)
- [GraphQL Setup](./frontend/GRAPHQL_SETUP.md)

For support, please file an issue on GitHub or contact support@clidey.com.