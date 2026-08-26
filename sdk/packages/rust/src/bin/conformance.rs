//! Rust SDK fixture runner: reads fixtures as JSON lines on stdin, executes
//! each against the SDK with a mock transport, writes one JSON result line
//! per fixture to stdout. Protocol shared with the other language runners.

use serde_json::{json, Map, Value};
use std::io::{self, BufRead, Write};
use std::sync::Mutex;

use whodb_sdk::{Client, Config, Error, ListOptions, Transport};

/// Replays a fixture's scripted transcript, asserting each request.
struct MockTransport {
    steps: Vec<Value>,
    index: Mutex<usize>,
}

fn canonical_json(value: &Value) -> String {
    // serde_json object keys preserve insertion order; normalize by sorting.
    fn sort(value: &Value) -> Value {
        match value {
            Value::Object(map) => {
                let mut keys: Vec<&String> = map.keys().collect();
                keys.sort();
                let mut sorted = Map::new();
                for key in keys {
                    sorted.insert(key.clone(), sort(&map[key]));
                }
                Value::Object(sorted)
            }
            Value::Array(items) => Value::Array(items.iter().map(sort).collect()),
            other => other.clone(),
        }
    }
    sort(value).to_string()
}

impl Transport for MockTransport {
    fn execute(
        &self,
        operation: &str,
        _document: &str,
        variables: Map<String, Value>,
    ) -> whodb_sdk::Result<Value> {
        let mut index = self.index.lock().unwrap();
        let step = self.steps.get(*index).ok_or_else(|| {
            Error::Transport(format!("unexpected call #{}: {}", *index + 1, operation))
        })?;
        *index += 1;
        let expect = step
            .get("expectRequest")
            .cloned()
            .unwrap_or_else(|| json!({}));
        if let Some(expected) = expect.get("operation").and_then(Value::as_str) {
            if expected != operation {
                return Err(Error::Transport(format!(
                    "expected operation {expected}, got {operation}"
                )));
            }
        }
        let variables_value = Value::Object(variables);
        let assert_contains =
            |actual: &Value, expected: &Value, path: &str| -> whodb_sdk::Result<()> {
                if canonical_json(actual) != canonical_json(expected) {
                    return Err(Error::Transport(format!(
                        "mismatch at {path}: expected {expected}, got {actual}"
                    )));
                }
                Ok(())
            };
        for source in ["variables", "variablesContain"] {
            if let Some(expected_map) = expect.get(source).and_then(Value::as_object) {
                for (key, expected) in expected_map {
                    let actual = variables_value.get(key).cloned().unwrap_or(Value::Null);
                    assert_contains(&actual, expected, &format!("variables.{key}"))?;
                }
            }
        }
        if let Some(expected_map) = expect.get("inputContains").and_then(Value::as_object) {
            let input = variables_value
                .get("input")
                .cloned()
                .unwrap_or_else(|| json!({}));
            for (key, expected) in expected_map {
                let actual = input.get(key).cloned().unwrap_or(Value::Null);
                assert_contains(&actual, expected, &format!("input.{key}"))?;
            }
        }
        let response = step.get("response").cloned().unwrap_or_else(|| json!({}));
        if let Some(errors) = response.get("errors").and_then(Value::as_array) {
            if !errors.is_empty() {
                let first = &errors[0];
                let message = first
                    .get("message")
                    .and_then(Value::as_str)
                    .unwrap_or("")
                    .to_string();
                let code = first
                    .get("extensions")
                    .and_then(|x| x.get("code"))
                    .and_then(Value::as_str)
                    .unwrap_or("");
                return Err(match code {
                    "UNAUTHENTICATED" | "FORBIDDEN" => Error::Auth(message),
                    "NOT_FOUND" => Error::NotFound(message),
                    "BAD_USER_INPUT" | "GRAPHQL_VALIDATION_FAILED" => Error::Validation(message),
                    _ => Error::Platform {
                        message,
                        code: code.to_string(),
                    },
                });
            }
        }
        Ok(response.get("data").cloned().unwrap_or(Value::Null))
    }
}

fn error_type(error: &Error) -> &'static str {
    match error {
        Error::Auth(_) => "AuthError",
        Error::NotFound(_) => "NotFoundError",
        Error::Validation(_) => "ValidationError",
        Error::Version(_) => "WhoDBVersionError",
        Error::CliCredentials(_) => "CliCredentialsError",
        Error::TransportCapability(_) => "TransportCapabilityError",
        Error::Platform { .. } => "PlatformError",
        Error::Transport(_) => "TransportError",
    }
}

fn list_options_from(args: &[Value]) -> ListOptions {
    let options = args.first().and_then(Value::as_object);
    ListOptions {
        where_filter: options.and_then(|o| o.get("where")).cloned(),
        sort: None,
        page_size: options
            .and_then(|o| o.get("pageSize"))
            .and_then(Value::as_u64)
            .unwrap_or(0) as usize,
        offset: 0,
    }
}

fn rows_to_value(rows: Vec<whodb_sdk::Row>) -> Value {
    Value::Array(rows.into_iter().map(Value::Object).collect())
}

