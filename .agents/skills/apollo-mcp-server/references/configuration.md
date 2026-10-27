# Apollo MCP Server Configuration Reference

Apollo MCP Server uses YAML configuration. The server watches the config file and auto-restarts on changes (in-place for `streamable_http`; exit code 75 for `stdio`). Changes to `logging` and `telemetry` require a full process restart.

## Top-Level Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `endpoint` | `URL` | `http://localhost:4000/` | Target GraphQL endpoint |
| `schema` | `SchemaSource` | | Schema source config |
| `operations` | `OperationSource` | | Operations config |
| `transport` | `Transport` | | Transport config |
| `headers` | `Map<string, string>` | `{}` | Static headers for all GraphQL requests |
| `forward_headers` | `List<string>` | `[]` | Headers to forward from MCP clients |
| `introspection` | `Introspection` | | Introspection tool config |
| `overrides` | `Overrides` | | Mutation mode, descriptions, annotations, scopes |
| `graphos` | `GraphOS` | | GraphOS credentials (`apollo_key`, `apollo_graph_ref`) |
| `cors` | `Cors` | | CORS config |
| `health_check` | `HealthCheck` | | Health check config |
| `logging` | `Logging` | | Logging config |
| `telemetry` | `Telemetry` | | OTLP metrics/traces export |
| `custom_scalars` | `FilePath` | | Path to custom scalar map |
| `server_info` | `ServerInfo` | | Server metadata for MCP `initialize` response |
| `instructions` | `string` | | MCP `initialize` instructions text |
| `rhai` | `Rhai` | | Rhai scripting config (`scripts` directory, default `rhai`) |
| `caching` | `Caching` | | `ttl_ms` for MCP response cache hints (default `300000`, protocol `2026-07-28`+) |

## Schema

| Source | Config | Description |
|--------|--------|-------------|
| Local | `source: local`, `path: ./schema.graphql` | Load from local file |
| Uplink | `source: uplink` (default) | Fetch from uplink (federated graphs, requires GraphOS creds) |
| GraphOS | `source: graphos` | Fetch from Platform API (non-federated graphs, requires GraphOS creds) |

## Operations

| Source | Config | Description |
|--------|--------|-------------|
| `infer` | `source: infer` (default) | Auto-discover from other config |
| `local` | `source: local`, `paths: [...]` | Load from `.graphql` files/directories |
| `collection` | `source: collection`, `id: ...` | Load from GraphOS operation collection |
| `manifest` | `source: manifest`, `path: ...` | Load from persisted query manifest |
| `uplink` | `source: uplink` | Load from GraphOS persisted queries |
| `introspect` | `source: introspect` | Derive from schema introspection |

For `collection` and `uplink`, the server polls for changes (up to 60s delay). Collection polling supports up to 100 operations.

## Transport

### Stdio (Default)

```yaml
transport:
  type: stdio
```

### Streamable HTTP

| Option | Default | Description |
|--------|---------|-------------|
| `address` | `127.0.0.1` | Bind address |
| `port` | `8000` | Listen port |
| `stateful_mode` | `true` | Enable sessions for pre-`2026-07-28` clients |

SSE transport was removed in v1.5.0.

### Host Validation

| Option | Default | Description |
|--------|---------|-------------|
| `enabled` | `true` | Enable Host/Origin validation |
| `allowed_hosts` | `[]` | Additional hostnames (loopback always allowed) |
| `allowed_origins` | `[]` | Allowed browser origins (opt-in; empty = disabled) |

### Auth

OAuth 2.1 for streamable_http only. Nested under `transport.auth`. See [auth reference](auth.md).

## Introspection

All tools disabled by default. Each supports `enabled`, `minify` (where applicable), and `hint` (appends instructions to tool description).

```yaml
introspection:
  introspect:
    enabled: true
    minify: true
  search:
    enabled: true
    index_memory_bytes: 50000000
    leaf_depth: 1
  validate:
    enabled: true
  execute:
    enabled: true
```

