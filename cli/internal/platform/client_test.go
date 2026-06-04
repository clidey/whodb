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
	sources, err := client.ProjectSources(context.Background(), "proj-1")
	if err != nil {
		t.Fatalf("ProjectSources() error = %v", err)
	}
	if request.Variables["projectId"] != "proj-1" {
		t.Fatalf("projectId variable = %#v, want proj-1", request.Variables["projectId"])
	}
	if len(sources) != 1 || sources[0].Name != "Warehouse" {
		t.Fatalf("sources = %#v, want Warehouse source", sources)
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
	objects, err := client.SourceObjects(context.Background(), "proj-1", "src-1", &SourceObjectRefInput{
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
	result, err := client.SourceRows(context.Background(), "proj-1", "src-1", SourceObjectRefInput{
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
	if request.Variables["pageSize"] != float64(25) || request.Variables["pageOffset"] != float64(5) {
		t.Fatalf("pagination variables = %#v, want pageSize=25 pageOffset=5", request.Variables)
	}
	if result.TotalCount != 1 || len(result.Rows) != 1 || result.Rows[0][0] != "1" {
		t.Fatalf("result = %#v, want one row", result)
	}
}
