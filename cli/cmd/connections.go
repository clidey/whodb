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
	"encoding/json"
	"fmt"

	"github.com/clidey/whodb/cli/internal/config"
	dbmgr "github.com/clidey/whodb/cli/internal/database"
	"github.com/clidey/whodb/cli/pkg/output"
	"github.com/spf13/cobra"
)

var (
	connectionsFormat string
	connectionsQuiet  bool
)

var connectionsCmd = &cobra.Command{
	Use:   "connections",
	Short: "Manage database connections",
	Long: `Manage database connections.

Subcommands:
  list    - List available connections
  add     - Add a new connection
  remove  - Remove a saved connection
  test    - Test a connection`,
	Example: `  # List all connections
  whodb-cli connections list

  # Add a new connection
  whodb-cli connections add --name mydb --type postgres --host localhost --user admin --database myapp

  # Test a connection
  whodb-cli connections test mydb

  # Remove a connection
  whodb-cli connections remove mydb`,
}

// connections list
var connectionsListCmd = &cobra.Command{
	Use:           "list",
	Short:         "List available connections",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		format, err := output.ParseFormat(connectionsFormat)
		if err != nil {
			return err
		}

		out := output.New(
			output.WithFormat(format),
			output.WithQuiet(connectionsQuiet),
		)

		mgr, err := dbmgr.NewManager()
		if err != nil {
			return fmt.Errorf("cannot initialize database manager: %w", err)
		}

		connections := mgr.ListConnectionsWithSource()
		if len(connections) == 0 {
			out.Info("No connections available. Create one with:")
			out.Info("  whodb-cli connect --type postgres --host localhost --user myuser --database mydb --name myconn")
			// Output empty result for scripting
			if format == output.FormatJSON {
				fmt.Println("[]")
			}
			return nil
		}

		// For JSON, output a clean structure without passwords
		if format == output.FormatJSON {
			type safeConnection struct {
				Name     string `json:"name"`
				Type     string `json:"type"`
				Host     string `json:"host"`
				Port     int    `json:"port,omitempty"`
				Username string `json:"username"`
				Database string `json:"database"`
				Schema   string `json:"schema,omitempty"`
				Source   string `json:"source"`
			}

			safeConns := make([]safeConnection, len(connections))
			for i, c := range connections {
				conn := c.Connection
				safeConns[i] = safeConnection{
					Name:     conn.Name,
					Type:     conn.Type,
					Host:     conn.Host,
					Port:     conn.Port,
					Username: conn.Username,
					Database: conn.Database,
					Schema:   conn.Schema,
					Source:   c.Source,
				}
			}

			encoder := json.NewEncoder(cmd.OutOrStdout())
			encoder.SetIndent("", "  ")
			return encoder.Encode(safeConns)
		}

		// For table/csv/plain formats
		columns := []output.Column{
			{Name: "name", Type: "string"},
			{Name: "type", Type: "string"},
			{Name: "host", Type: "string"},
			{Name: "port", Type: "int"},
			{Name: "database", Type: "string"},
			{Name: "username", Type: "string"},
			{Name: "source", Type: "string"},
		}

		rows := make([][]any, len(connections))
		for i, c := range connections {
			conn := c.Connection
			rows[i] = []any{conn.Name, conn.Type, conn.Host, conn.Port, conn.Database, conn.Username, c.Source}
		}

		result := &output.QueryResult{
			Columns: columns,
			Rows:    rows,
		}

		return out.WriteQueryResult(result)
	},
}

// connections add flags
var (
	connAddName     string
	connAddType     string
	connAddHost     string
	connAddPort     int
	connAddUser     string
	connAddPassword string
	connAddDatabase string
	connAddSchema   string
)

var connectionsAddCmd = &cobra.Command{
	Use:           "add",
	Short:         "Add a new connection",
	SilenceUsage:  true,
	SilenceErrors: true,
	Long: `Add a new database connection.

Supported database types:
  Postgres, MySQL, MariaDB, SQLite, MongoDB, Redis, ClickHouse, ElasticSearch`,
	Example: `  # Add a PostgreSQL connection
  whodb-cli connections add --name mydb --type Postgres --host localhost --port 5432 --user admin --password secret --database myapp

  # Add with schema
  whodb-cli connections add --name mydb --type Postgres --host localhost --user admin --database myapp --schema public`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if connAddName == "" {
			return fmt.Errorf("--name is required")
		}
		if connAddType == "" {
			return fmt.Errorf("--type is required")
		}
		if connAddHost == "" && connAddType != "SQLite" {
			return fmt.Errorf("--host is required")
		}
		if connAddDatabase == "" {
			return fmt.Errorf("--database is required")
		}

		out := output.New(output.WithQuiet(connectionsQuiet))

		cfg, err := config.LoadConfig()
		if err != nil {
			return fmt.Errorf("cannot load config: %w", err)
		}

		conn := config.Connection{
			Name:     connAddName,
			Type:     connAddType,
			Host:     connAddHost,
			Port:     connAddPort,
			Username: connAddUser,
			Password: connAddPassword,
			Database: connAddDatabase,
			Schema:   connAddSchema,
		}

		cfg.AddConnection(conn)
		if err := cfg.Save(); err != nil {
			return fmt.Errorf("failed to save connection: %w", err)
		}

		out.Success("Connection %q saved", connAddName)
		return nil
	},
}

