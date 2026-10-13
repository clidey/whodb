// Copyright 2026 Clidey, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//! Rust SDK fixture runner: reads fixtures as JSON lines on stdin, executes
//! each against the SDK with a mock transport, writes one JSON result line
//! per fixture to stdout. Protocol shared with the other language runners.

use serde_json::{json, Map, Value};
use std::io::{self, BufRead, Write};
use std::sync::Mutex;

use std::time::Duration;
use whodb_sdk::{
    ActionOptions, Client, Config, DatasetRowsOptions, Error, FunctionTestOptions, ListOptions,
    RowsPage, SourceObjectsOptions, SourceRowsOptions, Transport, WaitOptions,
};

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
    let domain = call.get("domain").and_then(Value::as_str).unwrap_or("");
    let handle_name = call.get("handle").and_then(Value::as_str).unwrap_or("");
    let args = call
        .get("args")
        .and_then(Value::as_array)
        .cloned()
        .unwrap_or_default();
    let method = call.get("method").and_then(Value::as_str).unwrap_or("");
    let collect_pages = call.get("collect").and_then(Value::as_str) == Some("pages");

    let outcome: whodb_sdk::Result<Value> = if domain == "ontology" {
        run_ontology(&client, handle_name, method, &args, collect_pages)
    } else {
        run_other(&client, domain, handle_name, method, &args)
    };
    compare(name, fixture, outcome)
}

