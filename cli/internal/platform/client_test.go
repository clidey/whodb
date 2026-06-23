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
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNormalizeHost(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"default", "", DefaultHost, false},
		{"adds https", "app.whodb.com", "https://app.whodb.com", false},
		{"trims trailing slash", "https://staging.whodb.com/", "https://staging.whodb.com", false},
		{"keeps path prefix", "https://example.com/whodb/", "https://example.com/whodb", false},
		{"allows localhost http", "http://localhost:8080/", "http://localhost:8080", false},
		{"allows ipv4 loopback http", "http://127.0.0.1:8080/", "http://127.0.0.1:8080", false},
		{"allows ipv6 loopback http", "http://[::1]:8080/", "http://[::1]:8080", false},
		{"rejects non-loopback http", "http://staging.whodb.com", "", true},
		{"rejects unsupported scheme", "ftp://example.com", "", true},
		{"rejects missing host", "https://", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeHost(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NormalizeHost(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("NormalizeHost(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestResolveAuthHost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/auth-config" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"mothergateUrl":"http://localhost:4000/"}`))
	}))
	defer server.Close()

	got, err := ResolveAuthHost(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("ResolveAuthHost() error = %v", err)
	}
	if got != "http://localhost:4000" {
		t.Fatalf("ResolveAuthHost() = %q, want %q", got, "http://localhost:4000")
	}
}

func TestResolveAuthHostRequiresMothergateURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	if _, err := ResolveAuthHost(context.Background(), server.URL); err == nil {
		t.Fatalf("ResolveAuthHost() error = nil, want error")
	}
}

func TestMeMatchesPlatformUserSchema(t *testing.T) {
	var request struct {
		Query string `json:"query"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if !strings.Contains(request.Query, "Me") {
			t.Fatalf("query = %q, want Me", request.Query)
		}
		if strings.Contains(request.Query, "orgId") {
			t.Fatalf("query = %q, should not request PlatformUser.orgId", request.Query)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"Me":{"id":"user-1","email":"ada@example.com","displayName":"Ada"}}}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "token")
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	user, err := client.Me(context.Background())
	if err != nil {
		t.Fatalf("Me() error = %v", err)
	}
	if user.ID != "user-1" || user.Email != "ada@example.com" {
		t.Fatalf("user = %#v, want Ada", user)
	}
}

func TestPlatformManifestFetchesContract(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"PlatformManifest":{"platformVersion":"1.2.3","manifestProtocolVersion":"1","generatedAt":"2026-06-13T00:00:00Z","operations":[{"name":"Me","kind":"Query","returns":"PlatformUser","args":[]}],"types":[{"name":"PlatformUser","fields":[{"name":"id","type":"ID","required":true,"list":false}]}]}}}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "token")
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	manifest, err := client.PlatformManifest(context.Background())
	if err != nil {
		t.Fatalf("PlatformManifest() error = %v", err)
	}
	if manifest.PlatformVersion != "1.2.3" || !manifest.HasOperation("Query", "Me") {
		t.Fatalf("manifest = %#v, want version and Me operation", manifest)
	}
	if fields := manifest.SelectFields("PlatformUser", []string{"id", "orgId"}); len(fields) != 1 || fields[0] != "id" {
		t.Fatalf("selected fields = %#v, want only id", fields)
	}
}

func TestRequireOperationBlocksUnsupportedFeature(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "token")
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	client.SetPlatformManifest(&PlatformManifest{
		Operations: []PlatformManifestOperation{
			{Name: "Me", Kind: "Query"},
		},
	})
	_, err = client.SourceConfig(context.Background(), "org-1", "proj-1", "src-1")
	var unsupported UnsupportedFeatureError
	if !errors.As(err, &unsupported) {
		t.Fatalf("SourceConfig() error = %T %v, want UnsupportedFeatureError", err, err)
	}
	if unsupported.Operation != "Query.SourceConfig" {
		t.Fatalf("operation = %q, want Query.SourceConfig", unsupported.Operation)
	}
	if requests != 0 {
		t.Fatalf("requests = %d, want no request for unsupported operation", requests)
	}
}

