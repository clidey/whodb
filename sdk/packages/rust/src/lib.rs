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

//! Official Rust SDK for the WhoDB hosted platform — the ontology, datasets,
//! and sources as in-code function APIs.
//!
//! ```no_run
//! use whodb_sdk::{Client, Config};
//!
//! let client = Client::new(Config {
//!     api_key: std::env::var("WHODB_API_KEY").ok(),
//!     ..Default::default()
//! }).unwrap();
//! let user = client.ontology("User").get("u_123").unwrap();
//! ```

pub mod auth;
pub mod client;
pub mod error;
pub mod gen;
mod hydrate;
mod manifest_check;
pub mod ontology;
pub mod transport;
pub mod transport_ipc;

pub use client::{Client, Config, DEFAULT_HOST};
pub use error::{Error, Result};
pub use hydrate::Row;
pub use ontology::{ActionOptions, ListOptions, OntologyHandle};
pub use transport::Transport;

/// SDK_VERSION is stamped by the release tooling (sync-versions.mjs) and used
/// only for the User-Agent header — the SDK's sole telemetry.
pub const SDK_VERSION: &str = "0.0.0";
