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
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
	"github.com/clidey/whodb/core/src/plugins"
	mapset "github.com/deckarep/golang-set/v2"
	"gorm.io/gorm"
)

type GormPlugin struct {
	engine.Plugin
	GormPluginFunctions
}

type GormPluginFunctions interface {
	// these below are meant to be generic-ish implementations by the base gorm plugin
	ParseConnectionConfig(config *engine.PluginConfig) (*ConnectionInput, error)

	ConvertStringValue(value, columnType string) (interface{}, error)
	ConvertRawToRows(raw *sql.Rows) (*engine.GetRowsResult, error)
	ConvertRecordValuesToMap(values []engine.Record) (map[string]interface{}, error)

	EscapeIdentifier(identifier string) string

	// these below are meant to be implemented by the specific database plugins
	DB(config *engine.PluginConfig) (*gorm.DB, error)
	GetSupportedColumnDataTypes() mapset.Set[string]
	GetPlaceholder(index int) string
	ShouldQuoteIdentifiers() bool

	GetTableInfoQuery() string
	GetSchemaTableQuery() string
	GetPrimaryKeyColQuery() string
	GetColTypeQuery() string
	GetAllSchemasQuery() string
	GetCreateTableQuery(schema string, storageUnit string, columns []engine.Record) string

	FormTableName(schema string, storageUnit string) string
	EscapeSpecificIdentifier(identifier string) string
	ConvertStringValueDuringMap(value, columnType string) (interface{}, error)

	GetSupportedOperators() map[string]string

	GetGraphQueryDB(db *gorm.DB, schema string) *gorm.DB
	GetTableNameAndAttributes(rows *sql.Rows, db *gorm.DB) (string, []engine.Record)

	// GetRowsOrderBy returns the ORDER BY clause for pagination queries
	GetRowsOrderBy(db *gorm.DB, schema string, storageUnit string) string

	// ShouldHandleColumnType returns true if the plugin wants to handle a specific column type
	ShouldHandleColumnType(columnType string) bool

	// GetColumnScanner returns a scanner for a specific column type
	// This is called when ShouldHandleColumnType returns true
	GetColumnScanner(columnType string) interface{}

	// FormatColumnValue formats a scanned value for a specific column type
	// This is called when ShouldHandleColumnType returns true
	FormatColumnValue(columnType string, scanner interface{}) (string, error)

	// GetCustomColumnTypeName returns a custom column type name for display
	// Return empty string to use the default type name
	GetCustomColumnTypeName(columnName string, defaultTypeName string) string

	// IsGeometryType returns true if the column type represents spatial/geometry data
	IsGeometryType(columnType string) bool

	// FormatGeometryValue formats geometry data for display
	// Return empty string to use default hex formatting
	FormatGeometryValue(rawBytes []byte, columnType string) string

	// HandleCustomDataType allows plugins to handle their own data type conversions
	// Return (value, true) if handled, or (nil, false) to use default handling
	HandleCustomDataType(value string, columnType string, isNullable bool) (interface{}, bool, error)
}

func (p *GormPlugin) GetStorageUnits(config *engine.PluginConfig, schema string) ([]engine.StorageUnit, error) {
	return plugins.WithConnection(config, p.DB, func(db *gorm.DB) ([]engine.StorageUnit, error) {
		storageUnits := []engine.StorageUnit{}
		rows, err := db.Raw(p.GetTableInfoQuery(), schema).Rows()
		if err != nil {
			log.Logger.WithError(err).Error("Failed to execute table info query for schema: " + schema)
			return nil, err
		}
		defer rows.Close()

		allTablesWithColumns, err := p.GetTableSchema(db, schema)
		if err != nil {
			log.Logger.WithError(err).Error("Failed to get table schema for schema: " + schema)
			return nil, err
		}

		for rows.Next() {
			tableName, attributes := p.GetTableNameAndAttributes(rows, db)
			if attributes == nil && tableName == "" {
				// skip if error getting attributes
				continue
			}

			attributes = append(attributes, allTablesWithColumns[tableName]...)

			storageUnits = append(storageUnits, engine.StorageUnit{
				Name:       tableName,
				Attributes: attributes,
			})
		}
		return storageUnits, nil
	})
}

