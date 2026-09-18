"""The whodb.dataset("weekly_kpis") facade, addressed by dataset name."""

from __future__ import annotations

from collections.abc import Callable
from typing import Any

from ._errors import NotFoundError
from ._generated import operations as ops
from ._hydrate import hydrate_rows
from ._manifest_check import warn_if_flagged


class DatasetHandle:
    """Handle for one dataset."""

    def __init__(self, execute: Callable[[ops.Request], Any], project_id: Callable[[], str], name: str):
        self._execute = execute
        self._project_id = project_id
        self._name = name
        self._dataset_cache: dict | None = None

    def meta(self) -> dict:
        """Resolve and cache the dataset metadata backing this handle."""
        if self._dataset_cache is not None:
            return self._dataset_cache
        warn_if_flagged("ProjectDatasets")
        datasets = self._execute(
            ops.project_datasets_request({"projectId": self._project_id()})
        )
        dataset = next((d for d in datasets or [] if d.get("name") == self._name), None)
        if dataset is None:
            raise NotFoundError(f'dataset "{self._name}" not found in this project')
        self._dataset_cache = dataset
        return dataset

    def rows(self, page_size: int = 100, page_offset: int = 0) -> tuple[list[dict], Any]:
        """Query dataset rows with paging; returns (rows, total_count)."""
        dataset = self.meta()
        warn_if_flagged("QueryDataset")
        result = self._execute(
            ops.query_dataset_request(
                {
                    "input": {
                        "projectId": self._project_id(),
                        "datasetId": dataset["id"],
                        "pageSize": page_size,
                        "pageOffset": page_offset,
                    }
                }
            )
        )
        return hydrate_rows(result)
