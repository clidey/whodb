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

"""The whodb.transforms() and whodb.transform(name) facades."""

from __future__ import annotations

from collections.abc import Callable
from typing import Any

from ._errors import NotFoundError, ValidationError
from ._generated import operations as ops
from ._hydrate import hydrate_rows
from ._manifest_check import warn_if_flagged
from ._polling import is_terminal_run_status, poll_until


class TransformsHandle:
    """The project's transform collection: list, create, preview."""

    def __init__(self, execute: Callable[[ops.Request], Any], project_id: Callable[[], str]):
        self._execute = execute
        self._project_id = project_id

    def list(self) -> list[dict]:
        """List the project's transforms."""
        warn_if_flagged("ProjectTransforms")
        return self._execute(ops.project_transforms_request({"projectId": self._project_id()}))

    def create(self, definition: dict[str, Any]) -> dict:
        """Create a transform from a pipeline graph; projectId is injected."""
        warn_if_flagged("SaveTransform")
        return self._execute(
            ops.save_transform_request({"input": {**definition, "projectId": self._project_id()}})
        )

    def preview(self, graph_json: str) -> tuple[list[dict], Any]:
        """Run a pipeline graph without saving it; returns (rows, total_count)."""
        warn_if_flagged("PreviewTransform")
        result = self._execute(
            ops.preview_transform_request({"projectId": self._project_id(), "graphJson": graph_json})
        )
        return hydrate_rows(result)


class TransformHandle:
    """Handle for one transform, addressed by name or id."""

    def __init__(self, execute: Callable[[ops.Request], Any], project_id: Callable[[], str], name: str):
        self._execute = execute
        self._project_id = project_id
        self._name = name
        self._transform_cache: dict | None = None

    def meta(self) -> dict:
        """Resolve and cache the transform backing this handle."""
        if self._transform_cache is not None:
            return self._transform_cache
        warn_if_flagged("ProjectTransforms")
        transforms = self._execute(ops.project_transforms_request({"projectId": self._project_id()})) or []
        match = next((t for t in transforms if t.get("id") == self._name or t.get("name") == self._name), None)
        if match is None:
            raise NotFoundError(f'transform "{self._name}" not found in this project')
        self._transform_cache = match
        return match

    def runs(self, limit: int = 20) -> list[dict]:
        """List recent runs, newest first."""
        transform = self.meta()
        warn_if_flagged("TransformRuns")
        return self._execute(
            ops.transform_runs_request(
                {"projectId": self._project_id(), "transformId": transform["id"], "limit": limit}
            )
        )

    def run(self) -> dict:
        """Start a run and return its record; follow it with wait()."""
        transform = self.meta()
        warn_if_flagged("RunTransform")
        return self._execute(ops.run_transform_request({"projectId": self._project_id(), "id": transform["id"]}))

    def cancel_run(self, run_id: str | None = None) -> None:
        """Cancel the active run, or a specific run when run_id is given."""
        transform = self.meta()
        warn_if_flagged("CancelTransformRun")
        self._execute(
            ops.cancel_transform_run_request(
                {"projectId": self._project_id(), "transformId": transform["id"], "runId": run_id}
            )
        )

    def wait(self, run_id: str, timeout_ms: int | None = None, poll_ms: int | None = None) -> dict:
        """Poll until the run reaches a terminal status and return it."""
        if not run_id:
            raise ValidationError("run_id is required")
        transform = self.meta()

        def read() -> dict:
            warn_if_flagged("TransformRuns")
            runs = self._execute(
                ops.transform_runs_request(
                    {"projectId": self._project_id(), "transformId": transform["id"], "limit": 100}
                )
            ) or []
            run = next((r for r in runs if r.get("id") == run_id), None)
            if run is None:
                raise NotFoundError(f'run "{run_id}" not found for transform "{self._name}"')
            return run

        return poll_until(
            read,
            lambda run: is_terminal_run_status(run.get("status")),
            f"transform run {run_id}",
            timeout_ms,
            poll_ms,
        )

    def update(self, definition: dict[str, Any]) -> dict:
        """Replace the transform definition."""
        transform = self.meta()
        warn_if_flagged("SaveTransform")
        saved = self._execute(
            ops.save_transform_request(
                {"input": {**definition, "id": transform["id"], "projectId": self._project_id()}}
            )
        )
        self._transform_cache = saved
        return saved

    def delete(self) -> None:
        """Soft-delete the transform."""
        transform = self.meta()
        warn_if_flagged("DeleteTransform")
        self._execute(ops.delete_transform_request({"projectId": self._project_id(), "id": transform["id"]}))
        self._transform_cache = None
