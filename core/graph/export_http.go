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
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/xuri/excelize/v2"

	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/source"
	"github.com/clidey/whodb/core/src/sourcecatalog"
)

const (
	// MaxExcelRows is the maximum number of rows allowed in an Excel export.
	MaxExcelRows = 100000 // 100k rows

	// DefaultCSVDelimiter is the delimiter used for CSV exports.
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
		return errors.New("delimiter must be a single character")
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
		// Columns lists the selected rows' columns in grid order with the
		// types the grid showed, so the export keeps both.
		Columns []struct {
			Name string `json:"name"`
			Type string `json:"type"`
		} `json:"columns,omitempty"`
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

	var selection *source.ExportSelection
	if len(req.SelectedRows) > 0 {
		selection = &source.ExportSelection{Rows: req.SelectedRows}
		for _, column := range req.Columns {
			selection.Columns = append(selection.Columns, source.Column{Name: column.Name, Type: column.Type})
		}
	}

	if resolvedRef == nil {
		if format == "ndjson" {
			handleSelectedRowsNDJSONExport(w, fileBaseName, req.SelectedRows)
			return
		}

		spec, _, err := getSourceSpecForContext(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		metadata, _ := sourcecatalog.ResolveSessionMetadata(spec.ID, spec.Connector)
		rows := func(writer source.ExportWriter) error {
			return selection.Write(writer, selection.FreeTextColumns(metadata))
		}
		if format == "excel" {
			handleExcelExport(w, fileBaseName, rows)
		} else {
			handleCSVExport(w, r, fileBaseName, delimiter, rows)
		}
		return
	}

	spec, session, err := getSourceSessionForContext(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	if err := source.ValidateObjectActionSupported(spec, resolvedRef.Kind, source.ActionViewRows); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	scope := sourceAuditScopeFromContext(r.Context(), spec)

	exporter, ok := source.AsTabularExporter(scope, session)
	if !ok {
		http.Error(w, "Export not supported for this source", http.StatusBadRequest)
		return
	}
	rows := func(writer source.ExportWriter) error {
		return exporter.ExportRows(r.Context(), *resolvedRef, writer, selection)
	}

	switch format {
	case "excel":
		handleExcelExport(w, fileBaseName, rows)
	case "ndjson":
		ndjsonExporter, ok := source.AsNDJSONExporter(scope, session)
		if !ok {
			http.Error(w, "NDJSON export not supported for this source", http.StatusBadRequest)
			return
		}
		handleNDJSONExport(r.Context(), w, ndjsonExporter, *resolvedRef, fileBaseName, req.SelectedRows)
	default:
		handleCSVExport(w, r, fileBaseName, delimiter, rows)
	}
}

// exportRows streams an export's header and rows into the given writer.
type exportRows func(writer source.ExportWriter) error

// trackingResponseWriter records whether any of the response has been sent.
type trackingResponseWriter struct {
	http.ResponseWriter
	wrote bool
}

func (t *trackingResponseWriter) Write(data []byte) (int, error) {
	t.wrote = true
	return t.ResponseWriter.Write(data)
}

func handleCSVExport(w http.ResponseWriter, r *http.Request, fileBaseName string, delimiter string, rows exportRows) {
	filename := fileBaseName + ".csv"
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	response := &trackingResponseWriter{ResponseWriter: w}
	csvWriter := csv.NewWriter(response)
	csvWriter.Comma = rune(delimiter[0])

	rowsWritten := 0
	writerFunc := func(row []string, text []bool) error {
		if err := csvWriter.Write(common.EscapeFormulaRow(row, text)); err != nil {
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

	err := rows(writerFunc)
	if err == nil {
		csvWriter.Flush()
		err = csvWriter.Error()
	}
	if err == nil {
		return
	}

	if !response.wrote {
		for _, header := range []string{"Content-Disposition", "Cache-Control", "Pragma", "Expires"} {
			w.Header().Del(header)
		}
		http.Error(w, "Export failed. Please check your file and try again.", http.StatusInternalServerError)
		return
	}

	// Part of the file is already on the wire with a 200 status. Dropping the
	// connection is the only way left to tell the client the file is
	// incomplete. The desktop webview handler has no connection to drop and
	// does not recover panics, so there the response just ends.
	if r.Context().Value(http.ServerContextKey) != nil {
		panic(http.ErrAbortHandler)
	}
}

// handleExcelExport writes every value as a plain string cell. Spreadsheets
// never evaluate string cells, so values need no formula guard here.
func handleExcelExport(w http.ResponseWriter, fileBaseName string, rows exportRows) {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()

	sheetName := "Data"
	index, err := f.NewSheet(sheetName)
	if err != nil {
		http.Error(w, "Failed to create Excel sheet", http.StatusInternalServerError)
		return
	}
	f.SetActiveSheet(index)
	_ = f.DeleteSheet("Sheet1")

	streamWriter, err := f.NewStreamWriter(sheetName)
	if err != nil {
		http.Error(w, "Failed to create Excel stream writer", http.StatusInternalServerError)
		return
	}

	var headers []string
	rowCount := 0
	currentRow := 1

	writerFunc := func(row []string, _ []bool) error {
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

	if err := rows(writerFunc); err != nil {
		http.Error(w, "Export failed", http.StatusInternalServerError)
		return
	}

	if len(headers) > 0 {
		for i := range headers {
			_ = streamWriter.SetColWidth(i+1, i+1, 15)
		}
	}

	// Flush the stream writer
	if err := streamWriter.Flush(); err != nil {
		http.Error(w, "Failed to flush Excel data", http.StatusInternalServerError)
		return
	}

	filename := fileBaseName + ".xlsx"
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
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
	filename := fileBaseName + ".ndjson"

	w.Header().Set("Content-Type", "application/x-ndjson; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
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

func handleSelectedRowsNDJSONExport(w http.ResponseWriter, fileBaseName string, selectedRows []map[string]any) {
	filename := fileBaseName + ".ndjson"

	w.Header().Set("Content-Type", "application/x-ndjson; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
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
