/*
 * Copyright 2026 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package com.clidey.whodb;

import com.clidey.whodb.gen.Operations;
import java.util.List;
import java.util.Map;

/**
 * The {@code client.serviceAccounts()} facade: API access principals for the
 * organization the client is scoped to.
 */
public final class ServiceAccountsHandle {
    private final WhoDB client;

    ServiceAccountsHandle(WhoDB client) {
        this.client = client;
    }

    /** Lists service accounts with key metadata; raw keys are never returned. */
    public List<Map<String, Object>> list() {
        ManifestCheck.warnIfFlagged("ServiceAccounts");
        return Facades.asList(client.execute(Operations.serviceAccountsRequest(Map.of("orgId", client.orgId()))));
    }

    /** Creates a service account. Issue a key with {@link #createKey}. */
    public Map<String, Object> create(String name, String description) {
        Facades.require(name, "name");
        ManifestCheck.warnIfFlagged("CreateServiceAccount");
        return Facades.asMap(client.execute(Operations.createServiceAccountRequest(Facades.vars(
            "orgId", client.orgId(), "name", name.trim(), "description", Facades.blankToNull(description)))));
    }

    /**
     * Issues an API key. The returned {@code rawKey} is shown exactly once;
     * store it securely and never log it. {@code expiresAt} is RFC 3339 or null.
     */
    public Map<String, Object> createKey(String serviceAccountId, String expiresAt) {
        Facades.require(serviceAccountId, "serviceAccountId");
        ManifestCheck.warnIfFlagged("CreateServiceAccountKey");
        return Facades.asMap(client.execute(Operations.createServiceAccountKeyRequest(Facades.vars(
            "orgId", client.orgId(), "serviceAccountId", serviceAccountId, "expiresAt", Facades.blankToNull(expiresAt)))));
    }

    /** Revokes one key. */
    public void revokeKey(String keyId) {
        Facades.require(keyId, "keyId");
        ManifestCheck.warnIfFlagged("RevokeServiceAccountKey");
        client.execute(Operations.revokeServiceAccountKeyRequest(Map.of("orgId", client.orgId(), "keyId", keyId)));
    }

    /** Deletes a service account and revokes all of its keys. */
    public void delete(String serviceAccountId) {
        Facades.require(serviceAccountId, "serviceAccountId");
        ManifestCheck.warnIfFlagged("DeleteServiceAccount");
        client.execute(Operations.deleteServiceAccountRequest(Map.of("orgId", client.orgId(), "id", serviceAccountId)));
    }
}
