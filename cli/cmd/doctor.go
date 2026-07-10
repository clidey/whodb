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

	"github.com/clidey/whodb/cli/internal/doctor"
	"github.com/clidey/whodb/cli/pkg/output"
	"github.com/spf13/cobra"
)

var (
	doctorConnection string
	doctorSchema     string
	doctorFormat     string
	doctorQuiet      bool
)

var doctorCmd = &cobra.Command{
	Use:           "doctor",
	Short:         "Run database connection diagnostics",
	SilenceUsage:  true,
	SilenceErrors: true,
	Long: `Run connection, schema, and metadata diagnostics for one database connection.

The JSON output is redacted and does not include passwords or connection
strings.`,
	Example: `  # Run diagnostics for one connection
  whodb-cli doctor --connection prod

  # Emit machine-readable diagnostics
  whodb-cli doctor --connection prod --format json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		format, err := resolveDoctorFormat(doctorFormat)
		if err != nil {
			return err
		}
		quiet := doctorQuiet || format == output.FormatJSON

		report, err := doctor.Run(context.Background(), doctor.Options{
			Connection: doctorConnection,
			Schema:     doctorSchema,
		})
		if err != nil {
			return err
		}

		if format == output.FormatJSON {
			return writeAutomationEnvelope(cmd, "doctor", report)
		}
		if !quiet {
			writeDoctorTable(cmd, report)
		}
		return nil
	},
}

func resolveDoctorFormat(value string) (output.Format, error) {
	switch value {
	case "", "table", "auto":
		return output.FormatTable, nil
	case "json":
		return output.FormatJSON, nil
	default:
		return "", fmt.Errorf("invalid --format %q (expected table or json)", value)
	}
}

func writeDoctorTable(cmd *cobra.Command, report doctor.Report) {
	out := newCommandOutput(cmd, output.FormatTable, false)
	if report.Connection.Name != "" {
		out.Info("Connection: %s (%s)", report.Connection.Name, report.Connection.Type)
	}
	for _, check := range report.Checks {
		message := check.Message
		if message == "" && check.Count != 0 {
			message = fmt.Sprintf("count=%d", check.Count)
		}
		if message == "" && check.Schema != "" {
			message = fmt.Sprintf("schema=%s", check.Schema)
		}
		out.Info("%s: %s %s", check.Name, check.Status, message)
	}
}

func init() {
	rootCmd.AddCommand(doctorCmd)
	doctorCmd.AddCommand(platformDoctorCmd)

	doctorCmd.Flags().StringVarP(&doctorConnection, "connection", "c", "", "connection name to inspect")
	doctorCmd.Flags().StringVarP(&doctorSchema, "schema", "s", "", "schema override for metadata checks")
	doctorCmd.Flags().StringVarP(&doctorFormat, "format", "f", "table", "output format: table or json")
	doctorCmd.Flags().BoolVarP(&doctorQuiet, "quiet", "q", false, "suppress informational messages")
	platformDoctorCmd.Flags().StringVar(&platformHost, "host", "", "hosted WhoDB URL (default app.whodb.com)")
	platformDoctorCmd.Flags().StringVarP(&platformFormat, "format", "f", "auto", "output format: auto, table, plain, json, ndjson, csv")
	platformDoctorCmd.Flags().BoolVarP(&platformQuiet, "quiet", "q", false, "suppress informational messages")

	doctorCmd.RegisterFlagCompletionFunc("connection", completeConnectionNames)
	doctorCmd.RegisterFlagCompletionFunc("format", completeOutputFormats)
	platformDoctorCmd.RegisterFlagCompletionFunc("format", completeOutputFormats)
}
