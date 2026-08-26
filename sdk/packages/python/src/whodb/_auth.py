"""Credential providers, mirrored from the TypeScript SDK's auth.ts."""

from __future__ import annotations

import json
import subprocess
from datetime import datetime, timedelta, timezone
from typing import Callable, Optional, Protocol

from ._errors import AuthError, CliCredentialsError

_CLI_REFRESH_SKEW = timedelta(seconds=60)


class CredentialProvider(Protocol):
    """Yields a bearer credential; refresh() runs once after a 401."""

    def token(self) -> str: ...

    def refresh(self) -> None: ...

    def defaults(self) -> dict:
        """Workspace defaults carried by the credential source, if any."""
        ...


class StaticCredentials:
    """Static API-key or raw-token credentials (production/headless usage)."""

    def __init__(self, value: str):
        self._value = value

    def token(self) -> str:
        """Return the configured credential."""
        return self._value

    def refresh(self) -> None:
        """Static credentials cannot refresh; no-op."""

    def defaults(self) -> dict:
        """Static credentials carry no workspace defaults."""
        return {}


class CallbackCredentials:
    """Caller-managed token callback."""

    def __init__(self, callback: Callable[[], str]):
        self._callback = callback

    def token(self) -> str:
        """Return a token from the caller's callback."""
        return self._callback()

    def refresh(self) -> None:
        """The callback is consulted every call; nothing to invalidate."""

    def defaults(self) -> dict:
        """Callback credentials carry no workspace defaults."""
        return {}


class CliCredentials:
    """CLI credentials: exec `whodb auth print-token` and cache until expiry.

    The gcloud-ADC pattern for local development — requires the whodb CLI on
    PATH and a prior `whodb login`.
    """

    def __init__(self, command: str = "whodb"):
        self._command = command
        self._cached: Optional[dict] = None

    def _exec(self) -> dict:
        try:
            completed = subprocess.run(
                [self._command, "auth", "print-token", "--format", "json"],
                capture_output=True,
                timeout=15,
                check=False,
            )
        except FileNotFoundError as exc:
            raise CliCredentialsError(
                "whodb CLI not found — install it or set WHODB_API_KEY"
            ) from exc
        if completed.returncode != 0:
            detail = completed.stderr.decode(errors="replace").strip()
            raise CliCredentialsError(f"whodb auth print-token failed: {detail}")
        try:
            return json.loads(completed.stdout)
        except json.JSONDecodeError as exc:
            raise CliCredentialsError("whodb auth print-token returned invalid JSON") from exc

    def _is_fresh(self, entry: dict) -> bool:
        expires_at = entry.get("expires_at")
        if not expires_at:
            return False  # no expiry info — re-exec every call
        expiry = datetime.fromisoformat(expires_at.replace("Z", "+00:00"))
        return expiry - datetime.now(timezone.utc) > _CLI_REFRESH_SKEW

    def token(self) -> str:
        """Return a fresh access token, re-execing the CLI near expiry."""
        if self._cached is None or not self._is_fresh(self._cached):
            self._cached = self._exec()
        token = self._cached.get("access_token")
        if not token:
            raise AuthError("whodb CLI returned an empty access token")
        return token

    def refresh(self) -> None:
        """Drop the cached token so the next call re-execs the CLI."""
        self._cached = None

    def defaults(self) -> dict:
        """Return the CLI's saved host/org/project defaults."""
        if self._cached is None:
            self._cached = self._exec()
        return {
            "host": self._cached.get("host"),
            "org_id": self._cached.get("org_id"),
            "project_id": self._cached.get("project_id"),
        }
