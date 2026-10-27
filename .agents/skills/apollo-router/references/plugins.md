# Router Customization

Extend Apollo Router functionality with Rhai scripts or external coprocessors.

## Customization Options

| Option | Language | Use Case |
|--------|----------|----------|
| **Rhai Scripts** | Rhai (embedded) | Simple transformations, header manipulation |
| **Coprocessors** | Any (HTTP service) | Complex logic, external service calls |
| **Native Plugins** | Rust | Maximum performance, deep integration |

## Rhai Scripts

Rhai is an embedded scripting language for lightweight customizations that run inside the Router.

### Enable Rhai

```yaml
# router.yaml
rhai:
  scripts: ./rhai
  main: main.rhai
```

### Script Location

```
project/
├── router.yaml
└── rhai/
    ├── main.rhai        # Entry point
    └── helpers.rhai     # Optional modules
```

### Basic Script Structure

```rhai
// main.rhai

// Called for each incoming request
fn supergraph_service(service) {
    service.map_request(|request| {
        // Modify request
        print(`Received request: ${request.uri.path}`);
    });
}

// Called for each subgraph request
fn subgraph_service(service, subgraph) {
    service.map_request(|request| {
        // Add header for specific subgraph
        if subgraph == "products" {
            request.subgraph.headers["x-products-version"] = "v2";
        }
    });
}
```

### Common Rhai Patterns

**Add Request Headers:**
```rhai
fn supergraph_service(service) {
    service.map_request(|request| {
        request.headers["x-custom-header"] = "value";
    });
}
```

**Access JWT Claims:**
```rhai
fn supergraph_service(service) {
    service.map_request(|request| {
        let claims = request.context["apollo_authentication::JWT::claims"];
        if claims != () {
            request.headers["x-user-id"] = claims["sub"];
        }
    });
}
```

**Modify Response:**
```rhai
fn supergraph_service(service) {
    service.map_response(|response| {
        response.headers["x-served-by"] = "apollo-router";
    });
}
```

**Early Return (Reject Request):**
```rhai
fn supergraph_service(service) {
    service.map_request(|request| {
        if request.headers["x-api-key"] == () {
            throw #{
                status: 401,
                message: "API key required"
            };
        }
    });
}
```

### Rhai Hooks

| Hook | Description |
|------|-------------|
| `supergraph_service` | Entry point for all requests |
| `execution_service` | After query planning |
| `subgraph_service` | Before each subgraph request |

## Coprocessors

External HTTP services that process requests/responses at various stages.

### Enable Coprocessor

```yaml
# router.yaml
coprocessor:
  url: http://localhost:8080
  timeout: 2s
  router:
    request:
      headers: true
      body: true
    response:
      headers: true
      body: true
```

### Coprocessor Stages

```yaml
coprocessor:
  url: http://localhost:8080
  
  # Router-level (full request)
  router:
    request:
      headers: true
      body: true
    response:
      headers: true
      body: false
  
  # Subgraph-level (per subgraph)
  subgraph:
    all:
      request:
        headers: true
        body: false
```

### Coprocessor Request Format

The Router sends POST requests with this structure:

```json
{
  "version": 1,
  "stage": "RouterRequest",
  "control": "continue",
  "id": "request-uuid",
  "headers": {
    "authorization": ["Bearer token"],
    "content-type": ["application/json"]
  },
  "body": "{\"query\": \"{ users { id } }\"}"
}
```

### Coprocessor Response Format

Return JSON to modify the request/response:

```json
{
  "version": 1,
  "stage": "RouterRequest",
  "control": "continue",
  "headers": {
    "x-user-id": ["123"]
  }
}
```

To reject a request:

```json
{
  "version": 1,
  "stage": "RouterRequest",
  "control": {
    "break": 403
  },
  "body": "{\"errors\": [{\"message\": \"Forbidden\"}]}"
}
```

### Example Coprocessor (Node.js)

```javascript
const express = require('express');
const app = express();
app.use(express.json());

app.post('/', (req, res) => {
  const { stage, headers, body } = req.body;
  
  // Authentication check
  if (stage === 'RouterRequest') {
    if (!headers.authorization) {
      return res.json({
        version: 1,
        stage,
        control: { break: 401 },
        body: JSON.stringify({
          errors: [{ message: 'Unauthorized' }]
        })
      });
    }
  }
  
  // Continue with optional modifications
  res.json({
    version: 1,
    stage,
    control: 'continue',
    headers: {
      'x-processed-by': ['coprocessor']
    }
  });
});

app.listen(8080);
```

## Choosing Between Rhai and Coprocessors

| Criteria | Rhai | Coprocessor |
|----------|------|-------------|
| Latency | ~1-5ms | ~10-50ms (network) |
| Language | Rhai only | Any language |
| Complexity | Simple logic | Complex business logic |
| External calls | Not supported | Supported |
| Deployment | Bundled with Router | Separate service |

### When to Use Rhai

- Header manipulation
- Simple request/response transformations
- Logging and debugging
- Context value extraction

### When to Use Coprocessors

- External API calls (validation, enrichment)
- Complex business logic
- Database lookups
- Integration with existing services
- Team uses a specific language

## Native Plugins (Advanced)

For maximum performance, build custom Router binaries with Rust plugins.

```bash
# Clone Router
git clone https://github.com/apollographql/router.git

# Create custom plugin in apollo-router/src/plugins/

# Build with custom plugins
cargo build --release
```

Native plugins require Rust knowledge and a custom build pipeline.
