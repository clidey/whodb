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
	"fmt"

	"github.com/clidey/whodb/cli/internal/config"
	"github.com/clidey/whodb/cli/pkg/output"
	"github.com/spf13/cobra"
)

var (
	profilesFormat       string
	profilesQuiet        bool
	profilesSaveConn     string
	profilesSaveTheme    string
	profilesSavePageSize int
	profilesSaveTimeout  int
)

var profilesCmd = &cobra.Command{
	Use:   "profiles",
	Short: "Manage saved connection profiles",
	Long: `Manage the same saved connection profiles used by the TUI.

Profiles bundle a saved connection with theme, page size, and timeout
preferences. To apply one, use the root --profile flag:

  whodb --profile prod`,
}

var profilesListCmd = &cobra.Command{
	Use:           "list",
	Short:         "List saved profiles",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		format, err := output.ParseFormat(profilesFormat)
		if err != nil {
			return err
		}
		quiet := profilesQuiet || shouldSuppressInformationalOutput(cmd, format)
		out := newCommandOutput(cmd, format, quiet)

		cfg, err := config.LoadConfig()
		if err != nil {
			return fmt.Errorf("cannot load config: %w", err)
		}

		profiles := cfg.GetProfiles()
		if len(profiles) == 0 {
			out.Info("No saved profiles")
			if effectiveCommandOutputFormat(cmd, format) == output.FormatJSON {
				return writeEmptyJSONArray(cmd)
			}
			return nil
		}

		rows := make([][]any, len(profiles))
		for i, profile := range profiles {
			rows[i] = []any{
				profile.Name,
				profile.Connection,
				profile.Theme,
				profile.PageSize,
				profile.TimeoutSeconds,
			}
		}

		return out.WriteQueryResult(&output.QueryResult{
			Columns: []output.Column{
				{Name: "name", Type: "string"},
				{Name: "connection", Type: "string"},
				{Name: "theme", Type: "string"},
				{Name: "pageSize", Type: "int"},
				{Name: "timeoutSeconds", Type: "int"},
			},
			Rows: rows,
		})
	},
}

var profilesSaveCmd = &cobra.Command{
	Use:           "save [name]",
	Short:         "Save a profile",
	SilenceUsage:  true,
	SilenceErrors: true,
	Args:          cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		format, err := output.ParseFormat(profilesFormat)
		if err != nil {
			return err
		}
		quiet := profilesQuiet || shouldSuppressInformationalOutput(cmd, format)
		out := newCommandOutput(cmd, format, quiet)

		if profilesSaveConn == "" {
			return fmt.Errorf("--connection is required")
		}

		cfg, err := config.LoadConfig()
		if err != nil {
			return fmt.Errorf("cannot load config: %w", err)
		}

		if _, err := cfg.GetConnection(profilesSaveConn); err != nil {
			return fmt.Errorf("saved connection %q not found", profilesSaveConn)
		}

		profile := config.Profile{
			Name:           args[0],
			Connection:     profilesSaveConn,
			Theme:          profilesSaveTheme,
			PageSize:       profilesSavePageSize,
			TimeoutSeconds: profilesSaveTimeout,
		}

		if profile.Theme == "" {
			profile.Theme = cfg.GetThemeName()
		}
		if profile.PageSize == 0 {
			profile.PageSize = cfg.GetPageSize()
		}
		if profile.TimeoutSeconds == 0 {
			profile.TimeoutSeconds = cfg.Query.TimeoutSeconds
		}

		cfg.AddProfile(profile)
		if err := cfg.Save(); err != nil {
			return fmt.Errorf("cannot save config: %w", err)
		}

		if effectiveCommandOutputFormat(cmd, format) == output.FormatJSON {
			return writeAutomationEnvelope(cmd, "profiles.save", profile)
		}

		out.Info("Saved profile: %s", profile.Name)
		return nil
	},
}

var profilesShowCmd = &cobra.Command{
	Use:           "show [name]",
	Short:         "Show one saved profile",
	SilenceUsage:  true,
	SilenceErrors: true,
	Args:          cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		format, err := output.ParseFormat(profilesFormat)
		if err != nil {
			return err
		}

		cfg, err := config.LoadConfig()
		if err != nil {
			return fmt.Errorf("cannot load config: %w", err)
		}

		profile := cfg.GetProfile(args[0])
		if profile == nil {
			return fmt.Errorf("profile %q not found", args[0])
		}

		if effectiveCommandOutputFormat(cmd, format) == output.FormatJSON {
			return writeCommandJSON(cmd, profile)
		}

		out := newCommandOutput(cmd, format, true)
		return out.WriteQueryResult(&output.QueryResult{
			Columns: []output.Column{
				{Name: "name", Type: "string"},
				{Name: "connection", Type: "string"},
				{Name: "theme", Type: "string"},
				{Name: "pageSize", Type: "int"},
				{Name: "timeoutSeconds", Type: "int"},
			},
			Rows: [][]any{{
				profile.Name,
				profile.Connection,
				profile.Theme,
				profile.PageSize,
				profile.TimeoutSeconds,
			}},
		})
	},
}

var profilesDeleteCmd = &cobra.Command{
	Use:           "delete [name]",
	Short:         "Delete a saved profile",
	SilenceUsage:  true,
	SilenceErrors: true,
	Args:          cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		format, err := output.ParseFormat(profilesFormat)
		if err != nil {
			return err
		}
		quiet := profilesQuiet || shouldSuppressInformationalOutput(cmd, format)
		out := newCommandOutput(cmd, format, quiet)

		cfg, err := config.LoadConfig()
		if err != nil {
			return fmt.Errorf("cannot load config: %w", err)
		}

		if !cfg.DeleteProfile(args[0]) {
			return fmt.Errorf("profile %q not found", args[0])
		}
		if err := cfg.Save(); err != nil {
			return fmt.Errorf("cannot save config: %w", err)
		}

		if effectiveCommandOutputFormat(cmd, format) == output.FormatJSON {
			return writeAutomationEnvelope(cmd, "profiles.delete", struct {
				Name string `json:"name"`
			}{Name: args[0]})
		}

		out.Info("Deleted profile: %s", args[0])
		return nil
	},
}

func init() {
	rootCmd.AddCommand(profilesCmd)

	profilesCmd.AddCommand(profilesListCmd)
	profilesCmd.AddCommand(profilesSaveCmd)
	profilesCmd.AddCommand(profilesShowCmd)
	profilesCmd.AddCommand(profilesDeleteCmd)

	for _, command := range []*cobra.Command{profilesListCmd, profilesSaveCmd, profilesShowCmd, profilesDeleteCmd} {
		command.Flags().StringVarP(&profilesFormat, "format", "f", "table", "output format: auto, table, plain, json, ndjson, or csv")
		command.Flags().BoolVarP(&profilesQuiet, "quiet", "q", false, "suppress informational messages")
		command.RegisterFlagCompletionFunc("format", completeOutputFormats)
	}

	profilesSaveCmd.Flags().StringVarP(&profilesSaveConn, "connection", "c", "", "saved connection name to include")
	profilesSaveCmd.Flags().StringVar(&profilesSaveTheme, "theme", "", "theme override for the profile")
	profilesSaveCmd.Flags().IntVar(&profilesSavePageSize, "page-size", 0, "page size override for the profile")
	profilesSaveCmd.Flags().IntVar(&profilesSaveTimeout, "timeout", 0, "query timeout override in seconds")
	profilesSaveCmd.RegisterFlagCompletionFunc("connection", completeConnectionNames)
}
