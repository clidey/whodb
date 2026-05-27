---
name: new-graphql-field
description: Add a new GraphQL query or mutation end-to-end (schema → resolver → frontend)
---

# Add a New GraphQL Field

## Steps

### 1. Update Schema
Edit `core/graph/schema.graphqls` (CE) or `ee/core/graph/schema.extension.graphqls` (EE).

Use source-first naming for public queries:
```graphql
type Query {
    SourceNewFeature(input: SourceObjectRefInput!): SourceNewFeatureResult!
}
```

### 2. Run Backend Codegen
```bash
cd core && go generate .
# Or for EE:
cd ee && go generate .
```
This creates resolver stubs in `*.resolvers.go`.

### 3. Implement Resolver
Fill in the generated stub. Source-first resolvers should:
- Look up the plugin via the source adapter
- Call the appropriate `PluginFunctions` method
- Return the source-typed result

### 4. Add Plugin Method (if new capability)
1. Add to `PluginFunctions` interface in `core/src/engine/plugin.go`
2. Add default in `core/src/plugins/gorm/plugin.go` (return `ErrUnsupported` if not universally applicable)
3. Implement in relevant plugins

### 5. Run Frontend Codegen
```bash
cd frontend && pnpm run generate
# Or for EE:
cd ee/frontend && pnpm run generate
```

### 6. Create Frontend Operation
Add `.graphql` file in `frontend/src/generated/` or alongside the component:
```graphql
query SourceNewFeature($input: SourceObjectRefInput!) {
  SourceNewFeature(input: $input) {
    ...fields
  }
}
```

Then re-run `pnpm run generate` to get the typed hook.

### 7. Use in Component
```typescript
import { useSourceNewFeatureQuery } from '@graphql';

const { data, loading } = useSourceNewFeatureQuery({ variables: { input } });
```

### 8. Verification
```bash
cd core && go build ./cmd/whodb && go vet ./...
cd frontend && pnpm run build:ce
```
