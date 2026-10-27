---
name: apollo-mcp-server
description: >
  Guide for using Apollo MCP Server to connect AI agents with GraphQL APIs.
  Use this skill when: (1) setting up or configuring Apollo MCP Server,
  (2) defining MCP tools from GraphQL operations, (3) using introspection
  tools (introspect, search, validate, execute, explorer), (4) configuring
  authentication, CORS, health checks, or telemetry, (5) writing Rhai
  scripts for request lifecycle hooks, (6) defining MCP prompts,
  (7) troubleshooting MCP server connectivity or tool execution issues.
license: MIT
compatibility: Works with Claude Code, Claude Desktop, Cursor.
metadata:
  author: apollographql
  version: "1.2.0"
allowed-tools: Bash(rover:*) Bash(npx:*) Read Write Edit Glob Grep
---

# Apollo MCP Server Guide

Apollo MCP Server exposes GraphQL operations as MCP tools, enabling AI agents to interact with GraphQL APIs through the Model Context Protocol.

## Quick Start

### Step 1: Install

```bash
# Linux / MacOS
curl -sSL https://mcp.apollo.dev/download/nix/latest | sh

# Windows
iwr 'https://mcp.apollo.dev/download/win/latest' | iex
```

### Step 2: Configure

Create `config.yaml` in your project root:

```yaml
transport:
  type: streamable_http
schema:
  source: local
  path: ./schema.graphql
operations:
  source: local
  paths:
    - ./operations/
introspection:
  introspect:
    enabled: true
  search:
    enabled: true
  validate:
    enabled: true
  execute:
    enabled: true
```

Start the server:

```bash
apollo-mcp-server ./config.yaml
```

The MCP endpoint is available at `http://127.0.0.1:8000/mcp` (streamable_http defaults: address `127.0.0.1`, port `8000`). The GraphQL endpoint defaults to `http://localhost:4000/` — override with the `endpoint` key if your API runs elsewhere.

The server watches the config file for changes and automatically restarts with updated configuration.

### Step 3: Connect

Add to your MCP client configuration:

**Streamable HTTP (recommended):**

