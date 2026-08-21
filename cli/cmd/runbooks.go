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

	"github.com/clidey/whodb/cli/internal/runbooks"
	"github.com/clidey/whodb/cli/pkg/output"
	"github.com/spf13/cobra"
)

var (
	runbooksFormat     string
	runbooksQuiet      bool
	runbooksConnection string
	runbooksSchema     string
	runbooksFrom       string
	runbooksTo         string
	runbooksFromSchema string
	runbooksToSchema   string
	runbooksDryRun     bool
)

var runbooksCmd = &cobra.Command{
	Use:   "runbooks",
	Short: "Run built-in database workflows",
	Long:  `List, describe, and run built-in database workflows for agent and operator use.`,
}

var runbooksListCmd = &cobra.Command{
	Use:           "list",
	Short:         "List built-in runbooks",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		format, err := resolveRunbooksFormat(runbooksFormat)
		if err != nil {
			return err
		}
		definitions := runbooks.List()
		if format == output.FormatJSON {
			return writeCommandJSON(cmd, definitions)
		}
		if !runbooksQuiet {
			writeRunbookDefinitions(cmd, definitions)
		}
		return nil
	},
}

var runbooksDescribeCmd = &cobra.Command{
	Use:           "describe [name]",
	Short:         "Describe a built-in runbook",
	SilenceUsage:  true,
	SilenceErrors: true,
	Args:          cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		format, err := resolveRunbooksFormat(runbooksFormat)
		if err != nil {
			return err
		}
		definition, ok := runbooks.Describe(args[0])
		if !ok {
			return fmt.Errorf("runbook %q not found", args[0])
		}
		if format == output.FormatJSON {
			return writeCommandJSON(cmd, definition)
		}
		if !runbooksQuiet {
			writeRunbookDefinitions(cmd, []runbooks.Definition{definition})
		}
		return nil
	},
}

var runbooksRunCmd = &cobra.Command{
	Use:           "run [name]",
	Short:         "Run a built-in runbook",
	SilenceUsage:  true,
	SilenceErrors: true,
	Args:          cobra.ExactArgs(1),
	Example: `  whodb runbooks run connection-doctor --connection prod
  whodb runbooks run schema-audit --connection prod --format json
  whodb runbooks run schema-diff --from staging --to prod --format json
  whodb runbooks run schema-audit --connection prod --dry-run`,
	RunE: func(cmd *cobra.Command, args []string) error {
		format, err := resolveRunbooksFormat(runbooksFormat)
		if err != nil {
			return err
		}
		result, err := runbooks.Run(context.Background(), args[0], runbooks.Options{
			Connection: runbooksConnection,
			Schema:     runbooksSchema,
			From:       runbooksFrom,
			To:         runbooksTo,
			FromSchema: runbooksFromSchema,
			ToSchema:   runbooksToSchema,
			DryRun:     runbooksDryRun,
		})
		if err != nil {
			return err
		}
		if format == output.FormatJSON {
			return writeAutomationEnvelope(cmd, "runbooks.run", result)
		}
		if !runbooksQuiet {
			writeRunbookResult(cmd, result)
		}
		return nil
	},
}

func resolveRunbooksFormat(value string) (output.Format, error) {
	switch value {
	case "", "table", "auto":
		return output.FormatTable, nil
	case "json":
		return output.FormatJSON, nil
	default:
		return "", fmt.Errorf("invalid --format %q (expected table or json)", value)
	}
}

func writeRunbookDefinitions(cmd *cobra.Command, definitions []runbooks.Definition) {
	out := newCommandOutput(cmd, output.FormatTable, false)
	for _, definition := range definitions {
		out.Info("%s: %s", definition.Name, definition.Description)
		for _, step := range definition.Steps {
			out.Info("  - %s: %s", step.Name, step.Description)
		}
	}
}

func writeRunbookResult(cmd *cobra.Command, result runbooks.Result) {
	out := newCommandOutput(cmd, output.FormatTable, false)
	out.Info("Runbook: %s", result.Name)
	for _, step := range result.Steps {
		message := step.Message
		if message != "" {
			message = " " + message
		}
		out.Info("%s: %s%s", step.Name, step.Status, message)
	}
}

func init() {
	rootCmd.AddCommand(runbooksCmd)
	runbooksCmd.AddCommand(runbooksListCmd)
	runbooksCmd.AddCommand(runbooksDescribeCmd)
	runbooksCmd.AddCommand(runbooksRunCmd)

	runbooksCmd.PersistentFlags().StringVarP(&runbooksFormat, "format", "f", "table", "output format: table or json")
	runbooksCmd.PersistentFlags().BoolVarP(&runbooksQuiet, "quiet", "q", false, "suppress informational messages")

	runbooksRunCmd.Flags().StringVarP(&runbooksConnection, "connection", "c", "", "connection name")
	runbooksRunCmd.Flags().StringVarP(&runbooksSchema, "schema", "s", "", "schema override")
	runbooksRunCmd.Flags().StringVar(&runbooksFrom, "from", "", "source connection for schema-diff")
	runbooksRunCmd.Flags().StringVar(&runbooksTo, "to", "", "target connection for schema-diff")
	runbooksRunCmd.Flags().StringVar(&runbooksFromSchema, "from-schema", "", "source schema override for schema-diff")
	runbooksRunCmd.Flags().StringVar(&runbooksToSchema, "to-schema", "", "target schema override for schema-diff")
	runbooksRunCmd.Flags().BoolVar(&runbooksDryRun, "dry-run", false, "show planned steps without executing")

	runbooksCmd.RegisterFlagCompletionFunc("format", completeOutputFormats)
	runbooksRunCmd.RegisterFlagCompletionFunc("connection", completeConnectionNames)
	runbooksRunCmd.RegisterFlagCompletionFunc("from", completeConnectionNames)
	runbooksRunCmd.RegisterFlagCompletionFunc("to", completeConnectionNames)
}
