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
	"strings"
	"testing"

	platformapi "github.com/clidey/whodb/cli/internal/platform"
)

// Fake client methods for the ontology analytics tools. Each records its input
// so handler tests can assert translation from MCP input to platform calls.

func (f *fakePlatformClient) OntologyQuery(ctx context.Context, projectID string, input platformapi.OntologyQueryInput) (*platformapi.DatasetQueryResult, error) {
	f.ontologyQueryProjectID = projectID
	f.ontologyQueryInput = &input
	return &platformapi.DatasetQueryResult{Columns: []string{"region", "count"}, Rows: [][]string{{"EU", "3"}, {"UK", "2"}}, Total: 2}, nil
}

func (f *fakePlatformClient) OntologyAggregate(ctx context.Context, projectID string, input platformapi.OntologyAggregateInput) (*platformapi.DatasetQueryResult, error) {
	f.ontologyAggregateInput = &input
	return &platformapi.DatasetQueryResult{Columns: []string{"count"}, Rows: [][]string{{"5"}}, Total: 1}, nil
}

func (f *fakePlatformClient) OntologyStats(ctx context.Context, projectID, id, property string, where map[string]any) (*platformapi.OntologyStatsResult, error) {
	f.ontologyStatsProperty = property
	return &platformapi.OntologyStatsResult{Property: property, Total: 5, NonNull: 4}, nil
}

func (f *fakePlatformClient) OntologyDescribe(ctx context.Context, projectID string, entities []string, includeInferred bool) (*platformapi.OntologyDescription, error) {
	f.ontologyDescribeEntities = entities
	f.ontologyDescribeInferred = includeInferred
	out := &platformapi.OntologyDescription{Text: "Customers buy orders."}
	for _, entity := range entities {
		out.Entities = append(out.Entities, platformapi.OntologyDescriptionEntity{Name: entity})
	}
	return out, nil
}

func (f *fakePlatformClient) OntologySimilar(ctx context.Context, projectID string, input platformapi.OntologySimilarInput) (*platformapi.OntologySimilarityResult, error) {
	f.ontologySimilarInput = &input
	return &platformapi.OntologySimilarityResult{Rows: []platformapi.OntologySimilarityRow{{Score: 0.9}}}, nil
}

func (f *fakePlatformClient) OntologyRecordCapabilities(ctx context.Context, projectID, ontologyID, recordKey string) (*platformapi.OntologyRecordCapabilities, error) {
	f.ontologyCapabilitiesRecordKey = recordKey
	return &platformapi.OntologyRecordCapabilities{BehaviorVersion: 1, Actions: []platformapi.OntologyActionCapability{{APIName: "approve", Allowed: true}}}, nil
}

func (f *fakePlatformClient) PreviewOntologyAction(ctx context.Context, input platformapi.PreviewOntologyActionInput) (*platformapi.OntologyActionPreview, error) {
	f.ontologyActionPreviewInput = &input
	return &platformapi.OntologyActionPreview{Allowed: true, EmittedEvents: []string{"approved"}}, nil
}

func (f *fakePlatformClient) OntologyActionExecutions(ctx context.Context, projectID, ontologyID, recordKey string, limit int) ([]platformapi.OntologyActionExecution, error) {
	f.ontologyExecutionsLimit = limit
	return []platformapi.OntologyActionExecution{{ID: "exec-1", ActionName: "approve", Status: "committed"}}, nil
}

func (f *fakePlatformClient) OntologyEmbeddingJobs(ctx context.Context, projectID, entityID string) ([]platformapi.OntologyEmbeddingJob, error) {
	return nil, nil
}

func (f *fakePlatformClient) PreviewTransform(ctx context.Context, projectID, graphJSON string) (*platformapi.DatasetQueryResult, error) {
	f.transformPreviewGraph = graphJSON
	return &platformapi.DatasetQueryResult{Columns: []string{"a"}, Rows: [][]string{{"1"}, {"2"}, {"3"}}, Total: 3}, nil
}

