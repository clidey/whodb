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
//! The client.versions() facade: version history, the active production
//! version, promotion, rollback, and restore-to-draft for any versionable
//! object (app, function, transform, dataset, ontology). PromoteObject
//! snapshots the draft and makes it the active production version in one step.

use serde_json::{json, Map, Value};

use crate::client::Client;
use crate::error::{Error, Result};
use crate::facade_util::{as_list, require, vars};
use crate::gen::operations as ops;
use crate::manifest_check::warn_if_flagged;

/// Versionable object types accepted by the versioning facade.
pub const OBJECT_TYPES: [&str; 5] = ["app", "function", "transform", "dataset", "ontology"];

/// Versioning and promotion for the selected project's objects.
pub struct VersionsHandle<'client> {
    pub(crate) client: &'client Client,
}

impl<'client> VersionsHandle<'client> {
    fn scope(&self, object_type: &str, object_id: &str) -> Result<Map<String, Value>> {
        if !OBJECT_TYPES.contains(&object_type) {
            return Err(Error::Validation(format!(
                "object_type must be one of {}; got \"{object_type}\"",
                OBJECT_TYPES.join(", ")
            )));
        }
        require(object_id, "object_id")?;
        Ok(vars(&[
            ("projectId", json!(self.client.project_id()?)),
            ("objectId", json!(object_id)),
            ("objectType", json!(object_type)),
        ]))
    }

    /// Lists an object's promoted versions, newest first.
    pub fn list(&self, object_type: &str, object_id: &str) -> Result<Vec<Value>> {
        let scope = self.scope(object_type, object_id)?;
        warn_if_flagged("ObjectVersions");
        Ok(as_list(
            self.client.execute(ops::object_versions_request(scope))?,
        ))
    }

    /// The active production version, or `Value::Null` when nothing has been promoted.
    pub fn active(&self, object_type: &str, object_id: &str) -> Result<Value> {
        let scope = self.scope(object_type, object_id)?;
        warn_if_flagged("ActiveProdVersion");
        self.client.execute(ops::active_prod_version_request(scope))
    }

    /// Every active production version in the project.
    pub fn active_all(&self) -> Result<Vec<Value>> {
        let project_id = self.client.project_id()?;
        warn_if_flagged("ProjectActiveProdVersions");
        Ok(as_list(self.client.execute(
            ops::project_active_prod_versions_request(vars(&[("projectId", json!(project_id))])),
        )?))
    }

    /// Promotes the draft: snapshots it as a new version and activates it for production.
    pub fn promote(
        &self,
        object_type: &str,
        object_id: &str,
        message: Option<&str>,
    ) -> Result<Value> {
        let mut input = self.scope(object_type, object_id)?;
        input.insert(
            "message".to_string(),
            message.map(|text| json!(text)).unwrap_or(Value::Null),
        );
        warn_if_flagged("PromoteObject");
        self.client.execute(ops::promote_object_request(vars(&[(
            "input",
            Value::Object(input),
        )])))
    }

    /// Makes an earlier promoted version the active production version.
    pub fn rollback(&self, object_type: &str, object_id: &str, version: i64) -> Result<Value> {
        let mut input = self.scope(object_type, object_id)?;
        input.insert("version".to_string(), json!(version));
        warn_if_flagged("RollbackObject");
        self.client.execute(ops::rollback_object_request(vars(&[(
            "input",
            Value::Object(input),
        )])))
    }

    /// Copies a promoted version back into the object's draft; returns the restored object.
    pub fn restore_to_draft(
        &self,
        object_type: &str,
        object_id: &str,
        version: i64,
    ) -> Result<Value> {
        let scope = self.scope(object_type, object_id)?;
        let project_id = scope["projectId"].clone();
        let (operation, id_arg): (&str, &str) = match object_type {
            "ontology" => ("RestoreOntologyVersionToDraft", "ontologyId"),
            "dataset" => ("RestoreDatasetVersionToDraft", "datasetId"),
            "transform" => ("RestoreTransformVersionToDraft", "transformId"),
            "function" => ("RestoreFunctionVersionToDraft", "functionId"),
            _ => ("RestoreAppVersionToDraft", "appId"),
        };
        let variables = vars(&[
            ("projectId", project_id),
            (id_arg, json!(object_id)),
            ("version", json!(version)),
        ]);
        warn_if_flagged(operation);
        let request = match object_type {
            "ontology" => ops::restore_ontology_version_to_draft_request(variables),
            "dataset" => ops::restore_dataset_version_to_draft_request(variables),
            "transform" => ops::restore_transform_version_to_draft_request(variables),
            "function" => ops::restore_function_version_to_draft_request(variables),
            _ => ops::restore_app_version_to_draft_request(variables),
        };
        self.client.execute(request)
    }
}
