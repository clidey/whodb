package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/clidey/whodb/core/src/env"
	"github.com/go-chi/chi/v5"
)

func TestSetupPlaygroundHandlerServesPlaygroundInDev(t *testing.T) {
	originalDev := env.IsDevelopment
	env.IsDevelopment = true
	t.Cleanup(func() { env.IsDevelopment = originalDev })

	r := chi.NewRouter()
	var srv *handler.Server // not used for GET in dev
	setupPlaygroundHandler(r, srv)

	req := httptest.NewRequest(http.MethodGet, "/api/query", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected playground to respond with 200, got %d", rr.Code)
	}
	if len(rr.Body.Bytes()) == 0 {
		t.Fatalf("expected playground HTML content")
	}
}

func TestSetupPlaygroundHandlerNoPlaygroundOutsideDev(t *testing.T) {
	originalDev := env.IsDevelopment
	env.IsDevelopment = false
	t.Cleanup(func() { env.IsDevelopment = originalDev })

	r := chi.NewRouter()
	var srv *handler.Server
	setupPlaygroundHandler(r, srv)

	req := httptest.NewRequest(http.MethodGet, "/api/query", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected handler to respond 200 even without playground, got %d", rr.Code)
	}
}
