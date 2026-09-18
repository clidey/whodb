//! The WhoDB client: entry point of the whodb-sdk crate.

use serde_json::{Map, Value};
use std::sync::{Mutex, OnceLock};

use crate::auth::{CliCredentials, CredentialProvider, StaticCredentials};
use crate::error::{interpret_server_error, Error, Result};
use crate::gen::operations as ops;
use crate::manifest_check::warn_if_flagged;
use crate::ontology::OntologyHandle;
use crate::transport::{HttpTransport, Transport};

/// The hosted WhoDB platform endpoint.
pub const DEFAULT_HOST: &str = "https://app.whodb.com";

fn is_uuid(text: &str) -> bool {
    let bytes = text.as_bytes();
    bytes.len() == 36
        && bytes[8] == b'-'
        && bytes[13] == b'-'
        && bytes[18] == b'-'
        && bytes[23] == b'-'
        && text
            .chars()
            .all(|character| character == '-' || character.is_ascii_hexdigit())
}

/// Configures the WhoDB client. Default values fall back to the standard
/// precedence: WHODB_API_KEY env → the whodb CLI's stored login; and inside
/// the Functions runtime, the IPC transport is auto-detected.
#[derive(Default)]
pub struct Config {
    /// Platform API key (whodb_sk_...). Highest-precedence credential.
    pub api_key: Option<String>,
    /// Raw OIDC access token.
    pub token: Option<String>,
    /// Fully custom credential provider.
    pub credentials: Option<Box<dyn CredentialProvider>>,
    /// Organization slug or ID. Optional with an API key.
    pub org: Option<String>,
    /// Project slug or ID. Optional with a single-grant API key.
    pub project: Option<String>,
    /// Platform host (default https://app.whodb.com).
    pub host: Option<String>,
    /// Wire transport override (IPC, tests). When set, org/project inputs
    /// are taken as IDs verbatim.
    pub transport: Option<Box<dyn Transport>>,
}

/// The WhoDB platform client.
pub struct Client {
    transport: Box<dyn Transport>,
    http: Option<HttpTransportHandle>,
    cli: Option<std::sync::Arc<CliCredentials>>,
    using_api_key: bool,
    skip_resolution: bool,
    org_input: String,
    project_input: String,
    workspace: OnceLock<std::result::Result<(String, String), String>>,
}

/// Shared handle for stamping workspace headers post-construction.
struct HttpTransportHandle(std::sync::Arc<HttpTransport>);

struct ArcTransport(std::sync::Arc<HttpTransport>);

impl Transport for ArcTransport {
    fn execute(
        &self,
        operation: &str,
        document: &str,
        variables: Map<String, Value>,
    ) -> Result<Value> {
        self.0.execute(operation, document, variables)
    }
}

fn env(name: &str) -> String {
    std::env::var(name).unwrap_or_default()
}

impl Client {
    /// Creates a client, applying the credential precedence: explicit Config
    /// → WHODB_API_KEY → CLI credentials; IPC auto-detected in the runtime.
    pub fn new(config: Config) -> Result<Client> {
        let org_input = config.org.unwrap_or_else(|| env("WHODB_ORG"));
        let project_input = config.project.unwrap_or_else(|| env("WHODB_PROJECT"));
        let mut transport_override = config.transport;
        if transport_override.is_none()
            && config.credentials.is_none()
            && config.api_key.is_none()
            && config.token.is_none()
            && env("WHODB_API_KEY").is_empty()
            && !env("WHODB_IPC_TOKEN").is_empty()
        {
            transport_override = Some(Box::new(crate::transport_ipc::IpcTransport::from_env()));
        }
        if let Some(transport) = transport_override {
            return Ok(Client {
                transport,
                http: None,
                cli: None,
                using_api_key: false,
                skip_resolution: true,
                org_input,
                project_input,
                workspace: OnceLock::new(),
            });
        }
        let mut using_api_key = false;
        let mut cli: Option<std::sync::Arc<CliCredentials>> = None;
        let credentials: Box<dyn CredentialProvider> = if let Some(custom) = config.credentials {
            custom
        } else if let Some(api_key) = config.api_key {
            using_api_key = true;
            Box::new(StaticCredentials(api_key))
        } else if let Some(token) = config.token {
            Box::new(StaticCredentials(token))
        } else if !env("WHODB_API_KEY").is_empty() {
            using_api_key = true;
            Box::new(StaticCredentials(env("WHODB_API_KEY")))
        } else {
            let provider = std::sync::Arc::new(CliCredentials::new());
            cli = Some(provider.clone());
            Box::new(ArcCliCredentials(provider))
        };
        let host = config
            .host
            .filter(|value| !value.is_empty())
            .unwrap_or_else(|| {
                let from_env = env("WHODB_HOST");
                if from_env.is_empty() {
                    DEFAULT_HOST.to_string()
                } else {
                    from_env
                }
            });
        let http = std::sync::Arc::new(HttpTransport::new(&host, credentials));
        Ok(Client {
            transport: Box::new(ArcTransport(http.clone())),
            http: Some(HttpTransportHandle(http)),
            cli,
            using_api_key,
            skip_resolution: false,
            org_input,
            project_input,
            workspace: OnceLock::new(),
        })
    }

