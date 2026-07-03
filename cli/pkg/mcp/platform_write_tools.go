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

package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	platformapi "github.com/clidey/whodb/cli/internal/platform"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// PlatformGenericWriteInput describes a hosted platform create, update, delete, or action request.
type PlatformGenericWriteInput struct {
	Resource    string `json:"resource" jsonschema:"Resource type, for example secret, ai_provider, ontology, dataset, transform, folder, file, function, source_object"`
	ID          string `json:"id,omitempty" jsonschema:"Resource id for update, delete, or action operations"`
	Action      string `json:"action,omitempty" jsonschema:"Action name for whodb_platform_action, for example deploy, run, rename, move, promote_to_dataset"`
	PayloadJSON string `json:"payload_json,omitempty" jsonschema:"JSON object payload for the hosted mutation. projectId and id are filled from selected workspace/id when appropriate."`
}

// PlatformGenericWriteOutput reports a hosted platform write result or pending confirmation.
type PlatformGenericWriteOutput struct {
	ConfirmationRequired bool                   `json:"confirmation_required,omitempty"`
	ConfirmationToken    string                 `json:"confirmation_token,omitempty"`
	ConfirmationAction   string                 `json:"confirmation_action,omitempty"`
	ConfirmationPreview  *PlatformActionPreview `json:"confirmation_preview,omitempty"`
	ConfirmationExpiry   string                 `json:"confirmation_expiry,omitempty"`
	Warning              string                 `json:"warning,omitempty"`
	Status               string                 `json:"status,omitempty"`
	ResultJSON           string                 `json:"result_json,omitempty"`
	Error                string                 `json:"error,omitempty"`
	RequestID            string                 `json:"request_id,omitempty"`
}

func registerPlatformGenericWriteTool(server *mcp.Server, tool *mcp.Tool, secOpts *SecurityOptions) bool {
	switch tool.Name {
	case "whodb_platform_create":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformGenericWriteInput) (*mcp.CallToolResult, any, error) {
			return handlePlatformGenericWrite(ctx, "platform_create", input, "create", secOpts.ConfirmWrites)
		})
	case "whodb_platform_update":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformGenericWriteInput) (*mcp.CallToolResult, any, error) {
			return handlePlatformGenericWrite(ctx, "platform_update", input, "update", secOpts.ConfirmWrites)
		})
	case "whodb_platform_delete":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformGenericWriteInput) (*mcp.CallToolResult, any, error) {
			return handlePlatformGenericWrite(ctx, "platform_delete", input, "delete", secOpts.ConfirmWrites)
		})
	case "whodb_platform_action":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformGenericWriteInput) (*mcp.CallToolResult, any, error) {
			return handlePlatformGenericWrite(ctx, "platform_action", input, "action", secOpts.ConfirmWrites)
		})
	default:
		return false
	}
	return true
}

func platformGenericWriteToolDefinitions() []*mcp.Tool {
	return []*mcp.Tool{
		{Name: "whodb_platform_create", Description: descPlatformCreate, Annotations: platformDestructiveAnnotations("Create Hosted Platform Resource")},
		{Name: "whodb_platform_update", Description: descPlatformUpdate, Annotations: platformDestructiveAnnotations("Update Hosted Platform Resource")},
		{Name: "whodb_platform_delete", Description: descPlatformDelete, Annotations: platformDestructiveAnnotations("Delete Hosted Platform Resource")},
		{Name: "whodb_platform_action", Description: descPlatformAction, Annotations: platformDestructiveAnnotations("Run Hosted Platform Action")},
	}
}

func handlePlatformGenericWrite(ctx context.Context, toolName string, input PlatformGenericWriteInput, operationKind string, confirmWrites bool) (*mcp.CallToolResult, PlatformGenericWriteOutput, error) {
	requestID := generateRequestID(toolName)
	startTime := time.Now()
	session, err := loadPlatformWorkspace(ctx)
	if err != nil {
		TrackToolCall(ctx, toolName, requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "platform_session"})
		return nil, PlatformGenericWriteOutput{Error: err.Error(), RequestID: requestID}, nil
	}
	spec, payload, err := buildPlatformGenericWrite(session, input, operationKind)
	if err != nil {
		TrackToolCall(ctx, toolName, requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "validation"})
		return nil, PlatformGenericWriteOutput{Error: err.Error(), RequestID: requestID}, nil
	}
	action := &PendingPlatformAction{
		Operation:   spec.Mutation,
		Resource:    spec.Resource,
		Action:      spec.Action,
		Host:        session.Host.URL,
		OrgID:       session.Host.DefaultOrgID,
		ProjectID:   session.Host.DefaultProjectID,
		ProjectName: session.Host.DefaultProjectName,
		Changes:     genericWriteChanges(payload),
		Mutation:    spec.Mutation,
		Variables:   payload,
	}
	if !confirmWrites {
		result, err := executePlatformMutation(ctx, session.Client, spec.Mutation, session.Host.DefaultProjectID, payload)
		if err != nil {
			TrackToolCall(ctx, toolName, requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "platform_action"})
			return nil, PlatformGenericWriteOutput{Error: err.Error(), RequestID: requestID}, nil
		}
		TrackToolCall(ctx, toolName, requestID, true, time.Since(startTime).Milliseconds(), map[string]any{"confirmation_required": false, "mutation": spec.Mutation})
		return nil, platformGenericWriteCompletedOutput(requestID, spec.Mutation, result, action.Preview()), nil
	}
	token, expiresAt := storePendingPlatformAction(action)
	TrackToolCall(ctx, toolName, requestID, true, time.Since(startTime).Milliseconds(), map[string]any{"confirmation_required": true, "mutation": spec.Mutation})
	return nil, platformGenericConfirmationOutput(requestID, token, expiresAt, spec.Mutation, action.Preview()), nil
}

