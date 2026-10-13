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

//! The client.source("src_...") facade: browse and read a connected data
//! source by ID.

use serde_json::{json, Map, Value};

use crate::client::Client;
use crate::error::Result;
use crate::facade_util::{as_list, vars};
use crate::gen::operations as ops;
use crate::hydrate::{hydrate_rows, Row};
use crate::manifest_check::warn_if_flagged;

const DEFAULT_PAGE_SIZE: usize = 100;

/// Scope and paging for a source object listing.
#[derive(Default, Debug, Clone)]
pub struct SourceObjectsOptions {
    /// Optional SourceObjectRefInput ({Kind, Locator, Path}).
    pub parent: Option<Value>,
    /// Page size; 0 leaves the platform default.
    pub page_size: usize,
    /// Page offset.
    pub page_offset: usize,
}

/// Filter, sort and paging for a source object read.
#[derive(Default, Debug, Clone)]
pub struct SourceRowsOptions {
    /// Platform WhereCondition object.
    pub where_filter: Option<Value>,
    /// List of SortCondition objects.
    pub sort: Option<Value>,
    /// Page size (default 100).
    pub page_size: usize,
    /// Page offset.
    pub offset: usize,
}

/// Handle for one connected source.
pub struct SourceHandle<'client> {
    pub(crate) client: &'client Client,
    pub(crate) source_id: String,
}

impl<'client> SourceHandle<'client> {
    /// Lists browsable objects (schemas, tables, collections...).
    pub fn objects(&self, options: &SourceObjectsOptions) -> Result<Vec<Value>> {
        let project_id = self.client.project_id()?;
        warn_if_flagged("PlatformSourceObjects");
        let variables = vars(&[
            ("projectId", json!(project_id)),
            ("sourceId", json!(self.source_id)),
            ("parent", options.parent.clone().unwrap_or(Value::Null)),
            ("kinds", Value::Null),
            (
                "pageSize",
                if options.page_size > 0 {
                    json!(options.page_size)
                } else {
                    Value::Null
                },
            ),
            (
                "pageOffset",
                if options.page_offset > 0 {
                    json!(options.page_offset)
                } else {
                    Value::Null
                },
            ),
        ]);
        Ok(as_list(self.client.execute(
            ops::platform_source_objects_request(variables),
        )?))
    }

    /// Lists the columns of one object (table/collection).
    pub fn columns(&self, object_ref: &Value) -> Result<Vec<Value>> {
        let project_id = self.client.project_id()?;
        warn_if_flagged("PlatformSourceColumns");
        let variables = vars(&[
            ("projectId", json!(project_id)),
            ("sourceId", json!(self.source_id)),
            ("ref", object_ref.clone()),
        ]);
        Ok(as_list(self.client.execute(
            ops::platform_source_columns_request(variables),
        )?))
    }

    /// Reads one page of rows from one object.
    pub fn rows(&self, object_ref: &Value, options: &SourceRowsOptions) -> Result<Vec<Row>> {
        let project_id = self.client.project_id()?;
        let page_size = if options.page_size == 0 {
            DEFAULT_PAGE_SIZE
        } else {
            options.page_size
        };
        warn_if_flagged("PlatformSourceRows");
        let variables = vars(&[
            ("projectId", json!(project_id)),
            ("sourceId", json!(self.source_id)),
            ("ref", object_ref.clone()),
            ("where", options.where_filter.clone().unwrap_or(Value::Null)),
            ("sort", options.sort.clone().unwrap_or(Value::Null)),
            ("pageSize", json!(page_size)),
            ("pageOffset", json!(options.offset)),
        ]);
        let result = self
            .client
            .execute(ops::platform_source_rows_request(variables))?;
        Ok(hydrate_rows(&result, &Map::new()).0)
    }
}
