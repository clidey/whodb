// Copyright 2025 Clidey, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
		pathHandler = playground.Handler("API Gateway", graphqlEndpointPath())
	}
	router.HandleFunc("/api/query", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" || r.Header.Get("Connection") == "upgrade" {
			server.ServeHTTP(w, r)
		} else if env.IsDevelopment {
			pathHandler.ServeHTTP(w, r)
		}
	})
}

func graphqlEndpointPath() string {
	if env.BasePath == "" {
		return "/api/query"
	}
	return env.BasePath + "/api/query"
}
