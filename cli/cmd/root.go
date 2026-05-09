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
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/clidey/whodb/cli/internal/config"
	"github.com/clidey/whodb/cli/internal/tui"
	"github.com/clidey/whodb/cli/pkg/analytics"
	"github.com/clidey/whodb/cli/pkg/identity"
	"github.com/clidey/whodb/cli/pkg/styles"
	"github.com/clidey/whodb/cli/pkg/updatecheck"
	"github.com/clidey/whodb/cli/pkg/version"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "whodb-cli",
	Short: "WhoDB CLI - Interactive database management tool",
	Long: `WhoDB CLI is an interactive, production-ready command-line interface for navigating SQL and NoSQL databases.

Features:
  - Split-pane TUI layouts (Single, Explore, Query, Full) — Ctrl+L to cycle
  - 8 color themes (Default, Monokai, Dracula, Nord, etc.) — Ctrl+T to cycle
  - Multi-database support (PostgreSQL, MySQL, SQLite, MongoDB, Redis, ClickHouse, etc.)
  - SQL editor with context-aware autocomplete, formatting (Ctrl+F), multi-tab buffers
  - External editor support (Ctrl+O opens $EDITOR)
  - ER diagram visualization (Ctrl+K) plus scriptable graph output (erd)
  - EXPLAIN query plan viewer (Ctrl+X) plus CLI explain output (explain)
  - Data import/export (CSV, Excel) — Ctrl+G for import wizard
  - FK-aware mock data generation (mock-data) with dependency analysis
  - Schema diff between saved connections in the TUI (Ctrl+V) and CLI (diff)
  - Cloud provider discovery commands (cloud) when provider support is enabled
  - Connect/save flows from discovered cloud resources (connect --discovered, connections add --from-discovered)
  - Backend-generated query suggestions (suggestions + editor empty state)
  - AI chat with streaming responses (OpenAI, Anthropic, Ollama, LM Studio)
  - SSH tunnel support for remote databases
  - SSL mode + certificate file support in commands and the TUI
  - Docker container auto-detection
  - Query bookmarks in the TUI (Ctrl+B) and CLI (bookmarks), history (Ctrl+H), command log (Ctrl+D)
  - Nested WHERE builder with AND/OR grouping
  - Connection profiles in the TUI (Ctrl+P) and CLI (profiles) — bundle connection + theme + settings
  - Workspace restore — resumes your last reconnectable TUI session on startup
  - Data quality audit with configurable thresholds (Ctrl+U)
  - Agent capability manifest (agent schema), connection diagnostics (doctor), and built-in runbooks
  - Bundled assistant skills and MCP integration installation (skills)
  - Read-only mode (Ctrl+Y)
  - JSON cell viewer, fish-style history suggestions

Press ? in any view for keyboard shortcuts.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		profileName := viper.GetString("profile")
		if profileName != "" {
			return runWithProfile(profileName)
		}
		// Start TUI directly
		m := tui.NewMainModel()
		p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
		finalModel, err := p.Run()
		if err != nil {
			return fmt.Errorf("error running interactive mode: %w", err)
		}
		if err := persistWorkspace(finalModel); err != nil {
			return err
		}
		return nil
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		if shouldSkipStartupSideEffects() || viper.GetBool("no-update-check") || version.Version == "dev" {
			return
		}
		if result := updatecheck.Check(version.Version); result != nil {
			fmt.Fprintf(os.Stderr, "\nA new version of %s is available: %s → %s\nUpgrade URL: %s\n",
				identity.Current().CommandName,
				version.Version,
				result.LatestVersion,
				identity.Current().UpdateCheckPageURL)
		}
	},
}

func Execute() {
	defer analytics.Shutdown()
	configureRuntime()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// runWithProfile loads a named profile, applies its settings, and connects.
func runWithProfile(name string) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("error loading config: %w", err)
	}

	profile := cfg.GetProfile(name)
	if profile == nil {
		return fmt.Errorf("profile %q not found", name)
	}

	conn, err := cfg.GetConnection(profile.Connection)
	if err != nil {
		return fmt.Errorf("profile %q references missing connection %q", name, profile.Connection)
	}

	// Apply display/query settings from the profile
	if profile.Theme != "" {
		if t := styles.GetThemeByName(profile.Theme); t != nil {
			styles.SetTheme(t)
			cfg.SetThemeName(profile.Theme)
		}
	}
	if profile.PageSize > 0 {
		cfg.SetPageSize(profile.PageSize)
	}
	if profile.TimeoutSeconds > 0 {
		cfg.Query.TimeoutSeconds = profile.TimeoutSeconds
	}

	m := tui.NewMainModelWithProfile(conn, cfg, name)
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("error running interactive mode: %w", err)
	}
	if err := persistWorkspace(finalModel); err != nil {
		return err
	}
	return nil
}

func persistWorkspace(model tea.Model) error {
	finalModel, ok := model.(*tui.MainModel)
	if !ok || finalModel == nil {
		return nil
	}
	if err := finalModel.PersistWorkspace(); err != nil {
		return fmt.Errorf("error saving workspace: %w", err)
	}
	return nil
}

func init() {
	cobra.OnInitialize(initConfig, initColorMode, initAnalytics)

	// Disable Cobra's default completion command; we provide our own with install support
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	rootCmd.PersistentFlags().String("profile", "", "load a named connection profile")
	rootCmd.PersistentFlags().Bool("debug", false, "enable debug mode")
	rootCmd.PersistentFlags().Bool("no-color", false, "disable colored output")
	rootCmd.PersistentFlags().Bool("no-analytics", false, "disable anonymous usage analytics")
	rootCmd.PersistentFlags().Bool("no-update-check", false, "disable update check notifications")

	viper.BindPFlag("profile", rootCmd.PersistentFlags().Lookup("profile"))
	viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug"))
	viper.BindPFlag("no-color", rootCmd.PersistentFlags().Lookup("no-color"))
	viper.BindPFlag("no-analytics", rootCmd.PersistentFlags().Lookup("no-analytics"))
	viper.BindPFlag("no-update-check", rootCmd.PersistentFlags().Lookup("no-update-check"))

	rootCmd.RegisterFlagCompletionFunc("profile", completeProfileNames)
}

func initAnalytics() {
	// Skip analytics if disabled via flag or env
	if shouldSkipStartupSideEffects() || viper.GetBool("no-analytics") || os.Getenv(identity.Current().AnalyticsDisabledEnv) == "true" {
		return
	}

	// Initialize analytics (errors are silently ignored - analytics should never block CLI)
	_ = analytics.Initialize(version.Version)

	// Track CLI startup with the command being run
	analytics.TrackCLIStartup(context.Background(), startupCommandName())
}

func shouldSkipStartupSideEffects() bool {
	return isTrivialInvocation(os.Args[1:])
}

func startupCommandName() string {
	command := firstNonFlagArg(os.Args[1:])
	if command == "" {
		return "tui"
	}
	return command
}

func isTrivialInvocation(args []string) bool {
	for _, arg := range args {
		if arg == "-h" || arg == "--help" {
			return true
		}
	}

	switch firstNonFlagArg(args) {
	case "help", "version", "completion", "__complete", "__completeNoDesc":
		return true
	default:
		return false
	}
}

func firstNonFlagArg(args []string) string {
	for i, arg := range args {
		if arg == "--" {
			if i+1 < len(args) {
				return args[i+1]
			}
			return ""
		}
		if arg == "--profile" {
			i++
			continue
		}
		if strings.HasPrefix(arg, "--profile=") {
			continue
		}
		if strings.HasPrefix(arg, "-") {
			continue
		}
		return arg
	}
	return ""
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
		configDir, err := identity.HomePath()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting config directory: %v\n", err)
			os.Exit(1)
		}

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

	viper.SetEnvPrefix(identity.Current().ViperEnvPrefix)
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		if viper.GetBool("debug") {
			fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
		}
	}
}
