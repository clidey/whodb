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

package clickhouse

import (
	"fmt"
	"strings"

	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
)

// ExportCSV exports ClickHouse table data to CSV format
func (p *ClickHousePlugin) ExportCSV(config *engine.PluginConfig, schema string, storageUnit string, writer func([]string) error, progressCallback func(int)) error {
	db, err := p.DB(config)
	if err != nil {
		return err
	}

	// Get column information
	query := `
		SELECT name, type 
		FROM system.columns 
		WHERE database = ? AND table = ?
		ORDER BY position`

	rows, err := db.Raw(query, schema, storageUnit).Rows()
	if err != nil {
		return fmt.Errorf("failed to get columns: %v", err)
	}
	defer rows.Close()

	var columns []string
	var types []string
	for rows.Next() {
		var col, typ string
		if err := rows.Scan(&col, &typ); err != nil {
			return err
		}
		columns = append(columns, col)
		types = append(types, typ)
	}

	// Write headers
	headers := make([]string, len(columns))
	for i, col := range columns {
		headers[i] = common.FormatCSVHeader(col, types[i])
	}
	if err := writer(headers); err != nil {
		return fmt.Errorf("failed to write headers: %v", err)
	}

	// Export data
	selectQuery := fmt.Sprintf("SELECT %s FROM %s.%s",
		strings.Join(columns, ", "), schema, storageUnit)

	dataRows, err := db.Raw(selectQuery).Rows()
	if err != nil {
		return fmt.Errorf("failed to query data: %v", err)
	}
	defer dataRows.Close()

	rowCount := 0
	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	for dataRows.Next() {
		if err := dataRows.Scan(valuePtrs...); err != nil {
			return fmt.Errorf("failed to scan row: %v", err)
		}

		row := make([]string, len(columns))
		for i, val := range values {
			row[i] = p.formatValue(val)
		}

		if err := writer(row); err != nil {
			return fmt.Errorf("failed to write row: %v", err)
		}

		rowCount++
		if progressCallback != nil && rowCount%10000 == 0 {
			progressCallback(rowCount)
		}
	}

	if progressCallback != nil {
		progressCallback(rowCount)
	}

	return dataRows.Err()
}

// ImportCSV imports CSV data into ClickHouse table
func (p *ClickHousePlugin) ImportCSV(config *engine.PluginConfig, schema string, storageUnit string, reader func() ([]string, error), mode engine.ImportMode, progressCallback func(engine.ImportProgress)) error {
	db, err := p.DB(config)
	if err != nil {
		return err
	}

	// Read headers
	headers, err := reader()
	if err != nil {
		return fmt.Errorf("failed to read headers: %v", err)
	}

	// Parse column names and types from headers
	columnNames, columnTypes, err := common.ParseCSVHeaders(headers)
	if err != nil {
		return err
	}

	// Get existing table columns
	query := `
		SELECT name, type 
		FROM system.columns 
		WHERE database = ? AND table = ?
		ORDER BY position`

	rows, err := db.Raw(query, schema, storageUnit).Rows()
	if err != nil {
		return fmt.Errorf("failed to get columns: %v", err)
	}
	defer rows.Close()

	var existingColumns []string
	var existingTypes []string
	for rows.Next() {
		var col, typ string
		if err := rows.Scan(&col, &typ); err != nil {
			return err
		}
		existingColumns = append(existingColumns, col)
		existingTypes = append(existingTypes, typ)
	}

	// Validate columns match
	if err := common.ValidateCSVColumns(existingColumns, columnNames); err != nil {
		return err
	}

	// Create column mapping
	columnMap := make(map[string]int)
	typeMap := make(map[string]string)
	for i, col := range columnNames {
		columnMap[col] = i
		typeMap[col] = columnTypes[i]
	}

	// Handle override mode
	if mode == engine.ImportModeOverride {
		if err := db.Exec(fmt.Sprintf("TRUNCATE TABLE %s.%s", schema, storageUnit)).Error; err != nil {
			return fmt.Errorf("failed to clear table: %v", err)
		}
	}

	// Prepare batch insert
	batch := db.Begin()
	if batch.Error != nil {
		return fmt.Errorf("failed to begin transaction: %v", batch.Error)
	}

	// Prepare placeholders for ClickHouse
	placeholders := make([]string, len(existingColumns))
	for i := range placeholders {
		placeholders[i] = "?"
	}

	insertQuery := fmt.Sprintf("INSERT INTO %s.%s (%s) VALUES (%s)",
		schema, storageUnit,
		strings.Join(existingColumns, ", "),
		strings.Join(placeholders, ", "))

	// Process rows
	rowCount := 0
	batchSize := 10000
	currentBatch := 0

	for {
		row, err := reader()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			batch.Rollback()
			return fmt.Errorf("failed to read row %d: %v", rowCount+1, err)
		}

		// Map CSV values to table columns
		values := make([]interface{}, len(existingColumns))
		for i, col := range existingColumns {
			csvIndex, exists := columnMap[col]
			if !exists {
				values[i] = nil
			} else if csvIndex >= len(row) {
				values[i] = nil
			} else {
				values[i] = p.parseClickHouseValue(row[csvIndex], existingTypes[i])
			}
		}

		if err := batch.Exec(insertQuery, values...).Error; err != nil {
			batch.Rollback()
			return fmt.Errorf("failed to insert row %d: %v", rowCount+1, err)
		}

		rowCount++
		currentBatch++

		// Commit batch
		if currentBatch >= batchSize {
			if err := batch.Commit().Error; err != nil {
				return fmt.Errorf("failed to commit batch: %v", err)
			}

			batch = db.Begin()
			if batch.Error != nil {
				return fmt.Errorf("failed to begin new transaction: %v", batch.Error)
			}

			currentBatch = 0
		}

		if progressCallback != nil && rowCount%1000 == 0 {
			progressCallback(engine.ImportProgress{
				ProcessedRows: rowCount,
				Status:        "importing",
			})
		}
	}

	// Commit final batch
	if err := batch.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit final batch: %v", err)
	}

	if progressCallback != nil {
		progressCallback(engine.ImportProgress{
			ProcessedRows: rowCount,
			Status:        "completed",
		})
	}

	return nil
}

// Helper functions

func (p *ClickHousePlugin) formatValue(val interface{}) string {
	if val == nil {
		return ""
	}

	switch v := val.(type) {
	case []byte:
		return string(v)
	case string:
		return v
	default:
		return fmt.Sprintf("%v", v)
	}
}

func (p *ClickHousePlugin) parseClickHouseValue(val string, dataType string) interface{} {
	if val == "" {
		return nil
	}

	// ClickHouse will handle type conversion
	return val
}
