package router

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/clidey/whodb/core/graph"
	"github.com/clidey/whodb/core/src/auth"
	"github.com/clidey/whodb/core/src/log"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

const defaultPort = "8080"

type OAuthLoginUrl struct {
	Url string `json:"url"`
}

func setupServer(router *chi.Mux) {
	fileServer(router, "/", http.Dir("./build"))

	server := handler.NewDefaultServer(graph.NewExecutableSchema(graph.Config{Resolvers: &graph.Resolver{}}))
	server.AddTransport(&transport.Websocket{})
	setupPlaygroundHandler(router, server)
}

func setupMiddlewares(router *chi.Mux) {
	router.Use(
		middleware.ThrottleBacklog(10000, 1000, time.Second*5),
		middleware.RequestID,
		middleware.RealIP,
		middleware.Logger,
		middleware.RedirectSlashes,
		middleware.Recoverer,
		middleware.Timeout(10*time.Minute),
		cors.Handler(cors.Options{
			AllowedOrigins:   []string{"https://*", "http://*"},
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

func InitializeRouter() {
	router := chi.NewRouter()

	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	setupMiddlewares(router)
	setupServer(router)

	log.Logger.Infof("connect to 0.0.0.0:%s/ for GraphQL playground", port)

	if err := http.ListenAndServe(fmt.Sprintf(":%v", port), router); err != nil {
		panic(err)
	}
}
