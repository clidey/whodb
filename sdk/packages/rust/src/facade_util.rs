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

//! Shared helpers for the non-ontology facades: run polling, input encoding,
//! and small JSON conveniences.

use serde_json::{Map, Value};
use std::time::{Duration, Instant};

use crate::error::{Error, Result};
use crate::hydrate::Row;

/// One hydrated page of rows plus the total the platform reported (Null when
/// the operation does not report one).
#[derive(Debug, Clone, PartialEq)]
pub struct RowsPage {
    /// Hydrated rows.
    pub rows: Vec<Row>,
    /// Total row count, or Null.
    pub total_count: Value,
}

/// Bounds for run-polling methods (`wait`).
#[derive(Debug, Clone, Copy)]
pub struct WaitOptions {
    /// Caps the wait (default 60s).
    pub timeout: Duration,
    /// Delay between reads (default 2s). Zero polls without delay.
    pub poll: Duration,
}

impl Default for WaitOptions {
    fn default() -> Self {
        WaitOptions {
            timeout: Duration::from_secs(60),
            poll: Duration::from_secs(2),
        }
    }
}

/// Statuses after which a function or transform run will not change.
pub(crate) fn is_terminal_run_status(status: &Value) -> bool {
    let text = match status {
        Value::String(text) => text.trim().to_lowercase(),
        Value::Null => return false,
        other => other.to_string().to_lowercase(),
    };
    matches!(
        text.as_str(),
        "success"
            | "succeeded"
            | "completed"
            | "complete"
            | "failed"
            | "failure"
            | "cancelled"
            | "canceled"
            | "error"
    )
}

/// Reads until `done` or the timeout elapses (Platform error RUN_TIMEOUT).
pub(crate) fn poll_until(
    mut read: impl FnMut() -> Result<Value>,
    done: impl Fn(&Value) -> bool,
    options: WaitOptions,
    describe: &str,
) -> Result<Value> {
    let deadline = Instant::now() + options.timeout;
    loop {
        let value = read()?;
        if done(&value) {
            return Ok(value);
        }
        if Instant::now() >= deadline {
            return Err(Error::Platform {
                message: format!(
                    "{describe} did not finish within {}ms",
                    options.timeout.as_millis()
                ),
                code: "RUN_TIMEOUT".to_string(),
            });
        }
        if !options.poll.is_zero() {
            std::thread::sleep(options.poll);
        }
    }
}

/// Renders a function input as the compact JSON string the platform expects.
pub(crate) fn encode_input(input: &Value) -> String {
    match input {
        Value::String(text) => text.clone(),
        Value::Null => "{}".to_string(),
        other => other.to_string(),
    }
}

/// Builds a variables map from pairs.
pub(crate) fn vars(pairs: &[(&str, Value)]) -> Map<String, Value> {
    let mut map = Map::new();
    for (key, value) in pairs {
        map.insert((*key).to_string(), value.clone());
    }
    map
}

pub(crate) fn as_list(value: Value) -> Vec<Value> {
    value.as_array().cloned().unwrap_or_default()
}

pub(crate) fn str_field(value: &Value, key: &str) -> Value {
    value.get(key).cloned().unwrap_or(Value::Null)
}

/// Resolves a project resource addressed by id or name.
pub(crate) fn find_by_id_or_name(items: Vec<Value>, needle: &str) -> Option<Value> {
    items.into_iter().find(|item| {
        item.get("id").and_then(Value::as_str) == Some(needle)
            || item.get("name").and_then(Value::as_str) == Some(needle)
    })
}

/// Copies definition fields and injects the given values.
pub(crate) fn merge_input(definition: &Map<String, Value>, injected: &[(&str, Value)]) -> Value {
    let mut input = definition.clone();
    for (key, value) in injected {
        input.insert((*key).to_string(), value.clone());
    }
    Value::Object(input)
}

pub(crate) fn require(value: &str, label: &str) -> Result<()> {
    if value.trim().is_empty() {
        return Err(Error::Validation(format!("{label} is required")));
    }
    Ok(())
}
