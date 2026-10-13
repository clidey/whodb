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

"""The whodb.lineage_providers() facade: OpenLineage providers feeding project lineage."""

from __future__ import annotations

from collections.abc import Callable
from typing import Any

from ._errors import ValidationError
from ._generated import operations as ops
from ._manifest_check import warn_if_flagged


class LineageProvidersHandle:
    """External OpenLineage providers for the selected project."""

    def __init__(self, execute: Callable[[ops.Request], Any], project_id: Callable[[], str]):
        self._execute = execute
        self._project_id = project_id

    def list(self) -> list[dict]:
        """List the project's providers; API keys are never returned."""
        warn_if_flagged("ProjectLineageProviders")
        return self._execute(ops.project_lineage_providers_request({"projectId": self._project_id()}))

    def save(self, definition: dict[str, Any]) -> dict:
        """Create a provider, or update the one whose id is given; projectId is injected."""
        warn_if_flagged("SaveLineageProvider")
        return self._execute(
            ops.save_lineage_provider_request({"input": {**definition, "projectId": self._project_id()}})
        )

    def delete(self, provider_id: str) -> None:
        """Delete a provider."""
        if not provider_id:
            raise ValidationError("provider_id is required")
        warn_if_flagged("DeleteLineageProvider")
        self._execute(
            ops.delete_lineage_provider_request({"projectId": self._project_id(), "providerId": provider_id})
        )
