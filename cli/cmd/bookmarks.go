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
	"strings"

	"github.com/clidey/whodb/cli/internal/config"
	"github.com/clidey/whodb/cli/pkg/output"
	"github.com/spf13/cobra"
)

var (
	bookmarkFormat string
	bookmarkQuiet  bool
)

var bookmarksCmd = &cobra.Command{
	Use:   "bookmarks",
	Short: "Manage saved query bookmarks",
	Long: `Manage the same saved query bookmarks used by the TUI editor.

Subcommands:
  list    List saved bookmarks
  save    Save a bookmark from SQL text
  load    Print a saved bookmark's SQL
  delete  Remove a saved bookmark`,
}

var bookmarksListCmd = &cobra.Command{
	Use:           "list",
	Short:         "List saved bookmarks",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		format, err := output.ParseFormat(bookmarkFormat)
		if err != nil {
			return err
		}
		quiet := bookmarkQuiet || shouldSuppressInformationalOutput(cmd, format)
		out := newCommandOutput(cmd, format, quiet)

		cfg, err := config.LoadConfig()
		if err != nil {
			return fmt.Errorf("cannot load config: %w", err)
		}

		bookmarks := cfg.GetSavedQueries()
		if len(bookmarks) == 0 {
			out.Info("No saved bookmarks")
			if effectiveCommandOutputFormat(cmd, format) == output.FormatJSON {
				return writeEmptyJSONArray(cmd)
			}
			return nil
		}

		rows := make([][]any, len(bookmarks))
		for i, bookmark := range bookmarks {
			rows[i] = []any{bookmark.Name, strings.ReplaceAll(bookmark.Query, "\n", " ")}
		}

		return out.WriteQueryResult(&output.QueryResult{
			Columns: []output.Column{
				{Name: "name", Type: "string"},
				{Name: "query", Type: "string"},
			},
			Rows: rows,
		})
	},
}

var bookmarksSaveCmd = &cobra.Command{
	Use:           "save [name] [SQL|-]",
	Short:         "Save a bookmarked query",
	SilenceUsage:  true,
	SilenceErrors: true,
	Args:          cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		format, err := output.ParseFormat(bookmarkFormat)
		if err != nil {
			return err
		}
		quiet := bookmarkQuiet || shouldSuppressInformationalOutput(cmd, format)
		out := newCommandOutput(cmd, format, quiet)

		sql, err := readSQLArgument(args[1])
		if err != nil {
			return err
		}

		cfg, err := config.LoadConfig()
		if err != nil {
			return fmt.Errorf("cannot load config: %w", err)
		}

		cfg.AddSavedQuery(args[0], sql)
		if err := cfg.Save(); err != nil {
			return fmt.Errorf("cannot save config: %w", err)
		}

		if effectiveCommandOutputFormat(cmd, format) == output.FormatJSON {
			return writeAutomationEnvelope(cmd, "bookmarks.save", config.SavedQuery{
				Name:  args[0],
				Query: sql,
			})
		}

		out.Info("Saved bookmark: %s", args[0])
		return nil
	},
}

var bookmarksLoadCmd = &cobra.Command{
	Use:           "load [name]",
	Short:         "Print a bookmarked query",
	SilenceUsage:  true,
	SilenceErrors: true,
	Args:          cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		format, err := output.ParseFormat(bookmarkFormat)
		if err != nil {
			return err
		}

		cfg, err := config.LoadConfig()
		if err != nil {
			return fmt.Errorf("cannot load config: %w", err)
		}

		var bookmark *config.SavedQuery
		for _, saved := range cfg.GetSavedQueries() {
			if saved.Name == args[0] {
				copy := saved
				bookmark = new(copy)
				break
			}
		}
		if bookmark == nil {
			return fmt.Errorf("bookmark %q not found", args[0])
		}

		if effectiveCommandOutputFormat(cmd, format) == output.FormatJSON {
			return writeCommandJSON(cmd, bookmark)
		}

		if effectiveCommandOutputFormat(cmd, format) == output.FormatNDJSON {
			return writeCommandNDJSON(cmd, []*config.SavedQuery{bookmark})
		}

		if effectiveCommandOutputFormat(cmd, format) == output.FormatTable ||
			effectiveCommandOutputFormat(cmd, format) == output.FormatCSV {
			out := newCommandOutput(cmd, format, true)
			return out.WriteQueryResult(&output.QueryResult{
				Columns: []output.Column{
					{Name: "name", Type: "string"},
					{Name: "query", Type: "string"},
				},
				Rows: [][]any{{bookmark.Name, bookmark.Query}},
			})
		}

		_, err = fmt.Fprintln(cmd.OutOrStdout(), bookmark.Query)
		return err
	},
}

var bookmarksDeleteCmd = &cobra.Command{
	Use:           "delete [name]",
	Short:         "Delete a bookmarked query",
	SilenceUsage:  true,
	SilenceErrors: true,
	Args:          cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		format, err := output.ParseFormat(bookmarkFormat)
		if err != nil {
			return err
		}
		quiet := bookmarkQuiet || shouldSuppressInformationalOutput(cmd, format)
		out := newCommandOutput(cmd, format, quiet)

		cfg, err := config.LoadConfig()
		if err != nil {
			return fmt.Errorf("cannot load config: %w", err)
		}

		if !cfg.DeleteSavedQuery(args[0]) {
			return fmt.Errorf("bookmark %q not found", args[0])
		}
		if err := cfg.Save(); err != nil {
			return fmt.Errorf("cannot save config: %w", err)
		}

		if effectiveCommandOutputFormat(cmd, format) == output.FormatJSON {
			return writeAutomationEnvelope(cmd, "bookmarks.delete", struct {
				Name string `json:"name"`
			}{Name: args[0]})
		}

		out.Info("Deleted bookmark: %s", args[0])
		return nil
	},
}

func init() {
	rootCmd.AddCommand(bookmarksCmd)

	bookmarksCmd.AddCommand(bookmarksListCmd)
	bookmarksCmd.AddCommand(bookmarksSaveCmd)
	bookmarksCmd.AddCommand(bookmarksLoadCmd)
	bookmarksCmd.AddCommand(bookmarksDeleteCmd)

	for _, command := range []*cobra.Command{bookmarksListCmd, bookmarksSaveCmd, bookmarksDeleteCmd} {
		command.Flags().StringVarP(&bookmarkFormat, "format", "f", "table", "output format: auto, table, plain, json, ndjson, or csv")
		command.Flags().BoolVarP(&bookmarkQuiet, "quiet", "q", false, "suppress informational messages")
		command.RegisterFlagCompletionFunc("format", completeOutputFormats)
	}

	bookmarksLoadCmd.Flags().StringVarP(&bookmarkFormat, "format", "f", "plain", "output format: auto, table, plain, json, ndjson, or csv")
	bookmarksLoadCmd.Flags().BoolVarP(&bookmarkQuiet, "quiet", "q", false, "suppress informational messages")
	bookmarksLoadCmd.RegisterFlagCompletionFunc("format", completeOutputFormats)
}
