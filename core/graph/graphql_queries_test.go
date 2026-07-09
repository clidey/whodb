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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"errors"

	"github.com/99designs/gqlgen/graphql/handler"

	"github.com/clidey/whodb/core/internal/testutil"
	"github.com/clidey/whodb/core/src/engine"
)

func TestSourceObjectsQuerySuccess(t *testing.T) {
	mock := testutil.NewPluginMock(engine.DatabaseType("Postgres"))
	mock.GetAllSchemasFunc = func(*engine.PluginConfig) ([]string, error) { return []string{"public"}, nil }
	setEngineMock(t, mock)

	srv := handler.NewDefaultServer(NewExecutableSchema(Config{Resolvers: &Resolver{}}))
	body, _ := json.Marshal(map[string]any{"query": `query { SourceObjects(parent:{Kind:Database, Path:["app"]}, kinds:[Schema]){ Name } }`})
	req := httptest.NewRequest(http.MethodPost, "/api/query", bytes.NewBuffer(body))
	req = req.WithContext(testSourceContext("Postgres", map[string]string{"Database": "app"}))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "public") {
		t.Fatalf("expected schema list in response, got %s", w.Body.String())
	}
}

func TestSourceFieldOptionsQueryError(t *testing.T) {
	mock := testutil.NewPluginMock(engine.DatabaseType("Postgres"))
	mock.GetDatabasesFunc = func(*engine.PluginConfig) ([]string, error) { return nil, errors.New("db error") }
	setEngineMock(t, mock)

	srv := handler.NewDefaultServer(NewExecutableSchema(Config{Resolvers: &Resolver{}}))
	body, _ := json.Marshal(map[string]any{"query": `query { SourceFieldOptions(sourceType:"Postgres", fieldKey:"Database") }`})
	req := httptest.NewRequest(http.MethodPost, "/api/query", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "db error") {
		t.Fatalf("expected graphql error for db error, got code=%d body=%s", w.Code, w.Body.String())
	}
}

func TestSourceObjectQuerySuccess(t *testing.T) {
	mock := testutil.NewPluginMock(engine.DatabaseType("Postgres"))
	mock.GetStorageUnitsFunc = func(*engine.PluginConfig, string) ([]engine.StorageUnit, error) {
		return []engine.StorageUnit{{Name: "users"}}, nil
	}
	setEngineMock(t, mock)

	srv := handler.NewDefaultServer(NewExecutableSchema(Config{Resolvers: &Resolver{}}))
	body, _ := json.Marshal(map[string]any{"query": `query { SourceObjects(parent:{Kind:Schema, Path:["app","public"]}){ Name } }`})
	req := httptest.NewRequest(http.MethodPost, "/api/query", bytes.NewBuffer(body))
	req = req.WithContext(testSourceContext("Postgres", map[string]string{"Database": "app"}))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "users") {
		t.Fatalf("expected storage units in response, got code=%d body=%s", w.Code, w.Body.String())
	}
}

func TestRunSourceQueryError(t *testing.T) {
	mock := testutil.NewPluginMock(engine.DatabaseType("Postgres"))
	mock.RawExecuteFunc = func(*engine.PluginConfig, string, ...any) (*engine.GetRowsResult, error) {
		return nil, errors.New("raw error")
	}
	setEngineMock(t, mock)

	srv := handler.NewDefaultServer(NewExecutableSchema(Config{Resolvers: &Resolver{}}))
	body, _ := json.Marshal(map[string]any{"query": `query { RunSourceQuery(query:"select 1"){ Rows } }`})
	req := httptest.NewRequest(http.MethodPost, "/api/query", bytes.NewBuffer(body))
	req = req.WithContext(testSourceContext("Postgres", nil))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "raw error") {
		t.Fatalf("expected raw execute error surfaced, got code=%d body=%s", w.Code, w.Body.String())
	}
}

func TestAIModelQueryMissingAPIKey(t *testing.T) {
	mock := testutil.NewPluginMock(engine.DatabaseType("Postgres"))
	setEngineMock(t, mock)
	srv := handler.NewDefaultServer(NewExecutableSchema(Config{Resolvers: &Resolver{}}))

	body, _ := json.Marshal(map[string]any{"query": `query { AIModel(modelType:"OpenAI", token:"") }`})
	req := httptest.NewRequest(http.MethodPost, "/api/query", bytes.NewBuffer(body))
	req = req.WithContext(testSourceContext("Postgres", nil))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "API key is required") {
		t.Fatalf("expected API key error, got code=%d body=%s", w.Code, w.Body.String())
	}
}
