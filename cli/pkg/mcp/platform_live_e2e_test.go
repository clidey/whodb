//go:build e2e_platform

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
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/clidey/whodb/cli/e2e/testharness"
	"github.com/clidey/whodb/cli/internal/config"
	platformapi "github.com/clidey/whodb/cli/internal/platform"
)

const (
	defaultLivePlatformE2EHost        = "http://localhost:18080"
	defaultLivePlatformE2EKeycloakURL = "http://localhost:14001"
	defaultLivePlatformE2EUser        = "owner@acme.test"
	defaultLivePlatformE2EPassword    = "password"
)

func TestPlatformMCP_RealReadWriteLifecycle(t *testing.T) {
	cleanup := testharness.SetupCleanEnv(t)
	defer cleanup()
	t.Setenv("WHODB_CLI_E2E_PLATFORM_TOKEN_DIR", filepath.Join(os.Getenv("HOME"), ".whodb-cli-platform-e2e-tokens"))
	config.ResetPathsForTesting()
	clearPendingPlatformActions(t)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	host := liveEnvOrDefault("WHODB_PLATFORM_E2E_HOST", defaultLivePlatformE2EHost)
	keycloakURL := liveEnvOrDefault("WHODB_PLATFORM_E2E_KEYCLOAK_URL", defaultLivePlatformE2EKeycloakURL)
	email := liveEnvOrDefault("WHODB_PLATFORM_E2E_USER", defaultLivePlatformE2EUser)
	password := liveEnvOrDefault("WHODB_PLATFORM_E2E_PASSWORD", defaultLivePlatformE2EPassword)
	refreshToken := liveMintDevRefreshToken(t, ctx, keycloakURL, liveEnvOrDefault("WHODB_PLATFORM_E2E_KEYCLOAK_HOST_HEADER", "127.0.0.1:4001"), email, password)
	liveSeedPlatformLogin(t, ctx, host, refreshToken)

	suffix := strings.ReplaceAll(time.Now().UTC().Format("20060102150405.000000000"), ".", "")
	sourceName := "mcp-e2e-source-" + suffix
	sourcePassword := liveEnvOrDefault("WHODB_PLATFORM_E2E_SOURCE_PASSWORD", "whodb")
	sourceHost := liveEnvOrDefault("WHODB_PLATFORM_E2E_SOURCE_HOST", "platform-db")
	sourcePort := liveEnvOrDefault("WHODB_PLATFORM_E2E_SOURCE_PORT", "5432")
	sourceUser := liveEnvOrDefault("WHODB_PLATFORM_E2E_SOURCE_USER", "postgres")
	sourceDatabase := liveEnvOrDefault("WHODB_PLATFORM_E2E_SOURCE_DATABASE", "whodb_platform")

	status := liveMustPlatformStatus(t, ctx)
	if status.Email != email || !status.WorkspaceSelected {
		t.Fatalf("platform status = %#v, want %s with selected workspace", status, email)
	}
	liveMustReadOrgsProjects(t, ctx)
	liveMustReadSourceTypes(t, ctx)

	sourceID := liveMustCreateSource(t, ctx, sourceName, sourceHost, sourcePort, sourceUser, sourcePassword, sourceDatabase)
	defer liveBestEffortSourceDelete(ctx, sourceName)
	liveMustReadSource(t, ctx, sourceName)
	liveMustReadSourceSchema(t, ctx, sourceName)
	liveMustUpdateSource(t, ctx, sourceName, sourceName+"-renamed")
	sourceName = sourceName + "-renamed"

	secretID := liveMustGenericWriteID(t, ctx, "platform_create", "create", PlatformGenericWriteInput{
		Resource:    "secret",
		PayloadJSON: liveJSON(t, map[string]any{"name": "mcp-e2e-secret-" + suffix, "description": "MCP e2e secret", "value": "secret-value"}),
	})
	defer liveBestEffortGenericDelete(ctx, "secret", secretID)
	liveMustReadProjectList(t, ctx, "secrets", func() (int, string) {
		_, out, err := HandlePlatformSecrets(ctx, nil, PlatformEmptyInput{Fields: []string{"id", "name"}})
		if err != nil {
			t.Fatalf("HandlePlatformSecrets() error = %v", err)
		}
		return out.Count, out.Error
	})
	liveMustGenericWrite(t, ctx, "platform_update", "update", PlatformGenericWriteInput{
		Resource:    "secret",
		ID:          secretID,
		PayloadJSON: liveJSON(t, map[string]any{"name": "mcp-e2e-secret-updated-" + suffix, "description": "updated", "value": "rotated-value"}),
	})

	providerID := liveMustGenericWriteID(t, ctx, "platform_create", "create", PlatformGenericWriteInput{
		Resource:    "ai_provider",
		PayloadJSON: liveJSON(t, map[string]any{"name": "mcp-e2e-provider-" + suffix, "providerType": "openai", "endpoint": "https://api.openai.com/v1", "apiKey": "test-key"}),
	})
	defer liveBestEffortGenericDelete(ctx, "ai_provider", providerID)
	liveMustReadProjectList(t, ctx, "ai providers", func() (int, string) {
		_, out, err := HandlePlatformAIProviders(ctx, nil, PlatformEmptyInput{Fields: []string{"id", "name", "providerType"}})
		if err != nil {
			t.Fatalf("HandlePlatformAIProviders() error = %v", err)
		}
		return out.Count, out.Error
	})
	liveMustGenericWrite(t, ctx, "platform_update", "update", PlatformGenericWriteInput{
		Resource:    "ai_provider",
		ID:          providerID,
		PayloadJSON: liveJSON(t, map[string]any{"name": "mcp-e2e-provider-updated-" + suffix, "endpoint": "https://api.openai.com/v1"}),
	})

	datasetID := liveMustGenericWriteID(t, ctx, "platform_create", "create", PlatformGenericWriteInput{
		Resource: "dataset",
		PayloadJSON: liveJSON(t, map[string]any{
			"name":        "mcp-e2e-dataset-" + suffix,
			"description": "MCP e2e dataset",
			"columns": []map[string]any{
				{"name": "id", "type": "text", "isNullable": false, "isPrimary": true},
				{"name": "name", "type": "text", "isNullable": true, "isPrimary": false},
			},
			"schemaMode": "manual",
		}),
	})
	defer liveBestEffortGenericDelete(ctx, "dataset", datasetID)
	liveMustReadEntity(t, ctx, "dataset", datasetID, func() (string, error) {
		_, out, err := HandlePlatformDataset(ctx, nil, PlatformEntityInput{ID: datasetID, Fields: []string{"data", "scope"}})
		return out.Error, err
	})
	liveMustGenericWrite(t, ctx, "platform_update", "update", PlatformGenericWriteInput{
		Resource:    "dataset",
		ID:          datasetID,
		PayloadJSON: liveJSON(t, map[string]any{"name": "mcp-e2e-dataset-updated-" + suffix, "description": "updated"}),
	})

	transformID := liveMustGenericWriteID(t, ctx, "platform_create", "create", PlatformGenericWriteInput{
		Resource: "transform",
		PayloadJSON: liveJSON(t, map[string]any{
			"name":         "mcp-e2e-transform-" + suffix,
			"description":  "MCP e2e transform",
			"graphJson":    `{"nodes":[],"edges":[]}`,
			"scheduleCron": "",
			"triggerMode":  "manual",
		}),
	})
	defer liveBestEffortGenericDelete(ctx, "transform", transformID)
	liveMustReadProjectList(t, ctx, "transforms", func() (int, string) {
		_, out, err := HandlePlatformTransforms(ctx, nil, PlatformEmptyInput{Fields: []string{"id", "name"}})
		if err != nil {
			t.Fatalf("HandlePlatformTransforms() error = %v", err)
		}
		return out.Count, out.Error
	})
	liveMustGenericWrite(t, ctx, "platform_update", "update", PlatformGenericWriteInput{
		Resource:    "transform",
		ID:          transformID,
		PayloadJSON: liveJSON(t, map[string]any{"name": "mcp-e2e-transform-updated-" + suffix, "description": "updated", "graphJson": `{"nodes":[],"edges":[]}`, "scheduleCron": "", "triggerMode": "manual"}),
	})
	liveMustReadEntity(t, ctx, "transform runs", transformID, func() (string, error) {
		_, out, err := HandlePlatformTransformRuns(ctx, nil, PlatformTransformRunsInput{TransformID: transformID, Limit: 5, Fields: []string{"items", "count"}})
		return out.Error, err
	})

	ontologyID := liveMustGenericWriteID(t, ctx, "platform_create", "create", PlatformGenericWriteInput{
		Resource: "ontology",
		PayloadJSON: liveJSON(t, map[string]any{
			"apiName":           "mcp_e2e_entity_" + suffix,
			"displayName":       "MCP E2E Entity " + suffix,
			"pluralDisplayName": "MCP E2E Entities " + suffix,
			"description":       "MCP e2e ontology",
			"primaryKey":        "id",
			"tableName":         "mcp_e2e_entity_" + suffix,
			"schemaName":        "public",
			"icon":              "table",
			"color":             "#3366ff",
			"properties": []map[string]any{{
				"apiName": "id", "displayName": "ID", "description": "ID", "columnName": "id", "dataType": "String",
				"isRequired": true, "visibility": "normal", "isSearchable": true, "isSortable": true, "isEditOnly": false,
			}},
			"links": []map[string]any{},
		}),
	})
	defer liveBestEffortGenericDelete(ctx, "ontology", ontologyID)
	liveMustReadEntity(t, ctx, "ontology", ontologyID, func() (string, error) {
		_, out, err := HandlePlatformOntology(ctx, nil, PlatformEntityInput{ID: ontologyID, Fields: []string{"data", "scope"}})
		return out.Error, err
	})
	lookupID := liveMustGenericWriteID(t, ctx, "platform_create", "create", PlatformGenericWriteInput{
		Resource:    "ontology_fast_lookup",
		PayloadJSON: liveJSON(t, map[string]any{"entityId": ontologyID, "fields": []string{"id"}, "reason": "MCP e2e lookup"}),
	})
	defer liveBestEffortGenericDelete(ctx, "ontology_fast_lookup", lookupID)
	liveMustReadEntity(t, ctx, "ontology fast lookups", ontologyID, func() (string, error) {
		_, out, err := HandlePlatformOntologyFastLookups(ctx, nil, PlatformEntityInput{ID: ontologyID, Fields: []string{"items", "count"}})
		return out.Error, err
	})
	liveMustGenericWrite(t, ctx, "platform_update", "update", PlatformGenericWriteInput{
		Resource: "ontology",
		ID:       ontologyID,
		PayloadJSON: liveJSON(t, map[string]any{
			"displayName":       "MCP E2E Entity Updated " + suffix,
			"pluralDisplayName": "MCP E2E Entities Updated " + suffix,
			"description":       "updated",
			"primaryKey":        "id",
			"tableName":         "mcp_e2e_entity_" + suffix,
			"schemaName":        "public",
			"status":            "active",
			"icon":              "table",
			"color":             "#3366ff",
			"properties": []map[string]any{{
				"apiName": "id", "displayName": "ID", "description": "ID", "columnName": "id", "dataType": "String",
				"isRequired": true, "visibility": "normal", "isSearchable": true, "isSortable": true, "isEditOnly": false,
			}},
			"links": []map[string]any{},
		}),
	})

	functionID := liveMustGenericWriteID(t, ctx, "platform_create", "create", PlatformGenericWriteInput{
		Resource: "function",
		PayloadJSON: liveJSON(t, map[string]any{
			"name":           "mcp-e2e-function-" + suffix,
			"description":    "MCP e2e function",
			"language":       "python",
			"entryPoint":     "main",
			"timeoutSeconds": 30,
			"memory":         "128Mi",
			"cpu":            "100m",
			"files":          []map[string]any{{"path": "main.py", "content": "def main(input):\n    return input\n"}},
			"dependencies":   []map[string]any{},
		}),
	})
	defer liveBestEffortGenericDelete(ctx, "function", functionID)
	liveMustReadEntity(t, ctx, "function", functionID, func() (string, error) {
		_, out, err := HandlePlatformFunction(ctx, nil, PlatformEntityInput{ID: functionID, Fields: []string{"data", "scope"}})
		return out.Error, err
	})
	liveMustGenericWrite(t, ctx, "platform_update", "update", PlatformGenericWriteInput{
		Resource:    "function",
		ID:          functionID,
		PayloadJSON: liveJSON(t, map[string]any{"name": "mcp-e2e-function-updated-" + suffix, "description": "updated", "language": "python", "entryPoint": "main", "files": []map[string]any{{"path": "main.py", "content": "def main(input):\n    return input\n"}}, "dependencies": []map[string]any{}}),
	})

	folderAID := liveMustGenericWriteID(t, ctx, "platform_create", "create", PlatformGenericWriteInput{
		Resource:    "folder",
		PayloadJSON: liveJSON(t, map[string]any{"name": "mcp-e2e-folder-a-" + suffix}),
	})
	defer liveBestEffortGenericDelete(ctx, "folder", folderAID)
	folderBID := liveMustGenericWriteID(t, ctx, "platform_create", "create", PlatformGenericWriteInput{
		Resource:    "folder",
		PayloadJSON: liveJSON(t, map[string]any{"name": "mcp-e2e-folder-b-" + suffix}),
	})
	defer liveBestEffortGenericDelete(ctx, "folder", folderBID)
	csvPath := filepath.Join(t.TempDir(), "mcp-e2e-"+suffix+".csv")
	if err := os.WriteFile(csvPath, []byte("id,name\n1,Ada\n"), 0600); err != nil {
		t.Fatalf("write test csv: %v", err)
	}
	fileID := liveMustGenericWriteID(t, ctx, "platform_action", "action", PlatformGenericWriteInput{
		Resource:    "file",
		Action:      "upload",
		PayloadJSON: liveJSON(t, map[string]any{"file_path": csvPath, "folderId": folderAID}),
	})
	defer liveBestEffortGenericDelete(ctx, "file", fileID)
	liveMustReadFiles(t, ctx, folderAID)
	liveMustReadEntity(t, ctx, "file preview", fileID, func() (string, error) {
		_, out, err := HandlePlatformFilePreview(ctx, nil, PlatformFilePreviewInput{FileID: fileID, Fields: []string{"data", "scope"}})
		return out.Error, err
	})
	liveMustGenericWrite(t, ctx, "platform_action", "action", PlatformGenericWriteInput{
		Resource:    "file",
		Action:      "rename",
		ID:          fileID,
		PayloadJSON: liveJSON(t, map[string]any{"name": "mcp-e2e-renamed-" + suffix + ".csv"}),
	})
	liveMustGenericWrite(t, ctx, "platform_action", "action", PlatformGenericWriteInput{
		Resource:    "file",
		Action:      "move",
		ID:          fileID,
		PayloadJSON: liveJSON(t, map[string]any{"newFolderId": folderBID}),
	})
	liveMustGenericWrite(t, ctx, "platform_action", "action", PlatformGenericWriteInput{
		Resource:    "folder",
		Action:      "rename",
		ID:          folderBID,
		PayloadJSON: liveJSON(t, map[string]any{"name": "mcp-e2e-folder-b-renamed-" + suffix}),
	})
	liveMustGenericWrite(t, ctx, "platform_action", "action", PlatformGenericWriteInput{
		Resource:    "folder",
		Action:      "move",
		ID:          folderAID,
		PayloadJSON: liveJSON(t, map[string]any{"newParentId": folderBID}),
	})
	promotedDatasetID := liveMustGenericWriteID(t, ctx, "platform_action", "action", PlatformGenericWriteInput{
		Resource: "file",
		Action:   "promote_to_dataset",
		ID:       fileID,
		PayloadJSON: liveJSON(t, map[string]any{
			"fileId":      fileID,
			"datasetName": "mcp-e2e-promoted-" + suffix,
			"description": "MCP e2e promoted dataset",
			"sheetIndex":  0,
			"columnMap": []map[string]any{
				{"sourceColumn": "id", "datasetColumn": "id", "dataType": "text", "isNullable": false, "isPrimary": true},
				{"sourceColumn": "name", "datasetColumn": "name", "dataType": "text", "isNullable": true, "isPrimary": false},
			},
		}),
	})
	defer liveBestEffortGenericDelete(ctx, "dataset", promotedDatasetID)
	liveMustReadEntity(t, ctx, "promoted dataset", promotedDatasetID, func() (string, error) {
		_, out, err := HandlePlatformDataset(ctx, nil, PlatformEntityInput{ID: promotedDatasetID, Fields: []string{"data", "scope"}})
		return out.Error, err
	})
	liveMustReadProjectList(t, ctx, "project lineage", func() (int, string) {
		_, out, err := HandlePlatformProjectLineage(ctx, nil, PlatformEmptyInput{Fields: []string{"data", "count"}})
		if err != nil {
			t.Fatalf("HandlePlatformProjectLineage() error = %v", err)
		}
		return out.Count, out.Error
	})

	liveMustGenericWrite(t, ctx, "platform_delete", "delete", PlatformGenericWriteInput{Resource: "file", ID: fileID})
	fileID = ""
	liveMustGenericWrite(t, ctx, "platform_delete", "delete", PlatformGenericWriteInput{Resource: "folder", ID: folderAID})
	folderAID = ""
	liveMustGenericWrite(t, ctx, "platform_delete", "delete", PlatformGenericWriteInput{Resource: "folder", ID: folderBID})
	folderBID = ""
	liveMustGenericWrite(t, ctx, "platform_delete", "delete", PlatformGenericWriteInput{Resource: "ontology_fast_lookup", ID: lookupID})
	lookupID = ""
	liveMustGenericWrite(t, ctx, "platform_delete", "delete", PlatformGenericWriteInput{Resource: "function", ID: functionID})
	functionID = ""
	liveMustGenericWrite(t, ctx, "platform_delete", "delete", PlatformGenericWriteInput{Resource: "transform", ID: transformID})
	transformID = ""
	liveMustGenericWrite(t, ctx, "platform_delete", "delete", PlatformGenericWriteInput{Resource: "dataset", ID: promotedDatasetID})
	promotedDatasetID = ""
	liveMustGenericWrite(t, ctx, "platform_delete", "delete", PlatformGenericWriteInput{Resource: "dataset", ID: datasetID})
	datasetID = ""
	liveMustGenericWrite(t, ctx, "platform_delete", "delete", PlatformGenericWriteInput{Resource: "ontology", ID: ontologyID})
	ontologyID = ""
	liveMustGenericWrite(t, ctx, "platform_delete", "delete", PlatformGenericWriteInput{Resource: "ai_provider", ID: providerID})
	providerID = ""
	liveMustGenericWrite(t, ctx, "platform_delete", "delete", PlatformGenericWriteInput{Resource: "secret", ID: secretID})
	secretID = ""
	liveMustDeleteSource(t, ctx, sourceName)
	_ = sourceID
}

