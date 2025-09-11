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

	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/plugins/gorm"
)

// ExportData exports ClickHouse table data to tabular format
func (p *ClickHousePlugin) ExportData(config *engine.PluginConfig, schema string, storageUnit string, writer func([]string) error, selectedRows []map[string]any) error {
	// If selected rows are provided, delegate to parent GORM implementation
	if len(selectedRows) > 0 {
		return p.GormPlugin.ExportData(config, schema, storageUnit, writer, selectedRows)
	}
	db, err := p.DB(config)
	if err != nil {
		return err
	}

	// Get column information using GORM's query builder
	rows, err := db.Table("system.columns").
		Select("name, type").
		Where("database = ? AND table = ?", schema, storageUnit).
		Order("position").
		Rows()
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

	// Use GORM query builder for export
	builder := gorm_plugin.NewSQLBuilder(db, p)
	fullTable := builder.BuildFullTableName(schema, storageUnit)

	exportQuery := db.Table(fullTable)
	if len(columns) > 0 {
		exportQuery = exportQuery.Select(columns)
	}

	dataRows, err := exportQuery.Rows()
	if err != nil {
		return fmt.Errorf("failed to query data: %v", err)
	}
	defer dataRows.Close()

	rowCount := 0
	values := make([]any, len(columns))
	valuePtrs := make([]any, len(columns))
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
	}

	return dataRows.Err()
}

// Helper functions

func (p *ClickHousePlugin) formatValue(val any) string {
	if val == nil {
		return ""
	}

	var strVal string
	switch v := val.(type) {
	case []byte:
		strVal = string(v)
	case string:
		strVal = v
	default:
		strVal = fmt.Sprintf("%v", v)
	}

	// Apply formula injection protection
	return common.EscapeFormula(strVal)
}
