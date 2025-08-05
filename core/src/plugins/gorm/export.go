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
)

// ExportData exports data to tabular format (CSV/Excel). If selectedRows is nil/empty, exports all rows from the table.
func (p *GormPlugin) ExportData(config *engine.PluginConfig, schema string, storageUnit string, writer func([]string) error, selectedRows []map[string]any) error {
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

	// Get column information using existing GetTableSchema
	tableSchema, err := p.GetTableSchema(db, schema)
	if err != nil {
		return fmt.Errorf("failed to get table schema: %v", err)
	}

	// Extract columns for the specific table
	tableColumns, exists := tableSchema[storageUnit]
	if !exists || len(tableColumns) == 0 {
		return fmt.Errorf("no columns found for table %s.%s", schema, storageUnit)
	}

	// Convert to separate arrays for columns and types
	columns := make([]string, len(tableColumns))
	columnTypes := make([]string, len(tableColumns))
	for i, col := range tableColumns {
		columns[i] = col.Key       // Column name
		columnTypes[i] = col.Value // Data type
	}

	tableName := p.FormTableName(schema, storageUnit)

	// Write headers with type information
	headers := make([]string, len(columns))
	for i, col := range columns {
		headers[i] = common.FormatCSVHeader(col, columnTypes[i])
	}
	if err := writer(headers); err != nil {
		return fmt.Errorf("failed to write headers: %v", err)
	}

	// Build query with proper escaping
	columnList := make([]string, len(columns))
	for i, col := range columns {
		columnList[i] = p.EscapeIdentifier(col)
	}
	// Ensure table name is properly escaped
	escapedTableName := tableName
	if !strings.Contains(tableName, ".") {
		escapedTableName = p.EscapeIdentifier(tableName)
	} else {
		// Handle schema.table format
		parts := strings.Split(tableName, ".")
		if len(parts) == 2 {
			escapedTableName = p.EscapeIdentifier(parts[0]) + "." + p.EscapeIdentifier(parts[1])
		}
	}
	query := fmt.Sprintf("SELECT %s FROM %s", strings.Join(columnList, ", "), escapedTableName)

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

// FormatValue converts interface{} values to strings appropriately for CSV
func (p *GormPlugin) FormatValue(val any) string {
	if val == nil {
		return ""
	}

	switch v := val.(type) {
	case []byte:
		return common.EscapeFormula(string(v))
	case string:
		return common.EscapeFormula(v)
	case time.Time:
		// Format time in ISO 8601 format that can be parsed back
		if v.Hour() == 0 && v.Minute() == 0 && v.Second() == 0 && v.Nanosecond() == 0 {
			// Date only
			return common.EscapeFormula(v.Format("2006-01-02"))
		}
		// Full datetime
		return common.EscapeFormula(v.Format("2006-01-02T15:04:05"))
	case *time.Time:
		if v == nil {
			return ""
		}
		// Format time in ISO 8601 format that can be parsed back
		if v.Hour() == 0 && v.Minute() == 0 && v.Second() == 0 && v.Nanosecond() == 0 {
			// Date only
			return common.EscapeFormula(v.Format("2006-01-02"))
		}
		// Full datetime
		return common.EscapeFormula(v.Format("2006-01-02T15:04:05"))
	case sql.NullTime:
		if !v.Valid {
			return ""
		}
		// Format time in ISO 8601 format that can be parsed back
		if v.Time.Hour() == 0 && v.Time.Minute() == 0 && v.Time.Second() == 0 && v.Time.Nanosecond() == 0 {
			// Date only
			return common.EscapeFormula(v.Time.Format("2006-01-02"))
		}
		// Full datetime
		return common.EscapeFormula(v.Time.Format("2006-01-02T15:04:05"))
	default:
		return common.EscapeFormula(fmt.Sprintf("%v", v))
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
