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

package graph

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/source"
	"github.com/xuri/excelize/v2"
)

const (
	// Maximum rows for Excel export to prevent memory issues
	MaxExcelRows = 100000 // 100k rows

	// Default CSV delimiter
	DefaultCSVDelimiter = ","
)

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
		Ref          *model.SourceObjectRefInput `json:"ref"`
		FileBaseName string                      `json:"fileBaseName,omitempty"`
		Delimiter    string                      `json:"delimiter,omitempty"`
		Format       string                      `json:"format,omitempty"`
		SelectedRows []map[string]any            `json:"selectedRows,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

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

	if req.Ref == nil && len(req.SelectedRows) == 0 {
		http.Error(w, "Missing required parameters", http.StatusBadRequest)
		return
	}
	if req.Ref == nil && strings.TrimSpace(req.FileBaseName) == "" {
		http.Error(w, "Missing required parameters", http.StatusBadRequest)
		return
	}

	resolvedRef := sourceRefFromInput(req.Ref)
	fileBaseName := sourceFileBaseName(resolvedRef, strings.TrimSpace(req.FileBaseName))
	if fileBaseName == "" {
		http.Error(w, "Missing required parameters", http.StatusBadRequest)
		return
	}

	if resolvedRef == nil {
		switch format {
		case "excel":
			handleSelectedRowsExcelExport(w, fileBaseName, req.SelectedRows)
		case "ndjson":
			handleSelectedRowsNDJSONExport(w, fileBaseName, req.SelectedRows)
		default:
			handleSelectedRowsCSVExport(w, fileBaseName, delimiter, req.SelectedRows)
		}
		return
	}

	spec, session, err := getSourceSessionForContext(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	scope := sourceAuditScopeFromContext(r.Context(), spec)

	exporter, ok := source.AsTabularExporter(scope, session)
	if !ok {
		http.Error(w, "Export not supported for this source", http.StatusBadRequest)
		return
	}

	switch format {
	case "excel":
		handleExcelExport(r.Context(), w, exporter, *resolvedRef, fileBaseName, req.SelectedRows)
	case "ndjson":
		ndjsonExporter, ok := source.AsNDJSONExporter(scope, session)
		if !ok {
			http.Error(w, "NDJSON export not supported for this source", http.StatusBadRequest)
			return
		}
		handleNDJSONExport(r.Context(), w, ndjsonExporter, *resolvedRef, fileBaseName, req.SelectedRows)
	default:
		handleCSVExport(r.Context(), w, exporter, *resolvedRef, fileBaseName, delimiter, req.SelectedRows)
	}
}

func handleCSVExport(ctx context.Context, w http.ResponseWriter, exporter source.TabularExporter, ref source.ObjectRef, fileBaseName string, delimiter string, selectedRows []map[string]any) {
	delimRune := rune(delimiter[0])

	filename := fmt.Sprintf("%s.csv", fileBaseName)
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
	err := exporter.ExportRows(ctx, ref, writerFunc, selectedRows)
	if err != nil {
		if rowsWritten == 0 {
			http.Error(w, "Export failed. Please check your file and try again.", http.StatusInternalServerError)
			return
		}
	}
}

func handleExcelExport(ctx context.Context, w http.ResponseWriter, exporter source.TabularExporter, ref source.ObjectRef, fileBaseName string, selectedRows []map[string]any) {
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
			cells := make([]any, len(row))
			for i, header := range row {
				cells[i] = excelize.Cell{StyleID: styleID, Value: header}
			}

			cell, _ := excelize.CoordinatesToCellName(1, currentRow)
			if err := streamWriter.SetRow(cell, cells); err != nil {
				return err
			}
		} else {
			// Write data row
			cells := make([]any, len(row))
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
	err = exporter.ExportRows(ctx, ref, writerFunc, selectedRows)
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

	filename := fmt.Sprintf("%s.xlsx", fileBaseName)
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

func handleNDJSONExport(ctx context.Context, w http.ResponseWriter, exporter source.NDJSONExporter, ref source.ObjectRef, fileBaseName string, selectedRows []map[string]any) {
	filename := fmt.Sprintf("%s.ndjson", fileBaseName)

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

	if err := exporter.ExportRowsNDJSON(ctx, ref, writerFunc, selectedRows); err != nil && rowsWritten == 0 {
		http.Error(w, "Export failed. Please try again.", http.StatusInternalServerError)
		return
	}
}

func handleSelectedRowsCSVExport(w http.ResponseWriter, fileBaseName string, delimiter string, selectedRows []map[string]any) {
	headers, rows := selectedRowsTable(selectedRows)
	if len(headers) == 0 {
		http.Error(w, "Missing required parameters", http.StatusBadRequest)
		return
	}

	delimRune := rune(delimiter[0])
	filename := fmt.Sprintf("%s.csv", fileBaseName)
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	csvWriter := csv.NewWriter(w)
	csvWriter.Comma = delimRune
	defer csvWriter.Flush()

	if err := csvWriter.Write(headers); err != nil {
		http.Error(w, "Export failed. Please check your file and try again.", http.StatusInternalServerError)
		return
	}
	for _, row := range rows {
		if err := csvWriter.Write(row); err != nil {
			http.Error(w, "Export failed. Please check your file and try again.", http.StatusInternalServerError)
			return
		}
	}
}

func handleSelectedRowsExcelExport(w http.ResponseWriter, fileBaseName string, selectedRows []map[string]any) {
	headers, rows := selectedRowsTable(selectedRows)
	if len(headers) == 0 {
		http.Error(w, "Missing required parameters", http.StatusBadRequest)
		return
	}

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

	headerCells := make([]any, len(headers))
	styleID, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#E0E0E0"},
			Pattern: 1,
		},
	})
	for i, header := range headers {
		headerCells[i] = excelize.Cell{StyleID: styleID, Value: header}
	}
	if err := streamWriter.SetRow("A1", headerCells); err != nil {
		http.Error(w, "Export failed", http.StatusInternalServerError)
		return
	}

	for rowIndex, row := range rows {
		cells := make([]any, len(row))
		for i, value := range row {
			cells[i] = value
		}
		cell, _ := excelize.CoordinatesToCellName(1, rowIndex+2)
		if err := streamWriter.SetRow(cell, cells); err != nil {
			http.Error(w, "Export failed", http.StatusInternalServerError)
			return
		}
	}
	if err := streamWriter.Flush(); err != nil {
		http.Error(w, "Failed to flush Excel data", http.StatusInternalServerError)
		return
	}

	filename := fmt.Sprintf("%s.xlsx", fileBaseName)
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	if err := f.Write(w); err != nil {
		http.Error(w, "Failed to generate Excel file", http.StatusInternalServerError)
	}
}

func handleSelectedRowsNDJSONExport(w http.ResponseWriter, fileBaseName string, selectedRows []map[string]any) {
	filename := fmt.Sprintf("%s.ndjson", fileBaseName)

	w.Header().Set("Content-Type", "application/x-ndjson; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	for _, row := range selectedRows {
		line, err := json.Marshal(row)
		if err != nil {
			http.Error(w, "Export failed. Please try again.", http.StatusInternalServerError)
			return
		}
		if _, err := w.Write([]byte(string(line) + "\n")); err != nil {
			http.Error(w, "Export failed. Please try again.", http.StatusInternalServerError)
			return
		}
	}
}

func selectedRowsTable(selectedRows []map[string]any) ([]string, [][]string) {
	if len(selectedRows) == 0 {
		return nil, nil
	}

	headerSet := map[string]bool{}
	for _, row := range selectedRows {
		for key := range row {
			headerSet[key] = true
		}
	}

	headers := make([]string, 0, len(headerSet))
	for key := range headerSet {
		headers = append(headers, key)
	}
	sort.Strings(headers)

	rows := make([][]string, 0, len(selectedRows))
	for _, row := range selectedRows {
		values := make([]string, len(headers))
		for i, header := range headers {
			values[i] = formatSelectedRowValue(row[header])
		}
		rows = append(rows, values)
	}
	return headers, rows
}

func formatSelectedRowValue(value any) string {
	if value == nil {
		return ""
	}

	switch typed := value.(type) {
	case string:
		return common.EscapeFormula(typed)
	default:
		return common.EscapeFormula(fmt.Sprintf("%v", typed))
	}
}
