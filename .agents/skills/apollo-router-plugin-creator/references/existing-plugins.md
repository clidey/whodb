# Existing Plugins Reference

Reference plugins from the [Apollo Router open-source plugins directory](https://github.com/apollographql/router/tree/dev/apollo-router/src/plugins). Use as reference when creating new plugins.

## Discovering Plugins

Browse the [Apollo Router plugins directory](https://github.com/apollographql/router/tree/dev/apollo-router/src/plugins) for the full list of built-in plugins.

**Find plugins by service hook in a local checkout of the Apollo Router repo:**
```bash
# Plugins with router_service
grep -r "fn router_service" apollo-router/src/plugins/

# Plugins with supergraph_service
grep -r "fn supergraph_service" apollo-router/src/plugins/

# Plugins with subgraph_service
grep -r "fn subgraph_service" apollo-router/src/plugins/
```

## Configuration Patterns

### Simple Boolean
```rust
// introspection_gate.rs
#[derive(Debug, Default, Deserialize, JsonSchema)]
struct Config {
  enabled: bool,
  proxy_request_header: String
}
```

### With Optional Fields
```rust
// header_plugin.rs
#[derive(Clone, Debug, Default, Deserialize, JsonSchema)]
struct Config {
  enabled: bool,
  subgraphs_allowed_to_set_cookie: Option<Arc<HashSet<String>>>
}
```

### Nested Configuration
```rust
// add_extensions.rs
#[derive(Debug, Default, Clone, Deserialize, JsonSchema)]
struct Extensions {
  name: String,
  rename_as: Option<String>
}

#[derive(Debug, Default, Clone, Deserialize, JsonSchema)]
struct Config {
  enabled: bool,
  extensions_to_forward: Arc<Vec<Extensions>>
}
```

### Per-Subgraph Configuration
```rust
// subgraph_circuit_breaker
#[derive(Debug, Clone, Deserialize, JsonSchema)]
struct Config {
  enabled: bool,
  default: CircuitBreakerSettings,
  subgraphs: HashMap<String, CircuitBreakerSettings>
}
```

