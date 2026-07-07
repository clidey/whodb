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
	"testing"
)

type fakeBundleClient struct {
	contents map[string]*FolderContents
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
	return nil, nil
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
