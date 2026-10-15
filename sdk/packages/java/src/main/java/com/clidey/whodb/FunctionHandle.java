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

/** The {@code client.function("normalize")} facade, addressed by function name or id. */
public final class FunctionHandle {
    private final WhoDB client;
    private final String name;
    private Map<String, Object> function;

    /**
     * Options for a one-off test execution of the draft function.
     *
     * @param files overrides the draft files as [{path, content}]
     * @param inputFileIds mounts project files as inputs
     */
    public record TestOptions(List<Map<String, Object>> files, List<String> inputFileIds) {
        /** No overrides. */
        public static TestOptions defaults() {
            return new TestOptions(null, null);
        }
    }

    FunctionHandle(WhoDB client, String name) {
        this.client = client;
        this.name = name;
    }

    /** Resolves and caches the function backing this handle. */
    public synchronized Map<String, Object> meta() {
        if (function != null) {
            return function;
        }
        ManifestCheck.warnIfFlagged("ProjectFunctions");
        List<Map<String, Object>> functions = Facades.asList(
            client.execute(Operations.projectFunctionsRequest(Map.of("projectId", client.projectId()))));
        function = Facades.findByIdOrName(functions, name);
        if (function == null) {
            throw new WhoDBException(WhoDBException.Kind.NOT_FOUND, "function \"" + name + "\" not found in this project");
        }
        return function;
    }

    private Object id() {
        return meta().get("id");
    }

    /** Runs the active version synchronously and returns its result. */
    public Map<String, Object> execute(Object input, List<String> inputFileIds) {
        Object id = id();
        ManifestCheck.warnIfFlagged("ExecuteFunction");
        return Facades.asMap(client.execute(Operations.executeFunctionRequest(Facades.vars(
            "projectId", client.projectId(), "functionId", id,
            "input", Facades.encodeInput(input), "inputFileIds", inputFileIds))));
    }

    /** Runs the draft version once with test input; nothing is persisted. */
    public Map<String, Object> test(Object input, TestOptions options) {
        Object id = id();
        TestOptions effective = options == null ? TestOptions.defaults() : options;
        ManifestCheck.warnIfFlagged("TestFunction");
        return Facades.asMap(client.execute(Operations.testFunctionRequest(Map.of("input", Facades.vars(
            "projectId", client.projectId(), "functionId", id, "input", Facades.encodeInput(input),
            "files", effective.files(), "inputFileIds", effective.inputFileIds())))));
    }

    /**
     * Starts a persisted run and returns it immediately; follow it with
     * {@link #wait}. {@code useActiveVersion} null leaves the platform default.
     */
    public Map<String, Object> startRun(Object input, Boolean useActiveVersion) {
        Object id = id();
        ManifestCheck.warnIfFlagged("StartFunctionRun");
        return Facades.asMap(client.execute(Operations.startFunctionRunRequest(Facades.vars(
            "projectId", client.projectId(), "functionId", id,
            "input", Facades.encodeInput(input), "useActiveVersion", useActiveVersion))));
    }

    /** Reads one persisted run. */
    public Map<String, Object> run(String runId) {
        Facades.require(runId, "runId");
        ManifestCheck.warnIfFlagged("FunctionRun");
        return Facades.asMap(client.execute(Operations.functionRunRequest(Map.of(
            "projectId", client.projectId(), "id", runId))));
    }

    /** Lists persisted runs, newest first (limit 0 = 20). */
    public List<Map<String, Object>> runs(int limit) {
        Object id = id();
        ManifestCheck.warnIfFlagged("FunctionRuns");
        return Facades.asList(client.execute(Operations.functionRunsRequest(Facades.vars(
            "projectId", client.projectId(), "functionId", id, "limit", limit <= 0 ? 20 : limit))));
    }

    /** Polls a run until it reaches a terminal status and returns it. */
    public Map<String, Object> wait(String runId, WaitOptions options) {
        Facades.require(runId, "runId");
        return Facades.pollUntil(() -> run(runId), run -> Facades.isTerminalRunStatus(run.get("status")),
            options, "function run " + runId);
    }

    /** Requests cancellation of a queued or running run. */
    public Map<String, Object> cancelRun(String runId) {
        Facades.require(runId, "runId");
        ManifestCheck.warnIfFlagged("CancelFunctionRun");
        return Facades.asMap(client.execute(Operations.cancelFunctionRunRequest(Map.of(
            "projectId", client.projectId(), "id", runId))));
    }

    /** Deploys the active version. */
    public void deploy() {
        Object id = id();
        ManifestCheck.warnIfFlagged("DeployFunction");
        client.execute(Operations.deployFunctionRequest(Facades.vars("projectId", client.projectId(), "id", id)));
    }

    /** Redeploys the active version. */
    public void redeploy() {
        Object id = id();
        ManifestCheck.warnIfFlagged("RedeployFunction");
        client.execute(Operations.redeployFunctionRequest(Facades.vars("projectId", client.projectId(), "id", id)));
    }

    /** Updates the draft function; send only changed fields. */
    public synchronized Map<String, Object> update(Map<String, Object> changes) {
        Object id = id();
        ManifestCheck.warnIfFlagged("UpdateFunction");
        Map<String, Object> updated = Facades.asMap(client.execute(Operations.updateFunctionRequest(Map.of(
            "input", Facades.mergeInput(changes, "id", id, "projectId", client.projectId())))));
        function = updated;
        return updated;
    }

    /** Soft-deletes the function. */
    public synchronized void delete() {
        Object id = id();
        ManifestCheck.warnIfFlagged("DeleteFunction");
        client.execute(Operations.deleteFunctionRequest(Facades.vars("projectId", client.projectId(), "id", id)));
        function = null;
    }

    /** Promotes the draft function: snapshots it and makes it the active production version. {@code message} may be null. */
    public Map<String, Object> promote(String message) {
        return client.versions().promote("function", String.valueOf(meta().get("id")), message);
    }

    /** Lists the function's promoted versions, newest first. */
    public List<Map<String, Object>> versions() {
        return client.versions().list("function", String.valueOf(meta().get("id")));
    }
}