func TestGraphQLValidationErrorRefreshesManifestAndRetriesOnce(t *testing.T) {
	requests := 0
	refreshes := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.Header().Set("Content-Type", "application/json")
		if requests == 1 {
			_, _ = w.Write([]byte(`{"errors":[{"message":"Cannot query field \"orgId\" on type \"PlatformUser\".","extensions":{"code":"GRAPHQL_VALIDATION_FAILED"}}],"data":null}`))
			return
		}
		_, _ = w.Write([]byte(`{"data":{"MyOrganizations":[{"id":"org-1","name":"Acme","slug":"acme"}]}}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "token")
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	client.SetManifestRefresher(func(ctx context.Context, refreshed *Client) (*PlatformManifest, error) {
		refreshes++
		manifest := &PlatformManifest{
			Operations: []PlatformManifestOperation{{Name: "MyOrganizations", Kind: "Query"}},
		}
		refreshed.SetPlatformManifest(manifest)
		return manifest, nil
	})
	orgs, err := client.Organizations(context.Background())
	if err != nil {
		t.Fatalf("Organizations() error = %v", err)
	}
	if len(orgs) != 1 || orgs[0].ID != "org-1" {
		t.Fatalf("orgs = %#v, want org-1", orgs)
	}
	if requests != 2 || refreshes != 1 {
		t.Fatalf("requests=%d refreshes=%d, want 2 requests and 1 refresh", requests, refreshes)
	}
}

func TestGraphQLValidationErrorDetectsWrappedError(t *testing.T) {
	err := errors.New("other")
	if IsGraphQLValidationError(err) {
		t.Fatal("IsGraphQLValidationError(non-graphql) = true, want false")
	}
	wrapped := errors.Join(GraphQLError{Code: "GRAPHQL_VALIDATION_FAILED", Message: "bad field"})
	if !IsGraphQLValidationError(wrapped) {
		t.Fatal("IsGraphQLValidationError(wrapped validation error) = false, want true")
	}
}

func TestMeUsesManifestFields(t *testing.T) {
	var request struct {
		Query string `json:"query"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if strings.Contains(request.Query, "orgId") {
			t.Fatalf("query = %q, should not request missing orgId", request.Query)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"Me":{"id":"user-1","email":"ada@example.com","displayName":"Ada"}}}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "token")
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	client.SetPlatformManifest(&PlatformManifest{
		Types: []PlatformManifestType{{
			Name: "PlatformUser",
			Fields: []PlatformManifestField{
				{Name: "id", Type: "ID", Required: true},
				{Name: "email", Type: "String", Required: true},
				{Name: "displayName", Type: "String", Required: true},
			},
		}},
	})
	if _, err := client.Me(context.Background()); err != nil {
		t.Fatalf("Me() error = %v", err)
	}
}

func TestProjectSourcesSendsProjectID(t *testing.T) {
	var request struct {
		Query     string         `json:"query"`
		Variables map[string]any `json:"variables"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if !strings.Contains(request.Query, "ProjectSources") {
			t.Fatalf("query = %q, want ProjectSources", request.Query)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"ProjectSources":[{"id":"src-1","projectId":"proj-1","name":"Warehouse","databaseType":"Postgres","createdBy":"Ada","createdAt":"2026-06-02T00:00:00Z"}]}}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "token")
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	sources, err := client.ProjectSources(context.Background(), "org-1", "proj-1")
	if err != nil {
		t.Fatalf("ProjectSources() error = %v", err)
	}
	if request.Variables["projectId"] != "proj-1" {
		t.Fatalf("projectId variable = %#v, want proj-1", request.Variables["projectId"])
	}
	if request.Variables["orgId"] != "org-1" {
		t.Fatalf("orgId variable = %#v, want org-1", request.Variables["orgId"])
	}
	if len(sources) != 1 || sources[0].Name != "Warehouse" {
		t.Fatalf("sources = %#v, want Warehouse source", sources)
	}
}

func TestGraphQLSendsWorkspaceContextHeaders(t *testing.T) {
	var gotOrgID string
	var gotProjectID string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotOrgID = r.Header.Get(workspaceOrgHeader)
		gotProjectID = r.Header.Get(workspaceProjectHeader)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"ProjectSecrets":[]}}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "token")
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	client.SetWorkspaceContext(" org-1 ", " proj-1 ")
	if _, err := client.ProjectSecrets(context.Background(), "proj-1"); err != nil {
		t.Fatalf("ProjectSecrets() error = %v", err)
	}
	if gotOrgID != "org-1" {
		t.Fatalf("%s = %q, want org-1", workspaceOrgHeader, gotOrgID)
	}
	if gotProjectID != "proj-1" {
		t.Fatalf("%s = %q, want proj-1", workspaceProjectHeader, gotProjectID)
	}
}

func TestFunctionsBuildsDynamicSelection(t *testing.T) {
	var request struct {
		Query string `json:"query"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if !strings.Contains(request.Query, "ProjectFunctions") {
			t.Fatalf("query = %q, want ProjectFunctions", request.Query)
		}
		if !strings.Contains(request.Query, "\n    id\n") || !strings.Contains(request.Query, "\n    name\n") {
			t.Fatalf("query = %q, want id and name selections", request.Query)
		}
		if strings.Contains(request.Query, "files") || strings.Contains(request.Query, "content") || strings.Contains(request.Query, "dependencies") {
			t.Fatalf("query = %q, should not request heavy function fields", request.Query)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"ProjectFunctions":[{"id":"fn-1","name":"Enrich"}]}}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "token")
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	functions, err := client.Functions(context.Background(), "proj-1", []string{"id", "name"})
	if err != nil {
		t.Fatalf("Functions() error = %v", err)
	}
	if len(functions) != 1 || functions[0].ID != "fn-1" || functions[0].Name != "Enrich" {
		t.Fatalf("functions = %#v, want minimal function", functions)
	}
}

