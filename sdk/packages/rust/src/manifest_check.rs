//! Two-tier versioning-policy checks (SDK design §2.3).

use std::collections::HashSet;
use std::sync::Mutex;
use std::sync::OnceLock;

use crate::gen::manifest::EMBEDDED_MANIFEST;

fn warned() -> &'static Mutex<HashSet<&'static str>> {
    static WARNED: OnceLock<Mutex<HashSet<&'static str>>> = OnceLock::new();
    WARNED.get_or_init(|| Mutex::new(HashSet::new()))
}

/// Emits the once-per-process deprecation/behavior-change warning for an
/// operation. Actually-removed operations surface as Error::Version.
pub(crate) fn warn_if_flagged(operation: &'static str) {
    let Some((_, policy)) = EMBEDDED_MANIFEST
        .iter()
        .find(|(name, _)| *name == operation)
    else {
        return;
    };
    if !policy.deprecated && !policy.behavior_changed {
        return;
    }
    let mut seen = warned().lock().expect("warned lock poisoned");
    if !seen.insert(operation) {
        return;
    }
    if policy.deprecated {
        let sunset = if policy.sunset_at.is_empty() {
            String::new()
        } else {
            format!(" and will be removed after {}", policy.sunset_at)
        };
        eprintln!(
            "[whodb] {operation} is deprecated{sunset} — upgrade the whodb-sdk crate before then. {}",
            policy.note
        );
    } else {
        eprintln!(
            "[whodb] {operation}'s behavior changed in this platform release — results may differ from previous SDK versions. {}",
            policy.note
        );
    }
}
