//! IPC transport: runs the same facades inside the WhoDB Functions runtime.
//!
//! Only the ontology operations are available; others return
//! Error::TransportCapability. GraphQL entity IDs resolve to apiNames via a
//! cached /entities call. Docker exposes the IPC server on a unix socket —
//! not supported by this transport yet (ureq is TCP-only); the K8s runtime's
//! TCP address works. Rust functions in the Docker runtime keep the legacy
//! helpers until unix-socket support lands here.

use serde_json::{json, Map, Value};
use std::sync::Mutex;

use crate::error::{Error, Result};
use crate::transport::Transport;

/// Functions-runtime IPC transport (TCP).
pub struct IpcTransport {
    address: String,
    job_id: String,
    token: String,
    agent: ureq::Agent,
    entities: Mutex<Option<Vec<Value>>>,
}

impl IpcTransport {
    /// Builds the transport from the runtime's WHODB_IPC_* env vars.
    pub fn from_env() -> Self {
        IpcTransport {
            address: std::env::var("WHODB_IPC_ADDRESS").unwrap_or_default(),
            job_id: std::env::var("WHODB_JOB_ID").unwrap_or_default(),
            token: std::env::var("WHODB_IPC_TOKEN").unwrap_or_default(),
            agent: ureq::Agent::new_with_defaults(),
            entities: Mutex::new(None),
        }
    }

    fn post(&self, path: &str, body: &Value) -> Result<Value> {
        if self.address.starts_with('/') {
            return Err(Error::TransportCapability(
                "unix-socket IPC is not supported by the Rust SDK yet — use the TCP runtime"
                    .to_string(),
            ));
        }
        let url = format!("http://{}{}", self.address, path);
        let mut response = self
            .agent
            .post(&url)
            .header("Content-Type", "application/json")
            .header("X-Job-ID", &self.job_id)
            .header("Authorization", &self.token)
            .send_json(body)
            .map_err(|error| match error {
                ureq::Error::StatusCode(code) => Error::Platform {
                    message: format!("IPC request {path} failed with HTTP {code}"),
                    code: format!("IPC_{code}"),
                },
                other => Error::Transport(other.to_string()),
            })?;
        response
            .body_mut()
            .read_json::<Value>()
            .map_err(|_| Error::Platform {
                message: format!("IPC request {path} returned invalid JSON"),
                code: "IPC_INVALID_JSON".to_string(),
            })
    }

    fn entity_list(&self) -> Result<Vec<Value>> {
        let mut cache = self.entities.lock().expect("entities lock poisoned");
        if let Some(entities) = cache.as_ref() {
            return Ok(entities.clone());
        }
        let raw = self.post("/entities", &json!({}))?;
        let entities = raw.as_array().cloned().unwrap_or_default();
        *cache = Some(entities.clone());
        Ok(entities)
    }

    fn entity_field(&self, entity_id: &str, field: &str) -> Result<String> {
        for entity in self.entity_list()? {
            if entity.get("id").and_then(Value::as_str) == Some(entity_id) {
                return Ok(entity
                    .get(field)
                    .and_then(Value::as_str)
                    .unwrap_or_default()
                    .to_string());
            }
        }
        Err(Error::NotFound(format!(
            "ontology entity {entity_id} not found in this function's scope"
        )))
    }
}

fn record_inputs_to_data(values: Option<&Value>) -> Map<String, Value> {
    let mut data = Map::new();
    if let Some(entries) = values.and_then(Value::as_array) {
        for entry in entries {
            if let Some(key) = entry.get("Key").and_then(Value::as_str) {
                data.insert(
                    key.to_string(),
                    entry.get("Value").cloned().unwrap_or(Value::Null),
                );
            }
        }
    }
    data
}

impl Transport for IpcTransport {
    fn execute(
        &self,
        operation: &str,
        _document: &str,
        variables: Map<String, Value>,
    ) -> Result<Value> {
        let payload = self.dispatch(operation, &variables)?;
        Ok(json!({ operation: payload }))
    }
}

