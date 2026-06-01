/*
 * Copyright 2026 Clidey, Inc.
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
	"time"

	"gorm.io/gorm"

	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
	"github.com/clidey/whodb/core/src/plugins"
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
			log.WithError(err).Error("Failed to write CSV headers for selected rows export")
			return fmt.Errorf("failed to write headers: %w", err)
		}

		// Write selected rows
		for i, row := range selectedRows {
			rowData := make([]string, len(columns))
			for j, col := range columns {
				if val, ok := row[col]; ok {
					rowData[j] = p.GormPluginFunctions.FormatValue(val)
				} else {
					rowData[j] = ""
				}
			}
			if err := writer(rowData); err != nil {
				log.WithError(err).Error(fmt.Sprintf("Failed to write selected row %d during export", i+1))
				return fmt.Errorf("failed to write row %d: %w", i+1, err)
			}
		}

		return nil
	}

	// Export all rows from the database
	_, err := plugins.WithConnection(config, p.DB, func(db *gorm.DB) (struct{}, error) {
		columnConfig := config
		if config != nil {
			clonedConfig := *config
			clonedConfig.Transaction = db
			columnConfig = &clonedConfig
		} else {
			columnConfig = &engine.PluginConfig{Transaction: db}
		}

		// Resolve columns through the plugin API so connector-specific overrides
		// (QuestDB, ClickHouse, SQLite, etc.) are reused for exports too.
		orderedColumns, err := p.PluginFunctions.GetColumnsForTable(columnConfig, schema, storageUnit)
		if err != nil {
			log.WithError(err).Error(fmt.Sprintf("Failed to get columns for export of table %s.%s", schema, storageUnit))
			return struct{}{}, fmt.Errorf("failed to get columns: %w", err)
		}

		if len(orderedColumns) == 0 {
			return struct{}{}, fmt.Errorf("no columns found for table %s.%s", schema, storageUnit)
		}

		// Convert to separate arrays for columns and types
		columns := make([]string, len(orderedColumns))
		columnTypes := make([]string, len(orderedColumns))
		for i, col := range orderedColumns {
			columns[i] = col.Name
			columnTypes[i] = col.Type
		}

		// Write headers with type information
		headers := make([]string, len(columns))
		for i, col := range columns {
			headers[i] = common.FormatCSVHeader(col, columnTypes[i])
		}
		if err := writer(headers); err != nil {
			log.WithError(err).Error(fmt.Sprintf("Failed to write CSV headers for table %s.%s export", schema, storageUnit))
			return struct{}{}, fmt.Errorf("failed to write headers: %w", err)
		}

		// Use batch processor for efficient export
		processor := NewBatchProcessor(p, p.Type, &BatchConfig{
			BatchSize:   10000, // Larger batch size for exports
			LogProgress: true,  // Log export progress
		})

		// Export data in batches to avoid memory issues
		totalRows := 0
		err = processor.ExportInBatches(db, schema, storageUnit, columns, func(batch []map[string]any) error {
			// Process each batch
			for _, record := range batch {
				row := make([]string, len(columns))
				for i, col := range columns {
					if val, exists := record[col]; exists {
						row[i] = p.GormPluginFunctions.FormatValue(val)
					} else {
						row[i] = ""
					}
				}

				if err := writer(row); err != nil {
					log.WithError(err).Error(fmt.Sprintf("Failed to write row %d during export of table %s.%s", totalRows+1, schema, storageUnit))
					return fmt.Errorf("failed to write row %d: %w", totalRows+1, err)
				}
				totalRows++
			}
			return nil
		})
		if err != nil {
			return struct{}{}, fmt.Errorf("export failed after %d rows: %w", totalRows, err)
		}

		log.WithField("totalRows", totalRows).
			WithField("table", fmt.Sprintf("%s.%s", schema, storageUnit)).
			Info("Export completed successfully")

		return struct{}{}, nil
	})
	return err
}

func formatExportValue(plugin GormPluginFunctions, val any) string {
	if val == nil {
		return ""
	}

	switch v := val.(type) {
	case []byte:
		return common.EscapeFormula(string(v))
	case string:
		return common.EscapeFormula(v)
	case time.Time:
		return common.EscapeFormula(plugin.FormatTimeForExport(v))
	case *time.Time:
		if v == nil {
			return ""
		}
		return common.EscapeFormula(plugin.FormatTimeForExport(*v))
	case sql.NullTime:
		if !v.Valid {
			return ""
		}
		return common.EscapeFormula(plugin.FormatTimeForExport(v.Time))
	default:
		return common.EscapeFormula(fmt.Sprintf("%v", v))
	}
}

// FormatValue converts any values to strings appropriately for CSV.
func (p *GormPlugin) FormatValue(val any) string {
	return formatExportValue(p.GormPluginFunctions, val)
}

// FormatTimeForExport returns the default string representation for exported timestamps.
func (p *GormPlugin) FormatTimeForExport(value time.Time) string {
	if value.Hour() == 0 && value.Minute() == 0 && value.Second() == 0 && value.Nanosecond() == 0 {
		return value.Format("2006-01-02")
	}
	return value.Format("2006-01-02T15:04:05")
}

// GetPlaceholder returns the placeholder for prepared statements
// Override this in database-specific implementations
func (p *GormPlugin) GetPlaceholder(index int) string {
	return "?"
}
