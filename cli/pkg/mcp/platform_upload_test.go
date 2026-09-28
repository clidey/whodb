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
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	platformapi "github.com/clidey/whodb/cli/internal/platform"
)

func TestPlatformUploadReviewSurfaces(t *testing.T) {
	t.Chdir(t.TempDir())
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"./customers.csv", "reports/customer names.csv", "quote\"line\n\t.csv", "~/.ssh/id_rsa", filepath.Join(cwd, "absolute.csv")} {
		t.Run(strconv.Quote(path), func(t *testing.T) {
			client := &fakePlatformClient{}
			session := testPlatformSession(client)
			withPlatformSessionLoader(t, func(context.Context) (*platformToolSession, error) { return session, nil })
			input := PlatformGenericWriteInput{Resource: "file", Action: "upload", Payload: map[string]any{"file_path": path, "folderId": "folder-1"}}
			expectedPath := path
			if !filepath.IsAbs(path) {
				expectedPath = filepath.Join(cwd, path)
			}
			_, output, err := handlePlatformGenericWrite(context.Background(), "platform_action", input, "action", true)
			if err != nil || output.Error != "" || !output.ConfirmationRequired {
				t.Fatalf("prepare upload: %v, %+v", err, output)
			}
			t.Cleanup(func() { consumePendingPlatformAction(output.ConfirmationToken) })
			preview := output.ConfirmationPreview
			if preview == nil {
				t.Fatal("missing preview")
			}
			if preview.Summary != "Upload file "+strconv.Quote(expectedPath) {
				t.Fatalf("summary = %q, want quoted absolute path %q", preview.Summary, expectedPath)
			}
			if preview.Host != session.Host.URL || preview.OrgID != "org-1" || preview.ProjectID != "proj-1" || preview.ProjectName != "Customer" {
				t.Fatalf("missing destination: %+v", preview)
			}
			fields := map[string]PlatformFieldChange{}
			for _, field := range preview.FieldChanges {
				fields[field.Field] = field
			}
			if fields["filePath"].Redacted || fields["filePath"].After["value"] != expectedPath || fields["folderId"].After["value"] != "folder-1" {
				t.Fatalf("incorrect upload fields: %+v", fields)
			}
			if slices.Contains(preview.Changes, "filePath (redacted)") || !slices.Contains(preview.Changes, "filePath") {
				t.Fatalf("inconsistent change labels: %v", preview.Changes)
			}
			spec, variables, err := buildPlatformGenericWrite(session, input, "action")
			if err != nil {
				t.Fatal(err)
			}
			plan := buildPlatformWritePlan(nil, session, spec, variables, "")
			if !reflect.DeepEqual(plan.Preview, preview) || !reflect.DeepEqual(plan.PayloadKeys, preview.Changes) {
				t.Fatalf("write plan differs from confirmation: %+v", plan)
			}
			_, pending, err := HandlePlatformPending(context.Background(), nil, PlatformPendingInput{})
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, item := range pending.Pending {
				if item.Token == output.ConfirmationToken {
					found = true
					if !reflect.DeepEqual(item.Action, *preview) {
						t.Fatalf("pending preview differs: %+v", item.Action)
					}
				}
			}
			if !found || client.mutationName != "" {
				t.Fatalf("pending found = %v, mutation = %q", found, client.mutationName)
			}
		})
	}
}

func TestPlatformUploadPreviewPreservesSensitiveFieldRules(t *testing.T) {
	for _, mutation := range []string{"UploadProjectFile", "UpdateSource"} {
		t.Run(mutation, func(t *testing.T) {
			variables := map[string]any{"filePath": "/private/input.csv", "outputPath": "/private/output", "password": "password-value", "token": "token-value", "content": "private-content"}
			action := &PendingPlatformAction{Mutation: mutation, Variables: variables, Changes: genericWriteChanges(variables)}
			preview := action.Preview()
			for _, field := range preview.FieldChanges {
				if mutation == "UploadProjectFile" && field.Field == "filePath" {
					if field.Redacted || field.After["value"] != variables[field.Field] {
						t.Fatalf("upload source hidden: %+v", field)
					}
				} else if !field.Redacted || field.After["value"] != "[redacted]" {
					t.Fatalf("sensitive field exposed: %+v", field)
				}
			}
			if !slices.Contains(action.Changes, "filePath (redacted)") {
				t.Fatal("preview mutated stored change labels")
			}
		})
	}
	for _, field := range []string{"file_path", "filePath", "outputPath", "password", "token", "content"} {
		if _, err := redactWorkflowPayload(map[string]any{field: "private"}); err == nil {
			t.Fatalf("workflow accepted sensitive field %q", field)
		}
	}
}

