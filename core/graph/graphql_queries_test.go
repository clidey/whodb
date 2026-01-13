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
	"strings"
	"testing"

	"errors"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/clidey/whodb/core/internal/testutil"
	"github.com/clidey/whodb/core/src/auth"
	"github.com/clidey/whodb/core/src/engine"
)

func TestSchemaQuerySuccess(t *testing.T) {
	mock := testutil.NewPluginMock(engine.DatabaseType("Test"))
	mock.GetAllSchemasFunc = func(*engine.PluginConfig) ([]string, error) { return []string{"public"}, nil }
	setEngineMock(t, mock)

	srv := handler.NewDefaultServer(NewExecutableSchema(Config{Resolvers: &Resolver{}}))
	body, _ := json.Marshal(map[string]any{"query": `query { Schema }`})
	req := httptest.NewRequest(http.MethodPost, "/api/query", bytes.NewBuffer(body))
	req = req.WithContext(context.WithValue(req.Context(), auth.AuthKey_Credentials, &engine.Credentials{Type: "Test"}))
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

func TestDatabaseQueryError(t *testing.T) {
	mock := testutil.NewPluginMock(engine.DatabaseType("Test"))
	mock.GetDatabasesFunc = func(*engine.PluginConfig) ([]string, error) { return nil, errors.New("db error") }
	setEngineMock(t, mock)

	srv := handler.NewDefaultServer(NewExecutableSchema(Config{Resolvers: &Resolver{}}))
	body, _ := json.Marshal(map[string]any{"query": `query { Database(type:"Test") }`})
	req := httptest.NewRequest(http.MethodPost, "/api/query", bytes.NewBuffer(body))
	req = req.WithContext(context.WithValue(req.Context(), auth.AuthKey_Credentials, &engine.Credentials{Type: "Test"}))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "db error") {
		t.Fatalf("expected graphql error for db error, got code=%d body=%s", w.Code, w.Body.String())
	}
}

func TestStorageUnitQuerySuccess(t *testing.T) {
	mock := testutil.NewPluginMock(engine.DatabaseType("Test"))
	mock.GetStorageUnitsFunc = func(*engine.PluginConfig, string) ([]engine.StorageUnit, error) {
		return []engine.StorageUnit{{Name: "users"}}, nil
	}
	setEngineMock(t, mock)

	srv := handler.NewDefaultServer(NewExecutableSchema(Config{Resolvers: &Resolver{}}))
	body, _ := json.Marshal(map[string]any{"query": `query { StorageUnit(schema:"public"){ Name } }`})
	req := httptest.NewRequest(http.MethodPost, "/api/query", bytes.NewBuffer(body))
	req = req.WithContext(context.WithValue(req.Context(), auth.AuthKey_Credentials, &engine.Credentials{Type: "Test"}))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "users") {
		t.Fatalf("expected storage units in response, got code=%d body=%s", w.Code, w.Body.String())
	}
}

func TestRawExecuteQueryError(t *testing.T) {
	mock := testutil.NewPluginMock(engine.DatabaseType("Test"))
	mock.RawExecuteFunc = func(*engine.PluginConfig, string) (*engine.GetRowsResult, error) { return nil, errors.New("raw error") }
	setEngineMock(t, mock)

	srv := handler.NewDefaultServer(NewExecutableSchema(Config{Resolvers: &Resolver{}}))
	body, _ := json.Marshal(map[string]any{"query": `query { RawExecute(query:"select 1"){ Rows } }`})
	req := httptest.NewRequest(http.MethodPost, "/api/query", bytes.NewBuffer(body))
	req = req.WithContext(context.WithValue(req.Context(), auth.AuthKey_Credentials, &engine.Credentials{Type: "Test"}))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "raw error") {
		t.Fatalf("expected raw execute error surfaced, got code=%d body=%s", w.Code, w.Body.String())
	}
}

func TestAIModelQueryMissingAPIKey(t *testing.T) {
	mock := testutil.NewPluginMock(engine.DatabaseType("Test"))
	setEngineMock(t, mock)
	srv := handler.NewDefaultServer(NewExecutableSchema(Config{Resolvers: &Resolver{}}))

	body, _ := json.Marshal(map[string]any{"query": `query { AIModel(modelType:"OpenAI", token:"") }`})
	req := httptest.NewRequest(http.MethodPost, "/api/query", bytes.NewBuffer(body))
	req = req.WithContext(context.WithValue(req.Context(), auth.AuthKey_Credentials, &engine.Credentials{Type: "Test"}))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "API key is required") {
		t.Fatalf("expected API key error, got code=%d body=%s", w.Code, w.Body.String())
	}
}

func TestAIChatQueryError(t *testing.T) {
	mock := testutil.NewPluginMock(engine.DatabaseType("Test"))
	mock.ChatFunc = func(*engine.PluginConfig, string, string, string) ([]*engine.ChatMessage, error) {
		return nil, errors.New("chat failed")
	}
	setEngineMock(t, mock)

	srv := handler.NewDefaultServer(NewExecutableSchema(Config{Resolvers: &Resolver{}}))
	body, _ := json.Marshal(map[string]any{
		"query": `query { AIChat(modelType:"Test", schema:"public", input:{PreviousConversation:"", Query:"hi", Model:"m"}){ Text } }`,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/query", bytes.NewBuffer(body))
	req = req.WithContext(context.WithValue(req.Context(), auth.AuthKey_Credentials, &engine.Credentials{Type: "Test"}))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "chat failed") {
		t.Fatalf("expected chat error surfaced, got code=%d body=%s", w.Code, w.Body.String())
	}
}
