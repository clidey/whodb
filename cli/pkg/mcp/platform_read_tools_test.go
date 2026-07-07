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
	"reflect"
	"strings"
	"testing"

	platformapi "github.com/clidey/whodb/cli/internal/platform"
)

func (f *fakePlatformClient) SourceFieldConstraints(ctx context.Context, projectID, sourceID string, ref platformapi.SourceObjectRefInput) ([]platformapi.SourceFieldConstraints, error) {
	return []platformapi.SourceFieldConstraints{{Name: "id", Type: "integer", Primary: true}}, nil
}

func (f *fakePlatformClient) SourceContent(ctx context.Context, projectID, sourceID string, ref platformapi.SourceObjectRefInput, fields []string) (*platformapi.SourceContent, error) {
	f.sourceContentFields = append([]string(nil), fields...)
	text := strings.Repeat("a", defaultPlatformContentLimit+1)
	return &platformapi.SourceContent{Text: &text, MIMEType: "text/plain", FileName: "large.txt", SizeBytes: "65537"}, nil
}

func (f *fakePlatformClient) ProjectSecrets(ctx context.Context, projectID string) ([]platformapi.ProjectSecret, error) {
	return []platformapi.ProjectSecret{{
		ID:          "secret-1",
		ProjectID:   projectID,
		Name:        "OPENAI_API_KEY",
		Description: "Provider key",
		UsedBy:      []platformapi.PlatformSecretUsage{{ConsumerType: "function", ConsumerID: "fn-1", ConsumerName: "Enrich", BindingName: "OPENAI_API_KEY", Mode: "ENV"}},
	}}, nil
}

func (f *fakePlatformClient) AIProviders(ctx context.Context, projectID string) ([]platformapi.AIProvider, error) {
	return []platformapi.AIProvider{{ID: "provider-1", ProjectID: projectID, Name: "OpenAI", ProviderType: "openai"}}, nil
}

func (f *fakePlatformClient) AIProviderModels(ctx context.Context, projectID, providerID string) ([]string, error) {
	return []string{"gpt-4.1"}, nil
}

func (f *fakePlatformClient) Ontologies(ctx context.Context, projectID string) ([]platformapi.Ontology, error) {
	return []platformapi.Ontology{{ID: "ont-1", ProjectID: projectID, APIName: "customer", DisplayName: "Customer"}}, nil
}

func (f *fakePlatformClient) Ontology(ctx context.Context, projectID, id string) (*platformapi.Ontology, error) {
	return &platformapi.Ontology{ID: id, ProjectID: projectID, APIName: "customer", DisplayName: "Customer"}, nil
}

func (f *fakePlatformClient) OntologyFastLookups(ctx context.Context, projectID, entityID string) ([]platformapi.OntologyFastLookup, error) {
	return []platformapi.OntologyFastLookup{{ID: "lookup-1", EntityID: entityID, DisplayName: "Search", Fields: []string{"email"}}}, nil
}

func (f *fakePlatformClient) OntologyFastLookupSuggestions(ctx context.Context, projectID, entityID string) ([]platformapi.OntologyFastLookupSuggestion, error) {
	return []platformapi.OntologyFastLookupSuggestion{{EntityID: entityID, DisplayName: "Suggested", Fields: []string{"name"}, CanCreate: true}}, nil
}

func (f *fakePlatformClient) OntologyRows(ctx context.Context, projectID, id string, pageSize, pageOffset int) (*platformapi.DatasetQueryResult, error) {
	f.rowsLimit = pageSize
	f.rowsOffset = pageOffset
	return &platformapi.DatasetQueryResult{Columns: []string{"id"}, Rows: [][]string{{"1"}}, Total: 2}, nil
}

func (f *fakePlatformClient) OntologyFollowLink(ctx context.Context, projectID, entityID, pk, linkAPIName string, pageSize, pageOffset int) (*platformapi.DatasetQueryResult, error) {
	return &platformapi.DatasetQueryResult{Columns: []string{"id"}, Rows: [][]string{{"2"}}, Total: 1}, nil
}

func (f *fakePlatformClient) Datasets(ctx context.Context, projectID string) ([]platformapi.Dataset, error) {
	return []platformapi.Dataset{{ID: "dataset-1", ProjectID: projectID, Name: "Customers"}}, nil
}