func TestPlatformUploadConfirmationTransmitsStoredFile(t *testing.T) {
	ctx := context.Background()
	t.Chdir(t.TempDir())
	const fileName = "synthetic private.txt"
	const content = "SYNTHETIC UPLOAD TEST CONTENT"
	var uploads atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path != "/api/query" || r.Header.Get("Authorization") != "Bearer synthetic-token" {
			t.Errorf("unexpected request destination or credentials")
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
			var request struct{ Query string }
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil || !strings.Contains(request.Query, "PlatformManifest") {
				t.Errorf("unexpected query: %+v, %v", request, err)
				http.Error(w, "invalid query", http.StatusBadRequest)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"PlatformManifest": &platformapi.PlatformManifest{
				Operations: []platformapi.PlatformManifestOperation{{Kind: "Mutation", Name: "UploadProjectFile"}},
			}}})
			return
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Error(err)
			http.Error(w, "invalid multipart", http.StatusBadRequest)
			return
		}
		defer r.MultipartForm.RemoveAll()
		file, header, err := r.FormFile("0")
		if err != nil {
			t.Error(err)
			http.Error(w, "missing file", http.StatusBadRequest)
			return
		}
		defer file.Close()
		body, err := io.ReadAll(file)
		if err != nil || string(body) != content || header.Filename != fileName {
			t.Errorf("unexpected uploaded file: %q, %q, %v", header.Filename, body, err)
		}
		var operations struct{ Variables map[string]any }
		if err := json.Unmarshal([]byte(r.FormValue("operations")), &operations); err != nil {
			t.Error(err)
		}
		if operations.Variables["projectId"] != "proj-1" || operations.Variables["folderId"] != "folder-1" {
			t.Errorf("unexpected upload destination: %+v", operations)
		}
		uploads.Add(1)
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"UploadProjectFile": &platformapi.ProjectFile{ID: "file-1", ProjectID: "proj-1", Name: fileName}}})
	}))
	defer server.Close()
	client, err := platformapi.NewClient(server.URL, "synthetic-token")
	if err != nil {
		t.Fatal(err)
	}
	session := testPlatformSession(client)
	session.Host.URL = server.URL
	withPlatformSessionLoader(t, func(context.Context) (*platformToolSession, error) { return session, nil })
	_, pending, err := handlePlatformGenericWrite(ctx, "platform_action", PlatformGenericWriteInput{
		Resource: "file", Action: "upload", Payload: map[string]any{"file_path": fileName, "folderId": "folder-1"},
	}, "action", true)
	if err != nil || pending.Error != "" || !pending.ConfirmationRequired || uploads.Load() != 0 {
		t.Fatalf("prepare upload: %v, %+v, uploads=%d", err, pending, uploads.Load())
	}
	t.Cleanup(func() { consumePendingPlatformAction(pending.ConfirmationToken) })
	// The file does not exist until after preparation: review must not read its contents.
	if err := os.WriteFile(fileName, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(t.TempDir())
	if err := os.WriteFile(fileName, []byte("WRONG WORKING DIRECTORY"), 0600); err != nil {
		t.Fatal(err)
	}
	_, confirmed, err := HandlePlatformConfirm(ctx, nil, ConfirmInput{Token: pending.ConfirmationToken})
	if err != nil || confirmed.Error != "" || uploads.Load() != 1 {
		t.Fatalf("confirm upload: %v, %+v, uploads=%d", err, confirmed, uploads.Load())
	}
	_, replay, err := HandlePlatformConfirm(ctx, nil, ConfirmInput{Token: pending.ConfirmationToken})
	if err != nil || replay.Error == "" || uploads.Load() != 1 {
		t.Fatalf("replayed upload: %v, %+v, uploads=%d", err, replay, uploads.Load())
	}
}

func TestPlatformUploadConfirmationRejectsChangedScopeAndExpiredToken(t *testing.T) {
	for _, change := range []string{"host", "organization", "project", "expired"} {
		t.Run(change, func(t *testing.T) {
			client := &fakePlatformClient{}
			session := testPlatformSession(client)
			withPlatformSessionLoader(t, func(context.Context) (*platformToolSession, error) { return session, nil })
			_, pending, err := handlePlatformGenericWrite(context.Background(), "platform_action", PlatformGenericWriteInput{
				Resource: "file", Action: "upload", Payload: map[string]any{"file_path": "./synthetic.txt"},
			}, "action", true)
			if err != nil || pending.Error != "" || !pending.ConfirmationRequired {
				t.Fatalf("prepare upload: %v, %+v", err, pending)
			}
			t.Cleanup(func() { consumePendingPlatformAction(pending.ConfirmationToken) })
			switch change {
			case "host":
				session.Host.URL = "https://other.example"
			case "organization":
				session.Host.DefaultOrgID = "other-org"
			case "project":
				session.Host.DefaultProjectID = "other-project"
			case "expired":
				platformPendingMutex.Lock()
				pendingPlatformActions[pending.ConfirmationToken].ExpiresAt = time.Now().Add(-time.Minute)
				platformPendingMutex.Unlock()
			}
			_, confirmed, err := HandlePlatformConfirm(context.Background(), nil, ConfirmInput{Token: pending.ConfirmationToken})
			if err != nil || confirmed.Error == "" || client.mutationName != "" {
				t.Fatalf("unexpected execution: %v, %+v, mutation=%q", err, confirmed, client.mutationName)
			}
		})
	}
}
