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

package gorm_plugin

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
	"gorm.io/gorm"
)

// ExportCSV exports data to CSV format. If selectedRows is nil/empty, exports all rows from the table.
func (p *GormPlugin) ExportCSV(config *engine.PluginConfig, schema string, storageUnit string, writer func([]string) error, selectedRows []map[string]interface{}) error {
	// If selected rows are provided, export only those
	if len(selectedRows) > 0 {
		// Extract column names from the first row
		columns := make([]string, 0, len(selectedRows[0]))
		for col := range selectedRows[0] {
			columns = append(columns, col)
		}

		// Write header row
		if err := writer(columns); err != nil {
			return fmt.Errorf("failed to write headers: %v", err)
		}

		// Write selected rows
		for i, row := range selectedRows {
			rowData := make([]string, len(columns))
			for j, col := range columns {
				if val, ok := row[col]; ok {
					rowData[j] = p.FormatValue(val)
				} else {
					rowData[j] = ""
				}
			}
			if err := writer(rowData); err != nil {
				return fmt.Errorf("failed to write row %d: %v", i+1, err)
			}
		}

		return nil
	}

	// Export all rows from the database
	db, err := p.DB(config)
	if err != nil {
		return err
	}

	// Get column information
	tableName := p.FormTableName(schema, storageUnit)
	columns, columnTypes, err := p.getTableColumns(db, schema, storageUnit)
	if err != nil {
		return fmt.Errorf("failed to get table columns: %v", err)
	}

	// Write headers with type information
	headers := make([]string, len(columns))
	for i, col := range columns {
		headers[i] = common.FormatCSVHeader(col, columnTypes[i])
	}
	if err := writer(headers); err != nil {
		return fmt.Errorf("failed to write headers: %v", err)
	}

	// Build query
	columnList := make([]string, len(columns))
	for i, col := range columns {
		columnList[i] = p.EscapeIdentifier(col)
	}
	query := fmt.Sprintf("SELECT %s FROM %s", strings.Join(columnList, ", "), tableName)

	// Execute query
	rows, err := db.Raw(query).Rows()
	if err != nil {
		return fmt.Errorf("failed to query data: %v", err)
	}
	defer rows.Close()

	// Stream results
	rowCount := 0
	values := make([]any, len(columns))
	valuePtrs := make([]any, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	for rows.Next() {
		if err := rows.Scan(valuePtrs...); err != nil {
			return fmt.Errorf("failed to scan row: %v", err)
		}

		row := make([]string, len(columns))
		for i, val := range values {
			row[i] = p.FormatValue(val)
		}

		if err := writer(row); err != nil {
			return fmt.Errorf("failed to write row: %v", err)
		}

		rowCount++
	}

	return rows.Err()
}

// ImportCSV imports CSV data into the table
func (p *GormPlugin) ImportCSV(config *engine.PluginConfig, schema string, storageUnit string, reader func() ([]string, error), mode engine.ImportMode, progressCallback func(engine.ImportProgress)) error {
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
	tableName := p.FormTableName(schema, storageUnit)
	existingColumns, existingTypes, err := p.getTableColumns(db, schema, storageUnit)
	if err != nil {
		return fmt.Errorf("failed to get table columns: %v", err)
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

	// Start transaction
	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Handle override mode
	if mode == engine.ImportModeOverride {
		if err := tx.Exec(fmt.Sprintf("DELETE FROM %s", tableName)).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to clear table: %v", err)
		}
	}

	// Prepare insert statement
	placeholders := make([]string, len(existingColumns))
	for i := range placeholders {
		placeholders[i] = p.GetPlaceholder(i + 1)
	}

	insertSQL := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		tableName,
		strings.Join(p.escapeIdentifiers(existingColumns), ", "),
		strings.Join(placeholders, ", "))

	// Process rows
	rowCount := 0
	for {
		row, err := reader()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			tx.Rollback()
			return fmt.Errorf("failed to read row %d: %v", rowCount+1, err)
		}

		// Map CSV values to table columns
		values := make([]any, len(existingColumns))
		for i, col := range existingColumns {
			csvIndex, exists := columnMap[col]
			if !exists {
				values[i] = nil // Column missing in CSV, use NULL
			} else if csvIndex >= len(row) {
				values[i] = nil // Row too short, use NULL
			} else {
				// Convert value based on column type
				val, err := p.ConvertStringValue(row[csvIndex], existingTypes[i])
				if err != nil {
					tx.Rollback()
					return fmt.Errorf("failed to convert value for column %s at row %d: %v", col, rowCount+1, err)
				}
				values[i] = val
			}
		}

		// Execute insert
		if err := tx.Exec(insertSQL, values...).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to insert row %d: %v", rowCount+1, err)
		}

		rowCount++
		if progressCallback != nil && rowCount%100 == 0 {
			progressCallback(engine.ImportProgress{
				ProcessedRows: rowCount,
				Status:        "importing",
			})
		}
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
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

func (p *GormPlugin) getTableColumns(db *gorm.DB, schema string, storageUnit string) ([]string, []string, error) {
	// Use information schema to get column info
	query := `
		SELECT column_name, data_type 
		FROM information_schema.columns 
		WHERE table_schema = ? AND table_name = ? 
		ORDER BY ordinal_position`

	rows, err := db.Raw(query, schema, storageUnit).Rows()
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var columns []string
	var types []string
	for rows.Next() {
		var col, typ string
		if err := rows.Scan(&col, &typ); err != nil {
			return nil, nil, err
		}
		columns = append(columns, col)
		types = append(types, typ)
	}

	return columns, types, rows.Err()
}

// FormatValue converts interface{} values to strings appropriately for CSV
func (p *GormPlugin) FormatValue(val any) string {
	if val == nil {
		return ""
	}

	switch v := val.(type) {
	case []byte:
		return string(v)
	case string:
		return v
	case time.Time:
		// Format time in ISO 8601 format that can be parsed back
		if v.Hour() == 0 && v.Minute() == 0 && v.Second() == 0 && v.Nanosecond() == 0 {
			// Date only
			return v.Format("2006-01-02")
		}
		// Full datetime
		return v.Format("2006-01-02T15:04:05")
	case *time.Time:
		if v == nil {
			return ""
		}
		// Format time in ISO 8601 format that can be parsed back
		if v.Hour() == 0 && v.Minute() == 0 && v.Second() == 0 && v.Nanosecond() == 0 {
			// Date only
			return v.Format("2006-01-02")
		}
		// Full datetime
		return v.Format("2006-01-02T15:04:05")
	case sql.NullTime:
		if !v.Valid {
			return ""
		}
		// Format time in ISO 8601 format that can be parsed back
		if v.Time.Hour() == 0 && v.Time.Minute() == 0 && v.Time.Second() == 0 && v.Time.Nanosecond() == 0 {
			// Date only
			return v.Time.Format("2006-01-02")
		}
		// Full datetime
		return v.Time.Format("2006-01-02T15:04:05")
	default:
		return fmt.Sprintf("%v", v)
	}
}

func (p *GormPlugin) escapeIdentifiers(identifiers []string) []string {
	escaped := make([]string, len(identifiers))
	for i, id := range identifiers {
		escaped[i] = p.EscapeIdentifier(id)
	}
	return escaped
}

// GetPlaceholder returns the placeholder for prepared statements
// Override this in database-specific implementations
func (p *GormPlugin) GetPlaceholder(index int) string {
	return "?"
}