func buildPlatformGenericWrite(session *platformToolSession, input PlatformGenericWriteInput, operationKind string) (platformapi.GenericWriteSpec, map[string]any, error) {
	resource := normalizePlatformWriteToken(input.Resource)
	action := normalizePlatformWriteToken(operationKind)
	if operationKind == "action" {
		action = normalizePlatformWriteToken(input.Action)
	}
	key := action + ":" + resource
	if operationKind == "action" {
		key = "action:" + action + ":" + resource
	}
	spec, ok := platformapi.GenericWriteSpecs[key]
	if !ok {
		return platformapi.GenericWriteSpec{}, nil, fmt.Errorf("unsupported platform %s for resource %q", action, resource)
	}
	payload, err := parsePayloadJSON(input.PayloadJSON)
	if err != nil {
		return platformapi.GenericWriteSpec{}, nil, err
	}
	id := strings.TrimSpace(input.ID)
	if spec.NeedsID && id == "" {
		return platformapi.GenericWriteSpec{}, nil, fmt.Errorf("id is required for %s %s", spec.Action, spec.Resource)
	}
	variables := map[string]any{}
	switch spec.Mode {
	case platformapi.GenericWriteModeInput:
		if spec.InjectProjectID {
			payload["projectId"] = session.Host.DefaultProjectID
		}
		if spec.NeedsID {
			if spec.Mutation == "PromoteFileToDataset" {
				payload["fileId"] = firstPayloadString(payload, "fileId", id)
			} else {
				payload["id"] = id
			}
		}
		if spec.Action == "move" && spec.Resource == "file" {
			payload["newFolderId"] = nullablePayloadString(payload, "newFolderId")
		}
		if spec.Action == "move" && spec.Resource == "folder" {
			payload["newParentId"] = nullablePayloadString(payload, "newParentId")
		}
		variables["input"] = payload
	case platformapi.GenericWriteModeProjectID:
		variables["projectId"] = session.Host.DefaultProjectID
		variables["id"] = id
	case platformapi.GenericWriteModeID:
		variables["id"] = id
	case platformapi.GenericWriteModeProjectIDName:
		name, _ := payload["name"].(string)
		if strings.TrimSpace(name) == "" {
			return platformapi.GenericWriteSpec{}, nil, fmt.Errorf("payload_json.name is required")
		}
		variables["projectId"] = session.Host.DefaultProjectID
		variables["id"] = id
		variables["name"] = strings.TrimSpace(name)
	case platformapi.GenericWriteModeDirect:
		for key, value := range payload {
			variables[key] = value
		}
		if spec.InjectProjectID {
			variables["projectId"] = session.Host.DefaultProjectID
		}
	case platformapi.GenericWriteModeFileUpload:
		filePath, _ := payload["file_path"].(string)
		if strings.TrimSpace(filePath) == "" {
			return platformapi.GenericWriteSpec{}, nil, fmt.Errorf("payload_json.file_path is required")
		}
		variables["filePath"] = strings.TrimSpace(filePath)
		variables["folderId"] = nullablePayloadString(payload, "folderId")
	default:
		return platformapi.GenericWriteSpec{}, nil, fmt.Errorf("unsupported write mode %q", spec.Mode)
	}
	return spec, variables, nil
}

func executePlatformMutation(ctx context.Context, client platformClient, mutation, projectID string, variables map[string]any) (*platformapi.PlatformMutationResult, error) {
	if mutation != "UploadProjectFile" {
		return client.PlatformMutation(ctx, mutation, variables)
	}
	filePath, _ := variables["filePath"].(string)
	var folderID *string
	if value, ok := variables["folderId"].(string); ok && strings.TrimSpace(value) != "" {
		trimmed := strings.TrimSpace(value)
		folderID = &trimmed
	}
	file, err := client.UploadProjectFile(ctx, projectID, folderID, filePath)
	if err != nil {
		return nil, err
	}
	raw, err := json.Marshal(file)
	if err != nil {
		return nil, err
	}
	return &platformapi.PlatformMutationResult{Operation: mutation, Data: raw}, nil
}

