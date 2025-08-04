package graph

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/clidey/whodb/core/src"
	"github.com/clidey/whodb/core/src/auth"
	"github.com/clidey/whodb/core/src/engine"
)

// HandleCSVExport handles HTTP requests for CSV export with streaming
func HandleCSVExport(w http.ResponseWriter, r *http.Request) {
	// Only allow POST requests
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse JSON body
	var req struct {
		Schema      string `json:"schema"`
		StorageUnit string `json:"storageUnit"`
		Delimiter   string `json:"delimiter,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	schema := req.Schema
	storageUnit := req.StorageUnit
	delimiter := req.Delimiter
	
	// Default to comma if no delimiter specified
	if delimiter == "" {
		delimiter = ","
	}
	
	// Validate delimiter is a single character
	if len(delimiter) != 1 {
		http.Error(w, "Delimiter must be a single character", http.StatusBadRequest)
		return
	}
	
	// Convert delimiter to rune
	delimRune := rune(delimiter[0])

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

	// Stream CSV data directly to response
	// Note: ExportCSV expects a nil progress callback for now
	err := plugin.ExportCSV(pluginConfig, schema, storageUnit, writerFunc, nil)
	if err != nil {
		// If we haven't written anything yet, we can send an error
		if rowsWritten == 0 {
			http.Error(w, fmt.Sprintf("Export failed: %v", err), http.StatusInternalServerError)
			return
		}
		// Otherwise, we've already started streaming, so log the error
		// The partial data will still be sent
		fmt.Fprintf(w, "\n# ERROR: %v\n", err)
	}
}
