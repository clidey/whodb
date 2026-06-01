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
	"github.com/go-chi/chi/v5"

	"github.com/clidey/whodb/core/src/env"
)

func setupPlaygroundHandler(router chi.Router, server *handler.Server) {
	var pathHandler http.HandlerFunc
	apiQueryPath := "/api/query"
	if env.BasePath != "" {
		apiQueryPath = env.BasePath + apiQueryPath
	}
	if env.IsDevelopment {
		pathHandler = playground.Handler("API Gateway", apiQueryPath)
	}
	router.HandleFunc("/api/query", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost || r.Header.Get("Connection") == "upgrade" {
			server.ServeHTTP(w, r)
		} else if env.IsDevelopment {
			pathHandler.ServeHTTP(w, r)
		}
	})
}
