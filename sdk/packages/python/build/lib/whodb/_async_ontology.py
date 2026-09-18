# Copyright 2026 Clidey, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

"""Async twin of OntologyHandle. Same behavior, awaitable surface."""

from __future__ import annotations

import json
from collections.abc import AsyncIterator, Awaitable, Callable
from typing import Any

from ._errors import NotFoundError, ValidationError
from ._generated import operations as ops
from ._hydrate import hydrate_rows, property_types_of
from ._manifest_check import warn_if_flagged
from .ontology import _DEFAULT_PAGE_SIZE, _to_record_inputs


class AsyncOntologyHandle:
    """Awaitable handle for one ontology entity, addressed by apiName."""

    def __init__(
        self,
        execute: Callable[[ops.Request], Awaitable[Any]],
        project_id: Callable[[], Awaitable[str]],
        api_name: str,
    ):
        self._execute = execute
        self._project_id = project_id
        self._api_name = api_name
        self._entity_cache: dict | None = None

    async def entity_meta(self) -> dict:
        """Resolve and cache the entity metadata backing this handle."""
        if self._entity_cache is not None:
            return self._entity_cache
        warn_if_flagged("OntologyEntities")
        entities = await self._execute(
            ops.ontology_entities_request({"projectId": await self._project_id()})
        )
        entity = next((e for e in entities or [] if e.get("apiName") == self._api_name), None)
        if entity is None:
            raise NotFoundError(f'ontology entity "{self._api_name}" not found in this project')
        self._entity_cache = entity
        return entity

    async def get(self, pk: Any) -> dict | None:
        """Fetch a single record by primary key, or None when absent."""
        entity = await self.entity_meta()
        primary_key = entity.get("primaryKey")
        if not primary_key:
            raise ValidationError(
                f'entity "{self._api_name}" has no primary key — use list() with a where filter'
            )
        warn_if_flagged("OntologyQuery")
        result = await self._execute(
            ops.ontology_query_request(
                {
                    "projectId": await self._project_id(),
                    "input": {
                        "entity": self._api_name,
                        "whereJson": json.dumps({primary_key: {"eq": str(pk)}}),
                        "pageSize": 1,
                        "offset": 0,
                    },
                }
            )
        )
        rows, _ = hydrate_rows(result, property_types_of(entity))
        return rows[0] if rows else None

    async def list(
        self,
        where: dict | None = None,
        sort: list[dict] | None = None,
        page_size: int = _DEFAULT_PAGE_SIZE,
        page_offset: int = 0,
    ) -> list[dict]:
        """List one page of records with optional filter/sort."""
        entity = await self.entity_meta()
        warn_if_flagged("OntologyQuery")
        result = await self._execute(
            ops.ontology_query_request(
                {
                    "projectId": await self._project_id(),
                    "input": {
                        "entity": self._api_name,
                        "whereJson": json.dumps(where) if where else None,
                        "sort": sort,
                        "pageSize": page_size,
                        "offset": page_offset,
                    },
                }
            )
        )
        rows, _ = hydrate_rows(result, property_types_of(entity))
        return rows

    async def pages(
        self, where: dict | None = None, page_size: int = _DEFAULT_PAGE_SIZE
    ) -> AsyncIterator[list[dict]]:
        """Iterate every page until a short page signals the end."""
        offset = 0
        while True:
            rows = await self.list(where=where, page_size=page_size, page_offset=offset)
            yield rows
            if len(rows) < page_size:
                return
            offset += page_size

    async def capabilities(self, record_key: Any | None = None) -> dict:
        """Compute the current actor's readable fields and allowed actions."""
        entity = await self.entity_meta()
        warn_if_flagged("OntologyRecordCapabilities")
        return await self._execute(
            ops.ontology_record_capabilities_request(
                {
                    "projectId": await self._project_id(),
                    "ontologyId": entity["id"],
                    "recordKey": None if record_key is None else str(record_key),
                }
            )
        )

    async def preview_action(
        self,
        action: str,
        record_key: Any | None,
        values: dict[str, Any] | None = None,
    ) -> dict:
        """Dry-run a named action without persisting records, effects, or events."""
        entity = await self.entity_meta()
        warn_if_flagged("PreviewOntologyAction")
        return await self._execute(
            ops.preview_ontology_action_request(
                {
                    "input": {
                        "projectId": await self._project_id(),
                        "ontologyId": entity["id"],
                        "recordKey": None if record_key is None else str(record_key),
                        "action": action,
                        "values": values or {},
                    }
                }
            )
        )

    async def action(
        self,
        action: str,
        record_key: Any | None,
        values: dict[str, Any] | None = None,
        *,
        expected_version: int | None = None,
        idempotency_key: str | None = None,
    ) -> dict:
        """Execute a named action with full behavior enforcement."""
        entity = await self.entity_meta()
        warn_if_flagged("ExecuteOntologyAction")
        return await self._execute(
            ops.execute_ontology_action_request(
                {
                    "input": {
                        "projectId": await self._project_id(),
                        "ontologyId": entity["id"],
                        "recordKey": None if record_key is None else str(record_key),
                        "action": action,
                        "values": values or {},
                        "expectedVersion": expected_version,
                        "idempotencyKey": idempotency_key,
                    }
                }
            )
        )

    async def action_executions(self, record_key: Any, limit: int = 50) -> list[dict]:
        """List recent named-action executions for one record."""
        entity = await self.entity_meta()
        warn_if_flagged("OntologyActionExecutions")
        return await self._execute(
            ops.ontology_action_executions_request(
                {
                    "projectId": await self._project_id(),
                    "ontologyId": entity["id"],
                    "recordKey": str(record_key),
                    "limit": limit,
                }
            )
        )

    async def create(self, values: dict[str, Any]) -> None:
        """Insert one record."""
        entity = await self.entity_meta()
        warn_if_flagged("OntologyAddRow")
        await self._execute(
            ops.ontology_add_row_request(
                {
                    "projectId": await self._project_id(),
                    "entityId": entity["id"],
                    "values": _to_record_inputs(values),
                }
            )
        )

    async def create_many(self, rows: list[dict[str, Any]], idempotency_key: str | None = None) -> dict:
        """Insert many records with optional idempotency key."""
        entity = await self.entity_meta()
        warn_if_flagged("OntologyAddRows")
        return await self._execute(
            ops.ontology_add_rows_request(
                {
                    "projectId": await self._project_id(),
                    "entityId": entity["id"],
                    "rows": [{"values": _to_record_inputs(row)} for row in rows],
                    "idempotencyKey": idempotency_key,
                }
            )
        )

    async def update(self, pk: Any, values: dict[str, Any]) -> None:
        """Update one record identified by primary key."""
        entity = await self.entity_meta()
        primary_key = entity.get("primaryKey")
        if not primary_key:
            raise ValidationError(
                f'entity "{self._api_name}" has no primary key — updates are not supported'
            )
        warn_if_flagged("OntologyUpdateRow")
        await self._execute(
            ops.ontology_update_row_request(
                {
                    "projectId": await self._project_id(),
                    "entityId": entity["id"],
                    "values": _to_record_inputs({**values, primary_key: str(pk)}),
                    "updatedColumns": list(values.keys()),
                }
            )
        )

    async def delete(self, pk: Any) -> None:
        """Delete one record identified by primary key."""
        entity = await self.entity_meta()
        primary_key = entity.get("primaryKey")
        if not primary_key:
            raise ValidationError(
                f'entity "{self._api_name}" has no primary key — deletes are not supported'
            )
        warn_if_flagged("OntologyDeleteRow")
        await self._execute(
            ops.ontology_delete_row_request(
                {
                    "projectId": await self._project_id(),
                    "entityId": entity["id"],
                    "values": _to_record_inputs({primary_key: str(pk)}),
                }
            )
        )
