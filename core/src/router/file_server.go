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
	"embed"
	"io"
	"io/fs"
	"net/http"
	"path"
	"strings"

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
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if hasExtension(r.URL.Path) {
			if isHashedAsset(r.URL.Path) {
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
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

func hasExtension(pathFile string) bool {
	ext := strings.ToLower(path.Ext(pathFile))
	return ext != ""
}

func isHashedAsset(urlPath string) bool {
	return strings.Contains(urlPath, "/assets/")
}