func liveMustPlatformStatus(t *testing.T, ctx context.Context) PlatformStatusOutput {
	t.Helper()
	_, output, err := HandlePlatformStatus(ctx, nil, PlatformStatusInput{})
	if err != nil {
		t.Fatalf("HandlePlatformStatus() error = %v", err)
	}
	if output.Error != "" {
		t.Fatalf("HandlePlatformStatus() output error = %q", output.Error)
	}
	return output
}

func liveMustReadOrgsProjects(t *testing.T, ctx context.Context) {
	t.Helper()
	_, orgs, err := HandlePlatformOrgs(ctx, nil, PlatformOrgsInput{Fields: []string{"id", "name", "slug"}})
	if err != nil {
		t.Fatalf("HandlePlatformOrgs() error = %v", err)
	}
	if orgs.Error != "" || orgs.Count == 0 {
		t.Fatalf("orgs output = %#v, want visible orgs", orgs)
	}
	_, projects, err := HandlePlatformProjects(ctx, nil, PlatformProjectsInput{Fields: []string{"id", "name", "slug"}})
	if err != nil {
		t.Fatalf("HandlePlatformProjects() error = %v", err)
	}
	if projects.Error != "" || projects.Count == 0 {
		t.Fatalf("projects output = %#v, want visible projects", projects)
	}
}

