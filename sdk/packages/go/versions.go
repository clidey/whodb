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

	"github.com/clidey/whodb/sdk/packages/go/gen"
)

// Versionable object types accepted by the versioning facade.
const (
	ObjectTypeApp       = "app"
	ObjectTypeFunction  = "function"
	ObjectTypeTransform = "transform"
	ObjectTypeDataset   = "dataset"
	ObjectTypeOntology  = "ontology"
)

type restoreOperation struct {
	operation string
	build     func(map[string]any) gen.Request
	idArg     string
}

var restoreOperations = map[string]restoreOperation{
	ObjectTypeOntology:  {"RestoreOntologyVersionToDraft", gen.NewRestoreOntologyVersionToDraftRequest, "ontologyId"},
	ObjectTypeDataset:   {"RestoreDatasetVersionToDraft", gen.NewRestoreDatasetVersionToDraftRequest, "datasetId"},
	ObjectTypeTransform: {"RestoreTransformVersionToDraft", gen.NewRestoreTransformVersionToDraftRequest, "transformId"},
	ObjectTypeFunction:  {"RestoreFunctionVersionToDraft", gen.NewRestoreFunctionVersionToDraftRequest, "functionId"},
	ObjectTypeApp:       {"RestoreAppVersionToDraft", gen.NewRestoreAppVersionToDraftRequest, "appId"},
}

// VersionsHandle is the client.Versions() facade: version history, the active
// production version, promotion, rollback, and restore-to-draft for any
// versionable object (app, function, transform, dataset, ontology).
type VersionsHandle struct {
	client *Client
}

func (h *VersionsHandle) scope(ctx context.Context, objectType, objectID string) (map[string]any, error) {
	if _, ok := restoreOperations[objectType]; !ok {
		return nil, fmt.Errorf("%w: objectType must be one of app, function, transform, dataset, ontology; got %q", ErrValidation, objectType)
	}
	if objectID == "" {
		return nil, fmt.Errorf("%w: objectID is required", ErrValidation)
	}
	projectID, err := h.client.projectID(ctx)
	if err != nil {
		return nil, err
	}
	return map[string]any{"projectId": projectID, "objectId": objectID, "objectType": objectType}, nil
}

// List lists an object's promoted versions, newest first.
func (h *VersionsHandle) List(ctx context.Context, objectType, objectID string) ([]map[string]any, error) {
	scope, err := h.scope(ctx, objectType, objectID)
	if err != nil {
		return nil, err
	}
	warnIfFlagged("ObjectVersions")
	result, err := h.client.execute(ctx, gen.NewObjectVersionsRequest(scope))
	if err != nil {
		return nil, err
	}
	return asList(result), nil
}

// Active returns the active production version, or nil when nothing has been promoted.
func (h *VersionsHandle) Active(ctx context.Context, objectType, objectID string) (map[string]any, error) {
	scope, err := h.scope(ctx, objectType, objectID)
	if err != nil {
		return nil, err
	}
	warnIfFlagged("ActiveProdVersion")
	result, err := h.client.execute(ctx, gen.NewActiveProdVersionRequest(scope))
	if err != nil {
		return nil, err
	}
	return asMap(result), nil
}

// ActiveAll lists every active production version in the project.
func (h *VersionsHandle) ActiveAll(ctx context.Context) ([]map[string]any, error) {
	projectID, err := h.client.projectID(ctx)
	if err != nil {
		return nil, err
	}
	warnIfFlagged("ProjectActiveProdVersions")
	result, err := h.client.execute(ctx, gen.NewProjectActiveProdVersionsRequest(map[string]any{"projectId": projectID}))
	if err != nil {
		return nil, err
	}
	return asList(result), nil
}

// Promote snapshots the object's draft as a new version and makes it the
// active production version in one step. message may be empty.
func (h *VersionsHandle) Promote(ctx context.Context, objectType, objectID, message string) (map[string]any, error) {
	scope, err := h.scope(ctx, objectType, objectID)
	if err != nil {
		return nil, err
	}
	scope["message"] = nil
	if message != "" {
		scope["message"] = message
	}
	warnIfFlagged("PromoteObject")
	result, err := h.client.execute(ctx, gen.NewPromoteObjectRequest(map[string]any{"input": scope}))
	if err != nil {
		return nil, err
	}
	return asMap(result), nil
}

// Rollback makes an earlier promoted version the active production version.
func (h *VersionsHandle) Rollback(ctx context.Context, objectType, objectID string, version int) (map[string]any, error) {
	scope, err := h.scope(ctx, objectType, objectID)
	if err != nil {
		return nil, err
	}
	scope["version"] = version
	warnIfFlagged("RollbackObject")
	result, err := h.client.execute(ctx, gen.NewRollbackObjectRequest(map[string]any{"input": scope}))
	if err != nil {
		return nil, err
	}
	return asMap(result), nil
}

// RestoreToDraft copies a promoted version back into the object's draft and
// returns the restored object.
func (h *VersionsHandle) RestoreToDraft(ctx context.Context, objectType, objectID string, version int) (map[string]any, error) {
	scope, err := h.scope(ctx, objectType, objectID)
	if err != nil {
		return nil, err
	}
	restore := restoreOperations[objectType]
	warnIfFlagged(restore.operation)
	result, err := h.client.execute(ctx, restore.build(map[string]any{"projectId": scope["projectId"], restore.idArg: objectID, "version": version}))
	if err != nil {
		return nil, err
	}
	return asMap(result), nil
}
