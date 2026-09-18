//! Error taxonomy for the whodb-sdk crate, mirrored across all SDK languages.

use std::fmt;

/// All errors produced by the WhoDB SDK.
#[derive(Debug)]
pub enum Error {
    /// Authentication failed: missing, invalid, expired, or revoked credentials.
    Auth(String),
    /// The requested resource does not exist or the caller cannot see it.
    NotFound(String),
    /// The request was rejected as invalid before execution.
    Validation(String),
    /// This SDK release targets an older platform API; upgrade the crate.
    Version(String),
    /// The whodb CLI credential helper is unavailable or not logged in.
    CliCredentials(String),
    /// An operation is not available over the current transport (e.g. IPC).
    TransportCapability(String),
    /// Any other platform-reported error, carrying the GraphQL error code.
    Platform {
        /// Human-readable message from the platform.
        message: String,
        /// The GraphQL extensions.code value.
        code: String,
    },
    /// Transport-level failure (network, serialization).
    Transport(String),
}

impl fmt::Display for Error {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Error::Auth(message) => write!(f, "whodb: authentication failed: {message}"),
            Error::NotFound(message) => write!(f, "whodb: not found: {message}"),
            Error::Validation(message) => write!(f, "whodb: invalid request: {message}"),
            Error::Version(message) => write!(f, "whodb: sdk outdated for platform API: {message}"),
            Error::CliCredentials(message) => {
                write!(f, "whodb: cli credentials unavailable: {message}")
            }
            Error::TransportCapability(message) => {
                write!(
                    f,
                    "whodb: operation not available over this transport: {message}"
                )
            }
            Error::Platform { message, code } => {
                write!(f, "whodb: platform error {code}: {message}")
            }
            Error::Transport(message) => write!(f, "whodb: transport error: {message}"),
        }
    }
}

impl std::error::Error for Error {}

/// SDK result alias.
pub type Result<T> = std::result::Result<T, Error>;

/// Maps a GraphQL errors array to the SDK error taxonomy. The first error
/// decides the type.
pub(crate) fn map_graphql_errors(errors: &[serde_json::Value]) -> Error {
    let first = errors.first();
    let message = first
        .and_then(|e| e.get("message"))
        .and_then(|m| m.as_str())
        .unwrap_or("unknown platform error")
        .to_string();
    let code = first
        .and_then(|e| e.get("extensions"))
        .and_then(|x| x.get("code"))
        .and_then(|c| c.as_str())
        .unwrap_or("");
    match code {
        "UNAUTHENTICATED" | "FORBIDDEN" => Error::Auth(message),
        "NOT_FOUND" => Error::NotFound(message),
        "BAD_USER_INPUT" | "GRAPHQL_VALIDATION_FAILED" => Error::Validation(message),
        _ => Error::Platform {
            message,
            code: code.to_string(),
        },
    }
}

const UNKNOWN_OPERATION_MARKERS: &[&str] = &[
    "cannot query field",
    "unknown field",
    "unknown type",
    "has no field",
];

/// Converts "unknown operation" server rejections — the operation was removed
/// after this SDK release — into the actionable upgrade error.
pub(crate) fn interpret_server_error(error: Error) -> Error {
    let message = error.to_string().to_lowercase();
    if UNKNOWN_OPERATION_MARKERS
        .iter()
        .any(|marker| message.contains(marker))
    {
        return Error::Version(format!(
            "this SDK ({}) was built for an older WhoDB platform API; upgrade the whodb-sdk crate",
            crate::SDK_VERSION
        ));
    }
    error
}