func (f *fakePlatformClient) Dataset(ctx context.Context, projectID, id string) (*platformapi.Dataset, error) {
	return &platformapi.Dataset{ID: id, ProjectID: projectID, Name: "Customers"}, nil
}

func (f *fakePlatformClient) DatasetRows(ctx context.Context, projectID, datasetID string, pageSize, pageOffset int) (*platformapi.DatasetQueryResult, error) {
	f.rowsLimit = pageSize
	f.rowsOffset = pageOffset
	return &platformapi.DatasetQueryResult{Columns: []string{"id"}, Rows: [][]string{{"1"}}, Total: 3}, nil
}

func (f *fakePlatformClient) Lineage(ctx context.Context, projectID, rootID, rootType, direction string, maxDepth int) (*platformapi.LineageGraph, error) {
	return &platformapi.LineageGraph{Nodes: []platformapi.LineageNode{{ID: rootID, NodeType: rootType, Name: "Root"}}}, nil
}

func (f *fakePlatformClient) LineageNeighbors(ctx context.Context, projectID, nodeID, nodeType string) (*platformapi.LineageGraph, error) {
	return &platformapi.LineageGraph{Nodes: []platformapi.LineageNode{{ID: nodeID, NodeType: nodeType, Name: "Node"}}}, nil
}

func (f *fakePlatformClient) ProjectLineage(ctx context.Context, projectID string) (*platformapi.LineageGraph, error) {
	return &platformapi.LineageGraph{
		Nodes: []platformapi.LineageNode{
			{ID: "file-1", NodeType: "file", Name: "customers.csv"},
			{ID: "dataset-1", NodeType: "dataset", Name: "Customers"},
		},
		Edges: []platformapi.LineageEdge{{SourceID: "file-1", SourceType: "file", TargetID: "dataset-1", TargetType: "dataset"}},
	}, nil
}

func (f *fakePlatformClient) Transforms(ctx context.Context, projectID string) ([]platformapi.Transform, error) {
	return []platformapi.Transform{{ID: "transform-1", ProjectID: projectID, Name: "Daily Load"}}, nil
}

func (f *fakePlatformClient) TransformRuns(ctx context.Context, projectID, transformID string, limit int) ([]platformapi.TransformRun, error) {
	return []platformapi.TransformRun{{ID: "run-1", TransformID: transformID, Status: "success"}}, nil
}

func (f *fakePlatformClient) Functions(ctx context.Context, projectID string, fields []string) ([]platformapi.Function, error) {
	f.functionsFields = append([]string(nil), fields...)
	return []platformapi.Function{{
		ID:                  "fn-1",
		ProjectID:           projectID,
		Name:                "Enrich",
		Language:            "python",
		EntryPoint:          "main",
		ProviderIDs:         []string{"provider-1"},
		OntologyIDs:         []string{"ont-1"},
		ReadOnlyOntologyIDs: []string{"ont-2"},
		SecretBindings:      []platformapi.FunctionSecretBinding{{Name: "OPENAI_API_KEY", SecretID: "secret-1", Target: "env"}},
		Files:               []platformapi.FunctionFile{{ID: "file-1", Path: "main.py", Content: strings.Repeat("x", defaultPlatformContentLimit+1)}},
	}}, nil
}

func (f *fakePlatformClient) Function(ctx context.Context, projectID, id string, fields []string) (*platformapi.Function, error) {
	f.functionFields = append([]string(nil), fields...)
	return &platformapi.Function{ID: id, ProjectID: projectID, Name: "Enrich", Files: []platformapi.FunctionFile{{ID: "file-1", Path: "main.py", Content: strings.Repeat("x", defaultPlatformContentLimit+1)}}}, nil
}

func (f *fakePlatformClient) FolderContents(ctx context.Context, projectID, folderID string, fields []string) (*platformapi.FolderContents, error) {
	f.folderContentsFields = append([]string(nil), fields...)
	folderIDValue := "folder-1"
	datasetID := "dataset-1"
	return &platformapi.FolderContents{
		Folders:     []platformapi.ProjectFolder{{ID: folderIDValue, ProjectID: projectID, Name: "Uploads", Path: "/Uploads"}},
		Files:       []platformapi.ProjectFile{{ID: "file-1", ProjectID: projectID, FolderID: &folderIDValue, Name: "customers.csv", MIMEType: "text/csv", IsTabular: true, DatasetID: &datasetID}},
		StorageUsed: 512,
	}, nil
}

