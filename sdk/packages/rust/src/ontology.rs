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

//! The client.ontology("User") facade: reads and record writes for one
//! ontology entity, addressed by apiName. Mirrors the other SDKs; languages
//! differ in ergonomics, never in behavior (conformance-pinned).

use serde_json::{json, Map, Value};
use std::sync::Mutex;

use crate::client::Client;
use crate::error::{Error, Result};
use crate::gen::operations as ops;
use crate::hydrate::{hydrate_rows, property_types_of, Row};
use crate::manifest_check::warn_if_flagged;

const DEFAULT_PAGE_SIZE: usize = 100;

/// Options for list-shaped reads. `where_filter` is a JSON filter object
/// (property → {"eq": ...} etc.) serialized into OntologyQuery.
#[derive(Default)]
pub struct ListOptions {
    /// JSON filter object (property → operator map).
    pub where_filter: Option<Value>,
    /// Sort specs ({field, desc}).
    pub sort: Option<Value>,
    /// Page size (default 100).
    pub page_size: usize,
    /// Page offset.
    pub offset: usize,
}

/// Concurrency and retry controls for a named ontology action.
#[derive(Default)]
pub struct ActionOptions {
    /// Expected current record version for optimistic concurrency.
    pub expected_version: Option<i64>,
    /// Stable key that makes retries return the original execution result.
    pub idempotency_key: Option<String>,
}

/// Handle for one ontology entity.
pub struct OntologyHandle<'client> {
    pub(crate) client: &'client Client,
    pub(crate) api_name: String,
    pub(crate) entity: Mutex<Option<Value>>,
}

impl<'client> OntologyHandle<'client> {
    /// Resolves and caches the entity metadata backing this handle.
    pub fn entity_meta(&self) -> Result<Value> {
        {
            let cached = self.entity.lock().expect("entity lock poisoned");
            if let Some(entity) = cached.as_ref() {
                return Ok(entity.clone());
            }
        }
        warn_if_flagged("OntologyEntities");
        let project_id = self.client.project_id()?;
        let mut variables = Map::new();
        variables.insert("projectId".to_string(), Value::String(project_id));
        let result = self
            .client
            .execute(ops::ontology_entities_request(variables))?;
        let entities = result.as_array().cloned().unwrap_or_default();
        for entity in entities {
            if entity.get("apiName").and_then(Value::as_str) == Some(self.api_name.as_str()) {
                *self.entity.lock().expect("entity lock poisoned") = Some(entity.clone());
                return Ok(entity);
            }
        }
        Err(Error::NotFound(format!(
            "ontology entity \"{}\" not found in this project",
            self.api_name
        )))
    }

    fn entity_id(&self) -> Result<String> {
        Ok(self
            .entity_meta()?
            .get("id")
            .and_then(Value::as_str)
            .unwrap_or_default()
            .to_string())
    }

    fn primary_key(&self) -> Result<String> {
        Ok(self
            .entity_meta()?
            .get("primaryKey")
            .and_then(Value::as_str)
            .unwrap_or_default()
            .to_string())
    }

    /// Describes the entity: schema, properties, links.
    pub fn describe(&self) -> Result<Value> {
        self.entity_meta()?; // NotFound for unknown entities
        warn_if_flagged("OntologyDescribe");
        let project_id = self.client.project_id()?;
        let mut variables = Map::new();
        variables.insert("projectId".to_string(), Value::String(project_id));
        variables.insert(
            "input".to_string(),
            json!({"entities": [self.api_name], "includeInferred": true}),
        );
        self.client
            .execute(ops::ontology_describe_request(variables))
    }

    fn query(&self, mut input: Map<String, Value>) -> Result<Value> {
        warn_if_flagged("OntologyQuery");
        let project_id = self.client.project_id()?;
        input.insert("entity".to_string(), Value::String(self.api_name.clone()));
        let mut variables = Map::new();
        variables.insert("projectId".to_string(), Value::String(project_id));
        variables.insert("input".to_string(), Value::Object(input));
        self.client.execute(ops::ontology_query_request(variables))
    }

