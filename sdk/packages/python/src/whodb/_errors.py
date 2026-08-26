"""Error taxonomy for the whodb SDK, mirrored across all SDK languages."""

from __future__ import annotations


class WhoDBError(Exception):
    """Base class for all errors raised by the whodb SDK."""


class AuthError(WhoDBError):
    """Authentication failed: missing, invalid, expired, or revoked credentials."""


class NotFoundError(WhoDBError):
    """The requested resource does not exist or the caller cannot see it."""


class ValidationError(WhoDBError):
    """The request was rejected as invalid before execution."""


class WhoDBVersionError(WhoDBError):
    """This SDK release targets an older platform API; upgrade the package."""


class CliCredentialsError(WhoDBError):
    """The whodb CLI credential helper is unavailable or not logged in."""


class TransportCapabilityError(WhoDBError):
    """An operation is not available over the current transport (e.g. IPC)."""


class PlatformError(WhoDBError):
    """Any other platform-reported error, carrying the GraphQL error code."""

    def __init__(self, message: str, code: str):
        super().__init__(message)
        self.code = code


def map_graphql_errors(errors: list[dict]) -> WhoDBError:
    """Map a GraphQL errors array to the SDK error taxonomy.

    The first error decides the type; its code is preserved on PlatformError
    for callers that need to branch on specifics.
    """
    first = errors[0] if errors else {"message": "unknown platform error"}
    code = (first.get("extensions") or {}).get("code", "")
    message = first.get("message", "unknown platform error")
    if code in ("UNAUTHENTICATED", "FORBIDDEN"):
        return AuthError(message)
    if code == "NOT_FOUND":
        return NotFoundError(message)
    if code in ("BAD_USER_INPUT", "GRAPHQL_VALIDATION_FAILED"):
        return ValidationError(message)
    return PlatformError(message, code)
