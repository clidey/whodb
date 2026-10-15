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
	"encoding/json"
	"strings"
	"testing"

	platformapi "github.com/clidey/whodb/cli/internal/platform"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// liveGapFixtures are the resources created earlier in the lifecycle test that
// the capability-gap coverage reuses.
type liveGapFixtures struct {
	suffix           string
	ontologyID       string
	ontologyAPIName  string
	targetOntologyID string
	functionID       string
	transformID      string
	datasetID        string
	appID            string
	packageID        string
	fileID           string
	folderID         string
}

// liveMustGapCoverage exercises the ontology analytics, bulk insert, run
// cancellation, restore/purge, sharing, lineage provider, service account,
// team and package-polish capabilities against the real platform.
func liveMustGapCoverage(t *testing.T, ctx context.Context, fx liveGapFixtures) {
	t.Helper()
	session, err := loadPlatformWorkspace(ctx)
	if err != nil {
		t.Fatalf("load workspace for gap coverage: %v", err)
	}
	projectID := session.Host.DefaultProjectID
	orgID := session.Host.DefaultOrgID

	// The fixture behavior only defines "approve"; inserts require a "create"
	// action, so extend the behavior document first (revision 1 follows the
	// initial save).
	liveMustGenericWrite(t, ctx, "platform_action", "action", PlatformGenericWriteInput{
		Resource: "ontology", Action: "save_behavior", ID: fx.ontologyID,
		PayloadJSON: liveJSON(t, map[string]any{
			"expectedRevision": 1,
			"document": map[string]any{
				"api_name": fx.ontologyAPIName, "ontology_id": fx.ontologyID,
				"actions": map[string]any{
					"approve": map[string]any{},
					"create":  map[string]any{"input": map[string]any{"id": map[string]any{"type": "string"}, "target_id": map[string]any{"type": "string"}}},
				},
			},
		}),
	})

	// Bulk insert three real rows, then answer questions about them.
	// Operational ontologies store the primary key as a UUID.
	const gapRow1, gapRow2, gapRow3 = "22222222-2222-4222-8222-000000000001", "22222222-2222-4222-8222-000000000002", "22222222-2222-4222-8222-000000000003"
	rows := []map[string]string{{"id": gapRow1, "target_id": "t-1"}, {"id": gapRow2, "target_id": "t-2"}, {"id": gapRow3, "target_id": "t-2"}}
	_, addOutput, err := handlePlatformOntologyRecordsWrite(ctx, PlatformOntologyRecordsInput{EntityID: fx.ontologyID, Rows: rows, IdempotencyKey: "gap-import-" + fx.suffix}, true)
	if err != nil || addOutput.Error != "" || !addOutput.ConfirmationRequired {
		t.Fatalf("add_ontology_records preview = %v %#v", err, addOutput)
	}
	liveCoverTool("whodb_platform_add_ontology_records")
	liveCoverGenericWrite("action", PlatformGenericWriteInput{Resource: "ontology", Action: "add_records"})
	addConfirm := liveMustConfirm(t, ctx, addOutput.ConfirmationToken)
	var inserted struct {
		Inserted int      `json:"inserted"`
		IDs      []string `json:"ids"`
	}
	if err := json.Unmarshal([]byte(liveConfirmColumn(t, addConfirm, "result_json")), &inserted); err != nil || inserted.Inserted != 3 {
		t.Fatalf("add_ontology_records result = %#v, %v; want 3 inserted", inserted, err)
	}

	_, describe, err := HandlePlatformOntologyDescribe(ctx, nil, PlatformOntologyDescribeInput{Entities: []string{fx.ontologyAPIName}, Fields: []string{"data", "count"}})
	if err != nil || describe.Error != "" || describe.Count != 1 {
		t.Fatalf("ontology_describe = %v %#v", err, describe)
	}
	liveCoverTool("whodb_platform_ontology_describe")
	_, describeAll, err := HandlePlatformOntologyDescribe(ctx, nil, PlatformOntologyDescribeInput{Fields: []string{"count"}})
	if err != nil || describeAll.Error != "" || describeAll.Count < 2 {
		t.Fatalf("ontology_describe (all) = %v %#v", err, describeAll)
	}

	_, query, err := HandlePlatformOntologyQuery(ctx, nil, PlatformOntologyQueryInput{
		Entity: fx.ontologyAPIName,
		Where:  map[string]any{"id": map[string]any{"neq": gapRow2}},
		Sort:   []PlatformOntologyQuerySortInput{{Field: "id"}},
		Limit:  10,
		Fields: []string{"data", "count"},
	}, &SecurityOptions{MaxRows: 100})
	if err != nil || query.Error != "" || query.Count != 2 {
		t.Fatalf("ontology_query = %v %#v; want 2 rows", err, query)
	}
	liveCoverTool("whodb_platform_ontology_query")
	_, grouped, err := HandlePlatformOntologyQuery(ctx, nil, PlatformOntologyQueryInput{
		Entity:  fx.ontologyAPIName,
		GroupBy: []string{"target_id"},
		Metrics: []PlatformOntologyMetricInput{{Op: "count", As: "n"}},
		Fields:  []string{"data", "count"},
	}, nil)
	if err != nil || grouped.Error != "" || grouped.Count != 2 {
		t.Fatalf("ontology_query grouped = %v %#v; want two groups", err, grouped)
	}

	_, aggregate, err := HandlePlatformOntologyAggregate(ctx, nil, PlatformOntologyAggregateInput{
		ID:      fx.ontologyID,
		Metrics: []PlatformOntologyMetricInput{{Op: "COUNT", As: "total"}},
		Fields:  []string{"columns", "rows", "total"},
	}, nil)
	if err != nil || aggregate.Error != "" || aggregate.Count != 1 {
		t.Fatalf("ontology_aggregate = %v %#v", err, aggregate)
	}
	liveCoverTool("whodb_platform_ontology_aggregate")
	var aggregateRows platformapi.DatasetQueryResult
	liveDecodeData(t, aggregate.Data, &aggregateRows)
	if len(aggregateRows.Rows) != 1 || aggregateRows.Rows[0][0] != "3" {
		t.Fatalf("ontology_aggregate rows = %#v; want COUNT 3", aggregate.Data)
	}
	_, groupedAggregate, err := HandlePlatformOntologyAggregate(ctx, nil, PlatformOntologyAggregateInput{
		ID:      fx.ontologyID,
		GroupBy: []string{"id"},
		Metrics: []PlatformOntologyMetricInput{{Op: "COUNT"}},
		Where:   map[string]any{"id": gapRow1},
		Sort:    []PlatformSortConditionInput{{Column: "id", Direction: "DESC"}},
		Limit:   5,
		Fields:  []string{"data", "count"},
	}, nil)
	if err != nil || groupedAggregate.Error != "" || groupedAggregate.Count != 1 {
		t.Fatalf("ontology_aggregate grouped = %v %#v; want one filtered group", err, groupedAggregate)
	}

	_, stats, err := HandlePlatformOntologyStats(ctx, nil, PlatformOntologyStatsInput{ID: fx.ontologyID, Property: "target_id"})
	if err != nil || stats.Error != "" {
		t.Fatalf("ontology_stats = %v %#v", err, stats)
	}
	liveCoverTool("whodb_platform_ontology_stats")
	var statsResult platformapi.OntologyStatsResult
	liveDecodeData(t, stats.Data, &statsResult)
	if statsResult.Total != 3 || statsResult.NonNull != 3 || statsResult.Min == nil || *statsResult.Min != "t-1" {
		t.Fatalf("ontology_stats result = %#v; want total 3", stats.Data)
	}

	_, capabilities, err := HandlePlatformOntologyRecordCapabilities(ctx, nil, PlatformOntologyRecordCapabilitiesInput{OntologyID: fx.ontologyID, RecordKey: gapRow1, Fields: []string{"data"}})
	if err != nil || capabilities.Error != "" {
		t.Fatalf("ontology_record_capabilities = %v %#v", err, capabilities)
	}
	liveCoverTool("whodb_platform_ontology_record_capabilities")

	_, preview, err := HandlePlatformOntologyActionPreview(ctx, nil, PlatformOntologyActionPreviewInput{OntologyID: fx.ontologyID, RecordKey: gapRow1, Action: "approve", Values: map[string]any{}, Fields: []string{"data"}})
	if err != nil || preview.Error != "" {
		t.Fatalf("ontology_action_preview = %v %#v", err, preview)
	}
	liveCoverTool("whodb_platform_ontology_action_preview")
	liveMustGenericWriteAttempt(t, ctx, "platform_action", "action", PlatformGenericWriteInput{
		Resource: "ontology", Action: "execute_behavior", ID: fx.ontologyID,
		PayloadJSON: liveJSON(t, map[string]any{"recordKey": gapRow1, "action": "approve", "values": map[string]any{}}),
	})
	_, executions, err := HandlePlatformOntologyActionExecutions(ctx, nil, PlatformOntologyActionExecutionsInput{OntologyID: fx.ontologyID, RecordKey: gapRow1, Limit: 5, Fields: []string{"items", "count"}})
	if err != nil || executions.Error != "" {
		t.Fatalf("ontology_action_executions = %v %#v", err, executions)
	}
	liveCoverTool("whodb_platform_ontology_action_executions")

	_, jobs, err := HandlePlatformOntologyEmbeddingJobs(ctx, nil, PlatformEntityInput{ID: fx.ontologyID, Fields: []string{"items", "count"}})
	if err != nil || jobs.Error != "" || jobs.Count != 0 {
		t.Fatalf("ontology_embedding_jobs = %v %#v; want no jobs", err, jobs)
	}
	liveCoverTool("whodb_platform_ontology_embedding_jobs")
	// Embeddings are disabled on the fixture ontology, so similarity and
	// re-embedding must fail with the platform's explanation rather than panic.
	liveMustReadExpectedOutputError(t, ctx, "ontology_similar without embeddings", func() (string, error) {
		_, out, err := HandlePlatformOntologySimilar(ctx, nil, PlatformOntologySimilarInput{EntityID: fx.ontologyID, RowID: gapRow1, TopK: 2})
		return out.Error, err
	})
	liveCoverTool("whodb_platform_ontology_similar")
	liveMustGenericWriteConfirmError(t, ctx, "platform_action", "action", PlatformGenericWriteInput{Resource: "ontology", Action: "reembed", ID: fx.ontologyID, PayloadJSON: `{"batchSize":2}`})

	// Transform preview runs an inline seed graph on the local compute backend
	// and returns its rows without saving a transform.
	seedGraph := `{"nodes":[{"id":"seed1","type":"seed","config":{"seedColumns":[{"name":"name","type":"text"},{"name":"amount","type":"text"}],"seedRows":[["a","1"],["b","2"],["c","3"]]}}],"edges":[]}`
	_, transformPreview, err := HandlePlatformTransformPreview(ctx, nil, PlatformTransformPreviewInput{GraphJSON: seedGraph, Fields: []string{"columns", "rows", "total"}}, &SecurityOptions{MaxRows: 10})
	if err != nil || transformPreview.Error != "" || transformPreview.Count != 3 {
		t.Fatalf("transform_preview = %v %#v; want 3 seed rows", err, transformPreview)
	}
	_, badPreview, err := HandlePlatformTransformPreview(ctx, nil, PlatformTransformPreviewInput{GraphJSON: `{"nodes":[],"edges":[]}`}, nil)
	if err != nil || !strings.Contains(badPreview.Error, "no source nodes") {
		t.Fatalf("transform_preview without sources = %v %#v; want the platform error", err, badPreview)
	}
	liveCoverTool("whodb_platform_transform_preview")

	// Run control: cancel outcomes depend on how fast the run finishes, so only
	// the plumbing is asserted, not the final status.
	if liveFunctionRuntimeAvailable {
		runID := liveMustGenericWriteID(t, ctx, "platform_action", "action", PlatformGenericWriteInput{
			Resource: "function", Action: "start_run", PayloadJSON: liveJSON(t, map[string]any{"functionId": fx.functionID, "input": "{}"}),
		})
		liveMustGenericWriteAttempt(t, ctx, "platform_action", "action", PlatformGenericWriteInput{Resource: "function_run", Action: "cancel", ID: runID})
	} else {
		t.Logf("SKIPPED function_run cancel against a real run: no container runtime; cancelling a missing run must still fail cleanly")
		liveMustGenericWriteConfirmError(t, ctx, "platform_action", "action", PlatformGenericWriteInput{Resource: "function_run", Action: "cancel", ID: "00000000-0000-0000-0000-000000000000"})
	}
	liveMustGenericWriteAttempt(t, ctx, "platform_action", "action", PlatformGenericWriteInput{Resource: "transform", Action: "cancel_run", ID: fx.transformID, PayloadJSON: `{}`})
	liveMustGenericWriteAttempt(t, ctx, "platform_action", "action", PlatformGenericWriteInput{Resource: "app", Action: "cancel_generation", ID: fx.appID})
	liveMustGenericWriteConfirmError(t, ctx, "platform_action", "action", PlatformGenericWriteInput{
		Resource: "function", Action: "generate_test_input", ID: fx.functionID,
		PayloadJSON: liveJSON(t, map[string]any{"ModelType": "OpenAI", "Model": "gpt-4o-mini", "Token": "invalid"}),
	})

	// Sharing: share from the selected project, reference it from the seeded
	// sandbox project via per-call workspace targeting, then tear down.
	liveMustGenericWrite(t, ctx, "platform_action", "action", PlatformGenericWriteInput{Resource: "ontology", Action: "share", ID: fx.ontologyID, PayloadJSON: `{"visibility":"organization"}`})
	liveMustExtendedRead(t, ctx, "whodb_platform_shared_ontologies", "SharedOntologies", nil, true)
	sandboxCtx, _ := platformRequestContext(ctx, nil, &PlatformWorkspaceTarget{Host: session.Host.URL, Org: orgID, Project: "sandbox"})
	liveMustExtendedRead(t, sandboxCtx, "whodb_platform_discover_shared_ontologies", "DiscoverSharedOntologies", nil, true)
	referenceID := liveMustGenericWriteID(t, sandboxCtx, "platform_create", "create", PlatformGenericWriteInput{
		Resource:    "ontology_reference",
		PayloadJSON: liveJSON(t, map[string]any{"ownerProjectId": projectID, "ontologyId": fx.ontologyID, "alias": "shared_gap_" + fx.suffix}),
	})
	liveMustExtendedRead(t, sandboxCtx, "whodb_platform_ontology_references", "OntologyReferences", nil, true)
	liveMustExtendedRead(t, sandboxCtx, "whodb_platform_ontology_reference_impact", "OntologyReferenceImpact", map[string]any{"referenceId": referenceID}, true)
	liveMustGenericWrite(t, sandboxCtx, "platform_delete", "delete", PlatformGenericWriteInput{Resource: "ontology_reference", ID: referenceID})
	liveMustGenericWrite(t, ctx, "platform_action", "action", PlatformGenericWriteInput{Resource: "ontology", Action: "unshare", ID: fx.ontologyID})

	// Lineage providers.
	providerID := liveMustGenericWriteID(t, ctx, "platform_create", "create", PlatformGenericWriteInput{
		Resource:    "lineage_provider",
		PayloadJSON: liveJSON(t, map[string]any{"name": "gap-marquez-" + fx.suffix, "endpoint": "http://127.0.0.1:1/api/v1/lineage", "readEndpoint": "http://127.0.0.1:1/api/v1", "namespace": "whodb-e2e", "authType": "none", "enabled": false}),
	})
	liveMustGenericWrite(t, ctx, "platform_update", "update", PlatformGenericWriteInput{
		Resource: "lineage_provider", ID: providerID,
		PayloadJSON: liveJSON(t, map[string]any{"name": "gap-marquez-renamed-" + fx.suffix, "endpoint": "http://127.0.0.1:1/api/v1/lineage", "readEndpoint": "http://127.0.0.1:1/api/v1", "namespace": "whodb-e2e", "authType": "none", "enabled": false}),
	})
	liveMustExtendedRead(t, ctx, "whodb_platform_lineage_providers", "ProjectLineageProviders", nil, true)
	liveMustGenericWrite(t, ctx, "platform_delete", "delete", PlatformGenericWriteInput{Resource: "lineage_provider", ID: providerID})

	// Service accounts: the raw key must come back exactly once.
	serviceAccountID := liveMustGenericWriteID(t, ctx, "platform_action", "action", PlatformGenericWriteInput{
		Resource: "organization", Action: "create_service_account",
		PayloadJSON: liveJSON(t, map[string]any{"name": "gap-sa-" + fx.suffix, "description": "MCP e2e service account"}),
	})
	keyResult := liveMustGenericWrite(t, ctx, "platform_action", "action", PlatformGenericWriteInput{
		Resource: "organization", Action: "create_service_account_key", ID: serviceAccountID,
	})
	var createdKey struct {
		Key    struct{ ID string } `json:"key"`
		RawKey string              `json:"rawKey"`
	}
	if err := json.Unmarshal([]byte(keyResult), &createdKey); err != nil || createdKey.RawKey == "" || createdKey.Key.ID == "" {
		t.Fatalf("create_service_account_key result = %s (%v); want rawKey and key id", keyResult, err)
	}
	liveMustExtendedRead(t, ctx, "whodb_platform_service_accounts", "ServiceAccounts", nil, true)
	liveMustGenericWrite(t, ctx, "platform_action", "action", PlatformGenericWriteInput{Resource: "organization", Action: "revoke_service_account_key", ID: createdKey.Key.ID})
	liveMustGenericWrite(t, ctx, "platform_action", "action", PlatformGenericWriteInput{Resource: "organization", Action: "delete_service_account", ID: serviceAccountID})

	// Teams: create, rename, delete now that the manifest advertises them.
	teamID := liveMustGenericWriteID(t, ctx, "platform_action", "action", PlatformGenericWriteInput{
		Resource: "organization", Action: "create_team", PayloadJSON: liveJSON(t, map[string]any{"name": "gap-team-" + fx.suffix}),
	})
	liveMustGenericWrite(t, ctx, "platform_action", "action", PlatformGenericWriteInput{Resource: "organization", Action: "rename_team", ID: teamID, PayloadJSON: `{"name":"gap-team-renamed"}`})
	liveMustGenericWrite(t, ctx, "platform_action", "action", PlatformGenericWriteInput{Resource: "organization", Action: "delete_team", ID: teamID})

	// Member administration and sharing dependencies for the signed-in owner.
	me, err := session.Client.Me(ctx)
	if err != nil {
		t.Fatalf("Me(): %v", err)
	}
	liveMustExtendedRead(t, ctx, "whodb_platform_user_owned_resources", "UserOwnedResources", map[string]any{"userId": me.ID}, true)
	liveMustExtendedRead(t, ctx, "whodb_platform_user_resource_access", "UserResourceAccess", map[string]any{"userId": me.ID}, true)
	liveMustExtendedRead(t, ctx, "whodb_platform_share_dependents", "ShareDependents", map[string]any{"resourceType": "dataset", "resourceId": fx.datasetID}, true)

	// Package polish and app page files.
	liveMustGenericWriteAttempt(t, ctx, "platform_action", "action", PlatformGenericWriteInput{Resource: "package", Action: "add_changelog", ID: fx.packageID, PayloadJSON: `{"version":"1.0.0","title":"Initial release","body":"MCP e2e"}`})
	liveMustGenericWriteAttempt(t, ctx, "platform_action", "action", PlatformGenericWriteInput{Resource: "package", Action: "add_media", ID: fx.packageID, PayloadJSON: `{"kind":"screenshot","url":"https://example.com/shot.png","caption":"MCP e2e"}`})
	liveMustGenericWriteAttempt(t, ctx, "platform_action", "action", PlatformGenericWriteInput{Resource: "package", Action: "review", ID: fx.packageID, PayloadJSON: `{"rating":5,"body":"MCP e2e review"}`})
	liveMustExtendedRead(t, ctx, "whodb_platform_org_package_installations", "OrganizationPackageInstallations", map[string]any{"packageId": fx.packageID}, true)
	liveMustExtendedRead(t, ctx, "whodb_platform_app_page_files", "AppPageFiles", map[string]any{"id": fx.appID, "page": "index", "env": optionalPlatformString("")}, false)

	// Data purges run last. The platform refuses to purge an ontology with an
	// active behavior, so that rule is asserted on the fixture entity and the
	// purge itself is proven on the behavior-free target ontology.
	liveMustGenericWriteConfirmError(t, ctx, "platform_action", "action", PlatformGenericWriteInput{Resource: "ontology", Action: "purge_data", ID: fx.ontologyID})
	_, targetAdd, err := handlePlatformOntologyRecordsWrite(ctx, PlatformOntologyRecordsInput{EntityID: fx.targetOntologyID, Rows: []map[string]string{{"id": "33333333-3333-4333-8333-000000000001"}}}, true)
	if err != nil || targetAdd.Error != "" || !targetAdd.ConfirmationRequired {
		t.Fatalf("add_ontology_records (target) preview = %v %#v", err, targetAdd)
	}
	liveMustConfirm(t, ctx, targetAdd.ConfirmationToken)
	liveMustGenericWrite(t, ctx, "platform_action", "action", PlatformGenericWriteInput{Resource: "ontology", Action: "purge_data", ID: fx.targetOntologyID})
	_, emptied, err := HandlePlatformOntologyAggregate(ctx, nil, PlatformOntologyAggregateInput{ID: fx.targetOntologyID, Metrics: []PlatformOntologyMetricInput{{Op: "COUNT"}}}, nil)
	if err != nil || emptied.Error != "" {
		t.Fatalf("aggregate after purge_data = %v %#v", err, emptied)
	}
	var emptiedRows platformapi.DatasetQueryResult
	liveDecodeData(t, emptied.Data, &emptiedRows)
	if len(emptiedRows.Rows) != 1 || emptiedRows.Rows[0][0] != "0" {
		t.Fatalf("aggregate after purge_data rows = %#v; want COUNT 0", emptied.Data)
	}
	liveMustGenericWriteAttempt(t, ctx, "platform_action", "action", PlatformGenericWriteInput{Resource: "dataset", Action: "purge_data", ID: fx.datasetID})
}

