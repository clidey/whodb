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

package graph

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/clidey/whodb/core/src"
	"github.com/clidey/whodb/core/src/auth"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/xuri/excelize/v2"
)

const (
	// Maximum rows for Excel export to prevent memory issues
	MaxExcelRows = 100000 // 100k rows

	// Default CSV delimiter
	DefaultCSVDelimiter = ","
)

// NDJSONExporter allows plugins to stream newline-delimited JSON.
type NDJSONExporter interface {
	ExportDataNDJSON(config *engine.PluginConfig, schema string, storageUnit string, writer func(string) error, selectedRows []map[string]any) error
}

// InvalidDelimiters contains characters that cannot be used as CSV delimiters
var InvalidDelimiters = map[byte]string{
	'=':  "formula indicator",
	'+':  "formula indicator",
	'-':  "formula indicator",
	'@':  "formula indicator",
	'\t': "tab character",
	'\r': "carriage return",
	'\'': "single quote (used for escaping)",
	'"':  "double quote (CSV escape character)",
}

// validateDelimiter checks if a delimiter is valid for CSV export
func validateDelimiter(delimiter string) error {
	if len(delimiter) != 1 {
		return fmt.Errorf("delimiter must be a single character")
	}

	delimChar := delimiter[0]
	if reason, invalid := InvalidDelimiters[delimChar]; invalid {
		return fmt.Errorf("invalid delimiter '%c': %s", delimChar, reason)
	}

	return nil
}

