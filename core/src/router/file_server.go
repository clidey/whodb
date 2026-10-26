/*
 * Copyright 2025 Clidey, Inc.
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
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"io"
	"io/fs"
	"net/http"
	"path"
	"strings"
	"sync"

	"github.com/go-chi/chi/v5"

	"github.com/clidey/whodb/core/src/env"
	"github.com/clidey/whodb/core/src/log"
)

const baseHrefPlaceholder = "__WHODB_BASE_HREF__"

func fileServer(r chi.Router, staticFiles embed.FS) {
	staticFS, found := resolveStaticFS(staticFiles)
	if !found {
		// In dev mode (no embedded frontend), skip file serving - frontend is served separately
		log.Warn("No embedded frontend assets found - running in API-only mode (use pnpm start for frontend)")
		return
	}

	baseHref := "/"
	if env.BasePath != "" {
		baseHref = env.BasePath + "/"
	}
	r.Handle("/*", newStaticFileHandler(staticFS, baseHref))
}

func resolveStaticFS(staticFiles fs.FS) (fs.FS, bool) {
	// Support assets embedded under different roots (server: "build", desktop: "frontend/dist").
	// Prefer a root that actually contains index.html. If not found (e.g., during build-time), proceed without fatal.
	candidates := []string{"build", "frontend/dist", "dist", "."}
	for _, base := range candidates {
		var sub fs.FS
		var err error
		if base == "." {
			sub = staticFiles
		} else {
			if sub, err = fs.Sub(staticFiles, base); err != nil {
				continue
			}
		}
		if f, openErr := sub.Open("index.html"); openErr == nil {
			_ = f.Close()
			return sub, true
		}
	}

	return nil, false
}

func hasEmbeddedFrontend(staticFiles fs.FS) bool {
	_, found := resolveStaticFS(staticFiles)
	return found
}

func newStaticFileHandler(staticFS fs.FS, baseHref string) http.Handler {
	server := http.FileServer(http.FS(staticFS))
	var etags sync.Map
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Cache headers are only set for existing files so a 404 for a not-yet-deployed
		// hashed asset is never cached by browsers or the CDN.
		etag, exists := staticFileETag(staticFS, &etags, r.URL.Path)
		if exists || isAssetRequest(r.URL.Path) {
			if exists {
				w.Header().Set("Cache-Control", staticCacheControl(r.URL.Path))
				w.Header().Set("ETag", etag)
			}
			server.ServeHTTP(w, r)
		} else {
			data, err := renderIndexHTML(staticFS, baseHref)
			if err != nil {
				http.Error(w, "index.html not found", http.StatusNotFound)
				log.Error("Failed to render index.html:", err)
				return
			}

			w.Header().Set("Content-Type", "text/html")
			w.Header().Set("Cache-Control", "no-cache")
			_, err = w.Write(data)
			if err != nil {
				return
			}
		}
	})
}

func renderIndexHTML(staticFS fs.FS, baseHref string) ([]byte, error) {
	file, err := staticFS.Open("index.html")
	if err != nil {
		return nil, err
	}
	defer func(file fs.File) {
		closeErr := file.Close()
		if closeErr != nil {
			log.Error("Failed to close file:", closeErr)
		}
	}(file)

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	html := strings.ReplaceAll(string(data), baseHrefPlaceholder, baseHref)
	return []byte(html), nil
}

// assetExtensions are file types that only ever name build output or static
// files, never an app route. A request for one that does not exist stays a 404
// so a browser or CDN is never handed index.html in place of a script or style.
var assetExtensions = map[string]struct{}{
	".js": {}, ".mjs": {}, ".css": {}, ".map": {}, ".json": {}, ".webmanifest": {},
	".txt": {}, ".xml": {}, ".ico": {}, ".png": {}, ".jpg": {}, ".jpeg": {}, ".gif": {},
	".svg": {}, ".webp": {}, ".avif": {}, ".woff": {}, ".woff2": {}, ".ttf": {}, ".otf": {},
	".wasm": {}, ".onnx": {}, ".pdf": {}, ".zip": {}, ".mp4": {}, ".webm": {},
}

// isAssetRequest reports whether a path that is not an existing file should
// still be answered by the file server (as a 404) rather than by the app
// shell. Anything under the build's asset directories or with a known asset
// extension qualifies; a page route whose last segment happens to contain a
// dot, such as /platform/report.v2, falls through to the app so its own
// not-found page renders instead of a bare "404 page not found".
func isAssetRequest(urlPath string) bool {
	if isHashedAsset(urlPath) || strings.Contains(urlPath, "/fonts/") || strings.Contains(urlPath, "/images/") {
		return true
	}
	_, ok := assetExtensions[strings.ToLower(path.Ext(urlPath))]
	return ok
}

func isHashedAsset(urlPath string) bool {
	return strings.Contains(urlPath, "/assets/")
}

// staticCacheControl returns the Cache-Control policy for an existing static file.
// Hashed build assets never change, fonts and images rarely change, and every other
// file keeps a stable name across deploys so browsers must revalidate it.
func staticCacheControl(urlPath string) string {
	switch {
	case isHashedAsset(urlPath):
		return "public, max-age=31536000, immutable"
	case strings.HasPrefix(urlPath, "/fonts/"), strings.HasPrefix(urlPath, "/images/"):
		return "public, max-age=86400"
	default:
		return "no-cache"
	}
}

// staticFileETag returns a content-based weak ETag for a static file, caching it per path.
// Embedded files have no modification time, so without an ETag browsers cannot revalidate
// them and re-download the full file. The ETag is weak because responses may be compressed.
func staticFileETag(staticFS fs.FS, cache *sync.Map, urlPath string) (string, bool) {
	name := strings.TrimPrefix(path.Clean("/"+urlPath), "/")
	if cached, ok := cache.Load(name); ok {
		return cached.(string), true
	}
	data, err := fs.ReadFile(staticFS, name)
	if err != nil {
		return "", false
	}
	sum := sha256.Sum256(data)
	etag := `W/"` + hex.EncodeToString(sum[:16]) + `"`
	cache.Store(name, etag)
	return etag, true
}