// liveMustGapRestorePurge deletes, restores, deletes again and purges a file
// and a folder so the soft-delete lifecycle is proven for both.
func liveMustGapRestorePurge(t *testing.T, ctx context.Context, fileID, folderID string) {
	t.Helper()
	liveMustGenericWrite(t, ctx, "platform_delete", "delete", PlatformGenericWriteInput{Resource: "file", ID: fileID})
	liveMustRestore(t, ctx, "file", fileID)
	liveMustGenericWrite(t, ctx, "platform_delete", "delete", PlatformGenericWriteInput{Resource: "file", ID: fileID})
	liveMustGenericWrite(t, ctx, "platform_action", "action", PlatformGenericWriteInput{Resource: "file", Action: "purge", ID: fileID})

	liveMustGenericWrite(t, ctx, "platform_delete", "delete", PlatformGenericWriteInput{Resource: "folder", ID: folderID})
	liveMustRestore(t, ctx, "folder", folderID)
	liveMustGenericWrite(t, ctx, "platform_delete", "delete", PlatformGenericWriteInput{Resource: "folder", ID: folderID})
	liveMustGenericWrite(t, ctx, "platform_action", "action", PlatformGenericWriteInput{Resource: "folder", Action: "purge", ID: folderID})
}

// liveMustGapProjectRestorePurge restores a soft-deleted project, deletes it
// again and purges it permanently.
func liveMustGapProjectRestorePurge(t *testing.T, ctx context.Context, projectID string) {
	t.Helper()
	liveMustRestore(t, ctx, "project", projectID)
	_, deleteOutput, err := HandlePlatformProjectDelete(ctx, nil, PlatformProjectDeleteInput{Project: projectID}, true)
	if err != nil || deleteOutput.Error != "" || !deleteOutput.ConfirmationRequired {
		t.Fatalf("project delete after restore = %v %#v", err, deleteOutput)
	}
	_ = liveMustConfirm(t, ctx, deleteOutput.ConfirmationToken)
	liveMustGenericWrite(t, ctx, "platform_action", "action", PlatformGenericWriteInput{Resource: "project", Action: "purge", ID: projectID})
}

