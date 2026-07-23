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

package cmd

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	cloudruntime "github.com/clidey/whodb/cli/internal/cloud"
	"github.com/clidey/whodb/cli/internal/config"
	"github.com/clidey/whodb/cli/internal/connectionopts"
	"github.com/clidey/whodb/cli/internal/sourcetypes"
	"github.com/clidey/whodb/cli/internal/tui"
	"github.com/clidey/whodb/core/src/source"
)

func resolveDiscoveredConnectionPrefill(ctx context.Context, id string) (config.Connection, error) {
	summary, err := cloudruntime.ResolveConnection(ctx, strings.TrimSpace(id))
	if err != nil {
		return config.Connection{}, err
	}
	return cloudruntime.BuildPrefillConnection(summary)
}

func mergeConnectionOverrides(base config.Connection, name, username, password, database, schema string, sslSettings connectionopts.SSLSettings) (config.Connection, source.TypeSpec, error) {
	conn := base
	if strings.TrimSpace(name) != "" {
		conn.Name = strings.TrimSpace(name)
	}
	if strings.TrimSpace(username) != "" {
		conn.Username = strings.TrimSpace(username)
	}
	if password != "" {
		conn.Password = password
	}
	if strings.TrimSpace(database) != "" {
		conn.Database = strings.TrimSpace(database)
	}
	if strings.TrimSpace(schema) != "" {
		conn.Schema = strings.TrimSpace(schema)
	}

	spec, ok := lookupDatabaseType(conn.Type)
	if !ok {
		return config.Connection{}, source.TypeSpec{}, fmt.Errorf("unsupported database type %q", conn.Type)
	}

	advanced, err := connectionopts.ApplySSLSettings(spec.ID, conn.Advanced, sslSettings)
	if err != nil {
		return config.Connection{}, source.TypeSpec{}, err
	}
	conn.Type = spec.ID
	conn.Advanced = advanced
	return conn, spec, nil
}

func normalizeDirectConnection(conn config.Connection, spec source.TypeSpec, defaultLocalhost bool) (config.Connection, error) {
	if strings.TrimSpace(conn.Host) == "" {
		switch {
		case isFileBasedDatabaseType(conn.Type) && strings.TrimSpace(conn.Database) != "":
			conn.Host = strings.TrimSpace(conn.Database)
		case isConnectionFieldRequired(conn.Type, "Hostname") && defaultLocalhost:
			conn.Host = "localhost"
		}
	}

	if conn.Port == 0 {
		if port, ok := sourcetypes.DefaultPort(spec.ID); ok {
			conn.Port = port
		}
	} else if conn.Port < 1024 || conn.Port > 65535 {
		return config.Connection{}, fmt.Errorf("invalid port number %d: must be between 1024 and 65535 (ports below 1024 are system reserved)", conn.Port)
	}

	return conn, nil
}

func hasDirectConnectInputs(conn config.Connection) bool {
	if isConnectionFieldRequired(conn.Type, "Hostname") && strings.TrimSpace(conn.Host) == "" {
		return false
	}
	if isConnectionFieldRequired(conn.Type, "Username") && strings.TrimSpace(conn.Username) == "" {
		return false
	}
	if isConnectionFieldRequired(conn.Type, "Database") && strings.TrimSpace(conn.Database) == "" {
		return false
	}
	return true
}

func runPrefilledConnectionForm(conn config.Connection) error {
	m := tui.NewMainModelWithConnectionPrefill(&conn)
	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running interactive mode: %w", err)
	}
	return nil
}
