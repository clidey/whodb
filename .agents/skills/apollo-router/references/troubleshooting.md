# Router Troubleshooting

Common issues and solutions when running Apollo Router.

## Startup Issues

### Router Fails to Start

**Error: No supergraph schema provided**

```
Error: No supergraph schema found. Provide --supergraph or set APOLLO_GRAPH_REF.
```

Fix: Provide a supergraph schema:
```bash
# Local file
router --supergraph ./supergraph.graphql

# Or GraphOS managed
export APOLLO_KEY=service:my-graph:key
export APOLLO_GRAPH_REF=my-graph@production
router
```

**Error: Invalid configuration**

```
Error: configuration error: unknown field `cors`
```

Fix: Check YAML syntax and field names:
```bash
# Validate config
router config validate router.yaml
```

### Port Already in Use

```
Error: Address already in use (os error 48)
```

Fix: Use a different port or stop the existing process:
```bash
# Check what's using the port
lsof -i :4000

# Use different port
router --supergraph ./supergraph.graphql --listen 127.0.0.1:4001
```

## Connection Issues

### Cannot Connect to Subgraphs

**Error: Connection refused**

```
Error: error sending request for url (http://localhost:4001/graphql): error trying to connect
```

Checklist:
1. Verify subgraph is running: `curl -X POST http://localhost:4001/graphql -H "Content-Type: application/json" -d '{"query":"{ __typename }"}'`
2. Check URL in supergraph schema is correct
3. For Docker, use host.docker.internal or service names

Override subgraph URL for local development:
```yaml
# router.yaml
override_subgraph_url:
  products: http://host.docker.internal:4001/graphql
```

**Error: Timeout**

```
Error: operation timed out
```

Increase timeout:
```yaml
traffic_shaping:
  subgraphs:
    slow-service:
      timeout: 60s
```

### GraphOS Connection Issues

**Error: Failed to fetch schema from Uplink**

```
Error: failed to fetch schema: Uplink request failed
```

Checklist:
1. Verify `APOLLO_KEY` is correct
2. Verify `APOLLO_GRAPH_REF` format: `graph-id@variant`
3. Check network connectivity to Apollo

```bash
# Test connection
curl -H "X-Api-Key: $APOLLO_KEY" \
  https://uplink.api.apollographql.com/
```

## Query Issues

### Introspection Not Working

**Error: Introspection disabled**

```json
{
  "errors": [{ "message": "Introspection has been disabled" }]
}
```

Fix: Enable introspection (development only):
```yaml
supergraph:
  introspection: true
```

Or use `--dev` mode:
```bash
router --dev --supergraph ./supergraph.graphql
```

### Sandbox Not Loading

Sandbox requires introspection. Enable both:
```yaml
supergraph:
  introspection: true

sandbox:
  enabled: true
```

Or use `--dev` mode which enables both.

### CORS Errors

**Error: Browser blocked by CORS policy**

```
Access to fetch has been blocked by CORS policy: No 'Access-Control-Allow-Origin' header
```

Fix: Configure CORS using the correct schema for your Router version:

**v1 (flat schema):**
```yaml
cors:
  origins:
    - http://localhost:3000
    - https://studio.apollographql.com
  allow_headers:
    - Content-Type
    - Authorization
```

**v2 (policies schema):**
```yaml
cors:
  max_age: 24h
  policies:
    - origins:
        - http://localhost:3000
        - https://studio.apollographql.com
      allow_headers:
        - Content-Type
        - Authorization
```

For development (not production):
```yaml
# v1
cors:
  origins:
    - "*"

# v2
cors:
  allow_any_origin: true
```

**CORS config ignored after upgrading to v2:**
If you upgraded from v1 to v2 and your CORS settings stopped working, you're
likely using the v1 flat schema (`cors.origins`). v2 requires the policies
array format (`cors.policies`). See the [divergence map](../divergence-map.md)
for the full diff. You can auto-migrate with: `router config upgrade router.yaml`

### JWT Issuer Field Mismatch (v2)

