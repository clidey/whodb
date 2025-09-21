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
	errorHandler *ErrorHandler
}

// InitPlugin initializes the plugin with an error handler
func (p *GormPlugin) InitPlugin() {
	if p.errorHandler == nil {
		p.errorHandler = NewErrorHandler(p)
	}
}

// CreateSQLBuilder creates a SQL builder instance - default implementation
// Can be overridden by specific database plugins (e.g., MySQL)
func (p *GormPlugin) CreateSQLBuilder(db *gorm.DB) SQLBuilderInterface {
	return NewSQLBuilder(db, p)
}

type GormPluginFunctions interface {
	// these below are meant to be generic-ish implementations by the base gorm plugin
	ParseConnectionConfig(config *engine.PluginConfig) (*ConnectionInput, error)

	ConvertStringValue(value, columnType string) (interface{}, error)
	ConvertRawToRows(raw *sql.Rows) (*engine.GetRowsResult, error)
	ConvertRecordValuesToMap(values []engine.Record) (map[string]interface{}, error)

	// CreateSQLBuilder creates a SQL builder instance - can be overridden by specific plugins
	CreateSQLBuilder(db *gorm.DB) SQLBuilderInterface

	// these below are meant to be implemented by the specific database plugins
	DB(config *engine.PluginConfig) (*gorm.DB, error)
	GetSupportedColumnDataTypes() mapset.Set[string]
	GetPlaceholder(index int) string

	GetTableInfoQuery() string
	GetSchemaTableQuery() string
	GetPrimaryKeyColQuery() string
	GetColTypeQuery() string
	GetAllSchemasQuery() string
	GetCreateTableQuery(db *gorm.DB, schema string, storageUnit string, columns []engine.Record) string

	FormTableName(schema string, storageUnit string) string
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

	// GetPrimaryKeyColumns returns the primary key columns for a table
	GetPrimaryKeyColumns(db *gorm.DB, schema string, tableName string) ([]string, error)

	// GetDatabaseType returns the database type
	GetDatabaseType() engine.DatabaseType

	// GetColumnConstraints retrieves column constraints for a table
	GetColumnConstraints(config *engine.PluginConfig, schema string, storageUnit string) (map[string]map[string]any, error)
}

func (p *GormPlugin) GetStorageUnits(config *engine.PluginConfig, schema string) ([]engine.StorageUnit, error) {
	return plugins.WithConnection(config, p.DB, func(db *gorm.DB) ([]engine.StorageUnit, error) {
		var storageUnits []engine.StorageUnit
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

	// Most plugins use information_schema.columns, but query structure varies
	// Keep using Raw query since it's defined by each plugin
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
			// Use error handler for consistent error handling
			return nil, p.errorHandler.HandleError(err, "GetAllSchemas", map[string]any{
				"query": query,
			})
		}
		var schemaNames []string
		for _, schema := range schemas {
			schemaNames = append(schemaNames, fmt.Sprintf("%s", schema))
		}
		return schemaNames, nil
	})
}

func (p *GormPlugin) GetRows(config *engine.PluginConfig, schema string, storageUnit string, where *model.WhereCondition, sort []*model.SortCondition, pageSize, pageOffset int) (*engine.GetRowsResult, error) {
	return plugins.WithConnection(config, p.DB, func(db *gorm.DB) (*engine.GetRowsResult, error) {
		// Use generic implementation; database-specific behavior should be handled in each plugin
		return p.getGenericRows(db, schema, storageUnit, where, sort, pageSize, pageOffset)
	})
}

func (p *GormPlugin) GetColumnsForTable(config *engine.PluginConfig, schema string, storageUnit string) ([]engine.Column, error) {
	return plugins.WithConnection(config, p.DB, func(db *gorm.DB) ([]engine.Column, error) {
		migrator := NewMigratorHelper(db, p)

		// Build full table name for Migrator
		var fullTableName string
		if schema != "" && p.Type != engine.DatabaseType_Sqlite3 {
			fullTableName = schema + "." + storageUnit
		} else {
			fullTableName = storageUnit
		}

		// Get ordered columns
		columns, err := migrator.GetOrderedColumns(fullTableName)
		if err != nil {
			log.Logger.WithError(err).Error(fmt.Sprintf("Failed to get columns for table %s.%s", schema, storageUnit))
			return nil, err
		}

		return columns, nil
	})
}

// SQLite-specific row retrieval is implemented in the sqlite3 plugin override.

