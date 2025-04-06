package router

import (
	"embed"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/clidey/whodb/core/graph"
	"github.com/clidey/whodb/core/src/auth"
	"github.com/clidey/whodb/core/src/env"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

type OAuthLoginUrl struct {
	Url string `json:"url"`
}

func NewGraphQLServer(es graphql.ExecutableSchema) *handler.Server {
	srv := handler.New(es)

	srv.AddTransport(&transport.Websocket{
		KeepAlivePingInterval: 10 * time.Second,
	})
	srv.AddTransport(&transport.Options{})
	srv.AddTransport(&transport.GET{})
	srv.AddTransport(&transport.POST{})
	srv.AddTransport(&transport.MultipartForm{})

	return srv
}

func setupServer(router *chi.Mux, staticFiles embed.FS) {
	if !env.IsAPIGatewayEnabled {
		fileServer(router, staticFiles)
	}

	server := NewGraphQLServer(graph.NewExecutableSchema(graph.Config{Resolvers: &graph.Resolver{}}))
	server.AddTransport(&transport.Websocket{})
	graph.SetupHTTPServer(router)
	setupPlaygroundHandler(router, server)
}

func setupMiddlewares(router *chi.Mux) {
	allowedOrigins := env.AllowedOrigins
	if len(allowedOrigins) == 0 {
		allowedOrigins = append(allowedOrigins, "https://*", "http://*")
	}

	router.Use(
		middleware.ThrottleBacklog(100, 50, time.Second*5),

		middleware.RequestID,
		middleware.RealIP,
		middleware.Logger,

		middleware.RedirectSlashes,
		middleware.Recoverer,
		middleware.Timeout(30*time.Second),

		cors.Handler(cors.Options{
			AllowedOrigins:   allowedOrigins,
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
			ExposedHeaders:   []string{},
			AllowCredentials: true,
			MaxAge:           300,
		}),

		contextMiddleware,
		auth.AuthMiddleware,
	)
}

func InitializeRouter(staticFiles embed.FS) *chi.Mux {
	router := chi.NewRouter()

	setupMiddlewares(router)
	setupServer(router, staticFiles)

	return router
}
