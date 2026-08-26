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
pub use ontology::{ListOptions, OntologyHandle};
pub use transport::Transport;

/// SDK_VERSION is stamped by the release tooling (sync-versions.mjs) and used
/// only for the User-Agent header — the SDK's sole telemetry.
pub const SDK_VERSION: &str = "0.0.0";
