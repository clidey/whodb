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
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/clidey/whodb/cli/e2e/testharness"
	"github.com/clidey/whodb/cli/internal/config"
	platformapi "github.com/clidey/whodb/cli/internal/platform"
	_ "github.com/jackc/pgx/v5/stdlib"
)

const (
	defaultLivePlatformE2EHost        = "http://localhost:18080"
	defaultLivePlatformE2EKeycloakURL = "http://localhost:14001"
	defaultLivePlatformE2EUser        = "owner@acme.test"
	defaultLivePlatformE2EPassword    = "password"
	defaultLivePlatformE2EDBPort      = "15431"
)

var livePlatformCoverage *livePlatformMCPExhaustiveCoverage

type livePlatformMCPExhaustiveCoverage struct {
	tools      map[string]bool
	writeSpecs map[string]bool
}

func newLivePlatformMCPExhaustiveCoverage() *livePlatformMCPExhaustiveCoverage {
	return &livePlatformMCPExhaustiveCoverage{
		tools:      map[string]bool{},
		writeSpecs: map[string]bool{},
	}
}

func (c *livePlatformMCPExhaustiveCoverage) Assert(t *testing.T) {
	t.Helper()
	var missingTools []string
	for _, tool := range platformToolDefinitions() {
		if !c.tools[tool.Name] {
			missingTools = append(missingTools, tool.Name)
		}
	}
	sort.Strings(missingTools)
	if len(missingTools) > 0 {
		t.Fatalf("live MCP platform e2e did not cover tools: %s", strings.Join(missingTools, ", "))
	}

	var missingSpecs []string
	for key := range platformapi.GenericWriteSpecs {
		if !c.writeSpecs[key] {
			missingSpecs = append(missingSpecs, key)
		}
	}
	sort.Strings(missingSpecs)
	if len(missingSpecs) > 0 {
		t.Fatalf("live MCP platform e2e did not cover generic write specs: %s", strings.Join(missingSpecs, ", "))
	}
}

func liveCoverTool(name string) {
	if livePlatformCoverage != nil {
		livePlatformCoverage.tools[name] = true
	}
}

func liveCoverGenericWrite(operationKind string, input PlatformGenericWriteInput) {
	if livePlatformCoverage == nil {
		return
	}
	resource := normalizePlatformWriteToken(input.Resource)
	action := normalizePlatformWriteToken(operationKind)
	if operationKind == "action" {
		action = normalizePlatformWriteToken(input.Action)
	}
	key := action + ":" + resource
	if operationKind == "action" {
		key = "action:" + action + ":" + resource
	}
	livePlatformCoverage.writeSpecs[key] = true
}

