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

package sqlite3

import (
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/env"
	"github.com/clidey/whodb/core/src/log"
	"github.com/clidey/whodb/core/src/plugins"
	gorm_plugin "github.com/clidey/whodb/core/src/plugins/gorm"
	"gorm.io/gorm"
)

// CreateSQLBuilder creates a SQLite-specific SQL builder.
func (p *Sqlite3Plugin) CreateSQLBuilder(db *gorm.DB) gorm_plugin.SQLBuilderInterface {
	return NewSQLiteSQLBuilder(db, p)
}

var (
	supportedOperators = map[string]string{
		"=": "=", ">=": ">=", ">": ">", "<=": "<=", "<": "<", "<>": "<>", "!=": "!=",
		"BETWEEN": "BETWEEN", "NOT BETWEEN": "NOT BETWEEN",
		"LIKE": "LIKE", "NOT LIKE": "NOT LIKE", "GLOB": "GLOB",
		"IN": "IN", "NOT IN": "NOT IN", "IS NULL": "IS NULL", "IS NOT NULL": "IS NOT NULL",
		"AND": "AND", "OR": "OR", "NOT": "NOT",
	}
)

type Sqlite3Plugin struct {
	gorm_plugin.GormPlugin
	strictTableCache map[string]bool
	cacheMutex       sync.RWMutex
}

func (p *Sqlite3Plugin) GetSupportedOperators() map[string]string {
	return supportedOperators
}

func (p *Sqlite3Plugin) GetAllSchemasQuery() string {
	return ""
}

func (p *Sqlite3Plugin) FormTableName(schema string, storageUnit string) string {
	return storageUnit
}

