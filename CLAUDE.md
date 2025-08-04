# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Development requirements
To develop WhoDB, follow the below requirements every time you do a task:
1. Clean code is paramount—make sure it is easy to understand and follow
2. Do not overengineer if you can help it—only add what is required.
3. Do not remove or modify existing functionally UNLESS you have to and UNLESS you can justify it.
4. Do not change existing variable names UNLESS absolutely necessary.
5. Do not leave unused code lying around.
6. Ask as many questions as you have to in order to understand your task.
7. You MUST use multiple subagents wherever possible to help you accomplish your task faster.

## Build & Development Commands

### Community Edition (CE)
```bash
./build.sh                    # Full build (frontend + backend)
./run.sh                      # Run the application
./dev.sh                      # Development mode with hot-reload
```

### Enterprise Edition (EE)
```bash
./build.sh --ee               # Full EE build
./run.sh --ee                 # Run EE application
./dev.sh --ee                 # EE development with hot-reload
```

### Testing
```bash
# Backend tests
cd core && go test ./... -cover

# Frontend E2E tests
cd frontend
npm run cypress:ce            # CE tests
npm run cypress:ee            # EE tests
```

### GraphQL Code Generation
```bash
# Backend CE (from core/)
go generate ./...

# Backend EE (from core/) 
GOWORK=../go.work.ee go generate ./...

# Frontend (from frontend/)
npm run generate              # Generates TypeScript types from GraphQL
```

## Architecture Overview

WhoDB is a database management tool with a **dual-edition architecture**:
- **Community Edition (CE)**: Open source core features
- **Enterprise Edition (EE)**: Extended features without modifying CE code

### Backend Structure (Go)
- **Location**: `/core/`
- **Main Entry**: `core/src/main.go`
- **Plugin System**: Database connectors in `core/src/plugins/`
- **GraphQL API**: Single endpoint at `/graphql` defined in `core/graph/schema.graphqls`
- **EE Extensions**: Separate modules in `ee/core` that register additional plugins

### Frontend Structure (React/TypeScript)
- **Location**: `/frontend/`
- **Main Entry**: `frontend/src/index.tsx`
- **State Management**: Redux Toolkit in `frontend/src/store/`
- **GraphQL Client**: Apollo Client with generated types
- **EE Components**: Conditionally loaded from `ee/frontend/`

### Key Architectural Patterns

1. **Plugin-Based Database Support**
   - Each database type implements the Plugin interface
   - Plugins register themselves with the engine
   - GraphQL resolvers dispatch to appropriate plugin

2. **Unified GraphQL API**
   - All database operations go through a single GraphQL schema
   - Database-agnostic queries that work across all supported databases
   - Type safety through code generation

3. **AI Integration**
   - Multiple LLM providers (Ollama, OpenAI, Anthropic)
   - Natural language to SQL conversion
   - Schema-aware query generation

4. **Embedded Frontend**
   - Go embeds the React build using `//go:embed`
   - Single binary deployment
   - Development mode runs separate servers

## Important Development Notes

1. **Adding New Database Support**
   - Create plugin in `core/src/plugins/`
   - Implement the Plugin interface methods
   - Register in `core/src/engine/registry.go`
   - For EE: Add to `ee/core/`

2. **GraphQL Changes**
   - Modify schema in `core/graph/schema.graphqls` (CE) or `core/ee/graph/schema.graphqls` (EE)
   - Run code generation for both backend and frontend
   - Update resolvers in `core/graph/`

3. **Frontend Feature Development**
   - CE features go in `frontend/src/`
   - EE features go in `ee/frontend/`
   - Use feature flags for conditional rendering
   - Follow existing Redux patterns for state management

4. **Environment Variables**
   - `OPENAI_API_KEY`: For ChatGPT integration
   - `ANTHROPIC_API_KEY`: For Claude integration
   - `OLLAMA_URL`: For local Ollama server

5. **Docker Development**
   - Multi-stage build optimizes image size
   - Supports AMD64
   - Uses Alpine Linux for minimal runtime

## GraphQL First Architecture

### Important: GraphQL is the Default API
- **Always use GraphQL** for new API endpoints unless explicitly instructed otherwise
- **Do NOT modify or add HTTP resolvers** in `http.resolvers.go` unless specifically requested
- The codebase follows a GraphQL-first approach for all data operations

### GraphQL Implementation Pattern
1. **GraphQL queries are NOT created with inline strings**
2. **Proper GraphQL workflow**:
   - Create `.graphql` files in the appropriate frontend directory (e.g., `src/pages/[feature]/query-name.graphql`)
   - Run `pnpm run generate` (with the backend running) to generate TypeScript types and hooks
   - Import the generated documents from `@graphql` alias
   - Use the generated hooks with Apollo Client

### Example of Correct GraphQL Usage
```typescript
// WRONG - Do not do this:
const QUERY = gql`query MyQuery { ... }`;

// CORRECT - Do this instead:
// 1. Create file: src/pages/feature/my-query.graphql
// 2. Run: pnpm run generate
// 3. Import and use:
import { useMyQuery } from '@graphql';
```

### Backend GraphQL Development
- Add new queries/mutations to `core/graph/schema.graphqls`
- Implement resolvers in appropriate resolver files (e.g., `core/graph/model.resolvers.go`)
- HTTP endpoints should only be used for special cases like file downloads that can't be handled via GraphQL

## Running GraphQL Code Generation
1. Ensure the backend is running: `cd core && go run .`
2. Run code generation: `cd frontend && pnpm run generate`
3. This will update `src/generated/graphql.tsx` with all types and hooks