func newOntologyToolsFake(t *testing.T) *fakePlatformClient {
	t.Helper()
	client := &fakePlatformClient{}
	withPlatformSessionLoader(t, func(context.Context) (*platformToolSession, error) {
		return testPlatformSession(client), nil
	})
	return client
}

func TestHandlePlatformOntologyQueryTranslatesInput(t *testing.T) {
	client := newOntologyToolsFake(t)
	_, output, err := HandlePlatformOntologyQuery(context.Background(), nil, PlatformOntologyQueryInput{
		Entity:  " customer ",
		Where:   map[string]any{"region": map[string]any{"neq": "US"}},
		GroupBy: []string{"region"},
		Metrics: []PlatformOntologyMetricInput{{Op: "COUNT"}},
		Sort:    []PlatformOntologyQuerySortInput{{Field: "count", Desc: true}},
		Limit:   500,
	}, &SecurityOptions{MaxRows: 50})
	if err != nil || output.Error != "" {
		t.Fatalf("query error = %v %q", err, output.Error)
	}
	got := client.ontologyQueryInput
	if client.ontologyQueryProjectID != "proj-1" || got.Entity != "customer" {
		t.Fatalf("project/entity = %q/%q", client.ontologyQueryProjectID, got.Entity)
	}
	if got.PageSize != 50 {
		t.Fatalf("PageSize = %d, want --max-rows cap 50", got.PageSize)
	}
	if len(got.Metrics) != 1 || got.Metrics[0].Op != "COUNT" || len(got.Sort) != 1 || !got.Sort[0].Desc {
		t.Fatalf("metrics/sort not forwarded: %+v", got)
	}
	if output.Count != 2 || output.Scope == nil || output.Scope.ProjectID != "proj-1" {
		t.Fatalf("output = %+v", output)
	}
}

func TestHandlePlatformOntologyQueryValidation(t *testing.T) {
	newOntologyToolsFake(t)
	cases := map[string]PlatformOntologyQueryInput{
		"entity is required":      {},
		"scan_limit must be":      {Entity: "customer", ScanLimit: 9000},
		"unsupported metric op":   {Entity: "customer", Metrics: []PlatformOntologyMetricInput{{Op: "median"}}},
		"offset must be non-neg":  {Entity: "customer", Offset: -1},
		"metrics[].op is require": {Entity: "customer", Metrics: []PlatformOntologyMetricInput{{Field: "x"}}},
	}
	for want, input := range cases {
		_, output, err := HandlePlatformOntologyQuery(context.Background(), nil, input, nil)
		if err != nil || !strings.Contains(output.Error, want) {
			t.Errorf("input %+v: error = %v %q, want %q", input, err, output.Error, want)
		}
	}
}

func TestHandlePlatformOntologyAggregateNormalizesOpsAndSort(t *testing.T) {
	client := newOntologyToolsFake(t)
	_, output, err := HandlePlatformOntologyAggregate(context.Background(), nil, PlatformOntologyAggregateInput{
		ID:      "ont-1",
		Metrics: []PlatformOntologyMetricInput{{Op: "sum", Field: "amount", As: "total"}},
		Sort:    []PlatformSortConditionInput{{Column: "total"}},
	}, nil)
	if err != nil || output.Error != "" {
		t.Fatalf("aggregate error = %v %q", err, output.Error)
	}
	got := client.ontologyAggregateInput
	if got.ID != "ont-1" || got.Metrics[0].Op != "sum" || got.Sort[0].Direction != "ASC" || got.GroupBy != nil {
		t.Fatalf("aggregate input = %+v", got)
	}
	if got.PageSize != defaultPlatformRowLimit {
		t.Fatalf("PageSize = %d, want default row limit", got.PageSize)
	}
	_, output, _ = HandlePlatformOntologyAggregate(context.Background(), nil, PlatformOntologyAggregateInput{ID: "ont-1"}, nil)
	if !strings.Contains(output.Error, "metrics is required") {
		t.Fatalf("missing metrics error = %q", output.Error)
	}
	_, output, _ = HandlePlatformOntologyAggregate(context.Background(), nil, PlatformOntologyAggregateInput{ID: "ont-1", Metrics: []PlatformOntologyMetricInput{{Op: "count_distinct"}}}, nil)
	if !strings.Contains(output.Error, "unsupported metric op") {
		t.Fatalf("aggregate accepted a query-only op: %q", output.Error)
	}
}

