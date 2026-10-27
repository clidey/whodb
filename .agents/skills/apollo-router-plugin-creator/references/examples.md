# Plugin Examples

## Table of Contents
- [Simple Gate Plugin (execution_service)](#simple-gate-plugin-execution_service)
- [Checkpoint with Response Transform (supergraph_service)](#checkpoint-with-response-transform-supergraph_service)
- [Testing Plugins](#testing-plugins)

---

## Simple Gate Plugin (execution_service)

From Apollo Router's `forbid_mutations.rs` - demonstrates checkpoint pattern to gate on query plan:

```rust
fn execution_service(&self, service: execution::BoxService) -> execution::BoxService {
  if self.forbid {
    ServiceBuilder::new()
      .checkpoint(|req: ExecutionRequest| {
        if req.query_plan.contains_mutations() {
          let error = Error::builder()
            .message("Mutations are forbidden".to_string())
            .extension_code("MUTATION_FORBIDDEN")
            .build();
          let res = ExecutionResponse::builder()
            .error(error)
            .status_code(StatusCode::BAD_REQUEST)
            .context(req.context)
            .build()?;
          Ok(ControlFlow::Break(res))
        } else {
          Ok(ControlFlow::Continue(req))
        }
      })
      .service(service)
      .boxed()
  } else {
    service
  }
}
```

---

## Checkpoint with Response Transform (supergraph_service)

Demonstrates checkpoint + map_response with context passing:

```rust
fn supergraph_service(&self, service: supergraph::BoxService) -> supergraph::BoxService {
  if !self.configuration.enabled {
    return service;
  }

  let proxy_request_header = self.configuration.proxy_request_header.clone();
  ServiceBuilder::new()
    .checkpoint(move |request: supergraph::Request| {
      // Store data in context for response phase
      let _ = request.context.insert(
        IS_CGP_REQUESTS,
        request.supergraph_request.headers().contains_key(&proxy_request_header)
      );
      Ok(ControlFlow::Continue(request))
    })
    .map_response(|response: supergraph::Response| {
      // Check context values set during request phase
      match (
        response.context.get::<_, bool>(IS_CGP_REQUESTS),
        response.context.get::<_, bool>(IS_INTROSPECTION_QUERY)
      ) {
        (Ok(Some(true)), Ok(Some(true))) => {
          supergraph::Response::error_builder()
            .error(Error::builder()
              .message("Not allowed")
              .extension_code("FORBIDDEN_REQUEST")
              .build())
            .status_code(StatusCode::OK)
            .context(response.context)
            .build()
            .unwrap()
        }
        _ => response
      }
    })
    .service(service)
    .boxed()
}
```

---

## Testing Plugins

Use Apollo Router's test utilities to test plugins:

```rust
#[cfg(test)]
mod tests {
  use apollo_router::plugin::{test, Plugin, PluginInit};
  use apollo_router::services::supergraph::{self, Response};
  use apollo_router::Context;
  use tower::ServiceExt;

  #[tokio::test]
  async fn test_plugin() {
    // 1. Create plugin with fake config
    let service = MyPlugin::new(
      PluginInit::fake_builder()
        .config(MyPluginConfig { enabled: true })
        .build()
    )
    .await
    .expect("failed to create plugin")
    // 2. Wrap a mock service with the plugin's hook
    .supergraph_service({
      let mut mock = test::MockSupergraphService::new();
      mock.expect_call()
        .returning(|_| Response::fake_builder().build());
      mock.boxed()
    });

    // 3. Send a fake request through the service
    let response = service
      .oneshot(
        supergraph::Request::fake_builder()
          .context(Context::new())
          .query("query { field }")
          .build()
          .unwrap()
      )
      .await
      .unwrap();

    // 4. Assert on response
    let graphql_response = response.next_response().await.unwrap();
    assert!(graphql_response.errors.is_empty());
  }
}
```

### Testing with Headers

```rust
#[tokio::test]
async fn test_with_headers() {
  let service = MyPlugin::new(/* ... */)
    .await
    .expect("failed to create plugin")
    .supergraph_service(mock_service());

  let response = service
    .oneshot(
      supergraph::Request::fake_builder()
        .context(Context::new())
        .header("x-custom-header", "value")
        .query("query { field }")
        .build()
        .unwrap()
    )
    .await
    .unwrap();
}
```

### Testing Error Responses

```rust
#[tokio::test]
async fn test_error_response() {
  let service = MyPlugin::new(/* ... */)
    .await
    .expect("failed to create plugin")
    .supergraph_service(mock_service());

  let mut response = service
    .oneshot(request)
    .await
    .unwrap();

  let status_code = response.response.status().as_u16();
  let graphql_response = response.next_response().await.unwrap();

  assert_eq!(status_code, 200);
  assert!(!graphql_response.errors.is_empty());
  assert_eq!(
    graphql_response.errors[0].message,
    "Expected error message"
  );
  assert_eq!(
    graphql_response.errors[0].extensions.get("code").unwrap(),
    "ERROR_CODE"
  );
}
```