    /// Fetches a single record by primary key; Ok(None) when absent.
    pub fn get(&self, pk: &str) -> Result<Option<Row>> {
        let primary_key = self.primary_key()?;
        if primary_key.is_empty() {
            return Err(Error::Validation(format!(
                "entity \"{}\" has no primary key — use list with a where filter",
                self.api_name
            )));
        }
        let where_json = serde_json::to_string(&json!({primary_key: {"eq": pk}}))
            .map_err(|error| Error::Transport(error.to_string()))?;
        let mut input = Map::new();
        input.insert("whereJson".to_string(), Value::String(where_json));
        input.insert("pageSize".to_string(), Value::from(1));
        input.insert("offset".to_string(), Value::from(0));
        let result = self.query(input)?;
        let entity = self.entity_meta()?;
        let (rows, _) = hydrate_rows(&result, &property_types_of(&entity));
        Ok(rows.into_iter().next())
    }

    /// Returns one page of records with optional filter/sort.
    pub fn list(&self, options: &ListOptions) -> Result<Vec<Row>> {
        let entity = self.entity_meta()?;
        let page_size = if options.page_size == 0 {
            DEFAULT_PAGE_SIZE
        } else {
            options.page_size
        };
        let mut input = Map::new();
        input.insert("pageSize".to_string(), Value::from(page_size));
        input.insert("offset".to_string(), Value::from(options.offset));
        if let Some(where_filter) = &options.where_filter {
            let where_json = serde_json::to_string(where_filter)
                .map_err(|error| Error::Transport(error.to_string()))?;
            input.insert("whereJson".to_string(), Value::String(where_json));
        }
        if let Some(sort) = &options.sort {
            input.insert("sort".to_string(), sort.clone());
        }
        let result = self.query(input)?;
        let (rows, _) = hydrate_rows(&result, &property_types_of(&entity));
        Ok(rows)
    }

    /// Walks every page, invoking `visit` per page until a short page ends
    /// the result set or `visit` returns false.
    pub fn pages(
        &self,
        options: &ListOptions,
        mut visit: impl FnMut(Vec<Row>) -> bool,
    ) -> Result<()> {
        let page_size = if options.page_size == 0 {
            DEFAULT_PAGE_SIZE
        } else {
            options.page_size
        };
        let mut offset = options.offset;
        loop {
            let page = self.list(&ListOptions {
                where_filter: options.where_filter.clone(),
                sort: options.sort.clone(),
                page_size,
                offset,
            })?;
            let len = page.len();
            if !visit(page) || len < page_size {
                return Ok(());
            }
            offset += page_size;
        }
    }

    /// Computes the current actor's readable fields and allowed actions.
    pub fn capabilities(&self, record_key: Option<&str>) -> Result<Value> {
        let entity_id = self.entity_id()?;
        warn_if_flagged("OntologyRecordCapabilities");
        let project_id = self.client.project_id()?;
        let mut variables = Map::new();
        variables.insert("projectId".to_string(), Value::String(project_id));
        variables.insert("ontologyId".to_string(), Value::String(entity_id));
        if let Some(key) = record_key {
            variables.insert("recordKey".to_string(), Value::String(key.to_string()));
        }
        self.client
            .execute(ops::ontology_record_capabilities_request(variables))
    }

    /// Dry-runs a named action without persisting records, effects, or events.
    pub fn preview_action(
        &self,
        action: &str,
        record_key: Option<&str>,
        values: &Map<String, Value>,
    ) -> Result<Value> {
        let entity_id = self.entity_id()?;
        warn_if_flagged("PreviewOntologyAction");
        let project_id = self.client.project_id()?;
        let mut input = Map::new();
        input.insert("projectId".to_string(), Value::String(project_id));
        input.insert("ontologyId".to_string(), Value::String(entity_id));
        input.insert("action".to_string(), Value::String(action.to_string()));
        input.insert("values".to_string(), Value::Object(values.clone()));
        if let Some(key) = record_key {
            input.insert("recordKey".to_string(), Value::String(key.to_string()));
        }
        let mut variables = Map::new();
        variables.insert("input".to_string(), Value::Object(input));
        self.client
            .execute(ops::preview_ontology_action_request(variables))
    }

