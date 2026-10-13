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

"""The whodb.sharing() facade: cross-project ontology sharing and references."""

from __future__ import annotations

from collections.abc import Callable
from typing import Any

from ._errors import ValidationError
from ._generated import operations as ops
from ._manifest_check import warn_if_flagged


class SharingHandle:
    """Share ontologies with other projects and reference shared ontologies locally."""

    def __init__(self, execute: Callable[[ops.Request], Any], project_id: Callable[[], str]):
        self._execute = execute
        self._project_id = project_id

    def shared(self) -> list[dict]:
        """Ontologies this project shares, with their consumers."""
        warn_if_flagged("SharedOntologies")
        return self._execute(ops.shared_ontologies_request({"projectId": self._project_id()}))

    def discover(self) -> list[dict]:
        """Ontologies other projects share that this project may reference."""
        warn_if_flagged("DiscoverSharedOntologies")
        return self._execute(ops.discover_shared_ontologies_request({"consumerProjectId": self._project_id()}))

    def references(self) -> list[dict]:
        """Shared ontologies this project references."""
        warn_if_flagged("OntologyReferences")
        return self._execute(ops.ontology_references_request({"projectId": self._project_id()}))

    def reference_impact(self, reference_id: str) -> dict:
        """Count dependents of a reference before removing it."""
        if not reference_id:
            raise ValidationError("reference_id is required")
        warn_if_flagged("OntologyReferenceImpact")
        return self._execute(ops.ontology_reference_impact_request({"referenceId": reference_id}))

    def share(self, ontology_id: str, visibility: str, allowed_project_ids: list[str] | None = None) -> dict:
        """Share one of this project's ontologies (visibility: organization or allowlist)."""
        if not ontology_id:
            raise ValidationError("ontology_id is required")
        warn_if_flagged("ShareOntology")
        return self._execute(
            ops.share_ontology_request(
                {
                    "input": {
                        "projectId": self._project_id(),
                        "ontologyId": ontology_id,
                        "visibility": visibility,
                        "allowedProjectIds": allowed_project_ids,
                    }
                }
            )
        )

    def unshare(self, ontology_id: str) -> None:
        """Stop sharing an ontology; existing references become broken."""
        if not ontology_id:
            raise ValidationError("ontology_id is required")
        warn_if_flagged("UnshareOntology")
        self._execute(ops.unshare_ontology_request({"projectId": self._project_id(), "ontologyId": ontology_id}))

    def create_reference(self, owner_project_id: str, ontology_id: str, alias: str) -> dict:
        """Reference a shared ontology from this project under a local alias."""
        if not owner_project_id or not ontology_id or not alias:
            raise ValidationError("owner_project_id, ontology_id and alias are required")
        warn_if_flagged("CreateOntologyReference")
        return self._execute(
            ops.create_ontology_reference_request(
                {
                    "input": {
                        "consumerProjectId": self._project_id(),
                        "ownerProjectId": owner_project_id,
                        "ontologyId": ontology_id,
                        "alias": alias,
                    }
                }
            )
        )

    def remove_reference(self, reference_id: str) -> None:
        """Remove a reference. Check reference_impact() first."""
        if not reference_id:
            raise ValidationError("reference_id is required")
        warn_if_flagged("RemoveOntologyReference")
        self._execute(ops.remove_ontology_reference_request({"referenceId": reference_id}))
