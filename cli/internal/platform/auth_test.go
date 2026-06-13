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

func TestLogoutPostsBearerTokenToAuthHost(t *testing.T) {
	var gotAuthorization string
	authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/auth/revoke-current-session" {
			t.Fatalf("unexpected auth path %q", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want %s", r.Method, http.MethodPost)
		}
		gotAuthorization = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"revoked":true}`))
	}))
	defer authServer.Close()

	platformServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/auth-config" {
			t.Fatalf("unexpected platform path %q", r.URL.Path)
		}
		authURL, err := json.Marshal(authServer.URL)
		if err != nil {
			t.Fatalf("marshal auth URL: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"mothergateUrl":` + string(authURL) + `}`))
	}))
	defer platformServer.Close()

	if err := Logout(context.Background(), platformServer.URL, "access-token"); err != nil {
		t.Fatalf("Logout() error = %v", err)
	}
	if gotAuthorization != "Bearer access-token" {
		t.Fatalf("Authorization = %q, want bearer token", gotAuthorization)
	}
}

func TestIsInvalidGrant(t *testing.T) {
	authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid_grant","error_description":"refresh token expired"}`))
	}))
	defer authServer.Close()

	_, err := postAuth(context.Background(), authServer.URL, "/auth/refresh", map[string]string{})
	if err == nil {
		t.Fatal("postAuth() error = nil, want auth error")
	}
	if !IsInvalidGrant(err) {
		t.Fatalf("IsInvalidGrant(%v) = false, want true", err)
	}
	if strings.Contains(err.Error(), "refresh token expired") {
		t.Fatalf("auth error leaked server description: %q", err.Error())
	}
}