If you migrated from v1 to v2 and kept the singular `issuer` field instead
of the plural `issuers` array, the config uses a v1-only field. Use `issuers`
for Router v2.

**Broken (v1 field in v2 config):**
```yaml
authentication:
  router:
    jwt:
      jwks:
        - url: https://auth.example.com/.well-known/jwks.json
          issuer: https://auth.example.com/   # WRONG for v2!
```

**Fixed (v2 field):**
```yaml
authentication:
  router:
    jwt:
      jwks:
        - url: https://auth.example.com/.well-known/jwks.json
          issuers:                              # Correct for v2
            - https://auth.example.com/
```

## Performance Issues

### Slow Queries

Debug steps:

1. **Enable query plan logs:**
```yaml
telemetry:
  exporters:
    logging:
      stdout:
        enabled: true
        format: json
```

2. **Check subgraph latency:**
Enable tracing to identify slow subgraphs.

3. **Enable caching:**
```yaml
supergraph:
  query_planning:
    cache:
      in_memory:
        limit: 512
```

### High Memory Usage

1. **Limit cache sizes:**
```yaml
supergraph:
  query_planning:
    cache:
      in_memory:
        limit: 256  # Reduce from default
```

2. **Check for complex queries** that generate large query plans.

### Response Cache Misses

1. **Check subgraph Cache-Control headers**: origin must return `Cache-Control` without `no-store`
2. **Verify Cache-Control headers**: subgraph responses need `Cache-Control: max-age=N` (via `@cacheControl` in Apollo Server, or set directly in other frameworks)
3. **Check scope**: `Cache-Control: private` requires `private_id` to be configured on the router
4. **Verify Redis connectivity**: check `apollo.router.cache.redis.errors` metric
5. **Enable cache debugger** (dev only): set `response_cache.debug: true` and use Apollo Sandbox to inspect cache state

## Federation Issues

### Composition Errors at Runtime

Router logs composition errors:
```
Error: Subgraph schema validation failed
```

Fix: Validate schema before deploying:
```bash
rover subgraph check my-graph@production \
  --name products \
  --schema ./schema.graphql
```

### Entity Resolution Failures

**Error: Cannot resolve entity**

```json
{
  "errors": [{ "message": "cannot resolve entity of type User" }]
}
```

Checklist:
1. Subgraph implements `_entities` query
2. Entity's `@key` fields match between subgraphs
3. Reference resolver returns correct data format

## Health Check

Check Router health on the configured health listener (the provided templates use `127.0.0.1:8088/health`):
```bash
curl http://localhost:8088/health
```

Expected response:
```json
{"status":"healthy"}
```

If you are not using an explicit `health_check` config, Router also exposes:
```bash
curl http://localhost:4000/.well-known/apollo/server-health
```

For Kubernetes readiness probe (matching this skill's templates):
```yaml
readinessProbe:
  httpGet:
    path: /health
    port: 8088
  initialDelaySeconds: 5
  periodSeconds: 10
```

## Debug Mode

Get more verbose output:
```bash
APOLLO_ROUTER_LOG=debug router --supergraph ./supergraph.graphql
```

Log levels:
- `error` - Errors only
- `warn` - Warnings and errors
- `info` - General information (default)
- `debug` - Detailed debug information
- `trace` - Very verbose tracing

## Common Mistakes

| Mistake | Solution |
|---------|----------|
| Introspection disabled in dev | Use `--dev` flag |
| Wrong subgraph URL in production | Use `override_subgraph_url` |
| CORS not configured | Add allowed origins |
| Timeout too short | Increase `traffic_shaping.timeout` |
| Missing APOLLO_KEY | Set environment variable |
| Wrong graph ref format | Use `graph-id@variant` |

## Getting Help

1. Check [Router documentation](https://www.apollographql.com/docs/router/)
2. Search [Apollo Community](https://community.apollographql.com/)
3. Check [GitHub Issues](https://github.com/apollographql/router/issues)
4. Enable debug logging for detailed error information