func TestHandlePlatformOntologyDescribeDefaultsToAllOntologies(t *testing.T) {
	client := newOntologyToolsFake(t)
	_, output, err := HandlePlatformOntologyDescribe(context.Background(), nil, PlatformOntologyDescribeInput{IncludeInferred: true})
	if err != nil || output.Error != "" {
		t.Fatalf("describe error = %v %q", err, output.Error)
	}
	if len(client.ontologyDescribeEntities) != 1 || client.ontologyDescribeEntities[0] != "customer" || !client.ontologyDescribeInferred {
		t.Fatalf("describe entities = %v inferred=%v", client.ontologyDescribeEntities, client.ontologyDescribeInferred)
	}
	if output.Count != 1 {
		t.Fatalf("count = %d", output.Count)
	}
}

func TestHandlePlatformOntologyStatsSimilarPreviewValidation(t *testing.T) {
	client := newOntologyToolsFake(t)
	if _, output, _ := HandlePlatformOntologyStats(context.Background(), nil, PlatformOntologyStatsInput{ID: "ont-1"}); !strings.Contains(output.Error, "property is required") {
		t.Fatalf("stats error = %q", output.Error)
	}
	if _, output, _ := HandlePlatformOntologyStats(context.Background(), nil, PlatformOntologyStatsInput{ID: "ont-1", Property: "amount"}); output.Error != "" || client.ontologyStatsProperty != "amount" {
		t.Fatalf("stats = %+v property=%q", output, client.ontologyStatsProperty)
	}
	if _, output, _ := HandlePlatformOntologySimilar(context.Background(), nil, PlatformOntologySimilarInput{EntityID: "ont-1"}); !strings.Contains(output.Error, "row_id is required") {
		t.Fatalf("similar error = %q", output.Error)
	}
	if _, output, _ := HandlePlatformOntologySimilar(context.Background(), nil, PlatformOntologySimilarInput{EntityID: "ont-1", RowID: "row-1", TopK: 3}); output.Error != "" || client.ontologySimilarInput.TopK != 3 {
		t.Fatalf("similar = %+v", output)
	}
	if _, output, _ := HandlePlatformOntologyActionPreview(context.Background(), nil, PlatformOntologyActionPreviewInput{OntologyID: "ont-1"}); !strings.Contains(output.Error, "action is required") {
		t.Fatalf("preview error = %q", output.Error)
	}
	_, output, _ := HandlePlatformOntologyActionPreview(context.Background(), nil, PlatformOntologyActionPreviewInput{OntologyID: "ont-1", RecordKey: "r1", Action: "approve", Values: map[string]any{"note": "ok"}})
	if output.Error != "" || client.ontologyActionPreviewInput.ProjectID != "proj-1" || client.ontologyActionPreviewInput.Action != "approve" {
		t.Fatalf("preview = %+v input=%+v", output, client.ontologyActionPreviewInput)
	}
	if _, output, _ := HandlePlatformOntologyRecordCapabilities(context.Background(), nil, PlatformOntologyRecordCapabilitiesInput{OntologyID: "ont-1", RecordKey: "r1"}); output.Error != "" || client.ontologyCapabilitiesRecordKey != "r1" || output.Count != 1 {
		t.Fatalf("capabilities = %+v", output)
	}
	if _, output, _ := HandlePlatformOntologyActionExecutions(context.Background(), nil, PlatformOntologyActionExecutionsInput{OntologyID: "ont-1", RecordKey: "r1"}); output.Error != "" || client.ontologyExecutionsLimit != 20 {
		t.Fatalf("executions = %+v limit=%d", output, client.ontologyExecutionsLimit)
	}
	if _, output, _ := HandlePlatformOntologyEmbeddingJobs(context.Background(), nil, PlatformEntityInput{ID: "ont-1"}); output.Error != "" || output.Count != 0 {
		t.Fatalf("embedding jobs = %+v", output)
	}
}

