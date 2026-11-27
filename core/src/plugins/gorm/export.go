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
	"fmt"
	"math"
	"slices"
	"strconv"

	"gorm.io/gorm"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/plugins"
	"github.com/clidey/whodb/core/src/source"
	"github.com/clidey/whodb/core/src/sourcecatalog"
)

// ExportData exports a table, or the selected grid rows, as tabular rows with
// the values the grid shows.
func (p *GormPlugin) ExportData(config *engine.PluginConfig, schema string, storageUnit string, writer engine.ExportWriter, selection *engine.ExportSelection) error {
	metadata, _ := sourcecatalog.ResolveSessionMetadata(string(p.Type))
	if selection.HasRows() {
		return selection.Write(writer, selection.FreeTextColumns(metadata))
	}

	_, err := plugins.WithConnection(config, p.DB, func(db *gorm.DB) (bool, error) {
		builder := p.GormPluginFunctions.CreateSQLBuilder(db)
		query := builder.GetTableQuery(schema, storageUnit)
		if orderBy := p.GormPluginFunctions.GetRowsOrderBy(db, schema, storageUnit); orderBy != "" {
			query = query.Order(orderBy)
		}
		// The grid reads a table as pages with a bound LIMIT, and some drivers
		// format values differently when a statement has no bound parameter.
		// Reading the table as one unbounded page keeps the grid's formatting
		// and, being a single statement, cannot repeat or skip rows.
		query = query.Limit(math.MaxInt)

		rows, err := query.Rows()
		if err != nil {
			return false, fmt.Errorf("failed to query data: %w", err)
		}
		defer func() { _ = rows.Close() }()

		columns, typeMap, resultColumns, err := p.describeRawColumns(rows)
		if err != nil {
			return false, err
		}

		// Some types are reported by numeric id, which the grid resolves after
		// reading its page. Resolve them before streaming, on a closed result,
		// so the lookup never competes with the open one for a connection.
		if slices.ContainsFunc(resultColumns, hasUnresolvedType) {
			_ = rows.Close()
			for i := range resultColumns {
				if hasUnresolvedType(resultColumns[i]) {
					resultColumns[i].Type = p.FindMissingDataType(db, resultColumns[i].Type)
				}
			}
			if rows, err = query.Rows(); err != nil {
				return false, fmt.Errorf("failed to query data: %w", err)
			}
		}

		exportWriter := &ExportStreamWriter{Writer: writer, Metadata: metadata}
		if err := exportWriter.WriteColumns(resultColumns); err != nil {
			return false, fmt.Errorf("failed to write headers: %w", err)
		}
		for rows.Next() {
			row, err := p.scanRawRow(rows, columns, typeMap)
			if err != nil {
				return false, err
			}
			if err := exportWriter.WriteRow(row); err != nil {
				return false, fmt.Errorf("failed to write row: %w", err)
			}
		}
		return true, rows.Err()
	})
	return err
}

func hasUnresolvedType(column engine.Column) bool {
	_, err := strconv.Atoi(column.Type)
	return err == nil
}

// ExportStreamWriter adapts a streamed query result to an export writer,
// marking the free-text columns from the source's declared types.
type ExportStreamWriter struct {
	Writer   engine.ExportWriter
	Metadata *source.TypeSessionMetadata
	text     []bool
}

// WriteColumns writes the header row and records which columns are free text.
func (w *ExportStreamWriter) WriteColumns(columns []engine.Column) error {
	headers := make([]string, len(columns))
	for i, column := range columns {
		headers[i] = column.Name
	}
	w.text = source.FreeTextColumns(w.Metadata, columns)
	return w.Writer(headers, nil)
}

// WriteRow writes one data row.
func (w *ExportStreamWriter) WriteRow(row []string) error {
	return w.Writer(row, w.text)
}

// GetPlaceholder returns the placeholder for prepared statements
// Override this in database-specific implementations
func (p *GormPlugin) GetPlaceholder(index int) string {
	return "?"
}
