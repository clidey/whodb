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

package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/clidey/whodb/cli/internal/config"
	"github.com/clidey/whodb/cli/internal/connectionopts"
	"github.com/clidey/whodb/cli/internal/docker"
	"github.com/clidey/whodb/cli/internal/tui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var _ = tea.ProgramOption(nil) // Used in RunE

var (
	dbType               string
	host                 string
	port                 int
	username             string
	database             string
	schema               string
	name                 string
	passwordFromStdin    bool
	useDocker            bool
	connectDiscovered    string
	connectSSLMode       string
	connectSSLCA         string
	connectSSLCert       string
	connectSSLKey        string
	connectSSLServerName string
)

var connectCmd = &cobra.Command{
	Use:   "connect",
	Short: "Connect to a database",
	Long: `Connect to a database and start the interactive TUI.

ALPHA WARNING:
  The discovered-resource and cloud-assisted connect flow is still in testing.
  It is not ready for production use yet, and its behavior may still change.
  Mileage may vary.

Usage modes:
  1) Flags path
     Provide --type and --database (optionally --host, --port, --user, --name).
     For databases that need a password, you'll be prompted on a TTY.
     For non-TTY (piped/CI), pass --password and pipe on stdin.
     If you pass --name, the connection is saved for later use.

  2) Docker auto-detection
     Use --docker to detect running database containers and connect.

  3) Discovered cloud resources
     Use --discovered with an ID from cloud connections list.
     If required credentials are still missing, the TUI opens prefilled.

  4) TUI connection form
     If required flags are missing, the interactive connection form opens.
     Docker containers and discovered cloud resources appear automatically in
     the connection list.
`,
	Example: `
  # Open connection form (interactive — shows saved + Docker connections)
  whodb-cli connect

  # Connect to PostgreSQL
  whodb-cli connect --type postgres --host localhost --user alice --database app

  # Connect to SQLite (no password needed)
  whodb-cli connect --type sqlite3 --database ./app.db

  # Auto-detect Docker database containers
  whodb-cli connect --docker

  # Prefill from a discovered cloud resource
  whodb-cli connect --discovered aws-prod-us-west-2/prod-db

  # One-shot connect from a discovered resource
  whodb-cli connect --discovered aws-prod-us-west-2/prod-db --user alice --database app

  # Non-interactive: read password from stdin
  printf "%s\n" "$DB_PASS" | whodb-cli connect --type postgres --host localhost --user alice --database app --password
  whodb-cli connect --type sqlite --host ./app.db --database ./app.db --name app-sqlite

  # Connect with SSL
  whodb-cli connect --type postgres --host localhost --user alice --database app --ssl-mode verify-ca --ssl-ca ./ca.pem`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if useDocker && strings.TrimSpace(connectDiscovered) != "" {
			return fmt.Errorf("--docker cannot be combined with --discovered")
		}

		// --docker: detect running database containers and connect to the first match
		if useDocker {
			containers := docker.DetectContainers()
			if len(containers) == 0 {
				return fmt.Errorf("no running database containers detected (is Docker running?)")
			}
			c := containers[0]
			fmt.Fprintf(os.Stderr, "Detected %d container(s); connecting to %s (%s on port %d)\n", len(containers), c.Name, c.Type, c.Port)
			conn := config.Connection{
				Type:     c.Type,
				Host:     "localhost",
				Port:     c.Port,
				Database: database,
			}
			m := tui.NewMainModelWithConnection(&conn)
			p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
			if _, err := p.Run(); err != nil {
				return fmt.Errorf("error running interactive mode: %w", err)
			}
			return nil
		}

		if discoveredID := strings.TrimSpace(connectDiscovered); discoveredID != "" {
			if strings.TrimSpace(dbType) != "" {
				return fmt.Errorf("--type cannot be used with --discovered")
			}
			if strings.TrimSpace(host) != "" {
				return fmt.Errorf("--host cannot be used with --discovered")
			}
			if port != 0 {
				return fmt.Errorf("--port cannot be used with --discovered")
			}

			writeAlphaWarningBlock(
				cmd.ErrOrStderr(),
				"DISCOVERED CONNECT IS ALPHA / IN TESTING",
				"The discovered-resource and cloud-assisted connect flow is not ready for production use yet.",
				"Expect rough edges, incomplete provider/resource coverage, and behavior changes.",
				"Mileage may vary.",
			)

			conn, err := resolveDiscoveredConnectionPrefill(context.Background(), discoveredID)
			if err != nil {
				return err
			}

			conn, resolvedType, err := mergeConnectionOverrides(conn, name, username, "", database, schema, connectionopts.SSLSettings{
				Mode:           connectSSLMode,
				CAFile:         connectSSLCA,
				ClientCertFile: connectSSLCert,
				ClientKeyFile:  connectSSLKey,
				ServerName:     connectSSLServerName,
			})
			if err != nil {
				return err
			}

			conn, err = normalizeDirectConnection(conn, resolvedType, false)
			if err != nil {
				return err
			}

			if !hasDirectConnectInputs(conn) {
				return runPrefilledConnectionForm(conn)
			}

			if isConnectionFieldRequired(conn.Type, "Password") && strings.TrimSpace(conn.Username) != "" {
				if term.IsTerminal(int(os.Stdin.Fd())) {
					fmt.Fprint(os.Stderr, "Password: ")
					b, err := term.ReadPassword(int(os.Stdin.Fd()))
					fmt.Fprintln(os.Stderr)
					if err == nil {
						conn.Password = string(b)
					}
				} else if passwordFromStdin {
					fi, _ := os.Stdin.Stat()
					if (fi.Mode() & os.ModeCharDevice) == 0 {
						r := bufio.NewReader(os.Stdin)
						line, _ := r.ReadString('\n')
						conn.Password = strings.Trim(line, "\r\n")
					}
				} else {
					return fmt.Errorf("stdin is not a TTY. Use --password and pipe the password on stdin, or run interactively without piping")
				}
			}

			saveName := strings.TrimSpace(name)
			if saveName != "" {
				conn.Name = saveName
				cfg, err := config.LoadConfig()
				if err != nil {
					return fmt.Errorf("error loading config: %w", err)
				}
				cfg.AddConnection(conn)
				if err := cfg.Save(); err != nil {
					return fmt.Errorf("error saving connection: %w", err)
				}
				fmt.Printf("Connection '%s' saved successfully\n", saveName)
			}

			m := tui.NewMainModelWithConnection(&conn)
			p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
			if _, err := p.Run(); err != nil {
				return fmt.Errorf("error running interactive mode: %w", err)
			}
			return nil
		}

		resolvedType, typeKnown := lookupDatabaseType(dbType)
		if dbType != "" && !typeKnown {
			return fmt.Errorf("unsupported database type %q", dbType)
		}

		// If type and database are provided, connect directly.
		// Username is optional for file-based databases (SQLite, DuckDB) and
		// some NoSQL databases (Redis, MongoDB).
		if typeKnown && (database != "" || !isConnectionFieldRequired(resolvedType.ID, "Database")) {
			advanced, err := connectionopts.ApplySSLSettings(resolvedType.ID, nil, connectionopts.SSLSettings{
				Mode:           connectSSLMode,
				CAFile:         connectSSLCA,
				ClientCertFile: connectSSLCert,
				ClientKeyFile:  connectSSLKey,
				ServerName:     connectSSLServerName,
			})
			if err != nil {
				return err
			}

			conn := config.Connection{
				Name:     name,
				Type:     resolvedType.ID,
				Host:     host,
				Port:     port,
				Username: username,
				Database: database,
				Schema:   schema,
				Advanced: advanced,
			}

			conn, err = normalizeDirectConnection(conn, resolvedType, true)
			if err != nil {
				return err
			}
			if !hasDirectConnectInputs(conn) {
				return runPrefilledConnectionForm(conn)
			}

			needsPassword := conn.Username != "" && isConnectionFieldRequired(conn.Type, "Password")
			if needsPassword {
				if term.IsTerminal(int(os.Stdin.Fd())) {
					fmt.Fprint(os.Stderr, "Password: ")
					b, err := term.ReadPassword(int(os.Stdin.Fd()))
					fmt.Fprintln(os.Stderr)
					if err == nil {
						conn.Password = string(b)
					}
				} else {
					if passwordFromStdin {
						fi, _ := os.Stdin.Stat()
						if (fi.Mode() & os.ModeCharDevice) == 0 {
							r := bufio.NewReader(os.Stdin)
							line, _ := r.ReadString('\n')
							conn.Password = strings.Trim(line, "\r\n")
						}
					} else {
						return fmt.Errorf("stdin is not a TTY. Use --password and pipe the password on stdin, or run interactively without piping")
					}
				}
			}

			if name != "" {
				cfg, err := config.LoadConfig()
				if err != nil {
					return fmt.Errorf("error loading config: %w", err)
				}

				cfg.AddConnection(conn)
				if err := cfg.Save(); err != nil {
					return fmt.Errorf("error saving connection: %w", err)
				}
				fmt.Printf("Connection '%s' saved successfully\n", name)
			}

			m := tui.NewMainModelWithConnection(&conn)
			p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())

			if _, err := p.Run(); err != nil {
				return fmt.Errorf("error running interactive mode: %w", err)
			}

			return nil
		}

		// Otherwise, launch TUI with connection form
		m := tui.NewMainModel()
		p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())

		if _, err := p.Run(); err != nil {
			return fmt.Errorf("error running interactive mode: %w", err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(connectCmd)

	connectCmd.Flags().StringVar(&dbType, "type", "", "database type (postgres, mysql, sqlite, duckdb, mongodb, redis, etc.)")
	connectCmd.Flags().StringVar(&host, "host", "", "database host")
	connectCmd.Flags().IntVar(&port, "port", 0, "database port (default depends on database type)")
	connectCmd.Flags().StringVar(&username, "user", "", "database username")
	connectCmd.Flags().StringVar(&database, "database", "", "database name")
	connectCmd.Flags().StringVar(&schema, "schema", "", "preferred schema (PostgreSQL: schema name; MySQL: not needed; MongoDB: not applicable)")
	connectCmd.Flags().StringVar(&name, "name", "", "connection name (save for later use)")
	connectCmd.Flags().BoolVar(&passwordFromStdin, "password", false, "read password from stdin when not using a TTY")
	connectCmd.Flags().BoolVar(&useDocker, "docker", false, "auto-detect running Docker database containers and connect to the first match")
	connectCmd.Flags().StringVar(&connectDiscovered, "discovered", "", "prefill from a discovered cloud resource ID from `cloud connections list`")
	connectCmd.Flags().StringVar(&connectSSLMode, "ssl-mode", "", "SSL mode from the selected database type's supported modes")
	connectCmd.Flags().StringVar(&connectSSLCA, "ssl-ca", "", "path to a CA certificate PEM file")
	connectCmd.Flags().StringVar(&connectSSLCert, "ssl-cert", "", "path to a client certificate PEM file")
	connectCmd.Flags().StringVar(&connectSSLKey, "ssl-key", "", "path to a client private key PEM file")
	connectCmd.Flags().StringVar(&connectSSLServerName, "ssl-server-name", "", "override server name used for SSL hostname verification")

	connectCmd.RegisterFlagCompletionFunc("type", completeDatabaseTypes)
	connectCmd.RegisterFlagCompletionFunc("ssl-mode", completeSSLModes)
}
