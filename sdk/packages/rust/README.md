# whodb-sdk (Rust)

Official Rust SDK for the [WhoDB](https://whodb.com) hosted platform — your
ontology as in-code function APIs.

## Install

```bash
cargo add whodb-sdk
```

Requires Rust ≥ 1.75. Synchronous API; two dependencies (`serde_json`, `ureq`).

## Quickstart

```rust
use serde_json::{json, Map};
use whodb_sdk::{Client, Config, ListOptions};

// Production: API key (create one in org settings → API keys).
let client = Client::new(Config {
    api_key: std::env::var("WHODB_API_KEY").ok(),
    ..Default::default()
})?;

// Local development: zero config — reuses your `whodb login` session.
// let client = Client::new(Config::default())?;

let users = client.ontology("User");

let user = users.get("u_123")?;               // Option<Row>
let rows = users.list(&ListOptions {
    where_filter: Some(json!({"status": {"eq": "active"}})),
    page_size: 100,
    ..Default::default()
})?;
users.create(&Map::from_iter([("email".into(), json!("a@b.co"))]))?;
users.create_many(&rows_to_insert, Some("import-42"))?; // idempotency key
users.update("u_123", &Map::from_iter([("plan".into(), json!("pro"))]))?;
let orders = users.follow_link("u_123", "orders", 50, 0)?;

// Iterate everything, page by page:
users.pages(&ListOptions { page_size: 500, ..Default::default() }, |rows| {
    println!("{}", rows.len());
    true // false stops iteration
})?;
```

Rows are `serde_json::Map<String, Value>` with native-typed values. Hydrated
timestamps carry an `@date:<RFC3339>` string marker (Rust has no std datetime
type) — parse with your preferred time crate.

## Authentication

Credential precedence: `Config` fields (`api_key` / `token` / `credentials`)
→ `WHODB_API_KEY` env var → the `whodb` CLI's stored login. Workspace
(`org` / `project`) is optional with an API key; set `project` when the key
has access to more than one project.

Inside a WhoDB Function (K8s/TCP runtime), `Client::new(Config::default())`
auto-detects the runtime. The Docker runtime's unix-socket IPC is not yet
supported by this crate.

## Errors

`whodb_sdk::Error` variants: `Auth`, `NotFound`, `Validation`, `Version`
(SDK outdated for the platform API — upgrade the crate), `CliCredentials`,
`TransportCapability`, `Platform { message, code }`, `Transport`.

## License

Apache-2.0