func TestHandlePlatformTransformPreviewCapsRows(t *testing.T) {
	client := newOntologyToolsFake(t)
	if _, output, _ := HandlePlatformTransformPreview(context.Background(), nil, PlatformTransformPreviewInput{GraphJSON: "{nope"}, nil); !strings.Contains(output.Error, "valid JSON") {
		t.Fatalf("invalid graph error = %q", output.Error)
	}
	_, output, err := HandlePlatformTransformPreview(context.Background(), nil, PlatformTransformPreviewInput{GraphJSON: `{"nodes":[]}`}, &SecurityOptions{MaxRows: 2})
	if err != nil || output.Error != "" {
		t.Fatalf("preview error = %v %q", err, output.Error)
	}
	if client.transformPreviewGraph != `{"nodes":[]}` || output.Count != 2 || !output.Truncated {
		t.Fatalf("preview = %+v graph=%q", output, client.transformPreviewGraph)
	}
}

func TestHandlePlatformAddOntologyRecordsBuildsRows(t *testing.T) {
	client := newOntologyToolsFake(t)
	if _, output, _ := handlePlatformOntologyRecordsWrite(context.Background(), PlatformOntologyRecordsInput{EntityID: "ont-1"}, false); !strings.Contains(output.Error, "rows is required") {
		t.Fatalf("missing rows error = %q", output.Error)
	}
	_, output, err := handlePlatformOntologyRecordsWrite(context.Background(), PlatformOntologyRecordsInput{
		EntityID:       "ont-1",
		Rows:           []map[string]string{{"name": "Ada", "region": "EU"}, {"name": "Grace"}},
		IdempotencyKey: "import-1",
	}, false)
	if err != nil || output.Error != "" || output.ConfirmationRequired {
		t.Fatalf("add records = %v %+v", err, output)
	}
	if client.mutationName != "OntologyAddRows" {
		t.Fatalf("mutation = %q", client.mutationName)
	}
	vars := client.mutationVariables
	if vars["projectId"] != "proj-1" || vars["entityId"] != "ont-1" || vars["idempotencyKey"] != "import-1" {
		t.Fatalf("variables = %#v", vars)
	}
	rows, ok := vars["rows"].([]any)
	if !ok || len(rows) != 2 {
		t.Fatalf("rows = %#v", vars["rows"])
	}
	first := rows[0].(map[string]any)["values"].([]any)
	if len(first) != 2 || first[0].(map[string]any)["Key"] != "name" || first[0].(map[string]any)["Value"] != "Ada" {
		t.Fatalf("first row values = %#v", first)
	}
}

func TestOntologyAnalyticsExtendedReadsInjectScope(t *testing.T) {
	client := newOntologyToolsFake(t)
	server := NewServer(&ServerOptions{PlatformEnabled: true})
	_ = server
	_, output, err := handlePlatformExtendedQuery(context.Background(), "platform_service_accounts", "ServiceAccounts", map[string]any{}, nil)
	if err != nil || output.Error != "" || client.platformQueryOperation != "ServiceAccounts" || client.platformQueryVariables["orgId"] != "org-1" {
		t.Fatalf("service accounts = %v %+v op=%q vars=%#v", err, output, client.platformQueryOperation, client.platformQueryVariables)
	}
	_, output, _ = handlePlatformExtendedQuery(context.Background(), "platform_ontology_reference_impact", "OntologyReferenceImpact", map[string]any{"referenceId": ""}, nil)
	if output.Error == "" {
		t.Fatalf("empty referenceId accepted")
	}
}
