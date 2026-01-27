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

	dbmgr "github.com/clidey/whodb/cli/internal/database"
)

// ResolveConnection resolves a connection name to a database.Connection using
// saved connections and environment profiles (for example, WHODB_POSTGRES='[...]' or WHODB_MYSQL_1='{...}').
func ResolveConnection(name string) (*dbmgr.Connection, error) {
	mgr, err := dbmgr.NewManager()
	if err != nil {
		return nil, fmt.Errorf("cannot initialize database manager: %w", err)
	}

	conn, _, err := mgr.ResolveConnection(name)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

// ListAvailableConnections returns all available connection names from saved
// connections and environment profiles (for example, WHODB_POSTGRES='[...]' or WHODB_MYSQL_1='{...}').
func ListAvailableConnections() ([]string, error) {
	mgr, err := dbmgr.NewManager()
	if err != nil {
		return nil, err
	}

	conns := mgr.ListAvailableConnections()
	names := make([]string, 0, len(conns))
	for _, conn := range conns {
		names = append(names, conn.Name)
	}

	return names, nil
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
		return nil, fmt.Errorf("no database connections available. Add one with: whodb-cli connections add or set WHODB_<DBTYPE> profiles")
	case 1:
		return ResolveConnection(conns[0])
	default:
		return nil, fmt.Errorf("multiple connections available (%d). Please specify which one to use. Available: %v", len(conns), conns)
	}
}
