"""WhoDB client: the entry point of the whodb SDK.

Configure with an API key (headless), a raw token, or nothing at all — local
development falls back to `whodb login` credentials via the CLI helper, and
inside the WhoDB Functions runtime the client auto-detects the IPC transport.
"""

from __future__ import annotations

import os
import re
from collections.abc import Callable
from typing import Any

from ._auth import (
    CallbackCredentials,
    CliCredentials,
    CredentialProvider,
    StaticCredentials,
)
from ._errors import ValidationError
from ._generated import operations as ops
from ._generated.manifest import MANIFEST_HASH
from ._manifest_check import interpret_server_error, warn_if_flagged
from ._transport import HttpTransport, Transport
from ._version import SDK_VERSION
from .dataset import DatasetHandle
from .ontology import OntologyHandle
from .source import SourceHandle, list_sources

DEFAULT_HOST = "https://app.whodb.com"

_UUID_PATTERN = re.compile(
    r"^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$", re.IGNORECASE
)


def _resolve_credentials(
    api_key: str | None,
    token: str | Callable[[], str] | None,
    credentials: CredentialProvider | None,
    env: dict | None = None,
) -> tuple[CredentialProvider, bool, bool]:
    """Apply the credential precedence: explicit args → WHODB_API_KEY → CLI.

    Returns (provider, using_cli, using_api_key).
    """
    environ = env if env is not None else os.environ
    if credentials is not None:
        return credentials, False, False
    if api_key:
        return StaticCredentials(api_key), False, True
    if token is not None:
        if callable(token):
            return CallbackCredentials(token), False, False
        return StaticCredentials(token), False, False
    env_key = environ.get("WHODB_API_KEY")
    if env_key:
        return StaticCredentials(env_key), False, True
    return CliCredentials(), True, False


class WhoDB:
    """Synchronous WhoDB platform client.

    ```python
    from whodb import WhoDB

    whodb = WhoDB(api_key=os.environ["WHODB_API_KEY"])
    user = whodb.ontology("User").get("u_123")
    ```
    """

    #: SHA-256 of the platform manifest this SDK release was generated from.
    manifest_hash = MANIFEST_HASH
    #: This SDK package's version.
    version = SDK_VERSION

    def __init__(
        self,
        api_key: str | None = None,
        token: str | Callable[[], str] | None = None,
        credentials: CredentialProvider | None = None,
        org: str | None = None,
        project: str | None = None,
        host: str | None = None,
        transport: Transport | None = None,
    ):
        environ = os.environ
        self._org_input = org or environ.get("WHODB_ORG")
        self._project_input = project or environ.get("WHODB_PROJECT")
        self._workspace: tuple[str, str] | None = None

        if transport is None and environ.get("WHODB_IPC_TOKEN"):
            # Functions runtime: no explicit transport + IPC env present.
            from ._transport_ipc import IpcTransport

            transport = IpcTransport()

        if transport is not None:
            # Custom transports (IPC, mocks) skip slug resolution: inputs are
            # taken as IDs verbatim (or left empty — IPC scopes server-side).
            self._transport = transport
            self._http: HttpTransport | None = None
            self._skip_resolution = True
            self._cli_provider: CliCredentials | None = None
            self._using_api_key = False
            return

        provider, using_cli, using_api_key = _resolve_credentials(api_key, token, credentials)
        self._cli_provider = provider if using_cli and isinstance(provider, CliCredentials) else None
        self._using_api_key = using_api_key
        self._skip_resolution = False
        resolved_host = host or environ.get("WHODB_HOST") or DEFAULT_HOST
        self._http = HttpTransport(resolved_host, provider)
        self._transport = self._http

    def _execute(self, request: ops.Request) -> Any:
        """Execute one prepared operation, mapping unknown-op rejections to
        the actionable version error."""
        try:
            data = self._transport.execute(request.operation, request.document, request.variables)
        except Exception as error:  # noqa: BLE001 — re-raise, possibly converted
            raise interpret_server_error(error, SDK_VERSION) from None
        return data.get(request.operation)

    def _resolve_workspace(self) -> tuple[str, str]:
        """Resolve org/project (slug or ID) to IDs once, stamping the
        workspace headers on the HTTP transport."""
        if self._workspace is not None:
            return self._workspace
        org_input = self._org_input
        project_input = self._project_input
        if (not org_input or not project_input) and self._cli_provider is not None:
            defaults = self._cli_provider.defaults()
            org_input = org_input or defaults.get("org_id")
            project_input = project_input or defaults.get("project_id")
        if (not org_input or not project_input) and self._using_api_key:
            # API keys carry their org, and the platform auto-resolves the
            # project when the key has exactly one grant — discover both.
            warn_if_flagged("MyWorkspace")
            mine = self._execute(ops.my_workspace_request({})) or {}
            org_input = org_input or mine.get("orgId")
            project_input = project_input or mine.get("projectId")
            if not project_input:
                raise ValidationError(
                    "this API key has access to multiple (or zero) projects — "
                    "pass project= to WhoDB() or set WHODB_PROJECT"
                )
        if self._skip_resolution:
            self._workspace = (org_input or "", project_input or "")
            return self._workspace
        if not org_input or not project_input:
            raise ValidationError(
                "org and project are required — pass org=/project= to WhoDB(), "
                "set WHODB_ORG/WHODB_PROJECT, or run: whodb use"
            )
        org_id = org_input
        if not _UUID_PATTERN.match(org_input):
            orgs = self._execute(ops.my_organizations_request({})) or []
            match = next((o for o in orgs if o.get("slug") == org_input or o.get("name") == org_input), None)
            if match is None:
                raise ValidationError(f'organization "{org_input}" not found for this account')
            org_id = match["id"]
        if self._http is not None:
            self._http.set_workspace(org_id, None)
        project_id = project_input
        if not _UUID_PATTERN.match(project_input):
            projects = self._execute(ops.projects_request({"orgId": org_id})) or []
            match = next(
                (p for p in projects if p.get("slug") == project_input or p.get("name") == project_input), None
            )
            if match is None:
                raise ValidationError(f'project "{project_input}" not found in this organization')
            project_id = match["id"]
        if self._http is not None:
            self._http.set_workspace(org_id, project_id)
        self._workspace = (org_id, project_id)
        return self._workspace

    def _project_id(self) -> str:
        return self._resolve_workspace()[1]

    def ontology(self, name: str) -> OntologyHandle:
        """Return a handle for one ontology entity, addressed by apiName."""
        return OntologyHandle(self._execute, self._project_id, name)

    def ontology_entities(self) -> list[dict]:
        """List all ontology entities in the project."""
        warn_if_flagged("OntologyEntities")
        return self._execute(ops.ontology_entities_request({"projectId": self._project_id()}))

    def dataset(self, name: str) -> DatasetHandle:
        """Return a handle for one dataset, addressed by name."""
        return DatasetHandle(self._execute, self._project_id, name)

    def source(self, source_id: str) -> SourceHandle:
        """Return a handle for one connected source, addressed by ID."""
        return SourceHandle(self._execute, self._project_id, source_id)

    def sources(self) -> list[dict]:
        """List the project's connected sources."""
        return list_sources(self._execute, self._project_id())