fn run_fixture(fixture: &Value) -> Value {
    let name = fixture
        .get("name")
        .and_then(Value::as_str)
        .unwrap_or("(unnamed)");
    let transcript = fixture
        .get("transcript")
        .and_then(Value::as_array)
        .cloned()
        .unwrap_or_default();
    let call = fixture.get("call").cloned().unwrap_or_else(|| json!({}));
    let fail = |reason: String| json!({"name": name, "pass": false, "reason": reason});

    let client = match Client::new(Config {
        org: Some("00000000-0000-0000-0000-000000000001".to_string()),
        project: Some("proj-1".to_string()),
        transport: Some(Box::new(MockTransport {
            steps: transcript,
            index: Mutex::new(0),
        })),
        ..Default::default()
    }) {
        Ok(client) => client,
        Err(error) => return fail(error.to_string()),
    };
    if call.get("domain").and_then(Value::as_str) != Some("ontology") {
        return fail("unsupported fixture domain".to_string());
    }
    let handle = client.ontology(call.get("handle").and_then(Value::as_str).unwrap_or(""));
    let args = call
        .get("args")
        .and_then(Value::as_array)
        .cloned()
        .unwrap_or_default();
    let method = call.get("method").and_then(Value::as_str).unwrap_or("");
    let collect_pages = call.get("collect").and_then(Value::as_str) == Some("pages");

    let outcome: whodb_sdk::Result<Value> = (|| match method {
        "get" => {
            let pk = args.first().and_then(Value::as_str).unwrap_or("");
            Ok(handle.get(pk)?.map(Value::Object).unwrap_or(Value::Null))
        }
        "describe" => handle.describe(),
        "list" => {
            let options = list_options_from(&args);
            if collect_pages {
                let mut pages: Vec<Value> = Vec::new();
                handle.pages(&options, |rows| {
                    pages.push(rows_to_value(rows));
                    true
                })?;
                Ok(Value::Array(pages))
            } else {
                Ok(rows_to_value(handle.list(&options)?))
            }
        }
        "create" => {
            let values = args
                .first()
                .and_then(Value::as_object)
                .cloned()
                .unwrap_or_default();
            handle.create(&values)?;
            Ok(Value::Null)
        }
        "createMany" => {
            let rows: Vec<Map<String, Value>> = args
                .first()
                .and_then(Value::as_array)
                .map(|entries| {
                    entries
                        .iter()
                        .filter_map(|entry| entry.as_object().cloned())
                        .collect()
                })
                .unwrap_or_default();
            let idempotency_key = args
                .get(1)
                .and_then(Value::as_object)
                .and_then(|options| options.get("idempotencyKey"))
                .and_then(Value::as_str);
            handle.create_many(&rows, idempotency_key)
        }
        "update" => {
            let pk = args.first().and_then(Value::as_str).unwrap_or("");
            let values = args
                .get(1)
                .and_then(Value::as_object)
                .cloned()
                .unwrap_or_default();
            handle.update(pk, &values)?;
            Ok(Value::Null)
        }
        "delete" => {
            let pk = args.first().and_then(Value::as_str).unwrap_or("");
            handle.delete(pk)?;
            Ok(Value::Null)
        }
        other => Err(Error::Transport(format!(
            "unsupported fixture method: {other}"
        ))),
    })();

    let expect_error = fixture.get("expectError");
    match outcome {
        Err(error) => {
            if let Some(expected) = expect_error {
                let want_type = expected.get("type").and_then(Value::as_str).unwrap_or("");
                let want_message = expected
                    .get("messageContains")
                    .and_then(Value::as_str)
                    .unwrap_or("");
                if error_type(&error) == want_type && error.to_string().contains(want_message) {
                    json!({"name": name, "pass": true})
                } else {
                    fail(format!(
                        "expected {want_type}({want_message}), got {}: {error}",
                        error_type(&error)
                    ))
                }
            } else {
                fail(error.to_string())
            }
        }
        Ok(value) => {
            if let Some(expected) = expect_error {
                return fail(format!(
                    "expected {}, got success",
                    expected.get("type").and_then(Value::as_str).unwrap_or("")
                ));
            }
            let want = fixture.get("expectResult").cloned().unwrap_or(Value::Null);
            if canonical_json(&value) == canonical_json(&want) {
                json!({"name": name, "pass": true})
            } else {
                fail(format!("result mismatch:\n  want {want}\n  got  {value}"))
            }
        }
    }
}

fn main() {
    let stdin = io::stdin();
    let stdout = io::stdout();
    let mut out = stdout.lock();
    for line in stdin.lock().lines() {
        let Ok(line) = line else { break };
        if line.trim().is_empty() {
            continue;
        }
        let result = match serde_json::from_str::<Value>(&line) {
            Ok(fixture) => run_fixture(&fixture),
            Err(error) => {
                json!({"name": "(parse error)", "pass": false, "reason": error.to_string()})
            }
        };
        let _ = writeln!(out, "{result}");
        let _ = out.flush();
    }
}
