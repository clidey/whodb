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
import java.util.Set;

/**
 * The {@code client.versions()} facade: version history, the active production
 * version, promotion, rollback, and restore-to-draft for any versionable object
 * (app, function, transform, dataset, ontology). PromoteObject snapshots the
 * draft and makes it the active production version in one step.
 */
public final class VersionsHandle {
    /** Versionable object types accepted by this facade. */
    public static final Set<String> OBJECT_TYPES = Set.of("app", "function", "transform", "dataset", "ontology");

    private final WhoDB client;

    VersionsHandle(WhoDB client) {
        this.client = client;
    }

    private Map<String, Object> scope(String objectType, String objectId) {
        if (objectType == null || !OBJECT_TYPES.contains(objectType)) {
            throw new WhoDBException(WhoDBException.Kind.VALIDATION,
                "objectType must be one of app, function, transform, dataset, ontology; got \"" + objectType + "\"");
        }
        Facades.require(objectId, "objectId");
        return Facades.vars("projectId", client.projectId(), "objectId", objectId, "objectType", objectType);
    }

    /** Lists an object's promoted versions, newest first. */
    public List<Map<String, Object>> list(String objectType, String objectId) {
        Map<String, Object> scope = scope(objectType, objectId);
        ManifestCheck.warnIfFlagged("ObjectVersions");
        return Facades.asList(client.execute(Operations.objectVersionsRequest(scope)));
    }

    /** The active production version, or null when nothing has been promoted. */
    public Map<String, Object> active(String objectType, String objectId) {
        Map<String, Object> scope = scope(objectType, objectId);
        ManifestCheck.warnIfFlagged("ActiveProdVersion");
        return Facades.asMap(client.execute(Operations.activeProdVersionRequest(scope)));
    }

    /** Every active production version in the project. */
    public List<Map<String, Object>> activeAll() {
        ManifestCheck.warnIfFlagged("ProjectActiveProdVersions");
        return Facades.asList(client.execute(Operations.projectActiveProdVersionsRequest(Map.of("projectId", client.projectId()))));
    }

    /** Promotes the draft: snapshots it as a new version and activates it for production. {@code message} may be null. */
    public Map<String, Object> promote(String objectType, String objectId, String message) {
        Map<String, Object> input = scope(objectType, objectId);
        input.put("message", Facades.blankToNull(message));
        ManifestCheck.warnIfFlagged("PromoteObject");
        return Facades.asMap(client.execute(Operations.promoteObjectRequest(Map.of("input", input))));
    }

    /** Makes an earlier promoted version the active production version. */
    public Map<String, Object> rollback(String objectType, String objectId, int version) {
        Map<String, Object> input = scope(objectType, objectId);
        input.put("version", version);
        ManifestCheck.warnIfFlagged("RollbackObject");
        return Facades.asMap(client.execute(Operations.rollbackObjectRequest(Map.of("input", input))));
    }

    /** Copies a promoted version back into the object's draft; returns the restored object. */
    public Map<String, Object> restoreToDraft(String objectType, String objectId, int version) {
        scope(objectType, objectId);
        String projectId = client.projectId();
        Object result = switch (objectType) {
            case "ontology" -> {
                ManifestCheck.warnIfFlagged("RestoreOntologyVersionToDraft");
                yield client.execute(Operations.restoreOntologyVersionToDraftRequest(Map.of("projectId", projectId, "ontologyId", objectId, "version", version)));
            }
            case "dataset" -> {
                ManifestCheck.warnIfFlagged("RestoreDatasetVersionToDraft");
                yield client.execute(Operations.restoreDatasetVersionToDraftRequest(Map.of("projectId", projectId, "datasetId", objectId, "version", version)));
            }
            case "transform" -> {
                ManifestCheck.warnIfFlagged("RestoreTransformVersionToDraft");
                yield client.execute(Operations.restoreTransformVersionToDraftRequest(Map.of("projectId", projectId, "transformId", objectId, "version", version)));
            }
            case "function" -> {
                ManifestCheck.warnIfFlagged("RestoreFunctionVersionToDraft");
                yield client.execute(Operations.restoreFunctionVersionToDraftRequest(Map.of("projectId", projectId, "functionId", objectId, "version", version)));
            }
            default -> {
                ManifestCheck.warnIfFlagged("RestoreAppVersionToDraft");
                yield client.execute(Operations.restoreAppVersionToDraftRequest(Map.of("projectId", projectId, "appId", objectId, "version", version)));
            }
        };
        return Facades.asMap(result);
    }
}
