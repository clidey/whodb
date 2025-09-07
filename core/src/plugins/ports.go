/*
 * Copyright 2025 Clidey, Inc.
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

package plugins

import (
	"github.com/clidey/whodb/core/src/engine"
)

// defaultDatabasePorts maps database systems to their standard default ports
var defaultDatabasePorts = map[engine.DatabaseType]string{
	engine.DatabaseType_MySQL:         "3306",
	engine.DatabaseType_MariaDB:       "3306",
	engine.DatabaseType_Postgres:      "5432",
	engine.DatabaseType_Sqlite3:       "0",    // SQLite is file-based, no port
	engine.DatabaseType_ClickHouse:    "9000", // TCP port (HTTP port is 8123)
	engine.DatabaseType_MongoDB:       "27017",
	engine.DatabaseType_ElasticSearch: "9200", // HTTP port (Transport port is 9300)
	engine.DatabaseType_Redis:         "6379",
}

// additionalPorts holds ports registered by external packages (e.g., enterprise edition)
var additionalPorts = make(map[engine.DatabaseType]string)

// GetDefaultPort returns the default port for a database type
func GetDefaultPort(dbType engine.DatabaseType) (string, bool) {
	// First check standard ports
	if port, ok := defaultDatabasePorts[dbType]; ok {
		return port, true
	}
	
	// Then check additional ports registered by external packages
	if port, ok := additionalPorts[dbType]; ok {
		return port, true
	}
	
	return "", false
}

// RegisterDatabasePort allows external packages to register additional database ports
func RegisterDatabasePort(dbType engine.DatabaseType, port string) {
	additionalPorts[dbType] = port
}