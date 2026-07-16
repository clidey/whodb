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

package auth

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/clidey/whodb/core/src/source"
)

func TestAuthMiddlewareSessionCookieInjectsCredentials(t *testing.T) {
	newTestStore(t)
	token, csrf, _, err := CreateSession(testCredentials(), time.Hour)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/query", bytes.NewBufferString(`{"operationName":"Other"}`))
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: token})
	req.Header.Set(csrfHeaderName, csrf)
	rr := httptest.NewRecorder()

	var got *source.Credentials
	AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = GetSourceCredentials(r.Context())
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if got == nil || got.SourceType != "Postgres" || got.Values["Password"] != "s3cr3t" {
		t.Fatalf("expected credentials from session, got %+v", got)
	}
}

func TestAuthMiddlewareSessionCookieRejectsMissingCSRF(t *testing.T) {
	newTestStore(t)
	token, _, _, err := CreateSession(testCredentials(), time.Hour)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/query", bytes.NewBufferString(`{"operationName":"Other"}`))
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: token})
	rr := httptest.NewRecorder()

	AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for missing CSRF, got %d", rr.Code)
	}
}

func TestAuthMiddlewareSessionCookieInvalidClearsAndRejects(t *testing.T) {
	newTestStore(t)

	req := httptest.NewRequest(http.MethodPost, "/api/query", bytes.NewBufferString(`{"operationName":"Other"}`))
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "does-not-exist"})
	req.Header.Set(csrfHeaderName, "whatever")
	rr := httptest.NewRecorder()

	AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unknown session, got %d", rr.Code)
	}
	// The invalid session cookie should be cleared.
	foundClear := false
	for _, c := range rr.Result().Cookies() {
		if c.Name == sessionCookieName && c.MaxAge < 0 {
			foundClear = true
		}
	}
	if !foundClear {
		t.Fatal("expected session cookie to be cleared on invalid session")
	}
}