func liveMustReadSourceTypes(t *testing.T, ctx context.Context) {
	t.Helper()
	_, output, err := HandlePlatformSourceTypes(ctx, nil, PlatformSourceTypesInput{Fields: []string{"id", "label"}})
	if err != nil {
		t.Fatalf("HandlePlatformSourceTypes() error = %v", err)
	}
	if output.Error != "" || output.Count == 0 {
		t.Fatalf("source types output = %#v, want source types", output)
	}
}

func liveMustCreateSource(t *testing.T, ctx context.Context, name, host, port, user, password, database string) string {
	t.Helper()
	_, output, err := HandlePlatformSourceCreate(ctx, nil, PlatformSourceCreateInput{
		SourceType: "Postgres",
		Name:       name,
		Hostname:   host,
		Port:       port,
		Username:   user,
		Password:   password,
		Database:   database,
	})
	if err != nil {
		t.Fatalf("HandlePlatformSourceCreate() error = %v", err)
	}
	if output.Error != "" || !output.ConfirmationRequired {
		t.Fatalf("source create output = %#v, want pending confirmation", output)
	}
	confirm := liveMustConfirm(t, ctx, output.ConfirmationToken)
	return liveConfirmColumn(t, confirm, "source_id")
}

func liveMustUpdateSource(t *testing.T, ctx context.Context, source, name string) {
	t.Helper()
	_, output, err := HandlePlatformSourceUpdate(ctx, nil, PlatformSourceUpdateInput{Source: source, Name: name})
	if err != nil {
		t.Fatalf("HandlePlatformSourceUpdate() error = %v", err)
	}
	if output.Error != "" || !output.ConfirmationRequired {
		t.Fatalf("source update output = %#v, want pending confirmation", output)
	}
	liveMustConfirm(t, ctx, output.ConfirmationToken)
}

