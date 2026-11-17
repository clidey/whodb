---
name: new-graphql-field
description: Add a new GraphQL query or mutation end-to-end (schema → resolver → frontend)
---

# Add a New GraphQL Field

## Steps

### 1. Update Schema
Edit `core/graph/schema.graphqls`.

Use source-first naming for public queries:
```graphql
type Query {
    SourceNewFeature(input: SourceObjectRefInput!): SourceNewFeatureResult!
}
```

Classify every new root field before implementing its resolver:

- Protected is the default and requires no policy entry.
- If unauthenticated access is an explicit requirement, add the actual root
  field name to the public-field policy in `core/src/auth/graphql_authorization.go`.
- Never authorize by GraphQL operation name. See `.agents/rules/graphql.md`.

### 2. Run Codegen
The agent hook runs `cd core && go generate .` and `cd frontend && pnpm run generate`
after a `.graphqls` edit. Without the hook, run both. Backend codegen creates the
resolver stubs in `*.resolvers.go`; frontend codegen reads the schema file, so the
backend does not need to be running.

### 3. Implement Resolver
Fill in the generated stub. Source-first resolvers should:
- Look up the plugin via the source adapter
- Call the appropriate `PluginFunctions` method
- Return the source-typed result

### 4. Add Plugin Method (if new capability)
1. Add to `PluginFunctions` interface in `core/src/engine/plugin.go`
2. Add defaults in `core/src/plugins/gorm/` and `core/src/engine/base_plugin.go` (return `errors.ErrUnsupported` if not universally applicable)
3. Implement in relevant plugins
4. If the capability is public, declare it in the source contract (`core/src/sourcecatalog/catalog.go`)

### 5. Create Frontend Operation
Add a `.graphql` file next to the component or in `frontend/src/graphql/` (codegen reads `src/**/*.graphql`):
```graphql
query SourceNewFeature($input: SourceObjectRefInput!) {
  SourceNewFeature(input: $input) {
    ...fields
  }
}
```

Then run `cd frontend && pnpm run generate` to get the typed hook (the hook does not run for `.graphql` operation files).

### 6. Use in Component
```typescript
import { useSourceNewFeatureQuery } from '@graphql';

const { data, loading } = useSourceNewFeatureQuery({ variables: { input } });
```

### 7. Verification
```bash
cd core && go build ./cmd/whodb && go vet ./...
cd frontend && pnpm run build:ce
```

For public fields or authorization-sensitive changes, add a regression that
uses a misleading public-looking operation name with a protected selected
field and proves the operation is rejected before resolver execution.