## Overrides

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `mutation_mode` | `string` | `none` | `none`, `explicit`, or `all` |
| `disable_type_description` | `bool` | `false` | Omit type descriptions |
| `disable_schema_description` | `bool` | `false` | Omit schema descriptions |
| `enable_output_schema` | `bool` | `false` | Generate output schema for tools |
| `enable_explorer` | `bool` | `false` | Expose `explorer` tool (requires GraphOS graph ref) |
| `descriptions` | `Map<String, String>` | `{}` | Tool descriptions by operation name |
| `annotations` | `Map<String, Object>` | `{}` | Tool annotation hints (`title`, `read_only_hint`, `destructive_hint`, `idempotent_hint`, `open_world_hint`) |
| `required_scopes` | `Map<String, List>` | `{}` | Per-operation OAuth scope requirements |

### Mutation Modes

| Mode | Predefined mutations | Ad hoc mutations (`execute`) |
|------|---------------------|------------------------------|
| `none` | Skipped | Rejected |
| `explicit` | Exposed | Rejected |
| `all` | Exposed | Allowed |

`explicit` exposes predefined mutations from operation sources but still rejects ad-hoc mutations through `execute`. It is a configuration gate, not a per-invocation approval.

## Server Info

| Option | Default | Description |
|--------|---------|-------------|
| `name` | `"Apollo MCP Server"` | Technical identifier |
| `version` | crate version | Implementation version |
| `title` | `"Apollo MCP Server"` | Human-readable title |
| `website_url` | Apollo docs URL | Website link |
| `description` | default text | Brief description |
| `icons` | none | Icons for client UIs (`src`, `mime_type`, `sizes`, `theme`) |

## CORS

Disabled by default. Streamable HTTP only.

| Option | Default | Description |
|--------|---------|-------------|
| `enabled` | `false` | Enable CORS |
| `origins` | `[]` | Allowed origins (exact match) |
| `match_origins` | `[]` | Regex patterns for origins |
| `allow_any_origin` | `false` | Allow any origin. Can't be combined with `allow_credentials`; use `origins` or `match_origins` instead |
| `allow_credentials` | `false` | Allow credentials |
| `allow_methods` | `["GET", "POST", "DELETE"]` | Allowed HTTP methods |
| `allow_headers` | `["content-type", "mcp-protocol-version", "mcp-session-id", "traceparent", "tracestate", "baggage"]` | Allowed request headers |
| `expose_headers` | `["mcp-session-id", "traceparent", "tracestate"]` | Exposed response headers |
| `max_age` | `7200` | Preflight cache seconds |

Setting `allow_methods`, `allow_headers`, or `expose_headers` **replaces** defaults — repeat entries you need.

## Health Check

Disabled by default. Streamable HTTP only.

| Option | Default | Description |
|--------|---------|-------------|
| `enabled` | `false` | Enable health endpoints |
| `path` | `/health` | Endpoint path |
| `listen` | | Separate address/port (e.g., `0.0.0.0:8088`) |
| `readiness.allowed` | `100` | Max rejections before unready |
| `readiness.interval.sampling` | `5s` | Rejection check interval |
| `readiness.interval.unready` | `10s` | Recovery wait (default: 2 * sampling) |

Endpoints: `GET /health`, `GET /health?live`, `GET /health?ready`.

## Logging

| Option | Default | Description |
|--------|---------|-------------|
| `level` | `info` | `trace`, `debug`, `info`, `warn`, `error` |
| `path` | | Directory for rolling log files |
| `rotation` | `hourly` | `minutely`, `hourly`, `daily`, `never` |

## Environment Variables

### Config Expansion

Use `${env.VAR_NAME}` in YAML. Supports defaults: `${env.VAR:-fallback}`.

### Overrides

`APOLLO_MCP_` prefix with `__` nesting: `APOLLO_MCP_TRANSPORT__TYPE=streamable_http`.

`APOLLO_MCP_*` overrides take precedence over `${env.*}` expansion.
