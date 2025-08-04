package graph

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/clidey/whodb/core/src"
	"github.com/clidey/whodb/core/src/auth"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/xuri/excelize/v2"
)

const (
	// Maximum file size: 50MB
	MaxFileSize = 50 * 1024 * 1024
	// Maximum memory for multipart parsing: 10MB
	MaxMemory = 10 * 1024 * 1024
)

// HandleImport handles HTTP requests for CSV/Excel import with security limits
func HandleImport(w http.ResponseWriter, r *http.Request) {
	// Import functionality is temporarily disabled
	http.Error(w, "Import functionality is temporarily disabled", http.StatusServiceUnavailable)
	return

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
	delimiter := r.FormValue("delimiter")

	if schema == "" || storageUnit == "" {
		http.Error(w, "Missing required parameters", http.StatusBadRequest)
		return
	}

	// Default delimiter to comma if not specified
	if delimiter == "" {
		delimiter = ","
	}

	// Validate delimiter is a single character
	if len(delimiter) != 1 {
		http.Error(w, "Delimiter must be a single character", http.StatusBadRequest)
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

	// Detect file format from extension
	filename := header.Filename
	ext := strings.ToLower(filepath.Ext(filename))
	isExcel := ext == ".xlsx" || ext == ".xls"
	isCSV := ext == ".csv" || header.Header.Get("Content-Type") == "text/csv"

	if !isExcel && !isCSV {
		http.Error(w, "Invalid file format. Only CSV and Excel files are allowed", http.StatusBadRequest)
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

	// Track rows processed
	rowCount := 0
	const MaxRows = 1000000 // 1 million row limit

	var readerFunc func() ([]string, error)

	if isExcel {
		// Handle Excel import
		readerFunc, err = createExcelReader(limitedReader, &rowCount, MaxRows)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to read Excel file: %v", err), http.StatusBadRequest)
			return
		}
	} else {
		// Handle CSV import with custom delimiter
		csvReader := csv.NewReader(limitedReader)
		csvReader.Comma = rune(delimiter[0])
		csvReader.LazyQuotes = true
		csvReader.TrimLeadingSpace = true

		readerFunc = func() ([]string, error) {
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

// createExcelReader creates a reader function for Excel files
func createExcelReader(reader io.Reader, rowCount *int, maxRows int) (func() ([]string, error), error) {
	// Read entire Excel file into memory (Excel format requires random access)
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read Excel file: %v", err)
	}

	// Open Excel file from bytes
	f, err := excelize.OpenReader(strings.NewReader(string(data)))
	if err != nil {
		return nil, fmt.Errorf("failed to open Excel file: %v", err)
	}

	// Get the first sheet
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("no sheets found in Excel file")
	}
	sheetName := sheets[0]

	// Get all rows from the sheet
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("failed to read rows from sheet %s: %v", sheetName, err)
	}

	// Create iterator
	currentRow := 0
	totalRows := len(rows)

	return func() ([]string, error) {
		if *rowCount >= maxRows {
			return nil, fmt.Errorf("row limit exceeded: maximum %d rows allowed", maxRows)
		}

		if currentRow >= totalRows {
			return nil, fmt.Errorf("EOF")
		}

		row := rows[currentRow]
		currentRow++
		(*rowCount)++

		// Ensure all cells are string values
		stringRow := make([]string, len(row))
		for i, cell := range row {
			stringRow[i] = cell
		}

		return stringRow, nil
	}, nil
}
