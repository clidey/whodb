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

package database

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/xuri/excelize/v2"
)

// ImportOptions configures how data is imported.
type ImportOptions struct {
	HasHeader   bool // first row is column headers
	Delimiter   rune // CSV delimiter (default: auto-detect)
	CreateTable bool // create table if it doesn't exist
	BatchSize   int  // rows per batch insert (default: 500)
}

// ImportResult reports the outcome of an import operation.
type ImportResult struct {
	RowsImported int
	TableCreated bool
}

// ReadCSV reads a CSV file and returns headers and rows.
// If hasHeader is true, the first row is used as column names.
func ReadCSV(filePath string, delimiter rune, hasHeader bool) ([]string, [][]string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, nil, fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	if delimiter != 0 {
		reader.Comma = delimiter
	}
	reader.LazyQuotes = true
	reader.FieldsPerRecord = -1 // allow variable field counts

	records, err := reader.ReadAll()
	if err != nil {
		return nil, nil, fmt.Errorf("read CSV: %w", err)
	}

	if len(records) == 0 {
		return nil, nil, fmt.Errorf("CSV file is empty")
	}

	if hasHeader {
		return records[0], records[1:], nil
	}

	// Generate column names: col1, col2, ...
	headers := make([]string, len(records[0]))
	for i := range headers {
		headers[i] = fmt.Sprintf("col%d", i+1)
	}
	return headers, records, nil
}

// ReadExcel reads the first sheet of an Excel file and returns headers and rows.
func ReadExcel(filePath string, hasHeader bool) ([]string, [][]string, error) {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, nil, fmt.Errorf("open Excel file: %w", err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, nil, fmt.Errorf("Excel file has no sheets")
	}

	rows, err := f.GetRows(sheets[0])
	if err != nil {
		return nil, nil, fmt.Errorf("read Excel rows: %w", err)
	}

	if len(rows) == 0 {
		return nil, nil, fmt.Errorf("Excel sheet is empty")
	}

	if hasHeader {
		return rows[0], rows[1:], nil
	}

	headers := make([]string, len(rows[0]))
	for i := range headers {
		headers[i] = fmt.Sprintf("col%d", i+1)
	}
	return headers, rows, nil
}

// DetectDelimiter guesses the CSV delimiter by examining the first few lines.
func DetectDelimiter(filePath string) rune {
	f, err := os.Open(filePath)
	if err != nil {
		return ','
	}
	defer f.Close()

	buf := make([]byte, 4096)
	n, _ := f.Read(buf)
	sample := string(buf[:n])

	// Count occurrences of common delimiters in the first line
	firstLine := sample
	if idx := strings.Index(sample, "\n"); idx > 0 {
		firstLine = sample[:idx]
	}

	delimiters := []rune{',', '\t', ';', '|'}
	maxCount := 0
	best := ','

	for _, d := range delimiters {
		count := strings.Count(firstLine, string(d))
		if count > maxCount {
			maxCount = count
			best = d
		}
	}

	return best
}

// DetectFormat returns "csv" or "excel" based on file extension.
func DetectFormat(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".xlsx", ".xls":
		return "excel"
	default:
		return "csv"
	}
}

// InferColumnTypes examines sample data rows and returns SQL type names.
func InferColumnTypes(headers []string, rows [][]string) []string {
	types := make([]string, len(headers))
	for i := range types {
		types[i] = inferColumnType(i, rows)
	}
	return types
}

func inferColumnType(colIdx int, rows [][]string) string {
	hasInt, hasFloat, hasBool := true, true, true
	nonEmpty := 0

	limit := len(rows)
	if limit > 100 {
		limit = 100 // sample first 100 rows
	}

	for _, row := range rows[:limit] {
		if colIdx >= len(row) || row[colIdx] == "" {
			continue
		}
		val := strings.TrimSpace(row[colIdx])
		nonEmpty++

		if _, err := strconv.ParseInt(val, 10, 64); err != nil {
			hasInt = false
		}
		if _, err := strconv.ParseFloat(val, 64); err != nil {
			hasFloat = false
		}
		lower := strings.ToLower(val)
		if lower != "true" && lower != "false" && lower != "0" && lower != "1" {
			hasBool = false
		}
	}

	if nonEmpty == 0 {
		return "TEXT"
	}
	if hasBool {
		return "BOOLEAN"
	}
	if hasInt {
		return "INTEGER"
	}
	if hasFloat {
		return "REAL"
	}
	return "TEXT"
}

