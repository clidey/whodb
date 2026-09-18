//! Credential providers, mirrored from the TypeScript SDK's auth module.

use serde_json::Value;
use std::process::Command;
use std::sync::Mutex;
use std::time::{Duration, SystemTime};

use crate::error::{Error, Result};

/// Yields a bearer credential; refresh() is invoked once after a 401.
pub trait CredentialProvider: Send + Sync {
    /// Returns the bearer credential for the next request.
    fn token(&self) -> Result<String>;
    /// Invalidates any cached credential.
    fn refresh(&self);
    /// Workspace defaults carried by the credential source, if any:
    /// (host, org_id, project_id).
    fn defaults(&self) -> Option<(String, String, String)> {
        None
    }
}

/// Static API-key or raw-token credentials (production/headless usage).
pub struct StaticCredentials(pub String);

impl CredentialProvider for StaticCredentials {
    fn token(&self) -> Result<String> {
        Ok(self.0.clone())
    }
    fn refresh(&self) {}
}

const CLI_REFRESH_SKEW: Duration = Duration::from_secs(60);

struct CliCache {
    token: String,
    expires_at: Option<SystemTime>,
    host: String,
    org_id: String,
    project_id: String,
}

/// CLI credentials: exec `whodb auth print-token` and cache until shortly
/// before expiry — the gcloud-ADC pattern for local development. Requires
/// the whodb CLI on PATH and a prior `whodb login`.
pub struct CliCredentials {
    command: String,
    cache: Mutex<Option<CliCache>>,
}

impl CliCredentials {
    /// Creates the provider using the `whodb` binary on PATH.
    pub fn new() -> Self {
        CliCredentials {
            command: "whodb".to_string(),
            cache: Mutex::new(None),
        }
    }

    fn exec(&self) -> Result<CliCache> {
        let output = Command::new(&self.command)
            .args(["auth", "print-token", "--format", "json"])
            .output()
            .map_err(|error| {
                if error.kind() == std::io::ErrorKind::NotFound {
                    Error::CliCredentials(
                        "whodb CLI not found — install it or set WHODB_API_KEY".to_string(),
                    )
                } else {
                    Error::CliCredentials(error.to_string())
                }
            })?;
        if !output.status.success() {
            return Err(Error::CliCredentials(format!(
                "whodb auth print-token failed: {}",
                String::from_utf8_lossy(&output.stderr).trim()
            )));
        }
        let parsed: Value = serde_json::from_slice(&output.stdout).map_err(|_| {
            Error::CliCredentials("whodb auth print-token returned invalid JSON".to_string())
        })?;
        let field = |name: &str| {
            parsed
                .get(name)
                .and_then(Value::as_str)
                .unwrap_or_default()
                .to_string()
        };
        let expires_at = parsed
            .get("expires_at")
            .and_then(Value::as_str)
            .and_then(parse_rfc3339);
        Ok(CliCache {
            token: field("access_token"),
            expires_at,
            host: field("host"),
            org_id: field("org_id"),
            project_id: field("project_id"),
        })
    }

    fn fresh(entry: &CliCache) -> bool {
        match entry.expires_at {
            // No expiry info — re-exec every call.
            None => false,
            Some(expiry) => expiry
                .duration_since(SystemTime::now())
                .map(|remaining| remaining > CLI_REFRESH_SKEW)
                .unwrap_or(false),
        }
    }
}

impl Default for CliCredentials {
    fn default() -> Self {
        Self::new()
    }
}

impl CredentialProvider for CliCredentials {
    fn token(&self) -> Result<String> {
        let mut cache = self.cache.lock().expect("cli cache lock poisoned");
        let needs_exec = !cache.as_ref().map(Self::fresh).unwrap_or(false);
        if needs_exec {
            *cache = Some(self.exec()?);
        }
        let entry = cache.as_ref().expect("cache populated above");
        if entry.token.is_empty() {
            return Err(Error::Auth(
                "whodb CLI returned an empty access token".to_string(),
            ));
        }
        Ok(entry.token.clone())
    }

    fn refresh(&self) {
        *self.cache.lock().expect("cli cache lock poisoned") = None;
    }

    fn defaults(&self) -> Option<(String, String, String)> {
        let mut cache = self.cache.lock().expect("cli cache lock poisoned");
        if cache.is_none() {
            *cache = self.exec().ok();
        }
        cache.as_ref().map(|entry| {
            (
                entry.host.clone(),
                entry.org_id.clone(),
                entry.project_id.clone(),
            )
        })
    }
}

