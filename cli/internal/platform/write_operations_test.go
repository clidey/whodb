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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestUploadProjectFilePostsMultipartGraphQL(t *testing.T) {
	tmp, err := os.CreateTemp(t.TempDir(), "upload-*.csv")
	if err != nil {
		t.Fatalf("CreateTemp() error = %v", err)
	}
	if _, err := tmp.WriteString("id,name\n1,Ada\n"); err != nil {
		t.Fatalf("WriteString() error = %v", err)
	}
	if err := tmp.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/query" {
			t.Fatalf("path = %q, want /api/query", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer access-token" {
			t.Fatalf("Authorization = %q, want bearer token", got)
		}
		if got := r.Header.Get(workspaceOrgHeader); got != "org-1" {
			t.Fatalf("%s = %q, want org-1", workspaceOrgHeader, got)
		}
		if got := r.Header.Get(workspaceProjectHeader); got != "proj-1" {
			t.Fatalf("%s = %q, want proj-1", workspaceProjectHeader, got)
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatalf("ParseMultipartForm() error = %v", err)
		}
		var operations struct {
			Query     string `json:"query"`
			Variables struct {
				ProjectID string  `json:"projectId"`
				FolderID  *string `json:"folderId"`
			} `json:"variables"`
		}
		if err := json.Unmarshal([]byte(r.FormValue("operations")), &operations); err != nil {
			t.Fatalf("unmarshal operations: %v", err)
		}
		if !strings.Contains(operations.Query, "UploadProjectFile") {
			t.Fatalf("query = %q, want UploadProjectFile", operations.Query)
		}
		if operations.Variables.ProjectID != "proj-1" {
			t.Fatalf("projectId = %q, want proj-1", operations.Variables.ProjectID)
		}
		if operations.Variables.FolderID == nil || *operations.Variables.FolderID != "folder-1" {
			t.Fatalf("folderId = %#v, want folder-1", operations.Variables.FolderID)
		}
		if got := r.FormValue("map"); got != `{"0":["variables.file"]}` {
			t.Fatalf("map = %q, want GraphQL multipart file map", got)
		}
		file, _, err := r.FormFile("0")
		if err != nil {
			t.Fatalf("FormFile() error = %v", err)
		}
		defer file.Close()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"UploadProjectFile":{"id":"file-1","projectId":"proj-1","name":"upload.csv","mimeType":"text/csv","sizeBytes":14,"isTabular":true,"uploadedBy":"user-1","createdAt":"now","updatedAt":"now"}}}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "access-token")
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	client.SetWorkspaceContext("org-1", "proj-1")
	folderID := "folder-1"
	file, err := client.UploadProjectFile(context.Background(), "proj-1", &folderID, tmp.Name())
	if err != nil {
		t.Fatalf("UploadProjectFile() error = %v", err)
	}
	if file.ID != "file-1" || file.ProjectID != "proj-1" {
		t.Fatalf("uploaded file = %#v, want decoded file", file)
	}
}
