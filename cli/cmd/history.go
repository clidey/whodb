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
	"fmt"
	"regexp"
	"strings"

	"github.com/clidey/whodb/cli/internal/history"
	"github.com/clidey/whodb/cli/pkg/output"
	"github.com/spf13/cobra"
)

var (
	historyFormat string
	historyQuiet  bool
	historyLimit  int
)

var historyCmd = &cobra.Command{
	Use:   "history",
	Short: "Access query history",
	Long: `Access and manage query history.

Subcommands:
  list   - List recent queries
  search - Search queries by pattern
  load   - Print a saved query by ID
  clear  - Clear all history`,
	Example: `  # List recent queries
  whodb-cli history list

  # List last 5 queries
  whodb-cli history list --limit 5

  # Search for queries containing "users"
  whodb-cli history search users

  # Print the full query for a history entry
  whodb-cli history load 1234567890

  # Clear all history
  whodb-cli history clear`,
}

var historyListCmd = &cobra.Command{
	Use:           "list",
	Short:         "List recent queries",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		format, err := output.ParseFormat(historyFormat)
		if err != nil {
			return err
		}

		quiet := historyQuiet || shouldSuppressInformationalOutput(cmd, format)
		out := newCommandOutput(cmd, format, quiet)

		mgr, err := history.NewManager()
		if err != nil {
			return fmt.Errorf("cannot load history: %w", err)
		}

		entries := mgr.GetAll()
		if len(entries) == 0 {
			out.Info("No query history found")
			if effectiveCommandOutputFormat(cmd, format) == output.FormatJSON {
				return writeEmptyJSONArray(cmd)
			}
			return nil
		}

		// Apply limit
		if historyLimit > 0 && historyLimit < len(entries) {
			entries = entries[:historyLimit]
		}

		// For JSON, output structured data
		if effectiveCommandOutputFormat(cmd, format) == output.FormatJSON {
			return writeCommandJSON(cmd, entries)
		}

		// For table/csv/plain formats
		columns := []output.Column{
			{Name: "id", Type: "string"},
			{Name: "timestamp", Type: "string"},
			{Name: "database", Type: "string"},
			{Name: "success", Type: "bool"},
			{Name: "query", Type: "string"},
		}

		rows := make([][]any, len(entries))
		for i, e := range entries {
			// Truncate query for display
			query := e.Query
			if len(query) > 60 {
				query = query[:57] + "..."
			}
			// Replace newlines with spaces for table display
			query = strings.ReplaceAll(query, "\n", " ")
			query = strings.ReplaceAll(query, "\t", " ")

			rows[i] = []any{
				e.ID,
				e.Timestamp.Format("2006-01-02 15:04:05"),
				e.Database,
				e.Success,
				query,
			}
		}

		result := &output.QueryResult{
			Columns: columns,
			Rows:    rows,
		}

		return out.WriteQueryResult(result)
	},
}

