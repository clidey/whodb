package graph

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/csv"
	"fmt"
	"io"
	"strings"

	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/src"
	"github.com/clidey/whodb/core/src/auth"
	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
)

// ExportCSV is the resolver for the ExportCSV field.
func (r *queryResolver) ExportCSV(ctx context.Context, schema string, storageUnit string) (string, error) {
	credentials, err := auth.GetCredentialsFromContext(ctx)
	if err != nil {
		return "", err
	}

	pluginConfig := engine.NewPluginConfig(credentials)
	plugin := src.MainEngine.Choose(credentials.Type)

	// Create a buffer to write CSV data
	var buf bytes.Buffer
	writer := common.CreateCSVWriter(&buf)

	// Create writer function that writes to our CSV writer
	writerFunc := func(row []string) error {
		return writer.Write(row)
	}

	// Export CSV data
	err = plugin.ExportCSV(pluginConfig, schema, storageUnit, writerFunc, nil)
	if err != nil {
		return "", fmt.Errorf("failed to export CSV: %v", err)
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return "", fmt.Errorf("CSV writer error: %v", err)
	}

	// Encode as base64 for safe transport
	encoded := base64.StdEncoding.EncodeToString(buf.Bytes())
	return encoded, nil
}

// ImportCSV is the resolver for the ImportCSV field.
func (r *mutationResolver) ImportCSV(ctx context.Context, schema string, storageUnit string, csvData string, mode model.ImportMode) (*model.ImportProgress, error) {
	credentials, err := auth.GetCredentialsFromContext(ctx)
	if err != nil {
		return nil, err
	}

	pluginConfig := engine.NewPluginConfig(credentials)
	plugin := src.MainEngine.Choose(credentials.Type)

	// Decode base64 CSV data
	decoded, err := base64.StdEncoding.DecodeString(csvData)
	if err != nil {
		return nil, fmt.Errorf("failed to decode CSV data: %v", err)
	}

	// Create CSV reader
	reader := common.CreateCSVReader(bytes.NewReader(decoded))
	
	// Track if we've read headers
	headersRead := false
	
	// Create reader function
	readerFunc := func() ([]string, error) {
		record, err := reader.Read()
		if err == io.EOF {
			return nil, fmt.Errorf("EOF")
		}
		if err != nil {
			return nil, err
		}
		
		// Skip empty rows
		if len(record) == 0 {
			return readerFunc()
		}
		
		// First row should be headers
		if !headersRead {
			headersRead = true
		}
		
		return record, nil
	}

	// Convert mode
	var importMode engine.ImportMode
	switch mode {
	case model.ImportModeAppend:
		importMode = engine.ImportModeAppend
	case model.ImportModeOverride:
		importMode = engine.ImportModeOverride
	default:
		return nil, fmt.Errorf("invalid import mode: %v", mode)
	}

	// Track progress
	var lastProgress engine.ImportProgress
	progressCallback := func(progress engine.ImportProgress) {
		lastProgress = progress
	}

	// Import CSV data
	err = plugin.ImportCSV(pluginConfig, schema, storageUnit, readerFunc, importMode, progressCallback)
	if err != nil {
		return &model.ImportProgress{
			TotalRows:     0,
			ProcessedRows: lastProgress.ProcessedRows,
			Status:        "failed",
			Error:         strPtr(err.Error()),
		}, nil
	}

	// Return final progress
	return &model.ImportProgress{
		TotalRows:     lastProgress.ProcessedRows,
		ProcessedRows: lastProgress.ProcessedRows,
		Status:        "completed",
		Error:         nil,
	}, nil
}

// strPtr returns a pointer to a string
func strPtr(s string) *string {
	return &s
}