func liveMustRestore(t *testing.T, ctx context.Context, resource, id string) {
	t.Helper()
	_, output, err := handlePlatformGenericWrite(ctx, "platform_restore", PlatformGenericWriteInput{Resource: resource, Action: "restore", ID: id}, "action", true)
	if err != nil || output.Error != "" || !output.ConfirmationRequired {
		t.Fatalf("%s restore = %v, %#v", resource, err, output)
	}
	liveCoverTool("whodb_platform_restore")
	liveCoverGenericWrite("action", PlatformGenericWriteInput{Resource: resource, Action: "restore"})
	liveMustConfirm(t, ctx, output.ConfirmationToken)
}

// liveMustGenericWriteAttempt requires a valid confirmation preview and then
// confirms it, accepting either success or a platform error for outcomes that
// depend on timing (for example cancelling a run that already finished).
func liveMustGenericWriteAttempt(t *testing.T, ctx context.Context, toolName, operationKind string, input PlatformGenericWriteInput) {
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
	if _, _, err := HandlePlatformConfirm(ctx, nil, ConfirmInput{Token: output.ConfirmationToken}); err != nil {
		t.Fatalf("HandlePlatformConfirm(%s/%s) handler error = %v", input.Resource, input.Action, err)
	}
	liveCoverTool("whodb_platform_confirm")
}

