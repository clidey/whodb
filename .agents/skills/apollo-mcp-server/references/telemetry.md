# OpenTelemetry Integration

## Overview

Apollo MCP Server exports metrics and traces using the OpenTelemetry Protocol (OTLP). Telemetry is disabled by default and must be configured to enable export.

| Signal | Default | Destination |
|--------|---------|-------------|
| Logs | Enabled at `info` | stderr (stdio) or stdout (streamable_http), or configured log files |
| Traces | Export disabled | Configured OTLP tracing endpoint |
| Metrics | Export disabled | Configured OTLP metrics endpoint |

## Configuration

### Quick Start

```yaml
telemetry:
  exporters:
    metrics:
      otlp:
        endpoint: "http://localhost:4318/v1/metrics"
        protocol: "http/protobuf"
    tracing:
      otlp:
        endpoint: "http://localhost:4318/v1/traces"
        protocol: "http/protobuf"
```

### Full Configuration Reference

#### Top-Level Telemetry

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `service_name` | `string` | `"apollo-mcp-server"` | Service name in telemetry data |
| `version` | `string` | Current crate version | Service version in telemetry data |
| `exporters` | `Exporters` | `null` | Exporter configuration |

#### Metrics Exporter

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `otlp.endpoint` | `URL` | `http://localhost:4317` | OTLP export endpoint |
| `otlp.protocol` | `string` | `grpc` | `grpc` or `http/protobuf` |
| `otlp.temporality` | `string` | `cumulative` | `cumulative`, `delta`, or `lowmemory` |
| `otlp.metadata` | key-value pairs | | gRPC metadata (only with `grpc` protocol) |
| `otlp.headers` | key-value pairs | | HTTP headers (only with `http/protobuf` protocol) |
| `export_interval` | `Duration` | `30s` | Interval for metric export |
| `omitted_attributes` | `List<String>` | | Metric attributes to omit before export |

#### Tracing Exporter

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `otlp.endpoint` | `URL` | `http://localhost:4317` | OTLP export endpoint |
| `otlp.protocol` | `string` | `grpc` | `grpc` or `http/protobuf` |
| `otlp.metadata` | key-value pairs | | gRPC metadata (only with `grpc` protocol) |
| `otlp.headers` | key-value pairs | | HTTP headers (only with `http/protobuf` protocol) |
| `sampler` | `string` | `always_on` | `always_on`, `always_off`, or `0.0`-`1.0` percentage |
| `omitted_attributes` | `List<String>` | | Span attributes to omit before export |

### Production Example

```yaml
telemetry:
  service_name: "mcp-server-prod"
  version: "1.0.0"
  exporters:
    metrics:
      otlp:
        endpoint: "https://your-metrics-endpoint"
        protocol: "http/protobuf"
      export_interval: 1m
    tracing:
      otlp:
        endpoint: "https://your-traces-endpoint"
        protocol: "http/protobuf"
      sampler: 0.1
```

## Emitted Metrics

| Metric | Type | Description | Attributes |
|--------|------|-------------|------------|
| `apollo.mcp.initialize.count` | Counter | Each `initialize` request | `client_name`, `client_version` |
| `apollo.mcp.list_tools.count` | Counter | Each `list_tools` request | |
| `apollo.mcp.get_info.count` | Counter | Each `get_info` request | |
| `apollo.mcp.tool.count` | Counter | Each tool call | `tool_name`, `success` |
| `apollo.mcp.tool.duration` | Histogram | Tool execution duration (ms) | `tool_name`, `success` |
| `apollo.mcp.operation.count` | Counter | Each downstream GraphQL operation | `operation.id`, `operation.type`, `success` |
| `apollo.mcp.operation.duration` | Histogram | GraphQL operation round-trip (ms) | `operation.id`, `operation.type`, `success` |

Standard HTTP server metrics (`http.server.request.duration`, `http.server.active_requests`, `http.server.request.body.size`, `http.server.response.body.size`) are also emitted.

## Emitted Traces

Spans are generated for:

- **HTTP requests** to the MCP endpoint (not health checks)
- **MCP handler methods** (tools, resources, prompts, initialization)
- **Tool execution** with nested spans per tool
- **Downstream GraphQL calls** with `traceparent`, `tracestate`, and `baggage` propagation

### Key Span Attributes

`call_tool` span:

| Attribute | Description |
|-----------|-------------|
| `apollo.mcp.tool_name` | Tool name |
| `apollo.mcp.request_id` | MCP request ID |
| `apollo.mcp.tool_arguments` | Tool input (JSON) |
| `apollo.mcp.tool_result` | Tool output (JSON) |

GraphQL `execute` span:

| Attribute | Description |
|-----------|-------------|
| `apollo.mcp.graphql_query` | GraphQL query sent |
| `apollo.mcp.graphql_response` | GraphQL response received |

## Sensitive Data

Tool arguments, results, queries, and responses can contain sensitive data. Omit specific attributes:

```yaml
telemetry:
  exporters:
    tracing:
      omitted_attributes:
        - tool_arguments
        - tool_result
        - graphql_query
        - graphql_response
```

Use short attribute names like `tool_name`, `operation_id`, `tool_arguments`, `graphql_response`.

## Trace Context Propagation

MCP requests can carry W3C `traceparent`, `tracestate`, and `baggage` in `params._meta` over stdio or streamable HTTP. A valid `_meta.traceparent` becomes the parent of the MCP handler span. Trace context propagates automatically to downstream GraphQL requests.

## Dashboard Templates

Apollo provides pre-built dashboard templates at [apollographql/apm-templates](https://github.com/apollographql/apm-templates):

| Platform | Location |
|----------|----------|
| Grafana | `grafana/` |
| Datadog | `datadog/` |
