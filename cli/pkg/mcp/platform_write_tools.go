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

// PlatformDatasetColumnInput describes a dataset column for typed MCP writes.
type PlatformDatasetColumnInput struct {
	Name       string `json:"name" jsonschema:"Column name"`
	Type       string `json:"type" jsonschema:"Column data type"`
	IsNullable bool   `json:"is_nullable,omitempty" jsonschema:"Whether the column may be null"`
	IsPrimary  bool   `json:"is_primary,omitempty" jsonschema:"Whether the column is part of the primary key"`
}

// PlatformCreateDatasetInput is the input for whodb_platform_create_dataset.
type PlatformCreateDatasetInput struct {
	Name        string                       `json:"name" jsonschema:"Dataset name"`
	Description string                       `json:"description,omitempty" jsonschema:"Optional dataset description"`
	SchemaMode  string                       `json:"schema_mode,omitempty" jsonschema:"Dataset schema mode, for example manual"`
	SourceID    string                       `json:"source_id,omitempty" jsonschema:"Optional hosted source id for source-backed datasets"`
	Columns     []PlatformDatasetColumnInput `json:"columns,omitempty" jsonschema:"Manual schema columns"`
}

// PlatformFileColumnMapInput describes one file-to-dataset promotion column.
type PlatformFileColumnMapInput struct {
	SourceColumn  string `json:"source_column" jsonschema:"Column name in the file"`
	DatasetColumn string `json:"dataset_column" jsonschema:"Column name in the dataset"`
	DataType      string `json:"data_type" jsonschema:"Dataset data type"`
	IsNullable    bool   `json:"is_nullable,omitempty" jsonschema:"Whether the dataset column may be null"`
	IsPrimary     bool   `json:"is_primary,omitempty" jsonschema:"Whether the dataset column is part of the primary key"`
}

// PlatformPromoteFileToDatasetInput is the input for whodb_platform_promote_file_to_dataset.
type PlatformPromoteFileToDatasetInput struct {
	FileID      string                       `json:"file_id" jsonschema:"Hosted project file id"`
	Name        string                       `json:"name" jsonschema:"Dataset name"`
	Description string                       `json:"description,omitempty" jsonschema:"Optional dataset description"`
	SheetIndex  *int                         `json:"sheet_index,omitempty" jsonschema:"Optional tabular sheet index"`
	ColumnMap   []PlatformFileColumnMapInput `json:"column_map" jsonschema:"Column mappings, usually from whodb_platform_file_inspect"`
}

// PlatformOntologyRecordInput is the input for ontology record write tools.
type PlatformOntologyRecordInput struct {
	EntityID      string            `json:"entity_id" jsonschema:"Ontology id"`
	Values        map[string]string `json:"values" jsonschema:"Record values keyed by ontology property"`
	UpdateColumns []string          `json:"update_columns,omitempty" jsonschema:"Ontology properties to update; required for update"`
}

// PlatformOntologyFastLookupInput is the input for whodb_platform_create_ontology_fast_lookup.
type PlatformOntologyFastLookupInput struct {
	EntityID string   `json:"entity_id" jsonschema:"Ontology id"`
	Fields   []string `json:"fields" jsonschema:"Ontology properties to include in the lookup"`
	Reason   string   `json:"reason,omitempty" jsonschema:"Optional reason for the lookup"`
}