func TestPlatformMCP_RealReadWriteLifecycle(t *testing.T) {
	cleanup := testharness.SetupCleanEnv(t)
	defer cleanup()
	t.Setenv("WHODB_CLI_E2E_PLATFORM_TOKEN_DIR", filepath.Join(os.Getenv("HOME"), ".whodb-cli-platform-e2e-tokens"))
	config.ResetPathsForTesting()
	clearPendingPlatformActions(t)
	coverage := newLivePlatformMCPExhaustiveCoverage()
	livePlatformCoverage = coverage
	t.Cleanup(func() { livePlatformCoverage = nil })

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
	sourceLocalPort := liveEnvOrDefault("WHODB_PLATFORM_E2E_SOURCE_LOCAL_PORT", defaultLivePlatformE2EDBPort)

	liveMustPlatformSetupStatus(t, ctx)
	status := liveMustPlatformStatus(t, ctx)
	if status.Email != email || !status.WorkspaceSelected {
		t.Fatalf("platform status = %#v, want %s with selected workspace", status, email)
	}
	liveMustPlatformDoctor(t, ctx)
	liveMustReadOrgsProjects(t, ctx)
	liveMustProjectTools(t, ctx, suffix)
	liveMustReadSourceTypes(t, ctx)

	sourceID := liveMustCreateSource(t, ctx, sourceName, sourceHost, sourcePort, sourceUser, sourcePassword, sourceDatabase)
	defer liveBestEffortSourceDelete(ctx, sourceName)
	liveMustReadSource(t, ctx, sourceName)
	liveMustReadSourceSchema(t, ctx, sourceName)
	sourceObjectTable := "mcp_e2e_source_object_" + suffix
	liveMustExerciseSourceObjectWrites(t, ctx, sourceID, sourceName, sourceLocalPort, sourceUser, sourcePassword, sourceDatabase, sourceObjectTable)
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
	liveCoverTool("whodb_platform_secrets")
	liveMustGenericWrite(t, ctx, "platform_update", "update", PlatformGenericWriteInput{
		Resource:    "secret",
		ID:          secretID,
		PayloadJSON: liveJSON(t, map[string]any{"name": "mcp-e2e-secret-updated-" + suffix, "description": "updated", "value": "rotated-value"}),
	})

	providerID := liveMustGenericWriteID(t, ctx, "platform_create", "create", PlatformGenericWriteInput{
		Resource:    "ai_provider",
		PayloadJSON: liveJSON(t, map[string]any{"name": "mcp-e2e-provider-" + suffix, "providerType": "openai", "endpoint": "http://127.0.0.1:1/v1", "apiKey": "test-key"}),
	})
	defer liveBestEffortGenericDelete(ctx, "ai_provider", providerID)
	liveMustReadProjectList(t, ctx, "ai providers", func() (int, string) {
		_, out, err := HandlePlatformAIProviders(ctx, nil, PlatformEmptyInput{Fields: []string{"id", "name", "providerType"}})
		if err != nil {
			t.Fatalf("HandlePlatformAIProviders() error = %v", err)
		}
		return out.Count, out.Error
	})
	liveCoverTool("whodb_platform_ai_providers")
	liveMustReadExpectedOutputError(t, ctx, "ai provider models", func() (string, error) {
		_, out, err := HandlePlatformAIProviderModels(ctx, nil, PlatformProviderModelsInput{ProviderID: providerID, Fields: []string{"data", "scope"}})
		return out.Error, err
	})
	liveCoverTool("whodb_platform_ai_provider_models")
	liveMustGenericWrite(t, ctx, "platform_update", "update", PlatformGenericWriteInput{
		Resource:    "ai_provider",
		ID:          providerID,
		PayloadJSON: liveJSON(t, map[string]any{"name": "mcp-e2e-provider-updated-" + suffix, "endpoint": "http://127.0.0.1:1/v1"}),
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
	liveMustTypedWriteConfirmation(t, ctx, "whodb_platform_create_dataset", func() (PlatformGenericWriteOutput, error) {
		_, out, err := handlePlatformCreateDataset(ctx, PlatformCreateDatasetInput{
			Name:       "mcp-e2e-typed-dataset-" + suffix,
			SchemaMode: "manual",
			Columns:    []PlatformDatasetColumnInput{{Name: "id", Type: "text", IsPrimary: true}},
		}, true)
		return out, err
	})
	liveMustReadProjectList(t, ctx, "datasets", func() (int, string) {
		_, out, err := HandlePlatformDatasets(ctx, nil, PlatformEmptyInput{Fields: []string{"id", "name"}})
		if err != nil {
			t.Fatalf("HandlePlatformDatasets() error = %v", err)
		}
		return out.Count, out.Error
	})
	liveCoverTool("whodb_platform_datasets")
	liveMustReadEntity(t, ctx, "dataset", datasetID, func() (string, error) {
		_, out, err := HandlePlatformDataset(ctx, nil, PlatformEntityInput{ID: datasetID, Fields: []string{"data", "scope"}})
		return out.Error, err
	})
	liveCoverTool("whodb_platform_dataset")
	liveMustReadEntity(t, ctx, "dataset rows", datasetID, func() (string, error) {
		_, out, err := HandlePlatformDatasetRows(ctx, nil, PlatformRowsInput{ID: datasetID, Limit: 5, Fields: []string{"data", "scope"}}, &SecurityOptions{MaxRows: 10})
		return out.Error, err
	})
	liveCoverTool("whodb_platform_dataset_rows")
	liveMustGenericWrite(t, ctx, "platform_update", "update", PlatformGenericWriteInput{
		Resource:    "dataset",
		ID:          datasetID,
		PayloadJSON: liveJSON(t, map[string]any{"name": "mcp-e2e-dataset-updated-" + suffix, "description": "updated"}),
	})
	datasetCloneID := liveMustClone(t, ctx, PlatformCloneInput{Resource: "dataset", Source: datasetID, NewName: "mcp-e2e-dataset-clone-" + suffix})
	defer liveBestEffortGenericDelete(ctx, "dataset", datasetCloneID)

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
	liveMustReadEntity(t, ctx, "transform", transformID, func() (string, error) {
		_, out, err := HandlePlatformTransform(ctx, nil, PlatformEntityInput{ID: transformID, Fields: []string{"data", "scope"}})
		return out.Error, err
	})
	liveMustGenericWrite(t, ctx, "platform_action", "action", PlatformGenericWriteInput{
		Resource: "transform",
		Action:   "run",
		ID:       transformID,
	})
	liveMustReadEntity(t, ctx, "transform runs", transformID, func() (string, error) {
		_, out, err := HandlePlatformTransformRuns(ctx, nil, PlatformTransformRunsInput{TransformID: transformID, Limit: 5, Fields: []string{"items", "count"}})
		return out.Error, err
	})
	liveCoverTool("whodb_platform_transforms")
	liveCoverTool("whodb_platform_transform")
	liveCoverTool("whodb_platform_transform_runs")

	targetOntologyID := liveMustGenericWriteID(t, ctx, "platform_create", "create", PlatformGenericWriteInput{
		Resource: "ontology",
		PayloadJSON: liveJSON(t, map[string]any{
			"apiName":           "mcp_e2e_target_" + suffix,
			"displayName":       "MCP E2E Target " + suffix,
			"pluralDisplayName": "MCP E2E Targets " + suffix,
			"description":       "MCP e2e target ontology",
			"primaryKey":        "id",
			"tableName":         "mcp_e2e_target_" + suffix,
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
	defer liveBestEffortGenericDelete(ctx, "ontology", targetOntologyID)
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
			"properties": []map[string]any{
				{
					"apiName": "id", "displayName": "ID", "description": "ID", "columnName": "id", "dataType": "String",
					"isRequired": true, "visibility": "normal", "isSearchable": true, "isSortable": true, "isEditOnly": false,
				},
				{
					"apiName": "target_id", "displayName": "Target ID", "description": "Target ID", "columnName": "target_id", "dataType": "String",
					"isRequired": false, "visibility": "normal", "isSearchable": true, "isSortable": true, "isEditOnly": false,
				},
			},
			"links": []map[string]any{{
				"apiName":                  "target",
				"targetOntologyApiName":    "mcp_e2e_target_" + suffix,
				"cardinality":              "MANY_TO_ONE",
				"foreignKeyProperty":       "target_id",
				"targetForeignKeyProperty": "id",
				"joinTable":                "",
				"sourceColumnInJoinTable":  "",
				"targetColumnInJoinTable":  "",
				"displayName":              "Target",
				"pluralDisplayName":        "Targets",
				"reverseDisplayName":       "Entities",
			}},
		}),
	})
	defer liveBestEffortGenericDelete(ctx, "ontology", ontologyID)
	liveMustReadProjectList(t, ctx, "ontologies", func() (int, string) {
		_, out, err := HandlePlatformOntologies(ctx, nil, PlatformEmptyInput{Fields: []string{"id", "apiName", "displayName"}})
		if err != nil {
			t.Fatalf("HandlePlatformOntologies() error = %v", err)
		}
		return out.Count, out.Error
	})
	liveCoverTool("whodb_platform_ontologies")
	liveMustReadEntity(t, ctx, "ontology", ontologyID, func() (string, error) {
		_, out, err := HandlePlatformOntology(ctx, nil, PlatformEntityInput{ID: ontologyID, Fields: []string{"data", "scope"}})
		return out.Error, err
	})
	liveCoverTool("whodb_platform_ontology")
	liveMustReadEntity(t, ctx, "ontology rows", ontologyID, func() (string, error) {
		_, out, err := HandlePlatformOntologyRows(ctx, nil, PlatformRowsInput{ID: ontologyID, Limit: 5, Fields: []string{"data", "scope"}}, &SecurityOptions{MaxRows: 10})
		return out.Error, err
	})
	liveCoverTool("whodb_platform_ontology_rows")
	liveMustReadEntity(t, ctx, "ontology follow link", ontologyID, func() (string, error) {
		_, out, err := HandlePlatformOntologyFollowLink(ctx, nil, PlatformOntologyFollowLinkInput{EntityID: ontologyID, PrimaryKey: "00000000-0000-0000-0000-000000000000", LinkAPIName: "target", Limit: 5, Fields: []string{"data", "scope"}}, &SecurityOptions{MaxRows: 10})
		return out.Error, err
	})
	liveCoverTool("whodb_platform_ontology_follow_link")
	lookupID := liveMustGenericWriteID(t, ctx, "platform_create", "create", PlatformGenericWriteInput{
		Resource:    "ontology_fast_lookup",
		PayloadJSON: liveJSON(t, map[string]any{"entityId": ontologyID, "fields": []string{"id"}, "reason": "MCP e2e lookup"}),
	})
	defer liveBestEffortGenericDelete(ctx, "ontology_fast_lookup", lookupID)
	liveMustTypedWriteConfirmation(t, ctx, "whodb_platform_create_ontology_fast_lookup", func() (PlatformGenericWriteOutput, error) {
		_, out, err := handlePlatformCreateOntologyFastLookup(ctx, PlatformOntologyFastLookupInput{EntityID: ontologyID, Fields: []string{"id"}, Reason: "typed MCP e2e"}, true)
		return out, err
	})
	liveMustTypedWriteConfirmation(t, ctx, "whodb_platform_delete_ontology_fast_lookup", func() (PlatformGenericWriteOutput, error) {
		_, out, err := handlePlatformTypedGenericWrite(ctx, "whodb_platform_delete_ontology_fast_lookup", PlatformGenericWriteInput{Resource: "ontology_fast_lookup", ID: lookupID}, "delete", true)
		return out, err
	})
	liveMustReadEntity(t, ctx, "ontology fast lookups", ontologyID, func() (string, error) {
		_, out, err := HandlePlatformOntologyFastLookups(ctx, nil, PlatformEntityInput{ID: ontologyID, Fields: []string{"items", "count"}})
		return out.Error, err
	})
	liveCoverTool("whodb_platform_ontology_fast_lookups")
	liveMustReadEntity(t, ctx, "ontology fast lookup suggestions", ontologyID, func() (string, error) {
		_, out, err := HandlePlatformOntologyFastLookupSuggestions(ctx, nil, PlatformEntityInput{ID: ontologyID, Fields: []string{"items", "count"}})
		return out.Error, err
	})
	liveCoverTool("whodb_platform_ontology_fast_lookup_suggestions")
	liveMustTypedWriteConfirmation(t, ctx, "whodb_platform_add_ontology_record", func() (PlatformGenericWriteOutput, error) {
		_, out, err := handlePlatformOntologyRecordWrite(ctx, "whodb_platform_add_ontology_record", PlatformOntologyRecordInput{EntityID: ontologyID, Values: map[string]string{"id": "typed-1", "target_id": ""}}, "add_record", true)
		return out, err
	})
	liveCoverGenericWrite("action", PlatformGenericWriteInput{Resource: "ontology", Action: "add_record", ID: ontologyID})
	liveMustTypedWriteConfirmation(t, ctx, "whodb_platform_update_ontology_record", func() (PlatformGenericWriteOutput, error) {
		_, out, err := handlePlatformOntologyRecordWrite(ctx, "whodb_platform_update_ontology_record", PlatformOntologyRecordInput{EntityID: ontologyID, Values: map[string]string{"id": "typed-1", "target_id": "typed-2"}, UpdateColumns: []string{"target_id"}}, "update_record", true)
		return out, err
	})
	liveCoverGenericWrite("action", PlatformGenericWriteInput{Resource: "ontology", Action: "update_record", ID: ontologyID})
	liveMustTypedWriteConfirmation(t, ctx, "whodb_platform_delete_ontology_record", func() (PlatformGenericWriteOutput, error) {
		_, out, err := handlePlatformOntologyRecordWrite(ctx, "whodb_platform_delete_ontology_record", PlatformOntologyRecordInput{EntityID: ontologyID, Values: map[string]string{"id": "typed-1"}}, "delete_record", true)
		return out, err
	})
	liveCoverGenericWrite("action", PlatformGenericWriteInput{Resource: "ontology", Action: "delete_record", ID: ontologyID})
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
			"properties": []map[string]any{
				{
					"apiName": "id", "displayName": "ID", "description": "ID", "columnName": "id", "dataType": "String",
					"isRequired": true, "visibility": "normal", "isSearchable": true, "isSortable": true, "isEditOnly": false,
				},
				{
					"apiName": "target_id", "displayName": "Target ID", "description": "Target ID", "columnName": "target_id", "dataType": "String",
					"isRequired": false, "visibility": "normal", "isSearchable": true, "isSortable": true, "isEditOnly": false,
				},
			},
			"links": []map[string]any{{
				"apiName":                  "target",
				"targetOntologyApiName":    "mcp_e2e_target_" + suffix,
				"cardinality":              "MANY_TO_ONE",
				"foreignKeyProperty":       "target_id",
				"targetForeignKeyProperty": "id",
				"joinTable":                "",
				"sourceColumnInJoinTable":  "",
				"targetColumnInJoinTable":  "",
				"displayName":              "Target",
				"pluralDisplayName":        "Targets",
				"reverseDisplayName":       "Entities",
			}},
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
	liveCoverTool("whodb_platform_function")
	liveMustReadProjectList(t, ctx, "functions", func() (int, string) {
		_, out, err := HandlePlatformFunctions(ctx, nil, PlatformEmptyInput{Fields: []string{"id", "name"}})
		if err != nil {
			t.Fatalf("HandlePlatformFunctions() error = %v", err)
		}
		return out.Count, out.Error
	})
	liveCoverTool("whodb_platform_functions")
	liveMustGenericWrite(t, ctx, "platform_update", "update", PlatformGenericWriteInput{
		Resource:    "function",
		ID:          functionID,
		PayloadJSON: liveJSON(t, map[string]any{"name": "mcp-e2e-function-updated-" + suffix, "description": "updated", "language": "python", "entryPoint": "main", "files": []map[string]any{{"path": "main.py", "content": "def main(input):\n    return input\n"}}, "dependencies": []map[string]any{}}),
	})
	liveMustGenericWriteConfirmError(t, ctx, "platform_action", "action", PlatformGenericWriteInput{Resource: "function", Action: "deploy", ID: functionID})
	liveMustGenericWriteConfirmError(t, ctx, "platform_action", "action", PlatformGenericWriteInput{Resource: "function", Action: "redeploy", ID: functionID})

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
	liveCoverTool("whodb_platform_file_preview")
	liveMustReadEntity(t, ctx, "file inspect", fileID, func() (string, error) {
		_, out, err := HandlePlatformFileInspect(ctx, nil, PlatformFileInspectInput{FileID: fileID, Fields: []string{"columns", "columnMapExample"}})
		if out.Error == "" {
			inspection, ok := out.Data.(map[string]any)
			columns, _ := inspection["columns"].([]any)
			columnMapExample, _ := inspection["columnMapExample"].(string)
			if !ok || len(columns) == 0 || columnMapExample == "" {
				return fmt.Sprintf("unexpected inspection payload: %#v", out.Data), err
			}
		}
		return out.Error, err
	})
	liveCoverTool("whodb_platform_file_inspect")
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
	liveMustTypedWriteConfirmation(t, ctx, "whodb_platform_promote_file_to_dataset", func() (PlatformGenericWriteOutput, error) {
		sheetIndex := 0
		_, out, err := handlePlatformPromoteFileToDataset(ctx, PlatformPromoteFileToDatasetInput{
			FileID:      fileID,
			Name:        "mcp-e2e-typed-promoted-" + suffix,
			Description: "typed MCP e2e promoted dataset",
			SheetIndex:  &sheetIndex,
			ColumnMap: []PlatformFileColumnMapInput{
				{SourceColumn: "id", DatasetColumn: "id", DataType: "text", IsPrimary: true},
				{SourceColumn: "name", DatasetColumn: "name", DataType: "text", IsNullable: true},
			},
		}, true)
		return out, err
	})
	liveMustReadEntity(t, ctx, "promoted dataset", promotedDatasetID, func() (string, error) {
		_, out, err := HandlePlatformDataset(ctx, nil, PlatformEntityInput{ID: promotedDatasetID, Fields: []string{"data", "scope"}})
		return out.Error, err
	})
	liveCoverTool("whodb_platform_dataset")
	liveMustReadEntity(t, ctx, "lineage", promotedDatasetID, func() (string, error) {
		_, out, err := HandlePlatformLineage(ctx, nil, PlatformLineageInput{RootID: promotedDatasetID, RootType: "dataset", Direction: "both", MaxDepth: 5, Fields: []string{"data", "count"}})
		return out.Error, err
	})
	liveCoverTool("whodb_platform_lineage")
	liveMustReadEntity(t, ctx, "lineage neighbors", promotedDatasetID, func() (string, error) {
		_, out, err := HandlePlatformLineageNeighbors(ctx, nil, PlatformLineageNeighborsInput{NodeID: promotedDatasetID, NodeType: "dataset", Fields: []string{"data", "count"}})
		return out.Error, err
	})
	liveCoverTool("whodb_platform_lineage_neighbors")
	liveMustReadProjectList(t, ctx, "project lineage", func() (int, string) {
		_, out, err := HandlePlatformProjectLineage(ctx, nil, PlatformEmptyInput{Fields: []string{"data", "count"}})
		if err != nil {
			t.Fatalf("HandlePlatformProjectLineage() error = %v", err)
		}
		return out.Count, out.Error
	})
	liveCoverTool("whodb_platform_project_lineage")
	liveMustWorkspaceIntelligence(t, ctx, promotedDatasetID)
	liveMustBundleTools(t, ctx, suffix)

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
	liveMustGenericWrite(t, ctx, "platform_delete", "delete", PlatformGenericWriteInput{Resource: "dataset", ID: datasetCloneID})
	datasetCloneID = ""
	liveMustGenericWrite(t, ctx, "platform_delete", "delete", PlatformGenericWriteInput{Resource: "dataset", ID: datasetID})
	datasetID = ""
	liveMustGenericWrite(t, ctx, "platform_delete", "delete", PlatformGenericWriteInput{Resource: "ontology", ID: ontologyID})
	ontologyID = ""
	liveMustGenericWrite(t, ctx, "platform_delete", "delete", PlatformGenericWriteInput{Resource: "ontology", ID: targetOntologyID})
	targetOntologyID = ""
	liveMustGenericWrite(t, ctx, "platform_delete", "delete", PlatformGenericWriteInput{Resource: "ai_provider", ID: providerID})
	providerID = ""
	liveMustGenericWrite(t, ctx, "platform_delete", "delete", PlatformGenericWriteInput{Resource: "secret", ID: secretID})
	secretID = ""
	liveMustDeleteSource(t, ctx, sourceName)
	_ = sourceID
	coverage.Assert(t)
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
	liveCoverTool("whodb_platform_status")
	return output
}

func liveMustPlatformSetupStatus(t *testing.T, ctx context.Context) {
	t.Helper()
	_, output, err := HandlePlatformSetupStatus(ctx, nil, PlatformSetupStatusInput{})
	if err != nil {
		t.Fatalf("HandlePlatformSetupStatus() error = %v", err)
	}
	if output.Error != "" || output.Status != "ready" || !output.Authenticated || !output.WorkspaceSelected {
		t.Fatalf("platform setup status = %#v, want ready authenticated workspace", output)
	}
	liveCoverTool("whodb_platform_setup_status")
}

func liveMustPlatformDoctor(t *testing.T, ctx context.Context) {
	t.Helper()
	_, output, err := HandlePlatformDoctor(ctx, nil, PlatformDoctorInput{})
	if err != nil {
		t.Fatalf("HandlePlatformDoctor() error = %v", err)
	}
	if output.Error != "" || !output.WorkspaceSelected {
		t.Fatalf("platform doctor = %#v, want selected workspace without error", output)
	}
	liveCoverTool("whodb_platform_doctor")
}

func liveMustBundleTools(t *testing.T, ctx context.Context, suffix string) {
	t.Helper()
	_, exported, err := HandlePlatformBundleExport(ctx, nil, PlatformBundleExportInput{})
	if err != nil {
		t.Fatalf("HandlePlatformBundleExport() error = %v", err)
	}
	if exported.Error != "" || exported.Bundle == nil {
		t.Fatalf("bundle export = %#v, want bundle without error", exported)
	}
	liveCoverTool("whodb_platform_bundle_export")
	rawBundle, err := json.Marshal(exported.Bundle)
	if err != nil {
		t.Fatalf("marshal exported bundle: %v", err)
	}
	_, diff, err := HandlePlatformBundlePlan(ctx, nil, PlatformBundlePlanInput{BundleJSON: string(rawBundle)}, true, "platform_bundle_diff")
	if err != nil {
		t.Fatalf("HandlePlatformBundlePlan(diff) error = %v", err)
	}
	if diff.Error != "" || diff.Plan == nil || len(diff.Plan.Actions) == 0 {
		t.Fatalf("bundle diff = %#v, want plan actions without error", diff)
	}
	liveCoverTool("whodb_platform_bundle_diff")
	_, plan, err := HandlePlatformBundlePlan(ctx, nil, PlatformBundlePlanInput{BundleJSON: string(rawBundle)}, true, "platform_bundle_import_plan")
	if err != nil {
		t.Fatalf("HandlePlatformBundlePlan(import plan) error = %v", err)
	}
	if plan.Error != "" || plan.Plan == nil || len(plan.Plan.Actions) == 0 {
		t.Fatalf("bundle import plan = %#v, want plan actions without error", plan)
	}
	liveCoverTool("whodb_platform_bundle_import_plan")

	importDatasetName := "mcp-e2e-bundle-import-dataset-" + suffix
	importBundle := platformapi.ProjectBundle{
		BundleVersion: 1,
		Datasets: []platformapi.Dataset{{
			Name:        importDatasetName,
			Description: "MCP e2e bundle imported dataset",
			SchemaMode:  "manual",
			Schema: []platformapi.ColumnDef{
				{Name: "id", Type: "text", IsPrimary: true},
				{Name: "name", Type: "text", IsNullable: true},
			},
		}},
	}
	rawImportBundle, err := json.Marshal(importBundle)
	if err != nil {
		t.Fatalf("marshal import bundle: %v", err)
	}
	_, importOutput, err := HandlePlatformBundleImport(ctx, nil, PlatformBundlePlanInput{BundleJSON: string(rawImportBundle)}, true)
	if err != nil {
		t.Fatalf("HandlePlatformBundleImport() error = %v", err)
	}
	if importOutput.Error != "" || !importOutput.ConfirmationRequired {
		t.Fatalf("bundle import output = %#v, want pending confirmation", importOutput)
	}
	liveCoverTool("whodb_platform_bundle_import")
	liveMustReadPending(t, ctx)
	confirm := liveMustConfirm(t, ctx, importOutput.ConfirmationToken)
	importedDatasetID := liveConfirmRowColumn(t, confirm, "dataset", importDatasetName, "target_id")
	defer liveBestEffortGenericDelete(ctx, "dataset", importedDatasetID)
	liveMustReadEntity(t, ctx, "bundle imported dataset", importedDatasetID, func() (string, error) {
		_, out, err := HandlePlatformDataset(ctx, nil, PlatformEntityInput{ID: importedDatasetID, Fields: []string{"data", "scope"}})
		return out.Error, err
	})
}

func liveMustWorkspaceIntelligence(t *testing.T, ctx context.Context, datasetID string) {
	t.Helper()
	_, workspaceMap, err := HandlePlatformWorkspaceMap(ctx, nil, PlatformWorkspaceMapInput{Fields: []string{"counts", "warnings", "lineage"}})
	if err != nil {
		t.Fatalf("HandlePlatformWorkspaceMap() error = %v", err)
	}
	if workspaceMap.Error != "" || workspaceMap.Count == 0 {
		t.Fatalf("workspace map = %#v, want mapped resources without error", workspaceMap)
	}
	liveCoverTool("whodb_platform_workspace_map")

	_, graph, err := HandlePlatformResourceGraph(ctx, nil, PlatformResourceGraphInput{Fields: []string{"nodes", "edges", "counts"}})
	if err != nil {
		t.Fatalf("HandlePlatformResourceGraph() error = %v", err)
	}
	if graph.Error != "" || graph.Count == 0 {
		t.Fatalf("resource graph = %#v, want graph nodes without error", graph)
	}
	liveCoverTool("whodb_platform_resource_graph")

	_, actions, err := HandlePlatformNextActions(ctx, nil, PlatformNextActionsInput{Goal: "validate e2e workspace", Fields: []string{"actions", "warnings", "goal"}})
	if err != nil {
		t.Fatalf("HandlePlatformNextActions() error = %v", err)
	}
	if actions.Error != "" || actions.Count == 0 {
		t.Fatalf("next actions = %#v, want suggested actions without error", actions)
	}
	liveCoverTool("whodb_platform_next_actions")

	_, summary, err := HandlePlatformWorkspaceSummary(ctx, nil, PlatformWorkspaceSummaryInput{Goal: "build a customer app", Fields: []string{"counts", "highlights", "gaps", "next_actions", "recommended_tools"}})
	if err != nil {
		t.Fatalf("HandlePlatformWorkspaceSummary() error = %v", err)
	}
	if summary.Error != "" || summary.Count == 0 {
		t.Fatalf("workspace summary = %#v, want summary without error", summary)
	}
	liveCoverTool("whodb_platform_workspace_summary")

	_, buildPlan, err := HandlePlatformBuildPlan(ctx, nil, PlatformBuildPlanInput{Goal: "build a customer app", Fields: []string{"phases", "prerequisites", "gaps", "warnings"}})
	if err != nil {
		t.Fatalf("HandlePlatformBuildPlan() error = %v", err)
	}
	if buildPlan.Error != "" || buildPlan.Count == 0 {
		t.Fatalf("build plan = %#v, want phases without error", buildPlan)
	}
	liveCoverTool("whodb_platform_build_plan")

	_, gaps, err := HandlePlatformGapAnalysis(ctx, nil, PlatformGapAnalysisInput{Goal: "build a customer app", Fields: []string{"ready", "gaps", "counts", "next_actions"}})
	if err != nil {
		t.Fatalf("HandlePlatformGapAnalysis() error = %v", err)
	}
	if gaps.Error != "" {
		t.Fatalf("gap analysis = %#v, want no error", gaps)
	}
	liveCoverTool("whodb_platform_gap_analysis")

	_, health, err := HandlePlatformProjectHealth(ctx, nil, PlatformNextActionsInput{Fields: []string{"counts", "checks", "warnings", "next"}})
	if err != nil {
		t.Fatalf("HandlePlatformProjectHealth() error = %v", err)
	}
	if health.Error != "" || health.Count == 0 {
		t.Fatalf("project health = %#v, want checks without error", health)
	}
	liveCoverTool("whodb_platform_project_health")

	_, model, err := HandlePlatformDataModelSummary(ctx, nil, PlatformResourceGraphInput{Fields: []string{"datasets", "ontologies", "relationships", "gaps"}})
	if err != nil {
		t.Fatalf("HandlePlatformDataModelSummary() error = %v", err)
	}
	if model.Error != "" {
		t.Fatalf("data model summary = %#v, want no error", model)
	}
	liveCoverTool("whodb_platform_data_model_summary")

	_, readiness, err := HandlePlatformRuntimeReadiness(ctx, nil, PlatformNextActionsInput{Fields: []string{"checks", "warnings", "functions", "transforms"}})
	if err != nil {
		t.Fatalf("HandlePlatformRuntimeReadiness() error = %v", err)
	}
	if readiness.Error != "" || readiness.Count == 0 {
		t.Fatalf("runtime readiness = %#v, want checks without error", readiness)
	}
	liveCoverTool("whodb_platform_runtime_readiness")

	_, impact, err := HandlePlatformChangeImpact(ctx, nil, PlatformChangeImpactInput{Resource: "dataset", ID: datasetID, Action: "update", Fields: []string{"target", "affected", "suggested_reads", "warnings"}})
	if err != nil {
		t.Fatalf("HandlePlatformChangeImpact() error = %v", err)
	}
	if impact.Error != "" {
		t.Fatalf("change impact = %#v, want no error", impact)
	}
	liveCoverTool("whodb_platform_change_impact")

	_, plan, err := HandlePlatformWritePlan(ctx, nil, PlatformWritePlanInput{
		Operation:   "update",
		Resource:    "dataset",
		ID:          datasetID,
		PayloadJSON: `{"description":"planned by MCP e2e"}`,
		Fields:      []string{"preview", "payload_keys", "suggested_reads", "warnings"},
	})
	if err != nil {
		t.Fatalf("HandlePlatformWritePlan() error = %v", err)
	}
	if plan.Error != "" {
		t.Fatalf("write plan = %#v, want no error", plan)
	}
	liveCoverTool("whodb_platform_write_plan")
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
	liveCoverTool("whodb_platform_orgs")
	_, projects, err := HandlePlatformProjects(ctx, nil, PlatformProjectsInput{Fields: []string{"id", "name", "slug"}})
	if err != nil {
		t.Fatalf("HandlePlatformProjects() error = %v", err)
	}
	if projects.Error != "" || projects.Count == 0 {
		t.Fatalf("projects output = %#v, want visible projects", projects)
	}
	liveCoverTool("whodb_platform_projects")
}

func liveMustProjectTools(t *testing.T, ctx context.Context, suffix string) {
	t.Helper()
	projectName := "mcp-e2e-project-" + suffix
	renamedProjectName := projectName + "-renamed"
	_, createOutput, err := HandlePlatformProjectCreate(ctx, nil, PlatformProjectCreateInput{Name: projectName, Description: "MCP e2e project"}, true)
	if err != nil {
		t.Fatalf("HandlePlatformProjectCreate() error = %v", err)
	}
	if createOutput.Error != "" || !createOutput.ConfirmationRequired {
		t.Fatalf("project create output = %#v, want pending confirmation", createOutput)
	}
	liveCoverTool("whodb_platform_project_create")
	liveMustReadPending(t, ctx)
	createConfirm := liveMustConfirm(t, ctx, createOutput.ConfirmationToken)
	projectID := liveConfirmResultID(t, createConfirm, "project create")

	_, renameOutput, err := HandlePlatformProjectRename(ctx, nil, PlatformProjectRenameInput{Project: projectID, Name: renamedProjectName}, true)
	if err != nil {
		t.Fatalf("HandlePlatformProjectRename() error = %v", err)
	}
	if renameOutput.Error != "" || !renameOutput.ConfirmationRequired {
		t.Fatalf("project rename output = %#v, want pending confirmation", renameOutput)
	}
	liveCoverTool("whodb_platform_project_rename")
	liveMustReadPending(t, ctx)
	_ = liveMustConfirm(t, ctx, renameOutput.ConfirmationToken)

	_, deleteOutput, err := HandlePlatformProjectDelete(ctx, nil, PlatformProjectDeleteInput{Project: projectID}, true)
	if err != nil {
		t.Fatalf("HandlePlatformProjectDelete() error = %v", err)
	}
	if deleteOutput.Error != "" || !deleteOutput.ConfirmationRequired {
		t.Fatalf("project delete output = %#v, want pending confirmation", deleteOutput)
	}
	liveCoverTool("whodb_platform_project_delete")
	liveMustReadPending(t, ctx)
	_ = liveMustConfirm(t, ctx, deleteOutput.ConfirmationToken)
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
	liveCoverTool("whodb_platform_source_types")
	_, fields, err := HandlePlatformSourceFields(ctx, nil, PlatformSourceFieldsInput{SourceType: "Postgres", Fields: []string{"key", "label", "required"}})
	if err != nil {
		t.Fatalf("HandlePlatformSourceFields() error = %v", err)
	}
	if fields.Error != "" || fields.Count == 0 {
		t.Fatalf("source fields output = %#v, want Postgres fields", fields)
	}
	liveCoverTool("whodb_platform_source_fields")
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
	liveCoverTool("whodb_platform_source_create")
	liveMustReadPending(t, ctx)
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
	liveCoverTool("whodb_platform_source_update")
	liveMustReadPending(t, ctx)
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
	liveCoverTool("whodb_platform_source_delete")
	liveMustReadPending(t, ctx)
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
			liveCoverTool("whodb_platform_sources")
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
	liveCoverTool("whodb_platform_source_objects")
	ref := "Table:whodb_platform.public.users"
	_, columns, err := HandlePlatformSourceColumns(ctx, nil, PlatformSourceColumnsInput{Source: source, Ref: ref})
	if err != nil {
		t.Fatalf("HandlePlatformSourceColumns() error = %v", err)
	}
	if columns.Error != "" || len(columns.Columns) == 0 {
		t.Fatalf("source columns output = %#v, want columns", columns)
	}
	liveCoverTool("whodb_platform_source_columns")
	_, rows, err := HandlePlatformSourceRows(ctx, nil, PlatformSourceRowsInput{Source: source, Ref: ref, Limit: 1}, &SecurityOptions{MaxRows: 10})
	if err != nil {
		t.Fatalf("HandlePlatformSourceRows() error = %v", err)
	}
	if rows.Error != "" {
		t.Fatalf("source rows output error = %q", rows.Error)
	}
	liveCoverTool("whodb_platform_source_rows")
	_, constraints, err := HandlePlatformSourceConstraints(ctx, nil, PlatformSourceConstraintsInput{Source: source, Ref: ref, Fields: []string{"data", "scope"}})
	if err != nil {
		t.Fatalf("HandlePlatformSourceConstraints() error = %v", err)
	}
	if constraints.Error != "" {
		t.Fatalf("source constraints output error = %q", constraints.Error)
	}
	liveCoverTool("whodb_platform_source_constraints")
	liveMustReadExpectedOutputError(t, ctx, "source content", func() (string, error) {
		_, content, err := HandlePlatformSourceContent(ctx, nil, PlatformSourceContentInput{Source: source, Ref: ref, Fields: []string{"data", "scope"}})
		return content.Error, err
	})
	liveCoverTool("whodb_platform_source_content")
	_, configOut, err := HandlePlatformSourceConfig(ctx, nil, PlatformSourceConfigInput{Source: source})
	if err != nil {
		t.Fatalf("HandlePlatformSourceConfig() error = %v", err)
	}
	if configOut.Error != "" || configOut.Config.Password != platformapi.RedactedValue() {
		t.Fatalf("source config output = %#v, want redacted password", configOut)
	}
	liveCoverTool("whodb_platform_source_config")
	_, testOut, err := HandlePlatformSourceTest(ctx, nil, PlatformSourceTestInput{Source: source})
	if err != nil {
		t.Fatalf("HandlePlatformSourceTest() error = %v", err)
	}
	if testOut.Error != "" || testOut.Status != "ok" {
		t.Fatalf("source test output = %#v, want ok", testOut)
	}
	liveCoverTool("whodb_platform_source_test")
}

func liveMustExerciseSourceObjectWrites(t *testing.T, ctx context.Context, sourceID, sourceName, dbPort, user, password, database, tableName string) {
	t.Helper()
	ref := "Table:" + database + ".public." + tableName
	liveMustGenericWrite(t, ctx, "platform_create", "create", PlatformGenericWriteInput{
		Resource: "source_object",
		PayloadJSON: liveJSON(t, map[string]any{
			"sourceId": sourceID,
			"parent":   map[string]any{"Kind": "Schema", "Path": []string{database, "public"}},
			"name":     tableName,
			"fields": []map[string]any{
				{
					"Key":   "id",
					"Value": "TEXT",
					"Extra": []map[string]any{
						{"Key": "nullable", "Value": "false"},
						{"Key": "primary", "Value": "true"},
					},
				},
				{
					"Key":   "name",
					"Value": "TEXT",
					"Extra": []map[string]any{{"Key": "nullable", "Value": "true"}},
				},
			},
		}),
	})
	liveSeedPostgresSourceObjectRow(t, ctx, dbPort, user, password, database, tableName, "1", "before")
	liveMustGenericWrite(t, ctx, "platform_update", "update", PlatformGenericWriteInput{
		Resource: "source_object",
		PayloadJSON: liveJSON(t, map[string]any{
			"sourceId":       sourceID,
			"ref":            map[string]any{"Kind": "Table", "Path": []string{database, "public", tableName}},
			"values":         []map[string]any{{"Key": "id", "Value": "1"}, {"Key": "name", "Value": "after"}},
			"updatedColumns": []string{"name"},
		}),
	})
	_, rows, err := HandlePlatformSourceRows(ctx, nil, PlatformSourceRowsInput{Source: sourceName, Ref: ref, Limit: 1}, &SecurityOptions{MaxRows: 10})
	if err != nil {
		t.Fatalf("HandlePlatformSourceRows(%s) error = %v", ref, err)
	}
	if rows.Error != "" || len(rows.Rows) == 0 || len(rows.Rows[0]) < 2 || rows.Rows[0][1] != "after" {
		t.Fatalf("source object updated rows = %#v, want name after", rows)
	}
	liveCoverTool("whodb_platform_source_rows")
	liveMustGenericWrite(t, ctx, "platform_delete", "delete", PlatformGenericWriteInput{
		Resource: "source_object",
		PayloadJSON: liveJSON(t, map[string]any{
			"sourceId": sourceID,
			"ref":      map[string]any{"Kind": "Table", "Path": []string{database, "public", tableName}},
			"values":   []map[string]any{{"Key": "id", "Value": "1"}},
		}),
	})
	_, rows, err = HandlePlatformSourceRows(ctx, nil, PlatformSourceRowsInput{Source: sourceName, Ref: ref, Limit: 1}, &SecurityOptions{MaxRows: 10})
	if err != nil {
		t.Fatalf("HandlePlatformSourceRows(%s) after delete error = %v", ref, err)
	}
	if rows.Error != "" || rows.Total != 0 {
		t.Fatalf("source object rows after delete = %#v, want empty table", rows)
	}
	liveCoverTool("whodb_platform_source_rows")
}

func liveSeedPostgresSourceObjectRow(t *testing.T, ctx context.Context, port, user, password, database, tableName, id, name string) {
	t.Helper()
	dsn := fmt.Sprintf("postgres://%s:%s@localhost:%s/%s?sslmode=disable", url.QueryEscape(user), url.QueryEscape(password), port, url.PathEscape(database))
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open Postgres fixture connection: %v", err)
	}
	defer db.Close()
	query := fmt.Sprintf("INSERT INTO public.%s (id, name) VALUES ($1, $2)", liveQuotePostgresIdentifier(tableName))
	if _, err := db.ExecContext(ctx, query, id, name); err != nil {
		t.Fatalf("seed source object row: %v", err)
	}
}

func liveQuotePostgresIdentifier(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
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
	liveCoverTool("whodb_platform_files")
	_, search, err := HandlePlatformFileSearch(ctx, nil, PlatformFileSearchInput{Query: "mcp-e2e", Fields: []string{"id", "name", "isTabular"}})
	if err != nil {
		t.Fatalf("HandlePlatformFileSearch() error = %v", err)
	}
	if search.Error != "" {
		t.Fatalf("file search error = %q", search.Error)
	}
	liveCoverTool("whodb_platform_file_search")
	_, tabular, err := HandlePlatformTabularFiles(ctx, nil, PlatformEmptyInput{Fields: []string{"id", "name", "isTabular"}})
	if err != nil {
		t.Fatalf("HandlePlatformTabularFiles() error = %v", err)
	}
	if tabular.Error != "" {
		t.Fatalf("tabular files error = %q", tabular.Error)
	}
	liveCoverTool("whodb_platform_tabular_files")
	_, usage, err := HandlePlatformStorageUsage(ctx, nil, PlatformEmptyInput{})
	if err != nil {
		t.Fatalf("HandlePlatformStorageUsage() error = %v", err)
	}
	if usage.Error != "" {
		t.Fatalf("storage usage error = %q", usage.Error)
	}
	liveCoverTool("whodb_platform_storage_usage")
}

func liveMustReadPending(t *testing.T, ctx context.Context) {
	t.Helper()
	_, pending, err := HandlePlatformPending(ctx, nil, PlatformPendingInput{})
	if err != nil {
		t.Fatalf("HandlePlatformPending() error = %v", err)
	}
	if pending.Error != "" {
		t.Fatalf("pending output error = %q", pending.Error)
	}
	if len(pending.Pending) == 0 {
		t.Fatalf("pending output = %#v, want at least one pending confirmation", pending)
	}
	liveCoverTool("whodb_platform_pending")
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

func liveMustReadExpectedOutputError(t *testing.T, ctx context.Context, name string, read func() (string, error)) {
	t.Helper()
	outputErr, err := read()
	if err != nil {
		t.Fatalf("read %s expected output error, got handler error = %v", name, err)
	}
	if strings.TrimSpace(outputErr) == "" {
		t.Fatalf("read %s expected non-empty output error", name)
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
	liveCoverTool("whodb_" + toolName)
	liveCoverGenericWrite(operationKind, input)
	liveMustReadPending(t, ctx)
	confirm := liveMustConfirm(t, ctx, output.ConfirmationToken)
	return liveConfirmColumn(t, confirm, "result_json")
}

func liveMustClone(t *testing.T, ctx context.Context, input PlatformCloneInput) string {
	t.Helper()
	_, output, err := HandlePlatformClone(ctx, nil, input, true)
	if err != nil {
		t.Fatalf("HandlePlatformClone(%s) error = %v", input.Resource, err)
	}
	if output.Error != "" || !output.ConfirmationRequired {
		t.Fatalf("clone output for %s = %#v, want pending confirmation", input.Resource, output)
	}
	liveCoverTool("whodb_platform_clone")
	liveCoverGenericWrite("create", PlatformGenericWriteInput{Resource: input.Resource})
	liveMustReadPending(t, ctx)
	confirm := liveMustConfirm(t, ctx, output.ConfirmationToken)
	result := liveConfirmColumn(t, confirm, "result_json")
	var decoded struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(result), &decoded); err != nil {
		t.Fatalf("decode clone result JSON: %v\n%s", err, result)
	}
	if decoded.ID == "" {
		t.Fatalf("clone result JSON did not include id: %s", result)
	}
	return decoded.ID
}

func liveMustGenericWriteConfirmError(t *testing.T, ctx context.Context, toolName, operationKind string, input PlatformGenericWriteInput) {
	t.Helper()
	_, output, err := handlePlatformGenericWrite(ctx, toolName, input, operationKind, true)
	if err != nil {
		t.Fatalf("handlePlatformGenericWrite(%s, %s/%s) error = %v", toolName, input.Resource, input.Action, err)
	}
	if output.Error != "" || !output.ConfirmationRequired {
		t.Fatalf("write output for %s %s/%s = %#v, want pending confirmation", toolName, input.Resource, input.Action, output)
	}
	liveCoverTool("whodb_" + toolName)
	liveCoverGenericWrite(operationKind, input)
	liveMustReadPending(t, ctx)
	_, confirm, err := HandlePlatformConfirm(ctx, nil, ConfirmInput{Token: output.ConfirmationToken})
	if err != nil {
		t.Fatalf("HandlePlatformConfirm() expected output error, got handler error = %v", err)
	}
	if strings.TrimSpace(confirm.Error) == "" {
		t.Fatalf("confirm output for %s %s/%s = %#v, want platform error", toolName, input.Resource, input.Action, confirm)
	}
	liveCoverTool("whodb_platform_confirm")
}

func liveMustTypedWriteConfirmation(t *testing.T, ctx context.Context, toolName string, run func() (PlatformGenericWriteOutput, error)) {
	t.Helper()
	output, err := run()
	if err != nil {
		t.Fatalf("%s error = %v", toolName, err)
	}
	if output.Error != "" {
		t.Fatalf("%s output error = %q", toolName, output.Error)
	}
	if !output.ConfirmationRequired || output.ConfirmationToken == "" {
		t.Fatalf("%s output = %#v, want confirmation token", toolName, output)
	}
	liveCoverTool(toolName)
	liveMustReadPending(t, ctx)
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
	liveCoverTool("whodb_platform_confirm")
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

func liveConfirmResultID(t *testing.T, output ConfirmOutput, label string) string {
	t.Helper()
	result := liveConfirmColumn(t, output, "result_json")
	var decoded struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(result), &decoded); err != nil {
		t.Fatalf("decode %s result JSON: %v\n%s", label, err, result)
	}
	if decoded.ID == "" {
		t.Fatalf("%s result JSON did not include id: %s", label, result)
	}
	return decoded.ID
}

func liveConfirmRowColumn(t *testing.T, output ConfirmOutput, resource, name, column string) string {
	t.Helper()
	resourceIndex := -1
	nameIndex := -1
	columnIndex := -1
	for i, columnName := range output.Columns {
		switch columnName {
		case "resource":
			resourceIndex = i
		case "name":
			nameIndex = i
		case column:
			columnIndex = i
		}
	}
	if resourceIndex < 0 || nameIndex < 0 || columnIndex < 0 {
		t.Fatalf("confirm columns = %#v, missing resource/name/%s", output.Columns, column)
	}
	for _, row := range output.Rows {
		if len(row) <= resourceIndex || len(row) <= nameIndex || len(row) <= columnIndex {
			continue
		}
		rowResource, _ := row[resourceIndex].(string)
		rowName, _ := row[nameIndex].(string)
		if rowResource != resource || rowName != name {
			continue
		}
		value, _ := row[columnIndex].(string)
		if strings.TrimSpace(value) == "" {
			t.Fatalf("confirm row for %s %s has empty %s: %#v", resource, name, column, row)
		}
		return value
	}
	t.Fatalf("confirm rows = %#v, missing %s %s", output.Rows, resource, name)
	return ""
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
