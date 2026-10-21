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
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/clidey/whodb/cli/e2e/testharness"
	"github.com/clidey/whodb/cli/internal/config"
	platformapi "github.com/clidey/whodb/cli/internal/platform"
)

func TestPlatformMCP_RealQueryCompleteness(t *testing.T) {
	cleanup := testharness.SetupCleanEnv(t)
	defer cleanup()
	t.Setenv("WHODB_CLI_E2E_PLATFORM_TOKEN_DIR", filepath.Join(t.TempDir(), "tokens"))
	config.ResetPathsForTesting()
	clearPendingPlatformActions(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	host := liveEnvOrDefault("WHODB_PLATFORM_E2E_HOST", defaultLivePlatformE2EHost)
	refreshToken := liveMintDevRefreshToken(t, ctx,
		liveEnvOrDefault("WHODB_PLATFORM_E2E_KEYCLOAK_URL", defaultLivePlatformE2EKeycloakURL),
		liveEnvOrDefault("WHODB_PLATFORM_E2E_KEYCLOAK_HOST_HEADER", "127.0.0.1:4001"),
		liveEnvOrDefault("WHODB_PLATFORM_E2E_USER", defaultLivePlatformE2EUser),
		liveEnvOrDefault("WHODB_PLATFORM_E2E_PASSWORD", defaultLivePlatformE2EPassword))
	liveSeedPlatformLogin(t, ctx, host, refreshToken)
	entityName := "mcp_completeness_" + time.Now().UTC().Format("20060102150405")
	properties := []map[string]any{}
	for _, name := range []string{"id", "target_id"} {
		properties = append(properties, map[string]any{
			"apiName": name, "displayName": name, "description": "", "columnName": name,
			"dataType": "String", "isRequired": true, "visibility": "normal",
			"isSearchable": false, "isSortable": true, "isEditOnly": false,
		})
	}
	entityID := liveMustGenericWriteID(t, ctx, "platform_create", "create", PlatformGenericWriteInput{
		Resource: "ontology", PayloadJSON: liveJSON(t, map[string]any{
			"apiName": entityName, "displayName": entityName, "pluralDisplayName": entityName,
			"description": "Query completeness regression", "primaryKey": "id",
			"tableName": entityName, "schemaName": "", "storageMode": "operational",
			"icon": "database", "color": "blue", "properties": properties, "links": []any{},
		}),
	})
	defer liveBestEffortGenericDelete(ctx, "ontology", entityID)
	rows := []map[string]string{
		{"id": "22222222-2222-4222-8222-000000000001", "target_id": "t-1"},
		{"id": "22222222-2222-4222-8222-000000000002", "target_id": "t-2"},
		{"id": "22222222-2222-4222-8222-000000000003", "target_id": "t-2"},
	}
	_, add, err := handlePlatformOntologyRecordsWrite(ctx, PlatformOntologyRecordsInput{EntityID: entityID, Rows: rows}, true)
	if err != nil || add.Error != "" || !add.ConfirmationRequired {
		t.Fatalf("add rows: %v %#v", err, add)
	}
	liveMustConfirm(t, ctx, add.ConfirmationToken)

	// Completeness must survive aggregation, empty filters and output projection
	// across the real GraphQL server, not just the MCP client fixtures.
	for _, tc := range []struct {
		name    string
		input   PlatformOntologyQueryInput
		partial bool
	}{
		{"grouped_partial", PlatformOntologyQueryInput{GroupBy: []string{"target_id"}, Metrics: []PlatformOntologyMetricInput{{Op: "count", As: "n"}}, ScanLimit: 2}, true},
		{"empty_partial", PlatformOntologyQueryInput{Where: map[string]any{"id": "00000000-0000-4000-8000-000000000000"}, ScanLimit: 2}, true},
		{"exact_scan_boundary", PlatformOntologyQueryInput{ScanLimit: 3}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.input.Entity = entityName
			_, out, err := HandlePlatformOntologyQuery(ctx, nil, tc.input, nil)
			if err != nil || out.Error != "" {
				t.Fatalf("ontology query: %v %#v", err, out)
			}
			var data platformapi.DatasetQueryResult
			liveDecodeData(t, out.Data, &data)
			if data.Partial == nil || *data.Partial != tc.partial || data.ScannedRows == nil || *data.ScannedRows != 3 {
				t.Fatalf("completeness = %#v; want partial=%v, scannedRows=3", data, tc.partial)
			}
			if out.Truncated != tc.partial || (len(out.Warnings) > 0) != tc.partial {
				t.Fatalf("MCP completeness = %#v; want partial=%v", out, tc.partial)
			}
			tc.input.Fields = []string{"rows"}
			_, projected, err := HandlePlatformOntologyQuery(ctx, nil, tc.input, nil)
			if err != nil || projected.Error != "" || projected.Truncated != tc.partial || (len(projected.Warnings) > 0) != tc.partial {
				t.Fatalf("projected completeness = %v %#v; want partial=%v", err, projected, tc.partial)
			}
		})
	}
}
