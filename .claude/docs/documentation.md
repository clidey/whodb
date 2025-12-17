# Documentation Standards

This document defines the documentation requirements for the WhoDB codebase.

## Go Documentation

### Doc Comments for Exported Items

Every exported function, type, constant, and variable MUST have a doc comment. The comment starts with the name of the item.

**Format:**
```go
// FunctionName does X for Y reason.
// It returns Z when condition is met.
func FunctionName(param Type) ReturnType {
```

**Example - Good:**
```go
// Initialize prepares the global PostHog client. It is safe to invoke multiple times.
// Only the first call performs actual initialization.
func Initialize() {
```

**Example - Bad (missing doc comment):**
```go
func Initialize() {  // NO - exported function without doc comment
```

### Comment Quality

**Good comments explain WHY:**
```go
// Fall back to empty constraints if Migrator fails.
// This maintains backward compatibility with older database versions.
return make(map[string]map[string]any), nil
```

**Bad comments restate WHAT:**
```go
// Build full table name
var fullTableName string  // NO - obvious from code
```

### What to Document

- Explain non-obvious behavior or edge cases
- Document database-specific quirks (e.g., "ClickHouse driver doesn't report affected rows")
- Explain security or performance trade-offs
- Cross-reference related functions when helpful

### What NOT to Document

- Self-explanatory variable names
- Standard library function calls
- Obvious control flow
- Index positions (e.g., "// index 0")
- Per-item comments in data lists (e.g., `dbtypes.go` type sets should NOT have comments like "// PostgreSQL canonical" next to each type)

## TypeScript/React Documentation

### JSDoc for Exported Items

Every exported function, component, type, and constant MUST have a JSDoc comment.

**Format:**
```typescript
/**
 * Brief description of what the function does.
 * @param paramName - Description of parameter
 * @returns Description of return value
 */
export function functionName(param: Type): ReturnType {
```

**Example - Good:**
```typescript
/**
 * Migrates AI-related data from the old database store to the new aiModels store.
 * This handles the transition from the legacy storage format.
 */
export function migrateAIModelsFromDatabase(): void {
```

**Example - Bad (missing JSDoc):**
```typescript
export function migrateAIModelsFromDatabase(): void {  // NO - exported without JSDoc
```

### React Components

```typescript
/**
 * Displays a table of storage unit data with editing, sorting, and export capabilities.
 * @param columns - Column definitions for the table
 * @param rows - Data rows to display
 * @param onUpdate - Callback when a row is updated
 */
export const StorageUnitTable: FC<TableProps> = ({ columns, rows, onUpdate }) => {
```

### Interface/Type Documentation

```typescript
/**
 * Configuration for a database connection profile.
 */
export interface ConnectionProfile {
  /** Unique identifier for this profile */
  id: string;
  /** Human-readable name for the connection */
  name: string;
  /** Database host address */
  host: string;
}
```

## Inline Comments

### When to Use

- Explain WHY something is done a certain way
- Document workarounds or hacks with context
- Note cross-component dependencies
- Flag database-specific behavior

**Good:**
```typescript
// Add autocomplete for SQL, but allow disabling it during Cypress tests to prevent flakiness.
// It is disabled by default in the test environment.
```

**Good:**
```go
// TODO: BIG EDGE CASE - ClickHouse driver doesn't report affected rows properly for DELETE
```

### When NOT to Use

- Restating what the code does
- Labeling obvious sections
- Commenting variable declarations with self-explanatory names

**Bad:**
```typescript
// SQL validation function
const isValidSQLQuery = (text: string): boolean => {  // NO - function name is clear
```

**Bad:**
```go
// Initialize form inputs
m.nameInput = textinput.New()  // NO - obvious from code
```

## Reference Implementation

For Go: See `core/src/analytics/posthog.go` - every exported function has a doc comment.

For TypeScript: See `frontend/src/utils/database-operators.ts` - proper JSDoc with params and returns.
