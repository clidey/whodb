package auth

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/clidey/whodb/core/src"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/env"
	"github.com/clidey/whodb/core/src/types"
	"github.com/zalando/go-keyring"
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

func TestAuthMiddlewareFallsBackToCookieWhenHeaderMissing(t *testing.T) {
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
	req.AddCookie(&http.Cookie{Name: string(AuthKey_Token), Value: token})
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
	if got == nil || got.Username != "alice" {
		t.Fatalf("expected credentials from cookie, got %+v", got)
	}
}

func TestAuthMiddlewareRejectsInvalidTokenEncodings(t *testing.T) {
	t.Run("bad base64", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/query", bytes.NewBufferString(`{"operationName":"Other"}`))
		req.Header.Set("Authorization", "Bearer not-base64")
		rr := httptest.NewRecorder()

		AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})).ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", rr.Code)
		}
	})

	t.Run("bad json", func(t *testing.T) {
		token := base64.StdEncoding.EncodeToString([]byte("{not-json"))
		req := httptest.NewRequest(http.MethodPost, "/api/query", bytes.NewBufferString(`{"operationName":"Other"}`))
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()

		AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})).ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rr.Code)
		}
	})
}

func TestAuthMiddlewareEnforcesAPIGatewayTokenValidation(t *testing.T) {
	origGateway := env.IsAPIGatewayEnabled
	origTokens := env.Tokens
	env.IsAPIGatewayEnabled = true
	env.Tokens = []string{"good"}
	t.Cleanup(func() {
		env.IsAPIGatewayEnabled = origGateway
		env.Tokens = origTokens
	})

	t.Run("missing access token", func(t *testing.T) {
		creds := engine.Credentials{Type: "Postgres", Hostname: "h", Username: "u", Password: "p", Database: "d"}
		payload, _ := json.Marshal(&creds)
		token := base64.StdEncoding.EncodeToString(payload)

		req := httptest.NewRequest(http.MethodPost, "/api/query", bytes.NewBufferString(`{"operationName":"Other"}`))
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()

		AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})).ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rr.Code)
		}
	})

	t.Run("invalid access token", func(t *testing.T) {
		bad := "bad"
		creds := engine.Credentials{Type: "Postgres", Hostname: "h", Username: "u", Password: "p", Database: "d", AccessToken: &bad}
		payload, _ := json.Marshal(&creds)
		token := base64.StdEncoding.EncodeToString(payload)

		req := httptest.NewRequest(http.MethodPost, "/api/query", bytes.NewBufferString(`{"operationName":"Other"}`))
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()

		AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})).ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rr.Code)
		}
	})

	t.Run("valid access token", func(t *testing.T) {
		good := "good"
		creds := engine.Credentials{Type: "Postgres", Hostname: "h", Username: "u", Password: "p", Database: "d", AccessToken: &good}
		payload, _ := json.Marshal(&creds)
		token := base64.StdEncoding.EncodeToString(payload)

		req := httptest.NewRequest(http.MethodPost, "/api/query", bytes.NewBufferString(`{"operationName":"Other"}`))
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()

		var got *engine.Credentials
		AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			got = GetCredentials(r.Context())
			w.WriteHeader(http.StatusOK)
		})).ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}
		if got == nil || got.AccessToken == nil || *got.AccessToken != "good" {
			t.Fatalf("expected credentials to be set on context, got %+v", got)
		}
	})
}

func TestAuthMiddlewareResolvesIDOnlyCredentialsFromProfiles(t *testing.T) {
	origEngine := src.MainEngine
	src.MainEngine = &engine.Engine{}
	t.Cleanup(func() { src.MainEngine = origEngine })

	src.MainEngine.AddLoginProfile(types.DatabaseCredentials{
		CustomId: "profile-1",
		Type:     "Postgres",
		Hostname: "db.local",
		Username: "alice",
		Password: "pw",
		Database: "app",
		Port:     "5432",
		Config:   map[string]string{},
	})

	id := "profile-1"
	creds := engine.Credentials{
		Id:       &id,
		Database: "override",
	}
	payload, _ := json.Marshal(&creds)
	token := base64.StdEncoding.EncodeToString(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/query", bytes.NewBufferString(`{"operationName":"Other"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	var got *engine.Credentials
	AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = GetCredentials(r.Context())
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if got == nil || got.Type != "Postgres" || got.Username != "alice" || got.Database != "override" {
		t.Fatalf("expected resolved credentials with overridden database, got %+v", got)
	}
}

func TestAuthMiddlewareResolvesIDOnlyCredentialsFromKeyring(t *testing.T) {
	keyring.MockInit()
	t.Setenv("WHODB_DESKTOP", "true")

	origEngine := src.MainEngine
	src.MainEngine = &engine.Engine{}
	t.Cleanup(func() { src.MainEngine = origEngine })

	id := "keyring-1"
	stored := &engine.Credentials{
		Type:     "Postgres",
		Hostname: "db.local",
		Username: "alice",
		Password: "pw",
		Database: "app",
	}
	if err := SaveCredentials(id, stored); err != nil {
		t.Fatalf("failed to seed keyring: %v", err)
	}

	requested := engine.Credentials{
		Id:       &id,
		Database: "override",
	}
	payload, _ := json.Marshal(&requested)
	token := base64.StdEncoding.EncodeToString(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/query", bytes.NewBufferString(`{"operationName":"Other"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	var got *engine.Credentials
	AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = GetCredentials(r.Context())
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if got == nil || got.Type != "Postgres" || got.Username != "alice" || got.Database != "override" {
		t.Fatalf("expected resolved credentials with overridden database, got %+v", got)
	}
}

func TestAuthMiddlewareRejectsOversizeBody(t *testing.T) {
	body := bytes.Repeat([]byte("a"), maxRequestBodySize+1)
	req := httptest.NewRequest(http.MethodPost, "/api/query", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})).ServeHTTP(rr, req)
	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d", rr.Code)
	}
}
