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

// TransformsHandle is the client.Transforms() facade: the project's transform
// collection.
type TransformsHandle struct {
	client *Client
}

// List lists the project's transforms.
func (h *TransformsHandle) List(ctx context.Context) ([]map[string]any, error) {
	projectID, err := h.client.projectID(ctx)
	if err != nil {
		return nil, err
	}
	warnIfFlagged("ProjectTransforms")
	result, err := h.client.execute(ctx, gen.NewProjectTransformsRequest(map[string]any{"projectId": projectID}))
	if err != nil {
		return nil, err
	}
	return asList(result), nil
}

// Create creates a transform from a pipeline graph (SaveTransformInput fields
// without id/projectId).
func (h *TransformsHandle) Create(ctx context.Context, definition map[string]any) (map[string]any, error) {
	projectID, err := h.client.projectID(ctx)
	if err != nil {
		return nil, err
	}
	warnIfFlagged("SaveTransform")
	result, err := h.client.execute(ctx, gen.NewSaveTransformRequest(map[string]any{
		"input": mergeInput(definition, map[string]any{"projectId": projectID}),
	}))
	if err != nil {
		return nil, err
	}
	return asMap(result), nil
}

// Preview runs a pipeline graph without saving it and returns the resulting rows.
func (h *TransformsHandle) Preview(ctx context.Context, graphJSON string) (RowsPage, error) {
	projectID, err := h.client.projectID(ctx)
	if err != nil {
		return RowsPage{}, err
	}
	warnIfFlagged("PreviewTransform")
	result, err := h.client.execute(ctx, gen.NewPreviewTransformRequest(map[string]any{"projectId": projectID, "graphJson": graphJSON}))
	if err != nil {
		return RowsPage{}, err
	}
	rows, total := hydrateRows(asMap(result), nil)
	return RowsPage{Rows: rows, TotalCount: total}, nil
}

// TransformHandle is the client.Transform("nightly-rollup") facade, addressed
// by transform name or id.
type TransformHandle struct {
	client *Client
	name   string

	mu        sync.Mutex
	transform map[string]any
}

// Meta resolves and caches the transform backing this handle.
func (h *TransformHandle) Meta(ctx context.Context) (map[string]any, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.transform != nil {
		return h.transform, nil
	}
	projectID, err := h.client.projectID(ctx)
	if err != nil {
		return nil, err
	}
	warnIfFlagged("ProjectTransforms")
	result, err := h.client.execute(ctx, gen.NewProjectTransformsRequest(map[string]any{"projectId": projectID}))
	if err != nil {
		return nil, err
	}
	transform := findByIDOrName(asList(result), h.name)
	if transform == nil {
		return nil, fmt.Errorf("%w: transform %q not found in this project", ErrNotFound, h.name)
	}
	h.transform = transform
	return transform, nil
}

// Runs lists recent runs, newest first.
func (h *TransformHandle) Runs(ctx context.Context, limit int) ([]map[string]any, error) {
	transform, err := h.Meta(ctx)
	if err != nil {
		return nil, err
	}
	projectID, _ := h.client.projectID(ctx)
	if limit <= 0 {
		limit = 20
	}
	warnIfFlagged("TransformRuns")
	result, err := h.client.execute(ctx, gen.NewTransformRunsRequest(map[string]any{"projectId": projectID, "transformId": transform["id"], "limit": limit}))
	if err != nil {
		return nil, err
	}
	return asList(result), nil
}

// Run starts a run and returns its record immediately; follow it with Wait.
func (h *TransformHandle) Run(ctx context.Context) (map[string]any, error) {
	transform, err := h.Meta(ctx)
	if err != nil {
		return nil, err
	}
	projectID, _ := h.client.projectID(ctx)
	warnIfFlagged("RunTransform")
	result, err := h.client.execute(ctx, gen.NewRunTransformRequest(map[string]any{"projectId": projectID, "id": transform["id"]}))
	if err != nil {
		return nil, err
	}
	return asMap(result), nil
}

// CancelRun cancels the active run, or a specific run when runID is given.
func (h *TransformHandle) CancelRun(ctx context.Context, runID string) error {
	transform, err := h.Meta(ctx)
	if err != nil {
		return err
	}
	projectID, _ := h.client.projectID(ctx)
	variables := map[string]any{"projectId": projectID, "transformId": transform["id"], "runId": nil}
	if runID != "" {
		variables["runId"] = runID
	}
	warnIfFlagged("CancelTransformRun")
	_, err = h.client.execute(ctx, gen.NewCancelTransformRunRequest(variables))
	return err
}

// Wait polls until the run reaches a terminal status and returns it.
func (h *TransformHandle) Wait(ctx context.Context, runID string, options WaitOptions) (map[string]any, error) {
	if runID == "" {
		return nil, fmt.Errorf("%w: runID is required", ErrValidation)
	}
	transform, err := h.Meta(ctx)
	if err != nil {
		return nil, err
	}
	projectID, _ := h.client.projectID(ctx)
	read := func() (map[string]any, error) {
		warnIfFlagged("TransformRuns")
		result, err := h.client.execute(ctx, gen.NewTransformRunsRequest(map[string]any{"projectId": projectID, "transformId": transform["id"], "limit": 100}))
		if err != nil {
			return nil, err
		}
		for _, run := range asList(result) {
			if run["id"] == runID {
				return run, nil
			}
		}
		return nil, fmt.Errorf("%w: run %q not found for transform %q", ErrNotFound, runID, h.name)
	}
	return pollUntil(ctx, read, func(run map[string]any) bool { return isTerminalRunStatus(run["status"]) }, options, "transform run "+runID)
}

// Update replaces the transform definition.
func (h *TransformHandle) Update(ctx context.Context, definition map[string]any) (map[string]any, error) {
	transform, err := h.Meta(ctx)
	if err != nil {
		return nil, err
	}
	projectID, _ := h.client.projectID(ctx)
	warnIfFlagged("SaveTransform")
	result, err := h.client.execute(ctx, gen.NewSaveTransformRequest(map[string]any{
		"input": mergeInput(definition, map[string]any{"id": transform["id"], "projectId": projectID}),
	}))
	if err != nil {
		return nil, err
	}
	saved := asMap(result)
	h.mu.Lock()
	h.transform = saved
	h.mu.Unlock()
	return saved, nil
}

// Delete soft-deletes the transform.
func (h *TransformHandle) Delete(ctx context.Context) error {
	transform, err := h.Meta(ctx)
	if err != nil {
		return err
	}
	projectID, _ := h.client.projectID(ctx)
	warnIfFlagged("DeleteTransform")
	if _, err := h.client.execute(ctx, gen.NewDeleteTransformRequest(map[string]any{"projectId": projectID, "id": transform["id"]})); err != nil {
		return err
	}
	h.mu.Lock()
	h.transform = nil
	h.mu.Unlock()
	return nil
}

// Promote snapshots the draft transform and makes it the active production version.
func (h *TransformHandle) Promote(ctx context.Context, message string) (map[string]any, error) {
	obj, err := h.Meta(ctx)
	if err != nil {
		return nil, err
	}
	return h.client.Versions().Promote(ctx, ObjectTypeTransform, fmt.Sprint(obj["id"]), message)
}

// Versions lists the transform's promoted versions, newest first.
func (h *TransformHandle) Versions(ctx context.Context) ([]map[string]any, error) {
	obj, err := h.Meta(ctx)
	if err != nil {
		return nil, err
	}
	return h.client.Versions().List(ctx, ObjectTypeTransform, fmt.Sprint(obj["id"]))
}
