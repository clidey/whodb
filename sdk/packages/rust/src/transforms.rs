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

//! The client.transforms() / client.transform("name") facades: the project's
//! transform collection and one transform with run control.

use serde_json::{json, Map, Value};
use std::sync::Mutex;

use crate::client::Client;
use crate::error::{Error, Result};
use crate::facade_util::{
    as_list, find_by_id_or_name, is_terminal_run_status, merge_input, poll_until, require,
    str_field, vars, RowsPage, WaitOptions,
};
use crate::gen::operations as ops;
use crate::hydrate::hydrate_rows;
use crate::manifest_check::warn_if_flagged;

/// The project's transform collection.
pub struct TransformsHandle<'client> {
    pub(crate) client: &'client Client,
}

impl<'client> TransformsHandle<'client> {
    /// Lists the project's transforms.
    pub fn list(&self) -> Result<Vec<Value>> {
        let project_id = self.client.project_id()?;
        warn_if_flagged("ProjectTransforms");
        Ok(as_list(self.client.execute(
            ops::project_transforms_request(vars(&[("projectId", json!(project_id))])),
        )?))
    }

    /// Creates a transform from a pipeline graph (SaveTransformInput fields
    /// without id/projectId).
    pub fn create(&self, definition: &Map<String, Value>) -> Result<Value> {
        let project_id = self.client.project_id()?;
        warn_if_flagged("SaveTransform");
        let input = merge_input(definition, &[("projectId", json!(project_id))]);
        self.client
            .execute(ops::save_transform_request(vars(&[("input", input)])))
    }

    /// Runs a pipeline graph without saving it and returns the resulting rows.
    pub fn preview(&self, graph_json: &str) -> Result<RowsPage> {
        let project_id = self.client.project_id()?;
        warn_if_flagged("PreviewTransform");
        let result = self.client.execute(ops::preview_transform_request(vars(&[
            ("projectId", json!(project_id)),
            ("graphJson", json!(graph_json)),
        ])))?;
        let (rows, total_count) = hydrate_rows(&result, &Map::new());
        Ok(RowsPage { rows, total_count })
    }
}

/// Handle for one transform, addressed by name or id.
pub struct TransformHandle<'client> {
    pub(crate) client: &'client Client,
    pub(crate) name: String,
    pub(crate) transform: Mutex<Option<Value>>,
}

impl<'client> TransformHandle<'client> {
    /// Resolves and caches the transform backing this handle.
    pub fn meta(&self) -> Result<Value> {
        if let Some(cached) = self
            .transform
            .lock()
            .expect("transform lock poisoned")
            .as_ref()
        {
            return Ok(cached.clone());
        }
        let project_id = self.client.project_id()?;
        warn_if_flagged("ProjectTransforms");
        let result = self
            .client
            .execute(ops::project_transforms_request(vars(&[(
                "projectId",
                json!(project_id),
            )])))?;
        let transform = find_by_id_or_name(as_list(result), &self.name).ok_or_else(|| {
            Error::NotFound(format!(
                "transform \"{}\" not found in this project",
                self.name
            ))
        })?;
        *self.transform.lock().expect("transform lock poisoned") = Some(transform.clone());
        Ok(transform)
    }

    fn scope(&self) -> Result<(String, Value)> {
        let transform = self.meta()?;
        Ok((self.client.project_id()?, str_field(&transform, "id")))
    }

    /// Lists recent runs, newest first (limit 0 = 20).
    pub fn runs(&self, limit: usize) -> Result<Vec<Value>> {
        let (project_id, id) = self.scope()?;
        warn_if_flagged("TransformRuns");
        Ok(as_list(self.client.execute(
            ops::transform_runs_request(vars(&[
                ("projectId", json!(project_id)),
                ("transformId", id),
                ("limit", json!(if limit == 0 { 20 } else { limit })),
            ])),
        )?))
    }

    /// Starts a run and returns its record immediately; follow it with `wait`.
    pub fn run(&self) -> Result<Value> {
        let (project_id, id) = self.scope()?;
        warn_if_flagged("RunTransform");
        self.client.execute(ops::run_transform_request(vars(&[
            ("projectId", json!(project_id)),
            ("id", id),
        ])))
    }

    /// Cancels the active run, or a specific run when `run_id` is given.
    pub fn cancel_run(&self, run_id: Option<&str>) -> Result<()> {
        let (project_id, id) = self.scope()?;
        warn_if_flagged("CancelTransformRun");
        self.client
            .execute(ops::cancel_transform_run_request(vars(&[
                ("projectId", json!(project_id)),
                ("transformId", id),
                (
                    "runId",
                    run_id.map(|value| json!(value)).unwrap_or(Value::Null),
                ),
            ])))?;
        Ok(())
    }

    /// Polls until the run reaches a terminal status and returns it.
    pub fn wait(&self, run_id: &str, options: WaitOptions) -> Result<Value> {
        require(run_id, "run_id")?;
        let (project_id, id) = self.scope()?;
        let read = || -> Result<Value> {
            warn_if_flagged("TransformRuns");
            let runs = as_list(self.client.execute(ops::transform_runs_request(vars(&[
                ("projectId", json!(project_id)),
                ("transformId", id.clone()),
                ("limit", json!(100)),
            ])))?);
            runs.into_iter()
                .find(|run| run.get("id").and_then(Value::as_str) == Some(run_id))
                .ok_or_else(|| {
                    Error::NotFound(format!(
                        "run \"{run_id}\" not found for transform \"{}\"",
                        self.name
                    ))
                })
        };
        poll_until(
            read,
            |run| is_terminal_run_status(&str_field(run, "status")),
            options,
            &format!("transform run {run_id}"),
        )
    }

    /// Replaces the transform definition.
    pub fn update(&self, definition: &Map<String, Value>) -> Result<Value> {
        let (project_id, id) = self.scope()?;
        warn_if_flagged("SaveTransform");
        let input = merge_input(definition, &[("id", id), ("projectId", json!(project_id))]);
        let saved = self
            .client
            .execute(ops::save_transform_request(vars(&[("input", input)])))?;
        *self.transform.lock().expect("transform lock poisoned") = Some(saved.clone());
        Ok(saved)
    }

    /// Soft-deletes the transform.
    pub fn delete(&self) -> Result<()> {
        let (project_id, id) = self.scope()?;
        warn_if_flagged("DeleteTransform");
        self.client.execute(ops::delete_transform_request(vars(&[
            ("projectId", json!(project_id)),
            ("id", id),
        ])))?;
        *self.transform.lock().expect("transform lock poisoned") = None;
        Ok(())
    }

    /// Promotes the draft transform: snapshots it and makes it the active production version.
    pub fn promote(&self, message: Option<&str>) -> Result<Value> {
        let id = str_field(&self.meta()?, "id");
        self.client
            .versions()
            .promote("transform", id.as_str().unwrap_or_default(), message)
    }

    /// Lists the transform's promoted versions, newest first.
    pub fn versions(&self) -> Result<Vec<Value>> {
        let id = str_field(&self.meta()?, "id");
        self.client
            .versions()
            .list("transform", id.as_str().unwrap_or_default())
    }
}
