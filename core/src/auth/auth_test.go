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

	creds := engine.Credentials{
		Type:     "Postgres",
		Hostname: "db.local",
		Username: "alice",
		Password: "pw",
		Database: "app",
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

func TestIsAllowedPermitsWhitelistedOperations(t *testing.T) {
	body := `{"operationName":"Login","variables":{}}`
	req := httptest.NewRequest(http.MethodPost, "/api/query", strings.NewReader(body))
	if !isAllowed(req, []byte(body)) {
		t.Fatalf("expected Login operation to be allowed without auth")
	}

	getDb := `{"operationName":"GetDatabase","variables":{"type":"Sqlite3"}}`
	req = httptest.NewRequest(http.MethodPost, "/api/query", strings.NewReader(getDb))
	if !isAllowed(req, []byte(getDb)) {
		t.Fatalf("expected GetDatabase for Sqlite3 to be allowed")
	}

	denied := `{"operationName":"GetDatabase","variables":{"type":"Postgres"}}`
	req = httptest.NewRequest(http.MethodPost, "/api/query", strings.NewReader(denied))
	if isAllowed(req, []byte(denied)) {
		t.Fatalf("expected GetDatabase for non-sqlite to be rejected")
	}
}
