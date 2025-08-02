package graph

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"

	"github.com/clidey/whodb/core/src"
	"github.com/clidey/whodb/core/src/auth"
	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
)

// HandleCSVExport handles HTTP requests for CSV export
func HandleCSVExport(w http.ResponseWriter, r *http.Request) {
	// Extract parameters from URL
	schema := r.URL.Query().Get("schema")
	storageUnit := r.URL.Query().Get("storageUnit")
	
	if schema == "" || storageUnit == "" {
		http.Error(w, "Missing schema or storageUnit parameter", http.StatusBadRequest)
		return
	}

	// Get credentials from context
	credentials, err := auth.GetCredentialsFromContext(r.Context())
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	pluginConfig := engine.NewPluginConfig(credentials)
	plugin := src.MainEngine.Choose(credentials.Type)

	// Set CSV headers
	filename := fmt.Sprintf("%s_%s.csv", schema, storageUnit)
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

	// Create CSV writer that writes directly to response
	writer := common.CreateCSVWriter(w)

	// Track if first write has happened
	firstWrite := true
	rowCount := 0

	// Create writer function
	writerFunc := func(row []string) error {
		if firstWrite {
			firstWrite = false
			// Write byte order mark for Excel compatibility
			w.Write([]byte{0xEF, 0xBB, 0xBF})
		}
		
		err := writer.Write(row)
		if err == nil {
			rowCount++
			// Flush every 100 rows for streaming
			if rowCount%100 == 0 {
				writer.Flush()
				if f, ok := w.(http.Flusher); ok {
					f.Flush()
				}
			}
		}
		return err
	}

	// Export CSV data
	err = plugin.ExportCSV(pluginConfig, schema, storageUnit, writerFunc, nil)
	if err != nil {
		// If we haven't written anything yet, we can return an error
		if firstWrite {
			http.Error(w, fmt.Sprintf("Failed to export CSV: %v", err), http.StatusInternalServerError)
			return
		}
		// Otherwise, we've already started writing, so just log the error
		fmt.Fprintf(w, "\n# Error: %v", err)
	}

	// Final flush
	writer.Flush()
}