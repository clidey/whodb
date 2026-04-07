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

package duckdb

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/env"
	"github.com/clidey/whodb/core/src/log"
	"github.com/clidey/whodb/core/src/plugins"
	gorm_plugin "github.com/clidey/whodb/core/src/plugins/gorm"
	duckdbDriver "github.com/duckdb/duckdb-go/v2"
	"gorm.io/gorm"
)

var supportedOperators = map[string]string{
	"=": "=", ">=": ">=", ">": ">", "<=": "<=", "<": "<", "<>": "<>", "!=": "!=",
	"BETWEEN": "BETWEEN", "NOT BETWEEN": "NOT BETWEEN",
	"LIKE": "LIKE", "NOT LIKE": "NOT LIKE",
	"ILIKE": "ILIKE", "NOT ILIKE": "NOT ILIKE",
	"IN": "IN", "NOT IN": "NOT IN", "IS NULL": "IS NULL", "IS NOT NULL": "IS NOT NULL",
	"AND": "AND", "OR": "OR", "NOT": "NOT",
}

// DuckDBPlugin implements the WhoDB plugin for DuckDB.
type DuckDBPlugin struct {
	gorm_plugin.GormPlugin
}

func (p *DuckDBPlugin) GetSupportedOperators() map[string]string {
	return supportedOperators
}

func (p *DuckDBPlugin) GetAllSchemasQuery() string {
	return `SELECT schema_name FROM information_schema.schemata WHERE catalog_name = current_database()`
}

func (p *DuckDBPlugin) GetTableInfoQuery() string {
	return `
		SELECT
			t.table_name,
			t.table_type
		FROM
			information_schema.tables t
		WHERE
			t.table_schema = ?
	`
}

func (p *DuckDBPlugin) GetStorageUnitExistsQuery() string {
	return `SELECT COUNT(*) > 0 FROM information_schema.tables WHERE table_schema = ? AND table_name = ?`
}

func (p *DuckDBPlugin) GetPrimaryKeyColQuery() string {
	return `
		SELECT unnest(dc.constraint_column_names) AS column_name
		FROM duckdb_constraints() dc
		WHERE dc.constraint_type = 'PRIMARY KEY' AND dc.schema_name = ? AND dc.table_name = ?
	`
}

func (p *DuckDBPlugin) GetPlaceholder(index int) string {
	return "?"
}

func (p *DuckDBPlugin) GetTableNameAndAttributes(rows *sql.Rows) (string, []engine.Record) {
	var tableName, tableType string
	if err := rows.Scan(&tableName, &tableType); err != nil {
		log.WithError(err).Error("Failed to scan DuckDB table info row")
		return "", nil
	}

	attributes := []engine.Record{
		{Key: "Type", Value: tableType},
	}

	return tableName, attributes
}

func (p *DuckDBPlugin) FormTableName(schema string, storageUnit string) string {
	if schema == "" {
		return storageUnit
	}
	return schema + "." + storageUnit
}

func (p *DuckDBPlugin) GetDatabases(config *engine.PluginConfig) ([]string, error) {
	if env.GetIsLocalMode() {
		return []string{}, nil
	}

	directory := getDefaultDirectory()
	entries, err := os.ReadDir(directory)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	var databases []string
	for _, e := range entries {
		databases = append(databases, e.Name())
	}
	return databases, nil
}

func (p *DuckDBPlugin) GetRowsOrderBy(db *gorm.DB, schema string, storageUnit string) string {
	return ""
}

// ConvertRawToRows overrides the base to check rows.Err() after iteration.
// The go-duckdb driver can fail during rows.Next() for unsupported types,
// and the base implementation does not check rows.Err().
func (p *DuckDBPlugin) ConvertRawToRows(rows *sql.Rows) (*engine.GetRowsResult, error) {
	result, err := p.GormPlugin.ConvertRawToRows(rows)
	if err != nil {
		return nil, err
	}
	if rowErr := rows.Err(); rowErr != nil {
		return nil, rowErr
	}
	return result, nil
}

// ShouldHandleColumnType returns true for all types. The go-duckdb driver returns
// native Go types (int8, int32, float32, duckdb.Decimal, duckdb.Interval, etc.)
// that are not in the database/sql driver.Value set, so sql.NullString.Scan() fails
// for most of them. We use a generic interface{} scanner for everything.
func (p *DuckDBPlugin) ShouldHandleColumnType(columnType string) bool {
	return true
}

// GetColumnScanner returns a generic scanner for DuckDB types.
func (p *DuckDBPlugin) GetColumnScanner(columnType string) any {
	return new(any)
}

