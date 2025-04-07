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

	server := http.FileServer(http.FS(staticFS))

	r.Handle("/*", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if hasExtension(r.URL.Path) {
			server.ServeHTTP(w, r)
		} else {
			file, err := staticFS.Open("index.html")
			if err != nil {
				http.Error(w, "index.html not found", http.StatusNotFound)
				log.Logger.Error("Failed to open index.html:", err)
				return
			}
			defer func(file fs.File) {
				err := file.Close()
				if err != nil {
					log.Logger.Error("Failed to close file:", err)
				}
			}(file)

			data, err := io.ReadAll(file)
			if err != nil {
				http.Error(w, "Failed to read index.html", http.StatusInternalServerError)
				log.Logger.Error("Failed to read index.html:", err)
				return
			}

			w.Header().Set("Content-Type", "text/html")
			_, err = w.Write(data)
			if err != nil {
				return
			}
		}
	}))
}

func hasExtension(pathFile string) bool {
	ext := strings.ToLower(path.Ext(pathFile))
	return ext != ""
}
