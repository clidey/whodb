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

"""The whodb.service_accounts() facade: API access principals for the organization."""

from __future__ import annotations

from collections.abc import Callable
from typing import Any

from ._errors import ValidationError
from ._generated import operations as ops
from ._manifest_check import warn_if_flagged


class ServiceAccountsHandle:
    """Service accounts and API keys for the organization the client is scoped to."""

    def __init__(self, execute: Callable[[ops.Request], Any], org_id: Callable[[], str]):
        self._execute = execute
        self._org_id = org_id

    def list(self) -> list[dict]:
        """List service accounts with key metadata; raw keys are never returned."""
        warn_if_flagged("ServiceAccounts")
        return self._execute(ops.service_accounts_request({"orgId": self._org_id()}))

    def create(self, name: str, description: str | None = None) -> dict:
        """Create a service account. Issue a key with create_key()."""
        if not (name or "").strip():
            raise ValidationError("name is required")
        warn_if_flagged("CreateServiceAccount")
        return self._execute(
            ops.create_service_account_request(
                {"orgId": self._org_id(), "name": name.strip(), "description": description}
            )
        )

    def create_key(self, service_account_id: str, expires_at: str | None = None) -> dict:
        """Issue an API key. ``rawKey`` is returned exactly once; never log it."""
        if not service_account_id:
            raise ValidationError("service_account_id is required")
        warn_if_flagged("CreateServiceAccountKey")
        return self._execute(
            ops.create_service_account_key_request(
                {"orgId": self._org_id(), "serviceAccountId": service_account_id, "expiresAt": expires_at}
            )
        )

    def revoke_key(self, key_id: str) -> None:
        """Revoke one key."""
        if not key_id:
            raise ValidationError("key_id is required")
        warn_if_flagged("RevokeServiceAccountKey")
        self._execute(ops.revoke_service_account_key_request({"orgId": self._org_id(), "keyId": key_id}))

    def delete(self, service_account_id: str) -> None:
        """Delete a service account and revoke all of its keys."""
        if not service_account_id:
            raise ValidationError("service_account_id is required")
        warn_if_flagged("DeleteServiceAccount")
        self._execute(ops.delete_service_account_request({"orgId": self._org_id(), "id": service_account_id}))
