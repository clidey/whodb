package router

import (
	"embed"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

//go:embed all:build
var testStaticFiles embed.FS

func TestFileServerServesIndexAndAssets(t *testing.T) {
	// Check if test files are available - the fileServer looks for "build" subdirectory with index.html
	buildFS, err := fs.Sub(testStaticFiles, "build")
	if err != nil {
		t.Skip("Build directory not available, skipping file server tests")
	}
	if _, err := buildFS.Open("index.html"); err != nil {
		t.Skip("No index.html in build directory - tests require actual build files")
	}

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
	// Check if test files are available - the fileServer looks for "build" subdirectory with index.html
	buildFS, err := fs.Sub(testStaticFiles, "build")
	if err != nil {
		t.Skip("Build directory not available, skipping file server tests")
	}
	if _, err := buildFS.Open("index.html"); err != nil {
		t.Skip("No index.html in build directory - tests require actual build files")
	}

	r := chi.NewRouter()
	fileServer(r, testStaticFiles)

	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected nested route to serve index.html, got status %d", rr.Code)
	}
}