/// Parses an RFC3339 timestamp into SystemTime without a chrono dependency.
/// Only the "seconds since epoch" precision needed for expiry skew.
fn parse_rfc3339(text: &str) -> Option<SystemTime> {
    // Minimal parser: YYYY-MM-DDTHH:MM:SS with optional fraction/Z or offset.
    let bytes = text.as_bytes();
    if bytes.len() < 19 || bytes[4] != b'-' || bytes[7] != b'-' || bytes[10] != b'T' {
        return None;
    }
    let digits = |from: usize, to: usize| text.get(from..to)?.parse::<i64>().ok();
    let (year, month, day) = (digits(0, 4)?, digits(5, 7)?, digits(8, 10)?);
    let (hour, minute, second) = (digits(11, 13)?, digits(14, 16)?, digits(17, 19)?);
    // Days since epoch via civil-from-days inverse (Howard Hinnant's algorithm).
    let years = if month <= 2 { year - 1 } else { year };
    let era = if years >= 0 { years } else { years - 399 } / 400;
    let year_of_era = years - era * 400;
    let month_adjusted = if month > 2 { month - 3 } else { month + 9 };
    let day_of_year = (153 * month_adjusted + 2) / 5 + day - 1;
    let day_of_era = year_of_era * 365 + year_of_era / 4 - year_of_era / 100 + day_of_year;
    let days = era * 146097 + day_of_era - 719468;
    let seconds = days * 86400 + hour * 3600 + minute * 60 + second;
    if seconds < 0 {
        return None;
    }
    Some(SystemTime::UNIX_EPOCH + Duration::from_secs(seconds as u64))
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::fs;
    use std::io::Write;
    use std::os::unix::fs::PermissionsExt;
    use std::path::PathBuf;

    /// Writes a fake whodb CLI script; appends to a counter file per run.
    fn fake_cli(body: &str) -> (CliCredentials, PathBuf) {
        let dir = std::env::temp_dir().join(format!(
            "whodb-cli-test-{}-{:?}",
            std::process::id(),
            std::thread::current().id(),
        ));
        fs::create_dir_all(&dir).expect("create test dir");
        let count_file = dir.join("count");
        let _ = fs::remove_file(&count_file);
        let command = dir.join("whodb");
        let mut file = fs::File::create(&command).expect("write fake cli");
        write!(
            file,
            "#!/bin/sh\necho x >> \"{}\"\n{}\n",
            count_file.display(),
            body
        )
        .expect("script body");
        fs::set_permissions(&command, fs::Permissions::from_mode(0o755)).expect("chmod");
        let credentials = CliCredentials {
            command: command.display().to_string(),
            cache: Mutex::new(None),
        };
        (credentials, count_file)
    }

    fn exec_count(count_file: &PathBuf) -> usize {
        fs::read(count_file).map(|data| data.len() / 2).unwrap_or(0)
    }

    fn future_expiry(seconds: u64) -> String {
        // Format a UTC RFC3339 timestamp `seconds` ahead using only std.
        let now = SystemTime::now()
            .duration_since(SystemTime::UNIX_EPOCH)
            .expect("clock")
            .as_secs()
            + seconds;
        let days = now / 86400;
        let time_of_day = now % 86400;
        // civil-from-days (Howard Hinnant) — inverse of parse_rfc3339 below.
        let days_shifted = days as i64 + 719_468;
        let era = days_shifted.div_euclid(146_097);
        let day_of_era = days_shifted - era * 146_097;
        let year_of_era =
            (day_of_era - day_of_era / 1460 + day_of_era / 36_524 - day_of_era / 146_096) / 365;
        let year = year_of_era + era * 400;
        let day_of_year = day_of_era - (365 * year_of_era + year_of_era / 4 - year_of_era / 100);
        let month_prime = (5 * day_of_year + 2) / 153;
        let day = day_of_year - (153 * month_prime + 2) / 5 + 1;
        let month = if month_prime < 10 {
            month_prime + 3
        } else {
            month_prime - 9
        };
        let year = if month <= 2 { year + 1 } else { year };
        format!(
            "{:04}-{:02}-{:02}T{:02}:{:02}:{:02}Z",
            year,
            month,
            day,
            time_of_day / 3600,
            (time_of_day % 3600) / 60,
            time_of_day % 60,
        )
    }

    #[test]
    fn missing_binary_is_cli_credentials_error() {
        let credentials = CliCredentials {
            command: "/no/such/whodb-binary".to_string(),
            cache: Mutex::new(None),
        };
        assert!(matches!(credentials.token(), Err(Error::CliCredentials(_))));
    }

    #[test]
    fn invalid_json_is_cli_credentials_error() {
        let (credentials, _count) = fake_cli("echo not-json");
        assert!(matches!(credentials.token(), Err(Error::CliCredentials(_))));
    }

    #[test]
    fn nonzero_exit_is_cli_credentials_error() {
        let (credentials, _count) = fake_cli("echo 'run: whodb login' >&2; exit 1");
        assert!(matches!(credentials.token(), Err(Error::CliCredentials(_))));
    }

    #[test]
    fn fresh_tokens_are_cached() {
        let (credentials, count_file) = fake_cli(&format!(
            "echo '{{\"access_token\":\"tok-1\",\"expires_at\":\"{}\"}}'",
            future_expiry(3600),
        ));
        for _ in 0..3 {
            assert_eq!(credentials.token().expect("token"), "tok-1");
        }
        assert_eq!(exec_count(&count_file), 1);
    }

    #[test]
    fn near_expiry_tokens_reexec() {
        let (credentials, count_file) = fake_cli(&format!(
            "echo '{{\"access_token\":\"tok-1\",\"expires_at\":\"{}\"}}'",
            future_expiry(30), // inside the 60s refresh skew
        ));
        credentials.token().expect("token");
        credentials.token().expect("token");
        assert_eq!(exec_count(&count_file), 2);
    }

    #[test]
    fn refresh_drops_the_cache() {
        let (credentials, count_file) = fake_cli(&format!(
            "echo '{{\"access_token\":\"tok-1\",\"expires_at\":\"{}\"}}'",
            future_expiry(3600),
        ));
        credentials.token().expect("token");
        credentials.refresh();
        credentials.token().expect("token");
        assert_eq!(exec_count(&count_file), 2);
    }
}
