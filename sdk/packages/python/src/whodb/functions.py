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

"""The whodb.functions() and whodb.function(name) facades."""

from __future__ import annotations

from collections.abc import Callable
from typing import Any

from ._errors import NotFoundError, ValidationError
from ._generated import operations as ops
from ._manifest_check import warn_if_flagged
from ._polling import encode_input, is_terminal_run_status, poll_until


class FunctionsHandle:
    """The project's function collection: list, create."""

    def __init__(self, execute: Callable[[ops.Request], Any], project_id: Callable[[], str]):
        self._execute = execute
        self._project_id = project_id

    def list(self) -> list[dict]:
        """List the project's functions."""
        warn_if_flagged("ProjectFunctions")
        return self._execute(ops.project_functions_request({"projectId": self._project_id()}))

    def create(self, definition: dict[str, Any]) -> dict:
        """Create a function from its files and runtime settings; projectId is injected."""
        warn_if_flagged("CreateFunction")
        return self._execute(
            ops.create_function_request({"input": {**definition, "projectId": self._project_id()}})
        )


class FunctionHandle:
    """Handle for one function, addressed by name or id."""

    def __init__(self, execute: Callable[[ops.Request], Any], project_id: Callable[[], str], name: str):
        self._execute = execute
        self._project_id = project_id
        self._name = name
        self._function_cache: dict | None = None

    def meta(self) -> dict:
        """Resolve and cache the function backing this handle."""
        if self._function_cache is not None:
            return self._function_cache
        warn_if_flagged("ProjectFunctions")
        functions = self._execute(ops.project_functions_request({"projectId": self._project_id()})) or []
        match = next((f for f in functions if f.get("id") == self._name or f.get("name") == self._name), None)
        if match is None:
            raise NotFoundError(f'function "{self._name}" not found in this project')
        self._function_cache = match
        return match

    def execute(self, input: Any = None, input_file_ids: list[str] | None = None) -> dict:  # noqa: A002
        """Execute the active version synchronously and return its result."""
        fn = self.meta()
        warn_if_flagged("ExecuteFunction")
        return self._execute(
            ops.execute_function_request(
                {
                    "projectId": self._project_id(),
                    "functionId": fn["id"],
                    "input": encode_input(input),
                    "inputFileIds": input_file_ids,
                }
            )
        )

    def test(self, input: Any = None, files: list[dict] | None = None, input_file_ids: list[str] | None = None) -> dict:  # noqa: A002
        """Run the draft version once with test input; nothing is persisted."""
        fn = self.meta()
        warn_if_flagged("TestFunction")
        return self._execute(
            ops.test_function_request(
                {
                    "input": {
                        "projectId": self._project_id(),
                        "functionId": fn["id"],
                        "input": encode_input(input),
                        "files": files,
                        "inputFileIds": input_file_ids,
                    }
                }
            )
        )

    def start_run(self, input: Any = None, use_active_version: bool | None = None) -> dict:  # noqa: A002
        """Start a persisted run and return it immediately; follow it with wait()."""
        fn = self.meta()
        warn_if_flagged("StartFunctionRun")
        return self._execute(
            ops.start_function_run_request(
                {
                    "projectId": self._project_id(),
                    "functionId": fn["id"],
                    "input": encode_input(input),
                    "useActiveVersion": use_active_version,
                }
            )
        )

    def run(self, run_id: str) -> dict:
        """Read one persisted run."""
        if not run_id:
            raise ValidationError("run_id is required")
        warn_if_flagged("FunctionRun")
        return self._execute(ops.function_run_request({"projectId": self._project_id(), "id": run_id}))

    def runs(self, limit: int = 20) -> list[dict]:
        """List persisted runs, newest first."""
        fn = self.meta()
        warn_if_flagged("FunctionRuns")
        return self._execute(
            ops.function_runs_request({"projectId": self._project_id(), "functionId": fn["id"], "limit": limit})
        )

    def wait(self, run_id: str, timeout_ms: int | None = None, poll_ms: int | None = None) -> dict:
        """Poll a run until it reaches a terminal status and return it."""
        if not run_id:
            raise ValidationError("run_id is required")
        return poll_until(
            lambda: self.run(run_id),
            lambda run: is_terminal_run_status(run.get("status")),
            f"function run {run_id}",
            timeout_ms,
            poll_ms,
        )

    def cancel_run(self, run_id: str) -> dict:
        """Request cancellation of a queued or running run."""
        if not run_id:
            raise ValidationError("run_id is required")
        warn_if_flagged("CancelFunctionRun")
        return self._execute(ops.cancel_function_run_request({"projectId": self._project_id(), "id": run_id}))

    def deploy(self) -> None:
        """Deploy the active version."""
        fn = self.meta()
        warn_if_flagged("DeployFunction")
        self._execute(ops.deploy_function_request({"projectId": self._project_id(), "id": fn["id"]}))

    def redeploy(self) -> None:
        """Redeploy the active version."""
        fn = self.meta()
        warn_if_flagged("RedeployFunction")
        self._execute(ops.redeploy_function_request({"projectId": self._project_id(), "id": fn["id"]}))

    def update(self, changes: dict[str, Any]) -> dict:
        """Update the draft function; send only changed fields."""
        fn = self.meta()
        warn_if_flagged("UpdateFunction")
        updated = self._execute(
            ops.update_function_request({"input": {**changes, "id": fn["id"], "projectId": self._project_id()}})
        )
        self._function_cache = updated
        return updated

    def delete(self) -> None:
        """Soft-delete the function."""
        fn = self.meta()
        warn_if_flagged("DeleteFunction")
        self._execute(ops.delete_function_request({"projectId": self._project_id(), "id": fn["id"]}))
        self._function_cache = None

    def promote(self, message: str | None = None) -> dict:
        """Promote the draft function: snapshot it and make it the active production version."""
        from .versions import VersionsHandle

        return VersionsHandle(self._execute, self._project_id).promote("function", self.meta()["id"], message)

    def versions(self) -> list[dict]:
        """List the function's promoted versions, newest first."""
        from .versions import VersionsHandle

        return VersionsHandle(self._execute, self._project_id).list("function", self.meta()["id"])
