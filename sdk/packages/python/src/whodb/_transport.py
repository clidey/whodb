"""Transports: how prepared operations reach the platform.

The generated core and the facades never speak HTTP directly — everything
routes through the Transport protocol, which is what lets the same facades run
over GraphQL/HTTP externally and over IPC inside the functions runtime.
"""

from __future__ import annotations

from typing import Any, Optional, Protocol

import httpx

from ._auth import CredentialProvider
from ._errors import AuthError, PlatformError, map_graphql_errors
from ._version import SDK_VERSION

_RETRYABLE_STATUS = {502, 503, 504}


class Transport(Protocol):
    """Executes one platform operation, returning the GraphQL data object."""

    def execute(self, operation: str, document: str, variables: dict[str, Any]) -> dict[str, Any]: ...


class AsyncTransport(Protocol):
    """Async twin of Transport."""

    async def execute(self, operation: str, document: str, variables: dict[str, Any]) -> dict[str, Any]: ...


def _interpret_response(operation: str, status_code: int, payload: Optional[dict]) -> dict[str, Any]:
    """Shared response handling for the sync and async HTTP transports."""
    if status_code == 401:
        raise AuthError("authentication failed — check your API key or run: whodb login")
    if status_code >= 400:
        raise PlatformError(f"platform request failed with HTTP {status_code}", f"HTTP_{status_code}")
    if payload is None:
        raise PlatformError(f"invalid response for {operation}", "INVALID_RESPONSE")
    errors = payload.get("errors")
    if errors:
        raise map_graphql_errors(errors)
    data = payload.get("data")
    if data is None:
        raise PlatformError(f"empty response for {operation}", "EMPTY_RESPONSE")
    return data


class _HttpBase:
    """Config shared by the sync and async HTTP transports."""

    def __init__(self, host: str, credentials: CredentialProvider):
        self._endpoint = host.rstrip("/") + "/api/query"
        self._credentials = credentials
        self._org_id: Optional[str] = None
        self._project_id: Optional[str] = None

    def set_workspace(self, org_id: Optional[str], project_id: Optional[str]) -> None:
        """Set the workspace scope headers used on subsequent requests."""
        self._org_id = org_id
        self._project_id = project_id

    def _headers(self) -> dict[str, str]:
        headers = {
            "Content-Type": "application/json",
            "Authorization": f"Bearer {self._credentials.token()}",
            "User-Agent": f"clidey-whodb-python/{SDK_VERSION}",
        }
        if self._org_id:
            headers["X-Whodb-Org-Id"] = self._org_id
        if self._project_id:
            headers["X-Whodb-Project-Id"] = self._project_id
        return headers


class HttpTransport(_HttpBase):
    """Synchronous GraphQL-over-HTTP transport (POST /api/query)."""

    def __init__(self, host: str, credentials: CredentialProvider, client: Optional[httpx.Client] = None):
        super().__init__(host, credentials)
        self._client = client or httpx.Client(timeout=30.0)

    def execute(self, operation: str, document: str, variables: dict[str, Any]) -> dict[str, Any]:
        """Execute one operation with a single 401-refresh and 5xx retry."""
        body = {"query": document, "variables": variables}
        response = self._client.post(self._endpoint, json=body, headers=self._headers())
        if response.status_code == 401:
            self._credentials.refresh()
            response = self._client.post(self._endpoint, json=body, headers=self._headers())
        elif response.status_code in _RETRYABLE_STATUS:
            response = self._client.post(self._endpoint, json=body, headers=self._headers())
        payload = response.json() if response.headers.get("content-type", "").startswith("application/json") else None
        return _interpret_response(operation, response.status_code, payload)


class AsyncHttpTransport(_HttpBase):
    """Asynchronous GraphQL-over-HTTP transport (POST /api/query)."""

    def __init__(self, host: str, credentials: CredentialProvider, client: Optional[httpx.AsyncClient] = None):
        super().__init__(host, credentials)
        self._client = client or httpx.AsyncClient(timeout=30.0)

    async def execute(self, operation: str, document: str, variables: dict[str, Any]) -> dict[str, Any]:
        """Execute one operation with a single 401-refresh and 5xx retry."""
        body = {"query": document, "variables": variables}
        response = await self._client.post(self._endpoint, json=body, headers=self._headers())
        if response.status_code == 401:
            self._credentials.refresh()
            response = await self._client.post(self._endpoint, json=body, headers=self._headers())
        elif response.status_code in _RETRYABLE_STATUS:
            response = await self._client.post(self._endpoint, json=body, headers=self._headers())
        payload = response.json() if response.headers.get("content-type", "").startswith("application/json") else None
        return _interpret_response(operation, response.status_code, payload)
