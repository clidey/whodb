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
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/internal/testutil"
	"github.com/clidey/whodb/core/src/auth"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/env"
	"github.com/go-chi/chi/v5"
)

func TestRESTHandlersAddRowAndGetRows(t *testing.T) {
	mock := testutil.NewPluginMock(engine.DatabaseType("Test"))
	mock.StorageUnitExistsFunc = func(*engine.PluginConfig, string, string) (bool, error) { return true, nil }
	mock.AddRowFunc = func(*engine.PluginConfig, string, string, []engine.Record) (bool, error) { return true, nil }
	mock.GetRowsFunc = func(*engine.PluginConfig, string, string, *model.WhereCondition, []*model.SortCondition, int, int) (*engine.GetRowsResult, error) {
		return &engine.GetRowsResult{
			Columns: []engine.Column{{Name: "id", Type: "int"}},
			Rows:    [][]string{{"1"}},
		}, nil
	}
	setEngineMock(t, mock)

	router := chi.NewRouter()
	SetupHTTPServer(router)

	// Add row via REST
	addPayload := `{"schema":"public","storageUnit":"users","values":[{"Key":"id","Value":"1"}]}`
	addReq := httptest.NewRequest(http.MethodPost, "/api/rows", bytes.NewBufferString(addPayload))
	addReq = addReq.WithContext(context.WithValue(addReq.Context(), auth.AuthKey_Credentials, &engine.Credentials{Type: "Test"}))
	addRec := httptest.NewRecorder()
	router.ServeHTTP(addRec, addReq)
	if addRec.Code != http.StatusOK {
		t.Fatalf("expected add row HTTP 200, got %d (%s)", addRec.Code, addRec.Body.String())
	}

	// Get rows via REST
	rowsReq := httptest.NewRequest(http.MethodGet, "/api/rows?schema=public&storageUnit=users&pageSize=10&pageOffset=0", nil)
	rowsReq = rowsReq.WithContext(context.WithValue(rowsReq.Context(), auth.AuthKey_Credentials, &engine.Credentials{Type: "Test"}))
	rowsRec := httptest.NewRecorder()
	router.ServeHTTP(rowsRec, rowsReq)
	if rowsRec.Code != http.StatusOK {
		t.Fatalf("expected get rows HTTP 200, got %d (%s)", rowsRec.Code, rowsRec.Body.String())
	}
	var parsed engine.GetRowsResult
	if err := json.Unmarshal(rowsRec.Body.Bytes(), &parsed); err != nil {
		t.Fatalf("failed to decode rows response: %v", err)
	}
	if parsed.TotalCount != 0 && len(parsed.Rows) == 0 {
		t.Fatalf("expected rows in response, got %#v", parsed)
	}
}

func TestRESTHandlersHandleErrors(t *testing.T) {
	mock := testutil.NewPluginMock(engine.DatabaseType("Test"))
	mock.GetDatabasesFunc = func(*engine.PluginConfig) ([]string, error) { return nil, errors.New("db error") }
	setEngineMock(t, mock)

	router := chi.NewRouter()
	SetupHTTPServer(router)

	req := httptest.NewRequest(http.MethodGet, "/api/databases?type=Test", nil)
	req = req.WithContext(context.WithValue(req.Context(), auth.AuthKey_Credentials, &engine.Credentials{Type: "Test"}))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 when resolver returns error, got %d", rec.Code)
	}
}

