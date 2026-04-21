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

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/env"
	"github.com/clidey/whodb/core/src/source"
)

func TestIsPublicRouteAllowsIntrospectionInDev(t *testing.T) {
	originalDev := env.IsDevelopment
	env.IsDevelopment = true
	t.Cleanup(func() {
		env.IsDevelopment = originalDev
	})

	req := httptest.NewRequest(http.MethodPost, "/api/query", bytes.NewBufferString(`{"query":"IntrospectionQuery"}`))
	if !isPublicRoute(req) {
		t.Fatalf("expected introspection query to be treated as public route in development")
	}
}

func TestIsPublicRouteBlocksWhenNotDev(t *testing.T) {
	originalDev := env.IsDevelopment
	env.IsDevelopment = false
	t.Cleanup(func() {
		env.IsDevelopment = originalDev
	})

	req := httptest.NewRequest(http.MethodPost, "/api/query", bytes.NewBufferString(`{"query":"IntrospectionQuery"}`))
	if isPublicRoute(req) {
		t.Fatalf("expected introspection query to require auth outside development")
	}
}

func TestAuthMiddlewareExtractsCredentialsFromBearer(t *testing.T) {
	originalDev := env.IsDevelopment
	originalGateway := env.IsAPIGatewayEnabled
	env.IsDevelopment = false
	env.IsAPIGatewayEnabled = false
	t.Cleanup(func() {
		env.IsDevelopment = originalDev
		env.IsAPIGatewayEnabled = originalGateway
	})

	creds := source.Credentials{
		SourceType: "Postgres",
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

	var captured *engine.Credentials
	handler := AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = GetCredentials(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected request to pass through middleware, got status %d", rr.Code)
	}

	if captured == nil || captured.Username != "alice" || captured.Database != "app" {
		t.Fatalf("expected credentials to be populated from bearer token, got %+v", captured)
	}
}

func TestAuthMiddlewareRejectsMissingToken(t *testing.T) {
	originalDev := env.IsDevelopment
	env.IsDevelopment = false
	t.Cleanup(func() {
		env.IsDevelopment = originalDev
	})

	req := httptest.NewRequest(http.MethodPost, "/api/query", strings.NewReader(`{"operationName":"Other"}`))
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
	originalGateway := env.IsAPIGatewayEnabled
	env.IsDevelopment = false
	env.IsAPIGatewayEnabled = false
	t.Cleanup(func() {
		env.IsDevelopment = originalDev
		env.IsAPIGatewayEnabled = originalGateway
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

	var captured *engine.Credentials
	handler := AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = GetCredentials(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected request to pass through middleware, got status %d", rr.Code)
	}

	if captured == nil {
		t.Fatal("expected credentials to be populated")
	}

	advancedByKey := map[string]string{}
	for _, record := range captured.Advanced {
		advancedByKey[record.Key] = record.Value
	}

	if advancedByKey["Port"] != "5432" {
		t.Fatalf("expected default Postgres port to be injected, got %+v", captured.Advanced)
	}
	if advancedByKey["SSL Mode"] != "verify-ca" {
		t.Fatalf("expected SSL Mode=verify-ca, got %+v", captured.Advanced)
	}
	if advancedByKey["SSL CA Certificate Path"] != "/path/to/ca.crt" {
		t.Fatalf("expected SSL CA Certificate Path to be preserved, got %+v", captured.Advanced)
	}
}

func TestAuthMiddlewareDecodesSourceCredentialsFormat(t *testing.T) {
	// This test verifies that credentials marshaled from the source-first format
	// are correctly unmarshaled into engine.Credentials.
	originalDev := env.IsDevelopment
	originalGateway := env.IsAPIGatewayEnabled
	env.IsDevelopment = false
	env.IsAPIGatewayEnabled = false
	t.Cleanup(func() {
		env.IsDevelopment = originalDev
		env.IsAPIGatewayEnabled = originalGateway
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

	var captured *engine.Credentials
	handler := AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = GetCredentials(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected request to pass through middleware, got status %d", rr.Code)
	}

	if captured == nil {
		t.Fatal("expected credentials to be populated")
	}

	advancedByKey := map[string]string{}
	for _, record := range captured.Advanced {
		advancedByKey[record.Key] = record.Value
	}

	if advancedByKey["Port"] != "5432" {
		t.Fatalf("expected default Postgres port to be injected, got %+v", captured.Advanced)
	}
	if advancedByKey["SSL Mode"] != "verify-ca" {
		t.Fatalf("expected SSL Mode=verify-ca, got %+v", captured.Advanced)
	}
	if advancedByKey["SSL CA Certificate Path"] != "/path/to/ca.crt" {
		t.Fatalf("expected SSL CA Certificate Path to be preserved, got %+v", captured.Advanced)
	}
}

func TestIsAllowedPermitsWhitelistedOperations(t *testing.T) {
	body := `{"operationName":"LoginSource","variables":{}}`
	req := httptest.NewRequest(http.MethodPost, "/api/query", strings.NewReader(body))
	if !isAllowed(req, []byte(body)) {
		t.Fatalf("expected LoginSource operation to be allowed without auth")
	}

	profiles := `{"operationName":"SourceProfiles","variables":{}}`
	req = httptest.NewRequest(http.MethodPost, "/api/query", strings.NewReader(profiles))
	if !isAllowed(req, []byte(profiles)) {
		t.Fatalf("expected SourceProfiles operation to be allowed without auth")
	}

	loginWithProfile := `{"operationName":"LoginWithSourceProfile","variables":{}}`
	req = httptest.NewRequest(http.MethodPost, "/api/query", strings.NewReader(loginWithProfile))
	if !isAllowed(req, []byte(loginWithProfile)) {
		t.Fatalf("expected LoginWithSourceProfile operation to be allowed without auth")
	}

	getDb := `{"operationName":"SourceFieldOptions","variables":{"sourceType":"Sqlite3"}}`
	req = httptest.NewRequest(http.MethodPost, "/api/query", strings.NewReader(getDb))
	if !isAllowed(req, []byte(getDb)) {
		t.Fatalf("expected SourceFieldOptions for Sqlite3 to be allowed")
	}

	denied := `{"operationName":"SourceFieldOptions","variables":{"sourceType":"Postgres"}}`
	req = httptest.NewRequest(http.MethodPost, "/api/query", strings.NewReader(denied))
	if isAllowed(req, []byte(denied)) {
		t.Fatalf("expected SourceFieldOptions for non-sqlite to be rejected")
	}
}

func TestIsAllowedRejectsUpdateSettingsWithoutAuth(t *testing.T) {
	// UpdateSettings mutates process-global runtime state and must require authentication.
	// See https://github.com/clidey/whodb/issues/924
	body := `{"operationName":"UpdateSettings","variables":{"newSettings":{"metricsEnabled":true}}}`
	req := httptest.NewRequest(http.MethodPost, "/api/query", strings.NewReader(body))
	if isAllowed(req, []byte(body)) {
		t.Fatalf("expected UpdateSettings to require authentication")
	}
}

func TestIsAllowedPermitsSettingsConfigWithoutAuth(t *testing.T) {
	// SettingsConfig is a read-only query and should remain publicly accessible so that
	// the frontend can render the login page with the correct configuration.
	body := `{"operationName":"SettingsConfig","variables":{}}`
	req := httptest.NewRequest(http.MethodPost, "/api/query", strings.NewReader(body))
	if !isAllowed(req, []byte(body)) {
		t.Fatalf("expected SettingsConfig query to be allowed without auth")
	}
}
