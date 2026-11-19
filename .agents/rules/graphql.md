---
paths:
  - "core/graph/**"
  - "**/*.graphqls"
  - "**/*.graphql"
---

# GraphQL Rules

## Schema Changes
- Edit `core/graph/schema.graphqls`, then implement the generated stubs in `*.resolvers.go`. After schema edits the agent hook runs `go generate .` in `core/` and `pnpm run generate` in `frontend/`; outside the hook, run both by hand. Frontend codegen reads the schema file, so the backend does not need to be running.

## Source-First API
- New public queries/mutations use `Source*` types: `SourceTypes`, `SourceProfiles`, `SourceObjects`, `SourceRows`, `RunSourceQuery`, `SourceGraph`
- Do NOT add new `Database*` queries or capability surfaces

## Resolver Patterns
- Never import `graph` from `src/` (import cycle: `src → router → graph → src`)

## Operation Authorization

- GraphQL authentication is decided after gqlgen parses and selects the operation. Never authorize from the client-controlled `operationName`, raw request text, or a second GraphQL parser.
- The CE policy lives in `core/src/auth/graphql_authorization.go`. An entry point can install its own policy through `app.AppConfig.GraphQLAuthorization`.
- Policies inspect gqlgen-collected root fields and reject the entire operation before resolvers run when any selected field is protected. This must remain true for aliases, fragments, directives, mixed public/protected selections, mutations, and subscriptions.
- New root fields are protected by default. Add a field to the public-field policy only when unauthenticated access is an explicit product requirement. Do not add operation-name allowlists.
- HTTP authentication middleware may extract identity/session state and defer GraphQL CSRF results, but it must not parse the GraphQL body to make authorization decisions.
- Return stable GraphQL error extension codes: `UNAUTHENTICATED`, `FORBIDDEN`, or `SERVICE_UNAVAILABLE`. The shared frontend session recovery handles `UNAUTHENTICATED` as well as legacy HTTP 401 responses.
- Security regressions need a live-server test that sends a protected field under a public-looking operation name and proves the resolver or mutation did not run (CE example: `frontend/e2e/tests/features/graphql-authorization.ce.spec.mjs`).
