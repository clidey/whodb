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
)

func (f *fakePlatformClient) SourceFieldConstraints(ctx context.Context, projectID, sourceID string, ref platformapi.SourceObjectRefInput) ([]platformapi.SourceFieldConstraints, error) {
	return []platformapi.SourceFieldConstraints{{Name: "id", Type: "integer", Primary: true}}, nil
}

func (f *fakePlatformClient) SourceContent(ctx context.Context, projectID, sourceID string, ref platformapi.SourceObjectRefInput) (*platformapi.SourceContent, error) {
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
	return &platformapi.LineageGraph{Nodes: []platformapi.LineageNode{{ID: "dataset-1", NodeType: "dataset", Name: "Customers"}}}, nil
}

func (f *fakePlatformClient) Transforms(ctx context.Context, projectID string) ([]platformapi.Transform, error) {
	return []platformapi.Transform{{ID: "transform-1", ProjectID: projectID, Name: "Daily Load"}}, nil
}

func (f *fakePlatformClient) TransformRuns(ctx context.Context, projectID, transformID string, limit int) ([]platformapi.TransformRun, error) {
	return []platformapi.TransformRun{{ID: "run-1", TransformID: transformID, Status: "success"}}, nil
}

func (f *fakePlatformClient) Functions(ctx context.Context, projectID string) ([]platformapi.Function, error) {
	return []platformapi.Function{{ID: "fn-1", ProjectID: projectID, Name: "Enrich", Files: []platformapi.FunctionFile{{ID: "file-1", Path: "main.py", Content: strings.Repeat("x", defaultPlatformContentLimit+1)}}}}, nil
}

func (f *fakePlatformClient) Function(ctx context.Context, projectID, id string) (*platformapi.Function, error) {
	return &platformapi.Function{ID: id, ProjectID: projectID, Name: "Enrich", Files: []platformapi.FunctionFile{{ID: "file-1", Path: "main.py", Content: strings.Repeat("x", defaultPlatformContentLimit+1)}}}, nil
}

func (f *fakePlatformClient) FolderContents(ctx context.Context, projectID, folderID string) (*platformapi.FolderContents, error) {
	return &platformapi.FolderContents{Files: []platformapi.ProjectFile{{ID: "file-1", ProjectID: projectID, Name: "customers.csv", IsTabular: true}}}, nil
}

func (f *fakePlatformClient) FilePreview(ctx context.Context, projectID, fileID string, sheetIndex *int) (*platformapi.FilePreviewResult, error) {
	text := strings.Repeat("z", defaultPlatformContentLimit+1)
	return &platformapi.FilePreviewResult{MIMEType: "text/plain", SizeBytes: len(text), TextContent: &text}, nil
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
	if !strings.Contains(output.Error, "whodb-cli use --org <org> --project <project>") {
		t.Fatalf("output.Error = %q, want whodb-cli use action", output.Error)
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