// HandleExport handles HTTP requests for data export (CSV, Excel, or NDJSON).
func HandleExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Schema       string           `json:"schema"`
		StorageUnit  string           `json:"storageUnit"`
		Delimiter    string           `json:"delimiter,omitempty"`
		Format       string           `json:"format,omitempty"`
		SelectedRows []map[string]any `json:"selectedRows,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	schema := req.Schema
	storageUnit := req.StorageUnit
	delimiter := req.Delimiter
	format := strings.ToLower(strings.TrimSpace(req.Format))

	if format == "" {
		format = "csv"
	}

	switch format {
	case "csv", "excel", "ndjson":
	default:
		http.Error(w, "Invalid format. Must be 'csv', 'excel', or 'ndjson'", http.StatusBadRequest)
		return
	}

	if format == "csv" {
		if delimiter == "" {
			delimiter = DefaultCSVDelimiter
		}

		if err := validateDelimiter(delimiter); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	if schema == "" && storageUnit == "" {
		http.Error(w, "Missing required parameters", http.StatusBadRequest)
		return
	}

	credentials := auth.GetCredentials(r.Context())
	if credentials == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	pluginConfig := engine.NewPluginConfig(credentials)
	plugin := src.MainEngine.Choose(engine.DatabaseType(credentials.Type))

	switch format {
	case "excel":
		handleExcelExport(w, plugin, pluginConfig, schema, storageUnit, req.SelectedRows)
	case "ndjson":
		handleNDJSONExport(w, plugin, pluginConfig, schema, storageUnit, req.SelectedRows)
	default:
		handleCSVExport(w, plugin, pluginConfig, schema, storageUnit, delimiter, req.SelectedRows)
	}
}

func handleCSVExport(w http.ResponseWriter, plugin *engine.Plugin, pluginConfig *engine.PluginConfig, schema, storageUnit, delimiter string, selectedRows []map[string]any) {
	delimRune := rune(delimiter[0])

	var filename string
	if schema != "" {
		filename = fmt.Sprintf("%s_%s.csv", schema, storageUnit)
	} else {
		filename = fmt.Sprintf("%s.csv", storageUnit)
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	csvWriter := csv.NewWriter(w)
	csvWriter.Comma = delimRune // Use user-specified delimiter
	defer csvWriter.Flush()

	// Track if we've written anything
	rowsWritten := 0

	writerFunc := func(row []string) error {
		if err := csvWriter.Write(row); err != nil {
			return err
		}
		rowsWritten++

		// Flush every 100 rows to ensure streaming
		if rowsWritten%100 == 0 {
			csvWriter.Flush()
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
		return nil
	}

	// Export rows (all or selected) using the unified method
	err := plugin.ExportData(pluginConfig, schema, storageUnit, writerFunc, selectedRows)
	if err != nil {
		if rowsWritten == 0 {
			http.Error(w, "Export failed. Please check your file and try again.", http.StatusInternalServerError)
			return
		}
	}
}

func handleExcelExport(w http.ResponseWriter, plugin *engine.Plugin, pluginConfig *engine.PluginConfig, schema, storageUnit string, selectedRows []map[string]any) {
	f := excelize.NewFile()
	defer f.Close()

	sheetName := "Data"
	index, err := f.NewSheet(sheetName)
	if err != nil {
		http.Error(w, "Failed to create Excel sheet", http.StatusInternalServerError)
		return
	}
	f.SetActiveSheet(index)
	f.DeleteSheet("Sheet1")

	streamWriter, err := f.NewStreamWriter(sheetName)
	if err != nil {
		http.Error(w, "Failed to create Excel stream writer", http.StatusInternalServerError)
		return
	}

	var headers []string
	rowCount := 0
	currentRow := 1

	writerFunc := func(row []string) error {
		if rowCount >= MaxExcelRows {
			return fmt.Errorf("excel export limit exceeded: maximum %d rows allowed", MaxExcelRows)
		}

		if len(headers) == 0 {
			headers = row
			styleID, _ := f.NewStyle(&excelize.Style{
				Font: &excelize.Font{Bold: true},
				Fill: excelize.Fill{
					Type:    "pattern",
					Color:   []string{"#E0E0E0"},
					Pattern: 1,
				},
			})

			// Write headers with style
			cells := make([]interface{}, len(row))
			for i, header := range row {
				cells[i] = excelize.Cell{StyleID: styleID, Value: header}
			}

			cell, _ := excelize.CoordinatesToCellName(1, currentRow)
			if err := streamWriter.SetRow(cell, cells); err != nil {
				return err
			}
		} else {
			// Write data row
			cells := make([]interface{}, len(row))
			for i, value := range row {
				cells[i] = value
			}

			cell, _ := excelize.CoordinatesToCellName(1, currentRow)
			if err := streamWriter.SetRow(cell, cells); err != nil {
				return err
			}
		}

		rowCount++
		currentRow++
		return nil
	}

	// Export rows using the plugin
	err = plugin.ExportData(pluginConfig, schema, storageUnit, writerFunc, selectedRows)
	if err != nil {
		http.Error(w, "Export failed", http.StatusInternalServerError)
		return
	}

	if len(headers) > 0 {
		for i := 0; i < len(headers); i++ {
			streamWriter.SetColWidth(i+1, i+1, 15)
		}
	}

	// Flush the stream writer
	if err := streamWriter.Flush(); err != nil {
		http.Error(w, "Failed to flush Excel data", http.StatusInternalServerError)
		return
	}

	// Include schema in filename only if it exists (for SQLite, schema is empty)
	var filename string
	if schema != "" {
		filename = fmt.Sprintf("%s_%s.xlsx", schema, storageUnit)
	} else {
		filename = fmt.Sprintf("%s.xlsx", storageUnit)
	}
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	// Write Excel file to response
	if err := f.Write(w); err != nil {
		http.Error(w, "Failed to generate Excel file", http.StatusInternalServerError)
		return
	}
}

func handleNDJSONExport(w http.ResponseWriter, plugin *engine.Plugin, pluginConfig *engine.PluginConfig, schema, storageUnit string, selectedRows []map[string]any) {
	exporter, ok := plugin.PluginFunctions.(NDJSONExporter)
	if !ok {
		http.Error(w, "NDJSON export not supported for this database", http.StatusBadRequest)
		return
	}

	var filename string
	if schema != "" {
		filename = fmt.Sprintf("%s_%s.ndjson", schema, storageUnit)
	} else {
		filename = fmt.Sprintf("%s.ndjson", storageUnit)
	}

	w.Header().Set("Content-Type", "application/x-ndjson; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	rowsWritten := 0
	writerFunc := func(line string) error {
		if _, err := w.Write([]byte(line + "\n")); err != nil {
			return err
		}
		rowsWritten++
		if rowsWritten%100 == 0 {
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
		return nil
	}

	if err := exporter.ExportDataNDJSON(pluginConfig, schema, storageUnit, writerFunc, selectedRows); err != nil && rowsWritten == 0 {
		http.Error(w, "Export failed. Please try again.", http.StatusInternalServerError)
		return
	}
}
