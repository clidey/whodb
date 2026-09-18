//! HTTP transport, workspace-resolution, and IPC dispatch tests against real
//! local TCP stubs — the layers the mock-transport conformance suite cannot
//! reach.

use std::io::{Read, Write};
use std::net::TcpListener;
use std::sync::atomic::{AtomicUsize, Ordering};
use std::sync::{Arc, Mutex};
use std::thread;

use serde_json::{json, Map, Value};
use whodb_sdk::auth::CredentialProvider;
use whodb_sdk::transport_ipc::IpcTransport;
use whodb_sdk::{Client, Config, Error, Transport};

/// Converts a `json!` object literal into the owned map `Transport` expects.
fn variables(value: Value) -> Map<String, Value> {
    value.as_object().expect("object literal").clone()
}

/// One request the stub saw: lowercase header lines plus the parsed JSON body.
struct Recorded {
    headers: Vec<String>,
    body: Value,
}

impl Recorded {
    fn header(&self, name: &str) -> Option<String> {
        let prefix = format!("{}:", name.to_lowercase());
        self.headers
            .iter()
            .find(|line| line.to_lowercase().starts_with(&prefix))
            .map(|line| line[prefix.len()..].trim().to_string())
    }

    fn operation(&self) -> String {
        let query = self.body["query"].as_str().unwrap_or_default();
        query
            .split_whitespace()
            .nth(1)
            .unwrap_or_default()
            .split(['(', '{'])
            .next()
            .unwrap_or_default()
            .to_string()
    }
}

/// Spawns a scripted HTTP/1.1 stub replaying `responses` (status, JSON body)
/// in order; returns its host URL and the recorded requests.
fn spawn_stub(responses: Vec<(u16, Value)>) -> (String, Arc<Mutex<Vec<Recorded>>>) {
    let listener = TcpListener::bind("127.0.0.1:0").expect("bind stub");
    let host = format!("http://{}", listener.local_addr().expect("stub addr"));
    let recorded = Arc::new(Mutex::new(Vec::new()));
    let sink = Arc::clone(&recorded);
    thread::spawn(move || {
        for (status, body) in responses {
            let (mut stream, _) = match listener.accept() {
                Ok(accepted) => accepted,
                Err(_) => return,
            };
            let mut raw = Vec::new();
            let mut chunk = [0u8; 4096];
            let (header_end, content_length) = loop {
                let read = stream.read(&mut chunk).unwrap_or(0);
                if read == 0 {
                    break (raw.len(), 0);
                }
                raw.extend_from_slice(&chunk[..read]);
                if let Some(position) = raw.windows(4).position(|w| w == b"\r\n\r\n") {
                    let headers = String::from_utf8_lossy(&raw[..position]).to_lowercase();
                    let length = headers
                        .lines()
                        .find_map(|line| line.strip_prefix("content-length:"))
                        .and_then(|value| value.trim().parse::<usize>().ok())
                        .unwrap_or(0);
                    break (position + 4, length);
                }
            };
            while raw.len() < header_end + content_length {
                let read = stream.read(&mut chunk).unwrap_or(0);
                if read == 0 {
                    break;
                }
                raw.extend_from_slice(&chunk[..read]);
            }
            let headers = String::from_utf8_lossy(&raw[..header_end])
                .lines()
                .map(str::to_string)
                .collect();
            let body_json =
                serde_json::from_slice(&raw[header_end..]).unwrap_or_else(|_| json!({}));
            sink.lock().expect("recorder lock").push(Recorded {
                headers,
                body: body_json,
            });
            let payload = body.to_string();
            let response = format!(
                "HTTP/1.1 {status} X\r\nContent-Type: application/json\r\nContent-Length: {}\r\nConnection: close\r\n\r\n{payload}",
                payload.len(),
            );
            let _ = stream.write_all(response.as_bytes());
        }
    });
    (host, recorded)
}

fn client_with(host: &str, config: Config) -> Client {
    Client::new(Config {
        host: Some(host.to_string()),
        ..config
    })
    .expect("client")
}

