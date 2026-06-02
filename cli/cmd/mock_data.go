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
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	dbmgr "github.com/clidey/whodb/cli/internal/database"
	"github.com/clidey/whodb/cli/pkg/output"
	coremockdata "github.com/clidey/whodb/core/src/mockdata"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	mockDataConnection     string
	mockDataSchema         string
	mockDataTable          string
	mockDataRows           int
	mockDataFormat         string
	mockDataQuiet          bool
	mockDataOverwrite      bool
	mockDataAnalyzeOnly    bool
	mockDataConfirm        bool
	mockDataFKDensityRatio int
)

var mockDataCmd = &cobra.Command{
	Use:           "mock-data",
	Short:         "Generate FK-aware mock data",
	SilenceUsage:  true,
	SilenceErrors: true,
	Long: `Generate mock data for a target table and any required parent tables.

The command always analyzes dependencies first. Use --analyze to inspect the
plan without writing data.

For safety, writes require an explicit connection and either:
  - an interactive confirmation prompt, or
  - --yes for non-interactive/automated runs.`,
	Example: `  # Analyze the dependency plan without writing data
  whodb-cli mock-data --connection mydb --table orders --rows 50 --analyze

  # Generate data with interactive confirmation
  whodb-cli mock-data --connection mydb --schema public --table orders --rows 50

  # Overwrite existing rows and skip the confirmation prompt
  whodb-cli mock-data --connection mydb --table orders --rows 50 --overwrite --yes

  # Emit machine-readable JSON
  whodb-cli mock-data --connection mydb --table orders --rows 50 --yes --format json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if strings.TrimSpace(mockDataConnection) == "" {
			return fmt.Errorf("--connection is required")
		}
		if strings.TrimSpace(mockDataTable) == "" {
			return fmt.Errorf("--table is required")
		}
		if mockDataRows <= 0 {
			return fmt.Errorf("--rows must be greater than 0")
		}

		format, err := output.ParseFormat(mockDataFormat)
		if err != nil {
			return err
		}
		if format == output.FormatCSV {
			return fmt.Errorf("csv format is not supported for mock-data")
		}
		quiet := mockDataQuiet || format == output.FormatJSON

		out := newCommandOutput(cmd, format, quiet)

		mgr, err := dbmgr.NewManager()
		if err != nil {
			return fmt.Errorf("cannot initialize database manager: %w", err)
		}

		conn, _, err := mgr.ResolveConnection(mockDataConnection)
		if err != nil {
			return err
		}

		spinner := output.NewSpinner(fmt.Sprintf("Connecting to %s...", conn.Type))
		if quiet {
			spinner = nil
		}
		spinner.Start()
		if err := mgr.Connect(conn); err != nil {
			spinner.StopWithError("Connection failed")
			return fmt.Errorf("cannot connect to database: %w", err)
		}
		spinner.Stop()
		defer mgr.Disconnect()

		schema, err := resolveMockDataSchema(mgr, conn, out, mockDataSchema)
		if err != nil {
			return err
		}

		spinner = output.NewSpinner("Analyzing mock data dependencies...")
		if quiet {
			spinner = nil
		}
		spinner.Start()
		analysis, err := mgr.AnalyzeMockDataDependencies(schema, mockDataTable, mockDataRows, mockDataFKDensityRatio)
		if err != nil {
			spinner.StopWithError("Analysis failed")
			return err
		}
		spinner.StopWithSuccess("Analysis complete")

		payload := buildMockDataPayload(conn.Name, schema, mockDataTable, mockDataRows, mockDataOverwrite, mockDataFKDensityRatio, analysis, nil)

		if mockDataAnalyzeOnly {
			if format == output.FormatJSON {
				return writeAutomationEnvelope(cmd, "mock-data.analyze", payload)
			}
			printMockDataAnalysis(cmd.OutOrStdout(), payload)
			return nil
		}

		if format != output.FormatJSON {
			printMockDataAnalysis(cmd.OutOrStdout(), payload)
		}

		approved, err := confirmMockDataRun(payload, mockDataConfirm)
		if err != nil {
			return err
		}
		if !approved {
			out.Info("Mock data generation cancelled")
			return nil
		}

		spinner = output.NewSpinner("Generating mock data...")
		if quiet {
			spinner = nil
		}
		spinner.Start()
		result, err := mgr.GenerateMockData(schema, mockDataTable, mockDataRows, mockDataOverwrite, mockDataFKDensityRatio)
		if err != nil {
			spinner.StopWithError("Mock data generation failed")
			return err
		}
		spinner.StopWithSuccess("Mock data generation complete")

		payload.Result = buildMockDataResult(result)

		if format == output.FormatJSON {
			return writeAutomationEnvelope(cmd, "mock-data.generate", payload)
		}

		printMockDataResult(cmd.OutOrStdout(), payload)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(mockDataCmd)

	mockDataCmd.Flags().StringVarP(&mockDataConnection, "connection", "c", "", "connection name to use")
	mockDataCmd.Flags().StringVarP(&mockDataSchema, "schema", "s", "", "target schema (optional)")
	mockDataCmd.Flags().StringVarP(&mockDataTable, "table", "t", "", "target table or collection name (required)")
	mockDataCmd.Flags().IntVarP(&mockDataRows, "rows", "r", 0, "number of target rows to generate (required)")
	mockDataCmd.Flags().BoolVar(&mockDataOverwrite, "overwrite", false, "clear existing target and dependent rows before generating")
	mockDataCmd.Flags().BoolVar(&mockDataAnalyzeOnly, "analyze", false, "show the dependency plan without generating data")
	mockDataCmd.Flags().BoolVarP(&mockDataConfirm, "yes", "y", false, "skip the confirmation prompt")
	mockDataCmd.Flags().IntVar(&mockDataFKDensityRatio, "fk-density-ratio", 0, "parent row density ratio override (0 uses the backend default)")
	mockDataCmd.Flags().StringVarP(&mockDataFormat, "format", "f", "table", "output format: table or json")
	mockDataCmd.Flags().BoolVarP(&mockDataQuiet, "quiet", "q", false, "suppress informational messages")

	mockDataCmd.RegisterFlagCompletionFunc("connection", completeConnectionNames)
	mockDataCmd.RegisterFlagCompletionFunc("format", completeAuditFormats)
}

type mockDataCommandOutput struct {
	Connection     string                 `json:"connection"`
	Schema         string                 `json:"schema,omitempty"`
	StorageUnit    string                 `json:"storageUnit"`
	RowCount       int                    `json:"rowCount"`
	Overwrite      bool                   `json:"overwriteExisting"`
	FKDensityRatio int                    `json:"fkDensityRatio,omitempty"`
	Analysis       mockDataAnalysisOutput `json:"analysis"`
	Result         *mockDataResultOutput  `json:"result,omitempty"`
}

type mockDataAnalysisOutput struct {
	GenerationOrder []string                  `json:"generationOrder"`
	Tables          []mockDataTableInfoOutput `json:"tables"`
	TotalRows       int                       `json:"totalRows"`
	Warnings        []string                  `json:"warnings,omitempty"`
}

type mockDataTableInfoOutput struct {
	Table            string `json:"table"`
	RowsToGenerate   int    `json:"rowsToGenerate"`
	IsBlocked        bool   `json:"isBlocked"`
	UsesExistingData bool   `json:"usesExistingData"`
}

type mockDataResultOutput struct {
	AmountGenerated int                         `json:"amountGenerated"`
	Details         []mockDataTableDetailOutput `json:"details,omitempty"`
	Warnings        []string                    `json:"warnings,omitempty"`
}

type mockDataTableDetailOutput struct {
	Table            string `json:"table"`
	RowsGenerated    int    `json:"rowsGenerated"`
	UsedExistingData bool   `json:"usedExistingData"`
}

func resolveMockDataSchema(
	mgr *dbmgr.Manager,
	conn *dbmgr.Connection,
	out *output.Writer,
	explicitSchema string,
) (string, error) {
	if strings.TrimSpace(explicitSchema) != "" {
		return explicitSchema, nil
	}
	if strings.TrimSpace(conn.Schema) != "" {
		return conn.Schema, nil
	}

	schemas, err := mgr.GetSchemas()
	if err != nil || len(schemas) == 0 {
		return "", nil
	}

	out.Info("Using schema: %s", schemas[0])
	return schemas[0], nil
}

func buildMockDataPayload(
	connection string,
	schema string,
	table string,
	rowCount int,
	overwrite bool,
	fkDensityRatio int,
	analysis *coremockdata.DependencyAnalysis,
	result *mockDataResultOutput,
) *mockDataCommandOutput {
	outputValue := &mockDataCommandOutput{
		Connection:     connection,
		Schema:         schema,
		StorageUnit:    table,
		RowCount:       rowCount,
		Overwrite:      overwrite,
		FKDensityRatio: fkDensityRatio,
		Result:         result,
	}

	if analysis == nil {
		return outputValue
	}

	outputValue.Analysis.GenerationOrder = append(outputValue.Analysis.GenerationOrder, analysis.GenerationOrder...)
	outputValue.Analysis.TotalRows = analysis.TotalRows
	outputValue.Analysis.Warnings = append(outputValue.Analysis.Warnings, analysis.Warnings...)
	outputValue.Analysis.Tables = make([]mockDataTableInfoOutput, 0, len(analysis.Tables))
	for _, tableInfo := range analysis.Tables {
		outputValue.Analysis.Tables = append(outputValue.Analysis.Tables, mockDataTableInfoOutput{
			Table:            tableInfo.Table,
			RowsToGenerate:   tableInfo.RowCount,
			IsBlocked:        tableInfo.IsBlocked,
			UsesExistingData: tableInfo.UsesExistingData,
		})
	}

	return outputValue
}

func buildMockDataResult(result *coremockdata.GenerationResult) *mockDataResultOutput {
	if result == nil {
		return nil
	}

	outputValue := &mockDataResultOutput{
		AmountGenerated: result.TotalGenerated,
		Warnings:        append([]string(nil), result.Warnings...),
		Details:         make([]mockDataTableDetailOutput, 0, len(result.Details)),
	}
	for _, detail := range result.Details {
		outputValue.Details = append(outputValue.Details, mockDataTableDetailOutput{
			Table:            detail.Table,
			RowsGenerated:    detail.RowsGenerated,
			UsedExistingData: detail.UsedExistingData,
		})
	}

	return outputValue
}

func printMockDataAnalysis(out io.Writer, payload *mockDataCommandOutput) {
	fmt.Fprintf(out, "\nMock Data Plan\n")
	fmt.Fprintf(out, "  Connection: %s\n", payload.Connection)
	if payload.Schema != "" {
		fmt.Fprintf(out, "  Schema: %s\n", payload.Schema)
	}
	fmt.Fprintf(out, "  Target: %s\n", payload.StorageUnit)
	fmt.Fprintf(out, "  Requested rows: %d\n", payload.RowCount)
	fmt.Fprintf(out, "  Total rows across dependencies: %d\n", payload.Analysis.TotalRows)
	fmt.Fprintf(out, "  Overwrite existing rows: %t\n", payload.Overwrite)

	if len(payload.Analysis.GenerationOrder) > 0 {
		fmt.Fprintf(out, "  Generation order: %s\n", strings.Join(payload.Analysis.GenerationOrder, " -> "))
	}

	fmt.Fprintln(out)
	for _, tableInfo := range payload.Analysis.Tables {
		status := "generate"
		if tableInfo.UsesExistingData {
			status = "use existing"
		}
		fmt.Fprintf(out, "  - %-24s %4d rows  [%s]\n", tableInfo.Table, tableInfo.RowsToGenerate, status)
	}

	if len(payload.Analysis.Warnings) > 0 {
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Warnings:")
		for _, warning := range payload.Analysis.Warnings {
			fmt.Fprintf(out, "  - %s\n", warning)
		}
	}
	fmt.Fprintln(out)
}

func printMockDataResult(out io.Writer, payload *mockDataCommandOutput) {
	if payload.Result == nil {
		return
	}

	fmt.Fprintf(out, "Generated %d rows\n", payload.Result.AmountGenerated)
	for _, detail := range payload.Result.Details {
		mode := "generated"
		if detail.UsedExistingData {
			mode = "existing"
		}
		fmt.Fprintf(out, "  - %-24s %4d rows  [%s]\n", detail.Table, detail.RowsGenerated, mode)
	}
}

func confirmMockDataRun(payload *mockDataCommandOutput, skipPrompt bool) (bool, error) {
	if skipPrompt {
		return true, nil
	}

	if !isInteractiveInput() {
		return false, fmt.Errorf("mock-data requires --yes when stdin is not interactive")
	}

	fmt.Fprintln(os.Stderr, "This command will write mock data to the target database.")
	if payload.Overwrite {
		fmt.Fprintln(os.Stderr, "Overwrite mode will clear the target table and dependent child tables first.")
	}
	fmt.Fprintf(os.Stderr, "Proceed with generating mock data for %s? [y/N]: ", payload.StorageUnit)

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("reading confirmation: %w", err)
	}

	return isAffirmativeConfirmation(response), nil
}

func isInteractiveInput() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}