var historySearchCmd = &cobra.Command{
	Use:           "search [pattern]",
	Short:         "Search queries by pattern",
	SilenceUsage:  true,
	SilenceErrors: true,
	Args:          cobra.ExactArgs(1),
	Example:       `  whodb-cli history search "SELECT.*users"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		pattern := args[0]

		format, err := output.ParseFormat(historyFormat)
		if err != nil {
			return err
		}

		quiet := historyQuiet || shouldSuppressInformationalOutput(cmd, format)
		out := newCommandOutput(cmd, format, quiet)

		mgr, err := history.NewManager()
		if err != nil {
			return fmt.Errorf("cannot load history: %w", err)
		}

		entries := mgr.GetAll()

		// Compile regex pattern
		re, err := regexp.Compile("(?i)" + pattern)
		if err != nil {
			// Fall back to simple substring match
			re = nil
		}

		// Filter entries
		var matches []history.Entry
		for _, e := range entries {
			if re != nil {
				if re.MatchString(e.Query) {
					matches = append(matches, e)
				}
			} else {
				if strings.Contains(strings.ToLower(e.Query), strings.ToLower(pattern)) {
					matches = append(matches, e)
				}
			}
		}

		if len(matches) == 0 {
			out.Info("No matching queries found")
			if effectiveCommandOutputFormat(cmd, format) == output.FormatJSON {
				return writeEmptyJSONArray(cmd)
			}
			return nil
		}

		// Apply limit
		if historyLimit > 0 && historyLimit < len(matches) {
			matches = matches[:historyLimit]
		}

		// For JSON, output structured data
		if effectiveCommandOutputFormat(cmd, format) == output.FormatJSON {
			return writeCommandJSON(cmd, matches)
		}

		// For table/csv/plain formats
		columns := []output.Column{
			{Name: "id", Type: "string"},
			{Name: "timestamp", Type: "string"},
			{Name: "database", Type: "string"},
			{Name: "success", Type: "bool"},
			{Name: "query", Type: "string"},
		}

		rows := make([][]any, len(matches))
		for i, e := range matches {
			query := e.Query
			if len(query) > 60 {
				query = query[:57] + "..."
			}
			query = strings.ReplaceAll(query, "\n", " ")
			query = strings.ReplaceAll(query, "\t", " ")

			rows[i] = []any{
				e.ID,
				e.Timestamp.Format("2006-01-02 15:04:05"),
				e.Database,
				e.Success,
				query,
			}
		}

		result := &output.QueryResult{
			Columns: columns,
			Rows:    rows,
		}

		return out.WriteQueryResult(result)
	},
}

var historyLoadCmd = &cobra.Command{
	Use:           "load [id]",
	Short:         "Print a saved query by ID",
	SilenceUsage:  true,
	SilenceErrors: true,
	Args:          cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		format, err := output.ParseFormat(historyFormat)
		if err != nil {
			return err
		}

		mgr, err := history.NewManager()
		if err != nil {
			return fmt.Errorf("cannot load history: %w", err)
		}

		entry, err := mgr.Get(args[0])
		if err != nil {
			return fmt.Errorf("history entry %q not found: %w", args[0], err)
		}

		if effectiveCommandOutputFormat(cmd, format) == output.FormatJSON {
			return writeCommandJSON(cmd, entry)
		}

		if effectiveCommandOutputFormat(cmd, format) == output.FormatNDJSON {
			return writeCommandNDJSON(cmd, []*history.Entry{entry})
		}

		if effectiveCommandOutputFormat(cmd, format) == output.FormatTable ||
			effectiveCommandOutputFormat(cmd, format) == output.FormatCSV {
			out := newCommandOutput(cmd, format, true)
			return out.WriteQueryResult(&output.QueryResult{
				Columns: []output.Column{
					{Name: "id", Type: "string"},
					{Name: "timestamp", Type: "string"},
					{Name: "database", Type: "string"},
					{Name: "success", Type: "bool"},
					{Name: "query", Type: "string"},
				},
				Rows: [][]any{{
					entry.ID,
					entry.Timestamp.Format("2006-01-02 15:04:05"),
					entry.Database,
					entry.Success,
					entry.Query,
				}},
			})
		}

		_, err = fmt.Fprintln(cmd.OutOrStdout(), entry.Query)
		return err
	},
}

var historyClearCmd = &cobra.Command{
	Use:           "clear",
	Short:         "Clear all query history",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		format, err := output.ParseFormat(historyFormat)
		if err != nil {
			return err
		}
		quiet := historyQuiet || shouldSuppressInformationalOutput(cmd, format)
		out := newCommandOutput(cmd, format, quiet)

		mgr, err := history.NewManager()
		if err != nil {
			return fmt.Errorf("cannot load history: %w", err)
		}

		removedCount := len(mgr.GetAll())
		if err := mgr.Clear(); err != nil {
			return fmt.Errorf("failed to clear history: %w", err)
		}

		if effectiveCommandOutputFormat(cmd, format) == output.FormatJSON {
			return writeAutomationEnvelope(cmd, "history.clear", struct {
				RemovedCount int `json:"removedCount"`
			}{
				RemovedCount: removedCount,
			})
		}
		out.Success("History cleared")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(historyCmd)

	// Global flags
	historyCmd.PersistentFlags().StringVarP(&historyFormat, "format", "f", "auto", "output format: auto, table, plain, json, ndjson, csv")
	historyCmd.PersistentFlags().BoolVarP(&historyQuiet, "quiet", "q", false, "suppress informational messages")
	historyCmd.PersistentFlags().IntVarP(&historyLimit, "limit", "l", 0, "limit number of results (0 = no limit)")

	// Subcommands
	historyCmd.AddCommand(historyListCmd)
	historyCmd.AddCommand(historySearchCmd)
	historyCmd.AddCommand(historyLoadCmd)
	historyCmd.AddCommand(historyClearCmd)

	historyCmd.RegisterFlagCompletionFunc("format", completeOutputFormats)
}