func liveMustDeleteSource(t *testing.T, ctx context.Context, source string) {
	t.Helper()
	_, output, err := HandlePlatformSourceDelete(ctx, nil, PlatformSourceDeleteInput{Source: source})
	if err != nil {
		t.Fatalf("HandlePlatformSourceDelete() error = %v", err)
	}
	if output.Error != "" || !output.ConfirmationRequired {
		t.Fatalf("source delete output = %#v, want pending confirmation", output)
	}
	liveMustConfirm(t, ctx, output.ConfirmationToken)
}

func liveMustReadSource(t *testing.T, ctx context.Context, sourceName string) {
	t.Helper()
	_, output, err := HandlePlatformSources(ctx, nil, PlatformSourcesInput{Fields: []string{"id", "name", "databaseType"}})
	if err != nil {
		t.Fatalf("HandlePlatformSources() error = %v", err)
	}
	if output.Error != "" {
		t.Fatalf("sources output error = %q", output.Error)
	}
	for _, source := range output.Sources {
		if source.Name == sourceName {
			return
		}
	}
	t.Fatalf("source %q not found in %#v", sourceName, output.Sources)
}

func liveMustReadSourceSchema(t *testing.T, ctx context.Context, source string) {
	t.Helper()
	_, objects, err := HandlePlatformSourceObjects(ctx, nil, PlatformSourceObjectsInput{Source: source, Parent: "Schema:public", Kinds: []string{"Table"}, PageSize: 10})
	if err != nil {
		t.Fatalf("HandlePlatformSourceObjects() error = %v", err)
	}
	if objects.Error != "" {
		t.Fatalf("source objects output error = %q", objects.Error)
	}
	ref := "Table:whodb_platform.public.users"
	_, columns, err := HandlePlatformSourceColumns(ctx, nil, PlatformSourceColumnsInput{Source: source, Ref: ref})
	if err != nil {
		t.Fatalf("HandlePlatformSourceColumns() error = %v", err)
	}
	if columns.Error != "" || len(columns.Columns) == 0 {
		t.Fatalf("source columns output = %#v, want columns", columns)
	}
	_, rows, err := HandlePlatformSourceRows(ctx, nil, PlatformSourceRowsInput{Source: source, Ref: ref, Limit: 1}, &SecurityOptions{MaxRows: 10})
	if err != nil {
		t.Fatalf("HandlePlatformSourceRows() error = %v", err)
	}
	if rows.Error != "" {
		t.Fatalf("source rows output error = %q", rows.Error)
	}
	_, configOut, err := HandlePlatformSourceConfig(ctx, nil, PlatformSourceConfigInput{Source: source})
	if err != nil {
		t.Fatalf("HandlePlatformSourceConfig() error = %v", err)
	}
	if configOut.Error != "" || configOut.Config.Password != platformapi.RedactedValue() {
		t.Fatalf("source config output = %#v, want redacted password", configOut)
	}
}