// ImportData imports rows into the database using the manager's current connection.
func (m *Manager) ImportData(schema, tableName string, headers []string, rows [][]string, opts ImportOptions) (*ImportResult, error) {
	if m.currentConnection == nil {
		return nil, fmt.Errorf("not connected to any database")
	}

	dbType := engine.DatabaseType(m.currentConnection.Type)
	plugin := m.engine.Choose(dbType)
	if plugin == nil {
		return nil, fmt.Errorf("plugin not found")
	}

	credentials := m.buildCredentials(m.currentConnection)
	pluginConfig := engine.NewPluginConfig(credentials)

	result := &ImportResult{}

	// Create table if requested
	if opts.CreateTable {
		types := InferColumnTypes(headers, rows)
		fields := make([]engine.Record, len(headers))
		for i, h := range headers {
			fields[i] = engine.Record{
				Key:   h,
				Value: types[i],
			}
		}

		created, err := plugin.AddStorageUnit(pluginConfig, schema, tableName, fields)
		if err != nil {
			return nil, fmt.Errorf("create table: %w", err)
		}
		result.TableCreated = created
	}

	// Import in batches
	batchSize := opts.BatchSize
	if batchSize <= 0 {
		batchSize = 500
	}

	for i := 0; i < len(rows); i += batchSize {
		end := i + batchSize
		if end > len(rows) {
			end = len(rows)
		}

		batch := make([][]engine.Record, 0, end-i)
		for _, row := range rows[i:end] {
			records := make([]engine.Record, len(headers))
			for j, h := range headers {
				val := ""
				if j < len(row) {
					val = row[j]
				}
				records[j] = engine.Record{Key: h, Value: val}
			}
			batch = append(batch, records)
		}

		if _, err := plugin.BulkAddRows(pluginConfig, schema, tableName, batch); err != nil {
			return nil, fmt.Errorf("insert batch at row %d: %w", i, err)
		}

		result.RowsImported += len(batch)
	}

	// Invalidate cache since schema changed
	m.cache.Clear()

	return result, nil
}

// ReadFileForImport reads a CSV or Excel file and returns headers, rows, and detected format.
func ReadFileForImport(filePath string, opts ImportOptions) ([]string, [][]string, error) {
	format := DetectFormat(filePath)

	switch format {
	case "excel":
		return ReadExcel(filePath, opts.HasHeader)
	default:
		delimiter := opts.Delimiter
		if delimiter == 0 {
			delimiter = DetectDelimiter(filePath)
		}
		return ReadCSV(filePath, delimiter, opts.HasHeader)
	}
}

// PreviewImport reads a file and returns the first N rows for preview.
func PreviewImport(filePath string, opts ImportOptions, maxRows int) ([]string, [][]string, error) {
	format := DetectFormat(filePath)

	if format == "excel" {
		headers, rows, err := ReadExcel(filePath, opts.HasHeader)
		if err != nil {
			return nil, nil, err
		}
		if len(rows) > maxRows {
			rows = rows[:maxRows]
		}
		return headers, rows, nil
	}

	// Stream CSV to avoid loading entire file
	f, err := os.Open(filePath)
	if err != nil {
		return nil, nil, fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	delimiter := opts.Delimiter
	if delimiter == 0 {
		delimiter = DetectDelimiter(filePath)
	}

	reader := csv.NewReader(f)
	reader.Comma = delimiter
	reader.LazyQuotes = true
	reader.FieldsPerRecord = -1

	var headers []string
	var rows [][]string

	for i := 0; ; i++ {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, fmt.Errorf("read CSV: %w", err)
		}

		if i == 0 && opts.HasHeader {
			headers = record
			continue
		}

		rows = append(rows, record)
		if len(rows) >= maxRows {
			break
		}
	}

	if headers == nil && len(rows) > 0 {
		headers = make([]string, len(rows[0]))
		for i := range headers {
			headers[i] = fmt.Sprintf("col%d", i+1)
		}
	}

	return headers, rows, nil
}
