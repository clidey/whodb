# Service Hooks Reference

## Table of Contents
- [router_service](#router_service)
- [supergraph_service](#supergraph_service)
- [execution_service](#execution_service)
- [subgraph_service](#subgraph_service)
- [Async Operations](#async-operations)
- [Error Responses](#error-responses)

---

## router_service

Intercepts at HTTP level before GraphQL parsing. Access raw HTTP request/response.

```rust
fn router_service(&self, service: router::BoxService) -> router::BoxService {
  if !self.configuration.enabled {
    return service;
  }

  ServiceBuilder::new()
    .map_request(|mut request: router::Request| {
      // Access/modify headers
      let auth = request.router_request.headers().get("authorization");
      request.router_request.headers_mut().insert("x-custom", "value".parse().unwrap());

      // Add to context for later hooks
      request.context.insert("my_key", value).unwrap();

      request
    })
    .map_response(|mut response: router::Response| {
      // Modify response headers
      response.response.headers_mut().insert("cache-control", "no-store".parse().unwrap());

      response
    })
    .service(service)
    .boxed()
}
```

**Request types:**
- `request.router_request` - The HTTP request
- `request.router_request.headers()` - Request headers
- `request.router_request.headers_mut()` - Mutable headers
- `request.context` - Shared context

---

## supergraph_service

Intercepts at GraphQL level after parsing. Access query, variables, extensions.

```rust
fn supergraph_service(&self, service: supergraph::BoxService) -> supergraph::BoxService {
  if !self.configuration.enabled {
    return service;
  }

  ServiceBuilder::new()
    .checkpoint(move |request: supergraph::Request| {
      // Check introspection
      if let Ok(Some(true)) = request.context.get::<_, bool>(IS_INTROSPECTION_QUERY) {
        return Ok(ControlFlow::Break(
          supergraph::Response::error_builder()
            .error(Error::builder().message("Forbidden").extension_code("FORBIDDEN").build())
            .status_code(StatusCode::OK)
            .context(request.context)
            .build()
            .unwrap()
        ));
      }

      Ok(ControlFlow::Continue(request))
    })
    .map_response(|response: supergraph::Response| {
      // Transform response stream
      response.map_stream(|mut graphql_response| {
        graphql_response.extensions.insert("custom", json!({"key": "value"}).into());
        graphql_response
      })
    })
    .service(service)
    .boxed()
}
```

**Request types:**
- `request.supergraph_request` - The GraphQL request
- `request.supergraph_request.headers()` - Request headers
- `request.query_plan` - The query plan (in execution_service)
- `request.context` - Shared context

**Response transformation:**
```rust
// For streaming responses
response.map_stream(|mut graphql_response| {
  // Modify each response chunk
  graphql_response
})
```

---

## execution_service

Intercepts after query planning, before subgraph calls. Access to the query plan.

```rust
fn execution_service(&self, service: execution::BoxService) -> execution::BoxService {
  if !self.configuration.enabled {
    return service;
  }

  ServiceBuilder::new()
    .checkpoint(move |request: execution::Request| {
      // Access query plan
      if request.query_plan.contains_mutations() {
        return Ok(ControlFlow::Break(
          execution::Response::error_builder()
            .error(
              Error::builder()
                .message("Mutations are forbidden")
                .extension_code("MUTATION_FORBIDDEN")
                .build()
            )
            .status_code(StatusCode::BAD_REQUEST)
            .context(request.context)
            .build()
            .unwrap()
        ));
      }

      Ok(ControlFlow::Continue(request))
    })
    .service(service)
    .boxed()
}
```

**Request types:**
- `request.query_plan` - The generated query plan
- `request.query_plan.contains_mutations()` - Check if plan has mutations
- `request.context` - Shared context

**Use cases:**
- Forbid mutations
- Query cost/complexity limits
- Query plan inspection and logging

---

## subgraph_service

Intercepts per-subgraph calls. Receives subgraph name as parameter.

```rust
fn subgraph_service(&self, name: &str, service: subgraph::BoxService) -> subgraph::BoxService {
  if !self.configuration.enabled {
    return service;
  }

  let subgraph_name = name.to_string();
  let config = self.configuration.clone();

  ServiceBuilder::new()
    .map_request(move |mut request: subgraph::Request| {
      // Add headers to subgraph request
      request.subgraph_request.headers_mut().insert("x-subgraph", subgraph_name.parse().unwrap());

      request
    })
    .map_response(move |mut response: subgraph::Response| {
      // Read subgraph response extensions
      let extensions = &response.response.body().extensions;

      // Collect headers from subgraph
      let cookies = response.response.headers().get_all("set-cookie");

      response
    })
    .service(service)
    .boxed()
}
```

**Request types:**
- `request.subgraph_request` - The HTTP request to subgraph
- `response.response.body()` - The GraphQL response body
- `response.response.body().extensions` - Response extensions

**Per-subgraph configuration:**
```rust
fn subgraph_service(&self, name: &str, service: subgraph::BoxService) -> subgraph::BoxService {
  // Check if this subgraph has specific config
  let subgraph_config = self.configuration.subgraphs.get(name);

  if let Some(config) = subgraph_config {
    // Apply subgraph-specific logic
  }

  service
}
```

---

## Async Operations

Use `checkpoint_async` for async operations:

```rust
fn supergraph_service(&self, service: supergraph::BoxService) -> supergraph::BoxService {
  let client = self.http_client.clone();

  ServiceBuilder::new()
    .checkpoint_async(move |request: supergraph::Request| {
      let client = client.clone();
      async move {
        // Async validation
        let result = client.validate(&request).await;

        if result.is_err() {
          return Ok(ControlFlow::Break(error_response()));
        }

        Ok(ControlFlow::Continue(request))
      }
    })
    .service(service)
    .boxed()
}
```

Use `buffered()` when cloning the service:

```rust
ServiceBuilder::new()
  .buffered()
  .checkpoint_async(...)
  .service(service)
  .boxed()
```

---

## Error Responses

### GraphQL Error (200 OK with errors)
```rust
supergraph::Response::error_builder()
  .error(
    Error::builder()
      .message("Error message")
      .extension_code("ERROR_CODE")
      .build()
  )
  .status_code(StatusCode::OK)
  .context(request.context)
  .build()
  .unwrap()
```

### HTTP Error
```rust
supergraph::Response::error_builder()
  .error(Error::builder().message("Forbidden").build())
  .status_code(StatusCode::FORBIDDEN)
  .context(request.context)
  .build()
  .unwrap()
```

### Router-level Error
```rust
router::Response::error_builder()
  .error(Error::builder().message("Not Found").build())
  .status_code(StatusCode::NOT_FOUND)
  .context(request.context)
  .build()
  .unwrap()
```