    /// Executes a named action with full behavior enforcement.
    pub fn action(
        &self,
        action: &str,
        record_key: Option<&str>,
        values: &Map<String, Value>,
        options: &ActionOptions,
    ) -> Result<Value> {
        let entity_id = self.entity_id()?;
        warn_if_flagged("ExecuteOntologyAction");
        let project_id = self.client.project_id()?;
        let mut input = Map::new();
        input.insert("projectId".to_string(), Value::String(project_id));
        input.insert("ontologyId".to_string(), Value::String(entity_id));
        input.insert("action".to_string(), Value::String(action.to_string()));
        input.insert("values".to_string(), Value::Object(values.clone()));
        if let Some(key) = record_key {
            input.insert("recordKey".to_string(), Value::String(key.to_string()));
        }
        if let Some(version) = options.expected_version {
            input.insert("expectedVersion".to_string(), Value::from(version));
        }
        if let Some(key) = &options.idempotency_key {
            input.insert("idempotencyKey".to_string(), Value::String(key.clone()));
        }
        let mut variables = Map::new();
        variables.insert("input".to_string(), Value::Object(input));
        self.client
            .execute(ops::execute_ontology_action_request(variables))
    }

    /// Lists recent named-action executions for one record.
    pub fn action_executions(&self, record_key: &str, limit: usize) -> Result<Value> {
        let entity_id = self.entity_id()?;
        warn_if_flagged("OntologyActionExecutions");
        let project_id = self.client.project_id()?;
        let mut variables = Map::new();
        variables.insert("projectId".to_string(), Value::String(project_id));
        variables.insert("ontologyId".to_string(), Value::String(entity_id));
        variables.insert(
            "recordKey".to_string(),
            Value::String(record_key.to_string()),
        );
        variables.insert(
            "limit".to_string(),
            Value::from(if limit == 0 { 50 } else { limit }),
        );
        self.client
            .execute(ops::ontology_action_executions_request(variables))
    }

    /// Inserts one record. Values are field name/value pairs.
    pub fn create(&self, values: &Map<String, Value>) -> Result<()> {
        let entity_id = self.entity_id()?;
        warn_if_flagged("OntologyAddRow");
        let project_id = self.client.project_id()?;
        let mut variables = Map::new();
        variables.insert("projectId".to_string(), Value::String(project_id));
        variables.insert("entityId".to_string(), Value::String(entity_id));
        variables.insert("values".to_string(), to_record_inputs(values));
        self.client
            .execute(ops::ontology_add_row_request(variables))?;
        Ok(())
    }

    /// Inserts many records; idempotency_key makes retries safe.
    pub fn create_many(
        &self,
        rows: &[Map<String, Value>],
        idempotency_key: Option<&str>,
    ) -> Result<Value> {
        let entity_id = self.entity_id()?;
        warn_if_flagged("OntologyAddRows");
        let project_id = self.client.project_id()?;
        let wire_rows: Vec<Value> = rows
            .iter()
            .map(|row| json!({"values": to_record_inputs(row)}))
            .collect();
        let mut variables = Map::new();
        variables.insert("projectId".to_string(), Value::String(project_id));
        variables.insert("entityId".to_string(), Value::String(entity_id));
        variables.insert("rows".to_string(), Value::Array(wire_rows));
        if let Some(key) = idempotency_key {
            variables.insert("idempotencyKey".to_string(), Value::String(key.to_string()));
        }
        self.client
            .execute(ops::ontology_add_rows_request(variables))
    }

