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
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/clidey/whodb/cli/internal/tui"
	"github.com/clidey/whodb/cli/pkg/analytics"
	"github.com/clidey/whodb/cli/pkg/styles"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Version is set at build time via ldflags
var Version = "dev"

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "whodb-cli",
	Short: "WhoDB CLI - Interactive database management tool",
	Long: `WhoDB CLI is an interactive, production-ready command-line interface for WhoDB with a Claude Code-like experience.

Features:
  - Interactive TUI with responsive design
  - Multi-database support (PostgreSQL, MySQL, SQLite, MongoDB, Redis, etc.)
  - Visual query builder with conditions and intellisense
  - SQL editor with syntax highlighting
  - Efficient data viewer with pagination
  - Export capabilities (CSV, Excel)
  - Query history with re-execution
  - Session persistence`,
	Run: func(cmd *cobra.Command, args []string) {
		// Start TUI directly
		m := tui.NewMainModel()
		p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
		if _, err := p.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func Execute() {
	defer analytics.Shutdown()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig, initColorMode, initAnalytics)

	// Disable Cobra's default completion command; we provide our own with install support
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	rootCmd.PersistentFlags().Bool("debug", false, "enable debug mode")
	rootCmd.PersistentFlags().Bool("no-color", false, "disable colored output")
	rootCmd.PersistentFlags().Bool("no-analytics", false, "disable anonymous usage analytics")

	viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug"))
	viper.BindPFlag("no-color", rootCmd.PersistentFlags().Lookup("no-color"))
	viper.BindPFlag("no-analytics", rootCmd.PersistentFlags().Lookup("no-analytics"))
}

func initAnalytics() {
	// Skip analytics if disabled via flag or env
	if viper.GetBool("no-analytics") || os.Getenv("WHODB_CLI_ANALYTICS_DISABLED") == "true" {
		return
	}

	// Initialize analytics (errors are silently ignored - analytics should never block CLI)
	_ = analytics.Initialize(Version)

	// Track CLI startup with the command being run
	if len(os.Args) > 1 {
		analytics.TrackCLIStartup(context.Background(), os.Args[1])
	} else {
		analytics.TrackCLIStartup(context.Background(), "tui")
	}
}

func initColorMode() {
	if viper.GetBool("no-color") {
		styles.DisableColor()
	}
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting home directory: %v\n", err)
			os.Exit(1)
		}

		configDir := fmt.Sprintf("%s/.whodb-cli", home)
		if err := os.MkdirAll(configDir, 0700); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating config directory: %v\n", err)
			os.Exit(1)
		}
		// Enforce strict permissions
		_ = os.Chmod(configDir, 0700)

		viper.AddConfigPath(configDir)
		viper.SetConfigType("yaml")
		viper.SetConfigName("config")
	}

	viper.SetEnvPrefix("WHODB_CLI")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		if viper.GetBool("debug") {
			fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
		}
	}
}
