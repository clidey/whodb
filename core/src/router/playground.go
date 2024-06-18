package router

import (
	"net/http"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/clidey/whodb/core/src/env"
	"github.com/go-chi/chi/v5"
)

func setupPlaygroundHandler(router chi.Router, server *handler.Server) {
	var pathHandler http.HandlerFunc
	if env.IsDevelopment {
		pathHandler = playground.Handler("API Gateway", "/api/query")
	}
	router.HandleFunc("/api*", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" || r.Header.Get("Connection") == "upgrade" {
			server.ServeHTTP(w, r)
		} else if env.IsDevelopment {
			pathHandler.ServeHTTP(w, r)
		}
	})
}
