package router

import (
	"embed"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

//go:embed build/*
var testStaticFiles embed.FS

func TestFileServerServesIndexAndAssets(t *testing.T) {
	r := chi.NewRouter()
	fileServer(r, testStaticFiles)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected index.html to be served, got status %d", rr.Code)
	}
	body, _ := io.ReadAll(rr.Body)
	if len(body) == 0 {
		t.Fatalf("expected index.html content in response")
	}

	req = httptest.NewRequest(http.MethodGet, "/app.js", nil)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected static asset to be served, got status %d", rr.Code)
	}
}

func TestFileServerFallsBackToIndexForNestedRoute(t *testing.T) {
	r := chi.NewRouter()
	fileServer(r, testStaticFiles)

	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected nested route to serve index.html, got status %d", rr.Code)
	}
}