func (f *fakePlatformClient) FilePreview(ctx context.Context, projectID, fileID string, sheetIndex *int, fields []string) (*platformapi.FilePreviewResult, error) {
	f.filePreviewFields = append([]string(nil), fields...)
	text := strings.Repeat("z", defaultPlatformContentLimit+1)
	return &platformapi.FilePreviewResult{
		MIMEType:    "text/csv",
		SizeBytes:   len(text),
		IsTabular:   true,
		TextContent: &text,
		Tabular: &platformapi.TabularPreviewData{
			Columns: []platformapi.FilePreviewColumn{{Name: "id", Type: "integer"}, {Name: "Customer Name", Type: "string"}},
			Rows:    [][]string{{"1", "Ada"}},
			Total:   1,
		},
	}, nil
}

func (f *fakePlatformClient) SearchProjectFiles(ctx context.Context, projectID, query string) ([]platformapi.ProjectFile, error) {
	return []platformapi.ProjectFile{{ID: "file-1", ProjectID: projectID, Name: "customers.csv"}}, nil
}

func (f *fakePlatformClient) ProjectTabularFiles(ctx context.Context, projectID string) ([]platformapi.ProjectFile, error) {
	return []platformapi.ProjectFile{{ID: "file-1", ProjectID: projectID, Name: "customers.csv", IsTabular: true}}, nil
}

func (f *fakePlatformClient) ProjectStorageUsage(ctx context.Context, projectID string) (int, error) {
	return 1024, nil
}

func TestHandlePlatformSecretsDoesNotExposeValues(t *testing.T) {
	client := &fakePlatformClient{}
	withPlatformSessionLoader(t, func(context.Context) (*platformToolSession, error) {
		return testPlatformSession(client), nil
	})

	_, output, err := HandlePlatformSecrets(context.Background(), nil, PlatformEmptyInput{})
	if err != nil {
		t.Fatalf("HandlePlatformSecrets() error = %v", err)
	}
	if output.Error != "" {
		t.Fatalf("HandlePlatformSecrets() output error = %q", output.Error)
	}
	raw, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("json.Marshal(output) error = %v", err)
	}
	if strings.Contains(strings.ToLower(string(raw)), "value") || strings.Contains(string(raw), "secret-value") {
		t.Fatalf("secret output exposed value-like fields: %s", raw)
	}
}

func TestHandlePlatformWorkspaceMapSummarizesSelectedProject(t *testing.T) {
	client := &fakePlatformClient{}
	withPlatformSessionLoader(t, func(context.Context) (*platformToolSession, error) {
		return testPlatformSession(client), nil
	})

	_, output, err := HandlePlatformWorkspaceMap(context.Background(), nil, PlatformWorkspaceMapInput{})
	if err != nil {
		t.Fatalf("HandlePlatformWorkspaceMap() error = %v", err)
	}
	if output.Error != "" {
		t.Fatalf("HandlePlatformWorkspaceMap() output error = %q", output.Error)
	}
	workspaceMap, ok := output.Data.(PlatformWorkspaceMap)
	if !ok {
		t.Fatalf("output.Data = %T, want PlatformWorkspaceMap", output.Data)
	}
	if workspaceMap.ProjectID != "proj-1" || workspaceMap.Counts["sources"] != 1 || workspaceMap.Counts["datasets"] != 1 || workspaceMap.Counts["files"] != 1 {
		t.Fatalf("workspace map = %#v, want selected project counts", workspaceMap)
	}
	if workspaceMap.Lineage == nil || workspaceMap.Lineage.EdgeCount != 1 {
		t.Fatalf("lineage summary = %#v, want one edge", workspaceMap.Lineage)
	}
	if output.Count == 0 {
		t.Fatalf("output.Count = 0, want aggregate resource count")
	}
}

