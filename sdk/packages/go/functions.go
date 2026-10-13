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

package whodb

import (
	"context"
	"fmt"
	"sync"

	"github.com/clidey/whodb/sdk/packages/go/gen"
)

// FunctionTestOptions configure a one-off test execution of the draft function.
type FunctionTestOptions struct {
	// Files override the draft files as [{path, content}].
	Files []map[string]any
	// InputFileIDs mounts project files as inputs.
	InputFileIDs []string
}

// FunctionsHandle is the client.Functions() facade: the project's function
// collection.
type FunctionsHandle struct {
	client *Client
}

// List lists the project's functions.
func (h *FunctionsHandle) List(ctx context.Context) ([]map[string]any, error) {
	projectID, err := h.client.projectID(ctx)
	if err != nil {
		return nil, err
	}
	warnIfFlagged("ProjectFunctions")
	result, err := h.client.execute(ctx, gen.NewProjectFunctionsRequest(map[string]any{"projectId": projectID}))
	if err != nil {
		return nil, err
	}
	return asList(result), nil
}

// Create creates a function (CreateFunctionInput fields without projectId).
func (h *FunctionsHandle) Create(ctx context.Context, definition map[string]any) (map[string]any, error) {
	projectID, err := h.client.projectID(ctx)
	if err != nil {
		return nil, err
	}
	warnIfFlagged("CreateFunction")
	result, err := h.client.execute(ctx, gen.NewCreateFunctionRequest(map[string]any{
		"input": mergeInput(definition, map[string]any{"projectId": projectID}),
	}))
	if err != nil {
		return nil, err
	}
	return asMap(result), nil
}

// FunctionHandle is the client.Function("normalize") facade, addressed by
// function name or id.
type FunctionHandle struct {
	client *Client
	name   string

	mu       sync.Mutex
	function map[string]any
}

// Meta resolves and caches the function backing this handle.
func (h *FunctionHandle) Meta(ctx context.Context) (map[string]any, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.function != nil {
		return h.function, nil
	}
	projectID, err := h.client.projectID(ctx)
	if err != nil {
		return nil, err
	}
	warnIfFlagged("ProjectFunctions")
	result, err := h.client.execute(ctx, gen.NewProjectFunctionsRequest(map[string]any{"projectId": projectID}))
	if err != nil {
		return nil, err
	}
	fn := findByIDOrName(asList(result), h.name)
	if fn == nil {
		return nil, fmt.Errorf("%w: function %q not found in this project", ErrNotFound, h.name)
	}
	h.function = fn
	return fn, nil
}

// Execute runs the active version synchronously and returns its result.
func (h *FunctionHandle) Execute(ctx context.Context, input any, inputFileIDs []string) (map[string]any, error) {
	fn, err := h.Meta(ctx)
	if err != nil {
		return nil, err
	}
	encoded, err := encodeInput(input)
	if err != nil {
		return nil, err
	}
	projectID, _ := h.client.projectID(ctx)
	variables := map[string]any{"projectId": projectID, "functionId": fn["id"], "input": encoded, "inputFileIds": nil}
	if inputFileIDs != nil {
		variables["inputFileIds"] = inputFileIDs
	}
	warnIfFlagged("ExecuteFunction")
	result, err := h.client.execute(ctx, gen.NewExecuteFunctionRequest(variables))
	if err != nil {
		return nil, err
	}
	return asMap(result), nil
}

// Test runs the draft version once with test input; nothing is persisted.
func (h *FunctionHandle) Test(ctx context.Context, input any, options FunctionTestOptions) (map[string]any, error) {
	fn, err := h.Meta(ctx)
	if err != nil {
		return nil, err
	}
	encoded, err := encodeInput(input)
	if err != nil {
		return nil, err
	}
	projectID, _ := h.client.projectID(ctx)
	testInput := map[string]any{"projectId": projectID, "functionId": fn["id"], "input": encoded, "files": nil, "inputFileIds": nil}
	if options.Files != nil {
		testInput["files"] = options.Files
	}
	if options.InputFileIDs != nil {
		testInput["inputFileIds"] = options.InputFileIDs
	}
	warnIfFlagged("TestFunction")
	result, err := h.client.execute(ctx, gen.NewTestFunctionRequest(map[string]any{"input": testInput}))
	if err != nil {
		return nil, err
	}
	return asMap(result), nil
}

// StartRun starts a persisted run and returns it immediately; follow it with
// Wait. useActiveVersion nil leaves the platform default.
func (h *FunctionHandle) StartRun(ctx context.Context, input any, useActiveVersion *bool) (map[string]any, error) {
	fn, err := h.Meta(ctx)
	if err != nil {
		return nil, err
	}
	encoded, err := encodeInput(input)
	if err != nil {
		return nil, err
	}
	projectID, _ := h.client.projectID(ctx)
	variables := map[string]any{"projectId": projectID, "functionId": fn["id"], "input": encoded, "useActiveVersion": nil}
	if useActiveVersion != nil {
		variables["useActiveVersion"] = *useActiveVersion
	}
	warnIfFlagged("StartFunctionRun")
	result, err := h.client.execute(ctx, gen.NewStartFunctionRunRequest(variables))
	if err != nil {
		return nil, err
	}
	return asMap(result), nil
}