    /// Updates one record identified by primary key.
    pub fn update(&self, pk: &str, values: &Map<String, Value>) -> Result<()> {
        let primary_key = self.primary_key()?;
        if primary_key.is_empty() {
            return Err(Error::Validation(format!(
                "entity \"{}\" has no primary key — updates are not supported",
                self.api_name
            )));
        }
        let entity_id = self.entity_id()?;
        warn_if_flagged("OntologyUpdateRow");
        let project_id = self.client.project_id()?;
        let mut merged = values.clone();
        merged.insert(primary_key, Value::String(pk.to_string()));
        let updated_columns: Vec<Value> = values
            .keys()
            .map(|key| Value::String(key.clone()))
            .collect();
        let mut variables = Map::new();
        variables.insert("projectId".to_string(), Value::String(project_id));
        variables.insert("entityId".to_string(), Value::String(entity_id));
        variables.insert("values".to_string(), to_record_inputs(&merged));
        variables.insert("updatedColumns".to_string(), Value::Array(updated_columns));
        self.client
            .execute(ops::ontology_update_row_request(variables))?;
        Ok(())
    }

    /// Deletes one record identified by primary key.
    pub fn delete(&self, pk: &str) -> Result<()> {
        let primary_key = self.primary_key()?;
        if primary_key.is_empty() {
            return Err(Error::Validation(format!(
                "entity \"{}\" has no primary key — deletes are not supported",
                self.api_name
            )));
        }
        let entity_id = self.entity_id()?;
        warn_if_flagged("OntologyDeleteRow");
        let project_id = self.client.project_id()?;
        let mut values = Map::new();
        values.insert(primary_key, Value::String(pk.to_string()));
        let mut variables = Map::new();
        variables.insert("projectId".to_string(), Value::String(project_id));
        variables.insert("entityId".to_string(), Value::String(entity_id));
        variables.insert("values".to_string(), to_record_inputs(&values));
        self.client
            .execute(ops::ontology_delete_row_request(variables))?;
        Ok(())
    }

    /// Follows an outgoing link from one record to its related records.
    pub fn follow_link(
        &self,
        pk: &str,
        link_api_name: &str,
        page_size: usize,
        offset: usize,
    ) -> Result<Vec<Row>> {
        let entity_id = self.entity_id()?;
        warn_if_flagged("OntologyFollowLink");
        let project_id = self.client.project_id()?;
        let mut variables = Map::new();
        variables.insert("projectId".to_string(), Value::String(project_id));
        variables.insert("entityId".to_string(), Value::String(entity_id));
        variables.insert("pk".to_string(), Value::String(pk.to_string()));
        variables.insert(
            "linkApiName".to_string(),
            Value::String(link_api_name.to_string()),
        );
        variables.insert(
            "pageSize".to_string(),
            Value::from(if page_size == 0 {
                DEFAULT_PAGE_SIZE
            } else {
                page_size
            }),
        );
        variables.insert("pageOffset".to_string(), Value::from(offset));
        let result = self
            .client
            .execute(ops::ontology_follow_link_request(variables))?;
        let (rows, _) = hydrate_rows(&result, &Map::new());
        Ok(rows)
    }
}

/// Converts a data map to GraphQL RecordInput pairs, JSON-encoding objects
/// and lowercasing booleans (cross-language behavior).
fn to_record_inputs(values: &Map<String, Value>) -> Value {
    let records: Vec<Value> = values
        .iter()
        .map(|(key, value)| {
            let encoded = match value {
                Value::Null => String::new(),
                Value::Bool(true) => "true".to_string(),
                Value::Bool(false) => "false".to_string(),
                Value::String(text) => text.clone(),
                Value::Object(_) | Value::Array(_) => {
                    serde_json::to_string(value).unwrap_or_default()
                }
                other => other.to_string(),
            };
            json!({"Key": key, "Value": encoded})
        })
        .collect();
    Value::Array(records)
}
