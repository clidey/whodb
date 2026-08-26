"""IPC transport: runs the same facades inside the WhoDB Functions runtime.

The runtime exposes an in-container HTTP server (unix socket in Docker, TCP in
K8s) with ontology endpoints. This transport maps supported operations onto
those endpoints and reshapes their results into the GraphQL data shape the
generated core expects — the facades and hydration layer are oblivious.

Two impedance differences are absorbed here:
- GraphQL operations address entities by ID; IPC endpoints address them by
  apiName. The transport resolves IDs via a cached /entities call.
- GraphQL write mutations carry RecordInput pairs; IPC takes plain dicts.
"""

from __future__ import annotations

import json
import os
from typing import Any

import httpx

from ._errors import NotFoundError, PlatformError, TransportCapabilityError


def _record_inputs_to_data(values: list[dict]) -> dict[str, Any]:
    """Convert GraphQL RecordInput pairs back to a plain data dict."""
    return {record["Key"]: record["Value"] for record in values or []}


class IpcTransport:
    """Transport over the functions runtime's in-container IPC server.

    Operations outside the ontology surface raise TransportCapabilityError —
    datasets, sources, files, and workspace resolution are not available
    inside the runtime in v1.
    """

    def __init__(
        self,
        address: str | None = None,
        job_id: str | None = None,
        token: str | None = None,
    ):
        self._address = address or os.environ.get("WHODB_IPC_ADDRESS", "")
        self._job_id = job_id or os.environ.get("WHODB_JOB_ID", "")
        self._token = token or os.environ.get("WHODB_IPC_TOKEN", "")
        self._entities_cache: list[dict] | None = None
        if self._address.startswith("/"):
            # Unix domain socket (Docker runtime).
            self._client = httpx.Client(
                transport=httpx.HTTPTransport(uds=self._address),
                base_url="http://whodb-ipc",
                timeout=30.0,
            )
        else:
            self._client = httpx.Client(base_url=f"http://{self._address}", timeout=30.0)

    def _post(self, path: str, body: dict[str, Any]) -> Any:
        response = self._client.post(
            path,
            json=body,
            headers={"X-Job-ID": self._job_id, "Authorization": self._token},
        )
        if response.status_code >= 400:
            raise PlatformError(
                f"IPC request {path} failed with HTTP {response.status_code}",
                f"IPC_{response.status_code}",
            )
        return response.json()

    def _entities(self) -> list[dict]:
        if self._entities_cache is None:
            self._entities_cache = self._post("/entities", {}) or []
        return self._entities_cache

    def _entity_name(self, entity_id: str) -> str:
        """Resolve a GraphQL entity ID to the apiName IPC endpoints expect."""
        for entity in self._entities():
            if entity.get("id") == entity_id:
                return entity.get("apiName", "")
        raise NotFoundError(f"ontology entity {entity_id} not found in this function's scope")

    def _entity_pk(self, entity_id: str) -> str:
        for entity in self._entities():
            if entity.get("id") == entity_id:
                return entity.get("primaryKey", "")
        return ""

    def execute(self, operation: str, document: str, variables: dict[str, Any]) -> dict[str, Any]:
        """Execute one operation via its IPC endpoint, reshaped to GraphQL form."""
        handler = getattr(self, f"_op_{operation}", None)
        if handler is None:
            raise TransportCapabilityError(
                f"{operation} is not available inside the function runtime in v1"
            )
        return {operation: handler(variables)}

    # ── Per-operation adapters ────────────────────────────────────────────

    def _op_OntologyEntities(self, variables: dict) -> Any:
        return self._entities()

    def _op_OntologyQuery(self, variables: dict) -> Any:
        query_input = dict(variables.get("input") or {})
        body = {key: value for key, value in query_input.items() if value is not None}
        where_json = body.pop("whereJson", None)
        if where_json:
            body["where"] = json.loads(where_json)
        return self._post("/query", body)

    def _op_OntologyDescribe(self, variables: dict) -> Any:
        return self._post("/describe", dict(variables.get("input") or {}))

    def _op_OntologyStats(self, variables: dict) -> Any:
        return self._post(
            "/stats",
            {
                "entity": self._entity_name(variables.get("id", "")),
                "property": variables.get("property"),
                "where": variables.get("where"),
            },
        )

    def _op_OntologySimilar(self, variables: dict) -> Any:
        similar_input = variables.get("input") or {}
        return self._post(
            "/similar",
            {
                "entity": self._entity_name(similar_input.get("entityId", "")),
                "rowId": similar_input.get("rowId"),
                "topK": similar_input.get("topK"),
                "properties": similar_input.get("properties"),
                "where": similar_input.get("where"),
            },
        )

    def _op_OntologyFastLookups(self, variables: dict) -> Any:
        raise TransportCapabilityError(
            "OntologyFastLookups is not available inside the function runtime in v1"
        )

    def _op_OntologyAggregate(self, variables: dict) -> Any:
        return self._post(
            "/query",
            {
                "entity": self._entity_name(variables.get("id", "")),
                "where": variables.get("where"),
                "groupBy": variables.get("groupBy"),
                "metrics": variables.get("metrics"),
                "sort": variables.get("sort"),
                "pageSize": variables.get("pageSize"),
                "offset": variables.get("pageOffset"),
            },
        )

    def _op_OntologyAddRow(self, variables: dict) -> Any:
        self._post(
            "/create",
            {
                "entity": self._entity_name(variables.get("entityId", "")),
                "data": _record_inputs_to_data(variables.get("values") or []),
            },
        )
        return {"Status": True}

    def _op_OntologyAddRows(self, variables: dict) -> Any:
        ids = self._post(
            "/create_many",
            {
                "entity": self._entity_name(variables.get("entityId", "")),
                "rows": [
                    _record_inputs_to_data(row.get("values") or [])
                    for row in variables.get("rows") or []
                ],
                "idempotencyKey": variables.get("idempotencyKey"),
            },
        )
        return {"inserted": len(ids or []), "ids": ids or []}

    def _op_OntologyUpdateRow(self, variables: dict) -> Any:
        entity_id = variables.get("entityId", "")
        data = _record_inputs_to_data(variables.get("values") or [])
        pk_key = self._entity_pk(entity_id)
        pk = data.pop(pk_key, "") if pk_key else ""
        self._post(
            "/update",
            {"entity": self._entity_name(entity_id), "pk": pk, "data": data},
        )
        return {"Status": True}

    def _op_OntologyDeleteRow(self, variables: dict) -> Any:
        entity_id = variables.get("entityId", "")
        data = _record_inputs_to_data(variables.get("values") or [])
        pk_key = self._entity_pk(entity_id)
        pk = data.get(pk_key, "") if pk_key else next(iter(data.values()), "")
        self._post("/delete", {"entity": self._entity_name(entity_id), "pk": pk})
        return {"Status": True}

    def _op_OntologyFollowLink(self, variables: dict) -> Any:
        return self._post(
            "/follow_link",
            {
                "entity": self._entity_name(variables.get("entityId", "")),
                "pk": variables.get("pk"),
                "link": variables.get("linkApiName"),
                "pageSize": variables.get("pageSize"),
                "offset": variables.get("pageOffset"),
            },
        )

    def _op_OntologyFollowIncomingLink(self, variables: dict) -> Any:
        return self._post(
            "/follow_incoming_link",
            {
                "entity": self._entity_name(variables.get("entityId", "")),
                "pk": variables.get("pk"),
                "sourceEntity": self._entity_name(variables.get("sourceEntityId", "")),
                "link": variables.get("linkApiName"),
                "pageSize": variables.get("pageSize"),
                "offset": variables.get("pageOffset"),
            },
        )
