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

package mcp

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	dbmgr "github.com/clidey/whodb/cli/internal/database"
)

// ResolveConnection resolves a connection name to a database.Connection using
// the hybrid approach:
//  1. Check environment variable: WHODB_{NAME}_URI (e.g., WHODB_PROD_URI)
//  2. Check saved connections in config
//
// This ensures credentials are never passed through MCP tool parameters.
func ResolveConnection(name string) (*dbmgr.Connection, error) {
	// 1. Check environment variable
	envName := "WHODB_" + strings.ToUpper(strings.ReplaceAll(name, "-", "_")) + "_URI"
	if uri := os.Getenv(envName); uri != "" {
		conn, err := ParseConnectionURI(uri, name)
		if err != nil {
			return nil, fmt.Errorf("invalid %s: %w", envName, err)
		}
		return conn, nil
	}

	// 2. Check saved connections
	mgr, err := dbmgr.NewManager()
	if err != nil {
		return nil, fmt.Errorf("cannot initialize database manager: %w", err)
	}

	conn, err := mgr.GetConnection(name)
	if err != nil {
		return nil, fmt.Errorf("connection %q not found (tried env var %s and saved connections)", name, envName)
	}

	return conn, nil
}

// ParseConnectionURI parses a database connection URI into a Connection struct.
// Supports formats like:
//   - postgres://user:pass@host:5432/dbname?schema=public
//   - mysql://user:pass@host:3306/dbname
//   - sqlite3:///path/to/db.sqlite
func ParseConnectionURI(uri, name string) (*dbmgr.Connection, error) {
	parsed, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("invalid URI format: %w", err)
	}

	conn := &dbmgr.Connection{
		Name: name,
	}

	// Map scheme to database type
	switch strings.ToLower(parsed.Scheme) {
	case "postgres", "postgresql":
		conn.Type = "Postgres"
	case "mysql":
		conn.Type = "MySQL"
	case "mariadb":
		conn.Type = "MariaDB"
	case "sqlite", "sqlite3":
		conn.Type = "SQLite3"
	case "mongodb", "mongo":
		conn.Type = "MongoDB"
	case "redis":
		conn.Type = "Redis"
	case "elasticsearch", "elastic", "es":
		conn.Type = "ElasticSearch"
	case "clickhouse":
		conn.Type = "ClickHouse"
	default:
		return nil, fmt.Errorf("unsupported database scheme: %s", parsed.Scheme)
	}

	// Extract host and port
	conn.Host = parsed.Hostname()
	if portStr := parsed.Port(); portStr != "" {
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return nil, fmt.Errorf("invalid port: %s", portStr)
		}
		conn.Port = port
	} else {
		// Default ports
		conn.Port = defaultPort(conn.Type)
	}

	// Extract username and password
	if parsed.User != nil {
		conn.Username = parsed.User.Username()
		conn.Password, _ = parsed.User.Password()
	}

	// Extract database name from path
	if parsed.Path != "" {
		conn.Database = strings.TrimPrefix(parsed.Path, "/")
	}

	// Extract schema from query params
	if schema := parsed.Query().Get("schema"); schema != "" {
		conn.Schema = schema
	}

	return conn, nil
}

// defaultPort returns the default port for a database type.
func defaultPort(dbType string) int {
	switch dbType {
	case "Postgres":
		return 5432
	case "MySQL", "MariaDB":
		return 3306
	case "MongoDB":
		return 27017
	case "Redis":
		return 6379
	case "ElasticSearch":
		return 9200
	case "ClickHouse":
		return 9000
	default:
		return 0
	}
}

// ListAvailableConnections returns all available connections from both
// environment variables and saved connections.
func ListAvailableConnections() ([]string, error) {
	connections := make(map[string]bool)

	// Check environment variables
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "WHODB_") && strings.HasSuffix(strings.Split(env, "=")[0], "_URI") {
			// Extract name from WHODB_NAME_URI
			parts := strings.Split(env, "=")
			name := strings.TrimPrefix(parts[0], "WHODB_")
			name = strings.TrimSuffix(name, "_URI")
			name = strings.ToLower(strings.ReplaceAll(name, "_", "-"))
			connections[name] = true
		}
	}

	// Check saved connections
	mgr, err := dbmgr.NewManager()
	if err != nil {
		return nil, err
	}

	for _, conn := range mgr.ListConnections() {
		connections[conn.Name] = true
	}

	// Convert to slice
	result := make([]string, 0, len(connections))
	for name := range connections {
		result = append(result, name)
	}

	return result, nil
}

// ResolveConnectionOrDefault resolves a connection by name, or returns the default
// connection if name is empty and exactly one connection is available.
// Returns an error if name is empty and zero or multiple connections exist.
func ResolveConnectionOrDefault(name string) (*dbmgr.Connection, error) {
	if name != "" {
		return ResolveConnection(name)
	}

	// No name provided - check if we can default
	conns, err := ListAvailableConnections()
	if err != nil {
		return nil, err
	}

	switch len(conns) {
	case 0:
		return nil, fmt.Errorf("no database connections available. Add one with: whodb-cli connections add")
	case 1:
		return ResolveConnection(conns[0])
	default:
		return nil, fmt.Errorf("multiple connections available (%d). Please specify which one to use. Available: %v", len(conns), conns)
	}
}