Claude Desktop (`claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "graphql-api": {
      "command": "npx",
      "args": ["mcp-remote", "http://127.0.0.1:8000/mcp"]
    }
  }
}
```

Claude Code:

```bash
claude mcp add graphql-api -- npx mcp-remote http://127.0.0.1:8000/mcp
```

**Stdio (client launches the server directly):**

```json
{
  "mcpServers": {
    "graphql-api": {
      "command": "./apollo-mcp-server",
      "args": ["./config.yaml"]
    }
  }
}
```

## Built-in Tools

| Tool | Purpose | When to Use |
|------|---------|-------------|
| `introspect` | Explore schema types in detail | Need type definitions, fields, relationships |
| `search` | Find types in schema | Looking for specific types or fields |
| `validate` | Check operation validity | Before executing operations |
| `execute` | Run ad-hoc GraphQL operations | Testing or one-off queries |
| `explorer` | Open operation in Apollo Explorer | Generate Explorer URLs (requires GraphOS graph ref) |

## Defining Custom Tools

MCP tools are created from GraphQL operations:

### 1. Operation Files (Recommended)

```yaml
operations:
  source: local
  paths:
    - ./operations/
```

### 2. Operation Collections

```yaml
operations:
  source: collection
  id: your-collection-id
```

### 3. Persisted Queries

```yaml
operations:
  source: manifest
  path: ./persisted-query-manifest.json
```

### 4. Schema Introspection

```yaml
operations:
  source: introspect
```

### 5. GraphOS Uplink

```yaml
operations:
  source: uplink
```

## Reference Files

- [Tools](references/tools.md) - Introspection tools, explorer, minify notation, tool annotations
- [Configuration](references/configuration.md) - All configuration options
- [Authentication](references/auth.md) - OAuth 2.1, scopes, token validation, per-operation scopes
- [Rhai Scripting](references/rhai-scripting.md) - Lifecycle hooks and built-in functions
- [MCP Prompts](references/prompts.md) - Reusable prompt templates
- [Telemetry](references/telemetry.md) - OpenTelemetry metrics, traces, and dashboards
- [Troubleshooting](references/troubleshooting.md) - Common issues and solutions

## Key Rules

### Security

> **Security: Sensitive operations.** Configure authentication before exposing to AI agents. Use `headers` for API keys, OAuth for user-scoped access.

- Disable introspection tools in production (they are disabled by default)
- Use `overrides.mutation_mode` to control mutation exposure

### Authentication

```yaml
# Static header
headers:
  Authorization: "Bearer ${env.API_TOKEN}"

# Dynamic header forwarding
forward_headers:
  - x-tenant-id

# OAuth (streamable_http transport)
transport:
  type: streamable_http
  auth:
    servers:
      - https://auth.example.com
    audiences:
      - https://api.example.com
    scopes:
      - read
    scope_mode: require_all
```

### Schema Sources

```yaml
# Local file
schema:
  source: local
  path: ./schema.graphql

# Federated graphs (uplink)
schema:
  source: uplink

# Non-federated graphs (graphos)
schema:
  source: graphos
```

### Mutations

Control mutation behavior via `overrides.mutation_mode`:

| Mode | Predefined mutations | Ad hoc mutations (`execute`) |
|------|---------------------|------------------------------|
| `none` | Skipped | Rejected |
| `explicit` | Exposed | Rejected |
| `all` | Exposed | Allowed |

### Token Optimization

Enable minification to reduce token usage:

```yaml
introspection:
  introspect:
    minify: true
  search:
    minify: true
```

## Common Patterns

### GraphOS Cloud Schema

```yaml
graphos:
  apollo_key: ${env.APOLLO_KEY}
  apollo_graph_ref: my-graph@production
schema:
  source: uplink # federated graphs; use graphos for non-federated graphs
```

### Local Development

```yaml
transport:
  type: streamable_http
schema:
  source: local
  path: ./schema.graphql
introspection:
  introspect:
    enabled: true
  search:
    enabled: true
  validate:
    enabled: true
  execute:
    enabled: true
overrides:
  mutation_mode: all
```

### Production Setup

```yaml
transport:
  type: streamable_http
  auth:
    servers:
      - https://auth.example.com
    audiences:
      - https://api.example.com
endpoint: https://api.production.com/graphql
operations:
  source: manifest
  path: ./persisted-query-manifest.json
graphos:
  apollo_key: ${env.APOLLO_KEY}
  apollo_graph_ref: ${env.APOLLO_GRAPH_REF}
schema:
  source: uplink # federated graphs; use graphos for non-federated graphs
health_check:
  enabled: true
telemetry:
  exporters:
    metrics:
      otlp:
        endpoint: https://your-metrics-endpoint
        protocol: http/protobuf
    tracing:
      otlp:
        endpoint: https://your-traces-endpoint
        protocol: http/protobuf
```

### Docker

```yaml
transport:
  type: streamable_http
  address: 0.0.0.0
  port: 8000
endpoint: ${env.GRAPHQL_ENDPOINT}
graphos:
  apollo_key: ${env.APOLLO_KEY}
  apollo_graph_ref: ${env.APOLLO_GRAPH_REF}
schema:
  source: uplink # federated graphs; use graphos for non-federated graphs
health_check:
  enabled: true
```

## Ground Rules

- ALWAYS configure authentication before exposing to AI agents
- ALWAYS use `mutation_mode: explicit` or `mutation_mode: none` in shared environments
- NEVER expose introspection tools with write access to production data
- NEVER place `auth` at the top level — it must be nested under `transport`
- PREFER operation files over ad-hoc execute for predictable behavior
- PREFER streamable_http transport for remote and multi-client deployments
- USE `schema.source: graphos` for non-federated graphs (uplink only serves federated graphs)
- USE stdio only when the MCP client launches the server process directly
- USE GraphOS Studio collections for team collaboration
