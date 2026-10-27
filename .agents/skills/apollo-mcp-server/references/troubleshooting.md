# Apollo MCP Server Troubleshooting

## Table of Contents

- [Debugging with MCP Inspector](#debugging-with-mcp-inspector)
- [Connection Issues](#connection-issues)
- [Authentication Problems](#authentication-problems)
- [Schema Issues](#schema-issues)
- [Tool Execution Errors](#tool-execution-errors)
- [Health Check](#health-check)

---

## Debugging with MCP Inspector

MCP Inspector provides visual debugging for MCP servers.

### Installation

```bash
npx @modelcontextprotocol/inspector
```

### Usage

```bash
# Start inspector with your MCP server using the Streamable HTTP transport
npx @modelcontextprotocol/inspector --transport http --server-url http://localhost:8000/mcp
```

```bash
# Start inspector with your MCP server using the STDIO transport
npx @modelcontextprotocol/inspector apollo-mcp-server ./config.yaml
```

### Inspector Features

- View available tools and their schemas
- Test tool invocations interactively
- Inspect request/response payloads
- Monitor server logs in real-time

---

## Connection Issues

### Server Won't Start

**Symptoms:** Server exits immediately or hangs

**Solutions:**

1. Check config file syntax:
```bash
# Validate YAML
npx yaml-lint config.yaml
```

2. Enable debug logging:
```bash
APOLLO_MCP_LOGGING__LEVEL=debug apollo-mcp-server ./config.yaml
```

### Client Can't Connect

**Symptoms:** MCP client shows "server not found" or timeout

**Solutions:**

**Streamable HTTP (recommended for remote/multi-client):**

Configure your client to connect via `npx mcp-remote`:

```json
{
  "mcpServers": {
    "graphql": {
      "command": "npx",
      "args": ["mcp-remote", "http://127.0.0.1:8000/mcp"]
    }
  }
}
```

**Stdio (client launches the server process):**

```json
{
  "mcpServers": {
    "graphql": {
      "command": "./apollo-mcp-server",
      "args": ["/absolute/path/to/config.yaml"]
    }
  }
}
```

Test server manually:
```bash
apollo-mcp-server ./config.yaml
# Should output JSON-RPC initialization
```

Check binary is installed:
```bash
which apollo-mcp-server
```

---

## Authentication Problems

### 401 Unauthorized

**Symptoms:** All requests return 401

**Solutions:**

1. Verify API token is set:
```bash
echo $API_TOKEN  # Should not be empty
```

2. Check header configuration:
```yaml
headers:
  Authorization: "Bearer ${env.API_TOKEN}"
```

3. For dynamic token forwarding:
```yaml
forward_headers:
  - x-forwarded-authorization
```

### Token Security Best Practices

- **Never commit tokens** to version control
- Use environment variables or secrets management
- Rotate tokens regularly
- Use minimum required permissions

### OAuth Authentication

For streamable_http transport, use `transport.auth` for OAuth:

```yaml
transport:
  type: streamable_http
  auth:
    servers:
      - https://auth.example.com
    audiences:
      - https://api.example.com
    scopes:
      - read
```

For forwarding user tokens to the upstream GraphQL API:

```yaml
forward_headers:
  - authorization
```

**Security Warning:** Forwarding OAuth tokens exposes them to the MCP server. Ensure:
- Server runs in trusted environment
- Transport is encrypted (HTTPS)
- Tokens have minimal scope

---

## Schema Issues

### Schema Not Found

**Symptoms:** "Schema file not found" error

**Solutions:**

1. Check file path (use absolute paths):
```yaml
schema:
  source: local
  path: /absolute/path/to/schema.graphql
```

2. Verify file exists:
```bash
ls -la ./schema.graphql
```

### GraphOS Uplink Errors

**Symptoms:** "Failed to fetch schema from uplink"

**Solutions:**

1. Verify GraphOS credentials:
```bash
echo $APOLLO_KEY
echo $APOLLO_GRAPH_REF
```

2. Check graph reference format:
```yaml
graphos:
  apollo_graph_ref: my-graph@production  # Format: graph@variant
```

---

## Tool Execution Errors

### Operation Validation Failed

**Symptoms:** "Field X not found on type Y"

**Solutions:**

1. Ensure schema is up to date
2. Use `validate` tool before `execute`:
```
validate(operation: "query { user { id name } }")
```

### Mutation Blocked

**Symptoms:** "Mutations are disabled"

**Solutions:**

Check mutation mode in config:
```yaml
overrides:
  mutation_mode: all  # Or 'explicit' for predefined mutations only
```

### Variable Type Mismatch

**Symptoms:** "Variable $id expected ID!, got String"

**Solutions:**

Ensure variable types match operation:
```graphql
# Operation expects ID!
query GetUser($id: ID!) { ... }

# Correct invocation
execute(variables: { id: "123" })  # String coerced to ID
```

---

## Health Check

For streamable_http transport, health endpoints help diagnose issues.

### Endpoints

| Endpoint | Purpose |
|----------|---------|
| `/health` | Overall health status |
| `/health?live` | Liveness probe (is server running?) |
| `/health?ready` | Readiness probe (can server handle requests?) |

### Status Codes

| Code | Meaning |
|------|---------|
| 200 | Healthy |
| 503 | Unhealthy |

### Example Check

```bash
curl http://localhost:8000/health
# {"status": "UP"}

curl http://localhost:8000/health?ready
# {"status": "UP"}
```

### Common Health Issues

- Schema file missing/invalid or GraphOS uplink unreachable
- GraphQL endpoint unreachable or authentication problems

## Getting Help

1. Enable debug logging: `APOLLO_MCP_LOGGING__LEVEL=debug apollo-mcp-server ./config.yaml`
2. Apollo docs: https://apollographql.com/docs
3. Report issues: https://github.com/apollographql/apollo-mcp-server/issues
