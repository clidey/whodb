"""The whodb.source("src_...") facade: browse and read a connected source."""

from __future__ import annotations

from collections.abc import Callable
from typing import Any

from ._generated import operations as ops
from ._hydrate import hydrate_rows
from ._manifest_check import warn_if_flagged
from ._pagination import ListCall, Page

_DEFAULT_PAGE_SIZE = 100


class SourceHandle:
    """Handle for one connected data source, addressed by ID."""

    def __init__(self, execute: Callable[[ops.Request], Any], project_id: Callable[[], str], source_id: str):
        self._execute = execute
        self._project_id = project_id
        self._source_id = source_id

    def objects(self, parent: dict | None = None, page_size: int | None = None, page_offset: int | None = None) -> list[dict]:
        """List browsable objects (schemas, tables, collections...)."""
        warn_if_flagged("PlatformSourceObjects")
        return self._execute(
            ops.platform_source_objects_request(
                {
                    "projectId": self._project_id(),
                    "sourceId": self._source_id,
                    "parent": parent,
                    "kinds": None,
                    "pageSize": page_size,
                    "pageOffset": page_offset,
                }
            )
        )

    def columns(self, ref: dict) -> list[dict]:
        """List the columns of one object (table/collection)."""
        warn_if_flagged("PlatformSourceColumns")
        return self._execute(
            ops.platform_source_columns_request(
                {"projectId": self._project_id(), "sourceId": self._source_id, "ref": ref}
            )
        )

    def rows(self, ref: dict, where: dict | None = None, sort: list[dict] | None = None, page_size: int = _DEFAULT_PAGE_SIZE) -> ListCall:
        """Read rows from one object; iterate or call .pages()."""

        def fetch_page(page_offset: int) -> Page:
            warn_if_flagged("PlatformSourceRows")
            result = self._execute(
                ops.platform_source_rows_request(
                    {
                        "projectId": self._project_id(),
                        "sourceId": self._source_id,
                        "ref": ref,
                        "where": where,
                        "sort": sort,
                        "pageSize": page_size,
                        "pageOffset": page_offset,
                    }
                )
            )
            rows, total = hydrate_rows(result)
            return Page(rows, total, page_offset)

        return ListCall(fetch_page, page_size)


def list_sources(execute: Callable[[ops.Request], Any], project_id: str) -> list[dict]:
    """List the project's sources."""
    warn_if_flagged("ProjectSources")
    return execute(ops.project_sources_request({"projectId": project_id}))
