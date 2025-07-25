# WhoDB Build and Run Guide

This guide consolidates all build and run information for WhoDB Community Edition (CE) and Enterprise Edition (EE).

## Prerequisites

### Required Tools
- **Go** 1.21 or higher - [Download](https://golang.org/dl/)
- **Node.js** 18 or higher - [Download](https://nodejs.org/)
- **pnpm** - Install with `npm install -g pnpm`
- **Git** (for version info in builds)

### Optional Tools
- **Docker** (for containerized builds and deployment)
- **Make** (for Makefile usage)
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

## Development Setup

### Manual Backend Setup

#### Community Edition
```bash
cd core
go run .
```

#### Enterprise Edition
```bash
cd core
go run -tags ee .
```

### Manual Frontend Setup

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

## GraphQL Generation

GraphQL types must be generated from a running backend before building the frontend.

### Prerequisites
- Backend must be running on http://localhost:8080
- Use development mode for introspection: `ENVIRONMENT=dev go run .`

### Generate Types

#### For Community Edition
```bash
# Terminal 1: Start CE backend
cd core
ENVIRONMENT=dev go run .

# Terminal 2: Generate types
cd frontend
pnpm run generate:ce
```

#### For Enterprise Edition
```bash
# Terminal 1: Start EE backend
cd core
ENVIRONMENT=dev go run -tags ee .

# Terminal 2: Generate types
cd frontend
pnpm run generate:ee
```

### Important Notes
- CE types are tracked in git at `frontend/src/generated/graphql.tsx`
- EE types are private in `ee/frontend/src/generated/graphql.tsx`
- Always use `@graphql` import alias, never direct imports
- Regenerate types after any backend GraphQL schema changes

## Building for Production

### Quick Build Commands

#### Build Everything
```bash
# Community Edition
./build.sh

# Enterprise Edition
./build.sh --ee
```

#### Build Specific Components
```bash
# Backend only
./build.sh --backend-only
./build.sh --ee --backend-only

# Frontend only
./build.sh --frontend-only
./build.sh --ee --frontend-only

# Clean build (removes artifacts first)
./build.sh --clean
./build.sh --ee --clean
```

### Build Output Locations
- **CE Backend**: `core/whodb`
- **EE Backend**: `core/whodb-ee`
- **Frontend**: `frontend/dist/`

### Manual Build Process

#### 1. Frontend Build (Must be done first)
```bash
# Ensure types are generated first (see GraphQL Generation section)

# Community Edition
cd frontend
pnpm run build

# Enterprise Edition
cd frontend
pnpm run build:ee
```

#### 2. Copy Frontend to Backend
```bash
# Remove old build and copy new frontend build
rm -rf ../core/build
cp -r build ../core/build
```

#### 3. Backend Build (Embeds the frontend)
```bash
# Community Edition
cd core
go build -o whodb

# Enterprise Edition
cd core
go build -tags ee -o whodb-ee
```

**Note**: The backend embeds the frontend build, so the frontend must be built and copied first.

## Docker Builds

### Build Docker Images

#### Single Architecture Build
```bash
# Community Edition
docker build -f core/Dockerfile -t whodb:ce .

# Enterprise Edition (requires EE access)
docker build -f core/Dockerfile.ee -t whodb:ee .
```

#### Multi-Architecture Build with Push
```bash
# Community Edition
docker buildx build --platform linux/amd64,linux/arm64 \
  -t whodb-ce-TEST:0.0.0 \
  -f core/Dockerfile . --push

# Enterprise Edition
docker buildx build --platform linux/amd64,linux/arm64 \
  -t whodb-ee-TEST:0.0.0 \
  -f core/Dockerfile.ee . --push
```

**Note**: For multi-architecture builds, ensure you have:
1. Docker buildx enabled
2. A builder instance created: `docker buildx create --use`
3. The target registry configured for pushing

### Docker Compose Configuration
```yaml
version: "3.8"
services:
  whodb:
    image: clidey/whodb
    environment:
      # Optional Ollama configuration
      - WHODB_OLLAMA_HOST=localhost
      - WHODB_OLLAMA_PORT=11434
      
      # Optional AI API keys
      - WHODB_ANTHROPIC_API_KEY=...
      - WHODB_OPENAI_API_KEY=...
    ports:
      - "8080:8080"
    volumes:
      # Optional for SQLite databases
      - ./data:/db
```

## Troubleshooting

### Common Issues

#### "pnpm is not installed"
```bash
npm install -g pnpm
```

#### "EE directory not found"
- Ensure you have access to EE modules
- The `ee` directory must be in the project root
- Run validation: `./scripts/validate-ee.sh`

#### GraphQL Generation Fails
```bash
# Ensure backend is running with introspection enabled
cd core
ENVIRONMENT=dev go run .

# Download dependencies
go mod download

# For EE, also run in ee directory
cd ee
go mod download
```

#### TypeScript Errors During Build
```bash
# Check types without building
cd frontend
pnpm exec tsc --noEmit

# Regenerate GraphQL types
pnpm run generate:ce  # or generate:ee
```

#### Frontend Build Missing
If the backend complains about missing `build/` directory:
```bash
cd frontend
pnpm install
pnpm run build
cp -r ./dist ../core/build/
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
# Full clean build for CE
./build.sh --clean

# Full clean build for EE
./build.sh --clean --ee

# Manual cleanup
rm -rf core/build frontend/dist frontend/node_modules
rm -f core/whodb core/whodb-ee
```

## Advanced Configuration

### Environment Variables

#### Backend
- `ENVIRONMENT=dev` - Enable GraphQL introspection
- `PORT=8080` - Change default port

#### Frontend
- `VITE_API_URL` - Custom API endpoint
- `VITE_BUILD_EDITION` - Set to 'ee' for Enterprise
- `VITE_DEFAULT_THEME` - Set default theme

### Custom Builds

#### Backend with Debug Info
```bash
cd core
go build -tags "ee,debug" -o whodb-debug
```

#### Frontend with Custom API
```bash
cd frontend
VITE_API_URL=https://api.example.com pnpm run build
```

## Version Information

Built binaries include version information:
```bash
./core/whodb --version
./core/whodb-ee --version
```

## Additional Resources

- [Architecture Documentation](./ARCHITECTURE.md)
- [Contributing Guide](./CONTRIBUTING.md)
- [Frontend Development Guide](./frontend/README.md)
- [Full Documentation](https://whodb.com/docs/)

For support, please reach out to support@clidey.com or file an issue on GitHub.