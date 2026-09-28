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

## Operation Authorization

- GraphQL authentication is decided after gqlgen parses and selects the operation. Never authorize from the client-controlled `operationName`, raw request text, or a second GraphQL parser.
- CE policy lives in `core/src/auth/graphql_authorization.go`; EE policy lives in `ee/core/src/auth/graphql_authorization.go` and is installed through `app.AppConfig.GraphQLAuthorization`.
- Policies inspect gqlgen-collected root fields and reject the entire operation before resolvers run when any selected field is protected. This must remain true for aliases, fragments, directives, mixed public/protected selections, mutations, and subscriptions.
- New root fields are protected by default. Add a field to an edition's public-field policy only when unauthenticated access is an explicit product requirement. Do not add operation-name allowlists.
- HTTP authentication middleware may extract identity/session state and defer GraphQL CSRF results, but it must not parse the GraphQL body to make authorization decisions.
- Return stable GraphQL error extension codes: `UNAUTHENTICATED`, `FORBIDDEN`, or `SERVICE_UNAVAILABLE`. The shared frontend session recovery handles `UNAUTHENTICATED` as well as legacy HTTP 401 responses.
- Security regressions need a live-server test that sends a protected field under a public-looking operation name and proves the resolver or mutation did not run.