func TestFunctionBuildsNestedDynamicSelection(t *testing.T) {
	var request struct {
		Query string `json:"query"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if !strings.Contains(request.Query, "FunctionDetail") {
			t.Fatalf("query = %q, want FunctionDetail", request.Query)
		}
		if !strings.Contains(request.Query, "files {") || !strings.Contains(request.Query, "content") {
			t.Fatalf("query = %q, want files content selection", request.Query)
		}
		if strings.Contains(request.Query, "dependencies") {
			t.Fatalf("query = %q, should not request dependencies", request.Query)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"FunctionDetail":{"id":"fn-1","files":[{"id":"file-1","path":"main.py","content":"print(1)"}]}}}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "token")
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	function, err := client.Function(context.Background(), "proj-1", "fn-1", []string{"id", "files"})
	if err != nil {
		t.Fatalf("Function() error = %v", err)
	}
	if function == nil || len(function.Files) != 1 || function.Files[0].Content != "print(1)" {
		t.Fatalf("function = %#v, want file content", function)
	}
}

func TestSourceContentBuildsDynamicSelection(t *testing.T) {
	var request struct {
		Query string `json:"query"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if !strings.Contains(request.Query, "fileName: FileName") || !strings.Contains(request.Query, "sizeBytes: SizeBytes") {
			t.Fatalf("query = %q, want fileName and sizeBytes aliases", request.Query)
		}
		if strings.Contains(request.Query, "text: Text") {
			t.Fatalf("query = %q, should not request text", request.Query)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"PlatformSourceContent":{"fileName":"report.txt","sizeBytes":"42"}}}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "token")
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	content, err := client.SourceContent(context.Background(), "proj-1", "src-1", SourceObjectRefInput{Kind: "File", Path: []string{"report.txt"}}, []string{"fileName", "sizeBytes"})
	if err != nil {
		t.Fatalf("SourceContent() error = %v", err)
	}
	if content == nil || content.FileName != "report.txt" || content.SizeBytes != "42" {
		t.Fatalf("content = %#v, want report metadata", content)
	}
}

func TestFilePreviewBuildsDynamicSelection(t *testing.T) {
	var request struct {
		Query string `json:"query"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if !strings.Contains(request.Query, "\n    mimeType\n") || !strings.Contains(request.Query, "\n    sizeBytes\n") {
			t.Fatalf("query = %q, want mimeType and sizeBytes", request.Query)
		}
		if strings.Contains(request.Query, "textContent") || strings.Contains(request.Query, "tabular") {
			t.Fatalf("query = %q, should not request preview body fields", request.Query)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"FilePreview":{"mimeType":"text/plain","sizeBytes":42}}}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "token")
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	preview, err := client.FilePreview(context.Background(), "proj-1", "file-1", nil, []string{"mimeType", "sizeBytes"})
	if err != nil {
		t.Fatalf("FilePreview() error = %v", err)
	}
	if preview == nil || preview.MIMEType != "text/plain" || preview.SizeBytes != 42 {
		t.Fatalf("preview = %#v, want file metadata", preview)
	}
}

func TestFolderContentsBuildsDynamicSelection(t *testing.T) {
	var request struct {
		Query string `json:"query"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if !strings.Contains(request.Query, "FolderContents") {
			t.Fatalf("query = %q, want FolderContents", request.Query)
		}
		if !strings.Contains(request.Query, "storageUsed") || !strings.Contains(request.Query, "files {") {
			t.Fatalf("query = %q, want storageUsed and files", request.Query)
		}
		if strings.Contains(request.Query, "breadcrumbs") || strings.Contains(request.Query, "folders {") {
			t.Fatalf("query = %q, should not request unselected folder fields", request.Query)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"FolderContents":{"storageUsed":42,"files":[{"id":"file-1","projectId":"proj-1","name":"customers.csv","mimeType":"text/csv","sizeBytes":10,"isTabular":true,"uploadedBy":"ada","createdAt":"2026-06-01T00:00:00Z","updatedAt":"2026-06-01T00:00:00Z"}]}}}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "token")
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	contents, err := client.FolderContents(context.Background(), "proj-1", "", []string{"storageUsed", "files"})
	if err != nil {
		t.Fatalf("FolderContents() error = %v", err)
	}
	if contents == nil || contents.StorageUsed != 42 || len(contents.Files) != 1 {
		t.Fatalf("contents = %#v, want storage with one file", contents)
	}
}

