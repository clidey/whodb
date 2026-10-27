# Router Telemetry

Configure logging, metrics, and tracing with OpenTelemetry-compatible exporters.

## Telemetry Overview

The Router supports three types of telemetry:

| Type | Description | Exporters |
|------|-------------|-----------|
| **Logs** | Structured event logging | stdout, file |
| **Metrics** | Numerical measurements | Prometheus, OTLP, Datadog |
| **Traces** | Distributed request tracing | Jaeger, Zipkin, OTLP, Datadog |

## Basic Configuration

```yaml
telemetry:
  exporters:
    logging:
      stdout:
        enabled: true
        format: json

    metrics:
      prometheus:
        enabled: true
        listen: 127.0.0.1:9090
        path: /metrics

    tracing:
      otlp:
        enabled: true
        endpoint: http://collector:4317
```

## Logging

### Stdout Logging

```yaml
telemetry:
  exporters:
    logging:
      stdout:
        enabled: true
        format: json  # or "text" for human-readable
```

### Log Level

Set via environment variable:
```bash
APOLLO_ROUTER_LOG=debug router
```

Or in configuration:
```yaml
telemetry:
  exporters:
    logging:
      stdout:
        enabled: true
      common:
        service_name: my-router
```

Log levels: `off`, `error`, `warn`, `info`, `debug`, `trace`

## Metrics

### Prometheus

```yaml
telemetry:
  exporters:
    metrics:
      prometheus:
        enabled: true
        listen: 127.0.0.1:9090
        path: /metrics
```

Access metrics at `http://localhost:9090/metrics`.

Common metrics:
- `apollo_router_http_requests_total` - Request count
- `apollo_router_http_request_duration_seconds` - Latency histogram
- `apollo_router_cache_hit_count` - Cache hit/miss

### Response Cache Metrics (v2.6.0+)

| Metric | Description |
|--------|-------------|
| `apollo.router.operations.response_cache.fetch` | Time to fetch from cache |
| `apollo.router.operations.response_cache.insert` | Time to insert into cache |
| `apollo.router.operations.response_cache.invalidation.entry` | Entries invalidated |
| `apollo.router.cache.redis.commands_executed` | Total Redis commands |
| `apollo.router.cache.redis.errors` | Redis errors by type |

Full metrics reference and telemetry config: [response-caching.md](response-caching.md#observability)

### OTLP Metrics

```yaml
telemetry:
  exporters:
    metrics:
      otlp:
        enabled: true
        endpoint: http://collector:4317
        protocol: grpc  # or "http"
```

### Datadog Metrics

```yaml
telemetry:
  exporters:
    metrics:
      datadog:
        enabled: true
        endpoint: https://api.datadoghq.com
```

Set `DD_API_KEY` environment variable.

## Tracing

### OTLP (OpenTelemetry)

```yaml
telemetry:
  exporters:
    tracing:
      otlp:
        enabled: true
        endpoint: http://collector:4317
        protocol: grpc

      common:
        service_name: apollo-router
```

### Jaeger

```yaml
telemetry:
  exporters:
    tracing:
      jaeger:
        enabled: true
        agent:
          endpoint: localhost:6831
```

Or via collector:
```yaml
telemetry:
  exporters:
    tracing:
      jaeger:
        enabled: true
        collector:
          endpoint: http://jaeger:14268/api/traces
```

### Zipkin

```yaml
telemetry:
  exporters:
    tracing:
      zipkin:
        enabled: true
        endpoint: http://zipkin:9411/api/v2/spans
```

### Datadog

```yaml
telemetry:
  exporters:
    tracing:
      datadog:
        enabled: true
        endpoint: http://localhost:8126
```

## Sampling

Control trace sampling rate:

```yaml
telemetry:
  exporters:
    tracing:
      common:
        sampler: 0.5  # Sample 50% of requests

      # Or always sample
      # sampler: always_on

      # Or never sample
      # sampler: always_off
```

## Custom Attributes

Add custom attributes to spans:

```yaml
telemetry:
  instrumentation:
    spans:
      router:
        attributes:
          # Static attribute
          environment:
            static_value: production

          # From request header
          client_id:
            request_header: x-client-id

          # From response header
          cache_status:
            response_header: x-cache-status
```

## Trace Context Propagation

```yaml
telemetry:
  exporters:
    tracing:
      propagation:
        # W3C Trace Context (default)
        trace_context: true

        # Jaeger propagation
        jaeger: true

        # Zipkin B3
        zipkin: true

        # Datadog
        datadog: true
```

## GraphOS Studio

Send telemetry to GraphOS Studio:

```yaml
telemetry:
  apollo:
    client_name_header: apollographql-client-name
    client_version_header: apollographql-client-version
```

Requires `APOLLO_KEY` and `APOLLO_GRAPH_REF` environment variables.

## Complete Example

```yaml
telemetry:
  apollo:
    client_name_header: apollographql-client-name
    client_version_header: apollographql-client-version

  exporters:
    logging:
      stdout:
        enabled: true
        format: json

    metrics:
      prometheus:
        enabled: true
        listen: 0.0.0.0:9090
        path: /metrics

    tracing:
      otlp:
        enabled: true
        endpoint: http://otel-collector:4317
        protocol: grpc

      common:
        service_name: apollo-router
        sampler: 0.1  # 10% sampling

      propagation:
        trace_context: true

  instrumentation:
    spans:
      router:
        attributes:
          environment:
            static_value: ${env.ENVIRONMENT:-development}
          client:
            request_header: x-client-id
```

## Docker Compose Example

```yaml
version: "3.8"
services:
  router:
    image: ghcr.io/apollographql/router:latest
    ports:
      - "4000:4000"
      - "9090:9090"
    environment:
      - APOLLO_ROUTER_LOG=info
    volumes:
      - ./router.yaml:/etc/router/router.yaml
      - ./supergraph.graphql:/etc/router/supergraph.graphql

  prometheus:
    image: prom/prometheus
    ports:
      - "9091:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml

  jaeger:
    image: jaegertracing/all-in-one
    ports:
      - "16686:16686"
      - "6831:6831/udp"
```
