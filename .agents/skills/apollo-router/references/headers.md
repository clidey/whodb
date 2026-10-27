# Header Configuration

Configure how the Router handles HTTP headers for requests to subgraphs and responses to clients.

## Header Propagation

Pass headers from client requests to subgraph requests.

### Propagate All Headers

```yaml
headers:
  all:
    request:
      - propagate:
          matching: ".*"  # Regex pattern
```

### Propagate Specific Headers

```yaml
headers:
  all:
    request:
      # Propagate by exact name
      - propagate:
          named: Authorization

      # Propagate by pattern
      - propagate:
          matching: "^x-.*"  # All x-* headers

      # Rename while propagating
      - propagate:
          named: Authorization
          rename: X-Auth-Token
```

### Per-Subgraph Headers

```yaml
headers:
  # Default for all subgraphs
  all:
    request:
      - propagate:
          named: Authorization

  # Override for specific subgraph
  subgraphs:
    products:
      request:
        - propagate:
            named: Authorization
        - propagate:
            named: X-Products-Key
```

## Inserting Headers

Add static or dynamic headers to subgraph requests.

### Static Headers

```yaml
headers:
  all:
    request:
      - insert:
          name: X-Router-Version
          value: "1.0"

      - insert:
          name: X-Api-Key
          value: ${env.API_KEY}  # From environment
```

### Dynamic Headers from Context

```yaml
headers:
  all:
    request:
      # Insert from request context
      - insert:
          name: X-Request-Id
          from_context: request_id

      # Insert from response context (for response headers)
      - insert:
          name: X-Trace-Id
          from_context: apollo_telemetry::trace_id
```

## Removing Headers

Remove headers before sending to subgraphs or clients.

```yaml
headers:
  all:
    request:
      # Remove specific header
      - remove:
          named: Cookie

      # Remove by pattern
      - remove:
          matching: "^x-internal-.*"
```

## Response Headers

Configure headers sent back to clients.

```yaml
headers:
  all:
    # Response headers to clients
    response:
      # Propagate from subgraph response
      - propagate:
          named: X-Cache-Status

      # Insert static header
      - insert:
          name: X-Powered-By
          value: "Apollo Router"

      # Remove sensitive headers
      - remove:
          named: X-Internal-Debug
```

## Default Headers

Headers sent to subgraphs by default:

| Header | Description |
|--------|-------------|
| `Content-Type` | `application/json` |
| `Accept` | `application/json` |
| `apollographql-client-name` | Client name (if provided) |
| `apollographql-client-version` | Client version (if provided) |

## Complete Example

```yaml
headers:
  all:
    request:
      # Propagate auth
      - propagate:
          named: Authorization

      # Propagate custom headers
      - propagate:
          matching: "^x-custom-.*"

      # Add router metadata
      - insert:
          name: X-Router-Request-Id
          from_context: request_id

      # Remove cookies (not needed by subgraphs)
      - remove:
          named: Cookie

    response:
      # Add cache headers
      - insert:
          name: Cache-Control
          value: "private, max-age=60"

      # Propagate trace ID
      - propagate:
          named: X-Trace-Id

  subgraphs:
    products:
      request:
        # Additional header for products
        - insert:
            name: X-Products-Version
            value: "v2"

    legacy-service:
      request:
        # Rename header for legacy service
        - propagate:
            named: Authorization
            rename: X-Legacy-Auth
```

## Header Order

Operations execute in order. Later operations can override earlier ones:

```yaml
headers:
  all:
    request:
      # First: propagate all
      - propagate:
          matching: ".*"
      # Then: remove sensitive ones
      - remove:
          matching: "^x-internal-.*"
      # Finally: add new ones
      - insert:
          name: X-Router
          value: "true"
```

## Environment Variable Expansion

```yaml
headers:
  all:
    request:
      - insert:
          name: X-Api-Key
          value: ${env.API_KEY}

      - insert:
          name: X-Environment
          value: ${env.ENVIRONMENT:-development}  # With default
```

## Common Patterns

### Authentication Propagation

```yaml
headers:
  all:
    request:
      - propagate:
          named: Authorization
      - propagate:
          named: Cookie
```

### Request Tracing

```yaml
headers:
  all:
    request:
      - propagate:
          named: X-Request-Id
      - propagate:
          named: X-Correlation-Id
      - insert:
          name: X-Router-Trace
          from_context: apollo_telemetry::trace_id
```

### Multi-Tenant Headers

```yaml
headers:
  all:
    request:
      - propagate:
          named: X-Tenant-Id
      - propagate:
          named: X-Organization-Id
```
