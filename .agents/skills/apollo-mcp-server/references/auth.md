# Apollo MCP Server Authentication and Authorization

## Overview

Apollo MCP Server supports OAuth 2.1 authentication for `streamable_http` transport. Auth is configured under `transport.auth` (not at the top level). The `stdio` transport does not support authentication.

The server validates signed JWT bearer access tokens. Each JWT must contain a `kid` header, plus `exp` and `sub` claims. Opaque tokens are not supported.

## Configuration

```yaml
transport:
  type: streamable_http
  auth:
    servers:
      - https://auth.example.com
    audiences:
      - https://api.example.com
    issuers:
      - "https://auth.example.com"
    allow_any_audience: false
    resource: https://hosted.mcp.server/mcp
    resource_documentation: https://info.mcp.server
    scopes:
      - read
      - write
    scope_mode: require_all
    discovery_timeout: 5s
    discovery_headers:
      User-Agent: apollo-mcp-server
    disable_auth_token_passthrough: false
    tls:
      ca_cert: /path/to/ca-certificate.pem
      danger_accept_invalid_certs: false
```

## Auth Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `servers` | `List<URL>` | | Issuer URLs of upstream OAuth servers, not discovery-document URLs (must support RFC 8414 or OIDC discovery) |
| `audiences` | `List<string>` | `[]` | Accepted JWT audience values |
| `issuers` | `List<string>` | `[]` | Allowlist for token issuers. When set, `iss` must match this list and discovery metadata |
| `allow_any_audience` | `bool` | `false` | Skip audience validation entirely |
| `resource` | `string` | | Externally available URL of this MCP server |
| `resource_documentation` | `string` | | Link to documentation for this MCP server |
| `scopes` | `List<string>` | `[]` | Global OAuth scopes. Advertised and enforced per `scope_mode` |
| `scope_mode` | `string` | `require_all` | `require_all`, `require_any`, or `disabled` |
| `disable_auth_token_passthrough` | `bool` | `false` | Disable automatic propagation of validated token to downstream API |
| `discovery_timeout` | `Duration` | `5s` | Timeout for metadata discovery requests |
| `discovery_headers` | `Map<string, string>` | `{}` | Custom headers for OIDC discovery and JWKS requests |
| `tls.ca_cert` | `string` | | Path to CA certificate (PEM) for private CA or self-signed certs |
| `tls.danger_accept_invalid_certs` | `bool` | `false` | Accept invalid TLS certificates (development only) |

## Scope Enforcement

| Mode | Behavior |
|------|----------|
| `require_all` | Token must have all configured scopes (default) |
| `require_any` | Token must have at least one scope |
| `disabled` | Skip global scope check; per-operation requirements still apply |

## Per-Operation Scope Requirements

Declare additional scopes per tool using `overrides.required_scopes`:

```yaml
overrides:
  required_scopes:
    GetUser:
      - user:read
    DeleteUser:
      - user:write
      - admin
```

A flat list requires all scopes. Nested lists express alternatives (inner = AND, outer = OR):

```yaml
overrides:
  required_scopes:
    UpdateUser:
      - [user:write, tenant:admin]
      - [admin]
```

When a token lacks required scopes, the server returns `403 Forbidden` with a `WWW-Authenticate` header. Per-operation requirements add to (not replace) the global requirement.

## Skipping Token Validation

Allow specific requests to bypass token validation when no `Authorization` header is present:

```yaml
transport:
  type: streamable_http
  auth:
    servers:
      - https://auth.example.com
    skip_token_validation:
      methods:
        - initialize
        - notifications/initialized
        - tools/list
      tools:
        - SearchDocs
      headers:
        - x-api-key
```

| List | Matches on | Notes |
|------|-----------|-------|
| `methods` | JSON-RPC method name | `tools/call` cannot be listed here (rejected at startup) |
| `tools` | Tool name in `tools/call` | Specific tools reachable without a token |
| `headers` | HTTP header name | Skips validation for any method when header is present |

> **Security:** A request with a listed header skips token validation for all methods and tools. Ensure something downstream validates the header value. Also list the header in `forward_headers` so it reaches your API.

A request that carries an `Authorization` header is always validated, even if it matches a skip rule. An expired token on a public tool returns `401`, while no token succeeds.

## HTTP Error Responses

| Status | When |
|--------|------|
| `401 Unauthorized` | Token missing, malformed, expired, bad signature, or fails audience/issuer check |
| `403 Forbidden` | Token valid but lacks required scopes |

Both responses include a `WWW-Authenticate` header with `resource_metadata` URL and `scope` parameters.

## Discovery Compatibility

The server tries discovery URLs in order:

1. RFC 8414 path-insertion (`/.well-known/oauth-authorization-server/<path>`)
2. OIDC Discovery path-insertion (`/.well-known/openid-configuration/<path>`)
3. OIDC Discovery path-appending (`/<path>/.well-known/openid-configuration`)

The first parseable metadata document is used.

## JWKS Caching

- Cached per issuer, entries stay fresh for 10 minutes
- Known `kid` values served from stale cache during outages
- Per-issuer rate limiting prevents fabricated `kid` attacks