const ORG_UUID: &str = "11111111-1111-1111-1111-111111111111";
const PROJECT_UUID: &str = "22222222-2222-2222-2222-222222222222";

#[test]
fn sends_bearer_user_agent_and_workspace_headers() {
    let (host, recorded) = spawn_stub(vec![(200, json!({"data": {"OntologyEntities": []}}))]);
    let client = client_with(
        &host,
        Config {
            token: Some("tok".to_string()),
            org: Some(ORG_UUID.to_string()),
            project: Some(PROJECT_UUID.to_string()),
            ..Default::default()
        },
    );
    client.ontology_entities().expect("entities");
    let requests = recorded.lock().expect("lock");
    let request = &requests[0];
    assert_eq!(
        request.header("authorization").as_deref(),
        Some("Bearer tok")
    );
    assert!(request
        .header("user-agent")
        .is_some_and(|ua| ua.starts_with("clidey-whodb-rust/")));
    assert_eq!(request.header("x-whodb-org-id").as_deref(), Some(ORG_UUID));
    assert_eq!(
        request.header("x-whodb-project-id").as_deref(),
        Some(PROJECT_UUID)
    );
}

#[test]
fn retries_once_on_transient_5xx() {
    let (host, recorded) = spawn_stub(vec![
        (503, json!({})),
        (200, json!({"data": {"OntologyEntities": []}})),
    ]);
    let client = client_with(
        &host,
        Config {
            token: Some("tok".to_string()),
            org: Some(ORG_UUID.to_string()),
            project: Some(PROJECT_UUID.to_string()),
            ..Default::default()
        },
    );
    client
        .ontology_entities()
        .expect("retried request must succeed");
    assert_eq!(recorded.lock().expect("lock").len(), 2);
}

/// Counts refresh calls and rotates the token afterwards.
struct CountingCredentials {
    refreshes: AtomicUsize,
}

impl CredentialProvider for CountingCredentials {
    fn token(&self) -> whodb_sdk::Result<String> {
        let count = self.refreshes.load(Ordering::SeqCst);
        Ok(if count == 0 {
            "stale".to_string()
        } else {
            format!("token-{count}")
        })
    }

    fn refresh(&self) {
        self.refreshes.fetch_add(1, Ordering::SeqCst);
    }
}

#[test]
fn refreshes_credentials_once_on_401() {
    let (host, recorded) = spawn_stub(vec![
        (401, json!({})),
        (200, json!({"data": {"OntologyEntities": []}})),
    ]);
    let client = client_with(
        &host,
        Config {
            credentials: Some(Box::new(CountingCredentials {
                refreshes: AtomicUsize::new(0),
            })),
            org: Some(ORG_UUID.to_string()),
            project: Some(PROJECT_UUID.to_string()),
            ..Default::default()
        },
    );
    client
        .ontology_entities()
        .expect("refreshed request must succeed");
    let requests = recorded.lock().expect("lock");
    assert_eq!(
        requests[1].header("authorization").as_deref(),
        Some("Bearer token-1")
    );
}

#[test]
fn persistent_401_is_auth_error() {
    let (host, _recorded) = spawn_stub(vec![(401, json!({})), (401, json!({}))]);
    let client = client_with(
        &host,
        Config {
            token: Some("bad".to_string()),
            org: Some(ORG_UUID.to_string()),
            project: Some(PROJECT_UUID.to_string()),
            ..Default::default()
        },
    );
    assert!(matches!(client.ontology_entities(), Err(Error::Auth(_))));
}

#[test]
fn graphql_errors_map_over_real_http() {
    let (host, _recorded) = spawn_stub(vec![(
        200,
        json!({"errors": [{"message": "nope", "extensions": {"code": "NOT_FOUND"}}]}),
    )]);
    let client = client_with(
        &host,
        Config {
            token: Some("tok".to_string()),
            org: Some(ORG_UUID.to_string()),
            project: Some(PROJECT_UUID.to_string()),
            ..Default::default()
        },
    );
    assert!(matches!(
        client.ontology_entities(),
        Err(Error::NotFound(_))
    ));
}

