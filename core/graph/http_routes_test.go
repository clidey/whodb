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
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func preserveHTTPRouteRegistrars(t *testing.T) {
	t.Helper()

	httpRouteRegistrarsMu.Lock()
	original := append([]HTTPRouteRegistrar(nil), httpRouteRegistrars...)
	httpRouteRegistrars = nil
	httpRouteRegistrarsMu.Unlock()

	t.Cleanup(func() {
		httpRouteRegistrarsMu.Lock()
		httpRouteRegistrars = original
		httpRouteRegistrarsMu.Unlock()
	})
}

func TestSetupHTTPServerDoesNotRegisterExtensionRoutesByDefault(t *testing.T) {
	preserveHTTPRouteRegistrars(t)

	router := chi.NewRouter()
	SetupHTTPServer(router)

	req := httptest.NewRequest(http.MethodPost, "/api/agent/stream", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected unregistered extension route to return 404, got %d", rec.Code)
	}
}

func TestRegisterHTTPRoutesAddsExtensionRoutes(t *testing.T) {
	preserveHTTPRouteRegistrars(t)

	RegisterHTTPRoutes(func(router chi.Router) {
		router.Post("/api/test-extension", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusAccepted)
		})
	})

	router := chi.NewRouter()
	SetupHTTPServer(router)

	req := httptest.NewRequest(http.MethodPost, "/api/test-extension", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected registered extension route to return 202, got %d", rec.Code)
	}
}