func (p *GormPlugin) GetTableSchema(db *gorm.DB, schema string) (map[string][]engine.Record, error) {
	var result []struct {
		TableName  string `gorm:"column:TABLE_NAME"`
		ColumnName string `gorm:"column:COLUMN_NAME"`
		DataType   string `gorm:"column:DATA_TYPE"`
	}

	query := p.GetSchemaTableQuery()

	if err := db.Raw(query, schema).Scan(&result).Error; err != nil {
		log.Logger.WithError(err).Error("Failed to execute schema table query for schema: " + schema)
		return nil, err
	}

	tableColumnsMap := make(map[string][]engine.Record)
	for _, row := range result {
		tableColumnsMap[row.TableName] = append(tableColumnsMap[row.TableName], engine.Record{Key: row.ColumnName, Value: row.DataType})
	}

	return tableColumnsMap, nil
}

func (p *GormPlugin) GetAllSchemas(config *engine.PluginConfig) ([]string, error) {
	return plugins.WithConnection(config, p.DB, func(db *gorm.DB) ([]string, error) {
		var schemas []any
		query := p.GetAllSchemasQuery()
		if err := db.Raw(query).Scan(&schemas).Error; err != nil {
			log.Logger.WithError(err).Error("Failed to execute get all schemas query")
			return nil, err
		}
		schemaNames := []string{}
		for _, schema := range schemas {
			schemaNames = append(schemaNames, fmt.Sprintf("%s", schema))
		}
		return schemaNames, nil
	})
}

func (p *GormPlugin) GetRows(config *engine.PluginConfig, schema string, storageUnit string, where *model.WhereCondition, sort []*model.SortCondition, pageSize, pageOffset int) (*engine.GetRowsResult, error) {
	return plugins.WithConnection(config, p.DB, func(db *gorm.DB) (*engine.GetRowsResult, error) {
		// Handle SQLite separately due to text conversion of date/time columns
		if p.Type == engine.DatabaseType_Sqlite3 {
			return p.getSQLiteRows(db, schema, storageUnit, sort, pageSize, pageOffset)
		}

		// General case for other databases
		return p.getGenericRows(db, schema, storageUnit, where, sort, pageSize, pageOffset)
	})
}

func (p *GormPlugin) GetColumnsForTable(config *engine.PluginConfig, schema string, storageUnit string) ([]engine.Column, error) {
	return plugins.WithConnection(config, p.DB, func(db *gorm.DB) ([]engine.Column, error) {
		columnTypes, err := p.GetColumnTypes(db, schema, storageUnit)
		if err != nil {
			log.Logger.WithError(err).Error(fmt.Sprintf("Failed to get column types for table %s.%s", schema, storageUnit))
			return nil, err
		}

		columns := make([]engine.Column, 0, len(columnTypes))
		for columnName, columnType := range columnTypes {
			columns = append(columns, engine.Column{
				Name: columnName,
				Type: columnType,
			})
		}

		return columns, nil
	})
}

func (p *GormPlugin) getSQLiteRows(db *gorm.DB, schema, storageUnit string, sort []*model.SortCondition, pageSize, pageOffset int) (*engine.GetRowsResult, error) {
	columnInfo, err := p.GetColumnTypes(db, schema, storageUnit)
	if err != nil {
		log.Logger.WithError(err).Error(fmt.Sprintf("Failed to get column types for table %s.%s", schema, storageUnit))
		return nil, err
	}

	selects := make([]string, 0, len(columnInfo))
	for col, colType := range columnInfo {
		colType = strings.ToUpper(colType)
		if colType == "DATE" || colType == "DATETIME" || colType == "TIMESTAMP" {
			selects = append(selects, fmt.Sprintf("CAST(%s AS TEXT) AS %s", col, col))
		} else {
			selects = append(selects, col)
		}
	}

	query := fmt.Sprintf(
		"SELECT %s FROM %s",
		strings.Join(selects, ", "),
		p.EscapeIdentifier(storageUnit),
	)

	// Add ORDER BY clause if sort conditions are provided
	if len(sort) > 0 {
		orderByParts := []string{}
		for _, s := range sort {
			direction := "ASC"
			if s.Direction == model.SortDirectionDesc {
				direction = "DESC"
			}
			orderByParts = append(orderByParts, fmt.Sprintf("%s %s", p.EscapeIdentifier(s.Column), direction))
		}
		query += " ORDER BY " + strings.Join(orderByParts, ", ")
	}

	query += fmt.Sprintf(" LIMIT %d OFFSET %d", pageSize, pageOffset)

	rows, err := db.Raw(query).Rows()
	if err != nil {
		log.Logger.WithError(err).Error(fmt.Sprintf("Failed to execute SQLite rows query for table %s.%s", schema, storageUnit))
		return nil, err
	}
	defer rows.Close()

	return p.ConvertRawToRows(rows)
}

