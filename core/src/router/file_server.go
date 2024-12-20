package router

import (
	"embed"
	"io"
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/clidey/whodb/core/src/log"
	"github.com/go-chi/chi/v5"
)

func fileServer(r chi.Router, staticFiles embed.FS) {
	staticFS, err := fs.Sub(staticFiles, "build")
	if err != nil {
		log.Logger.Fatal("Failed to create sub filesystem:", err)
	}

	fs := http.FileServer(http.FS(staticFS))

	r.Handle("/*", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if hasExtension(r.URL.Path) {
			fs.ServeHTTP(w, r)
		} else {
			file, err := staticFS.Open("index.html")
			if err != nil {
				http.Error(w, "index.html not found", http.StatusNotFound)
				log.Logger.Error("Failed to open index.html:", err)
				return
			}
			defer file.Close()

			data, err := io.ReadAll(file)
			if err != nil {
				http.Error(w, "Failed to read index.html", http.StatusInternalServerError)
				log.Logger.Error("Failed to read index.html:", err)
				return
			}

			w.Header().Set("Content-Type", "text/html")
			w.Write(data)
		}
	}))
}

func hasExtension(pathFile string) bool {
	ext := strings.ToLower(path.Ext(pathFile))
	return ext != ""
}
