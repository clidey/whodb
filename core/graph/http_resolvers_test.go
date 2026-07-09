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
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/clidey/whodb/core/internal/testutil"
	"github.com/clidey/whodb/core/src/engine"
)

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