// liveDecodeData converts a projected read-output data value back into a typed struct.
func liveDecodeData(t *testing.T, data any, out any) {
	t.Helper()
	raw, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("marshal read data: %v", err)
	}
	if err := json.Unmarshal(raw, out); err != nil {
		t.Fatalf("decode read data %s: %v", string(raw), err)
	}
}

func liveMustExtendedRead(t *testing.T, ctx context.Context, toolName, operation string, variables map[string]any, requireData bool) {
	t.Helper()
	if variables == nil {
		variables = map[string]any{}
	}
	_, output, err := handlePlatformExtendedQuery(ctx, strings.TrimPrefix(toolName, "whodb_"), operation, variables, []string{"data", "count"})
	if err != nil {
		t.Fatalf("%s error = %v", toolName, err)
	}
	if output.Error != "" && requireData {
		t.Fatalf("%s output error = %q", toolName, output.Error)
	}
	if output.Error != "" {
		t.Logf("PLATFORM ISSUE: %s returned a platform error: %s", toolName, output.Error)
	}
	if output.Error == "" && (output.Scope == nil || output.Scope.ProjectID == "") {
		t.Fatalf("%s returned incomplete scope: %#v", toolName, output.Scope)
	}
	liveCoverTool(toolName)
}

// liveMustScopeAndCaptureTools covers the workspace-scope tools and the app
// capture tools, which need no fixtures beyond the signed-in login and an app.
func liveMustScopeAndCaptureTools(t *testing.T, ctx context.Context, appID string) {
	t.Helper()
	session, err := loadPlatformWorkspace(ctx)
	if err != nil {
		t.Fatalf("load workspace for scope tools: %v", err)
	}
	_, hosts, err := handlePlatformHosts(ctx, nil, PlatformStatusInput{})
	if err != nil || len(hosts.Hosts) == 0 {
		t.Fatalf("platform_hosts = %v %#v; want the seeded login", err, hosts)
	}
	liveCoverTool("whodb_platform_hosts")

	target := &PlatformWorkspaceTarget{Host: session.Host.URL, Org: session.Host.DefaultOrgID, Project: session.Host.DefaultProjectID}
	targetCtx, _ := platformRequestContext(ctx, nil, target)
	_, resolved, err := HandlePlatformStatus(targetCtx, nil, PlatformStatusInput{})
	if err != nil || resolved.Error != "" || resolved.DefaultProjectID != session.Host.DefaultProjectID {
		t.Fatalf("workspace_resolve = %v %#v", err, resolved)
	}
	liveCoverTool("whodb_platform_workspace_resolve")

	// Session defaults are keyed by MCP session; a nil session verifies the
	// target and reports it without persisting anything.
	req := &mcp.CallToolRequest{}
	_, used, err := handlePlatformSessionUse(targetCtx, req, PlatformSessionScopeInput{})
	if err != nil || used.Error != "" || used.SessionScope == nil || used.SessionScope.ProjectID != session.Host.DefaultProjectID {
		t.Fatalf("session_use = %v %#v", err, used)
	}
	liveCoverTool("whodb_platform_session_use")
	_, cleared, err := handlePlatformSessionClear(ctx, req, PlatformSessionScopeInput{})
	if err != nil || cleared.Status != "cleared" {
		t.Fatalf("session_clear = %v %#v", err, cleared)
	}
	liveCoverTool("whodb_platform_session_clear")

	// App capture needs a published app and a browser; the draft fixture app has
	// neither, so the tools must return a clear output error, not a crash.
	_, views, err := handlePlatformAppViews(ctx, PlatformAppCaptureInput{App: appID})
	if err != nil {
		t.Fatalf("app_views handler error = %v", err)
	}
	if views.Error != "" {
		t.Logf("app_views for the draft fixture app returned: %s", views.Error)
	}
	liveCoverTool("whodb_platform_app_views")
	_, shot, err := handlePlatformAppScreenshot(ctx, PlatformAppCaptureInput{App: appID})
	if err != nil {
		t.Fatalf("app_screenshot handler error = %v", err)
	}
	if shot.Error != "" {
		t.Logf("app_screenshot for the draft fixture app returned: %s", shot.Error)
	}
	liveCoverTool("whodb_platform_app_screenshot")
}
