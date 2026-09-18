"""Two-tier versioning-policy checks (see ee SDK_DESIGN.md §2.3)."""

from __future__ import annotations

import re
import warnings

from ._errors import WhoDBVersionError
from ._generated.manifest import EMBEDDED_MANIFEST

_warned: set[str] = set()

_UNKNOWN_OPERATION_PATTERNS = [
    re.compile(pattern, re.IGNORECASE)
    for pattern in (r"Cannot query field", r"Unknown field", r"Unknown type", r"has no field")
]


def warn_if_flagged(operation_name: str) -> None:
    """Emit the once-per-process deprecation/behavior-change warning."""
    entry = EMBEDDED_MANIFEST.get(operation_name)
    if not entry or operation_name in _warned:
        return
    if entry.get("deprecated"):
        _warned.add(operation_name)
        sunset = entry.get("sunsetAt")
        note = entry.get("note") or ""
        warnings.warn(
            f"[whodb] {operation_name} is deprecated"
            + (f" and will be removed after {sunset}" if sunset else "")
            + f" — upgrade the whodb-sdk package before then. {note}".rstrip(),
            DeprecationWarning,
            stacklevel=3,
        )
    elif entry.get("behaviorChanged"):
        _warned.add(operation_name)
        note = entry.get("note") or ""
        warnings.warn(
            f"[whodb] {operation_name}'s behavior changed in this platform release"
            f" — results may differ from previous SDK versions. {note}".rstrip(),
            UserWarning,
            stacklevel=3,
        )


def interpret_server_error(error: Exception, sdk_version: str) -> Exception:
    """Convert 'unknown operation' server rejections into WhoDBVersionError."""
    message = str(error)
    if any(pattern.search(message) for pattern in _UNKNOWN_OPERATION_PATTERNS):
        return WhoDBVersionError(
            f"this SDK ({sdk_version}) was built for an older WhoDB platform API; "
            "upgrade the whodb-sdk package"
        )
    return error


def reset_warnings() -> None:
    """Test hook: clear the once-per-process warning memory."""
    _warned.clear()
