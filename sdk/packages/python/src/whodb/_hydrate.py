"""Row hydration: stringly-typed wire results → native-typed dict rows.

Behavior is pinned by the shared cross-language conformance fixtures — keep
in lockstep with the TypeScript hydrate.ts.
"""

from __future__ import annotations

import json
from datetime import date, datetime
from typing import Any, Optional

from ._generated.hydration import HYDRATION_DEFAULT, HYDRATION_RULES


def coerce_value(raw: Any, column_type: str) -> Any:
    """Coerce one stringly-typed cell into its native type per the shared rules.

    Non-string values pass through unchanged: the IPC transport (functions
    runtime) delivers natively-typed values that need no coercion.
    """
    if raw is None:
        return None
    if not isinstance(raw, str):
        return raw
    kind = HYDRATION_RULES.get(column_type.lower(), HYDRATION_DEFAULT)
    if kind == "int":
        try:
            return int(raw)
        except ValueError:
            return raw
    if kind == "float":
        try:
            return float(raw)
        except ValueError:
            return raw
    if kind == "bool":
        return raw in ("true", "t", "1")
    if kind in ("timestamp", "date"):
        try:
            return datetime.fromisoformat(raw.replace("Z", "+00:00"))
        except ValueError:
            return raw
    if kind == "json":
        try:
            return json.loads(raw)
        except (json.JSONDecodeError, TypeError):
            return raw
    return raw


def _normalize(result: dict) -> tuple[list[dict], list[list], Optional[int]]:
    """Normalize the two wire result shapes to (columns, rows, total).

    DatasetQueryResult: {columns: [str] (names only), rows, total}
    RowsResult (CE-derived): {Columns: [{Name, Type}], Rows, TotalCount}
    DatasetQueryResult carries no column types — coercion for it comes from
    ontology property metadata, falling back to string.
    """
    if isinstance(result.get("columns"), list):
        columns = [{"name": name, "type": ""} for name in result.get("columns") or []]
        return columns, result.get("rows") or [], result.get("total")
    columns = [
        {"name": column.get("Name"), "type": column.get("Type", "")}
        for column in result.get("Columns") or []
    ]
    return columns, result.get("Rows") or [], result.get("TotalCount")


def property_types_of(entity: dict) -> dict[str, str]:
    """Build a property-type map (apiName → dataType) from entity metadata."""
    types: dict[str, str] = {}
    for prop in entity.get("properties") or []:
        if prop.get("apiName") and prop.get("dataType"):
            types[prop["apiName"]] = prop["dataType"]
    return types


def hydrate_rows(
    result: dict, property_types: Optional[dict[str, str]] = None
) -> tuple[list[dict], Optional[int]]:
    """Hydrate a wire result into native-typed row dicts.

    Ontology property metadata, when supplied, overrides the wire column type
    — the ontology's dataType is more precise than the storage column type.
    """
    columns, rows, total = _normalize(result)
    hydrated = []
    for cells in rows:
        row: dict[str, Any] = {}
        for index, column in enumerate(columns):
            column_type = (property_types or {}).get(column["name"], column["type"])
            raw = cells[index] if index < len(cells) else None
            row[column["name"]] = coerce_value(raw, column_type or "")
        hydrated.append(row)
    return hydrated, total
