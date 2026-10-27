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

package router

import (
	"embed"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/go-chi/chi/v5"
)

// Empty embed.FS — tests skip gracefully if build/ is not available.
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

func TestStaticFileHandlerCacheHeaders(t *testing.T) {
	staticFS := fstest.MapFS{
		"index.html":             {Data: []byte("<html></html>")},
		"assets/index-abc123.js": {Data: []byte("console.log(1)")},
		"fonts/brand/font.woff2": {Data: []byte("font")},
		"images/logo.svg":        {Data: []byte("<svg/>")},
		"sandbox-kernel.js":      {Data: []byte("kernel")},
	}
	handler := newStaticFileHandler(staticFS, "/")

	cases := []struct {
		path         string
		cacheControl string
	}{
		{"/assets/index-abc123.js", "public, max-age=31536000, immutable"},
		{"/fonts/brand/font.woff2", "public, max-age=86400"},
		{"/images/logo.svg", "public, max-age=86400"},
		{"/sandbox-kernel.js", "no-cache"},
	}
	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, tc.path, nil))
			if rr.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d", rr.Code)
			}
			if got := rr.Header().Get("Cache-Control"); got != tc.cacheControl {
				t.Fatalf("expected Cache-Control %q, got %q", tc.cacheControl, got)
			}
			etag := rr.Header().Get("ETag")
			if !strings.HasPrefix(etag, `W/"`) {
				t.Fatalf("expected weak ETag, got %q", etag)
			}

			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			req.Header.Set("If-None-Match", etag)
			rr = httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
			if rr.Code != http.StatusNotModified {
				t.Fatalf("expected 304 for matching ETag, got %d", rr.Code)
			}
		})
	}
}

func TestStaticFileHandlerMissingFileHasNoCacheHeaders(t *testing.T) {
	handler := newStaticFileHandler(fstest.MapFS{"index.html": {Data: []byte("<html></html>")}}, "/")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/assets/index-missing.js", nil))
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
	if got := rr.Header().Get("Cache-Control"); got != "" {
		t.Fatalf("expected no Cache-Control on a missing asset, got %q", got)
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

// A page route whose last segment contains a dot is still a page route: it must
// get the app shell so the frontend's own not-found page renders, while anything
// that names a static asset keeps a real 404.
func TestStaticFileHandlerDottedRouteFallsBackToApp(t *testing.T) {
	handler := newStaticFileHandler(fstest.MapFS{
		"index.html":          {Data: []byte("<html>app</html>")},
		"manifest.json":       {Data: []byte("{}")},
		"assets/index-abc.js": {Data: []byte("console.log(1)")},
		"images/logo.svg":     {Data: []byte("<svg/>")},
	}, "/")

	serve := func(path string) *httptest.ResponseRecorder {
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, path, nil))
		return rr
	}

	for _, route := range []string{"/test/replica-tests/platform/dashboardd.dd", "/platform/report.v2", "/org/project.name/apps"} {
		rr := serve(route)
		if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), "app") {
			t.Fatalf("%s: expected the app shell, got %d %q", route, rr.Code, rr.Body.String())
		}
	}

	for _, asset := range []string{"/assets/index-old.js", "/images/missing.png", "/fonts/missing.woff2", "/sitemap.xml", "/favicon.ico", "/platform/data.json"} {
		if rr := serve(asset); rr.Code != http.StatusNotFound {
			t.Fatalf("%s: expected 404 for a missing asset, got %d", asset, rr.Code)
		}
	}

	for _, existing := range []string{"/manifest.json", "/assets/index-abc.js", "/images/logo.svg"} {
		if rr := serve(existing); rr.Code != http.StatusOK || rr.Header().Get("ETag") == "" {
			t.Fatalf("%s: expected the file with an ETag, got %d", existing, rr.Code)
		}
	}
}
