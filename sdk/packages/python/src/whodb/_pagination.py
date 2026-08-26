"""Pagination: list-shaped facade calls return ListCall objects."""

from __future__ import annotations

from collections.abc import Callable, Iterator
from typing import Any


class Page:
    """One page of hydrated rows."""

    def __init__(self, rows: list[dict], total_count: Any, page_offset: int):
        self.rows = rows
        self.total_count = total_count
        self.page_offset = page_offset


class ListCall:
    """Lazy list result: iterate rows, call .all() for the first page, or
    .pages() to walk every page (a short page terminates iteration)."""

    def __init__(self, fetch_page: Callable[[int], Page], page_size: int):
        self._fetch_page = fetch_page
        self._page_size = page_size

    def all(self) -> list[dict]:
        """Return the first page's rows (mirrors awaiting the TS ListCall)."""
        return self._fetch_page(0).rows

    def pages(self) -> Iterator[Page]:
        """Iterate pages until a short page signals the end."""
        offset = 0
        while True:
            page = self._fetch_page(offset)
            yield page
            if len(page.rows) < self._page_size:
                return
            offset += self._page_size

    def __iter__(self) -> Iterator[dict]:
        for page in self.pages():
            yield from page.rows
