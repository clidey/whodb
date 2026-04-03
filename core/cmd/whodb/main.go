/*
 * Copyright 2026 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	_ "github.com/clidey/whodb/core/src/bamlconfig" // Must be first - sets BAML_LOG before native library loads

	"github.com/clidey/whodb/core/graph"
	"github.com/clidey/whodb/core/src/app"

	// CE plugins — each registers itself via init()
	_ "github.com/clidey/whodb/core/src/plugins/clickhouse"
	_ "github.com/clidey/whodb/core/src/plugins/elasticsearch"
	_ "github.com/clidey/whodb/core/src/plugins/memcached"
	_ "github.com/clidey/whodb/core/src/plugins/mongodb"
	_ "github.com/clidey/whodb/core/src/plugins/mysql"
	_ "github.com/clidey/whodb/core/src/plugins/postgres"
	_ "github.com/clidey/whodb/core/src/plugins/redis"
	_ "github.com/clidey/whodb/core/src/plugins/sqlite3"
)

func main() {
	app.Run(app.AppConfig{
		Schema: graph.NewExecutableSchema(graph.Config{Resolvers: &graph.Resolver{}}),
	}, staticFiles)
}