func liveMustReadFiles(t *testing.T, ctx context.Context, folderID string) {
	t.Helper()
	_, files, err := HandlePlatformFiles(ctx, nil, PlatformFilesInput{FolderID: folderID, Fields: []string{"files", "storageUsed"}})
	if err != nil {
		t.Fatalf("HandlePlatformFiles() error = %v", err)
	}
	if files.Error != "" || files.Count == 0 {
		t.Fatalf("files output = %#v, want folder contents", files)
	}
	_, search, err := HandlePlatformFileSearch(ctx, nil, PlatformFileSearchInput{Query: "mcp-e2e", Fields: []string{"id", "name", "isTabular"}})
	if err != nil {
		t.Fatalf("HandlePlatformFileSearch() error = %v", err)
	}
	if search.Error != "" {
		t.Fatalf("file search error = %q", search.Error)
	}
	_, tabular, err := HandlePlatformTabularFiles(ctx, nil, PlatformEmptyInput{Fields: []string{"id", "name", "isTabular"}})
	if err != nil {
		t.Fatalf("HandlePlatformTabularFiles() error = %v", err)
	}
	if tabular.Error != "" {
		t.Fatalf("tabular files error = %q", tabular.Error)
	}
	_, usage, err := HandlePlatformStorageUsage(ctx, nil, PlatformEmptyInput{})
	if err != nil {
		t.Fatalf("HandlePlatformStorageUsage() error = %v", err)
	}
	if usage.Error != "" {
		t.Fatalf("storage usage error = %q", usage.Error)
	}
}