// FormatColumnValue converts a scanned DuckDB value to its string representation.
func (p *DuckDBPlugin) FormatColumnValue(columnType string, scanner any) (string, error) {
	ptr, ok := scanner.(*any)
	if !ok || ptr == nil || *ptr == nil {
		return "", nil
	}
	val := *ptr

	switch v := val.(type) {
	case []byte:
		if len(v) == 16 && strings.ToUpper(columnType) == "UUID" {
			return fmt.Sprintf("%x-%x-%x-%x-%x", v[0:4], v[4:6], v[6:8], v[8:10], v[10:16]), nil
		}
		return "0x" + fmt.Sprintf("%X", v), nil
	case time.Time:
		if v.IsZero() {
			return "", nil
		}
		upper := strings.ToUpper(columnType)
		if upper == "DATE" {
			return v.Format("2006-01-02"), nil
		}
		if upper == "TIME" || upper == "TIME WITH TIME ZONE" {
			return v.Format("15:04:05"), nil
		}
		return v.Format("2006-01-02 15:04:05"), nil
	case duckdbDriver.Interval:
		return formatInterval(v), nil
	case map[string]any, []any:
		b, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v), nil
		}
		return string(b), nil
	default:
		return fmt.Sprintf("%v", v), nil
	}
}

func (p *DuckDBPlugin) GetCustomColumnTypeName(columnName string, defaultTypeName string) string {
	return ""
}

func (p *DuckDBPlugin) IsGeometryType(columnType string) bool {
	return false
}

func (p *DuckDBPlugin) FormatGeometryValue(rawBytes []byte, columnType string) string {
	return ""
}

// HandleCustomDataType converts string values back to DuckDB-compatible types for writes.
// This ensures values round-trip correctly (display → edit → save).
func (p *DuckDBPlugin) HandleCustomDataType(value string, columnType string, isNullable bool) (any, bool, error) {
	if value == "" && isNullable {
		return nil, true, nil
	}

	escaped := strings.ReplaceAll(value, "'", "''")
	upper := strings.ToUpper(columnType)
	switch {
	case upper == "INTERVAL":
		return gorm.Expr(fmt.Sprintf("CAST('%s' AS INTERVAL)", escaped)), true, nil
	case upper == "JSON":
		return gorm.Expr(fmt.Sprintf("CAST('%s' AS JSON)", escaped)), true, nil
	case upper == "UUID":
		return gorm.Expr(fmt.Sprintf("CAST('%s' AS UUID)", escaped)), true, nil
	case upper == "BLOB":
		return gorm.Expr(fmt.Sprintf("'%s'::BLOB", escaped)), true, nil
	case upper == "HUGEINT":
		return gorm.Expr(fmt.Sprintf("CAST('%s' AS HUGEINT)", escaped)), true, nil
	case strings.HasPrefix(upper, "DECIMAL"):
		return gorm.Expr(fmt.Sprintf("CAST('%s' AS %s)", escaped, columnType)), true, nil
	}
	return nil, false, nil
}

func (p *DuckDBPlugin) IsArrayType(columnType string) bool {
	return false
}

func (p *DuckDBPlugin) ResolveGraphSchema(config *engine.PluginConfig, schema string) string {
	return schema
}

func (p *DuckDBPlugin) ShouldCheckRowsAffected() bool {
	return true
}

// GetForeignKeyRelationships uses duckdb_constraints() instead of information_schema
// because DuckDB does not populate constraint_column_usage for foreign keys.
func (p *DuckDBPlugin) GetForeignKeyRelationships(config *engine.PluginConfig, schema string, storageUnit string) (map[string]*engine.ForeignKeyRelationship, error) {
	query := `
		SELECT
			unnest(dc.constraint_column_names) AS column_name,
			dc.referenced_table AS referenced_table,
			unnest(dc.referenced_column_names) AS referenced_column
		FROM duckdb_constraints() dc
		WHERE dc.constraint_type = 'FOREIGN KEY'
			AND dc.schema_name = ?
			AND dc.table_name = ?
	`
	return p.QueryForeignKeyRelationships(config, query, schema, storageUnit)
}

func (p *DuckDBPlugin) NormalizeType(typeName string) string {
	return NormalizeType(typeName)
}

// GetLastInsertID returns 0 for DuckDB — DuckDB has no session-scoped lastval().
// The actual ID retrieval is handled by AddRowReturningID using INSERT ... RETURNING.
func (p *DuckDBPlugin) GetLastInsertID(db *gorm.DB) (int64, error) {
	return 0, nil
}

