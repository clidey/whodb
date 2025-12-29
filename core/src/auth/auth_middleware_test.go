package auth

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/env"
)

func TestAuthMiddlewarePrefersAuthorizationHeader(t *testing.T) {
	creds := engine.Credentials{
		Type:     "Postgres",
		Hostname: "db.local",
		Username: "alice",
		Password: "pw",
		Database: "app",
	}
	payload, _ := json.Marshal(&creds)
	token := base64.StdEncoding.EncodeToString(payload)

	body := `{"operationName":"Other"}`
	req := httptest.NewRequest(http.MethodPost, "/api/query", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	var got *engine.Credentials
	handler := AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = GetCredentials(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected request to pass, got status %d", rr.Code)
	}
	if got == nil || got.Username != "alice" || got.Database != "app" {
		t.Fatalf("expected credentials from Authorization header, got %+v", got)
	}
}

func TestAuthMiddlewareRejectsOversizeHeader(t *testing.T) {
	huge := bytes.Repeat([]byte("a"), 20000)
	req := httptest.NewRequest(http.MethodPost, "/api/query", bytes.NewBufferString(`{"operationName":"Other"}`))
	req.Header.Set("Authorization", "Bearer "+base64.StdEncoding.EncodeToString(huge))
	rr := httptest.NewRecorder()

	handler := AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for oversized auth header, got %d", rr.Code)
	}
}

func TestAuthMiddlewareAllowsPublicRouteInDev(t *testing.T) {
	origDev := env.IsDevelopment
	env.IsDevelopment = true
	t.Cleanup(func() { env.IsDevelopment = origDev })

	req := httptest.NewRequest(http.MethodPost, "/graphql", bytes.NewBufferString(`{"query":"IntrospectionQuery"}`))
	rr := httptest.NewRecorder()

	handler := AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected public route to bypass auth, got %d", rr.Code)
	}
}
