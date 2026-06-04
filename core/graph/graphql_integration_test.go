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

package graph

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/99designs/gqlgen/graphql/handler"

	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/internal/testutil"
	"github.com/clidey/whodb/core/src"
	"github.com/clidey/whodb/core/src/auth"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/env"
	"github.com/clidey/whodb/core/src/query"
	"github.com/clidey/whodb/core/src/source"
	"github.com/clidey/whodb/core/src/types"
)

func TestGraphQLAddRowMutation(t *testing.T) {
	mock := testutil.NewPluginMock(engine.DatabaseType("Postgres"))
	mock.StorageUnitExistsFunc = func(*engine.PluginConfig, string, string) (bool, error) { return true, nil }
	mock.AddRowFunc = func(*engine.PluginConfig, string, string, []engine.Record) (bool, error) { return true, nil }
	setEngineMock(t, mock)

	srv := handler.NewDefaultServer(NewExecutableSchema(Config{Resolvers: &Resolver{}}))

	query := `
	mutation AddSourceRow($ref: SourceObjectRefInput!) {
		AddSourceRow(ref: $ref, values: [{Key:"id", Value:"1"}]) {
			Status
		}
	}`
	body := map[string]any{
		"query":     query,
		"variables": map[string]any{"ref": map[string]any{"Kind": "Table", "Path": []string{"app", "public", "users"}}},
	}
	payload, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/query", bytes.NewReader(payload))
	req = req.WithContext(testSourceContext("Postgres", map[string]string{"Database": "app"}))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 response, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data struct {
			AddSourceRow model.StatusResponse `json:"AddSourceRow"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if !resp.Data.AddSourceRow.Status {
		t.Fatalf("expected AddSourceRow status true, got %+v", resp.Data.AddSourceRow)
	}
}

func TestGraphQLSourceSessionMetadataQueryReturnsDefaultsWhenNotProvided(t *testing.T) {
	mock := testutil.NewPluginMock(engine.DatabaseType("Postgres"))
	setEngineMock(t, mock)

	srv := handler.NewDefaultServer(NewExecutableSchema(Config{Resolvers: &Resolver{}}))

	query := `query { SourceSessionMetadata { SourceType } }`
	body := map[string]any{"query": query}
	payload, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/query", bytes.NewReader(payload))
	ctx := context.WithValue(req.Context(), auth.AuthKey_Source, &source.Credentials{SourceType: "Postgres"})
	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 response, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data struct {
			SourceSessionMetadata *model.SourceSessionMetadata `json:"SourceSessionMetadata"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.Data.SourceSessionMetadata == nil {
		t.Fatalf("expected source metadata to be returned")
	}
	if resp.Data.SourceSessionMetadata.SourceType != "Postgres" {
		t.Fatalf("expected source type metadata, got %+v", resp.Data.SourceSessionMetadata)
	}
}

func TestGraphQLRowQueryWithSortAndWhere(t *testing.T) {
	mock := testutil.NewPluginMock(engine.DatabaseType("Postgres"))
	mock.StorageUnitExistsFunc = func(*engine.PluginConfig, string, string) (bool, error) { return true, nil }
	mock.GetRowsFunc = func(_ *engine.PluginConfig, req *engine.GetRowsRequest) (*engine.GetRowsResult, error) {
		where, sort := req.Where, req.Sort
		if where == nil || where.Atomic == nil || where.Atomic.Key != "id" || where.Atomic.Operator != "=" || where.Atomic.Value != "1" {
			t.Fatalf("unexpected where clause passed to plugin: %#v", where)
		}
		if len(sort) != 1 || sort[0].Column != "id" || sort[0].Direction != query.SortDirectionAsc {
			t.Fatalf("unexpected sort: %#v", sort)
		}
		return &engine.GetRowsResult{
			Columns: []engine.Column{{Name: "id", Type: "int"}},
			Rows:    [][]string{{"1"}},
		}, nil
	}
	setEngineMock(t, mock)

	srv := handler.NewDefaultServer(NewExecutableSchema(Config{Resolvers: &Resolver{}}))

	query := `
	query SourceRows($ref: SourceObjectRefInput!){
		SourceRows(ref:$ref, where:{Type:Atomic, Atomic:{Key:"id", Operator:"=", Value:"1", ColumnType:"int"}}, sort:[{Column:"id", Direction:ASC}], pageSize:10, pageOffset:0){
			Rows
		}
	}`
	body := map[string]any{
		"query":     query,
		"variables": map[string]any{"ref": map[string]any{"Kind": "Table", "Path": []string{"app", "public", "users"}}},
	}
	payload, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/query", bytes.NewReader(payload))
	req = req.WithContext(testSourceContext("Postgres", map[string]string{"Database": "app"}))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 response, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data struct {
			SourceRows *model.RowsResult `json:"SourceRows"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.Data.SourceRows == nil || len(resp.Data.SourceRows.Rows) != 1 {
		t.Fatalf("expected one row, got %#v body=%s", resp.Data.SourceRows, w.Body.String())
	}
}

func TestGraphQLSourceProfilesQueryUsesEngineProfiles(t *testing.T) {
	origEngine := src.MainEngine
	src.MainEngine = &engine.Engine{}
	src.MainEngine.AddLoginProfile(types.DatabaseCredentials{
		Alias:    "alias",
		Hostname: "db.local",
		Username: "alice",
		Database: "app",
		Type:     "Test",
		Source:   "environment",
	})
	t.Cleanup(func() { src.MainEngine = origEngine })

	srv := handler.NewDefaultServer(NewExecutableSchema(Config{Resolvers: &Resolver{}}))
	query := `query { SourceProfiles { Id DisplayName SourceType Values { Key Value } IsEnvironmentDefined } }`
	body, _ := json.Marshal(map[string]any{"query": query})

	req := httptest.NewRequest(http.MethodPost, "/api/query", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 response, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data struct {
			SourceProfiles []struct {
				ID          string `json:"Id"`
				DisplayName string `json:"DisplayName"`
				SourceType  string `json:"SourceType"`
				Values      []struct {
					Key   string `json:"Key"`
					Value string `json:"Value"`
				} `json:"Values"`
				IsEnvironmentDefined bool `json:"IsEnvironmentDefined"`
			} `json:"SourceProfiles"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	var found bool
	for _, profile := range resp.Data.SourceProfiles {
		if profile.DisplayName == "alias" {
			found = true
			database := ""
			for _, record := range profile.Values {
				if record.Key == "Database" {
					database = record.Value
					break
				}
			}
			if profile.ID == "" || database != "app" || profile.SourceType != "Test" || !profile.IsEnvironmentDefined {
				t.Fatalf("expected merged profile fields to be preserved, got %#v", profile)
			}
		}
	}
	if !found {
		t.Fatalf("expected engine-defined profile to be returned, got %#v", resp.Data.SourceProfiles)
	}
}

func TestGraphQLVersionUsesEnvFallback(t *testing.T) {
	origVersion := env.ApplicationVersion
	env.ApplicationVersion = ""
	t.Cleanup(func() { env.ApplicationVersion = origVersion })

	srv := handler.NewDefaultServer(NewExecutableSchema(Config{Resolvers: &Resolver{}}))
	body, _ := json.Marshal(map[string]any{"query": `query { Version }`})
	req := httptest.NewRequest(http.MethodPost, "/api/query", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 response, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Data struct {
			Version string `json:"Version"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.Data.Version != "development" {
		t.Fatalf("expected development fallback version, got %s", resp.Data.Version)
	}
}
