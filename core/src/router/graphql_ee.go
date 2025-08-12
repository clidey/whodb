//go:build ee

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
	eegraph "github.com/clidey/whodb/ee/core/graph"
	"github.com/99designs/gqlgen/graphql/handler"
)

// createGraphQLServer creates a GraphQL server for Enterprise Edition
func createGraphQLServer() *handler.Server {
	return NewGraphQLServer(eegraph.NewExecutableSchema(eegraph.Config{Resolvers: eegraph.NewResolver()}))
}