// PlatformEntityWriteInput is a typed write input with one resource id.
type PlatformEntityWriteInput struct {
	ID string `json:"id" jsonschema:"Resource id"`
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
	case "whodb_platform_create_dataset":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformCreateDatasetInput) (*mcp.CallToolResult, any, error) {
			return handlePlatformCreateDataset(ctx, input, secOpts.ConfirmWrites)
		})
	case "whodb_platform_promote_file_to_dataset":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformPromoteFileToDatasetInput) (*mcp.CallToolResult, any, error) {
			return handlePlatformPromoteFileToDataset(ctx, input, secOpts.ConfirmWrites)
		})
	case "whodb_platform_add_ontology_record":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformOntologyRecordInput) (*mcp.CallToolResult, any, error) {
			return handlePlatformOntologyRecordWrite(ctx, "whodb_platform_add_ontology_record", input, "add_record", secOpts.ConfirmWrites)
		})
	case "whodb_platform_update_ontology_record":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformOntologyRecordInput) (*mcp.CallToolResult, any, error) {
			return handlePlatformOntologyRecordWrite(ctx, "whodb_platform_update_ontology_record", input, "update_record", secOpts.ConfirmWrites)
		})
	case "whodb_platform_delete_ontology_record":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformOntologyRecordInput) (*mcp.CallToolResult, any, error) {
			return handlePlatformOntologyRecordWrite(ctx, "whodb_platform_delete_ontology_record", input, "delete_record", secOpts.ConfirmWrites)
		})
	case "whodb_platform_create_ontology_fast_lookup":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformOntologyFastLookupInput) (*mcp.CallToolResult, any, error) {
			return handlePlatformCreateOntologyFastLookup(ctx, input, secOpts.ConfirmWrites)
		})
	case "whodb_platform_delete_ontology_fast_lookup":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformEntityWriteInput) (*mcp.CallToolResult, any, error) {
			return handlePlatformTypedGenericWrite(ctx, "whodb_platform_delete_ontology_fast_lookup", PlatformGenericWriteInput{Resource: "ontology_fast_lookup", ID: input.ID}, "delete", secOpts.ConfirmWrites)
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
		{Name: "whodb_platform_create_dataset", Description: descPlatformCreateDataset, Annotations: platformDestructiveAnnotations("Create Hosted Dataset")},
		{Name: "whodb_platform_promote_file_to_dataset", Description: descPlatformPromoteFileToDataset, Annotations: platformDestructiveAnnotations("Promote Hosted File To Dataset")},
		{Name: "whodb_platform_add_ontology_record", Description: descPlatformAddOntologyRecord, Annotations: platformDestructiveAnnotations("Add Hosted Ontology Record")},
		{Name: "whodb_platform_update_ontology_record", Description: descPlatformUpdateOntologyRecord, Annotations: platformDestructiveAnnotations("Update Hosted Ontology Record")},
		{Name: "whodb_platform_delete_ontology_record", Description: descPlatformDeleteOntologyRecord, Annotations: platformDestructiveAnnotations("Delete Hosted Ontology Record")},
		{Name: "whodb_platform_create_ontology_fast_lookup", Description: descPlatformCreateOntologyFastLookup, Annotations: platformDestructiveAnnotations("Create Hosted Ontology Fast Lookup")},
		{Name: "whodb_platform_delete_ontology_fast_lookup", Description: descPlatformDeleteOntologyFastLookup, Annotations: platformDestructiveAnnotations("Delete Hosted Ontology Fast Lookup")},
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

func handlePlatformTypedGenericWrite(ctx context.Context, toolName string, input PlatformGenericWriteInput, operationKind string, confirmWrites bool) (*mcp.CallToolResult, PlatformGenericWriteOutput, error) {
	return handlePlatformGenericWrite(ctx, strings.TrimPrefix(toolName, "whodb_"), input, operationKind, confirmWrites)
}

func handlePlatformCreateDataset(ctx context.Context, input PlatformCreateDatasetInput, confirmWrites bool) (*mcp.CallToolResult, PlatformGenericWriteOutput, error) {
	if strings.TrimSpace(input.Name) == "" {
		return typedWriteValidationError("platform_create_dataset", "name is required")
	}
	payload := map[string]any{
		"name": strings.TrimSpace(input.Name),
	}
	if strings.TrimSpace(input.Description) != "" {
		payload["description"] = strings.TrimSpace(input.Description)
	}
	if strings.TrimSpace(input.SchemaMode) != "" {
		payload["schemaMode"] = strings.TrimSpace(input.SchemaMode)
	}
	if strings.TrimSpace(input.SourceID) != "" {
		payload["sourceId"] = strings.TrimSpace(input.SourceID)
	}
	if len(input.Columns) > 0 {
		payload["columns"] = datasetColumnPayload(input.Columns)
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, PlatformGenericWriteOutput{Error: err.Error(), RequestID: generateRequestID("platform_create_dataset")}, nil
	}
	return handlePlatformTypedGenericWrite(ctx, "whodb_platform_create_dataset", PlatformGenericWriteInput{Resource: "dataset", PayloadJSON: string(raw)}, "create", confirmWrites)
}

func handlePlatformPromoteFileToDataset(ctx context.Context, input PlatformPromoteFileToDatasetInput, confirmWrites bool) (*mcp.CallToolResult, PlatformGenericWriteOutput, error) {
	if strings.TrimSpace(input.FileID) == "" {
		return typedWriteValidationError("platform_promote_file_to_dataset", "file_id is required")
	}
	if strings.TrimSpace(input.Name) == "" {
		return typedWriteValidationError("platform_promote_file_to_dataset", "name is required")
	}
	if len(input.ColumnMap) == 0 {
		return typedWriteValidationError("platform_promote_file_to_dataset", "column_map is required")
	}
	payload := map[string]any{
		"datasetName": strings.TrimSpace(input.Name),
		"columnMap":   fileColumnMapPayload(input.ColumnMap),
	}
	if strings.TrimSpace(input.Description) != "" {
		payload["description"] = strings.TrimSpace(input.Description)
	}
	if input.SheetIndex != nil {
		payload["sheetIndex"] = *input.SheetIndex
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, PlatformGenericWriteOutput{Error: err.Error(), RequestID: generateRequestID("platform_promote_file_to_dataset")}, nil
	}
	return handlePlatformTypedGenericWrite(ctx, "whodb_platform_promote_file_to_dataset", PlatformGenericWriteInput{Resource: "file", Action: "promote_to_dataset", ID: input.FileID, PayloadJSON: string(raw)}, "action", confirmWrites)
}

func handlePlatformOntologyRecordWrite(ctx context.Context, toolName string, input PlatformOntologyRecordInput, action string, confirmWrites bool) (*mcp.CallToolResult, PlatformGenericWriteOutput, error) {
	requestName := strings.TrimPrefix(toolName, "whodb_")
	if strings.TrimSpace(input.EntityID) == "" {
		return typedWriteValidationError(requestName, "entity_id is required")
	}
	if len(input.Values) == 0 {
		return typedWriteValidationError(requestName, "values is required")
	}
	if action == "update_record" && len(normalizedPlatformWriteStrings(input.UpdateColumns)) == 0 {
		return typedWriteValidationError(requestName, "update_columns is required")
	}
	payload := map[string]any{"values": recordInputPayload(input.Values)}
	if action == "update_record" {
		payload["updatedColumns"] = normalizedPlatformWriteStrings(input.UpdateColumns)
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, PlatformGenericWriteOutput{Error: err.Error(), RequestID: generateRequestID(requestName)}, nil
	}
	return handlePlatformTypedGenericWrite(ctx, toolName, PlatformGenericWriteInput{Resource: "ontology", Action: action, ID: input.EntityID, PayloadJSON: string(raw)}, "action", confirmWrites)
}

func handlePlatformCreateOntologyFastLookup(ctx context.Context, input PlatformOntologyFastLookupInput, confirmWrites bool) (*mcp.CallToolResult, PlatformGenericWriteOutput, error) {
	if strings.TrimSpace(input.EntityID) == "" {
		return typedWriteValidationError("platform_create_ontology_fast_lookup", "entity_id is required")
	}
	if len(normalizedPlatformWriteStrings(input.Fields)) == 0 {
		return typedWriteValidationError("platform_create_ontology_fast_lookup", "fields is required")
	}
	payload := map[string]any{
		"entityId": strings.TrimSpace(input.EntityID),
		"fields":   normalizedPlatformWriteStrings(input.Fields),
	}
	if strings.TrimSpace(input.Reason) != "" {
		payload["reason"] = strings.TrimSpace(input.Reason)
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, PlatformGenericWriteOutput{Error: err.Error(), RequestID: generateRequestID("platform_create_ontology_fast_lookup")}, nil
	}
	return handlePlatformTypedGenericWrite(ctx, "whodb_platform_create_ontology_fast_lookup", PlatformGenericWriteInput{Resource: "ontology_fast_lookup", PayloadJSON: string(raw)}, "create", confirmWrites)
}

func typedWriteValidationError(toolName, message string) (*mcp.CallToolResult, PlatformGenericWriteOutput, error) {
	return nil, PlatformGenericWriteOutput{Error: message, RequestID: generateRequestID(toolName)}, nil
}

func datasetColumnPayload(columns []PlatformDatasetColumnInput) []map[string]any {
	out := make([]map[string]any, 0, len(columns))
	for _, column := range columns {
		out = append(out, map[string]any{
			"name":       strings.TrimSpace(column.Name),
			"type":       strings.TrimSpace(column.Type),
			"isNullable": column.IsNullable,
			"isPrimary":  column.IsPrimary,
		})
	}
	return out
}

func fileColumnMapPayload(columns []PlatformFileColumnMapInput) []map[string]any {
	out := make([]map[string]any, 0, len(columns))
	for _, column := range columns {
		out = append(out, map[string]any{
			"sourceColumn":  strings.TrimSpace(column.SourceColumn),
			"datasetColumn": strings.TrimSpace(column.DatasetColumn),
			"dataType":      strings.TrimSpace(column.DataType),
			"isNullable":    column.IsNullable,
			"isPrimary":     column.IsPrimary,
		})
	}
	return out
}

func recordInputPayload(values map[string]string) []map[string]any {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		out = append(out, map[string]any{"key": key, "value": values[key]})
	}
	return out
}

func normalizedPlatformWriteStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
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
		if spec.NeedsID {
			switch spec.Mutation {
			case "OntologyAddRow", "OntologyUpdateRow", "OntologyDeleteRow":
				variables["entityId"] = id
			default:
				variables["id"] = id
			}
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

const descPlatformCreateDataset = `Create a hosted dataset through the selected project.

Use this typed wrapper for normal dataset creation instead of raw whodb_platform_create when possible. Default mode returns a confirmation token; do not call whodb_platform_confirm until the user approves the preview.`

const descPlatformPromoteFileToDataset = `Promote a hosted project file to a dataset.

Use whodb_platform_file_inspect first to infer column_map values. Default mode returns a confirmation token; do not call whodb_platform_confirm until the user approves the preview.`

const descPlatformAddOntologyRecord = `Add one row to a hosted ontology backing table.

Values are keyed by ontology property. Default mode returns a confirmation token; do not call whodb_platform_confirm until the user approves the preview.`

const descPlatformUpdateOntologyRecord = `Update one hosted ontology row.

Pass values containing both matcher values and replacement values, plus update_columns naming the properties to update. Default mode returns a confirmation token; do not call whodb_platform_confirm until the user approves the preview.`

const descPlatformDeleteOntologyRecord = `Delete one hosted ontology row.

Pass values that identify the row, usually the primary key property. Default mode returns a confirmation token; do not call whodb_platform_confirm until the user approves the preview.`

const descPlatformCreateOntologyFastLookup = `Create a fast lookup for one hosted ontology.

Pass ontology property names in fields. Default mode returns a confirmation token; do not call whodb_platform_confirm until the user approves the preview.`

const descPlatformDeleteOntologyFastLookup = `Delete one hosted ontology fast lookup by id.

Default mode returns a confirmation token; do not call whodb_platform_confirm until the user approves the preview.`
