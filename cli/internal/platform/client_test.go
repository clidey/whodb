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
