---
name: apollo-router-plugin-creator
description: >
  Guide for writing Apollo Router native Rust plugins. Use this skill when:
  (1) users want to create a new router plugin,
  (2) users want to add service hooks (router_service, supergraph_service, execution_service, subgraph_service),
  (3) users want to modify an existing router plugin,
  (4) users need to understand router plugin patterns or the request lifecycle.
  (5) triggers on requests like "create a new plugin", "add a router plugin", "modify the X plugin", or "add subgraph_service hook".
license: MIT
allowed-tools: Read Write Edit Glob Grep
metadata:
  author: apollographql
  version: "1.0.0"
  compatibility: "Requires Apollo Router with native plugin support"
---

# Apollo Router Plugin Creator

Create native Rust plugins for Apollo Router.

## Request Lifecycle

```
┌────────┐             ┌────────────────┐                                   ┌────────────────────┐               ┌───────────────────┐       ┌─────────────────────┐
│ Client │             │ Router Service │                                   │ Supergraph Service │               │ Execution Service │       │ Subgraph Service(s) │
└────┬───┘             └────────┬───────┘                                   └──────────┬─────────┘               └─────────┬─────────┘       └──────────┬──────────┘
     │                          │                                                      │                                   │                            │
     │      Sends request       │                                                      │                                   │                            │
     │──────────────────────────▶                                                      │                                   │                            │
     │                          │                                                      │                                   │                            │
     │                          │  Converts raw HTTP request to GraphQL/JSON request   │                                   │                            │
     │                          │──────────────────────────────────────────────────────▶                                   │                            │
     │                          │                                                      │                                   │                            │
     │                          │                                                      │  Initiates query plan execution   │                            │
     │                          │                                                      │───────────────────────────────────▶                            │
     │                          │                                                      │                                   │                            │
     │                          │                                                      │                               ┌par [Initiates sub-operation]───────┐
     │                          │                                                      │                               │   │                            │   │
     │                          │                                                      │                               │   │  Initiates sub-operation   │   │
     │                          │                                                      │                               │   │────────────────────────────▶   │
     │                          │                                                      │                               │   │                            │   │
     │                          │                                                      │                               ├[Initiates sub-operation]╌╌╌╌╌╌╌╌╌╌╌┤
     │                          │                                                      │                               │   │                            │   │
     │                          │                                                      │                               │   │  Initiates sub-operation   │   │
     │                          │                                                      │                               │   │────────────────────────────▶   │
     │                          │                                                      │                               │   │                            │   │
     │                          │                                                      │                               ├[Initiates sub-operation]╌╌╌╌╌╌╌╌╌╌╌┤
     │                          │                                                      │                               │   │                            │   │
     │                          │                                                      │                               │   │  Initiates sub-operation   │   │
     │                          │                                                      │                               │   │────────────────────────────▶   │
     │                          │                                                      │                               │   │                            │   │
     │                          │                                                      │                               └────────────────────────────────────┘
     │                          │                                                      │                                   │                            │
     │                          │                                                      │  Assembles and returns response   │                            │
     │                          │                                                      ◀╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌│                            │
     │                          │                                                      │                                   │                            │
     │                          │            Returns GraphQL/JSON response             │                                   │                            │
     │                          ◀╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌│                                   │                            │
     │                          │                                                      │                                   │                            │
     │  Returns HTTP response   │                                                      │                                   │                            │
     ◀╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌│                                                      │                                   │                            │
     │                          │                                                      │                                   │                            │
┌────┴───┐             ┌────────┴───────┐                                   ┌──────────┴─────────┐               ┌─────────┴─────────┐       ┌──────────┴──────────┐
│ Client │             │ Router Service │                                   │ Supergraph Service │               │ Execution Service │       │ Subgraph Service(s) │
└────────┘             └────────────────┘                                   └────────────────────┘               └───────────────────┘       └─────────────────────┘
```

## Service Hooks

### Service Overview

| Service              | Description                                                                                                                                                                                                                                                                                                                                                                                                                                           |
|----------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `router_service`     | Runs at the very beginning and very end of the HTTP request lifecycle.For example, JWT authentication is performed within the RouterService.Define router_service if your customization needs to interact with HTTP context and headers. It doesn't support access to the body property                                                                                                                                                               |
| `supergraph_service` | Runs at the very beginning and very end of the GraphQL request lifecycle.Define supergraph_service if your customization needs to interact with the GraphQL request or the GraphQL response. For example, you can add a check for anonymous queries.                                                                                                                                                                                                  |
| `execution_service`  | Handles initiating the execution of a query plan after it's been generated.Define execution_service if your customization includes logic to govern execution (for example, if you want to block a particular query based on a policy decision).                                                                                                                                                                                                       |
| `subgraph_service`   | Handles communication between the router and your subgraphs.Define subgraph_service to configure this communication (for example, to dynamically add HTTP headers to pass to a subgraph).Whereas other services are called once per client request, this service is called once per subgraph request that's required to resolve the client's request. Each call is passed a subgraph parameter that indicates the name of the corresponding subgraph. |

**Signatures:**
```rust
fn router_service(&self, service: router::BoxService) -> router::BoxService
fn supergraph_service(&self, service: supergraph::BoxService) -> supergraph::BoxService
fn execution_service(&self, service: execution::BoxService) -> execution::BoxService
fn subgraph_service(&self, name: &str, service: subgraph::BoxService) -> subgraph::BoxService
```

