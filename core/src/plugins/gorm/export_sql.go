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
	"io"

	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/plugins"
	"github.com/clidey/whodb/core/src/sqlexport"
	"gorm.io/gorm"
)

// ExportSQLData streams SQL Data Export statements for a SQL Table.
func (p *GormPlugin) ExportSQLData(config *engine.PluginConfig, req *engine.SQLDataExportRequest, writer io.Writer) error {
	if req == nil {
		return fmt.Errorf("missing SQL data export request")
	}

	_, err := plugins.WithConnection(config, p.GormPluginFunctions.DB, func(db *gorm.DB) (bool, error) {
		columns, err := p.GormPluginFunctions.GetColumnsForTable(config, req.Schema, req.StorageUnit)
		if err != nil {
			return false, fmt.Errorf("failed to get columns for SQL data export: %w", err)
		}
		if err := p.GormPluginFunctions.MarkGeneratedColumns(config, req.Schema, req.StorageUnit, columns); err != nil {
			return false, fmt.Errorf("failed to inspect generated columns for SQL data export: %w", err)
		}

		insertColumns, setColumns, primaryKeyColumns, rowColumns, err := selectSQLDataExportColumns(req.Mode, columns)
		if err != nil {
			return false, err
		}

		query, err := p.buildSQLDataExportQuery(db, req, rowColumns)
		if err != nil {
			return false, err
		}

		rows, err := query.Rows()
		if err != nil {
			return false, fmt.Errorf("failed to read rows for SQL data export: %w", err)
		}
		defer rows.Close()

		dialect := sqlexport.GenericDialect{
			QuoteIdentifierFunc: p.GormPluginFunctions.CreateSQLBuilder(db).QuoteIdentifier,
		}
		table := sqlexport.Table{Schema: req.Schema, Name: req.StorageUnit}

		switch req.Mode {
		case engine.SQLDataExportModeInsert:
			err = streamInsertSQLDataExport(writer, dialect, table, insertColumns, rowColumns, rows)
		case engine.SQLDataExportModeUpdate:
			err = streamUpdateSQLDataExport(writer, dialect, table, setColumns, primaryKeyColumns, rowColumns, rows)
		default:
			err = fmt.Errorf("unsupported SQL data export mode: %s", req.Mode)
		}
		if err != nil {
			return false, err
		}
		if err := rows.Err(); err != nil {
			return false, fmt.Errorf("failed to scan rows for SQL data export: %w", err)
		}
		return true, nil
	})
	return err
}

func (p *GormPlugin) buildSQLDataExportQuery(db *gorm.DB, req *engine.SQLDataExportRequest, columns []engine.Column) (*gorm.DB, error) {
	if req.Limit != nil && *req.Limit < 0 {
		return nil, fmt.Errorf("row limit must not be negative")
	}

	var columnTypes map[string]ColumnTypeInfo
	if req.Where != nil {
		var err error
		columnTypes, err = p.GormPluginFunctions.GetColumnTypes(db, req.Schema, req.StorageUnit)
		if err != nil {
			return nil, fmt.Errorf("failed to get column types for SQL data export filters: %w", err)
		}
	}

	builder := p.GormPluginFunctions.CreateSQLBuilder(db)
	fullTable := builder.BuildFullTableName(req.Schema, req.StorageUnit)
	query := db.Table(fullTable).Select(columnNames(columns))

	var err error
	query, err = p.ApplyWhereConditions(query, req.Where, columnTypes)
	if err != nil {
		return nil, err
	}

	if len(req.Sort) > 0 {
		query = builder.BuildOrderBy(query, toPluginSort(req.Sort))
	}

	if req.Limit != nil {
		query = query.Limit(*req.Limit)
	}

	return query, nil
}

