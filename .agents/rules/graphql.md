---
paths:
  - "core/graph/**"
  - "ee/core/graph/**"
  - "**/*.graphqls"
  - "**/*.graphql"
---

# GraphQL Rules

## Schema Changes
1. Edit `core/graph/schema.graphqls` (CE) or `ee/core/graph/schema.extension.graphqls` (EE)
2. Run codegen: `cd core && go generate .` (or `cd ee && go generate .` for EE)
3. Implement resolvers in `*.resolvers.go`
4. Regenerate frontend types: `cd frontend && pnpm run generate`

## Source-First API
- New public queries/mutations use `Source*` types: `SourceTypes`, `SourceProfiles`, `SourceObjects`, `SourceRows`, `RunSourceQuery`, `SourceGraph`
- Do NOT add new `Database*` queries or capability surfaces
- EE extensions use `schema.extension.graphqls` and delegate to CE resolver

## Resolver Patterns
- CE resolvers in `core/graph/*.resolvers.go`
- EE resolvers in `ee/core/graph/*.ee.resolvers.go` — wrap CE via delegation
- Never import `graph` from `src/` (import cycle: `src → router → graph → src`)