func (p *GormPlugin) getGenericRows(db *gorm.DB, schema, storageUnit string, where *model.WhereCondition, sort []*model.SortCondition, pageSize, pageOffset int) (*engine.GetRowsResult, error) {
	var columnTypes map[string]string
	if where != nil {
		columnTypes, _ = p.GetColumnTypes(db, schema, storageUnit)
	}

	schema = p.EscapeIdentifier(schema)
	storageUnit = p.EscapeIdentifier(storageUnit)
	fullTable := p.FormTableName(schema, storageUnit)

	query := db.Table(fullTable)
	query, err := p.applyWhereConditions(query, where, columnTypes)
	if err != nil {
		log.Logger.WithError(err).Error(fmt.Sprintf("Failed to apply where conditions for table %s.%s", schema, storageUnit))
		return nil, err
	}

	// Apply sorting conditions if provided
	if len(sort) > 0 {
		for _, s := range sort {
			column := p.EscapeIdentifier(s.Column)
			direction := "ASC"
			if s.Direction == model.SortDirectionDesc {
				direction = "DESC"
			}
			query = query.Order(fmt.Sprintf("%s %s", column, direction))
		}
	} else {
		// Apply custom ordering if specified by the database plugin
		if orderBy := p.GormPluginFunctions.GetRowsOrderBy(db, schema, storageUnit); orderBy != "" {
			query = query.Order(orderBy)
		}
	}

	query = query.Limit(pageSize).Offset(pageOffset)

	rows, err := query.Rows()
	if err != nil {
		log.Logger.WithError(err).Error(fmt.Sprintf("Failed to execute generic rows query for table %s.%s", schema, storageUnit))
		return nil, err
	}
	defer rows.Close()

	result, err := p.GormPluginFunctions.ConvertRawToRows(rows)
	if err != nil {
		log.Logger.WithError(err).Error(fmt.Sprintf("Failed to convert raw rows for table %s.%s", schema, storageUnit))
		return nil, err
	}

	// Fix any missing column type metadata
	for i, col := range result.Columns {
		if _, err := strconv.Atoi(col.Type); err == nil {
			result.Columns[i].Type = p.FindMissingDataType(db, col.Type)
		}
	}

	return result, nil
}

func (p *GormPlugin) applyWhereConditions(query *gorm.DB, condition *model.WhereCondition, columnTypes map[string]string) (*gorm.DB, error) {
	if condition == nil {
		return query, nil
	}

	switch condition.Type {
	case model.WhereConditionTypeAtomic:
		if condition.Atomic != nil {
			// Use actual column type from database if available
			columnType := condition.Atomic.ColumnType
			if columnTypes != nil {
				if dbType, exists := columnTypes[condition.Atomic.Key]; exists && dbType != "" {
					columnType = dbType
				}
			}

			value, err := p.GormPluginFunctions.ConvertStringValue(condition.Atomic.Value, columnType)
			if err != nil {
				log.Logger.WithError(err).Error(fmt.Sprintf("Failed to convert string value '%s' for column type '%s'", condition.Atomic.Value, columnType))
				return nil, err
			}
			operator, ok := p.GetSupportedOperators()[condition.Atomic.Operator]
			if !ok {
				return nil, fmt.Errorf("invalid SQL operator: %s", condition.Atomic.Operator)
			}
			query = query.Where(fmt.Sprintf("%s %s ?", p.EscapeIdentifier(condition.Atomic.Key), operator), value)
		}

	case model.WhereConditionTypeAnd:
		if condition.And != nil {
			for _, child := range condition.And.Children {
				var err error
				query, err = p.applyWhereConditions(query, child, columnTypes)
				if err != nil {
					log.Logger.WithError(err).Error("Failed to apply AND where condition")
					return nil, err
				}
			}
		}

	case model.WhereConditionTypeOr:
		if condition.Or != nil {
			orQueries := query
			for _, child := range condition.Or.Children {
				childQuery, err := p.applyWhereConditions(query, child, columnTypes)
				if err != nil {
					log.Logger.WithError(err).Error("Failed to apply OR where condition")
					return nil, err
				}
				orQueries = orQueries.Or(childQuery)
			}
			query = orQueries
		}
	}

	return query, nil
}