#[test]
fn resolves_workspace_slugs_to_ids() {
    let (host, recorded) = spawn_stub(vec![
        (
            200,
            json!({"data": {"MyOrganizations": [{"id": ORG_UUID, "slug": "acme"}]}}),
        ),
        (
            200,
            json!({"data": {"Projects": [{"id": PROJECT_UUID, "slug": "analytics"}]}}),
        ),
        (200, json!({"data": {"OntologyEntities": []}})),
    ]);
    let client = client_with(
        &host,
        Config {
            token: Some("tok".to_string()),
            org: Some("acme".to_string()),
            project: Some("analytics".to_string()),
            ..Default::default()
        },
    );
    client.ontology_entities().expect("entities");
    let requests = recorded.lock().expect("lock");
    let operations: Vec<String> = requests.iter().map(Recorded::operation).collect();
    assert_eq!(
        operations,
        ["MyOrganizations", "Projects", "OntologyEntities"]
    );
    assert_eq!(
        requests[2].header("x-whodb-org-id").as_deref(),
        Some(ORG_UUID)
    );
    assert_eq!(
        requests[2].header("x-whodb-project-id").as_deref(),
        Some(PROJECT_UUID)
    );
}

#[test]
fn api_key_discovers_workspace_via_my_workspace() {
    let (host, recorded) = spawn_stub(vec![
        (
            200,
            json!({"data": {"MyWorkspace": {"orgId": ORG_UUID, "projectId": PROJECT_UUID}}}),
        ),
        (200, json!({"data": {"OntologyEntities": []}})),
    ]);
    let client = client_with(
        &host,
        Config {
            api_key: Some("whodb_sk_test".to_string()),
            ..Default::default()
        },
    );
    client.ontology_entities().expect("entities");
    let requests = recorded.lock().expect("lock");
    assert_eq!(requests[0].operation(), "MyWorkspace");
    assert_eq!(
        requests[1].header("x-whodb-project-id").as_deref(),
        Some(PROJECT_UUID)
    );
}

#[test]
fn multi_grant_key_without_project_is_validation_error() {
    let (host, _recorded) = spawn_stub(vec![(
        200,
        json!({"data": {"MyWorkspace": {"orgId": ORG_UUID, "projectId": null}}}),
    )]);
    let client = client_with(
        &host,
        Config {
            api_key: Some("whodb_sk_test".to_string()),
            ..Default::default()
        },
    );
    assert!(matches!(
        client.ontology_entities(),
        Err(Error::Validation(_))
    ));
}

/// Recorded IPC calls: (path, body) per request.
type IpcCalls = Arc<Mutex<Vec<(String, Value)>>>;

