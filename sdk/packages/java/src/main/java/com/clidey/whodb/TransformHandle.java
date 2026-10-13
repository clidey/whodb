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

/** The {@code client.transform("nightly-rollup")} facade, addressed by transform name or id. */
public final class TransformHandle {
    private final WhoDB client;
    private final String name;
    private Map<String, Object> transform;

    TransformHandle(WhoDB client, String name) {
        this.client = client;
        this.name = name;
    }

    /** Resolves and caches the transform backing this handle. */
    public synchronized Map<String, Object> meta() {
        if (transform != null) {
            return transform;
        }
        ManifestCheck.warnIfFlagged("ProjectTransforms");
        List<Map<String, Object>> transforms = Facades.asList(
            client.execute(Operations.projectTransformsRequest(Map.of("projectId", client.projectId()))));
        transform = Facades.findByIdOrName(transforms, name);
        if (transform == null) {
            throw new WhoDBException(WhoDBException.Kind.NOT_FOUND, "transform \"" + name + "\" not found in this project");
        }
        return transform;
    }

    private Object id() {
        return meta().get("id");
    }

    /** Lists recent runs, newest first (limit 0 = 20). */
    public List<Map<String, Object>> runs(int limit) {
        Object id = id();
        ManifestCheck.warnIfFlagged("TransformRuns");
        return Facades.asList(client.execute(Operations.transformRunsRequest(Facades.vars(
            "projectId", client.projectId(), "transformId", id, "limit", limit <= 0 ? 20 : limit))));
    }

    /** Starts a run and returns its record immediately; follow it with {@link #wait}. */
    public Map<String, Object> run() {
        Object id = id();
        ManifestCheck.warnIfFlagged("RunTransform");
        return Facades.asMap(client.execute(Operations.runTransformRequest(Facades.vars(
            "projectId", client.projectId(), "id", id))));
    }

    /** Cancels the active run, or a specific run when {@code runId} is non-null. */
    public void cancelRun(String runId) {
        Object id = id();
        ManifestCheck.warnIfFlagged("CancelTransformRun");
        client.execute(Operations.cancelTransformRunRequest(Facades.vars(
            "projectId", client.projectId(), "transformId", id, "runId", Facades.blankToNull(runId))));
    }

    /** Polls until the run reaches a terminal status and returns it. */
    public Map<String, Object> wait(String runId, WaitOptions options) {
        Facades.require(runId, "runId");
        Object id = id();
        String projectId = client.projectId();
        return Facades.pollUntil(() -> {
            ManifestCheck.warnIfFlagged("TransformRuns");
            List<Map<String, Object>> runs = Facades.asList(client.execute(Operations.transformRunsRequest(Facades.vars(
                "projectId", projectId, "transformId", id, "limit", 100))));
            for (Map<String, Object> run : runs) {
                if (runId.equals(run.get("id"))) {
                    return run;
                }
            }
            throw new WhoDBException(WhoDBException.Kind.NOT_FOUND,
                "run \"" + runId + "\" not found for transform \"" + name + "\"");
        }, run -> Facades.isTerminalRunStatus(run.get("status")), options, "transform run " + runId);
    }

    /** Replaces the transform definition. */
    public synchronized Map<String, Object> update(Map<String, Object> definition) {
        Object id = id();
        ManifestCheck.warnIfFlagged("SaveTransform");
        Map<String, Object> saved = Facades.asMap(client.execute(Operations.saveTransformRequest(Map.of(
            "input", Facades.mergeInput(definition, "id", id, "projectId", client.projectId())))));
        transform = saved;
        return saved;
    }

    /** Soft-deletes the transform. */
    public synchronized void delete() {
        Object id = id();
        ManifestCheck.warnIfFlagged("DeleteTransform");
        client.execute(Operations.deleteTransformRequest(Facades.vars("projectId", client.projectId(), "id", id)));
        transform = null;
    }
}