class AsyncWhoDB:
    """Asynchronous WhoDB platform client.

    Same facade surface as WhoDB but awaitable, over httpx.AsyncClient. The
    handles execute through an internal sync-style bridge, so behavior stays
    identical to the sync client.
    """

    manifest_hash = MANIFEST_HASH
    version = SDK_VERSION

    def __init__(
        self,
        api_key: str | None = None,
        token: str | Callable[[], str] | None = None,
        credentials: CredentialProvider | None = None,
        org: str | None = None,
        project: str | None = None,
        host: str | None = None,
    ):
        from ._transport import AsyncHttpTransport

        provider, _using_cli, using_api_key = _resolve_credentials(api_key, token, credentials)
        self._using_api_key = using_api_key
        self._org_input = org or os.environ.get("WHODB_ORG")
        self._project_input = project or os.environ.get("WHODB_PROJECT")
        resolved_host = host or os.environ.get("WHODB_HOST") or DEFAULT_HOST
        self._transport = AsyncHttpTransport(resolved_host, provider)
        self._workspace: tuple[str, str] | None = None

    async def _execute(self, request: ops.Request) -> Any:
        try:
            data = await self._transport.execute(request.operation, request.document, request.variables)
        except Exception as error:  # noqa: BLE001
            raise interpret_server_error(error, SDK_VERSION) from None
        return data.get(request.operation)

    async def _resolve_workspace(self) -> tuple[str, str]:
        if self._workspace is not None:
            return self._workspace
        org_input = self._org_input
        project_input = self._project_input
        if (not org_input or not project_input) and self._using_api_key:
            mine = await self._execute(ops.my_workspace_request({})) or {}
            org_input = org_input or mine.get("orgId")
            project_input = project_input or mine.get("projectId")
            if not project_input:
                raise ValidationError(
                    "this API key has access to multiple (or zero) projects — "
                    "pass project= to AsyncWhoDB() or set WHODB_PROJECT"
                )
        if not org_input or not project_input:
            raise ValidationError(
                "org and project are required — pass org=/project= to AsyncWhoDB() "
                "or set WHODB_ORG/WHODB_PROJECT"
            )
        org_id = org_input
        if not _UUID_PATTERN.match(org_input):
            orgs = await self._execute(ops.my_organizations_request({})) or []
            match = next((o for o in orgs if o.get("slug") == org_input or o.get("name") == org_input), None)
            if match is None:
                raise ValidationError(f'organization "{org_input}" not found for this account')
            org_id = match["id"]
        self._transport.set_workspace(org_id, None)
        project_id = project_input
        if not _UUID_PATTERN.match(project_input):
            projects = await self._execute(ops.projects_request({"orgId": org_id})) or []
            match = next(
                (p for p in projects if p.get("slug") == project_input or p.get("name") == project_input), None
            )
            if match is None:
                raise ValidationError(f'project "{project_input}" not found in this organization')
            project_id = match["id"]
        self._transport.set_workspace(org_id, project_id)
        self._workspace = (org_id, project_id)
        return self._workspace

    async def _project_id(self) -> str:
        return (await self._resolve_workspace())[1]

    def ontology(self, name: str):
        """Return an awaitable handle for one ontology entity."""
        from ._async_ontology import AsyncOntologyHandle

        return AsyncOntologyHandle(self._execute, self._project_id, name)

    async def ontology_entities(self) -> list[dict]:
        """List all ontology entities in the project."""
        project_id = await self._project_id()
        return await self._execute(ops.ontology_entities_request({"projectId": project_id}))