    /// SHA-256 of the platform manifest this SDK release was generated from.
    pub fn manifest_hash() -> &'static str {
        crate::gen::manifest::MANIFEST_HASH
    }

    pub(crate) fn execute(&self, request: crate::gen::operations::Request) -> Result<Value> {
        let data = self
            .transport
            .execute(request.operation, request.document, request.variables)
            .map_err(interpret_server_error)?;
        Ok(data.get(request.operation).cloned().unwrap_or(Value::Null))
    }

    fn resolve_workspace(&self) -> Result<(String, String)> {
        let outcome = self.workspace.get_or_init(|| {
            self.resolve_workspace_inner()
                .map_err(|error| error.to_string())
        });
        match outcome {
            Ok(workspace) => Ok(workspace.clone()),
            Err(message) => Err(Error::Validation(message.clone())),
        }
    }

    fn resolve_workspace_inner(&self) -> Result<(String, String)> {
        let mut org_input = self.org_input.clone();
        let mut project_input = self.project_input.clone();
        if org_input.is_empty() || project_input.is_empty() {
            if let Some((_, org_id, project_id)) = self.cli.as_ref().and_then(|cli| cli.defaults())
            {
                if org_input.is_empty() {
                    org_input = org_id;
                }
                if project_input.is_empty() {
                    project_input = project_id;
                }
            }
        }
        if (org_input.is_empty() || project_input.is_empty()) && self.using_api_key {
            warn_if_flagged("MyWorkspace");
            let workspace = self.execute(ops::my_workspace_request(Map::new()))?;
            if org_input.is_empty() {
                org_input = workspace
                    .get("orgId")
                    .and_then(Value::as_str)
                    .unwrap_or_default()
                    .to_string();
            }
            if project_input.is_empty() {
                project_input = workspace
                    .get("projectId")
                    .and_then(Value::as_str)
                    .unwrap_or_default()
                    .to_string();
            }
            if project_input.is_empty() {
                return Err(Error::Validation(
                    "this API key has access to multiple (or zero) projects — set project in Config or WHODB_PROJECT".to_string(),
                ));
            }
        }
        if self.skip_resolution {
            return Ok((org_input, project_input));
        }
        if org_input.is_empty() || project_input.is_empty() {
            return Err(Error::Validation(
                "org and project are required — set them in Config, WHODB_ORG/WHODB_PROJECT, or run: whodb use".to_string(),
            ));
        }
        let mut org_id = org_input.clone();
        if !is_uuid(&org_input) {
            let orgs = self.execute(ops::my_organizations_request(Map::new()))?;
            org_id = match_entry(&orgs, &org_input).ok_or_else(|| {
                Error::Validation(format!(
                    "organization \"{org_input}\" not found for this account"
                ))
            })?;
        }
        if let Some(handle) = &self.http {
            handle.0.set_workspace(&org_id, "");
        }
        let mut project_id = project_input.clone();
        if !is_uuid(&project_input) {
            let mut variables = Map::new();
            variables.insert("orgId".to_string(), Value::String(org_id.clone()));
            let projects = self.execute(ops::projects_request(variables))?;
            project_id = match_entry(&projects, &project_input).ok_or_else(|| {
                Error::Validation(format!(
                    "project \"{project_input}\" not found in this organization"
                ))
            })?;
        }
        if let Some(handle) = &self.http {
            handle.0.set_workspace(&org_id, &project_id);
        }
        Ok((org_id, project_id))
    }

    pub(crate) fn project_id(&self) -> Result<String> {
        Ok(self.resolve_workspace()?.1)
    }

    /// Returns a handle for one ontology entity, addressed by apiName.
    pub fn ontology(&self, name: &str) -> OntologyHandle<'_> {
        OntologyHandle {
            client: self,
            api_name: name.to_string(),
            entity: Mutex::new(None),
        }
    }

    /// Lists all ontology entities in the project.
    pub fn ontology_entities(&self) -> Result<Vec<Value>> {
        let project_id = self.project_id()?;
        warn_if_flagged("OntologyEntities");
        let mut variables = Map::new();
        variables.insert("projectId".to_string(), Value::String(project_id));
        let result = self.execute(ops::ontology_entities_request(variables))?;
        Ok(result.as_array().cloned().unwrap_or_default())
    }
}

/// Finds an entry by slug or name and returns its id.
fn match_entry(result: &Value, input: &str) -> Option<String> {
    result.as_array()?.iter().find_map(|entry| {
        let matches = entry.get("slug").and_then(Value::as_str) == Some(input)
            || entry.get("name").and_then(Value::as_str) == Some(input);
        if matches {
            entry.get("id").and_then(Value::as_str).map(str::to_string)
        } else {
            None
        }
    })
}

struct ArcCliCredentials(std::sync::Arc<CliCredentials>);

impl CredentialProvider for ArcCliCredentials {
    fn token(&self) -> Result<String> {
        self.0.token()
    }
    fn refresh(&self) {
        self.0.refresh()
    }
    fn defaults(&self) -> Option<(String, String, String)> {
        self.0.defaults()
    }
}
