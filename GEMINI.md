# AI Agent Development Guide for WhoDB

You are an AI assistant working on the WhoDB repository. Adhere strictly to these guidelines to ensure high-quality,
consistent, and secure contributions.

## Core Directives

These are non-negotiable principles. Follow them at all times.

1. **Analyze Before Coding**: Before writing any code, you MUST analyze the existing codebase to understand its
   patterns, conventions, and architecture. Read relevant files and form a clear plan.
2. **Adhere to Project Conventions**: Your code must mimic the style, structure, and libraries of the surrounding code.
   Do not introduce new patterns or dependencies without justification.
3. **Prioritize GraphQL**: This is a GraphQL-first project. All new API functionality must be implemented via GraphQL
   unless a specific exception (like file downloads) is required. Do not add new HTTP resolvers.
4. **Security First**: Preventing SQL injection is critical. **NEVER** use string formatting (`fmt.Sprintf`) to build
   SQL queries with user input. Always use parameterized queries or GORM's builder methods.
5. **Strictly Adhere to Plugin Architecture**: All database-specific logic MUST be implemented within the corresponding
   plugin. **NEVER** use `switch` statements or `if/else` chains based on database type in shared code. If you see no
   other way around this requirement, stop and tell me why.
6. **Be consistent**: Follow existing file name patterns, variable name patterns, and so on. Do not rename variables or
   files for the sake of it, only when necessary.
7. **Tone**: Maintain a professional, neutral tone in all communications. Use exclamation points sparingly and only when
   genuinely necessary.
8. **Approach to problems**: Approach problems with the measured perspective of an experienced principal software
   engineer. Keep a level head when discussing technical challenges and solutions. Focus on clear, factual explanations
   without unnecessary enthusiasm.
9. **Ask questions**: You must ask as many questions as you have to in order to understand.
10. **Be clean**: Do not leave unused code lying around. Make sure code is easy to follow and understand.
11. **Separation between CE and EE versions**: All EE code and related functionality MUST be in the ee submodule. No
    excuses.
12. **Avoid shell scripts or adhoc solutions**: Never use shell scripts or adhoc solutions UNLESS absolutely necessary.
13. **Avoid simple comments**: Never add comments that are simple or basic - only edge cases, complicated actions, or
    proccesses can have comments explaining them.

## Development Requirements

1. Clean code is paramount—make sure it is easy to understand and follow
2. Do not overengineer if you can help it—only add what is required
3. Do not remove or modify existing functionality UNLESS you have to and UNLESS you can justify it
4. Do not change existing variable names UNLESS absolutely necessary
5. Do not leave unused code lying around
6. Ask as many questions as you have to in order to understand your task
7. You MUST use multiple subagents wherever possible to help you accomplish your task faster
8. If you do a build to test something (e.g., `go build`), delete the binary afterwards to keep the workspace clean
9. Use `any` instead of `interface{}` in all Go code (Go 1.18+ modern syntax). In general use modern Go syntax
   everywhere.
10. When updating dependencies, ensure versions are identical between Community Edition (`core/go.mod`) and Enterprise
    Edition (`ee/go.mod`) for shared dependencies. This also includes the desktop versions for CE (`desktop-ce/go.mod`) and EE (`desktop-ee/go.mod`). In general, the Community Edition (`core/go.mod`) has to be the reference point for dependency versions.
11. Never log sensitive data such as passwords, API keys, tokens, or full connection strings.
12. Always use PNPM instead of NPM.

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

## Plugin Architecture Principles

**IMPORTANT**: The plugin architecture is designed to avoid hardcoded database type checks. Follow these principles:

### 1. No Switch Statements on Database Type

- NEVER write `switch dbType` or `if dbType == "Postgres"` in shared code
- All database-specific logic MUST be implemented in the respective plugin
- Use the plugin interface methods for all database operations

### 2. Plugin Interface Extension

- When adding new functionality, add methods to the `PluginFunctions` interface
- Provide a default implementation in the base plugin (e.g., `GormPlugin`)
- Override in specific plugins (PostgreSQL, MySQL, SQLite) as needed
- NoSQL plugins should return appropriate errors for SQL-specific features

### 3. Example: Mock Data Generation

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

### 4. Plugin Method Delegation

- Helper functions in `core/src/plugins/` should simply delegate to plugin methods
- No database type checking or switching in helper functions
- Let each plugin handle its own implementation details

### 5. Enterprise Edition Compatibility

- EE plugins in `ee/core/src/` will automatically inherit interface methods
- This design allows new database types to be added without modifying CE code
- Plugin architecture ensures clean separation between CE and EE

## Go Backend Guidelines

### Modern Go Syntax

- Use modern Go syntax wherever possible (e.g., `any` instead of `interface{}`)
- Use Go 1.18+ features where appropriate

### Dependency Management

- Ensure `go.mod` versions are identical between Community (`core/`) and Enterprise (`ee/`) editions for shared
  dependencies
- While Enterprise will have dependencies that do not exist in Community, shared dependencies MUST have identical
  versions