// Run reads one persisted run.
func (h *FunctionHandle) Run(ctx context.Context, runID string) (map[string]any, error) {
	if runID == "" {
		return nil, fmt.Errorf("%w: runID is required", ErrValidation)
	}
	projectID, err := h.client.projectID(ctx)
	if err != nil {
		return nil, err
	}
	warnIfFlagged("FunctionRun")
	result, err := h.client.execute(ctx, gen.NewFunctionRunRequest(map[string]any{"projectId": projectID, "id": runID}))
	if err != nil {
		return nil, err
	}
	return asMap(result), nil
}

// Runs lists persisted runs, newest first.
func (h *FunctionHandle) Runs(ctx context.Context, limit int) ([]map[string]any, error) {
	fn, err := h.Meta(ctx)
	if err != nil {
		return nil, err
	}
	projectID, _ := h.client.projectID(ctx)
	if limit <= 0 {
		limit = 20
	}
	warnIfFlagged("FunctionRuns")
	result, err := h.client.execute(ctx, gen.NewFunctionRunsRequest(map[string]any{"projectId": projectID, "functionId": fn["id"], "limit": limit}))
	if err != nil {
		return nil, err
	}
	return asList(result), nil
}

// Wait polls a run until it reaches a terminal status and returns it.
func (h *FunctionHandle) Wait(ctx context.Context, runID string, options WaitOptions) (map[string]any, error) {
	if runID == "" {
		return nil, fmt.Errorf("%w: runID is required", ErrValidation)
	}
	return pollUntil(ctx, func() (map[string]any, error) { return h.Run(ctx, runID) }, func(run map[string]any) bool { return isTerminalRunStatus(run["status"]) }, options, "function run "+runID)
}

// CancelRun requests cancellation of a queued or running run.
func (h *FunctionHandle) CancelRun(ctx context.Context, runID string) (map[string]any, error) {
	if runID == "" {
		return nil, fmt.Errorf("%w: runID is required", ErrValidation)
	}
	projectID, err := h.client.projectID(ctx)
	if err != nil {
		return nil, err
	}
	warnIfFlagged("CancelFunctionRun")
	result, err := h.client.execute(ctx, gen.NewCancelFunctionRunRequest(map[string]any{"projectId": projectID, "id": runID}))
	if err != nil {
		return nil, err
	}
	return asMap(result), nil
}

// Deploy deploys the active version.
func (h *FunctionHandle) Deploy(ctx context.Context) error {
	return h.deployWith(ctx, "DeployFunction", gen.NewDeployFunctionRequest)
}

// Redeploy redeploys the active version.
func (h *FunctionHandle) Redeploy(ctx context.Context) error {
	return h.deployWith(ctx, "RedeployFunction", gen.NewRedeployFunctionRequest)
}

func (h *FunctionHandle) deployWith(ctx context.Context, operation string, build func(map[string]any) gen.Request) error {
	fn, err := h.Meta(ctx)
	if err != nil {
		return err
	}
	projectID, _ := h.client.projectID(ctx)
	warnIfFlagged(operation)
	_, err = h.client.execute(ctx, build(map[string]any{"projectId": projectID, "id": fn["id"]}))
	return err
}

// Update updates the draft function; send only changed fields.
func (h *FunctionHandle) Update(ctx context.Context, changes map[string]any) (map[string]any, error) {
	fn, err := h.Meta(ctx)
	if err != nil {
		return nil, err
	}
	projectID, _ := h.client.projectID(ctx)
	warnIfFlagged("UpdateFunction")
	result, err := h.client.execute(ctx, gen.NewUpdateFunctionRequest(map[string]any{
		"input": mergeInput(changes, map[string]any{"id": fn["id"], "projectId": projectID}),
	}))
	if err != nil {
		return nil, err
	}
	updated := asMap(result)
	h.mu.Lock()
	h.function = updated
	h.mu.Unlock()
	return updated, nil
}

// Delete soft-deletes the function.
func (h *FunctionHandle) Delete(ctx context.Context) error {
	fn, err := h.Meta(ctx)
	if err != nil {
		return err
	}
	projectID, _ := h.client.projectID(ctx)
	warnIfFlagged("DeleteFunction")
	if _, err := h.client.execute(ctx, gen.NewDeleteFunctionRequest(map[string]any{"projectId": projectID, "id": fn["id"]})); err != nil {
		return err
	}
	h.mu.Lock()
	h.function = nil
	h.mu.Unlock()
	return nil
}

// Promote snapshots the draft function and makes it the active production version.
func (h *FunctionHandle) Promote(ctx context.Context, message string) (map[string]any, error) {
	obj, err := h.Meta(ctx)
	if err != nil {
		return nil, err
	}
	return h.client.Versions().Promote(ctx, ObjectTypeFunction, fmt.Sprint(obj["id"]), message)
}

// Versions lists the function's promoted versions, newest first.
func (h *FunctionHandle) Versions(ctx context.Context) ([]map[string]any, error) {
	obj, err := h.Meta(ctx)
	if err != nil {
		return nil, err
	}
	return h.client.Versions().List(ctx, ObjectTypeFunction, fmt.Sprint(obj["id"]))
}
