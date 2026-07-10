package auth

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zalando/go-keyring"

	"github.com/clidey/whodb/core/src"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/env"
	"github.com/clidey/whodb/core/src/source"
	"github.com/clidey/whodb/core/src/types"
)

func testSourceCredentials(sourceType, hostname, username, password, database string) source.Credentials {
	values := map[string]string{}
	if hostname != "" {
		values["Hostname"] = hostname
	}
	if username != "" {
		values["Username"] = username
	}
	if password != "" {
		values["Password"] = password
	}
	if database != "" {
		values["Database"] = database
	}
	return source.Credentials{
		SourceType: sourceType,
		Values:     values,
	}
}

func TestAuthMiddlewarePrefersAuthorizationHeader(t *testing.T) {
	creds := testSourceCredentials("Postgres", "db.local", "alice", "pw", "app")
	payload, _ := json.Marshal(&creds)
	token := base64.StdEncoding.EncodeToString(payload)

	body := `{"operationName":"Other"}`
	req := httptest.NewRequest(http.MethodPost, "/api/query", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	var got *source.Credentials
	handler := AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = GetSourceCredentials(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected request to pass, got status %d", rr.Code)
	}
	if got == nil || got.Values["Username"] != "alice" || got.Values["Database"] != "app" {
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
	creds := testSourceCredentials("Postgres", "db.local", "alice", "pw", "app")
	payload, _ := json.Marshal(&creds)
	token := base64.StdEncoding.EncodeToString(payload)

	body := `{"operationName":"Other"}`
	req := httptest.NewRequest(http.MethodPost, "/api/query", bytes.NewBufferString(body))
	req.AddCookie(&http.Cookie{Name: string(AuthKey_Token), Value: token})
	rr := httptest.NewRecorder()

	var got *source.Credentials
	handler := AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = GetSourceCredentials(r.Context())
		w.WriteHeader(http.StatusOK)
	}))
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected request to pass, got status %d", rr.Code)
	}
	if got == nil || got.Values["Username"] != "alice" {
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
		Advanced: map[string]string{},
	})

	id := "profile-1"
	creds := source.Credentials{
		ID:     &id,
		Values: map[string]string{"Database": "override"},
	}
	payload, _ := json.Marshal(&creds)
	token := base64.StdEncoding.EncodeToString(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/query", bytes.NewBufferString(`{"operationName":"Other"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	var got *source.Credentials
	AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = GetSourceCredentials(r.Context())
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if got == nil || got.SourceType != "Postgres" || got.Values["Username"] != "alice" || got.Values["Database"] != "override" {
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
	stored := &source.Credentials{
		SourceType: "Postgres",
		Values: map[string]string{
			"Hostname": "db.local",
			"Username": "alice",
			"Password": "pw",
			"Database": "app",
		},
	}
	if err := SaveCredentials(id, stored); err != nil {
		t.Fatalf("failed to seed keyring: %v", err)
	}

	requested := source.Credentials{
		ID:     &id,
		Values: map[string]string{"Database": "override"},
	}
	payload, _ := json.Marshal(&requested)
	token := base64.StdEncoding.EncodeToString(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/query", bytes.NewBufferString(`{"operationName":"Other"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	var got *source.Credentials
	AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = GetSourceCredentials(r.Context())
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if got == nil || got.SourceType != "Postgres" || got.Values["Username"] != "alice" || got.Values["Database"] != "override" {
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
