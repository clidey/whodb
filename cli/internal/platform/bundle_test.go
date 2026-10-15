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

package platform

import (
	"context"
	"fmt"
	"testing"
)

type fakeBundleClient struct {
	contents        map[string]*FolderContents
	functions       []Function
	functionFetches int
}

func (c *fakeBundleClient) ProjectSecrets(context.Context, string) ([]ProjectSecret, error) {
	return nil, nil
}

func (c *fakeBundleClient) AIProviders(context.Context, string) ([]AIProvider, error) {
	return nil, nil
}

func (c *fakeBundleClient) Datasets(context.Context, string) ([]Dataset, error) {
	return nil, nil
}

func (c *fakeBundleClient) Ontologies(context.Context, string) ([]Ontology, error) {
	return nil, nil
}

func (c *fakeBundleClient) Transforms(context.Context, string) ([]Transform, error) {
	return nil, nil
}

func (c *fakeBundleClient) Functions(context.Context, string, []string) ([]Function, error) {
	summaries := make([]Function, 0, len(c.functions))
	for _, function := range c.functions {
		summaries = append(summaries, Function{ID: function.ID, Name: function.Name})
	}
	return summaries, nil
}

func (c *fakeBundleClient) Function(_ context.Context, _ string, id string, _ []string) (*Function, error) {
	c.functionFetches++
	for _, function := range c.functions {
		if function.ID == id {
			copied := function
			return &copied, nil
		}
	}
	return nil, fmt.Errorf("function %s not found", id)
}

func (c *fakeBundleClient) FolderContents(_ context.Context, _ string, folderID string, _ []string) (*FolderContents, error) {
	return c.contents[folderID], nil
}

func (c *fakeBundleClient) FilePreview(context.Context, string, string, *int, []string) (*FilePreviewResult, error) {
	return nil, nil
}

func TestPlanBundleImportPreservesFileFolderPath(t *testing.T) {
	client := &fakeBundleClient{contents: map[string]*FolderContents{
		"": {Folders: nil, Files: nil},
	}}
	folderID := "folder-source"
	bundle := &ProjectBundle{
		BundleVersion: 1,
		ProjectID:     "source-project",
		ProjectName:   "Source",
		Folders: []ProjectFolder{{
			ID:        folderID,
			ProjectID: "source-project",
			Name:      "imports",
			Path:      "imports",
		}},
		Files: []ProjectFile{{
			ID:         "file-source",
			ProjectID:  "source-project",
			FolderID:   &folderID,
			Name:       "customers.csv",
			Path:       "imports/customers.csv",
			FolderPath: "imports",
			Content:    "id,name\n1,Ada\n",
		}},
	}

	plan, err := PlanBundleImportWithOptions(context.Background(), client, "https://app.whodb.com", &Project{ID: "target-project", Name: "Target"}, bundle, BundleImportOptions{})
	if err != nil {
		t.Fatalf("PlanBundleImportWithOptions() error = %v", err)
	}
	if len(plan.Actions) != 2 {
		t.Fatalf("len(plan.Actions) = %d, want 2", len(plan.Actions))
	}
	folder := plan.Actions[0]
	if folder.Resource != "folder" || folder.Action != "create" {
		t.Fatalf("folder action = %#v, want folder create", folder)
	}
	if got := folder.Payload["path"]; got != "imports" {
		t.Fatalf("folder path = %#v, want imports", got)
	}
	file := plan.Actions[1]
	if file.Resource != "file" || file.Action != "create" {
		t.Fatalf("file action = %#v, want file create", file)
	}
	if got := file.Payload["path"]; got != "imports/customers.csv" {
		t.Fatalf("file path = %#v, want imports/customers.csv", got)
	}
	if got := file.Payload["folderId"]; got != folderID {
		t.Fatalf("file folderId = %#v, want source folder id for dependency remap", got)
	}

	file.TargetID = ""
	folder.TargetID = "folder-target"
	plan.Actions[0] = folder
	plan.Actions[1] = file
	dependencies := BundleDependencyMap{}
	for _, action := range plan.Actions {
		AddBundleDependencyMapping(dependencies, action.SourceID, action.TargetID)
	}
	ApplyBundleDependencyMap(&plan.Actions[1], dependencies)
	if got := plan.Actions[1].Payload["folderId"]; got != "folder-target" {
		t.Fatalf("remapped file folderId = %#v, want folder-target", got)
	}
}

func TestBuildProjectBundleFetchesEachFunctionSeparately(t *testing.T) {
	client := &fakeBundleClient{
		contents:  map[string]*FolderContents{"": {}},
		functions: []Function{{ID: "fn-1", Name: "post_voucher", Files: []FunctionFile{{Path: "main.py", Content: "print(1)"}}}, {ID: "fn-2", Name: "report", Files: []FunctionFile{{Path: "main.py", Content: "print(2)"}}}},
	}
	bundle, err := BuildProjectBundleWithOptions(context.Background(), client, "https://app.example.com", "org-1", "Org", &Project{ID: "project-1", Name: "Books"}, BundleExportOptions{})
	if err != nil {
		t.Fatalf("build bundle: %v", err)
	}
	if client.functionFetches != 2 {
		t.Fatalf("function fetches = %d, want one per function", client.functionFetches)
	}
	if len(bundle.Functions) != 2 || len(bundle.Functions[0].Files) != 1 || bundle.Functions[1].Files[0].Content != "print(2)" {
		t.Fatalf("bundle functions = %#v", bundle.Functions)
	}
}
