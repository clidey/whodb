package router

import (
	"embed"
	"io"
	"io/fs"
	"net/http"

	"github.com/clidey/whodb/core/src/log"
	"github.com/go-chi/chi/v5"
)

func fileServer(r chi.Router, staticFiles embed.FS) {
	staticFS, err := fs.Sub(staticFiles, "build")
	if err != nil {
		log.Logger.Fatal(err)
	}

	fs := http.FileServer(http.FS(staticFS))

	r.Handle("/static/*", fs)
	r.Handle("/images/*", fs)
	r.Handle("/asset-manifest.json", fs)
	r.Handle("/manifest.json", fs)
	r.Handle("/robots.txt", fs)

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		file, err := staticFS.Open("index.html")
		if err != nil {
			http.Error(w, "index.html not found", http.StatusInternalServerError)
			return
		}
		defer file.Close()

		data, err := io.ReadAll(file)
		if err != nil {
			http.Error(w, "index.html read error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		w.Write(data)
	})
}
