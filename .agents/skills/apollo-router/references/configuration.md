# Router Configuration Reference

The Router is configured via a YAML file (`router.yaml`). This reference covers the most common configuration options.

## Basic Structure (v2 default)

```yaml
supergraph:
  listen: 127.0.0.1:4000
  introspection: true
  path: /graphql

sandbox:
  enabled: true

homepage:
  enabled: false

cors:
  allow_any_origin: true  # development only

headers:
  all:
    request:
      - propagate:
          matching: ".*"

telemetry:
  # ... telemetry config
```

## Basic Structure (v1 legacy)

```yaml
cors:
  origins:
    - "*"
```

## Supergraph Configuration

```yaml
supergraph:
  # Address to listen on
  listen: 127.0.0.1:4000

  # GraphQL endpoint path
  path: /graphql

  # Enable introspection queries
  introspection: true

  # Query planning options
  query_planning:
    # Enable query plan caching
    cache:
      in_memory:
        limit: 512
```

## Sandbox and Introspection

```yaml
# Apollo Sandbox (GraphQL IDE)
sandbox:
  enabled: true  # Disabled by default in production

# Introspection (required for Sandbox)
supergraph:
  introspection: true  # Disabled by default in production
```

For development mode, both are enabled automatically with `--dev`.

## CORS Configuration

> **v1 vs v2**: CORS schemas are incompatible. See [divergence-map.md](../divergence-map.md) for details.

### v1 (flat schema)

```yaml
cors:
  origins:
    - http://localhost:3000
    - https://studio.apollographql.com
  allow_headers:
    - Content-Type
    - Authorization
  methods:
    - GET
    - POST
    - OPTIONS
  allow_credentials: true
  max_age: 24h  # duration string
```

### v2 (policies schema)

```yaml
cors:
  allow_credentials: true
  methods:
    - GET
    - POST
    - OPTIONS
  max_age: 24h  # duration string, not integer
  policies:
    - origins:
        - http://localhost:3000
        - https://studio.apollographql.com
      allow_headers:
        - Content-Type
        - Authorization
```

## Subgraph Configuration

```yaml
# Override subgraph URLs (useful for local development)
override_subgraph_url:
  products: http://localhost:4001/graphql
  reviews: http://localhost:4002/graphql

# Subgraph-specific settings
traffic_shaping:
  all:
    timeout: 30s
  subgraphs:
    products:
      timeout: 60s  # Override for slow subgraph
```

## Traffic Shaping

```yaml
traffic_shaping:
  # Apply to all subgraphs
  all:
    # Request timeout
    timeout: 30s

    # Rate limiting
    global_rate_limit:
      capacity: 1000
      interval: 1s

  # Router-level settings
  router:
    timeout: 60s

  # Per-subgraph settings
  subgraphs:
    slow-service:
      timeout: 120s
```

## Authentication (JWT)

> **v1 vs v2**: The `issuer` field was renamed to `issuers` (plural array) in v2. `issuer` is v1-only.

### v1

```yaml
authentication:
  router:
    jwt:
      jwks:
        - url: https://auth.example.com/.well-known/jwks.json
          issuer: https://auth.example.com/  # singular string

authorization:
  require_authentication: true
```

### v2

```yaml
authentication:
  router:
    jwt:
      jwks:
        - url: https://auth.example.com/.well-known/jwks.json
          issuers:                              # plural array
            - https://auth.example.com/

authorization:
  require_authentication: true
```

## Authorization Directives (field-level)

`authorization.require_authentication` above is an all-or-nothing gate on the whole request. For **field- and type-level** access control, use the declarative authorization directives — `@authenticated`, `@requiresScopes`, and `@policy` — applied in your **subgraph schemas**. The router filters out unauthorized fields before query planning and returns `UNAUTHORIZED_FIELD_OR_TYPE` errors for them.

> **GraphOS feature.** Requires a router connected to GraphOS — Router v1.29.1+; on Developer/Standard plans, Router v2.6.0+. The directives need a **claims source**: configure JWT authentication (claims land at the `apollo::authentication::jwt_claims` context key) or inject claims with a coprocessor.

> **Enabled by default.** The directives are on as soon as your router is GraphOS-connected and your schema uses them. Config only *disables* them:

```yaml
authorization:
  directives:
    enabled: false   # default is true — only set this to turn directives OFF
```

The directives themselves live in subgraph schemas, imported via `@link`:

```graphql
extend schema
  @link(url: "https://specs.apollo.dev/federation/v2.6",
        import: ["@authenticated", "@requiresScopes", "@policy"])

type Query {
  me: User @authenticated
  users: [User!]! @requiresScopes(scopes: [["read:others"]])
}

type User {
  id: ID!
  email: String @requiresScopes(scopes: [["read:email"]])
  creditCard: String @policy(policies: [["read_credit_card"]])
}
```

- `@authenticated` — field requires any valid identity (claims present).
- `@requiresScopes(scopes: [[...]])` — requires specific scopes. Inner array = **AND**, outer array = **OR**. Reads the `scope` key (space-separated string) from the claims object.
- `@policy(policies: [[...]])` — custom authorization. Requires a **Supergraph plugin** (Rhai script or coprocessor): the router populates `apollo::authorization::required_policies` as a `policy -> null` map, and your plugin sets each to `true`/`false`. Unset (`null`) is treated as `false`.