func TestHandlePlatformResourceGraphBuildsRelationships(t *testing.T) {
	client := &fakePlatformClient{}
	withPlatformSessionLoader(t, func(context.Context) (*platformToolSession, error) {
		return testPlatformSession(client), nil
	})

	_, output, err := HandlePlatformResourceGraph(context.Background(), nil, PlatformResourceGraphInput{})
	if err != nil {
		t.Fatalf("HandlePlatformResourceGraph() error = %v", err)
	}
	if output.Error != "" {
		t.Fatalf("HandlePlatformResourceGraph() output error = %q", output.Error)
	}
	graph, ok := output.Data.(PlatformResourceGraph)
	if !ok {
		t.Fatalf("output.Data = %T, want PlatformResourceGraph", output.Data)
	}
	if len(graph.Nodes) == 0 || output.Count != len(graph.Nodes) {
		t.Fatalf("graph node count = %d output.Count=%d, want nodes", len(graph.Nodes), output.Count)
	}
	if !hasPlatformGraphEdge(graph.Edges, "fn-1", "function", "provider-1", "ai_provider", "uses_provider") {
		t.Fatalf("edges = %#v, want function provider edge", graph.Edges)
	}
	if !hasPlatformGraphEdge(graph.Edges, "secret-1", "secret", "fn-1", "function", "secret_binding") {
		t.Fatalf("edges = %#v, want secret binding edge", graph.Edges)
	}
	if !hasPlatformGraphEdge(graph.Edges, "file-1", "file", "dataset-1", "dataset", "promoted_to_dataset") {
		t.Fatalf("edges = %#v, want file dataset edge", graph.Edges)
	}
}

func TestHandlePlatformNextActionsSuggestsWorkspaceFlow(t *testing.T) {
	client := &fakePlatformClient{}
	withPlatformSessionLoader(t, func(context.Context) (*platformToolSession, error) {
		return testPlatformSession(client), nil
	})

	_, output, err := HandlePlatformNextActions(context.Background(), nil, PlatformNextActionsInput{Goal: "understand customer data"})
	if err != nil {
		t.Fatalf("HandlePlatformNextActions() error = %v", err)
	}
	if output.Error != "" {
		t.Fatalf("HandlePlatformNextActions() output error = %q", output.Error)
	}
	next, ok := output.Data.(PlatformNextActions)
	if !ok {
		t.Fatalf("output.Data = %T, want PlatformNextActions", output.Data)
	}
	if next.Goal != "understand customer data" || len(next.Actions) == 0 {
		t.Fatalf("next actions = %#v, want goal and actions", next)
	}
	if !hasPlatformNextActionTool(next.Actions, "whodb_platform_source_objects") {
		t.Fatalf("actions = %#v, want source inspection suggestion", next.Actions)
	}
	if !hasPlatformNextActionTool(next.Actions, "whodb_platform_action") {
		t.Fatalf("actions = %#v, want deploy/run-aware write suggestion", next.Actions)
	}
}

func TestHandlePlatformListFilters(t *testing.T) {
	client := &fakePlatformClient{}
	withPlatformSessionLoader(t, func(context.Context) (*platformToolSession, error) {
		return testPlatformSession(client), nil
	})

	_, datasets, err := HandlePlatformDatasets(context.Background(), nil, PlatformEmptyInput{Name: "cust"})
	if err != nil {
		t.Fatalf("HandlePlatformDatasets() error = %v", err)
	}
	if datasets.Count != 1 {
		t.Fatalf("filtered dataset count = %d, want 1", datasets.Count)
	}
	_, datasets, err = HandlePlatformDatasets(context.Background(), nil, PlatformEmptyInput{Name: "orders"})
	if err != nil {
		t.Fatalf("HandlePlatformDatasets() error = %v", err)
	}
	if datasets.Count != 0 {
		t.Fatalf("filtered dataset count = %d, want 0", datasets.Count)
	}

	_, files, err := HandlePlatformFiles(context.Background(), nil, PlatformFilesInput{Name: "customers", Kind: "file", MIMEType: "csv"})
	if err != nil {
		t.Fatalf("HandlePlatformFiles() error = %v", err)
	}
	if files.Count != 1 {
		t.Fatalf("filtered file count = %d, want 1", files.Count)
	}
	_, files, err = HandlePlatformFiles(context.Background(), nil, PlatformFilesInput{Name: "missing", Kind: "folder"})
	if err != nil {
		t.Fatalf("HandlePlatformFiles() error = %v", err)
	}
	if files.Count != 0 {
		t.Fatalf("filtered folder count = %d, want 0", files.Count)
	}

	_, functions, err := HandlePlatformFunctions(context.Background(), nil, PlatformEmptyInput{Name: "enrich", Deployed: "false"})
	if err != nil {
		t.Fatalf("HandlePlatformFunctions() error = %v", err)
	}
	if functions.Count != 1 {
		t.Fatalf("filtered function count = %d, want 1", functions.Count)
	}
	_, functions, err = HandlePlatformFunctions(context.Background(), nil, PlatformEmptyInput{Deployed: "true"})
	if err != nil {
		t.Fatalf("HandlePlatformFunctions() error = %v", err)
	}
	if functions.Count != 0 {
		t.Fatalf("filtered function count = %d, want 0", functions.Count)
	}
}