func liveMustReadProjectList(t *testing.T, ctx context.Context, name string, read func() (int, string)) {
	t.Helper()
	count, outputErr := read()
	if outputErr != "" {
		t.Fatalf("%s output error = %q", name, outputErr)
	}
	if count < 0 {
		t.Fatalf("%s count = %d, want non-negative", name, count)
	}
}

func liveMustReadEntity(t *testing.T, ctx context.Context, name, id string, read func() (string, error)) {
	t.Helper()
	if strings.TrimSpace(id) == "" {
		t.Fatalf("%s id is empty", name)
	}
	outputErr, err := read()
	if err != nil {
		t.Fatalf("read %s %s error = %v", name, id, err)
	}
	if outputErr != "" {
		t.Fatalf("read %s %s output error = %q", name, id, outputErr)
	}
}

func liveMustGenericWriteID(t *testing.T, ctx context.Context, toolName, operationKind string, input PlatformGenericWriteInput) string {
	t.Helper()
	result := liveMustGenericWrite(t, ctx, toolName, operationKind, input)
	var decoded struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(result), &decoded); err != nil {
		t.Fatalf("decode %s result JSON: %v\n%s", toolName, err, result)
	}
	if decoded.ID == "" {
		t.Fatalf("%s result JSON did not include id: %s", toolName, result)
	}
	return decoded.ID
}