> **Context key differs by version** (relevant if a coprocessor reads claims): v2 uses `apollo::authentication::jwt_claims`; v1 used `apollo_authentication::JWT::claims`. See [divergence-map.md](../divergence-map.md).

## Response Caching (v2.6.0+)

> Response caching uses the `response_cache` top-level key (not `supergraph.cache`).
> See [response-caching.md](response-caching.md) for full setup, schema directives, invalidation, and observability.

```yaml
# Minimal response caching setup
response_cache:
  enabled: true
  subgraph:
    all:
      enabled: true
      ttl: 5m
      redis:
        urls: ["redis://localhost:6379"]
```

## Automatic Persisted Queries (APQ)

> **Performance only — not a security control.** APQ lets clients send the SHA-256 hash of an operation instead of the full string to save bandwidth. The router caches **any** operation it receives at runtime, so APQ does **not** restrict which operations can run. For an operation allowlist, use [Persisted Query Safelisting](#persisted-query-safelisting-pql) below — and note the two are mutually exclusive.

```yaml
apq:
  enabled: true
  router:
    cache:
      in_memory:
        limit: 512
      # Or Redis
      # redis:
      #   urls:
      #     - redis://localhost:6379
```

## Persisted Query Safelisting (PQL)

> **Security control, distinct from APQ.** Clients register trusted operations to a GraphOS-managed **Persisted Query List (PQL)** at build time (via `rover persisted-queries publish` in their CI/CD). The router fetches the PQL on startup and can **reject any operation not on the list**. Requires a router connected to GraphOS (`APOLLO_KEY` + `APOLLO_GRAPH_REF`), or `local_manifests` for offline licenses.
>
> **Config key:** GA `persisted_queries` since v1.32.0 (was `preview_persisted_queries` in v1.25.0–v1.32.0); GA in all v2.

Adopt incrementally — start in audit mode, then enforce once you confirm all clients are registered.

**Audit mode** (logs unregistered operations, rejects nothing):

```yaml
persisted_queries:
  enabled: true
  log_unknown: true
```

**Safelisting** (rejects unregistered operations; registered IDs *and* full strings both accepted):

```yaml
persisted_queries:
  enabled: true
  safelist:
    enabled: true
apq:
  enabled: false   # REQUIRED: APQ and safelisting are mutually exclusive
```

**Safelisting, IDs only** (also rejects freeform operation strings, even registered ones):

```yaml
persisted_queries:
  enabled: true
  safelist:
    enabled: true
    require_id: true
apq:
  enabled: false
```

**Offline / air-gapped** (use a local manifest instead of fetching from GraphOS Uplink):

```yaml
persisted_queries:
  enabled: true
  local_manifests:
    - ./persisted-query-manifest.json
  hot_reload: true   # optional: reload the manifest file on change (local_manifests only)
```

You can opt individual requests out of enforcement from a Rhai script or coprocessor by setting the `apollo_persisted_queries::safelist::skip_enforcement` context key to `true`.

## Limits and Security

```yaml
limits:
  # Maximum request body size
  http_max_request_bytes: 2000000  # 2MB

  # Query complexity limits
  max_depth: 15
  max_height: 200
  max_aliases: 30
  max_root_fields: 20
```

## Include Subgraph Errors

```yaml
# Control which subgraph errors are exposed to clients
include_subgraph_errors:
  all: true  # Include all subgraph errors (development)

  # Or selectively
  # all: false
  # subgraphs:
  #   products: true  # Only expose products errors
```

## Subscriptions

```yaml
subscription:
  enabled: true
  mode:
    # WebSocket-based subscriptions
    passthrough:
      all:
        path: /ws

    # Or callback-based
    # callback:
    #   public_url: https://router.example.com/callback
```

## Development vs Production

### Development Configuration

```yaml
# router.dev.yaml
supergraph:
  introspection: true

sandbox:
  enabled: true

include_subgraph_errors:
  all: true

telemetry:
  exporters:
    logging:
      stdout:
        enabled: true
        format: text
```

### Production Configuration

```yaml
# router.prod.yaml
supergraph:
  listen: 0.0.0.0:4000
  introspection: false

sandbox:
  enabled: false

homepage:
  enabled: false

health_check:
  enabled: true
  listen: 0.0.0.0:8088
  path: /health

include_subgraph_errors:
  all: false

# CORS — use v1 or v2 format as appropriate (see CORS section above)
cors:
  origins:
    - https://app.example.com

telemetry:
  exporters:
    tracing:
      otlp:
        enabled: true
        endpoint: http://collector:4317
```

For complete production templates with all features, see:
- [templates/v1/production.yaml](../templates/v1/production.yaml)
- [templates/v2/production.yaml](../templates/v2/production.yaml)

## Environment Variable Expansion

Use environment variables in configuration:

```yaml
supergraph:
  listen: ${env.ROUTER_LISTEN_ADDRESS:-127.0.0.1:4000}

override_subgraph_url:
  products: ${env.PRODUCTS_URL}

authentication:
  router:
    jwt:
      jwks:
        - url: ${env.JWKS_URL}
```

## Configuration Validation

Validate configuration without starting the Router:

```bash
router config validate router.yaml
```