func TestDynamicSelectionFallsBackForUnknownFields(t *testing.T) {
	var request struct {
		Query string `json:"query"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if !strings.Contains(request.Query, "\n    id\n") {
			t.Fatalf("query = %q, want id fallback", request.Query)
		}
		if strings.Contains(request.Query, "bogus") {
			t.Fatalf("query = %q, should not include unknown field", request.Query)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"ProjectFunctions":[{"id":"fn-1"}]}}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "token")
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	if _, err := client.Functions(context.Background(), "proj-1", []string{"bogus"}); err != nil {
		t.Fatalf("Functions() error = %v", err)
	}
}

func TestSourceTypesMapsConnectionFields(t *testing.T) {
	var request struct {
		Query string `json:"query"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if !strings.Contains(request.Query, "SourceTypes") {
			t.Fatalf("query = %q, want SourceTypes", request.Query)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"SourceTypes":[{"id":"Postgres","label":"Postgres","connector":"Postgres","category":"Database","connectionFields":[{"key":"Hostname","kind":"Text","section":"Primary","required":true,"labelKey":"hostName","placeholderKey":"enterHostName","defaultValue":null,"supportsOptions":false},{"key":"Port","kind":"Text","section":"Primary","required":false,"labelKey":"advancedFields.port","placeholderKey":null,"defaultValue":"5432","supportsOptions":false},{"key":"Password","kind":"Password","section":"Primary","required":true,"labelKey":"password","placeholderKey":"enterPassword","defaultValue":null,"supportsOptions":false}]}]}}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "token")
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	types, err := client.SourceTypes(context.Background())
	if err != nil {
		t.Fatalf("SourceTypes() error = %v", err)
	}
	if len(types) != 1 || types[0].ID != "Postgres" {
		t.Fatalf("types = %#v, want Postgres", types)
	}
	if len(types[0].ConnectionFields) != 3 {
		t.Fatalf("connection fields = %#v, want 3 fields", types[0].ConnectionFields)
	}
	port := types[0].ConnectionFields[1]
	if port.DefaultValue == nil || *port.DefaultValue != "5432" {
		t.Fatalf("port default = %#v, want 5432", port.DefaultValue)
	}
	if !types[0].ConnectionFields[2].Required || types[0].ConnectionFields[2].Kind != "Password" {
		t.Fatalf("password field = %#v, want required password", types[0].ConnectionFields[2])
	}
}

