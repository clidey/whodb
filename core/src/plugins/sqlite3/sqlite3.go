// Copyright 2025 Clidey, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package sqlite3

import (
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/src/plugins"
	gorm_plugin "github.com/clidey/whodb/core/src/plugins/gorm"
	mapset "github.com/deckarep/golang-set/v2"

	"github.com/clidey/whodb/core/src/engine"
	"gorm.io/gorm"
)

var (
	supportedColumnDataTypes = mapset.NewSet(
		"NULL", "INTEGER", "REAL", "TEXT", "BLOB",
		"NUMERIC", "BOOLEAN", "DATE", "DATETIME",
	)
)

type Sqlite3Plugin struct {
	gorm_plugin.GormPlugin
}

func (p *Sqlite3Plugin) GetSupportedColumnDataTypes() mapset.Set[string] {
	return supportedColumnDataTypes
}

func (p *Sqlite3Plugin) GetAllSchemasQuery() string {
	return ""
}

func (p *Sqlite3Plugin) FormTableName(schema string, storageUnit string) string {
	return storageUnit
}

func (p *Sqlite3Plugin) GetDatabases(config *engine.PluginConfig) ([]string, error) {
	directory := getDefaultDirectory()
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, err
	}

	databases := []string{}
	for _, e := range entries {
		databases = append(databases, e.Name())
	}

	return databases, nil
}

func (p *Sqlite3Plugin) GetAllSchemas(config *engine.PluginConfig) ([]string, error) {
	return nil, errors.ErrUnsupported
}

func (p *Sqlite3Plugin) GetTableInfoQuery() string {
	return `
		SELECT
			name AS table_name,
			type AS table_type
		FROM
			sqlite_master
		WHERE
			type='table' AND name NOT LIKE 'sqlite_%'
	`
}

func (p *Sqlite3Plugin) GetTableNameAndAttributes(rows *sql.Rows, db *gorm.DB) (string, []engine.Record) {
	var tableName, tableType string
	if err := rows.Scan(&tableName, &tableType); err != nil {
		log.Fatal(err)
	}

	var rowCount int64
	escapedTableName := p.EscapeIdentifier(tableName)
	rowCountRow := db.Raw(fmt.Sprintf("SELECT COUNT(*) FROM %s", escapedTableName)).Row()
	err := rowCountRow.Scan(&rowCount)
	if err != nil {
		return "", nil
	}

	attributes := []engine.Record{
		{Key: "Type", Value: tableType},
		{Key: "Count", Value: fmt.Sprintf("%d", rowCount)},
	}

	return tableName, attributes
}

func (p *Sqlite3Plugin) GetSchemaTableQuery() string {
	return `
		SELECT m.name AS TABLE_NAME,
			   p.name AS COLUMN_NAME,
			   p.type AS DATA_TYPE
		FROM sqlite_master m,
			 pragma_table_info(m.name) p
		WHERE m.type = 'table'
		  AND m.name NOT LIKE 'sqlite_%';
	`
}

func (p *Sqlite3Plugin) executeRawSQL(config *engine.PluginConfig, query string, params ...interface{}) (*engine.GetRowsResult, error) {
	return plugins.WithConnection(config, p.DB, func(db *gorm.DB) (*engine.GetRowsResult, error) {
		rows, err := db.Raw(query, params...).Rows()
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		return p.ConvertRawToRows(rows)
	})
}

func (p *Sqlite3Plugin) RawExecute(config *engine.PluginConfig, query string) (*engine.GetRowsResult, error) {
	return p.executeRawSQL(config, query)
}

// ConvertRawToRows converts raw SQL rows to structured result with custom datetime handling
func (p *Sqlite3Plugin) ConvertRawToRows(rows *sql.Rows) (*engine.GetRowsResult, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, err
	}

	// Create a map for faster column type lookup
	typeMap := make(map[string]*sql.ColumnType, len(columnTypes))
	for _, colType := range columnTypes {
		typeMap[colType.Name()] = colType
	}

	result := &engine.GetRowsResult{
		Columns: make([]engine.Column, 0, len(columns)),
		Rows:    make([][]string, 0, 100),
	}

	// Build columns with type information
	for _, col := range columns {
		if colType, exists := typeMap[col]; exists {
			colTypeName := colType.DatabaseTypeName()
			result.Columns = append(result.Columns, engine.Column{Name: col, Type: colTypeName})
		}
	}

	for rows.Next() {
		columnPointers := make([]interface{}, len(columns))
		row := make([]string, len(columns))

		for i, col := range columns {
			colType := typeMap[col]
			typeName := strings.ToUpper(colType.DatabaseTypeName())

			// Use custom DateTimeString type for datetime columns to prevent parsing
			switch typeName {
			case "DATE", "DATETIME", "TIMESTAMP":
				columnPointers[i] = new(DateTimeString)
			case "BLOB":
				columnPointers[i] = new(sql.RawBytes)
			default:
				columnPointers[i] = new(sql.NullString)
			}
		}

		if err := rows.Scan(columnPointers...); err != nil {
			return nil, err
		}

		for i, colPtr := range columnPointers {
			colType := typeMap[columns[i]]
			typeName := strings.ToUpper(colType.DatabaseTypeName())

			switch typeName {
			case "DATE", "DATETIME", "TIMESTAMP":
				// Get the string value from our custom type
				dateStr := colPtr.(*DateTimeString)
				row[i] = string(*dateStr)
			case "BLOB":
				rawBytes := colPtr.(*sql.RawBytes)
				if rawBytes == nil || len(*rawBytes) == 0 {
					row[i] = ""
				} else {
					row[i] = "0x" + hex.EncodeToString(*rawBytes)
				}
			default:
				val := colPtr.(*sql.NullString)
				if val.Valid {
					row[i] = val.String
				} else {
					row[i] = ""
				}
			}
		}

		result.Rows = append(result.Rows, row)
	}

	return result, nil
}

