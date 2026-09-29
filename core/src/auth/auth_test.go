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
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/clidey/whodb/core/src/common/ssl"
	"github.com/clidey/whodb/core/src/env"
	"github.com/clidey/whodb/core/src/source"
)

func TestIsPublicRouteUsesOnlyTheRequestPath(t *testing.T) {
	publicRequest := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	if !isPublicRoute(publicRequest) {
		t.Fatal("expected a non-API path to be public")
	}

	graphQLRequest := httptest.NewRequest(http.MethodPost, "/api/query", bytes.NewBufferString(`{"query":"query { Version }"}`))
	if isPublicRoute(graphQLRequest) {
		t.Fatal("expected GraphQL authorization to be deferred until gqlgen selects the operation")
	}
}

func TestMergeSourceProfileValuesRejectsSSLPathOverrides(t *testing.T) {
	for _, key := range []string{ssl.KeySSLCACertPath, ssl.KeySSLClientCertPath, ssl.KeySSLClientKeyPath} {
		t.Run(key, func(t *testing.T) {
			base := map[string]string{key: "/trusted/path", "Database": "default"}
			if _, err := MergeSourceProfileValues(base, map[string]string{key: "/attacker/path"}); err == nil {
				t.Fatalf("expected %s override to be rejected", key)
			}

			merged, err := MergeSourceProfileValues(base, map[string]string{key: "/trusted/path", "Database": "override"})
			if err != nil {
				t.Fatalf("expected unchanged path to be accepted: %v", err)
			}
			if merged["Database"] != "override" {
				t.Fatalf("expected ordinary profile override to be retained, got %#v", merged)
			}
		})
	}
}

func TestAuthMiddlewareExtractsCredentialsFromBearer(t *testing.T) {
	originalDev := env.IsDevelopment
	env.IsDevelopment = false
	t.Cleanup(func() {
		env.IsDevelopment = originalDev
	})

	creds := source.Credentials{
		SourceType: "Postgres",
		IsProfile:  true, // A client cannot grant itself server-side file access.
		Values: map[string]string{
			"Hostname": "db.local",
			"Username": "alice",
			"Password": "pw",
			"Database": "app",
		},
	}
	payload, err := json.Marshal(&creds)
	if err != nil {
		t.Fatalf("failed to marshal credentials: %v", err)
	}
	token := base64.StdEncoding.EncodeToString(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/query", strings.NewReader(`{"operationName":"Other"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	var captured *source.Credentials
	handler := AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = GetSourceCredentials(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected request to pass through middleware, got status %d", rr.Code)
	}

	if captured == nil || captured.Values["Username"] != "alice" || captured.Values["Database"] != "app" {
		t.Fatalf("expected credentials to be populated from bearer token, got %+v", captured)
	}
	if captured.IsProfile {
		t.Fatal("client-supplied IsProfile must not be trusted")
	}
}

func TestAuthMiddlewareRejectsMissingTokenForNonGraphQLAPI(t *testing.T) {
	originalDev := env.IsDevelopment
	env.IsDevelopment = false
	t.Cleanup(func() {
		env.IsDevelopment = originalDev
	})

	req := httptest.NewRequest(http.MethodPost, "/api/private", nil)
	rr := httptest.NewRecorder()

	handler := AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized status for missing token, got %d", rr.Code)
	}
}

func TestAuthMiddlewarePreservesAdvancedOptions(t *testing.T) {
	originalDev := env.IsDevelopment
	env.IsDevelopment = false
	t.Cleanup(func() {
		env.IsDevelopment = originalDev
	})

	creds := source.Credentials{
		SourceType: "Postgres",
		Values: map[string]string{
			"Hostname":                "db.local",
			"Username":                "alice",
			"Password":                "pw",
			"Database":                "app",
			"SSL Mode":                "verify-ca",
			"SSL CA Certificate Path": "/path/to/ca.crt",
		},
	}
	payload, err := json.Marshal(&creds)
	if err != nil {
		t.Fatalf("failed to marshal credentials: %v", err)
	}
	token := base64.StdEncoding.EncodeToString(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/query", strings.NewReader(`{"operationName":"Other"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	var captured *source.Credentials
	handler := AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = GetSourceCredentials(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected request to pass through middleware, got status %d", rr.Code)
	}

	if captured == nil {
		t.Fatal("expected credentials to be populated")
	}

	if captured.Values["SSL Mode"] != "verify-ca" {
		t.Fatalf("expected SSL Mode=verify-ca, got %+v", captured.Values)
	}
	if captured.Values["SSL CA Certificate Path"] != "/path/to/ca.crt" {
		t.Fatalf("expected SSL CA Certificate Path to be preserved, got %+v", captured.Values)
	}
}

func TestAuthMiddlewareDecodesSourceCredentialsFormat(t *testing.T) {
	// This test verifies that credentials marshaled from the source-first format
	// are correctly unmarshaled into source.Credentials.
	originalDev := env.IsDevelopment
	env.IsDevelopment = false
	t.Cleanup(func() {
		env.IsDevelopment = originalDev
	})

	// This JSON matches the format produced by the source-first auth payload.
	loginPayload := `{
		"SourceType": "Postgres",
		"Values": {
			"Hostname": "db.local",
			"Username": "alice",
			"Password": "pw",
			"Database": "app",
			"SSL Mode": "verify-ca",
			"SSL CA Certificate Path": "/path/to/ca.crt"
		}
	}`
	token := base64.StdEncoding.EncodeToString([]byte(loginPayload))

	req := httptest.NewRequest(http.MethodPost, "/api/query", strings.NewReader(`{"operationName":"Other"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	var captured *source.Credentials
	handler := AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = GetSourceCredentials(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected request to pass through middleware, got status %d", rr.Code)
	}

	if captured == nil {
		t.Fatal("expected credentials to be populated")
	}

	if captured.Values["SSL Mode"] != "verify-ca" {
		t.Fatalf("expected SSL Mode=verify-ca, got %+v", captured.Values)
	}
	if captured.Values["SSL CA Certificate Path"] != "/path/to/ca.crt" {
		t.Fatalf("expected SSL CA Certificate Path to be preserved, got %+v", captured.Values)
	}
}
