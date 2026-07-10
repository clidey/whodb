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

package mcp

import (
	"fmt"
	"strings"

	connresolver "github.com/clidey/whodb/cli/internal/connections"
	dbmgr "github.com/clidey/whodb/cli/internal/database"
)

type connectionResolver struct {
	resolver *connresolver.Resolver
}

func newConnectionResolver(includeSecrets bool) (*connectionResolver, error) {
	resolver, err := connresolver.NewResolver(includeSecrets)
	if err != nil {
		return nil, err
	}
	return &connectionResolver{resolver: resolver}, nil
}

func (r *connectionResolver) Count() int {
	if r == nil {
		return 0
	}
	return r.resolver.Count()
}

func (r *connectionResolver) Resolve(name string) (*dbmgr.Connection, error) {
	conn, _, err := r.resolver.Resolve(name)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func (r *connectionResolver) ResolveOrDefault(name string) (*dbmgr.Connection, error) {
	if strings.TrimSpace(name) != "" {
		return r.Resolve(name)
	}

	infos := r.resolver.ListWithSource()
	switch len(infos) {
	case 0:
		return nil, fmt.Errorf("no database connections available. Add one with: whodb-cli connections add or set WHODB_<DBTYPE> profiles")
	case 1:
		conn := infos[0].Connection
		return new(conn), nil
	default:
		names := make([]string, len(infos))
		for i, info := range infos {
			names[i] = info.Connection.Name
		}
		return nil, fmt.Errorf("multiple connections available (%d). Please specify which one to use. Available: %v", len(names), names)
	}
}

func (r *connectionResolver) ListNames() []string {
	if r == nil {
		return nil
	}

	infos := r.resolver.ListWithSource()
	names := make([]string, 0, len(infos))
	for _, info := range infos {
		names = append(names, info.Connection.Name)
	}
	return names
}

func (r *connectionResolver) ListWithSource() []connresolver.ConnectionSourceInfo {
	if r == nil {
		return nil
	}
	return r.resolver.ListWithSource()
}

// ResolveConnection resolves a connection name to a database.Connection using
// saved connections and environment profiles (for example, WHODB_POSTGRES='[...]' or WHODB_MYSQL_1='{...}').
func ResolveConnection(name string) (*dbmgr.Connection, error) {
	resolver, err := newConnectionResolver(true)
	if err != nil {
		return nil, fmt.Errorf("cannot initialize database manager: %w", err)
	}

	return resolver.Resolve(name)
}

// ListAvailableConnections returns all available connection names from saved
// connections and environment profiles (for example, WHODB_POSTGRES='[...]' or WHODB_MYSQL_1='{...}').
func ListAvailableConnections() ([]string, error) {
	resolver, err := newConnectionResolver(false)
	if err != nil {
		return nil, err
	}

	return resolver.ListNames(), nil
}

// ResolveConnectionOrDefault resolves a connection by name, or returns the default
// connection if name is empty and exactly one connection is available.
// Returns an error if name is empty and zero or multiple connections exist.
func ResolveConnectionOrDefault(name string) (*dbmgr.Connection, error) {
	resolver, err := newConnectionResolver(true)
	if err != nil {
		return nil, err
	}

	return resolver.ResolveOrDefault(name)
}