impl IpcTransport {
    fn dispatch(&self, operation: &str, variables: &Map<String, Value>) -> Result<Value> {
        let entity_of = |key: &str| -> Result<String> {
            let entity_id = variables
                .get(key)
                .and_then(Value::as_str)
                .unwrap_or_default();
            self.entity_field(entity_id, "apiName")
        };
        match operation {
            "OntologyEntities" => Ok(Value::Array(self.entity_list()?)),
            "OntologyQuery" => {
                let input = variables
                    .get("input")
                    .and_then(Value::as_object)
                    .cloned()
                    .unwrap_or_default();
                let mut body = Map::new();
                for (key, value) in input {
                    if value.is_null() {
                        continue;
                    }
                    if key == "whereJson" {
                        if let Some(where_json) = value.as_str() {
                            if let Ok(parsed) = serde_json::from_str::<Value>(where_json) {
                                body.insert("where".to_string(), parsed);
                            }
                        }
                        continue;
                    }
                    body.insert(key, value);
                }
                self.post("/query", &Value::Object(body))
            }
            "OntologyDescribe" => {
                let input = variables.get("input").cloned().unwrap_or_else(|| json!({}));
                self.post("/describe", &input)
            }
            "OntologyAddRow" => {
                let entity = entity_of("entityId")?;
                self.post(
                    "/create",
                    &json!({"entity": entity, "data": record_inputs_to_data(variables.get("values"))}),
                )?;
                Ok(json!({"Status": true}))
            }
            "OntologyAddRows" => {
                let entity = entity_of("entityId")?;
                let rows: Vec<Value> = variables
                    .get("rows")
                    .and_then(Value::as_array)
                    .map(|entries| {
                        entries
                            .iter()
                            .map(|entry| Value::Object(record_inputs_to_data(entry.get("values"))))
                            .collect()
                    })
                    .unwrap_or_default();
                let ids = self.post(
                    "/create_many",
                    &json!({"entity": entity, "rows": rows, "idempotencyKey": variables.get("idempotencyKey")}),
                )?;
                let count = ids.as_array().map(Vec::len).unwrap_or(0);
                Ok(json!({"inserted": count, "ids": ids}))
            }
            "OntologyUpdateRow" => {
                let entity_id = variables
                    .get("entityId")
                    .and_then(Value::as_str)
                    .unwrap_or_default();
                let entity = self.entity_field(entity_id, "apiName")?;
                let pk_key = self
                    .entity_field(entity_id, "primaryKey")
                    .unwrap_or_default();
                let mut data = record_inputs_to_data(variables.get("values"));
                let pk = data
                    .remove(&pk_key)
                    .and_then(|value| value.as_str().map(str::to_string))
                    .unwrap_or_default();
                self.post(
                    "/update",
                    &json!({"entity": entity, "pk": pk, "data": data}),
                )?;
                Ok(json!({"Status": true}))
            }
            "OntologyDeleteRow" => {
                let entity_id = variables
                    .get("entityId")
                    .and_then(Value::as_str)
                    .unwrap_or_default();
                let entity = self.entity_field(entity_id, "apiName")?;
                let pk_key = self
                    .entity_field(entity_id, "primaryKey")
                    .unwrap_or_default();
                let data = record_inputs_to_data(variables.get("values"));
                let pk = data
                    .get(&pk_key)
                    .and_then(Value::as_str)
                    .unwrap_or_default();
                self.post("/delete", &json!({"entity": entity, "pk": pk}))?;
                Ok(json!({"Status": true}))
            }
            "OntologyFollowLink" => {
                let entity = entity_of("entityId")?;
                self.post(
                    "/follow_link",
                    &json!({
                        "entity": entity,
                        "pk": variables.get("pk"),
                        "link": variables.get("linkApiName"),
                        "pageSize": variables.get("pageSize"),
                        "offset": variables.get("pageOffset"),
                    }),
                )
            }
            other => Err(Error::TransportCapability(format!(
                "{other} is not available inside the function runtime in v1"
            ))),
        }
    }
}
