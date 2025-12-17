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
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/clidey/whodb/cli/internal/config"
	"github.com/clidey/whodb/cli/internal/tui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var _ = tea.ProgramOption(nil) // Used in RunE

var (
	dbType            string
	host              string
	port              int
	username          string
	database          string
	schema            string
	name              string
	passwordFromStdin bool
)

var connectCmd = &cobra.Command{
	Use:   "connect",
	Short: "Connect to a database",
	Long: `Connect to a database and start the interactive TUI.

Usage modes:
  1) Flags (non-form) path
     Provide --type, --user, and --database (optionally --host, --port, --name).
     - On a TTY, you will be prompted for the password with no echo.
     - On non‑TTY (piped/CI), pass --password and pipe a single line on stdin.
       Without --password, the command errors to avoid accidental stdin reads.
     - If you pass --name, the connection (including password) is saved to
       ~/.whodb-cli/config.yaml for later use (e.g. with 'query').

  2) TUI connection form
     If required flags are missing, the interactive connection form opens.
     Fill fields (including the masked password) and press Connect.
     If you provide a Name, the connection is saved for reuse.
`,
	Example: `
  # Open connection form (interactive)
  whodb-cli connect

  # Flags path with TTY password prompt
  whodb-cli connect --type postgres --host localhost --user alice --database app --name app-local

  # Non-interactive: read password from stdin (note the --password flag)
  printf "%s\n" "$DB_PASS" | whodb-cli connect --type postgres --host localhost --user alice --database app --name app-local --password

  # SQLite example (no password)
  whodb-cli connect --type sqlite --host ./app.db --database ./app.db --name app-sqlite`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// If all required parameters are provided, connect directly
		if dbType != "" && username != "" && database != "" {
			// Use defaults if not provided
			if host == "" {
				host = "localhost"
			}
			if port == 0 {
				port = getDefaultPort(dbType)
			} else if port < 1024 || port > 65535 {
				return fmt.Errorf("invalid port number %d: must be between 1024 and 65535 (ports below 1024 are system reserved)", port)
			}

			// Normalize database type to match plugin names
			normalizedType := normalizeDBType(dbType)

			// Secure password prompt when using flags interactively
			var password string
			if term.IsTerminal(int(os.Stdin.Fd())) {
				fmt.Fprint(os.Stderr, "Password: ")
				b, err := term.ReadPassword(int(os.Stdin.Fd()))
				fmt.Fprintln(os.Stderr)
				if err == nil {
					password = string(b)
				}
			} else {
				// Non-TTY: only read from stdin when --password is provided
				if passwordFromStdin {
					fi, _ := os.Stdin.Stat()
					if (fi.Mode() & os.ModeCharDevice) == 0 {
						r := bufio.NewReader(os.Stdin)
						line, _ := r.ReadString('\n')
						password = strings.Trim(line, "\r\n")
					}
				} else {
					return fmt.Errorf("stdin is not a TTY. Use --password and pipe the password on stdin, or run interactively without piping")
				}
			}

			conn := config.Connection{
				Name:     name,
				Type:     normalizedType,
				Host:     host,
				Port:     port,
				Username: username,
				Password: password,
				Database: database,
				Schema:   schema,
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

func normalizeDBType(dbType string) string {
	switch strings.ToLower(dbType) {
	case "postgres", "postgresql":
		return "Postgres"
	case "mysql":
		return "MySQL"
	case "mariadb":
		return "MariaDB"
	case "mongodb":
		return "MongoDB"
	case "redis":
		return "Redis"
	case "clickhouse":
		return "ClickHouse"
	case "elasticsearch":
		return "ElasticSearch"
	case "sqlite":
		return "SQLite"
	default:
		return dbType
	}
}

func getDefaultPort(dbType string) int {
	switch strings.ToLower(dbType) {
	case "postgres", "postgresql":
		return 5432
	case "mysql", "mariadb":
		return 3306
	case "mongodb":
		return 27017
	case "redis":
		return 6379
	case "clickhouse":
		return 9000
	case "elasticsearch":
		return 9200
	default:
		return 5432
	}
}

func init() {
	rootCmd.AddCommand(connectCmd)

	connectCmd.Flags().StringVar(&dbType, "type", "", "database type (postgres, mysql, sqlite, mongodb, redis, etc.)")
	connectCmd.Flags().StringVar(&host, "host", "", "database host")
	connectCmd.Flags().IntVar(&port, "port", 0, "database port (default depends on database type)")
	connectCmd.Flags().StringVar(&username, "user", "", "database username")
	connectCmd.Flags().StringVar(&database, "database", "", "database name")
	connectCmd.Flags().StringVar(&schema, "schema", "", "default schema (optional)")
	connectCmd.Flags().StringVar(&name, "name", "", "connection name (save for later use)")
	connectCmd.Flags().BoolVar(&passwordFromStdin, "password", false, "read password from stdin when not using a TTY")
}
