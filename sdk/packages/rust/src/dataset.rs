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

//! The client.dataset("weekly_kpis") facade: metadata and paged reads for one
//! dataset, addressed by name.

use serde_json::{json, Map, Value};
use std::sync::Mutex;

use crate::client::Client;
use crate::error::{Error, Result};
use crate::facade_util::{as_list, str_field, vars, RowsPage};
use crate::gen::operations as ops;
use crate::hydrate::hydrate_rows;
use crate::manifest_check::warn_if_flagged;

const DEFAULT_PAGE_SIZE: usize = 100;

/// Paging for dataset reads.
#[derive(Default, Debug, Clone, Copy)]
pub struct DatasetRowsOptions {
    /// Page size (default 100).
    pub page_size: usize,
    /// Page offset.
    pub page_offset: usize,
}

/// Handle for one dataset.
pub struct DatasetHandle<'client> {
    pub(crate) client: &'client Client,
    pub(crate) name: String,
    pub(crate) dataset: Mutex<Option<Value>>,
}

impl<'client> DatasetHandle<'client> {
    /// Resolves and caches the dataset metadata backing this handle.
    pub fn meta(&self) -> Result<Value> {
        if let Some(cached) = self.dataset.lock().expect("dataset lock poisoned").as_ref() {
            return Ok(cached.clone());
        }
        let project_id = self.client.project_id()?;
        warn_if_flagged("ProjectDatasets");
        let result = self.client.execute(ops::project_datasets_request(vars(&[(
            "projectId",
            json!(project_id),
        )])))?;
        let dataset = as_list(result)
            .into_iter()
            .find(|item| item.get("name").and_then(Value::as_str) == Some(self.name.as_str()))
            .ok_or_else(|| {
                Error::NotFound(format!(
                    "dataset \"{}\" not found in this project",
                    self.name
                ))
            })?;
        *self.dataset.lock().expect("dataset lock poisoned") = Some(dataset.clone());
        Ok(dataset)
    }

    /// Queries dataset rows with paging.
    pub fn rows(&self, options: DatasetRowsOptions) -> Result<RowsPage> {
        let dataset = self.meta()?;
        let project_id = self.client.project_id()?;
        let page_size = if options.page_size == 0 {
            DEFAULT_PAGE_SIZE
        } else {
            options.page_size
        };
        warn_if_flagged("QueryDataset");
        let input = json!({
            "projectId": project_id,
            "datasetId": str_field(&dataset, "id"),
            "pageSize": page_size,
            "pageOffset": options.page_offset,
        });
        let result = self
            .client
            .execute(ops::query_dataset_request(vars(&[("input", input)])))?;
        let (rows, total_count) = hydrate_rows(&result, &Map::new());
        Ok(RowsPage { rows, total_count })
    }
}