func (p *Sqlite3Plugin) GetDatabases(config *engine.PluginConfig) ([]string, error) {
	if env.GetIsDesktopMode() {
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

func (p *Sqlite3Plugin) GetStorageUnitExistsQuery() string {
	// First param is schema (ignored for SQLite), second is table name
	// LENGTH(IFNULL(?, '')) >= 0 always evaluates to true, consuming the schema param
	return `SELECT LENGTH(IFNULL(?, '')) >= 0 AND EXISTS(SELECT 1 FROM sqlite_master WHERE type='table' AND name = ?)`
}

// IsTableStrict checks if a table is STRICT using PRAGMA table_list
// Returns false if detection fails or SQLite version doesn't support STRICT tables
func (p *Sqlite3Plugin) IsTableStrict(db *gorm.DB, tableName string) bool {
	// Check cache first
	p.cacheMutex.RLock()
	if p.strictTableCache != nil {
		if isStrict, exists := p.strictTableCache[tableName]; exists {
			p.cacheMutex.RUnlock()
			return isStrict
		}
	}
	p.cacheMutex.RUnlock()

	// Query PRAGMA table_list to check if table is STRICT
	// The strict column is available in SQLite 3.37.0+
	var strict int
	err := db.Raw("SELECT strict FROM pragma_table_list WHERE name = ?", tableName).Scan(&strict).Error

	// If error or no result, assume non-STRICT for backward compatibility
	isStrict := err == nil && strict == 1

	// Cache the result
	p.cacheMutex.Lock()
	if p.strictTableCache == nil {
		p.strictTableCache = make(map[string]bool)
	}
	p.strictTableCache[tableName] = isStrict
	p.cacheMutex.Unlock()

	return isStrict
}

func (p *Sqlite3Plugin) GetPlaceholder(index int) string {
	return "?"
}

// GetColumnsForTable overrides the base implementation to properly detect SQLite auto-increment.
func (p *Sqlite3Plugin) GetColumnsForTable(config *engine.PluginConfig, schema, storageUnit string) ([]engine.Column, error) {
	columns, err := p.GormPlugin.GetColumnsForTable(config, schema, storageUnit)
	if err != nil {
		return nil, err
	}

	// SQLite's INTEGER PRIMARY KEY is always auto-increment
	_, err = plugins.WithConnection(config, p.DB, func(db *gorm.DB) (bool, error) {
		builder, ok := p.CreateSQLBuilder(db).(*SQLiteSQLBuilder)
		if !ok {
			return false, fmt.Errorf("failed to create SQLite SQL builder")
		}

		tableInfoQuery, err := builder.PragmaQuery("table_info", storageUnit)
		if err != nil {
			return false, err
		}

		rows, err := db.Raw(tableInfoQuery).Rows()
		if err != nil {
			return false, err
		}
		defer rows.Close()

		// Build a map of primary key columns with their types
		pkColTypes := make(map[string]string)
		for rows.Next() {
			var cid int
			var name string
			var dataType string
			var notNull int
			var dfltValue any
			var pk int

			if err := rows.Scan(&cid, &name, &dataType, &notNull, &dfltValue, &pk); err != nil {
				continue
			}

			// pk = 1 means this column is part of the PRIMARY KEY
			if pk == 1 {
				pkColTypes[name] = strings.ToUpper(dataType)
			}
		}

		// Update IsAutoIncrement for INTEGER PRIMARY KEY columns
		for i := range columns {
			if columns[i].IsPrimary {
				colType := strings.ToUpper(columns[i].Type)
				pkType, isPK := pkColTypes[columns[i].Name]

				// SQLite INTEGER PRIMARY KEY is auto-increment when it's the only PK column
				// and the type is exactly INTEGER (not INT, not BIGINT)
				if isPK && (pkType == "INTEGER" || colType == "INTEGER") && len(pkColTypes) == 1 {
					if !columns[i].IsAutoIncrement {
						log.Logger.WithFields(map[string]any{
							"table":  storageUnit,
							"column": columns[i].Name,
						}).Debug("Detected SQLite INTEGER PRIMARY KEY as auto-increment")
						columns[i].IsAutoIncrement = true
					}
				}
			}
		}

		return true, nil
	})

	if err != nil {
		log.Logger.WithError(err).Warn("Failed to detect SQLite auto-increment columns, using GORM defaults")
	}

	return columns, nil
}

func (p *Sqlite3Plugin) GetTableNameAndAttributes(rows *sql.Rows) (string, []engine.Record) {
	var tableName, tableType string
	if err := rows.Scan(&tableName, &tableType); err != nil {
		log.Logger.WithError(err).Error("Failed to scan SQLite table information from rows")
		return "", nil
	}

	attributes := []engine.Record{
		{Key: "Type", Value: tableType},
	}

	return tableName, attributes
}

// GetRows overrides the base GORM implementation to handle SQLite datetime quirks
func (p *Sqlite3Plugin) GetRows(config *engine.PluginConfig, schema string, storageUnit string, where *model.WhereCondition, sort []*model.SortCondition, pageSize, pageOffset int) (*engine.GetRowsResult, error) {
	return plugins.WithConnection(config, p.DB, func(db *gorm.DB) (*engine.GetRowsResult, error) {
		builder := gorm_plugin.NewSQLBuilder(db, p)
		fullTable := builder.BuildFullTableName("", storageUnit)

		// Start count query in a separate goroutine for parallel execution
		var totalCount int64
		countDone := make(chan error, 1)
		go func() {
			columnTypes, _ := p.GetColumnTypes(db, schema, storageUnit)
			// codeql[go/sql-injection]: table name validated by StorageUnitExists before reaching this code
			countQuery := db.Table(fullTable)
			var err error
			countQuery, err = p.ApplyWhereConditions(countQuery, where, columnTypes)
			if err != nil {
				countDone <- err
				return
			}
			countDone <- countQuery.Count(&totalCount).Error
		}()

		// Check if table is STRICT
		isStrict := p.IsTableStrict(db, storageUnit)

		var result *engine.GetRowsResult

		// For STRICT tables, delegate to parent GORM implementation without CAST
		if isStrict {
			// codeql[go/sql-injection]: table name validated by StorageUnitExists before reaching this code
			query := db.Table(fullTable)

			// Get column types for WHERE conditions
			columnTypes, _ := p.GetColumnTypes(db, schema, storageUnit)

			query, err := p.ApplyWhereConditions(query, where, columnTypes)
			if err != nil {
				log.Logger.WithError(err).Error(fmt.Sprintf("Failed to apply where conditions for STRICT table %s", storageUnit))
				return nil, err
			}

			// Apply sorting
			if len(sort) > 0 {
				sortList := make([]plugins.Sort, len(sort))
				for i, s := range sort {
					sortList[i] = plugins.Sort{Column: s.Column, Direction: plugins.Down}
					if s.Direction == model.SortDirectionAsc {
						sortList[i].Direction = plugins.Up
					}
				}
				query = builder.BuildOrderBy(query, sortList)
			} else {
				if orderBy := p.GormPluginFunctions.GetRowsOrderBy(db, schema, storageUnit); orderBy != "" {
					query = query.Order(orderBy)
				}
			}

			query = query.Limit(pageSize).Offset(pageOffset)

			rows, err := query.Rows()
			if err != nil {
				log.Logger.WithError(err).Error(fmt.Sprintf("Failed to execute SQLite rows query for STRICT table %s", storageUnit))
				return nil, err
			}
			defer rows.Close()

			// Use parent's ConvertRawToRows for STRICT tables
			result, err = p.GormPlugin.ConvertRawToRows(rows)
			if err != nil {
				return nil, err
			}
		} else {
			// For non-STRICT tables, use custom handling with CAST for date/time types
			orderedColumns, columnTypes, err := p.GetOrderedColumnsWithTypes(db, schema, storageUnit)
			if err != nil {
				log.Logger.WithError(err).Error(fmt.Sprintf("Failed to get column types for table %s.%s", schema, storageUnit))
				return nil, err
			}

			selects := make([]string, 0, len(orderedColumns))
			for _, col := range orderedColumns {
				upper := strings.ToUpper(col.Type)
				// Only apply CAST for non-STRICT tables
				if upper == "DATE" || upper == "DATETIME" || upper == "TIMESTAMP" {
					selects = append(selects, fmt.Sprintf("CAST(%s AS TEXT) AS %s", builder.QuoteIdentifier(col.Name), builder.QuoteIdentifier(col.Name)))
				} else {
					selects = append(selects, builder.QuoteIdentifier(col.Name))
				}
			}

			// codeql[go/sql-injection]: table name validated by StorageUnitExists before reaching this code
			query := db.Table(fullTable).Select(selects)

			query, err = p.ApplyWhereConditions(query, where, columnTypes)
			if err != nil {
				log.Logger.WithError(err).Error(fmt.Sprintf("Failed to apply where conditions for table %s.%s", schema, storageUnit))
				return nil, err
			}

			// Sorting
			if len(sort) > 0 {
				sortList := make([]plugins.Sort, len(sort))
				for i, s := range sort {
					sortList[i] = plugins.Sort{Column: s.Column, Direction: plugins.Down}
					if s.Direction == model.SortDirectionAsc {
						sortList[i].Direction = plugins.Up
					}
				}
				query = builder.BuildOrderBy(query, sortList)
			} else {
				if orderBy := p.GormPluginFunctions.GetRowsOrderBy(db, schema, storageUnit); orderBy != "" {
					query = query.Order(orderBy)
				}
			}

			query = query.Limit(pageSize).Offset(pageOffset)

			rows, err := query.Rows()
			if err != nil {
				log.Logger.WithError(err).Error(fmt.Sprintf("Failed to execute SQLite rows query for table %s.%s", schema, storageUnit))
				return nil, err
			}
			defer rows.Close()

			result, err = p.ConvertRawToRows(rows)
			if err != nil {
				return nil, err
			}
		}

		// Wait for count query to complete and set TotalCount
		if countErr := <-countDone; countErr != nil {
			log.Logger.WithError(countErr).Warn(fmt.Sprintf("Failed to get row count for table %s", storageUnit))
		} else {
			result.TotalCount = totalCount
		}

		return result, nil
	})
}

func (p *Sqlite3Plugin) executeRawSQL(config *engine.PluginConfig, query string, params ...any) (*engine.GetRowsResult, error) {
	return plugins.WithConnection(config, p.DB, func(db *gorm.DB) (*engine.GetRowsResult, error) {
		rows, err := db.Raw(query, params...).Rows()
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		// For raw SQL, we default to non-strict behavior for backward compatibility
		// since we can't reliably determine table strictness from arbitrary queries
		return p.ConvertRawToRows(rows)
	})
}

func (p *Sqlite3Plugin) RawExecute(config *engine.PluginConfig, query string) (*engine.GetRowsResult, error) {
	return p.executeRawSQL(config, query)
}

// ConvertRawToRows overrides the parent to handle SQLite datetime columns specially
// This maintains backward compatibility for non-STRICT tables
func (p *Sqlite3Plugin) ConvertRawToRows(rows *sql.Rows) (*engine.GetRowsResult, error) {
	// Default to non-STRICT handling for backward compatibility
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, err
	}

	// Check if we have any datetime columns
	hasDateTimeColumns := false
	for _, colType := range columnTypes {
		typeName := strings.ToUpper(colType.DatabaseTypeName())
		if typeName == "DATE" || typeName == "DATETIME" || typeName == "TIMESTAMP" {
			hasDateTimeColumns = true
			break
		}
	}

	// If no datetime columns even in non-STRICT table, use parent implementation
	if !hasDateTimeColumns {
		return p.GormPlugin.ConvertRawToRows(rows)
	}

	// Custom implementation for datetime preservation
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
		columnPointers := make([]any, len(columns))
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

func (p *Sqlite3Plugin) GetForeignKeyRelationships(config *engine.PluginConfig, schema string, storageUnit string) (map[string]*engine.ForeignKeyRelationship, error) {
	return plugins.WithConnection(config, p.DB, func(db *gorm.DB) (map[string]*engine.ForeignKeyRelationship, error) {
		escapedTable := strings.ReplaceAll(storageUnit, "'", "''")
		query := fmt.Sprintf("PRAGMA foreign_key_list('%s')", escapedTable)

		rows, err := db.Raw(query).Rows()
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		relationships := make(map[string]*engine.ForeignKeyRelationship)
		for rows.Next() {
			var id, seq int
			var table, from, to, onUpdate, onDelete, match string
			if err := rows.Scan(&id, &seq, &table, &from, &to, &onUpdate, &onDelete, &match); err != nil {
				log.Logger.WithError(err).Error("Failed to scan foreign key relationship")
				continue
			}
			relationships[from] = &engine.ForeignKeyRelationship{
				ColumnName:       from,
				ReferencedTable:  table,
				ReferencedColumn: to,
			}
		}

		return relationships, nil
	})
}

// NormalizeType converts SQLite type aliases to their canonical form.
func (p *Sqlite3Plugin) NormalizeType(typeName string) string {
	return NormalizeType(typeName)
}

func NewSqlite3Plugin() *engine.Plugin {
	plugin := &Sqlite3Plugin{}
	plugin.Type = engine.DatabaseType_Sqlite3
	plugin.PluginFunctions = plugin
	plugin.GormPluginFunctions = plugin
	return &plugin.Plugin
}