func TestHandlePlatformFunctionTruncatesFileContent(t *testing.T) {
	client := &fakePlatformClient{}
	withPlatformSessionLoader(t, func(context.Context) (*platformToolSession, error) {
		return testPlatformSession(client), nil
	})

	_, output, err := HandlePlatformFunction(context.Background(), nil, PlatformEntityInput{ID: "fn-1"})
	if err != nil {
		t.Fatalf("HandlePlatformFunction() error = %v", err)
	}
	if output.Error != "" {
		t.Fatalf("HandlePlatformFunction() output error = %q", output.Error)
	}
	if !output.Truncated {
		t.Fatalf("output.Truncated = false, want true")
	}
	function, ok := output.Data.(*platformapi.Function)
	if !ok {
		t.Fatalf("output.Data = %T, want *platformapi.Function", output.Data)
	}
	if len(function.Files) != 1 || len(function.Files[0].Content) != defaultPlatformContentLimit {
		t.Fatalf("function file content length = %d, want %d", len(function.Files[0].Content), defaultPlatformContentLimit)
	}
}

func TestHandlePlatformFileInspectOmitsRowsByDefault(t *testing.T) {
	client := &fakePlatformClient{}
	withPlatformSessionLoader(t, func(context.Context) (*platformToolSession, error) {
		return testPlatformSession(client), nil
	})

	_, output, err := HandlePlatformFileInspect(context.Background(), nil, PlatformFileInspectInput{FileID: "file-1"})
	if err != nil {
		t.Fatalf("HandlePlatformFileInspect() error = %v", err)
	}
	if output.Error != "" {
		t.Fatalf("HandlePlatformFileInspect() output error = %q", output.Error)
	}
	inspection, ok := output.Data.(*platformapi.FileInspection)
	if !ok {
		t.Fatalf("output.Data = %T, want *platformapi.FileInspection", output.Data)
	}
	if len(inspection.Columns) != 2 {
		t.Fatalf("columns = %#v, want inferred columns", inspection.Columns)
	}
	if inspection.Rows != nil {
		t.Fatalf("rows = %#v, want omitted by default", inspection.Rows)
	}
	if inspection.ColumnMapExample == "" {
		t.Fatal("column map example empty")
	}
}

func TestHandlePlatformFunctionPassesFieldsToClient(t *testing.T) {
	client := &fakePlatformClient{}
	withPlatformSessionLoader(t, func(context.Context) (*platformToolSession, error) {
		return testPlatformSession(client), nil
	})

	fields := []string{"id", "name"}
	_, output, err := HandlePlatformFunction(context.Background(), nil, PlatformEntityInput{ID: "fn-1", Fields: fields})
	if err != nil {
		t.Fatalf("HandlePlatformFunction() error = %v", err)
	}
	if output.Error != "" {
		t.Fatalf("HandlePlatformFunction() output error = %q", output.Error)
	}
	if !reflect.DeepEqual(client.functionFields, fields) {
		t.Fatalf("function fields = %#v, want %#v", client.functionFields, fields)
	}
}

func TestHandlePlatformSourceContentPassesFieldsToClient(t *testing.T) {
	client := &fakePlatformClient{}
	withPlatformSessionLoader(t, func(context.Context) (*platformToolSession, error) {
		return testPlatformSession(client), nil
	})

	fields := []string{"fileName", "sizeBytes"}
	_, output, err := HandlePlatformSourceContent(context.Background(), nil, PlatformSourceContentInput{Source: "Warehouse", Ref: "file:notes/report.txt", Fields: fields})
	if err != nil {
		t.Fatalf("HandlePlatformSourceContent() error = %v", err)
	}
	if output.Error != "" {
		t.Fatalf("HandlePlatformSourceContent() output error = %q", output.Error)
	}
	if !reflect.DeepEqual(client.sourceContentFields, fields) {
		t.Fatalf("source content fields = %#v, want %#v", client.sourceContentFields, fields)
	}
}