func selectSQLDataExportColumns(mode engine.SQLDataExportMode, columns []engine.Column) ([]engine.Column, []engine.Column, []engine.Column, []engine.Column, error) {
	insertColumns := make([]engine.Column, 0, len(columns))
	setColumns := make([]engine.Column, 0, len(columns))
	primaryKeyColumns := make([]engine.Column, 0)
	rowColumns := make([]engine.Column, 0, len(columns))

	for _, column := range columns {
		if column.IsComputed {
			continue
		}
		insertColumns = append(insertColumns, column)
		rowColumns = append(rowColumns, column)
		if column.IsPrimary {
			primaryKeyColumns = append(primaryKeyColumns, column)
			continue
		}
		setColumns = append(setColumns, column)
	}

	switch mode {
	case engine.SQLDataExportModeInsert:
		if len(insertColumns) == 0 {
			return nil, nil, nil, nil, fmt.Errorf("no writable columns found for SQL Data Export")
		}
		return insertColumns, setColumns, primaryKeyColumns, rowColumns, nil
	case engine.SQLDataExportModeUpdate:
		if len(primaryKeyColumns) == 0 {
			return nil, nil, nil, nil, fmt.Errorf("SQL Update Export requires a primary key")
		}
		if len(setColumns) == 0 {
			return nil, nil, nil, nil, fmt.Errorf("SQL Update Export requires at least one non-primary writable column")
		}
		return insertColumns, setColumns, primaryKeyColumns, rowColumns, nil
	default:
		return nil, nil, nil, nil, fmt.Errorf("unsupported SQL data export mode: %s", mode)
	}
}

func streamInsertSQLDataExport(w io.Writer, dialect sqlexport.Dialect, table sqlexport.Table, insertColumns []engine.Column, rowColumns []engine.Column, rows *sql.Rows) error {
	batch := make([]sqlexport.Row, 0, sqlexport.DefaultInsertBatchSize)
	for rows.Next() {
		row, err := scanSQLDataExportRow(rows, rowColumns)
		if err != nil {
			return err
		}
		batch = append(batch, row)
		if len(batch) == sqlexport.DefaultInsertBatchSize {
			if err := sqlexport.WriteInsert(w, dialect, table, insertColumns, batch); err != nil {
				return err
			}
			batch = batch[:0]
		}
	}
	if len(batch) > 0 {
		return sqlexport.WriteInsert(w, dialect, table, insertColumns, batch)
	}
	return nil
}

func streamUpdateSQLDataExport(w io.Writer, dialect sqlexport.Dialect, table sqlexport.Table, setColumns []engine.Column, primaryKeyColumns []engine.Column, rowColumns []engine.Column, rows *sql.Rows) error {
	for rows.Next() {
		row, err := scanSQLDataExportRow(rows, rowColumns)
		if err != nil {
			return err
		}
		if err := sqlexport.WriteUpdate(w, dialect, table, setColumns, primaryKeyColumns, row); err != nil {
			return err
		}
	}
	return nil
}

func scanSQLDataExportRow(rows *sql.Rows, columns []engine.Column) (sqlexport.Row, error) {
	values := make([]any, len(columns))
	destinations := make([]any, len(columns))
	for i := range values {
		destinations[i] = &values[i]
	}
	if err := rows.Scan(destinations...); err != nil {
		return nil, fmt.Errorf("failed to scan row for SQL data export: %w", err)
	}

	row := make(sqlexport.Row, len(columns))
	for i, column := range columns {
		if bytesValue, ok := values[i].([]byte); ok {
			row[column.Name] = append([]byte(nil), bytesValue...)
			continue
		}
		row[column.Name] = values[i]
	}
	return row, nil
}

func columnNames(columns []engine.Column) []string {
	names := make([]string, len(columns))
	for i, column := range columns {
		names[i] = column.Name
	}
	return names
}

func toPluginSort(sortConditions []*model.SortCondition) []plugins.Sort {
	sortList := make([]plugins.Sort, len(sortConditions))
	for i, sort := range sortConditions {
		sortList[i] = plugins.Sort{
			Column:    sort.Column,
			Direction: plugins.Down,
		}
		if sort.Direction == model.SortDirectionAsc {
			sortList[i].Direction = plugins.Up
		}
	}
	return sortList
}
