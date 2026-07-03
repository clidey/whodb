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
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/clidey/whodb/core/internal/testutil"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/env"
	"github.com/clidey/whodb/core/src/llm/providers"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestRESTHandlersAddRowAndGetRows(t *testing.T) {
	mock := testutil.NewPluginMock(engine.DatabaseType("Postgres"))
	mock.StorageUnitExistsFunc = func(*engine.PluginConfig, string, string) (bool, error) { return true, nil }
	mock.AddRowFunc = func(*engine.PluginConfig, string, string, []engine.Record) (bool, error) { return true, nil }
	mock.GetRowsFunc = func(*engine.PluginConfig, *engine.GetRowsRequest) (*engine.GetRowsResult, error) {
		return &engine.GetRowsResult{
			Columns: []engine.Column{{Name: "id", Type: "int"}},
			Rows:    [][]string{{"1"}},
		}, nil
	}
	setEngineMock(t, mock)

	router := chi.NewRouter()
	SetupHTTPServer(router)

	// Add row via REST
	addPayload := `{"ref":{"Kind":"Table","Path":["app","public","users"]},"values":[{"Key":"id","Value":"1"}]}`
	addReq := httptest.NewRequest(http.MethodPost, "/api/rows", bytes.NewBufferString(addPayload))
	addReq = addReq.WithContext(testSourceContext("Postgres", map[string]string{"Database": "app"}))
	addRec := httptest.NewRecorder()
	router.ServeHTTP(addRec, addReq)
	if addRec.Code != http.StatusOK {
		t.Fatalf("expected add row HTTP 200, got %d (%s)", addRec.Code, addRec.Body.String())
	}

	// Get rows via REST
	rowsReq := httptest.NewRequest(http.MethodGet, "/api/rows?kind=Table&path=app&path=public&path=users&pageSize=10&pageOffset=0", nil)
	rowsReq = rowsReq.WithContext(testSourceContext("Postgres", map[string]string{"Database": "app"}))
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
	mock := testutil.NewPluginMock(engine.DatabaseType("Postgres"))
	mock.GetDatabasesFunc = func(*engine.PluginConfig) ([]string, error) { return nil, errors.New("db error") }
	setEngineMock(t, mock)

	router := chi.NewRouter()
	SetupHTTPServer(router)

	req := httptest.NewRequest(http.MethodGet, "/api/databases?sourceType=Postgres", nil)
	req = req.WithContext(testSourceContext("Postgres", nil))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 when resolver returns error, got %d", rec.Code)
	}
}

func TestRESTHandlersAIModelsAndChat(t *testing.T) {
	// Avoid httptest.NewServer() to keep this test hermetic in environments where
	// binding a local TCP port is not permitted.
	// The provider HTTP client uses an SSRF-aware transport that does its own
	// dialing, so inject the mock through the provider client factory rather than
	// swapping http.DefaultTransport (which the transport bypasses).
	originalFactory := providers.HTTPClientFactory
	t.Cleanup(func() { providers.HTTPClientFactory = originalFactory })
	providers.HTTPClientFactory = func() *http.Client {
		return &http.Client{
			Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				if r.Method == http.MethodGet && r.URL.Scheme == "http" && r.URL.Host == "ollama.test:11434" && r.URL.Path == "/api/tags" {
					body := io.NopCloser(strings.NewReader(`{"models":[]}`))
					return &http.Response{
						StatusCode: http.StatusOK,
						Header:     http.Header{"Content-Type": []string{"application/json"}},
						Body:       body,
						Request:    r,
					}, nil
				}
				return nil, fmt.Errorf("unexpected http request: %s %s", r.Method, r.URL.String())
			}),
		}
	}

	originalOllamaHost := env.OllamaHost
	originalOllamaPort := env.OllamaPort
	env.OllamaHost = "ollama.test"
	env.OllamaPort = "11434"
	t.Cleanup(func() {
		env.OllamaHost = originalOllamaHost
		env.OllamaPort = originalOllamaPort
	})

	mock := testutil.NewPluginMock(engine.DatabaseType("Postgres"))
	mock.ChatFunc = func(*engine.PluginConfig, string, string, string) ([]*engine.ChatMessage, error) {
		return nil, errors.New("chat error")
	}
	mock.GetDatabasesFunc = func(*engine.PluginConfig) ([]string, error) { return []string{"db"}, nil }
	setEngineMock(t, mock)

	router := chi.NewRouter()
	SetupHTTPServer(router)

	// AI models
	modelReq := httptest.NewRequest(http.MethodGet, "/api/ai-models?modelType=Ollama&token=", nil)
	modelReq = modelReq.WithContext(testSourceContext("Postgres", nil))
	modelRec := httptest.NewRecorder()
	router.ServeHTTP(modelRec, modelReq)
	if modelRec.Code != http.StatusOK {
		t.Fatalf("expected ai-models to return 200, got %d", modelRec.Code)
	}

	// AI chat
	// AI chat with unsupported type should surface error (no network)
	payload := `{"modelType":"Test","token":"","ref":{"Kind":"Schema","Path":["app","public"]},"input":{"PreviousConversation":"","Query":"select","Model":"gpt-4"}}`
	chatReq := httptest.NewRequest(http.MethodPost, "/api/ai-chat", bytes.NewBufferString(payload))
	chatReq = chatReq.WithContext(testSourceContext("Postgres", map[string]string{"Database": "app"}))
	chatRec := httptest.NewRecorder()
	router.ServeHTTP(chatRec, chatReq)
	if chatRec.Code != http.StatusInternalServerError {
		t.Fatalf("expected ai-chat to return 500 when chat fails, got %d (%s)", chatRec.Code, chatRec.Body.String())
	}
}

