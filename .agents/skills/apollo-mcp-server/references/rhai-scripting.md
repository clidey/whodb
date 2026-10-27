# Rhai Scripting

## Overview

Apollo MCP Server supports [Rhai](https://rhai.rs), a lightweight embedded scripting language for customizing server behavior at runtime. Rhai scripts hook into the request lifecycle to inspect and modify outgoing GraphQL requests without recompiling the server.

Common use cases:

- Forward authentication headers from MCP requests to your GraphQL endpoint
- Route requests to different GraphQL endpoints based on request properties
- Reject requests that fail custom validation

> **Security:** Treat Rhai scripts as trusted code. Scripts can read environment variables, make outbound HTTP requests, and change the endpoint or headers of outgoing GraphQL requests. Restrict write access to the script directory and review script changes before deploying.

## Setup

Create a `rhai/` directory with a `main.rhai` entry point:

```
your-project/
├── config.yaml
└── rhai/
    └── main.rhai
```

If `rhai/main.rhai` does not exist, the server starts normally without scripting. Configure a custom directory with:

```yaml
rhai:
  scripts: /config/rhai
```

## Lifecycle Hook: on_execute_graphql_operation

Called before every outgoing GraphQL HTTP request. Define the function in `main.rhai`:

```rhai
fn on_execute_graphql_operation(ctx) {
    ctx.headers["x-custom-header"] = "my-value";
}
```

### Context Object Properties

| Property | Type | Access | Description |
|----------|------|--------|-------------|
| `endpoint` | `String` | read/write | GraphQL endpoint URL for the request |
| `headers` | `HeaderMap` | read/write | HTTP headers for the outgoing request |
| `incoming_request` | `HttpParts` | read-only | Original HTTP request (streamable_http only) |
| `tool_name` | `String` | read-only | MCP tool that triggered this operation |
| `trace_id` | `String` | read-only | Current OpenTelemetry trace ID (32-char hex, or empty) |

### incoming_request Object

| Property | Type | Description |
|----------|------|-------------|
| `method` | `String` | HTTP method (e.g., `"POST"`) |
| `uri` | `String` | Request URI path (e.g., `"/mcp"`) |
| `headers` | `HeaderMap` | HTTP headers from the incoming request |

Empty when using `stdio` transport.

## Common Patterns

### Forward Auth Headers

```rhai
fn on_execute_graphql_operation(ctx) {
    let token = ctx.incoming_request.headers["authorization"];
    if token != "" {
        ctx.headers["x-forwarded-auth"] = token;
    }
}
```

### Route by Environment

```rhai
let region = Env::get("GRAPHQL_REGION");

fn on_execute_graphql_operation(ctx) {
    if region == "eu" {
        ctx.endpoint = "https://eu.api.example.com/graphql";
    }
}
```

### Require an Authorization Header

This only checks that the header is present. It does not validate the token. Use `transport.auth` (see auth.md) for real authentication.

```rhai
fn on_execute_graphql_operation(ctx) {
    if ctx.incoming_request.headers["authorization"] == "" {
        throw #{
            message: "Missing authorization header",
            code: ErrorCode::INVALID_REQUEST
        };
    }
}
```

## Error Handling

Use `throw` with a map containing `message` and `code`:

| Constant | Description |
|----------|-------------|
| `ErrorCode::INVALID_REQUEST` | Client error (missing header, bad input) |
| `ErrorCode::INTERNAL_ERROR` | Server error (default if no code provided) |

Throwing a non-map value logs the value server-side but returns a generic internal error to the client.

## Built-in Functions

### Env

| Function | Description |
|----------|-------------|
| `Env::get(name)` | Read environment variable (returns empty string if unset) |

### JSON

| Function | Description |
|----------|-------------|
| `JSON::parse(input)` | Parse JSON string into Rhai value |
| `JSON::stringify(value)` | Convert Rhai value to JSON string |

### Sha256

| Function | Description |
|----------|-------------|
| `Sha256::digest(input)` | Hex-encoded SHA-256 hash |

### Regex

| Function | Description |
|----------|-------------|
| `Regex::is_match(string, pattern)` | Test if pattern matches |
| `Regex::replace(string, pattern, replacement)` | Replace all matches (supports capture groups) |
| `Regex::matches(string, pattern)` | Return all matching substrings |

### Http

| Function | Description |
|----------|-------------|
| `Http::get(url)` / `Http::get(url, options)` | GET request, returns `Promise` |
| `Http::post(url)` / `Http::post(url, options)` | POST request, returns `Promise` |

Options: `headers` (Map), `body` (String), `timeout` (Integer, default 30s).

Call `.wait()` on the promise. Response has `.status` (Integer), `.text()` (String), `.json()` (Dynamic).

## Modules

Split scripts into multiple files:

```rhai
// rhai/main.rhai
import "helpers" as helpers;

fn on_execute_graphql_operation(ctx) {
    helpers::add_auth_headers(ctx);
}
```

## Global Variables

Top-level variables are available to every hook but are copied per invocation. Hooks cannot carry state between requests.
