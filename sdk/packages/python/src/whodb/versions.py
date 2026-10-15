"""The whodb.versions() facade: version history, promotion, rollback, and
restore-to-draft for any versionable object (app, function, transform, dataset,
ontology). PromoteObject snapshots the draft and makes it the active production
version in one step."""

from __future__ import annotations

from collections.abc import Callable
from typing import Any

from ._errors import ValidationError
from ._generated import operations as ops
from ._manifest_check import warn_if_flagged

OBJECT_TYPES = ("app", "function", "transform", "dataset", "ontology")

_RESTORE = {
    "ontology": ("RestoreOntologyVersionToDraft", ops.restore_ontology_version_to_draft_request, "ontologyId"),
    "dataset": ("RestoreDatasetVersionToDraft", ops.restore_dataset_version_to_draft_request, "datasetId"),
    "transform": ("RestoreTransformVersionToDraft", ops.restore_transform_version_to_draft_request, "transformId"),
    "function": ("RestoreFunctionVersionToDraft", ops.restore_function_version_to_draft_request, "functionId"),
    "app": ("RestoreAppVersionToDraft", ops.restore_app_version_to_draft_request, "appId"),
}


def assert_object_type(object_type: str) -> str:
    """Validate an object type against the platform's versionable set."""
    if object_type not in OBJECT_TYPES:
        raise ValidationError(f"object_type must be one of {', '.join(OBJECT_TYPES)}; got {object_type!r}")
    return object_type


class VersionsHandle:
    """Versioning and promotion for the selected project's objects."""

    def __init__(self, execute: Callable[[ops.Request], Any], project_id: Callable[[], str]):
        self._execute = execute
        self._project_id = project_id

    def _scope(self, object_type: str, object_id: str) -> dict[str, Any]:
        assert_object_type(object_type)
        if not object_id:
            raise ValidationError("object_id is required")
        return {"projectId": self._project_id(), "objectId": object_id, "objectType": object_type}

    def list(self, object_type: str, object_id: str) -> list[dict]:
        """List an object's promoted versions, newest first."""
        scope = self._scope(object_type, object_id)
        warn_if_flagged("ObjectVersions")
        return self._execute(ops.object_versions_request(scope))

    def active(self, object_type: str, object_id: str) -> dict | None:
        """The active production version, or None when nothing has been promoted."""
        scope = self._scope(object_type, object_id)
        warn_if_flagged("ActiveProdVersion")
        return self._execute(ops.active_prod_version_request(scope))

    def active_all(self) -> list[dict]:
        """Every active production version in the project."""
        warn_if_flagged("ProjectActiveProdVersions")
        return self._execute(ops.project_active_prod_versions_request({"projectId": self._project_id()}))

    def promote(self, object_type: str, object_id: str, message: str | None = None) -> dict:
        """Promote the draft: snapshot it as a new version and activate it for production."""
        scope = self._scope(object_type, object_id)
        warn_if_flagged("PromoteObject")
        return self._execute(ops.promote_object_request({"input": {**scope, "message": message}}))

    def rollback(self, object_type: str, object_id: str, version: int) -> dict:
        """Make an earlier promoted version the active production version."""
        scope = self._scope(object_type, object_id)
        warn_if_flagged("RollbackObject")
        return self._execute(ops.rollback_object_request({"input": {**scope, "version": version}}))

    def restore_to_draft(self, object_type: str, object_id: str, version: int) -> dict:
        """Copy a promoted version back into the object's draft; returns the restored object."""
        self._scope(object_type, object_id)
        operation, build, id_arg = _RESTORE[object_type]
        warn_if_flagged(operation)
        return self._execute(build({"projectId": self._project_id(), id_arg: object_id, "version": version}))
