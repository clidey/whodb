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

//! The client.functions() / client.function("name") facades: the project's
//! function collection and one function with execution and run control.

use serde_json::{json, Map, Value};
use std::sync::Mutex;

use crate::client::Client;
use crate::error::{Error, Result};
use crate::facade_util::{
    as_list, encode_input, find_by_id_or_name, is_terminal_run_status, merge_input, poll_until,
    require, str_field, vars, WaitOptions,
};
use crate::gen::operations as ops;
use crate::manifest_check::warn_if_flagged;

/// Options for a one-off test execution of the draft function.
#[derive(Default, Debug, Clone)]
pub struct FunctionTestOptions {
    /// Overrides the draft files as [{path, content}].
    pub files: Option<Value>,
    /// Mounts project files as inputs.
    pub input_file_ids: Option<Vec<String>>,
}

/// The project's function collection.
pub struct FunctionsHandle<'client> {
    pub(crate) client: &'client Client,
}

impl<'client> FunctionsHandle<'client> {
    /// Lists the project's functions.
    pub fn list(&self) -> Result<Vec<Value>> {
        let project_id = self.client.project_id()?;
        warn_if_flagged("ProjectFunctions");
        Ok(as_list(self.client.execute(
            ops::project_functions_request(vars(&[("projectId", json!(project_id))])),
        )?))
    }

    /// Creates a function (CreateFunctionInput fields without projectId).
    pub fn create(&self, definition: &Map<String, Value>) -> Result<Value> {
        let project_id = self.client.project_id()?;
        warn_if_flagged("CreateFunction");
        let input = merge_input(definition, &[("projectId", json!(project_id))]);
        self.client
            .execute(ops::create_function_request(vars(&[("input", input)])))
    }
}

/// Handle for one function, addressed by name or id.
pub struct FunctionHandle<'client> {
    pub(crate) client: &'client Client,
    pub(crate) name: String,
    pub(crate) function: Mutex<Option<Value>>,
}

impl<'client> FunctionHandle<'client> {
    /// Resolves and caches the function backing this handle.
    pub fn meta(&self) -> Result<Value> {
        if let Some(cached) = self
            .function
            .lock()
            .expect("function lock poisoned")
            .as_ref()
        {
            return Ok(cached.clone());
        }
        let project_id = self.client.project_id()?;
        warn_if_flagged("ProjectFunctions");
        let result = self.client.execute(ops::project_functions_request(vars(&[(
            "projectId",
            json!(project_id),
        )])))?;
        let function = find_by_id_or_name(as_list(result), &self.name).ok_or_else(|| {
            Error::NotFound(format!(
                "function \"{}\" not found in this project",
                self.name
            ))
        })?;
        *self.function.lock().expect("function lock poisoned") = Some(function.clone());
        Ok(function)
    }

    fn scope(&self) -> Result<(String, Value)> {
        let function = self.meta()?;
        Ok((self.client.project_id()?, str_field(&function, "id")))
    }

    /// Runs the active version synchronously and returns its result.
    pub fn execute(&self, input: &Value, input_file_ids: Option<&[String]>) -> Result<Value> {
        let (project_id, id) = self.scope()?;
        warn_if_flagged("ExecuteFunction");
        self.client.execute(ops::execute_function_request(vars(&[
            ("projectId", json!(project_id)),
            ("functionId", id),
            ("input", json!(encode_input(input))),
            (
                "inputFileIds",
                input_file_ids.map(|ids| json!(ids)).unwrap_or(Value::Null),
            ),
        ])))
    }

    /// Runs the draft version once with test input; nothing is persisted.
    pub fn test(&self, input: &Value, options: &FunctionTestOptions) -> Result<Value> {
        let (project_id, id) = self.scope()?;
        warn_if_flagged("TestFunction");
        let test_input = json!({
            "projectId": project_id,
            "functionId": id,
            "input": encode_input(input),
            "files": options.files.clone().unwrap_or(Value::Null),
            "inputFileIds": options.input_file_ids.as_ref().map(|ids| json!(ids)).unwrap_or(Value::Null),
        });
        self.client
            .execute(ops::test_function_request(vars(&[("input", test_input)])))
    }