### Security Guidelines

- **No Sensitive Logging**: Do not log passwords, API keys, tokens, or full connection strings
- **Safe DB Operations**: Use the `plugins.WithConnection()` pattern for all database operations to ensure proper
  connection handling

### SQL Security Guidelines

**CRITICAL**: Never use string formatting for SQL queries. Always use prepared statements or GORM builder functions.

#### 1. Never Use String Formatting for SQL

```go
// WRONG - SQL Injection vulnerability:
query := fmt.Sprintf("SELECT * FROM %s WHERE id = '%s'", table, userInput)
db.Raw(query).Scan(&result)

// CORRECT - Use prepared statements:
db.Raw("SELECT * FROM users WHERE id = ?", userInput).Scan(&result)

// CORRECT - Use GORM builder:
db.Table("users").Where("id = ?", userInput).Find(&result)
```

#### 2. Proper Identifier Escaping

For identifiers (table/column names) that can't use placeholders, use plugin's `EscapeIdentifier()` method:

```go
// For SQLite PRAGMA (which doesn't support placeholders):
escapedTable := p.EscapeIdentifier(tableName)
query := fmt.Sprintf("PRAGMA table_info(%s)", escapedTable)
```

#### 3. Use WithConnection Pattern

Always use `plugins.WithConnection()` for database operations:

```go
_, err := plugins.WithConnection(config, p.DB, func (db *gorm.DB) (bool, error) {
rows, err := db.Raw("SELECT * FROM users WHERE status = ?", "active").Rows()
// ... process rows
return true, nil
})
```

#### 4. GORM Query Patterns

- Use `db.Raw()` with placeholders for complex queries
- Use GORM's query builder for simple operations
- Always close rows when done

```go
rows, err := db.Raw("SELECT * FROM table WHERE col = ?", value).Rows()
defer rows.Close()
```

## TypeScript Frontend Guidelines

### GraphQL First Architecture

**Important: GraphQL is the Default API**
- **Always use GraphQL** for new API endpoints unless explicitly instructed otherwise
- **Do NOT modify or add HTTP resolvers** in `http.resolvers.go` unless specifically requested
- The codebase follows a GraphQL-first approach for all data operations

### GraphQL Workflow

All GraphQL operations must be defined in `.graphql` files:

1. Create a file (e.g., `src/mutations/my-mutation.graphql`)
2. Run `pnpm run generate` in the `frontend/` directory (the backend must be running)
3. Import and use the auto-generated hook from the `@graphql` alias

```typescript
// WRONG: Inline gql strings are forbidden
const MY_QUERY = gql`query MyQuery { ... }`;

// CORRECT: Use the generated hook
import { useMyQuery } from '@graphql';
```

### Frontend Feature Development

- CE features go in `frontend/src/`
- EE features go in `ee/frontend/`
- Use feature flags for conditional rendering
- Follow existing Redux patterns for state management

## GraphQL Code Generation

### Resolver Architecture

The project uses a dual-resolver architecture to maintain separation between CE and EE:

- CE resolvers are in `core/graph/schema.resolvers.go` and are never overwritten by EE generation
- EE resolvers extend CE resolvers through embedding and are generated as `*.ee.resolvers.go`
- Build tags control which resolver is used at compile time

### Backend GraphQL Development
- Add new queries/mutations to `core/graph/schema.graphqls`
- Implement resolvers in appropriate resolver files (e.g., `core/graph/model.resolvers.go`)
- HTTP endpoints should only be used for special cases like file downloads that can't be handled via GraphQL

## Essential Commands

### Community Edition (CE)

```bash
# Run backend
cd core && go run .

# Run frontend (separate terminal)
cd frontend && pnpm start

# Frontend E2E tests
cd frontend && pnpm run cypress:ce

# GraphQL Generation
# Backend:
cd core && go generate ./...
# Frontend (backend must be running):
cd frontend && pnpm run generate
```

### Enterprise Edition (EE)

```bash
# Run backend (from project root)
GOWORK=$PWD/go.work.ee go run -tags ee ./core

# Run frontend (separate terminal)
cd frontend && pnpm start:ee

# Frontend E2E tests
cd frontend && pnpm run cypress:ee

# GraphQL Generation
# Backend:
cd ee && GOWORK=$PWD/../go.work.ee go generate .
# Frontend (backend must be running):
cd frontend && pnpm run generate:ee
```

## Important Development Notes

### Adding New Database Support

1. Create plugin in `core/src/plugins/`
2. Implement the Plugin interface methods
3. Register in `core/src/engine/registry.go`
4. For EE: Add to `ee/core/`

### GraphQL Changes

1. Modify schema in `core/graph/schema.graphqls` (CE) or `core/ee/graph/schema.graphqls` (EE)
2. Run code generation for both backend and frontend
3. Update resolvers in `core/graph/`

## Summary

Remember: When in doubt, analyze the existing code first. Your contributions should be indistinguishable from code
written by the core maintainers. Follow the patterns, respect the architecture, and prioritize security above all else.