# Apollo MCP Server Tools Reference

## Introspection Tools

Apollo MCP Server provides four introspection tools and one explorer tool. All are disabled by default and must be enabled in configuration.

Each introspection tool supports an optional `hint` config option for providing custom instructions to the AI agent.

### introspect

Explore schema types in detail with configurable depth.

**Parameters:**

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `type` | String | required | Type name to introspect |
| `depth` | Int | 1 | Recursion depth for related types |
| `minify` | Boolean | false | Use compact notation |

**Depth behavior:**

- `depth: 1` — Only the requested type
- `depth: 2` — Requested type + directly referenced types
- `depth: 3` — Two levels of related types

**Example output (normal):**

```graphql
type User {
  id: ID!
  name: String!
  email: String
  posts: [Post!]!
}
```

**Example output (minified):**

```
T User { id:d! name:s! email:s posts:[Post!]! }
```

### search

Find types in the schema matching a query.

**Parameters:**

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `query` | String | required | Search term |
| `leafDepth` | Int | 1 | Depth for leaf type expansion |
| `minify` | Boolean | false | Use compact notation |

**Config options:**

| Option | Default | Description |
|--------|---------|-------------|
| `index_memory_bytes` | `50000000` | Memory budget for the search index |
| `leaf_depth` | `1` | Default leaf type expansion depth |

### validate

Check if a GraphQL operation is valid against the schema. Validates syntax, schema conformance, variable definitions, and fragment validity. Does not execute the operation.

**Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `operation` | String | GraphQL operation to validate |

### execute

Run ad-hoc GraphQL operations against the endpoint. Parses the operation, applies mutation mode restrictions, and forwards to the GraphQL endpoint.

**Parameters:**

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `operation` | String | required | GraphQL operation |
| `variables` | Object | `{}` | Operation variables |

**Mutation mode behavior:**

| Mode | Queries | Mutations |
|------|---------|-----------|
| `all` | Allowed | Allowed |
| `explicit` | Allowed | Rejected |
| `none` | Allowed | Rejected |

Both `explicit` and `none` reject ad-hoc mutations through `execute`. The difference is that `explicit` exposes predefined mutation tools from operation sources, while `none` skips them.

> **Security:** `execute` accepts the GraphQL document from each tool call. It checks syntax, rejects subscriptions, and applies `mutation_mode`, but does not validate fields or types against the schema. The upstream GraphQL endpoint performs validation during execution.

### explorer

Returns a URL that opens a GraphQL operation in Apollo Explorer. Does not read the schema or call the GraphQL endpoint — it compresses the document, variables, and headers into the link.

Enable with `overrides.enable_explorer: true` (requires a GraphOS graph reference).

Annotations on the `explorer` tool are not configurable via `overrides.annotations`.

---

## Minify Notation

Compact notation reduces token usage. Enable globally via config or per-request.

### Type Abbreviations

| Symbol | Meaning |
|--------|---------|
| `T` | type |
| `I` | input |
| `E` | enum |
| `U` | union |
| `F` | interface |

### Scalar Abbreviations

| Symbol | Meaning |
|--------|---------|
| `s` | String |
| `i` | Int |
| `f` | Float |
| `b` | Boolean |
| `d` | ID |

### Modifiers

| Symbol | Meaning |
|--------|---------|
| `!` | Non-null (required) |
| `[]` | List |
| `@D` | Deprecated |
| `<>` | Implements |

---

## Custom Tools

Each GraphQL operation becomes an MCP tool:

- **Tool name**: Operation name (e.g., `GetUser`, `CreateProduct`)
- **Parameters**: Operation variables become tool parameters
- **Description**: From config-level overrides, operation descriptions, `#` comments, or schema (in that precedence order)

### Description Precedence

1. Config-level `overrides.descriptions`
2. Operation descriptions (GraphQL September 2025 spec `"""..."""` syntax)
3. `#` comments above the operation definition
4. Auto-generated from schema

Variable descriptions follow the same order except config-level descriptions do not apply to them.

### Tool Annotations

Auto-detected defaults:

- **Queries**: `read_only_hint: true`, `destructive_hint: false`, `idempotent_hint: true`, `open_world_hint: true`
- **Mutations**: `read_only_hint: false`, `destructive_hint: true`, `open_world_hint: true`

Customize via `overrides.annotations` in config.

### Output Schema

With `overrides.enable_output_schema: true`, tools include an `outputSchema` describing the GraphQL response envelope (`data`, `errors`, `extensions`).

### Tool Ordering

Predefined operation tools appear first in `tools/list`, alphabetically by operation name. Built-in and app tools follow. Deterministic ordering supports client-side caching and LLM prompt cache reuse.