fn run_ontology(
    client: &Client,
    handle_name: &str,
    method: &str,
    args: &[Value],
    collect_pages: bool,
) -> whodb_sdk::Result<Value> {
    let handle = client.ontology(handle_name);
    (|| match method {
        "get" => {
            let pk = args.first().and_then(Value::as_str).unwrap_or("");
            Ok(handle.get(pk)?.map(Value::Object).unwrap_or(Value::Null))
        }
        "describe" => handle.describe(),
        "list" => {
            let options = list_options_from(args);
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
        "capabilities" => handle.capabilities(args.first().and_then(Value::as_str)),
        "previewAction" => {
            let action = args.first().and_then(Value::as_str).unwrap_or("");
            let record_key = args.get(1).and_then(Value::as_str);
            let values = args
                .get(2)
                .and_then(Value::as_object)
                .cloned()
                .unwrap_or_default();
            handle.preview_action(action, record_key, &values)
        }
        "action" => {
            let action = args.first().and_then(Value::as_str).unwrap_or("");
            let record_key = args.get(1).and_then(Value::as_str);
            let values = args
                .get(2)
                .and_then(Value::as_object)
                .cloned()
                .unwrap_or_default();
            let options = args.get(3).and_then(Value::as_object);
            handle.action(
                action,
                record_key,
                &values,
                &ActionOptions {
                    expected_version: options
                        .and_then(|item| item.get("expectedVersion"))
                        .and_then(Value::as_i64),
                    idempotency_key: options
                        .and_then(|item| item.get("idempotencyKey"))
                        .and_then(Value::as_str)
                        .map(str::to_string),
                },
            )
        }
        "actionExecutions" => handle.action_executions(
            args.first().and_then(Value::as_str).unwrap_or(""),
            args.get(1).and_then(Value::as_u64).unwrap_or(50) as usize,
        ),
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
        "promote" => handle.promote(opt_str_arg(args, 0)),
        "versions" => Ok(Value::Array(handle.versions()?)),
        "followLink" => {
            let pk = args.first().and_then(Value::as_str).unwrap_or("");
            let link = args.get(1).and_then(Value::as_str).unwrap_or("");
            let options = args.get(2).and_then(Value::as_object);
            let page_size = options
                .and_then(|o| o.get("pageSize"))
                .and_then(Value::as_u64)
                .unwrap_or(0) as usize;
            let offset = options
                .and_then(|o| o.get("offset"))
                .and_then(Value::as_u64)
                .unwrap_or(0) as usize;
            let rows = handle.follow_link(pk, link, page_size, offset)?;
            Ok(Value::Array(rows.into_iter().map(Value::Object).collect()))
        }
        other => Err(Error::Transport(format!(
            "unsupported fixture method: {other}"
        ))),
    })()
}

fn str_arg(args: &[Value], index: usize) -> &str {
    args.get(index).and_then(Value::as_str).unwrap_or("")
}

fn opt_str_arg(args: &[Value], index: usize) -> Option<&str> {
    args.get(index).and_then(Value::as_str)
}

fn map_arg(args: &[Value], index: usize) -> Map<String, Value> {
    args.get(index)
        .and_then(Value::as_object)
        .cloned()
        .unwrap_or_default()
}

fn string_list_arg(args: &[Value], index: usize) -> Option<Vec<String>> {
    args.get(index).and_then(Value::as_array).map(|items| {
        items
            .iter()
            .map(|item| item.as_str().unwrap_or("").to_string())
            .collect()
    })
}

fn usize_option(options: &Map<String, Value>, key: &str) -> usize {
    options.get(key).and_then(Value::as_u64).unwrap_or(0) as usize
}

fn wait_options(options: &Map<String, Value>) -> WaitOptions {
    let mut wait = WaitOptions::default();
    if let Some(timeout) = options.get("timeoutMs").and_then(Value::as_u64) {
        wait.timeout = Duration::from_millis(timeout);
    }
    if let Some(poll) = options.get("pollMs").and_then(Value::as_u64) {
        wait.poll = Duration::from_millis(poll);
    }
    wait
}

fn page_value(page: RowsPage) -> Value {
    json!({"rows": rows_to_value(page.rows), "totalCount": page.total_count})
}

fn unit(result: whodb_sdk::Result<()>) -> whodb_sdk::Result<Value> {
    result.map(|_| Value::Null)
}

fn unsupported(domain: &str, method: &str) -> whodb_sdk::Result<Value> {
    Err(Error::Transport(format!(
        "unsupported fixture call: {domain}.{method}"
    )))
}

/// Dispatches non-ontology fixture calls.
fn run_other(
    client: &Client,
    domain: &str,
    handle_name: &str,
    method: &str,
    args: &[Value],
) -> whodb_sdk::Result<Value> {
    match domain {
        "dataset" => {
            let handle = client.dataset(handle_name);
            match method {
                "meta" => handle.meta(),
                "rows" => {
                    let options = map_arg(args, 0);
                    handle
                        .rows(DatasetRowsOptions {
                            page_size: usize_option(&options, "pageSize"),
                            page_offset: usize_option(&options, "pageOffset"),
                        })
                        .map(page_value)
                }
                "promote" => handle.promote(opt_str_arg(args, 0)),
                "versions" => Ok(Value::Array(handle.versions()?)),
                _ => unsupported(domain, method),
            }
        }
        "source" => {
            let handle = client.source(handle_name);
            match method {
                "objects" => {
                    let options = map_arg(args, 0);
                    Ok(Value::Array(handle.objects(&SourceObjectsOptions {
                        parent: options.get("parent").cloned(),
                        page_size: usize_option(&options, "pageSize"),
                        page_offset: usize_option(&options, "pageOffset"),
                    })?))
                }
                "columns" => Ok(Value::Array(
                    handle.columns(args.first().unwrap_or(&Value::Null))?,
                )),
                "rows" => {
                    let options = map_arg(args, 1);
                    let rows = handle.rows(
                        args.first().unwrap_or(&Value::Null),
                        &SourceRowsOptions {
                            where_filter: options.get("where").cloned(),
                            sort: options.get("sort").cloned(),
                            page_size: usize_option(&options, "pageSize"),
                            offset: usize_option(&options, "offset"),
                        },
                    )?;
                    Ok(rows_to_value(rows))
                }
                _ => unsupported(domain, method),
            }
        }
        "transforms" => {
            let handle = client.transforms();
            match method {
                "list" => Ok(Value::Array(handle.list()?)),
                "create" => handle.create(&map_arg(args, 0)),
                "preview" => handle.preview(str_arg(args, 0)).map(page_value),
                _ => unsupported(domain, method),
            }
        }
        "transform" => {
            let handle = client.transform(handle_name);
            match method {
                "meta" => handle.meta(),
                "runs" => Ok(Value::Array(
                    handle.runs(usize_option(&map_arg(args, 0), "limit"))?,
                )),
                "run" => handle.run(),
                "cancelRun" => unit(handle.cancel_run(opt_str_arg(args, 0))),
                "wait" => handle.wait(str_arg(args, 0), wait_options(&map_arg(args, 1))),
                "update" => handle.update(&map_arg(args, 0)),
                "delete" => unit(handle.delete()),
                "promote" => handle.promote(opt_str_arg(args, 0)),
                "versions" => Ok(Value::Array(handle.versions()?)),
                _ => unsupported(domain, method),
            }
        }
        "functions" => {
            let handle = client.functions();
            match method {
                "list" => Ok(Value::Array(handle.list()?)),
                "create" => handle.create(&map_arg(args, 0)),
                _ => unsupported(domain, method),
            }
        }
        "function" => {
            let handle = client.function(handle_name);
            let input = args.first().cloned().unwrap_or(Value::Null);
            match method {
                "meta" => handle.meta(),
                "execute" => handle.execute(&input, string_list_arg(args, 1).as_deref()),
                "test" => {
                    let options = map_arg(args, 1);
                    handle.test(
                        &input,
                        &FunctionTestOptions {
                            files: options.get("files").cloned(),
                            input_file_ids: options
                                .get("inputFileIds")
                                .and_then(Value::as_array)
                                .map(|ids| {
                                    ids.iter()
                                        .map(|id| id.as_str().unwrap_or("").to_string())
                                        .collect()
                                }),
                        },
                    )
                }
                "startRun" => handle.start_run(
                    &input,
                    map_arg(args, 1)
                        .get("useActiveVersion")
                        .and_then(Value::as_bool),
                ),
                "run" => handle.run(str_arg(args, 0)),
                "runs" => Ok(Value::Array(
                    handle.runs(usize_option(&map_arg(args, 0), "limit"))?,
                )),
                "wait" => handle.wait(str_arg(args, 0), wait_options(&map_arg(args, 1))),
                "cancelRun" => handle.cancel_run(str_arg(args, 0)),
                "deploy" => unit(handle.deploy()),
                "redeploy" => unit(handle.redeploy()),
                "update" => handle.update(&map_arg(args, 0)),
                "delete" => unit(handle.delete()),
                "promote" => handle.promote(opt_str_arg(args, 0)),
                "versions" => Ok(Value::Array(handle.versions()?)),
                _ => unsupported(domain, method),
            }
        }
        "versions" => {
            let handle = client.versions();
            let version = args.get(2).and_then(Value::as_i64).unwrap_or(0);
            match method {
                "list" => Ok(Value::Array(
                    handle.list(str_arg(args, 0), str_arg(args, 1))?,
                )),
                "active" => handle.active(str_arg(args, 0), str_arg(args, 1)),
                "activeAll" => Ok(Value::Array(handle.active_all()?)),
                "promote" => {
                    handle.promote(str_arg(args, 0), str_arg(args, 1), opt_str_arg(args, 2))
                }
                "rollback" => handle.rollback(str_arg(args, 0), str_arg(args, 1), version),
                "restoreToDraft" => {
                    handle.restore_to_draft(str_arg(args, 0), str_arg(args, 1), version)
                }
                _ => unsupported(domain, method),
            }
        }
        "serviceAccounts" => {
            let handle = client.service_accounts();
            match method {
                "list" => Ok(Value::Array(handle.list()?)),
                "create" => handle.create(str_arg(args, 0), opt_str_arg(args, 1)),
                "createKey" => handle.create_key(str_arg(args, 0), opt_str_arg(args, 1)),
                "revokeKey" => unit(handle.revoke_key(str_arg(args, 0))),
                "delete" => unit(handle.delete(str_arg(args, 0))),
                _ => unsupported(domain, method),
            }
        }
        "lineageProviders" => {
            let handle = client.lineage_providers();
            match method {
                "list" => Ok(Value::Array(handle.list()?)),
                "save" => handle.save(&map_arg(args, 0)),
                "delete" => unit(handle.delete(str_arg(args, 0))),
                _ => unsupported(domain, method),
            }
        }
        "sharing" => {
            let handle = client.sharing();
            match method {
                "shared" => Ok(Value::Array(handle.shared()?)),
                "discover" => Ok(Value::Array(handle.discover()?)),
                "references" => Ok(Value::Array(handle.references()?)),
                "referenceImpact" => handle.reference_impact(str_arg(args, 0)),
                "share" => handle.share(
                    str_arg(args, 0),
                    str_arg(args, 1),
                    string_list_arg(args, 2).as_deref(),
                ),
                "unshare" => unit(handle.unshare(str_arg(args, 0))),
                "createReference" => {
                    handle.create_reference(str_arg(args, 0), str_arg(args, 1), str_arg(args, 2))
                }
                "removeReference" => unit(handle.remove_reference(str_arg(args, 0))),
                _ => unsupported(domain, method),
            }
        }
        _ => unsupported(domain, method),
    }
}

/// Compares a call outcome with the fixture's expectation.
fn compare(name: &str, fixture: &Value, outcome: whodb_sdk::Result<Value>) -> Value {
    let fail = |reason: String| json!({"name": name, "pass": false, "reason": reason});
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