func liveMustGenericWrite(t *testing.T, ctx context.Context, toolName, operationKind string, input PlatformGenericWriteInput) string {
	t.Helper()
	_, output, err := handlePlatformGenericWrite(ctx, toolName, input, operationKind, true)
	if err != nil {
		t.Fatalf("handlePlatformGenericWrite(%s, %s/%s) error = %v", toolName, input.Resource, input.Action, err)
	}
	if output.Error != "" || !output.ConfirmationRequired {
		t.Fatalf("write output for %s %s/%s = %#v, want pending confirmation", toolName, input.Resource, input.Action, output)
	}
	confirm := liveMustConfirm(t, ctx, output.ConfirmationToken)
	return liveConfirmColumn(t, confirm, "result_json")
}

func liveMustConfirm(t *testing.T, ctx context.Context, token string) ConfirmOutput {
	t.Helper()
	_, confirm, err := HandlePlatformConfirm(ctx, nil, ConfirmInput{Token: token})
	if err != nil {
		t.Fatalf("HandlePlatformConfirm() error = %v", err)
	}
	if confirm.Error != "" {
		t.Fatalf("HandlePlatformConfirm() output error = %q", confirm.Error)
	}
	if len(confirm.Rows) == 0 {
		t.Fatalf("confirm output = %#v, want at least one row", confirm)
	}
	return confirm
}

func liveConfirmColumn(t *testing.T, output ConfirmOutput, column string) string {
	t.Helper()
	index := -1
	for i, name := range output.Columns {
		if name == column {
			index = i
			break
		}
	}
	if index < 0 {
		t.Fatalf("confirm columns = %#v, missing %q", output.Columns, column)
	}
	if len(output.Rows) == 0 || len(output.Rows[0]) <= index {
		t.Fatalf("confirm rows = %#v, missing column %q", output.Rows, column)
	}
	value, _ := output.Rows[0][index].(string)
	if strings.TrimSpace(value) == "" {
		t.Fatalf("confirm %s = %#v, want non-empty string", column, output.Rows[0][index])
	}
	return value
}

func liveBestEffortSourceDelete(ctx context.Context, source string) {
	if strings.TrimSpace(source) == "" {
		return
	}
	_, output, err := HandlePlatformSourceDelete(ctx, nil, PlatformSourceDeleteInput{Source: source})
	if err == nil && output.Error == "" && output.ConfirmationToken != "" {
		_, _, _ = HandlePlatformConfirm(ctx, nil, ConfirmInput{Token: output.ConfirmationToken})
	}
}