func parsePayloadJSON(value string) (map[string]any, error) {
	if strings.TrimSpace(value) == "" {
		return map[string]any{}, nil
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(value), &payload); err != nil {
		return nil, fmt.Errorf("payload_json must be a JSON object: %w", err)
	}
	return payload, nil
}

func nullablePayloadString(payload map[string]any, key string) any {
	value, ok := payload[key]
	if !ok {
		return nil
	}
	text, ok := value.(string)
	if ok && strings.TrimSpace(text) == "" {
		return nil
	}
	return value
}

func firstPayloadString(payload map[string]any, key string, fallback string) string {
	value, ok := payload[key].(string)
	if ok && strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	return fallback
}

func normalizePlatformWriteToken(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func genericWriteChanges(variables map[string]any) []string {
	keys := make([]string, 0)
	if input, ok := variables["input"].(map[string]any); ok {
		for key := range input {
			keys = append(keys, key)
		}
	} else {
		for key := range variables {
			keys = append(keys, key)
		}
	}
	redacted := keys[:0]
	for _, key := range keys {
		if sensitivePlatformWriteKey(key) {
			redacted = append(redacted, key+" (redacted)")
			continue
		}
		redacted = append(redacted, key)
	}
	sort.Strings(redacted)
	return redacted
}

func sensitivePlatformWriteKey(key string) bool {
	key = strings.ToLower(key)
	return strings.Contains(key, "password") || strings.Contains(key, "secret") || strings.Contains(key, "key") || strings.Contains(key, "token") || strings.Contains(key, "value") || strings.Contains(key, "content") || strings.Contains(key, "path")
}

func platformGenericConfirmationOutput(requestID, token string, expiresAt time.Time, mutation string, preview *PlatformActionPreview) PlatformGenericWriteOutput {
	return PlatformGenericWriteOutput{
		ConfirmationRequired: true,
		ConfirmationToken:    token,
		ConfirmationAction:   mutation,
		ConfirmationPreview:  preview,
		ConfirmationExpiry:   expiresAt.UTC().Format(time.RFC3339),
		Warning:              platformConfirmationWarning(preview),
		RequestID:            requestID,
	}
}

func platformGenericWriteCompletedOutput(requestID, mutation string, result *platformapi.PlatformMutationResult, preview *PlatformActionPreview) PlatformGenericWriteOutput {
	output := PlatformGenericWriteOutput{
		Status:              "ok",
		ConfirmationAction:  mutation,
		ConfirmationPreview: preview,
		RequestID:           requestID,
	}
	if result != nil {
		output.ResultJSON = string(result.Data)
	}
	return output
}

const descPlatformCreate = `Create a hosted platform resource through the selected project.

Supported resources: secret, ai_provider, ontology, ontology_fast_lookup, dataset, transform, folder, function, source_object.
Pass payload_json as a JSON object matching the hosted platform mutation input. The selected projectId is injected automatically when applicable.
Default mode returns a confirmation token; do not call whodb_platform_confirm until the user approves the preview. Permission checks are enforced by the hosted platform.`

const descPlatformUpdate = `Update a hosted platform resource through the selected project.

Supported resources: secret, ai_provider, ontology, dataset, transform, function, source_object.
Pass id when required and payload_json with changed fields only. Secret-like payload fields are never shown in confirmation previews.
Default mode returns a confirmation token; do not call whodb_platform_confirm until the user approves the preview. Permission checks are enforced by the hosted platform.`

const descPlatformDelete = `Delete a hosted platform resource through the selected project.

Supported resources: secret, ai_provider, ontology, ontology_fast_lookup, dataset, transform, file, folder, function, source_object.
This is destructive. Pass id for normal resources. For source_object, pass payload_json with sourceId, ref, and values.
Default mode returns a confirmation token; explain exactly what will be deleted and do not call whodb_platform_confirm until the user approves the preview. Permission checks are enforced by the hosted platform.`

const descPlatformAction = `Run a hosted platform resource action through the selected project.

Supported actions include transform/run, file/upload, file/rename, file/move, file/promote_to_dataset, folder/rename, folder/move, function/deploy, and function/redeploy.
Some actions create, move, deploy, or otherwise mutate hosted resources. Pass id and payload_json as required by the action. For file/upload, payload_json requires file_path and may include folderId.
Default mode returns a confirmation token; do not call whodb_platform_confirm until the user approves the preview. Permission checks are enforced by the hosted platform.`
