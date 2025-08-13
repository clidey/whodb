# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Communication Style
- Maintain a professional, neutral tone in all communications
- Use exclamation points sparingly and only when genuinely necessary
- Approach problems with the measured perspective of an experienced software engineer
- Keep a level head when discussing technical challenges and solutions
- Focus on clear, factual explanations without unnecessary enthusiasm

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

### Plugin Architecture Principles

**IMPORTANT**: The plugin architecture is designed to avoid hardcoded database type checks. Follow these principles:

1. **No Switch Statements on Database Type**
   - NEVER write `switch dbType` or `if dbType == "Postgres"` in shared code
   - All database-specific logic MUST be implemented in the respective plugin
   - Use the plugin interface methods for all database operations

2. **Plugin Interface Extension**
   - When adding new functionality, add methods to the `PluginFunctions` interface
   - Provide a default implementation in the base plugin (e.g., `GormPlugin`)
   - Override in specific plugins (PostgreSQL, MySQL, SQLite) as needed
   - NoSQL plugins should return appropriate errors for SQL-specific features

3. **Example: Mock Data Generation**
   ```go
   // WRONG - Don't do this:
   func GetConstraints(dbType string, ...) {
       switch dbType {
       case "Postgres":
           // PostgreSQL logic
       case "MySQL":
           // MySQL logic
       }
   }
   
   // CORRECT - Do this instead:
   // In engine/plugin.go - add to interface:
   GetColumnConstraints(config *PluginConfig, schema string, storageUnit string) (map[string]map[string]interface{}, error)
   
   // In each plugin - implement the method:
   func (p *PostgresPlugin) GetColumnConstraints(...) { /* PostgreSQL-specific */ }
   func (p *MySQLPlugin) GetColumnConstraints(...) { /* MySQL-specific */ }
   ```

4. **Plugin Method Delegation**
   - Helper functions in `core/src/plugins/` should simply delegate to plugin methods
   - No database type checking or switching in helper functions
   - Let each plugin handle its own implementation details

5. **Enterprise Edition Compatibility**
   - EE plugins in `ee/core/src/` will automatically inherit interface methods
   - This design allows new database types to be added without modifying CE code
   - Plugin architecture ensures clean separation between CE and EE

### SQL Security Guidelines

**CRITICAL**: Never use string formatting for SQL queries. Always use prepared statements or GORM builder functions.

1. **Never Use String Formatting for SQL**
   ```go
   // WRONG - SQL Injection vulnerability:
   query := fmt.Sprintf("SELECT * FROM %s WHERE id = '%s'", table, userInput)
   db.Raw(query).Scan(&result)
   
   // CORRECT - Use prepared statements:
   db.Raw("SELECT * FROM users WHERE id = ?", userInput).Scan(&result)
   
   // CORRECT - Use GORM builder:
   db.Table("users").Where("id = ?", userInput).Find(&result)
   ```

2. **Proper Identifier Escaping**
   - For identifiers (table/column names) that can't use placeholders, use plugin's `EscapeIdentifier()` method
   - This is particularly important for SQLite PRAGMA commands
   ```go
   // For SQLite PRAGMA (which doesn't support placeholders):
   escapedTable := p.EscapeIdentifier(tableName)
   query := fmt.Sprintf("PRAGMA table_info(%s)", escapedTable)
   ```

3. **Use WithConnection Pattern**
   - Always use `plugins.WithConnection()` for database operations
   - This ensures proper connection handling and cleanup
   ```go
   _, err := plugins.WithConnection(config, p.DB, func(db *gorm.DB) (bool, error) {
       rows, err := db.Raw("SELECT * FROM users WHERE status = ?", "active").Rows()
       // ... process rows
       return true, nil
   })
   ```

4. **GORM Query Patterns**
   - Use `db.Raw()` with placeholders for complex queries
   - Use GORM's query builder for simple operations
   - Always close rows when done
   ```go
   rows, err := db.Raw("SELECT * FROM table WHERE col = ?", value).Rows()
   defer rows.Close()
   ```

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