var connectionsRemoveCmd = &cobra.Command{
	Use:           "remove [name]",
	Short:         "Remove a saved connection",
	SilenceUsage:  true,
	SilenceErrors: true,
	Args:          cobra.ExactArgs(1),
	Example:       `  whodb-cli connections remove mydb`,
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		out := output.New(output.WithQuiet(connectionsQuiet))

		mgr, err := dbmgr.NewManager()
		if err != nil {
			return fmt.Errorf("cannot initialize database manager: %w", err)
		}

		_, source, err := mgr.ResolveConnection(name)
		if err != nil {
			return err
		}
		if source == dbmgr.ConnectionSourceEnv {
			return fmt.Errorf("connection %q is defined via environment variables and cannot be removed", name)
		}

		cfg, err := config.LoadConfig()
		if err != nil {
			return fmt.Errorf("cannot load config: %w", err)
		}

		if !cfg.RemoveConnection(name) {
			return fmt.Errorf("connection %q not found", name)
		}

		if err := cfg.Save(); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		out.Success("Connection %q removed", name)
		return nil
	},
}

var connectionsTestCmd = &cobra.Command{
	Use:           "test [name]",
	Short:         "Test a connection",
	SilenceUsage:  true,
	SilenceErrors: true,
	Args:          cobra.ExactArgs(1),
	Example:       `  whodb-cli connections test mydb`,
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		out := output.New(output.WithQuiet(connectionsQuiet))

		mgr, err := dbmgr.NewManager()
		if err != nil {
			return fmt.Errorf("cannot initialize database manager: %w", err)
		}

		conn, _, err := mgr.ResolveConnection(name)
		if err != nil {
			return err
		}

		var spinner *output.Spinner
		if !connectionsQuiet {
			spinner = output.NewSpinner(fmt.Sprintf("Testing connection to %s...", conn.Type))
		}
		spinner.Start()

		if err := mgr.Connect(conn); err != nil {
			spinner.StopWithError("Connection failed")
			return fmt.Errorf("connection test failed: %w", err)
		}
		defer mgr.Disconnect()

		spinner.StopWithSuccess("Connection successful")
		out.Success("Successfully connected to %s (%s)", name, conn.Type)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(connectionsCmd)

	// Global flags for connections command
	connectionsCmd.PersistentFlags().StringVarP(&connectionsFormat, "format", "f", "auto", "output format: auto, table, plain, json, csv")
	connectionsCmd.PersistentFlags().BoolVarP(&connectionsQuiet, "quiet", "q", false, "suppress informational messages")

	// Subcommands
	connectionsCmd.AddCommand(connectionsListCmd)
	connectionsCmd.AddCommand(connectionsAddCmd)
	connectionsCmd.AddCommand(connectionsRemoveCmd)
	connectionsCmd.AddCommand(connectionsTestCmd)

	// Add command flags
	connectionsAddCmd.Flags().StringVar(&connAddName, "name", "", "connection name (required)")
	connectionsAddCmd.Flags().StringVar(&connAddType, "type", "", "database type: Postgres, MySQL, MariaDB, SQLite, MongoDB, Redis, ClickHouse, ElasticSearch (required)")
	connectionsAddCmd.Flags().StringVar(&connAddHost, "host", "", "database host")
	connectionsAddCmd.Flags().IntVar(&connAddPort, "port", 0, "database port")
	connectionsAddCmd.Flags().StringVar(&connAddUser, "user", "", "database username")
	connectionsAddCmd.Flags().StringVar(&connAddPassword, "password", "", "database password")
	connectionsAddCmd.Flags().StringVar(&connAddDatabase, "database", "", "database name (required)")
	connectionsAddCmd.Flags().StringVar(&connAddSchema, "schema", "", "default schema (optional)")
}
