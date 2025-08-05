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

const (
	// Maximum rows for Excel export to prevent memory issues
	MaxExcelRows = 100000 // 100k rows
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

	// Sanitize delimiter to prevent injection attacks
	if len(delimiter) == 1 {
		delimChar := delimiter[0]
		if delimChar == '=' || delimChar == '+' || delimChar == '-' || delimChar == '@' || delimChar == '\t' || delimChar == '\r' {
			http.Error(w, "Invalid delimiter character", http.StatusBadRequest)
			return
		}
	}

	if schema == "" && storageUnit == "" {
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

	if plugin.Type == engine.DatabaseType_Redis {
		http.Error(w, "Export not supported for Redis databases", http.StatusBadRequest)
		return
	}

	if format == "excel" {
		// Handle Excel export
		handleExcelExport(w, plugin, pluginConfig, schema, storageUnit, req.SelectedRows)
	} else {
		// Handle CSV export
		handleCSVExport(w, plugin, pluginConfig, schema, storageUnit, delimiter, req.SelectedRows)
	}
}

func handleCSVExport(w http.ResponseWriter, plugin *engine.Plugin, pluginConfig *engine.PluginConfig, schema, storageUnit, delimiter string, selectedRows []map[string]any) {
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
		// Apply formula injection protection for CSV
		for i, value := range row {
			if len(value) > 0 {
				firstChar := value[0]
				if firstChar == '=' || firstChar == '+' || firstChar == '-' || firstChar == '@' || firstChar == '\t' || firstChar == '\r' {
					row[i] = "'" + value
				}
			}
		}

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
	// Create a new Excel file
	f := excelize.NewFile()
	defer f.Close()

	// Create a new sheet
	sheetName := "Data"
	index, err := f.NewSheet(sheetName)
	if err != nil {
		http.Error(w, "Failed to create Excel sheet", http.StatusInternalServerError)
		return
	}
	f.SetActiveSheet(index)
	f.DeleteSheet("Sheet1")

	// Get stream writer for efficient memory usage
	streamWriter, err := f.NewStreamWriter(sheetName)
	if err != nil {
		http.Error(w, "Failed to create Excel stream writer", http.StatusInternalServerError)
		return
	}

	var headers []string
	rowCount := 0
	currentRow := 1

	// Create writer function that streams rows directly
	writerFunc := func(row []string) error {
		if rowCount >= MaxExcelRows {
			return fmt.Errorf("excel export limit exceeded: maximum %d rows allowed", MaxExcelRows)
		}

		// Escape formula injection for Excel
		for i, value := range row {
			if len(value) > 0 {
				firstChar := value[0]
				if firstChar == '=' || firstChar == '+' || firstChar == '-' || firstChar == '@' || firstChar == '\t' || firstChar == '\r' {
					row[i] = "'" + value
				}
			}
		}

		if len(headers) == 0 {
			// First row is headers
			headers = row
			// Create header style
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

	// Set column widths
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

	// Set response headers for Excel download
	filename := fmt.Sprintf("%s_%s.xlsx", schema, storageUnit)
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