### Individual Hooks (Tower Layers)

Use `ServiceBuilder` to compose these hooks within any service:

| Hook                      | Purpose                                      | Sync/Async |
|---------------------------|----------------------------------------------|------------|
| `map_request(fn)`         | Transform request before proceeding          | Sync       |
| `map_response(fn)`        | Transform response before returning          | Sync       |
| `checkpoint(fn)`          | Validate/filter, can short-circuit           | Sync       |
| `checkpoint_async(fn)`    | Async validation, can short-circuit          | Async      |
| `buffered()`              | Enable service cloning (needed for async)    | -          |
| `instrument(span)`        | Add tracing span around service              | -          |
| `rate_limit(num, period)` | Control request throughput                   | -          |
| `timeout(duration)`       | Set operation time limit                     | -          |

### Choosing a Service Hook

**By data needed:**
- HTTP headers only → `router_service`
- GraphQL query/variables → `supergraph_service`
- Query plan → `execution_service`
- Per-subgraph control → `subgraph_service`

**By timing:**
- Before GraphQL parsing → `router_service` request
- After parsing, before planning → `supergraph_service` request
- After planning, before execution → `execution_service` request
- Before/after each subgraph call → `subgraph_service`
- Final response to client → `router_service` response

See [references/service-hooks.md](references/service-hooks.md) for implementation patterns.

## Quick Start

### Step 1: Create Plugin File

Create a new file `src/plugins/my_plugin.rs` with required imports:

```rust
use std::ops::ControlFlow;
use apollo_router::plugin::{Plugin, PluginInit};
use apollo_router::register_plugin;
use apollo_router::services::{router, subgraph, supergraph};
use schemars::JsonSchema;
use serde::Deserialize;
use tower::{BoxError, ServiceBuilder, ServiceExt};

const PLUGIN_NAME: &str = "my_plugin";
```

### Step 2: Define Configuration Struct

Every plugin needs a configuration struct with `Deserialize` and `JsonSchema` derives. The `JsonSchema` enables configuration validation in editors:

```rust
#[derive(Debug, Clone, Default, Deserialize, JsonSchema)]
struct MyPluginConfig {
  /// Enable the plugin
  enabled: bool,
  // Add other configuration fields as needed
}
```

### Step 3: Define Plugin Struct

```rust
#[derive(Debug)]
struct MyPlugin {
  configuration: MyPluginConfig,
}
```

### Step 4: Implement Plugin Trait

Implement the `Plugin` trait with the required `Config` type and `new` constructor:

```rust
#[async_trait::async_trait]
impl Plugin for MyPlugin {
  type Config = MyPluginConfig;

  async fn new(init: PluginInit<Self::Config>) -> Result<Self, BoxError> {
    Ok(MyPlugin { configuration: init.config })
  }

  // Add service hooks based on your needs (see "Choosing a Service Hook" section)
}
```

### Step 5: Add Service Hooks

Choose which service(s) to hook based on your requirements, see [Service Overview](#service-overview) for details.

Example service hook:
```rust
fn supergraph_service(&self, service: supergraph::BoxService) -> supergraph::BoxService {
  if !self.configuration.enabled {
    return service;
  }

  ServiceBuilder::new()
    .map_request(|req| { /* transform request */ req })
    .map_response(|res| { /* transform response */ res })
    .service(service)
    .boxed()
}
```

### Step 6: Register Plugin

At the bottom of your plugin file, register it with the router:

```rust
register_plugin!("acme", "my_plugin", MyPlugin);
```

### Step 7: Add Module to mod.rs

In `src/plugins/mod.rs`, add your module:

```rust
pub mod my_plugin;
```

### Step 8: Configure in YAML

Enable your plugin in the router configuration:

```yaml
plugins:
  acme.my_plugin:
    enabled: true
```

## Common Patterns

For implementation patterns and code examples, see [references/service-hooks.md](references/service-hooks.md):
- Enable/disable pattern
- Request/response transformation (`map_request`, `map_response`)
- Checkpoint (early return/short-circuit)
- Context passing between hooks
- Async operations (`checkpoint_async`, `buffered`)
- Error response builders

## Examples

### Apollo Router Examples

Located in the [Apollo Router plugins directory](https://github.com/apollographql/router/tree/dev/apollo-router/src/plugins):

| Plugin                 | Service Hook           | Pattern           | Description                 |
|------------------------|------------------------|-------------------|-----------------------------|
| `forbid_mutations.rs`  | `execution_service`    | checkpoint        | Simple gate on query plan   |
| `expose_query_plan.rs` | execution + supergraph | Context passing   | Multi-service coordination  |
| `cors.rs`              | `router_service`       | HTTP layer        | CORS handling at HTTP level |
| `headers/`             | `subgraph_service`     | Layer composition | Complex header manipulation |

For full code examples and testing patterns, see [references/examples.md](references/examples.md).

## Prerequisites

It is advised to have the [rust-best-practices](https://skills.sh/apollographql/skills/rust-best-practices) skill installed for writing idiomatic Rust code when developing router plugins. If installed, follow those best practices when generating or modifying plugin code.

## Resources

- [references/service-hooks.md](references/service-hooks.md) - Detailed service hook implementations
- [references/existing-plugins.md](references/existing-plugins.md) - Index of existing plugins
- [references/examples.md](references/examples.md) - Full code examples and testing patterns
- Apollo Router plugins: https://github.com/apollographql/router/tree/dev/apollo-router/src/plugins