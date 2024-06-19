package router

import (
	"net/http"
	"path/filepath"

	"github.com/go-chi/chi/v5"
)

func fileServer(r chi.Router) {
	fs := http.FileServer(http.Dir("./build"))
	r.Handle("/static/*", fs)
	r.Handle("/images/*", fs)
	r.Handle("/favicon.ico", fs)
	r.Handle("/asset-manifest.json", fs)
	r.Handle("/manifest.json", fs)
	r.Handle("/logo192.png", fs)
	r.Handle("/logo512.png", fs)
	r.Handle("/robots.txt", fs)
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join("build", "index.html"))
	})
}