func TestRESTHandlersAIModelsAndChat(t *testing.T) {
	// Spin up a fake Ollama server so the test doesn't depend on a real network
	fakeOllama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"models":[]}`))
	}))
	defer fakeOllama.Close()

	// Point Ollama host/port at the fake server
	// Parse host:port from fakeOllama.URL (format: http://127.0.0.1:PORT)
	// GetOllamaEndpoint constructs http://host:port/api, and Ollama appends /tags for model listing.
	// The fake server handles all paths, so we just need host and port.
	fakeURL := fakeOllama.URL[len("http://"):]
	colonIdx := strings.LastIndex(fakeURL, ":")
	t.Setenv("WHODB_OLLAMA_HOST", fakeURL[:colonIdx])
	t.Setenv("WHODB_OLLAMA_PORT", fakeURL[colonIdx+1:])

	originalCustom := env.CustomModels
	originalCompatKey := env.OpenAICompatibleAPIKey
	originalCompatEndpoint := env.OpenAICompatibleEndpoint
	env.CustomModels = []string{"mixtral"}
	env.OpenAICompatibleAPIKey = "token"
	env.OpenAICompatibleEndpoint = "http://compat.local"
	t.Cleanup(func() {
		env.CustomModels = originalCustom
		env.OpenAICompatibleAPIKey = originalCompatKey
		env.OpenAICompatibleEndpoint = originalCompatEndpoint
	})

	mock := testutil.NewPluginMock(engine.DatabaseType("Test"))
	mock.ChatFunc = func(*engine.PluginConfig, string, string, string) ([]*engine.ChatMessage, error) {
		return nil, errors.New("chat error")
	}
	mock.GetDatabasesFunc = func(*engine.PluginConfig) ([]string, error) { return []string{"db"}, nil }
	setEngineMock(t, mock)

	router := chi.NewRouter()
	SetupHTTPServer(router)

	// AI models
	modelReq := httptest.NewRequest(http.MethodGet, "/api/ai-models?modelType=Ollama&token=", nil)
	modelReq = modelReq.WithContext(context.WithValue(modelReq.Context(), auth.AuthKey_Credentials, &engine.Credentials{Type: "Test"}))
	modelRec := httptest.NewRecorder()
	router.ServeHTTP(modelRec, modelReq)
	if modelRec.Code != http.StatusOK {
		t.Fatalf("expected ai-models to return 200, got %d", modelRec.Code)
	}

	// AI chat
	// AI chat with unsupported type should surface error (no network)
	payload := `{"modelType":"Test","token":"","schema":"public","input":{"PreviousConversation":"","Query":"select","Model":"gpt-4"}}`
	chatReq := httptest.NewRequest(http.MethodPost, "/api/ai-chat", bytes.NewBufferString(payload))
	chatReq = chatReq.WithContext(context.WithValue(chatReq.Context(), auth.AuthKey_Credentials, &engine.Credentials{Type: "Test"}))
	chatRec := httptest.NewRecorder()
	router.ServeHTTP(chatRec, chatReq)
	if chatRec.Code != http.StatusInternalServerError {
		t.Fatalf("expected ai-chat to return 500 when chat fails, got %d (%s)", chatRec.Code, chatRec.Body.String())
	}
}

func TestRESTRawExecutePropagatesErrors(t *testing.T) {
	mock := testutil.NewPluginMock(engine.DatabaseType("Test"))
	mock.RawExecuteFunc = func(*engine.PluginConfig, string) (*engine.GetRowsResult, error) {
		return nil, errors.New("raw error")
	}
	setEngineMock(t, mock)

	router := chi.NewRouter()
	SetupHTTPServer(router)

	body := `{"query":"select 1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/raw-execute", bytes.NewBufferString(body))
	req = req.WithContext(context.WithValue(req.Context(), auth.AuthKey_Credentials, &engine.Credentials{Type: "Test"}))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected raw execute to return 500 on error, got %d", rec.Code)
	}
}

func TestRESTHandlersEnforceAuthForProtectedRoutes(t *testing.T) {
	mock := testutil.NewPluginMock(engine.DatabaseType("Test"))
	mock.GetDatabasesFunc = func(*engine.PluginConfig) ([]string, error) { return nil, errors.New("unauthorized") }
	setEngineMock(t, mock)

	router := chi.NewRouter()
	SetupHTTPServer(router)

	req := httptest.NewRequest(http.MethodGet, "/api/databases?type=Test", nil)
	// Provide empty credentials to avoid panic but simulate unauthorized flow
	req = req.WithContext(context.WithValue(req.Context(), auth.AuthKey_Credentials, &engine.Credentials{Type: "Test"}))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected error when plugin returns unauthorized, got %d", rec.Code)
	}
}

func TestRESTExportSuccessAndFailure(t *testing.T) {
	successMock := testutil.NewPluginMock(engine.DatabaseType("Test"))
	successMock.ExportDataFunc = func(*engine.PluginConfig, string, string, func([]string) error, []map[string]any) error {
		return nil
	}
	setEngineMock(t, successMock)

	router := chi.NewRouter()
	SetupHTTPServer(router)

	payload := `{"schema":"public","storageUnit":"users","rows":[{"id":1}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/export", bytes.NewBufferString(payload))
	req = req.WithContext(context.WithValue(req.Context(), auth.AuthKey_Credentials, &engine.Credentials{Type: "Test"}))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected export to succeed, got %d", rec.Code)
	}

	// Failure case
	failMock := testutil.NewPluginMock(engine.DatabaseType("Test"))
	failMock.ExportDataFunc = func(*engine.PluginConfig, string, string, func([]string) error, []map[string]any) error {
		return errors.New("export failed")
	}
	setEngineMock(t, failMock)

	req = httptest.NewRequest(http.MethodPost, "/api/export", bytes.NewBufferString(payload))
	req = req.WithContext(context.WithValue(req.Context(), auth.AuthKey_Credentials, &engine.Credentials{Type: "Test"}))
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected export failure to return 500, got %d", rec.Code)
	}
}