func TestCreateSourceMapsAdvancedRecords(t *testing.T) {
	var request struct {
		Query     string         `json:"query"`
		Variables map[string]any `json:"variables"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if !strings.Contains(request.Query, "CreateSource") {
			t.Fatalf("query = %q, want CreateSource", request.Query)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"CreateSource":{"id":"src-1","projectId":"proj-1","name":"Warehouse","databaseType":"Postgres","createdBy":"Ada","createdAt":"2026-06-02T00:00:00Z"}}}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "token")
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	source, err := client.CreateSource(context.Background(), CreateSourceInput{
		OrgID:        "org-1",
		ProjectID:    "proj-1",
		Name:         "Warehouse",
		DatabaseType: "Postgres",
		Hostname:     "db.example.com",
		Port:         "5432",
		Username:     "user",
		Password:     "secret",
		Database:     "analytics",
		Advanced: map[string]string{
			"sslmode":  "require",
			"timezone": "UTC",
		},
	})
	if err != nil {
		t.Fatalf("CreateSource() error = %v", err)
	}
	if source.ID != "src-1" {
		t.Fatalf("source ID = %q, want src-1", source.ID)
	}
	input, ok := request.Variables["input"].(map[string]any)
	if !ok {
		t.Fatalf("input variable = %#v, want object", request.Variables["input"])
	}
	if input["password"] != "secret" {
		t.Fatalf("password variable = %#v, want secret", input["password"])
	}
	if input["orgId"] != "org-1" || input["projectId"] != "proj-1" {
		t.Fatalf("scope variables = %#v, want org/project IDs", input)
	}
	advanced, ok := input["advanced"].([]any)
	if !ok {
		t.Fatalf("advanced variable = %#v, want list", input["advanced"])
	}
	if len(advanced) != 2 {
		t.Fatalf("len(advanced) = %d, want 2", len(advanced))
	}
	first := advanced[0].(map[string]any)
	if first["Key"] != "sslmode" || first["Value"] != "require" {
		t.Fatalf("first advanced record = %#v, want sorted sslmode record", first)
	}
}

func TestSourceConfigMapsAdvancedRecords(t *testing.T) {
	var request struct {
		Query     string         `json:"query"`
		Variables map[string]any `json:"variables"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if !strings.Contains(request.Query, "SourceConfig") {
			t.Fatalf("query = %q, want SourceConfig", request.Query)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"SourceConfig":{"hostname":"db.example.com","port":"5432","username":"user","password":"********","database":"analytics","advanced":[{"key":"sslmode","value":"require"}]}}}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "token")
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	config, err := client.SourceConfig(context.Background(), "org-1", "proj-1", "src-1")
	if err != nil {
		t.Fatalf("SourceConfig() error = %v", err)
	}
	if request.Variables["orgId"] != "org-1" || request.Variables["projectId"] != "proj-1" || request.Variables["sourceId"] != "src-1" {
		t.Fatalf("variables = %#v, want org/project/source IDs", request.Variables)
	}
	if config.Hostname != "db.example.com" || config.Advanced["sslmode"] != "require" {
		t.Fatalf("config = %#v, want mapped source config", config)
	}
}

func TestUpdateSourceMapsFullConfig(t *testing.T) {
	var request struct {
		Query     string         `json:"query"`
		Variables map[string]any `json:"variables"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if !strings.Contains(request.Query, "UpdateSource") {
			t.Fatalf("query = %q, want UpdateSource", request.Query)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"UpdateSource":{"id":"src-1","projectId":"proj-1","name":"Warehouse","databaseType":"Postgres","createdBy":"Ada","createdAt":"2026-06-02T00:00:00Z"}}}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "token")
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	name := "Warehouse"
	_, err = client.UpdateSource(context.Background(), UpdateSourceInput{
		OrgID:     "org-1",
		ProjectID: "proj-1",
		ID:        "src-1",
		Name:      &name,
		Config: &SourceConfig{
			Hostname: "db.example.com",
			Port:     "5432",
			Username: "user",
			Password: "********",
			Database: "analytics",
			Advanced: map[string]string{"sslmode": "require"},
		},
	})
	if err != nil {
		t.Fatalf("UpdateSource() error = %v", err)
	}
	input := request.Variables["input"].(map[string]any)
	if input["orgId"] != "org-1" || input["projectId"] != "proj-1" || input["id"] != "src-1" || input["name"] != "Warehouse" {
		t.Fatalf("input = %#v, want org/project/id/name", input)
	}
	config := input["config"].(map[string]any)
	if config["password"] != "********" {
		t.Fatalf("password = %#v, want masked password passthrough", config["password"])
	}
	advanced := config["advanced"].([]any)
	if len(advanced) != 1 || advanced[0].(map[string]any)["Key"] != "sslmode" {
		t.Fatalf("advanced = %#v, want sslmode record", advanced)
	}
}

func TestTestSourceConnectionMapsValues(t *testing.T) {
	var request struct {
		Query     string         `json:"query"`
		Variables map[string]any `json:"variables"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if !strings.Contains(request.Query, "TestSourceConnection") {
			t.Fatalf("query = %q, want TestSourceConnection", request.Query)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"TestSourceConnection":{"Status":true}}}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "token")
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	err = client.TestSourceConnection(context.Background(), CreateSourceInput{
		DatabaseType: "Postgres",
		Hostname:     "db.example.com",
		Port:         "5432",
		Username:     "user",
		Password:     "secret",
		Database:     "analytics",
		Advanced:     map[string]string{"sslmode": "require"},
	})
	if err != nil {
		t.Fatalf("TestSourceConnection() error = %v", err)
	}
	credentials := request.Variables["credentials"].(map[string]any)
	if credentials["SourceType"] != "Postgres" {
		t.Fatalf("SourceType = %#v, want Postgres", credentials["SourceType"])
	}
	values := credentials["Values"].([]any)
	got := map[string]string{}
	for _, value := range values {
		record := value.(map[string]any)
		got[record["Key"].(string)] = record["Value"].(string)
	}
	if got["Hostname"] != "db.example.com" || got["Password"] != "secret" || got["sslmode"] != "require" {
		t.Fatalf("values = %#v, want host/password/sslmode", got)
	}
}

