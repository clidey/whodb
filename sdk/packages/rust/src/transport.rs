//! Transports: how prepared operations reach the platform. The generated
//! core and the facades never speak HTTP directly.

use serde_json::{json, Map, Value};
use std::sync::RwLock;

use crate::auth::CredentialProvider;
use crate::error::{map_graphql_errors, Error, Result};
use crate::SDK_VERSION;

/// Executes one platform operation, returning the GraphQL data object.
pub trait Transport: Send + Sync {
    /// Runs the operation and returns the `data` object keyed by operation name.
    fn execute(
        &self,
        operation: &str,
        document: &str,
        variables: Map<String, Value>,
    ) -> Result<Value>;
}

const RETRYABLE_STATUS: &[u16] = &[502, 503, 504];

/// Default GraphQL-over-HTTP transport (POST /api/query).
pub(crate) struct HttpTransport {
    endpoint: String,
    credentials: Box<dyn CredentialProvider>,
    workspace: RwLock<(String, String)>,
    agent: ureq::Agent,
}

impl HttpTransport {
    pub(crate) fn new(host: &str, credentials: Box<dyn CredentialProvider>) -> Self {
        HttpTransport {
            endpoint: format!("{}/api/query", host.trim_end_matches('/')),
            credentials,
            workspace: RwLock::new((String::new(), String::new())),
            agent: ureq::Agent::new_with_defaults(),
        }
    }

    pub(crate) fn set_workspace(&self, org_id: &str, project_id: &str) {
        let mut workspace = self.workspace.write().expect("workspace lock poisoned");
        *workspace = (org_id.to_string(), project_id.to_string());
    }

    fn post(&self, body: &Value) -> Result<(u16, Value)> {
        let token = self.credentials.token()?;
        let (org_id, project_id) = self
            .workspace
            .read()
            .expect("workspace lock poisoned")
            .clone();
        let mut request = self
            .agent
            .post(&self.endpoint)
            .header("Content-Type", "application/json")
            .header("Authorization", &format!("Bearer {token}"))
            .header("User-Agent", &format!("clidey-whodb-rust/{SDK_VERSION}"));
        if !org_id.is_empty() {
            request = request.header("X-Whodb-Org-Id", &org_id);
        }
        if !project_id.is_empty() {
            request = request.header("X-Whodb-Project-Id", &project_id);
        }
        match request.send_json(body) {
            Ok(mut response) => {
                let status = response.status().as_u16();
                let payload = response
                    .body_mut()
                    .read_json::<Value>()
                    .unwrap_or(Value::Null);
                Ok((status, payload))
            }
            Err(ureq::Error::StatusCode(code)) => Ok((code, Value::Null)),
            Err(error) => Err(Error::Transport(error.to_string())),
        }
    }
}

impl Transport for HttpTransport {
    fn execute(
        &self,
        operation: &str,
        document: &str,
        variables: Map<String, Value>,
    ) -> Result<Value> {
        let body = json!({"query": document, "variables": Value::Object(variables)});
        let (mut status, mut payload) = self.post(&body)?;
        if status == 401 {
            self.credentials.refresh();
            (status, payload) = self.post(&body)?;
        } else if RETRYABLE_STATUS.contains(&status) {
            (status, payload) = self.post(&body)?;
        }
        if status == 401 {
            return Err(Error::Auth(
                "check your API key or run: whodb login".to_string(),
            ));
        }
        if status >= 400 {
            return Err(Error::Platform {
                message: format!("platform request failed with HTTP {status}"),
                code: format!("HTTP_{status}"),
            });
        }
        if let Some(errors) = payload.get("errors").and_then(Value::as_array) {
            if !errors.is_empty() {
                return Err(map_graphql_errors(errors));
            }
        }
        match payload.get("data") {
            Some(data) if !data.is_null() => Ok(data.clone()),
            _ => Err(Error::Platform {
                message: format!("empty response for {operation}"),
                code: "EMPTY_RESPONSE".to_string(),
            }),
        }
    }
}