// AddRowReturningID overrides the base implementation to use INSERT ... RETURNING,
// since DuckDB has no lastval() or last_insert_rowid() function.
func (p *DuckDBPlugin) AddRowReturningID(config *engine.PluginConfig, schema string, storageUnit string, values []engine.Record) (int64, error) {
	if storageUnit == "" {
		return 0, fmt.Errorf("storage unit name cannot be empty")
	}

	return plugins.WithConnection(config, p.DB, func(db *gorm.DB) (int64, error) {
		var lastID int64

		err := db.Transaction(func(tx *gorm.DB) error {
			// Fetch column types for proper conversion
			columnTypeInfos, err := p.GetColumnTypes(tx, schema, storageUnit)
			if err != nil {
				columnTypeInfos = make(map[string]gorm_plugin.ColumnTypeInfo)
			}

			// Enrich values with type info (same as GormPlugin.addRowWithDB)
			for i, value := range values {
				if values[i].Extra == nil {
					values[i].Extra = make(map[string]string)
				}
				if values[i].Extra["Type"] == "" {
					if colInfo, ok := columnTypeInfos[value.Key]; ok {
						values[i].Extra["Type"] = colInfo.Type
						if colInfo.IsNullable {
							values[i].Extra["IsNullable"] = "true"
						}
					}
				}
			}

			valuesToAdd, err := p.ConvertRecordValuesToMap(values)
			if err != nil {
				return err
			}

			// Get PK column for RETURNING clause
			pkCols, pkErr := p.GetPrimaryKeyColumns(tx, schema, storageUnit)
			if pkErr != nil || len(pkCols) == 0 {
				// No PK — fall back to regular insert without RETURNING
				builder := p.CreateSQLBuilder(tx)
				return builder.InsertRow(schema, storageUnit, valuesToAdd)
			}

			// Build INSERT ... RETURNING pk_column
			builder := p.CreateSQLBuilder(tx)
			tableName := builder.BuildFullTableName(schema, storageUnit)
			pkColQuoted := builder.QuoteIdentifier(pkCols[0])

			var cols []string
			var placeholders []string
			var args []any
			for col, val := range valuesToAdd {
				cols = append(cols, builder.QuoteIdentifier(col))
				placeholders = append(placeholders, "?")
				args = append(args, val)
			}

			query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s) RETURNING %s",
				tableName,
				strings.Join(cols, ", "),
				strings.Join(placeholders, ", "),
				pkColQuoted,
			)

			if err := tx.Raw(query, args...).Scan(&lastID).Error; err != nil {
				return err
			}

			return nil
		})

		if err != nil {
			return 0, err
		}
		return lastID, nil
	})
}

func (p *DuckDBPlugin) GetMaxBulkInsertParameters() int {
	return 65535
}

func (p *DuckDBPlugin) MarkGeneratedColumns(config *engine.PluginConfig, schema string, storageUnit string, columns []engine.Column) error {
	computed, err := p.QueryComputedColumns(config, `
		SELECT column_name FROM information_schema.columns
		WHERE table_schema = ? AND table_name = ?
			AND (is_identity = 'YES' OR is_generated = 'ALWAYS')
	`, schema, storageUnit)
	if err != nil {
		return err
	}

	for i := range columns {
		if computed[columns[i].Name] {
			columns[i].IsComputed = true
		}
	}
	return nil
}

func (p *DuckDBPlugin) RawExecute(config *engine.PluginConfig, query string, params ...any) (*engine.GetRowsResult, error) {
	return p.ExecuteRawSQL(config, nil, query, params...)
}

func formatInterval(v duckdbDriver.Interval) string {
	var parts []string
	if v.Months != 0 {
		years := v.Months / 12
		months := v.Months % 12
		if years != 0 {
			parts = append(parts, fmt.Sprintf("%d year%s", years, plural(int64(years))))
		}
		if months != 0 {
			parts = append(parts, fmt.Sprintf("%d month%s", months, plural(int64(months))))
		}
	}
	if v.Days != 0 {
		parts = append(parts, fmt.Sprintf("%d day%s", v.Days, plural(int64(v.Days))))
	}
	if v.Micros != 0 {
		hours := v.Micros / 3_600_000_000
		remaining := v.Micros % 3_600_000_000
		minutes := remaining / 60_000_000
		remaining = remaining % 60_000_000
		seconds := remaining / 1_000_000
		if hours != 0 {
			parts = append(parts, fmt.Sprintf("%d hour%s", hours, plural(hours)))
		}
		if minutes != 0 {
			parts = append(parts, fmt.Sprintf("%d minute%s", minutes, plural(minutes)))
		}
		if seconds != 0 {
			parts = append(parts, fmt.Sprintf("%d second%s", seconds, plural(seconds)))
		}
	}
	if len(parts) == 0 {
		return "0 seconds"
	}
	return strings.Join(parts, " ")
}

func plural(v int64) string {
	if v == 1 || v == -1 {
		return ""
	}
	return "s"
}

// NewDuckDBPlugin creates a new DuckDB plugin instance.
func NewDuckDBPlugin() *engine.Plugin {
	plugin := &DuckDBPlugin{}
	plugin.Type = engine.DatabaseType_DuckDB
	plugin.PluginFunctions = plugin
	plugin.GormPluginFunctions = plugin
	return &plugin.Plugin
}