    /// Starts a persisted run and returns it immediately; follow it with
    /// `wait`. `use_active_version` None leaves the platform default.
    pub fn start_run(&self, input: &Value, use_active_version: Option<bool>) -> Result<Value> {
        let (project_id, id) = self.scope()?;
        warn_if_flagged("StartFunctionRun");
        self.client.execute(ops::start_function_run_request(vars(&[
            ("projectId", json!(project_id)),
            ("functionId", id),
            ("input", json!(encode_input(input))),
            (
                "useActiveVersion",
                use_active_version.map(Value::Bool).unwrap_or(Value::Null),
            ),
        ])))
    }

    /// Reads one persisted run.
    pub fn run(&self, run_id: &str) -> Result<Value> {
        require(run_id, "run_id")?;
        let project_id = self.client.project_id()?;
        warn_if_flagged("FunctionRun");
        self.client.execute(ops::function_run_request(vars(&[
            ("projectId", json!(project_id)),
            ("id", json!(run_id)),
        ])))
    }

    /// Lists persisted runs, newest first (limit 0 = 20).
    pub fn runs(&self, limit: usize) -> Result<Vec<Value>> {
        let (project_id, id) = self.scope()?;
        warn_if_flagged("FunctionRuns");
        Ok(as_list(self.client.execute(ops::function_runs_request(
            vars(&[
                ("projectId", json!(project_id)),
                ("functionId", id),
                ("limit", json!(if limit == 0 { 20 } else { limit })),
            ]),
        ))?))
    }

    /// Polls a run until it reaches a terminal status and returns it.
    pub fn wait(&self, run_id: &str, options: WaitOptions) -> Result<Value> {
        require(run_id, "run_id")?;
        poll_until(
            || self.run(run_id),
            |run| is_terminal_run_status(&str_field(run, "status")),
            options,
            &format!("function run {run_id}"),
        )
    }

    /// Requests cancellation of a queued or running run.
    pub fn cancel_run(&self, run_id: &str) -> Result<Value> {
        require(run_id, "run_id")?;
        let project_id = self.client.project_id()?;
        warn_if_flagged("CancelFunctionRun");
        self.client.execute(ops::cancel_function_run_request(vars(&[
            ("projectId", json!(project_id)),
            ("id", json!(run_id)),
        ])))
    }

    /// Deploys the active version.
    pub fn deploy(&self) -> Result<()> {
        let (project_id, id) = self.scope()?;
        warn_if_flagged("DeployFunction");
        self.client.execute(ops::deploy_function_request(vars(&[
            ("projectId", json!(project_id)),
            ("id", id),
        ])))?;
        Ok(())
    }

    /// Redeploys the active version.
    pub fn redeploy(&self) -> Result<()> {
        let (project_id, id) = self.scope()?;
        warn_if_flagged("RedeployFunction");
        self.client.execute(ops::redeploy_function_request(vars(&[
            ("projectId", json!(project_id)),
            ("id", id),
        ])))?;
        Ok(())
    }

    /// Updates the draft function; send only changed fields.
    pub fn update(&self, changes: &Map<String, Value>) -> Result<Value> {
        let (project_id, id) = self.scope()?;
        warn_if_flagged("UpdateFunction");
        let input = merge_input(changes, &[("id", id), ("projectId", json!(project_id))]);
        let updated = self
            .client
            .execute(ops::update_function_request(vars(&[("input", input)])))?;
        *self.function.lock().expect("function lock poisoned") = Some(updated.clone());
        Ok(updated)
    }

    /// Soft-deletes the function.
    pub fn delete(&self) -> Result<()> {
        let (project_id, id) = self.scope()?;
        warn_if_flagged("DeleteFunction");
        self.client.execute(ops::delete_function_request(vars(&[
            ("projectId", json!(project_id)),
            ("id", id),
        ])))?;
        *self.function.lock().expect("function lock poisoned") = None;
        Ok(())
    }
}
