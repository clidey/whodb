package router

import (
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/clidey/whodb/core/src/env"
	"github.com/go-chi/chi/v5"
)

func setupPlaygroundHandler(router chi.Router, server *handler.Server) {
	if env.IsDevelopment {
		router.Handle("/", playground.Handler("GraphQL playground", "/query"))
	}
	router.Handle("/query", server)
}