/// Spawns a fake functions-runtime IPC server; returns address and calls.
fn spawn_ipc_stub() -> (String, IpcCalls) {
    let listener = TcpListener::bind("127.0.0.1:0").expect("bind ipc stub");
    let address = listener.local_addr().expect("ipc addr").to_string();
    let calls = Arc::new(Mutex::new(Vec::new()));
    let sink = Arc::clone(&calls);
    thread::spawn(move || {
        for stream in listener.incoming() {
            let mut stream = match stream {
                Ok(stream) => stream,
                Err(_) => return,
            };
            let mut raw = Vec::new();
            let mut chunk = [0u8; 4096];
            let (header_end, content_length) = loop {
                let read = stream.read(&mut chunk).unwrap_or(0);
                if read == 0 {
                    break (raw.len(), 0);
                }
                raw.extend_from_slice(&chunk[..read]);
                if let Some(position) = raw.windows(4).position(|w| w == b"\r\n\r\n") {
                    let headers = String::from_utf8_lossy(&raw[..position]).to_lowercase();
                    let length = headers
                        .lines()
                        .find_map(|line| line.strip_prefix("content-length:"))
                        .and_then(|value| value.trim().parse::<usize>().ok())
                        .unwrap_or(0);
                    break (position + 4, length);
                }
            };
            while raw.len() < header_end + content_length {
                let read = stream.read(&mut chunk).unwrap_or(0);
                if read == 0 {
                    break;
                }
                raw.extend_from_slice(&chunk[..read]);
            }
            let head = String::from_utf8_lossy(&raw[..header_end]).to_string();
            let path = head
                .lines()
                .next()
                .and_then(|line| line.split_whitespace().nth(1))
                .unwrap_or_default()
                .to_string();
            let body: Value = serde_json::from_slice(&raw[header_end..]).unwrap_or(json!({}));
            sink.lock().expect("calls lock").push((path.clone(), body));
            let payload = match path.as_str() {
                "/entities" => json!([{"id": "ent-1", "apiName": "User", "primaryKey": "id"}]),
                "/query" => json!({"columns": ["id"], "rows": [["u1"]], "total": 1}),
                "/create" | "/update" | "/delete" => json!({}),
                "/create_many" => json!(["id-1", "id-2"]),
                _ => json!({}),
            }
            .to_string();
            let response = format!(
                "HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nContent-Length: {}\r\nConnection: close\r\n\r\n{payload}",
                payload.len(),
            );
            let _ = stream.write_all(response.as_bytes());
        }
    });
    (address, calls)
}

#[test]
fn ipc_query_converts_where_json() {
    let (address, calls) = spawn_ipc_stub();
    let transport = IpcTransport::new(&address, "job-1", "ipc-token");
    let data = transport
        .execute(
            "OntologyQuery",
            "",
            variables(json!({"input": {"entity": "User", "whereJson": "{\"id\":{\"eq\":\"u1\"}}", "pageSize": 1, "sort": null}})),
        )
        .expect("query");
    assert!(data["OntologyQuery"].is_object());
    let calls = calls.lock().expect("lock");
    let (path, body) = &calls[0];
    assert_eq!(path, "/query");
    assert_eq!(body["where"], json!({"id": {"eq": "u1"}}));
    assert!(body.get("whereJson").is_none());
    assert!(body.get("sort").is_none());
}

#[test]
fn ipc_add_row_resolves_entity_and_converts_record_inputs() {
    let (address, calls) = spawn_ipc_stub();
    let transport = IpcTransport::new(&address, "job-1", "ipc-token");
    let data = transport
        .execute(
            "OntologyAddRow",
            "",
            variables(json!({"entityId": "ent-1", "values": [{"Key": "name", "Value": "Ada"}]})),
        )
        .expect("add row");
    assert_eq!(data["OntologyAddRow"]["Status"], json!(true));
    let calls = calls.lock().expect("lock");
    let (path, body) = &calls[1]; // call 0 is /entities
    assert_eq!(path, "/create");
    assert_eq!(body["entity"], "User");
    assert_eq!(body["data"], json!({"name": "Ada"}));
}

#[test]
fn ipc_update_row_splits_primary_key() {
    let (address, calls) = spawn_ipc_stub();
    let transport = IpcTransport::new(&address, "job-1", "ipc-token");
    transport
        .execute(
            "OntologyUpdateRow",
            "",
            variables(json!({"entityId": "ent-1", "values": [
                {"Key": "id", "Value": "u1"}, {"Key": "plan", "Value": "pro"}]})),
        )
        .expect("update");
    let calls = calls.lock().expect("lock");
    let (path, body) = &calls[1];
    assert_eq!(path, "/update");
    assert_eq!(body["pk"], "u1");
    assert_eq!(body["data"], json!({"plan": "pro"}));
}

#[test]
fn ipc_unsupported_operation_is_capability_error() {
    let (address, _calls) = spawn_ipc_stub();
    let transport = IpcTransport::new(&address, "job-1", "ipc-token");
    let result = transport.execute("RunSourceQuery", "", Map::new());
    assert!(matches!(result, Err(Error::TransportCapability(_))));
}