func liveBestEffortGenericDelete(ctx context.Context, resource, id string) {
	if strings.TrimSpace(id) == "" {
		return
	}
	_, output, err := handlePlatformGenericWrite(ctx, "platform_delete", PlatformGenericWriteInput{Resource: resource, ID: id}, "delete", true)
	if err == nil && output.Error == "" && output.ConfirmationToken != "" {
		_, _, _ = HandlePlatformConfirm(ctx, nil, ConfirmInput{Token: output.ConfirmationToken})
	}
}

func liveJSON(t *testing.T, value any) string {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal JSON payload: %v", err)
	}
	return string(raw)
}

func liveMintDevRefreshToken(t *testing.T, ctx context.Context, keycloakURL, hostHeader, username, password string) string {
	t.Helper()
	endpoint := strings.TrimRight(keycloakURL, "/") + "/realms/mothergate/protocol/openid-connect/token"
	form := url.Values{
		"client_id":  {"whodb-cli"},
		"grant_type": {"password"},
		"username":   {username},
		"password":   {password},
		"scope":      {"openid email profile"},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatalf("create token request: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if hostHeader != "" {
		req.Host = hostHeader
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("mint local dev token from %s: %v", endpoint, err)
	}
	defer resp.Body.Close()
	var body bytes.Buffer
	if _, err := body.ReadFrom(resp.Body); err != nil {
		t.Fatalf("read token response: %v", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		t.Fatalf("mint local dev token failed: %s: %s", resp.Status, strings.TrimSpace(body.String()))
	}
	var tokens platformapi.TokenResponse
	if err := json.Unmarshal(body.Bytes(), &tokens); err != nil {
		t.Fatalf("decode token response: %v", err)
	}
	if strings.TrimSpace(tokens.RefreshToken) == "" {
		t.Fatalf("token response did not include refresh token")
	}
	return tokens.RefreshToken
}

func liveSeedPlatformLogin(t *testing.T, ctx context.Context, host, refreshToken string) {
	t.Helper()
	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("load CLI config: %v", err)
	}
	tokens, err := platformapi.RefreshToken(ctx, host, refreshToken)
	if err != nil {
		t.Fatalf("refresh local dev token through WhoDB auth host: %v", err)
	}
	client, err := platformapi.NewClient(host, tokens.AccessToken)
	if err != nil {
		t.Fatalf("new platform client: %v", err)
	}
	manifest, _ := client.PlatformManifest(ctx)
	client.SetPlatformManifest(manifest)
	user, err := client.Me(ctx)
	if err != nil {
		t.Fatalf("load platform user: %v", err)
	}
	hostEntry := config.PlatformHost{URL: client.Host(), AccountID: user.ID, Email: user.Email}
	if manifest != nil {
		raw, err := json.Marshal(manifest)
		if err != nil {
			t.Fatalf("marshal manifest: %v", err)
		}
		hostEntry.Manifest = &config.PlatformManifestCache{
			PlatformVersion:         manifest.PlatformVersion,
			ManifestProtocolVersion: manifest.ManifestProtocolVersion,
			FetchedAt:               time.Now().UTC().Format(time.RFC3339),
			Raw:                     raw,
		}
	}
	orgs, err := client.Organizations(ctx)
	if err != nil {
		t.Fatalf("load organizations: %v", err)
	}
	for _, org := range orgs {
		if org.Slug == "acme" {
			hostEntry.DefaultOrgID = org.ID
			hostEntry.DefaultOrgName = org.Name
			break
		}
	}
	if hostEntry.DefaultOrgID == "" {
		t.Fatalf("seeded acme organization was not visible")
	}
	projects, err := client.Projects(ctx, hostEntry.DefaultOrgID)
	if err != nil {
		t.Fatalf("load projects: %v", err)
	}
	for _, project := range projects {
		if project.Slug == "default" {
			hostEntry.DefaultProjectID = project.ID
			hostEntry.DefaultProjectName = project.Name
			break
		}
	}
	if hostEntry.DefaultProjectID == "" {
		t.Fatalf("seeded default project was not visible")
	}
	cfg.SetOnlyPlatformHost(hostEntry)
	tokenToStore := tokens.RefreshToken
	if tokenToStore == "" {
		tokenToStore = refreshToken
	}
	if err := cfg.SavePlatformRefreshToken(client.Host(), user.ID, tokenToStore); err != nil {
		t.Fatalf("save platform refresh token: %v", err)
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("save CLI config: %v", err)
	}
}

func liveEnvOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