func (p *GormPlugin) getGenericRows(db *gorm.DB, schema, storageUnit string, where *model.WhereCondition, sort []*model.SortCondition, pageSize, pageOffset int) (*engine.GetRowsResult, error) {
	var columnTypes map[string]string
	if where != nil {
		columnTypes, _ = p.GetColumnTypes(db, schema, storageUnit)
	}

	// Use SQL builder for table name construction
	builder := p.GormPluginFunctions.CreateSQLBuilder(db)
	fullTable := builder.BuildFullTableName(schema, storageUnit)

	query := db.Table(fullTable)
	query, err := p.ApplyWhereConditions(query, where, columnTypes)
	if err != nil {
		log.Logger.WithError(err).Error(fmt.Sprintf("Failed to apply where conditions for table %s.%s", schema, storageUnit))
		return nil, err
	}

	// Apply sorting conditions if provided
	if len(sort) > 0 {
		// Convert to Sort type for builder
		sortList := make([]plugins.Sort, len(sort))
		for i, s := range sort {
			sortList[i] = plugins.Sort{
				Column:    s.Column,
				Direction: plugins.Down,
			}
			if s.Direction == model.SortDirectionAsc {
				sortList[i].Direction = plugins.Up
			}
		}
		query = builder.BuildOrderBy(query, sortList)
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

func (p *GormPlugin) ApplyWhereConditions(query *gorm.DB, condition *model.WhereCondition, columnTypes map[string]string) (*gorm.DB, error) {
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

			operator, ok := p.GetSupportedOperators()[condition.Atomic.Operator]
			if !ok {
				return nil, fmt.Errorf("invalid SQL operator: %s", condition.Atomic.Operator)
			}

			builder := p.GormPluginFunctions.CreateSQLBuilder(query.Session(&gorm.Session{NewDB: true}))
			col := builder.QuoteIdentifier(condition.Atomic.Key)

			// Handle operators that don't take values
			switch operator {
			case "IS NULL", "IS NOT NULL":
				query = query.Where(col + " " + operator)
				return query, nil
			}

			// Convert value for typed comparison
			// Special handling for operators with multiple values
			switch operator {
			case "IN", "NOT IN":
				// Expect comma-separated list; split and convert each
				raw := strings.TrimSpace(condition.Atomic.Value)
				if raw == "" {
					// empty IN list should return no rows; use a false predicate
					query = query.Where("1 = 0")
					return query, nil
				}
				parts := strings.Split(raw, ",")
				vals := make([]interface{}, 0, len(parts))
				for _, part := range parts {
					v := strings.TrimSpace(part)
					cv, err := p.GormPluginFunctions.ConvertStringValue(v, columnType)
					if err != nil {
						log.Logger.WithError(err).Error(fmt.Sprintf("Failed to convert IN value '%s' for column type '%s'", v, columnType))
						return nil, err
					}
					vals = append(vals, cv)
				}
				query = query.Where(col+" "+operator+" ?", vals)
				return query, nil

			case "BETWEEN", "NOT BETWEEN":
				// Expect two values separated by comma: min,max
				raw := strings.TrimSpace(condition.Atomic.Value)
				parts := strings.Split(raw, ",")
				if len(parts) != 2 {
					return nil, fmt.Errorf("invalid BETWEEN value; expected 'min,max'")
				}
				v1, err1 := p.GormPluginFunctions.ConvertStringValue(strings.TrimSpace(parts[0]), columnType)
				if err1 != nil {
					log.Logger.WithError(err1).Error("Failed to convert BETWEEN min value")
					return nil, err1
				}
				v2, err2 := p.GormPluginFunctions.ConvertStringValue(strings.TrimSpace(parts[1]), columnType)
				if err2 != nil {
					log.Logger.WithError(err2).Error("Failed to convert BETWEEN max value")
					return nil, err2
				}
				if operator == "BETWEEN" {
					query = query.Where(col+" BETWEEN ? AND ?", v1, v2)
				} else {
					query = query.Where(col+" NOT BETWEEN ? AND ?", v1, v2)
				}
				return query, nil
			default:
				// Single value operators (=, <, >, LIKE, etc.)
				value, err := p.GormPluginFunctions.ConvertStringValue(condition.Atomic.Value, columnType)
				if err != nil {
					log.Logger.WithError(err).Error(fmt.Sprintf("Failed to convert string value '%s' for column type '%s'", condition.Atomic.Value, columnType))
					return nil, err
				}
				query = query.Where(col+" "+operator+" ?", value)
			}
		}

	case model.WhereConditionTypeAnd:
		if condition.And != nil {
			for _, child := range condition.And.Children {
				var err error
				query, err = p.ApplyWhereConditions(query, child, columnTypes)
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
				childQuery, err := p.ApplyWhereConditions(query, child, columnTypes)
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
			// TODO: BIG EDGE CASE - PostgreSQL array types start with underscore
			// This should be handled in the PostgreSQL plugin's GetCustomColumnTypeName
			/*
				if p.Type == engine.DatabaseType_Postgres && strings.HasPrefix(colTypeName, "_") {
					colTypeName = strings.Replace(colTypeName, "_", "[]", 1)
				}
			*/
			// Keep for now until PostgreSQL plugin properly handles this
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
		if err := db.Table("pg_type").
			Select("typname").
			Where("oid = ?", columnType).
			Scan(&typname).Error; err != nil {
			log.Logger.WithError(err).Error(fmt.Sprintf("Failed to find PostgreSQL type name for OID: %s", columnType))
			typname = columnType
		}
		return strings.ToUpper(typname)
	}
	return columnType
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

// GetDatabaseType returns the database type
func (p *GormPlugin) GetDatabaseType() engine.DatabaseType {
	return p.Type
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