func TestHandlePlatformDatasetRowsCapsLimit(t *testing.T) {
	client := &fakePlatformClient{}
	withPlatformSessionLoader(t, func(context.Context) (*platformToolSession, error) {
		return testPlatformSession(client), nil
	})

	_, output, err := HandlePlatformDatasetRows(context.Background(), nil, PlatformRowsInput{ID: "dataset-1", Limit: 100, Offset: 4}, &SecurityOptions{MaxRows: 5})
	if err != nil {
		t.Fatalf("HandlePlatformDatasetRows() error = %v", err)
	}
	if output.Error != "" {
		t.Fatalf("HandlePlatformDatasetRows() output error = %q", output.Error)
	}
	if client.rowsLimit != 5 || client.rowsOffset != 4 {
		t.Fatalf("rows limit/offset = %d/%d, want 5/4", client.rowsLimit, client.rowsOffset)
	}
	if !output.Truncated {
		t.Fatalf("output.Truncated = false, want true")
	}
}

func TestHandlePlatformDatasetsReportsMissingWorkspaceAction(t *testing.T) {
	client := &fakePlatformClient{}
	withPlatformSessionLoader(t, func(context.Context) (*platformToolSession, error) {
		session := testPlatformSession(client)
		session.Host.URL = "http://localhost:8080"
		session.Host.DefaultOrgID = ""
		session.Host.DefaultOrgName = ""
		session.Host.DefaultProjectID = ""
		session.Host.DefaultProjectName = ""
		return session, nil
	})

	_, output, err := HandlePlatformDatasets(context.Background(), nil, PlatformEmptyInput{})
	if err != nil {
		t.Fatalf("HandlePlatformDatasets() error = %v", err)
	}
	want := "whodb-cli use --host http://localhost:8080 --org <org> --project <project>"
	if !strings.Contains(output.Error, want) {
		t.Fatalf("output.Error = %q, want %q", output.Error, want)
	}
}

func TestHandlePlatformDatasetsProjectsFieldsAndScope(t *testing.T) {
	client := &fakePlatformClient{}
	withPlatformSessionLoader(t, func(context.Context) (*platformToolSession, error) {
		return testPlatformSession(client), nil
	})

	_, output, err := HandlePlatformDatasets(context.Background(), nil, PlatformEmptyInput{Fields: []string{"id", "name"}})
	if err != nil {
		t.Fatalf("HandlePlatformDatasets() error = %v", err)
	}
	if output.Error != "" {
		t.Fatalf("HandlePlatformDatasets() output error = %q", output.Error)
	}
	if output.Count != 1 {
		t.Fatalf("output.Count = %d, want 1", output.Count)
	}
	if output.Scope == nil || output.Scope.ProjectID != "proj-1" || output.Scope.ProjectName != "Customer" {
		t.Fatalf("output.Scope = %#v, want selected project scope", output.Scope)
	}
	if len(output.Items) != 1 {
		t.Fatalf("output.Items = %#v, want one projected item", output.Items)
	}
	item := output.Items[0]
	if item["id"] != "dataset-1" || item["name"] != "Customers" {
		t.Fatalf("projected item = %#v, want id/name", item)
	}
	if _, ok := item["projectId"]; ok {
		t.Fatalf("projected item includes unrequested projectId: %#v", item)
	}
}

func TestPlatformReadOutputWarnsOnEmptyList(t *testing.T) {
	session := testPlatformSession(&fakePlatformClient{})
	output := platformReadOutput(session, "platform_datasets", []platformapi.Dataset{}, 0, false, "req-1", nil)
	if output.Count != 0 {
		t.Fatalf("output.Count = %d, want 0", output.Count)
	}
	if len(output.Warnings) != 1 {
		t.Fatalf("output.Warnings = %#v, want one warning", output.Warnings)
	}
	if !strings.Contains(output.Warnings[0], "Clidey / Customer") {
		t.Fatalf("warning = %q, want workspace name", output.Warnings[0])
	}
}

func hasPlatformGraphEdge(edges []PlatformResourceGraphEdge, fromID, fromType, toID, toType, kind string) bool {
	for _, edge := range edges {
		if edge.FromID == fromID && edge.FromType == fromType && edge.ToID == toID && edge.ToType == toType && edge.Kind == kind {
			return true
		}
	}
	return false
}

func hasPlatformNextActionTool(actions []PlatformNextAction, toolName string) bool {
	for _, action := range actions {
		for _, suggested := range action.SuggestedTools {
			if suggested == toolName {
				return true
			}
		}
	}
	return false
}