func TestRESTRawExecutePropagatesErrors(t *testing.T) {
	mock := testutil.NewPluginMock(engine.DatabaseType("Test"))
	mock.RawExecuteFunc = func(*engine.PluginConfig, string, ...any) (*engine.GetRowsResult, error) {
		return nil, errors.New("raw error")
	}
	setEngineMock(t, mock)

	router := chi.NewRouter()
	SetupHTTPServer(router)

	body := `{"query":"select 1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/raw-execute", bytes.NewBufferString(body))
	req = req.WithContext(testSourceContext("Test", nil))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected raw execute to return 500 on error, got %d", rec.Code)
	}
}

func TestRESTHandlersEnforceAuthForProtectedRoutes(t *testing.T) {
	mock := testutil.NewPluginMock(engine.DatabaseType("Postgres"))
	mock.GetDatabasesFunc = func(*engine.PluginConfig) ([]string, error) { return nil, errors.New("unauthorized") }
	setEngineMock(t, mock)

	router := chi.NewRouter()
	SetupHTTPServer(router)

	req := httptest.NewRequest(http.MethodGet, "/api/databases?sourceType=Postgres", nil)
	// Provide empty credentials to avoid panic but simulate unauthorized flow
	req = req.WithContext(testSourceContext("Postgres", nil))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected error when plugin returns unauthorized, got %d", rec.Code)
	}
}

func TestRESTExportSuccessAndFailure(t *testing.T) {
	successMock := testutil.NewPluginMock(engine.DatabaseType("Postgres"))
	successMock.ExportDataFunc = func(*engine.PluginConfig, string, string, func([]string) error, []map[string]any) error {
		return nil
	}
	setEngineMock(t, successMock)

	router := chi.NewRouter()
	SetupHTTPServer(router)

	payload := `{"ref":{"Kind":"Table","Path":["app","public","users"]},"selectedRows":[{"id":1}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/export", bytes.NewBufferString(payload))
	req = req.WithContext(testSourceContext("Postgres", map[string]string{"Database": "app"}))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected export to succeed, got %d", rec.Code)
	}

	unsupportedPayload := `{"ref":{"Kind":"Schema","Path":["app","public"]},"selectedRows":[{"id":1}]}`
	req = httptest.NewRequest(http.MethodPost, "/api/export", bytes.NewBufferString(unsupportedPayload))
	req = req.WithContext(testSourceContext("Postgres", map[string]string{"Database": "app"}))
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "viewing rows is not supported") {
		t.Fatalf("expected export action rejection, got %d (%s)", rec.Code, rec.Body.String())
	}

	// Failure case
	failMock := testutil.NewPluginMock(engine.DatabaseType("Postgres"))
	failMock.ExportDataFunc = func(*engine.PluginConfig, string, string, func([]string) error, []map[string]any) error {
		return errors.New("export failed")
	}
	setEngineMock(t, failMock)

	req = httptest.NewRequest(http.MethodPost, "/api/export", bytes.NewBufferString(payload))
	req = req.WithContext(testSourceContext("Postgres", map[string]string{"Database": "app"}))
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected export failure to return 500, got %d", rec.Code)
	}
}