func (p *GormPlugin) ConvertRawToRows(rows *sql.Rows) (*engine.GetRowsResult, error) {
	columns, err := rows.Columns()
	if err != nil {
		log.Logger.WithError(err).Error("Failed to get column names from result set")
		return nil, err
	}

	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		log.Logger.WithError(err).Error("Failed to get column types from result set")
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

	// todo: might have to extract some of this stuff into db specific functions
	// Build columns with type information
	for _, col := range columns {
		if colType, exists := typeMap[col]; exists {
			colTypeName := colType.DatabaseTypeName()
			if p.Type == engine.DatabaseType_Postgres && strings.HasPrefix(colTypeName, "_") {
				colTypeName = strings.Replace(colTypeName, "_", "[]", 1)
			}
			if customName := p.GormPluginFunctions.GetCustomColumnTypeName(col, colTypeName); customName != "" {
				colTypeName = customName
			}
			result.Columns = append(result.Columns, engine.Column{Name: col, Type: colTypeName})
		}
	}

	for rows.Next() {
		columnPointers := make([]interface{}, len(columns))
		row := make([]string, len(columns))

		for i, col := range columns {
			colType := typeMap[col]
			typeName := colType.DatabaseTypeName()

			// Check if the plugin wants to handle this column type
			if p.GormPluginFunctions.ShouldHandleColumnType(typeName) {
				columnPointers[i] = p.GormPluginFunctions.GetColumnScanner(typeName)
			} else {
				// Default handling
				switch typeName {
				case "VARBINARY", "BINARY", "IMAGE", "BYTEA", "BLOB", "HIERARCHYID",
					"GEOMETRY", "POINT", "LINESTRING", "POLYGON", "GEOGRAPHY",
					"MULTIPOINT", "MULTILINESTRING", "MULTIPOLYGON":
					columnPointers[i] = new(sql.RawBytes)
				default:
					columnPointers[i] = new(sql.NullString)
				}
			}
		}

		if err := rows.Scan(columnPointers...); err != nil {
			log.Logger.WithError(err).Error("Failed to scan row data")
			return nil, err
		}

		for i, colPtr := range columnPointers {
			colType := typeMap[columns[i]]
			typeName := colType.DatabaseTypeName()

			// Check if the plugin wants to handle this column type
			if p.GormPluginFunctions.ShouldHandleColumnType(typeName) {
				value, err := p.GormPluginFunctions.FormatColumnValue(typeName, colPtr)
				if err != nil {
					row[i] = "ERROR: " + err.Error()
				} else {
					row[i] = value
				}
			} else {
				// Default handling
				switch typeName {
				case "VARBINARY", "BINARY", "IMAGE", "BYTEA", "BLOB":
					rawBytes := colPtr.(*sql.RawBytes)
					if rawBytes == nil || len(*rawBytes) == 0 {
						row[i] = ""
					} else {
						row[i] = "0x" + hex.EncodeToString(*rawBytes)
					}
				// todo: geometry types are not yet ready for production. please do not use them.
				case "GEOMETRY", "POINT", "LINESTRING", "POLYGON", "GEOGRAPHY",
					"MULTIPOINT", "MULTILINESTRING", "MULTIPOLYGON":
					rawBytes := colPtr.(*sql.RawBytes)
					if rawBytes == nil || len(*rawBytes) == 0 {
						row[i] = ""
					} else {
						// Try custom geometry formatting first
						if formatted := p.GormPluginFunctions.FormatGeometryValue(*rawBytes, typeName); formatted != "" {
							row[i] = formatted
						} else {
							// Fallback to hex
							row[i] = "0x" + hex.EncodeToString(*rawBytes)
						}
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
		}

		result.Rows = append(result.Rows, row)
	}

	return result, nil
}

// todo: extract this into a gormplugin default if needed for other plugins
func (p *GormPlugin) FindMissingDataType(db *gorm.DB, columnType string) string {
	if p.Type == engine.DatabaseType_Postgres {
		var typname string
		if err := db.Raw("SELECT typname FROM pg_type WHERE oid = ?", columnType).Scan(&typname).Error; err != nil {
			log.Logger.WithError(err).Error(fmt.Sprintf("Failed to find PostgreSQL type name for OID: %s", columnType))
			typname = columnType
		}
		return strings.ToUpper(typname)
	}
	return columnType
}

// ShouldQuoteIdentifiers returns whether column identifiers should be quoted in queries
// Default implementation returns false; specific database plugins can override
func (p *GormPlugin) ShouldQuoteIdentifiers() bool {
	return false
}

// GetRowsOrderBy returns the ORDER BY clause for pagination queries
// Default implementation returns empty string (no ordering)
func (p *GormPlugin) GetRowsOrderBy(db *gorm.DB, schema string, storageUnit string) string {
	return ""
}

// ShouldHandleColumnType returns false by default
func (p *GormPlugin) ShouldHandleColumnType(columnType string) bool {
	return false
}

// GetColumnScanner returns nil by default
func (p *GormPlugin) GetColumnScanner(columnType string) interface{} {
	return nil
}

// FormatColumnValue returns empty string by default
func (p *GormPlugin) FormatColumnValue(columnType string, scanner interface{}) (string, error) {
	return "", nil
}

// GetCustomColumnTypeName returns empty string by default
func (p *GormPlugin) GetCustomColumnTypeName(columnName string, defaultTypeName string) string {
	return ""
}

// IsGeometryType returns true for common geometry type names
func (p *GormPlugin) IsGeometryType(columnType string) bool {
	switch columnType {
	case "GEOMETRY", "POINT", "LINESTRING", "POLYGON", "GEOGRAPHY",
		"MULTIPOINT", "MULTILINESTRING", "MULTIPOLYGON", "GEOMETRYCOLLECTION":
		return true
	default:
		return false
	}
}

// FormatGeometryValue returns empty string by default (use hex formatting)
func (p *GormPlugin) FormatGeometryValue(rawBytes []byte, columnType string) string {
	return ""
}

// HandleCustomDataType returns false by default (no custom handling)
func (p *GormPlugin) HandleCustomDataType(value string, columnType string, isNullable bool) (interface{}, bool, error) {
	return nil, false, nil
}

// WithTransaction executes the given operation within a database transaction
func (p *GormPlugin) WithTransaction(config *engine.PluginConfig, operation func(tx any) error) error {
	_, err := plugins.WithConnection(config, p.DB, func(db *gorm.DB) (bool, error) {
		// Begin transaction
		tx := db.Begin()
		if tx.Error != nil {
			return false, fmt.Errorf("failed to begin transaction: %w", tx.Error)
		}

		// Execute the operation
		if err := operation(tx); err != nil {
			// Rollback on error
			tx.Rollback()
			return false, err
		}

		// Commit transaction
		if err := tx.Commit().Error; err != nil {
			return false, fmt.Errorf("failed to commit transaction: %w", err)
		}

		return true, nil
	})

	return err
}

// ExecuteInTransaction wraps common database operations in a transaction
func (p *GormPlugin) ExecuteInTransaction(config *engine.PluginConfig, operations func(tx *gorm.DB) error) error {
	return p.WithTransaction(config, func(txInterface any) error {
		tx, ok := txInterface.(*gorm.DB)
		if !ok {
			return fmt.Errorf("invalid transaction type")
		}
		return operations(tx)
	})
}

// AddRowInTx adds a row using an existing transaction
func (p *GormPlugin) AddRowInTx(tx *gorm.DB, schema string, storageUnit string, values []engine.Record) error {
	return p.addRowWithDB(tx, schema, storageUnit, values)
}

// ClearTableDataInTx clears table data using an existing transaction
func (p *GormPlugin) ClearTableDataInTx(tx *gorm.DB, schema string, storageUnit string) error {
	return p.clearTableDataWithDB(tx, schema, storageUnit)
}
