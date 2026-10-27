# Post-Generation Validation Checklist

Run through this checklist after generating a `router.yaml` to catch common mistakes.

## Security

- [ ] **Introspection disabled** (production): `supergraph.introspection: false`
- [ ] **Sandbox disabled** (production): `sandbox.enabled: false`
- [ ] **Homepage disabled** (production): `homepage.enabled: false`
- [ ] **Subgraph errors hidden** (production): `include_subgraph_errors.all: false`
- [ ] **No wildcard CORS** (production): explicit origins only (no `"*"` and no `allow_any_origin: true`)
- [ ] **Authentication required** (if applicable): `authorization.require_authentication: true`
- [ ] **Declarative authorization not silently disabled** (if using `@authenticated`/`@requiresScopes`/`@policy`): `authorization.directives.enabled` is absent or `true`, router is GraphOS-connected, and a claims source (JWT auth or coprocessor) is configured
- [ ] **`@policy` evaluator wired** (if `@policy` used): a Rhai script or coprocessor evaluates `apollo::authorization::required_policies` at the Supergraph stage
- [ ] **Safelisting vs APQ not confused**: if the goal is an operation allowlist, `persisted_queries.safelist.enabled: true` is set (NOT just `apq`)
- [ ] **APQ disabled when safelisting** (if `persisted_queries.safelist.enabled: true`): `apq.enabled: false` (mutually exclusive)
- [ ] **Persisted queries use GA key**: `persisted_queries` (not `preview_persisted_queries`)

## Version Correctness

- [ ] **CORS schema matches version**:
  - v1: flat `cors.origins: [...]`
  - v2: `cors.policies: [{ origins: [...] }]`
- [ ] **JWT issuer field matches version**:
  - v1: `issuer: <string>` (singular)
  - v2: `issuers: [<string>]` (plural array)
- [ ] **Max age format matches version**:
  - v1: duration string (`max_age: 24h`)
  - v2: duration string (`max_age: 24h`)
- [ ] **Limits key is correct**: Use `limits` (v1.17+ and v2), not `preview_operation_limits`
- [ ] **Connectors key is correct** (if applicable): early v2 preview = `preview_connectors`, current v2 GA = `connectors`

## Operational

- [ ] **Health check enabled**: `health_check.enabled: true` with a listen address
- [ ] **Rate limiting on `router:`**: `traffic_shaping.router.global_rate_limit` limits client requests; `all:` limits subgraph requests
- [ ] **All env vars documented**: Every `${env.VAR}` in the config has a corresponding entry in deployment docs
- [ ] **APOLLO_KEY not hardcoded**: API key is via environment variable, never in config file or logs
- [ ] **Secrets use env var expansion**: `${env.JWKS_URL}`, `${env.JWT_ISSUER}`, etc.
- [ ] **Config is version-controlled**: `router.yaml` is committed to git (safe to share — no secrets) and changes go through review
- [ ] **CI validates config**: `router config validate router.yaml` runs on every PR, pinned to the deployed Router version

## Telemetry

- [ ] **Logging format is JSON** (production): `telemetry.exporters.logging.stdout.format: json`
- [ ] **Tracing sampler is set**: `sampler: 0.1` (10%) is a reasonable default; tune for your traffic volume
- [ ] **Service name is set**: `common.service_name` identifies this router instance

## Response Caching (if enabled)

### Security

- [ ] **User-specific fields identified**: Confirmed with the user which fields return per-user data
- [ ] **Private scope set**: User-specific subgraph responses include `Cache-Control: private` (via `@cacheControl(scope: PRIVATE)` in Apollo Server, or by setting the header directly in other frameworks)
- [ ] **private_id configured**: Every subgraph serving private-scoped data has `private_id` set
- [ ] **User identifier extracted**: Rhai script or coprocessor populates the `private_id` context key from auth token
- [ ] **No unprotected user data**: Verified that no user-specific response is cached without `Cache-Control: private`
- [ ] **Debug mode disabled** (production): `response_cache.debug` is absent or `false`
- [ ] **Invalidation endpoint not publicly exposed** (production): bind to `127.0.0.1`, not `0.0.0.0`
- [ ] **Invalidation shared key uses env var**: `${env.INVALIDATION_SHARED_KEY}`

### Operational

- [ ] **Redis URL uses env var**: `${env.CACHE_REDIS_URL}`, not hardcoded
- [ ] **TTL is set**: explicit `ttl` on `subgraph.all` or per-subgraph
- [ ] **Redis credentials use env vars**: No hardcoded passwords or usernames

## Recommended Final Step

```bash
# Validate config syntax against the Router's schema
router config validate router.yaml
```

If migrating from v1 to v2, use the built-in upgrade tool:
```bash
router config upgrade router.yaml
```
