"""The whodb.ontology("User") facade: reads and record writes for one
ontology entity, addressed by apiName. Mirrors the TypeScript OntologyHandle;
languages differ in ergonomics, never in behavior (conformance-pinned)."""

from __future__ import annotations

import json
from typing import Any, Callable, Optional

from ._errors import NotFoundError, ValidationError
from ._generated import operations as ops
from ._hydrate import hydrate_rows, property_types_of
from ._manifest_check import warn_if_flagged
from ._pagination import ListCall, Page

_DEFAULT_PAGE_SIZE = 100


def _to_record_inputs(values: dict[str, Any]) -> list[dict[str, str]]:
    records = []
    for key, value in values.items():
        if value is None:
            encoded = ""
        elif isinstance(value, (dict, list)):
            encoded = json.dumps(value)
        elif isinstance(value, bool):
            encoded = "true" if value else "false"
        else:
            encoded = str(value)
        records.append({"Key": key, "Value": encoded})
    return records


class OntologyHandle:
    """Typed-by-metadata handle for one ontology entity."""

    def __init__(self, execute: Callable[[ops.Request], Any], project_id: Callable[[], str], api_name: str):
        self._execute = execute
        self._project_id = project_id
        self._api_name = api_name
        self._entity_cache: Optional[dict] = None

    def entity_meta(self) -> dict:
        """Resolve and cache the entity metadata backing this handle."""
        if self._entity_cache is not None:
            return self._entity_cache
        warn_if_flagged("OntologyEntities")
        entities = self._execute(
            ops.ontology_entities_request({"projectId": self._project_id()})
        )
        entity = next((e for e in entities or [] if e.get("apiName") == self._api_name), None)
        if entity is None:
            raise NotFoundError(f'ontology entity "{self._api_name}" not found in this project')
        self._entity_cache = entity
        return entity

    def _property_types(self) -> dict[str, str]:
        return property_types_of(self.entity_meta())

    def describe(self) -> dict:
        """Describe the entity: schema, properties, links."""
        self.entity_meta()  # NotFoundError for unknown entities
        warn_if_flagged("OntologyDescribe")
        return self._execute(
            ops.ontology_describe_request(
                {
                    "projectId": self._project_id(),
                    "input": {"entities": [self._api_name], "includeInferred": True},
                }
            )
        )

    def get(self, pk: Any) -> Optional[dict]:
        """Fetch a single record by primary key, or None when absent."""
        entity = self.entity_meta()
        primary_key = entity.get("primaryKey")
        if not primary_key:
            raise ValidationError(
                f'entity "{self._api_name}" has no primary key — use list() with a where filter'
            )
        warn_if_flagged("OntologyQuery")
        result = self._execute(
            ops.ontology_query_request(
                {
                    "projectId": self._project_id(),
                    "input": {
                        "entity": self._api_name,
                        "whereJson": json.dumps({primary_key: {"eq": str(pk)}}),
                        "pageSize": 1,
                        "offset": 0,
                    },
                }
            )
        )
        rows, _ = hydrate_rows(result, self._property_types())
        return rows[0] if rows else None

    def list(
        self,
        where: Optional[dict] = None,
        sort: Optional[list[dict]] = None,
        page_size: int = _DEFAULT_PAGE_SIZE,
    ) -> ListCall:
        """List records with optional filter/sort; iterate or call .pages()."""

        def fetch_page(page_offset: int) -> Page:
            property_types = self._property_types()
            warn_if_flagged("OntologyQuery")
            result = self._execute(
                ops.ontology_query_request(
                    {
                        "projectId": self._project_id(),
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
            rows, total = hydrate_rows(result, property_types)
            return Page(rows, total, page_offset)

        return ListCall(fetch_page, page_size)

    def query(self, **options: Any) -> list[dict]:
        """Flexible query: text search, joins, grouping, metrics.

        Keyword options mirror OntologyQueryInput (search, search_fields,
        joins, group_by, metrics, sort, page_size, offset, where_json).
        """
        rename = {
            "search_fields": "searchFields",
            "group_by": "groupBy",
            "page_size": "pageSize",
            "where_json": "whereJson",
            "scan_limit": "scanLimit",
        }
        query_input: dict[str, Any] = {"entity": self._api_name}
        for key, value in options.items():
            query_input[rename.get(key, key)] = value
        warn_if_flagged("OntologyQuery")
        result = self._execute(
            ops.ontology_query_request({"projectId": self._project_id(), "input": query_input})
        )
        rows, _ = hydrate_rows(result, self._property_types())
        return rows

    def aggregate(
        self,
        group_by: list[str],
        metrics: Optional[list[dict]] = None,
        where: Optional[dict] = None,
        sort: Optional[list[dict]] = None,
        page_size: int = _DEFAULT_PAGE_SIZE,
    ) -> list[dict]:
        """Aggregate records grouped by properties with metric functions."""
        entity = self.entity_meta()
        warn_if_flagged("OntologyAggregate")
        result = self._execute(
            ops.ontology_aggregate_request(
                {
                    "projectId": self._project_id(),
                    "id": entity["id"],
                    "groupBy": group_by,
                    "metrics": metrics or [],
                    "where": where,
                    "sort": sort,
                    "pageSize": page_size,
                    "pageOffset": 0,
                }
            )
        )
        rows, _ = hydrate_rows(result)
        return rows

    def stats(self, property_name: str, where: Optional[dict] = None) -> dict:
        """Statistical summary of one property."""
        entity = self.entity_meta()
        warn_if_flagged("OntologyStats")
        return self._execute(
            ops.ontology_stats_request(
                {
                    "projectId": self._project_id(),
                    "id": entity["id"],
                    "property": property_name,
                    "where": where,
                }
            )
        )

    def similar(self, row_id: str, top_k: int = 10, properties: Optional[list[str]] = None, where: Optional[dict] = None) -> dict:
        """Embedding-based similarity search over this entity's records."""
        entity = self.entity_meta()
        warn_if_flagged("OntologySimilar")
        return self._execute(
            ops.ontology_similar_request(
                {
                    "projectId": self._project_id(),
                    "input": {
                        "entityId": entity["id"],
                        "rowId": row_id,
                        "topK": top_k,
                        "properties": properties,
                        "where": where,
                    },
                }
            )
        )

    def follow_link(self, pk: Any, link_api_name: str, page_size: int = _DEFAULT_PAGE_SIZE) -> ListCall:
        """Follow an outgoing link from one record to its related records."""

        def fetch_page(page_offset: int) -> Page:
            entity = self.entity_meta()
            warn_if_flagged("OntologyFollowLink")
            result = self._execute(
                ops.ontology_follow_link_request(
                    {
                        "projectId": self._project_id(),
                        "entityId": entity["id"],
                        "pk": str(pk),
                        "linkApiName": link_api_name,
                        "pageSize": page_size,
                        "pageOffset": page_offset,
                    }
                )
            )
            rows, total = hydrate_rows(result)
            return Page(rows, total, page_offset)

        return ListCall(fetch_page, page_size)

    def follow_incoming_link(
        self, pk: Any, source_entity_api_name: str, link_api_name: str, page_size: int = _DEFAULT_PAGE_SIZE
    ) -> ListCall:
        """Follow a link inbound from another entity's records to this record."""

        def fetch_page(page_offset: int) -> Page:
            entity = self.entity_meta()
            warn_if_flagged("OntologyEntities")
            entities = self._execute(
                ops.ontology_entities_request({"projectId": self._project_id()})
            )
            source = next(
                (e for e in entities or [] if e.get("apiName") == source_entity_api_name), None
            )
            if source is None:
                raise NotFoundError(
                    f'ontology entity "{source_entity_api_name}" not found in this project'
                )
            warn_if_flagged("OntologyFollowIncomingLink")
            result = self._execute(
                ops.ontology_follow_incoming_link_request(
                    {
                        "projectId": self._project_id(),
                        "entityId": entity["id"],
                        "pk": str(pk),
                        "sourceEntityId": source["id"],
                        "linkApiName": link_api_name,
                        "pageSize": page_size,
                        "pageOffset": page_offset,
                    }
                )
            )
            rows, total = hydrate_rows(result)
            return Page(rows, total, page_offset)

        return ListCall(fetch_page, page_size)

    def fast_lookups(self) -> list[dict]:
        """List the entity's fast lookups."""
        entity = self.entity_meta()
        warn_if_flagged("OntologyFastLookups")
        return self._execute(
            ops.ontology_fast_lookups_request(
                {"projectId": self._project_id(), "entityId": entity["id"]}
            )
        )

    def create(self, values: dict[str, Any]) -> None:
        """Insert one record. Values are field name/value pairs."""
        entity = self.entity_meta()
        warn_if_flagged("OntologyAddRow")
        self._execute(
            ops.ontology_add_row_request(
                {
                    "projectId": self._project_id(),
                    "entityId": entity["id"],
                    "values": _to_record_inputs(values),
                }
            )
        )

    def create_many(self, rows: list[dict[str, Any]], idempotency_key: Optional[str] = None) -> dict:
        """Insert many records; idempotency_key makes safe retries possible."""
        entity = self.entity_meta()
        warn_if_flagged("OntologyAddRows")
        return self._execute(
            ops.ontology_add_rows_request(
                {
                    "projectId": self._project_id(),
                    "entityId": entity["id"],
                    "rows": [{"values": _to_record_inputs(row)} for row in rows],
                    "idempotencyKey": idempotency_key,
                }
            )
        )

    def update(self, pk: Any, values: dict[str, Any]) -> None:
        """Update one record identified by primary key."""
        entity = self.entity_meta()
        primary_key = entity.get("primaryKey")
        if not primary_key:
            raise ValidationError(
                f'entity "{self._api_name}" has no primary key — updates are not supported'
            )
        warn_if_flagged("OntologyUpdateRow")
        self._execute(
            ops.ontology_update_row_request(
                {
                    "projectId": self._project_id(),
                    "entityId": entity["id"],
                    "values": _to_record_inputs({**values, primary_key: str(pk)}),
                    "updatedColumns": list(values.keys()),
                }
            )
        )

    def delete(self, pk: Any) -> None:
        """Delete one record identified by primary key."""
        entity = self.entity_meta()
        primary_key = entity.get("primaryKey")
        if not primary_key:
            raise ValidationError(
                f'entity "{self._api_name}" has no primary key — deletes are not supported'
            )
        warn_if_flagged("OntologyDeleteRow")
        self._execute(
            ops.ontology_delete_row_request(
                {
                    "projectId": self._project_id(),
                    "entityId": entity["id"],
                    "values": _to_record_inputs({primary_key: str(pk)}),
                }
            )
        )