func TestSourceObjectsMapsParentAndKinds(t *testing.T) {
	var request struct {
		Query     string         `json:"query"`
		Variables map[string]any `json:"variables"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if !strings.Contains(request.Query, "PlatformSourceObjects") {
			t.Fatalf("query = %q, want PlatformSourceObjects", request.Query)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"PlatformSourceObjects":[{"ref":{"kind":"Table","locator":"loc","path":["public","users"]},"kind":"Table","name":"users","path":["public","users"],"hasChildren":false,"actions":["ViewRows"],"metadata":[{"key":"rows","value":"10"}]}]}}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "token")
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	objects, err := client.SourceObjects(context.Background(), "org-1", "proj-1", "src-1", &SourceObjectRefInput{
		Kind: "Schema",
		Path: []string{"public"},
	}, []SourceObjectKind{"Table", "View"}, 50, 10)
	if err != nil {
		t.Fatalf("SourceObjects() error = %v", err)
	}
	parent, ok := request.Variables["parent"].(map[string]any)
	if !ok {
		t.Fatalf("parent variable = %#v, want object", request.Variables["parent"])
	}
	if parent["Kind"] != "Schema" {
		t.Fatalf("parent Kind = %#v, want Schema", parent["Kind"])
	}
	if request.Variables["orgId"] != "org-1" {
		t.Fatalf("orgId variable = %#v, want org-1", request.Variables["orgId"])
	}
	kinds, ok := request.Variables["kinds"].([]any)
	if !ok || len(kinds) != 2 || kinds[0] != "Table" || kinds[1] != "View" {
		t.Fatalf("kinds variable = %#v, want Table/View", request.Variables["kinds"])
	}
	if request.Variables["pageOffset"] != float64(10) {
		t.Fatalf("pageOffset variable = %#v, want 10", request.Variables["pageOffset"])
	}
	if len(objects) != 1 || objects[0].Name != "users" {
		t.Fatalf("objects = %#v, want users object", objects)
	}
}

func TestSourceRowsMapsRefAndPagination(t *testing.T) {
	var request struct {
		Query     string         `json:"query"`
		Variables map[string]any `json:"variables"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if !strings.Contains(request.Query, "PlatformSourceRows") {
			t.Fatalf("query = %q, want PlatformSourceRows", request.Query)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"PlatformSourceRows":{"columns":[{"name":"id","type":"integer","metadataFidelity":"Exact","isPrimary":true,"isForeignKey":false}],"rows":[["1"]],"disableUpdate":false,"totalCount":1}}}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "token")
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	result, err := client.SourceRows(context.Background(), "org-1", "proj-1", "src-1", SourceObjectRefInput{
		Kind: "Table",
		Path: []string{"public", "users"},
	}, 25, 5)
	if err != nil {
		t.Fatalf("SourceRows() error = %v", err)
	}
	ref, ok := request.Variables["ref"].(map[string]any)
	if !ok {
		t.Fatalf("ref variable = %#v, want object", request.Variables["ref"])
	}
	if ref["Kind"] != "Table" {
		t.Fatalf("ref Kind = %#v, want Table", ref["Kind"])
	}
	if request.Variables["orgId"] != "org-1" {
		t.Fatalf("orgId variable = %#v, want org-1", request.Variables["orgId"])
	}
	if request.Variables["pageSize"] != float64(25) || request.Variables["pageOffset"] != float64(5) {
		t.Fatalf("pagination variables = %#v, want pageSize=25 pageOffset=5", request.Variables)
	}
	if result.TotalCount != 1 || len(result.Rows) != 1 || result.Rows[0][0] != "1" {
		t.Fatalf("result = %#v, want one row", result)
	}
}
