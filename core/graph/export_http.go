package graph

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/clidey/whodb/core/src"
	"github.com/clidey/whodb/core/src/auth"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/xuri/excelize/v2"
)

// HandleExport handles HTTP requests for data export (CSV or Excel)
func HandleExport(w http.ResponseWriter, r *http.Request) {
	// Only allow POST requests
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse JSON body
	var req struct {
		Schema       string                   `json:"schema"`
		StorageUnit  string                   `json:"storageUnit"`
		Delimiter    string                   `json:"delimiter,omitempty"`
		Format       string                   `json:"format,omitempty"`
		SelectedRows []map[string]interface{} `json:"selectedRows,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	schema := req.Schema
	storageUnit := req.StorageUnit
	delimiter := req.Delimiter
	format := req.Format

	// Default format to CSV if not specified
	if format == "" {
		format = "csv"
	}

	// Validate format
	if format != "csv" && format != "excel" {
		http.Error(w, "Invalid format. Must be 'csv' or 'excel'", http.StatusBadRequest)
		return
	}

	// Default to comma if no delimiter specified for CSV
	if delimiter == "" {
		delimiter = ","
	}

	if schema == "" || storageUnit == "" {
		http.Error(w, "Missing required parameters", http.StatusBadRequest)
		return
	}

	// Get credentials
	credentials := auth.GetCredentials(r.Context())
	if credentials == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Set up plugin
	pluginConfig := engine.NewPluginConfig(credentials)
	plugin := src.MainEngine.Choose(engine.DatabaseType(credentials.Type))

	if format == "excel" {
		// Handle Excel export
		handleExcelExport(w, plugin, pluginConfig, schema, storageUnit, req.SelectedRows)
	} else {
		// Handle CSV export
		handleCSVExport(w, plugin, pluginConfig, schema, storageUnit, delimiter, req.SelectedRows)
	}
}

func handleCSVExport(w http.ResponseWriter, plugin *engine.Plugin, pluginConfig *engine.PluginConfig, schema, storageUnit, delimiter string, selectedRows []map[string]interface{}) {
	// Validate delimiter is a single character
	if len(delimiter) != 1 {
		http.Error(w, "Delimiter must be a single character", http.StatusBadRequest)
		return
	}

	// Convert delimiter to rune
	delimRune := rune(delimiter[0])

	// Set response headers for CSV download
	filename := fmt.Sprintf("%s_%s.csv", schema, storageUnit)
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	// Create CSV writer that writes directly to response
	csvWriter := csv.NewWriter(w)
	csvWriter.Comma = delimRune // Use user-specified delimiter
	defer csvWriter.Flush()

	// Track if we've written anything
	rowsWritten := 0

	// Create writer function that streams to HTTP response
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
	err := plugin.ExportCSV(pluginConfig, schema, storageUnit, writerFunc, selectedRows)
	if err != nil {
		// If we haven't written anything yet, we can send an error
		if rowsWritten == 0 {
			http.Error(w, fmt.Sprintf("Export failed: %v", err), http.StatusInternalServerError)
			return
		}
		// Otherwise, we've already started streaming, so log the error
		fmt.Fprintf(w, "\n# ERROR: %v\n", err)
	}
}

func handleExcelExport(w http.ResponseWriter, plugin *engine.Plugin, pluginConfig *engine.PluginConfig, schema, storageUnit string, selectedRows []map[string]interface{}) {
	// Create a new Excel file
	f := excelize.NewFile()
	defer f.Close()

	// Create a new sheet
	sheetName := "Data"
	f.SetSheetName("Sheet1", sheetName)

	// Collect all rows in memory (Excel requires full data)
	var allRows [][]string
	var headers []string

	// Create writer function that collects rows
	writerFunc := func(row []string) error {
		if len(headers) == 0 {
			// First row is headers
			headers = row
			allRows = append(allRows, row)
		} else {
			allRows = append(allRows, row)
		}
		return nil
	}

	// Export rows using the plugin
	err := plugin.ExportCSV(pluginConfig, schema, storageUnit, writerFunc, selectedRows)
	if err != nil {
		http.Error(w, fmt.Sprintf("Export failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Write data to Excel
	for rowIdx, row := range allRows {
		for colIdx, value := range row {
			cell, _ := excelize.CoordinatesToCellName(colIdx+1, rowIdx+1)
			f.SetCellValue(sheetName, cell, value)
		}
	}

	// Format headers (bold, background color)
	if len(headers) > 0 {
		headerStyle, _ := f.NewStyle(&excelize.Style{
			Font: &excelize.Font{Bold: true},
			Fill: excelize.Fill{
				Type:    "pattern",
				Color:   []string{"#E0E0E0"},
				Pattern: 1,
			},
		})

		// Apply style to header row
		endCell, _ := excelize.CoordinatesToCellName(len(headers), 1)
		f.SetCellStyle(sheetName, "A1", endCell, headerStyle)

		// Auto-fit columns
		for i := 0; i < len(headers); i++ {
			col, _ := excelize.ColumnNumberToName(i + 1)
			f.SetColWidth(sheetName, col, col, 15)
		}
	}

	// Set response headers for Excel download
	filename := fmt.Sprintf("%s_%s.xlsx", schema, storageUnit)
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	// Write Excel file to response
	if err := f.Write(w); err != nil {
		http.Error(w, fmt.Sprintf("Failed to write Excel file: %v", err), http.StatusInternalServerError)
		return
	}
}
