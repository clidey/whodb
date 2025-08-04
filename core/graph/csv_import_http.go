package graph

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/clidey/whodb/core/src"
	"github.com/clidey/whodb/core/src/auth"
	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
)

const (
	// Maximum file size: 50MB
	MaxFileSize = 50 * 1024 * 1024
	// Maximum memory for multipart parsing: 10MB
	MaxMemory = 10 * 1024 * 1024
)

// HandleCSVImport handles HTTP requests for CSV import with security limits
func HandleCSVImport(w http.ResponseWriter, r *http.Request) {
	// Check request size immediately
	if r.ContentLength > MaxFileSize {
		http.Error(w, fmt.Sprintf("File too large. Maximum size is %d MB", MaxFileSize/(1024*1024)), http.StatusRequestEntityTooLarge)
		return
	}

	// Parse multipart form with memory limit
	if err := r.ParseMultipartForm(MaxMemory); err != nil {
		http.Error(w, "Failed to parse upload", http.StatusBadRequest)
		return
	}
	defer r.MultipartForm.RemoveAll()

	// Get parameters
	schema := r.FormValue("schema")
	storageUnit := r.FormValue("storageUnit")
	mode := r.FormValue("mode")

	if schema == "" || storageUnit == "" {
		http.Error(w, "Missing required parameters", http.StatusBadRequest)
		return
	}

	// Get file
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Failed to get file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Double-check file size
	if header.Size > MaxFileSize {
		http.Error(w, fmt.Sprintf("File too large. Maximum size is %d MB", MaxFileSize/(1024*1024)), http.StatusRequestEntityTooLarge)
		return
	}

	// Get credentials
	credentials := auth.GetCredentials(r.Context())
	if credentials == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Process import with streaming
	pluginConfig := engine.NewPluginConfig(credentials)
	plugin := src.MainEngine.Choose(engine.DatabaseType(credentials.Type))

	// Create a limited reader to prevent DOS attacks
	limitedReader := io.LimitReader(file, MaxFileSize)

	// Create CSV reader
	csvReader := common.CreateCSVReader(limitedReader)

	// Track rows processed
	rowCount := 0
	const MaxRows = 1000000 // 1 million row limit

	// Create reader function with row limit
	readerFunc := func() ([]string, error) {
		if rowCount >= MaxRows {
			return nil, fmt.Errorf("row limit exceeded: maximum %d rows allowed", MaxRows)
		}

		record, err := csvReader.Read()
		if err == io.EOF {
			return nil, fmt.Errorf("EOF")
		}
		if err != nil {
			return nil, err
		}

		rowCount++
		return record, nil
	}

	// Convert mode
	var importMode engine.ImportMode
	switch mode {
	case "append", "Append":
		importMode = engine.ImportModeAppend
	case "override", "Override":
		importMode = engine.ImportModeOverride
	default:
		http.Error(w, "Invalid import mode", http.StatusBadRequest)
		return
	}

	// Progress tracking
	var lastProgress engine.ImportProgress
	progressCallback := func(progress engine.ImportProgress) {
		lastProgress = progress
		// Could send SSE or websocket updates here
	}

	// Set response headers for JSON
	w.Header().Set("Content-Type", "application/json")

	// Perform import
	err = plugin.ImportCSV(pluginConfig, schema, storageUnit, readerFunc, importMode, progressCallback)

	response := map[string]interface{}{
		"totalRows":     lastProgress.ProcessedRows,
		"processedRows": lastProgress.ProcessedRows,
	}

	if err != nil {
		response["status"] = "failed"
		response["error"] = err.Error()
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		response["status"] = "completed"
	}

	json.NewEncoder(w).Encode(response)
}