// GetRows overrides the base implementation to handle datetime columns specially
func (p *Sqlite3Plugin) GetRows(config *engine.PluginConfig, schema string, storageUnit string, where *model.WhereCondition, pageSize int, pageOffset int) (*engine.GetRowsResult, error) {
	return plugins.WithConnection(config, p.DB, func(db *gorm.DB) (*engine.GetRowsResult, error) {
		// First, get column information to identify datetime columns
		var columnInfo []struct {
			Name string `gorm:"column:name"`
			Type string `gorm:"column:type"`
		}

		colQuery := `SELECT name, type FROM pragma_table_info(?)`
		if err := db.Raw(colQuery, storageUnit).Scan(&columnInfo).Error; err != nil {
			// Fall back to base implementation if we can't get column info
			return p.GormPlugin.GetRows(config, schema, storageUnit, where, pageSize, pageOffset)
		}

		// Build a SELECT query that casts datetime columns as TEXT
		selectParts := make([]string, 0, len(columnInfo))
		for _, col := range columnInfo {
			colType := strings.ToUpper(col.Type)
			escapedName := p.EscapeIdentifier(col.Name)

			// For datetime columns, explicitly cast to TEXT to prevent driver parsing
			if colType == "DATE" || colType == "DATETIME" || colType == "TIMESTAMP" {
				selectParts = append(selectParts, fmt.Sprintf("CAST(%s AS TEXT) AS %s", escapedName, escapedName))
			} else {
				selectParts = append(selectParts, escapedName)
			}
		}

		// If no columns found, fall back to base implementation
		if len(selectParts) == 0 {
			return p.GormPlugin.GetRows(config, schema, storageUnit, where, pageSize, pageOffset)
		}

		// Build the full query
		escapedTable := p.EscapeIdentifier(storageUnit)
		selectClause := strings.Join(selectParts, ", ")
		query := fmt.Sprintf("SELECT %s FROM %s", selectClause, escapedTable)

		// Apply WHERE conditions if any
		var args []interface{}
		if where != nil {
			whereClause, whereArgs, err := p.buildWhereClause(where)
			if err != nil {
				return nil, err
			}
			if whereClause != "" {
				query += " WHERE " + whereClause
				args = whereArgs
			}
		}

		// Apply pagination
		query += fmt.Sprintf(" LIMIT %d OFFSET %d", pageSize, pageOffset)

		// Execute the query
		rows, err := db.Raw(query, args...).Rows()
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		return p.ConvertRawToRows(rows)
	})
}

// buildWhereClause converts WhereCondition to SQL WHERE clause
func (p *Sqlite3Plugin) buildWhereClause(condition *model.WhereCondition) (string, []interface{}, error) {
	if condition == nil {
		return "", nil, nil
	}

	var clauses []string
	var args []interface{}

	switch condition.Type {
	case model.WhereConditionTypeAtomic:
		if condition.Atomic != nil {
			value, err := p.ConvertStringValue(condition.Atomic.Value, condition.Atomic.ColumnType)
			if err != nil {
				return "", nil, err
			}
			clauses = append(clauses, fmt.Sprintf("%s = ?", p.EscapeIdentifier(condition.Atomic.Key)))
			args = append(args, value)
		}

	case model.WhereConditionTypeAnd:
		if condition.And != nil {
			for _, child := range condition.And.Children {
				childClause, childArgs, err := p.buildWhereClause(child)
				if err != nil {
					return "", nil, err
				}
				if childClause != "" {
					clauses = append(clauses, "("+childClause+")")
					args = append(args, childArgs...)
				}
			}
			return strings.Join(clauses, " AND "), args, nil
		}

	case model.WhereConditionTypeOr:
		if condition.Or != nil {
			for _, child := range condition.Or.Children {
				childClause, childArgs, err := p.buildWhereClause(child)
				if err != nil {
					return "", nil, err
				}
				if childClause != "" {
					clauses = append(clauses, "("+childClause+")")
					args = append(args, childArgs...)
				}
			}
			return strings.Join(clauses, " OR "), args, nil
		}
	}

	return strings.Join(clauses, " AND "), args, nil
}

func NewSqlite3Plugin() *engine.Plugin {
	plugin := &Sqlite3Plugin{}
	plugin.Type = engine.DatabaseType_Sqlite3
	plugin.PluginFunctions = plugin
	plugin.GormPluginFunctions = plugin
	return &plugin.Plugin
}
