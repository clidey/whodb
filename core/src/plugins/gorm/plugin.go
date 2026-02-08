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
	"encoding/hex"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
	"github.com/clidey/whodb/core/src/plugins"
	"github.com/dromara/carbon/v2"
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

	ConvertStringValue(value, columnType string) (any, error)
	ConvertRawToRows(raw *sql.Rows) (*engine.GetRowsResult, error)
	ConvertRecordValuesToMap(values []engine.Record) (map[string]any, error)

	// CreateSQLBuilder creates a SQL builder instance - can be overridden by specific plugins
	CreateSQLBuilder(db *gorm.DB) SQLBuilderInterface

	// these below are meant to be implemented by the specific database plugins
	DB(config *engine.PluginConfig) (*gorm.DB, error)
	GetPlaceholder(index int) string

	GetTableInfoQuery() string
	GetStorageUnitExistsQuery() string
	GetPrimaryKeyColQuery() string
	GetAllSchemasQuery() string
	GetCreateTableQuery(db *gorm.DB, schema string, storageUnit string, columns []engine.Record) string

	FormTableName(schema string, storageUnit string) string

	GetSupportedOperators() map[string]string

	GetGraphQueryDB(db *gorm.DB, schema string) *gorm.DB
	GetTableNameAndAttributes(rows *sql.Rows) (string, []engine.Record)

	// GetRowsOrderBy returns the ORDER BY clause for pagination queries
	GetRowsOrderBy(db *gorm.DB, schema string, storageUnit string) string

	// ShouldHandleColumnType returns true if the plugin wants to handle a specific column type
	ShouldHandleColumnType(columnType string) bool

	// GetColumnScanner returns a scanner for a specific column type
	// This is called when ShouldHandleColumnType returns true
	GetColumnScanner(columnType string) any

	// FormatColumnValue formats a scanned value for a specific column type
	// This is called when ShouldHandleColumnType returns true
	FormatColumnValue(columnType string, scanner any) (string, error)

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
	HandleCustomDataType(value string, columnType string, isNullable bool) (any, bool, error)

	// GetPrimaryKeyColumns returns the primary key columns for a table
	GetPrimaryKeyColumns(db *gorm.DB, schema string, tableName string) ([]string, error)

	// GetDatabaseType returns the database type
	GetDatabaseType() engine.DatabaseType

	// GetColumnConstraints retrieves column constraints for a table
	GetColumnConstraints(config *engine.PluginConfig, schema string, storageUnit string) (map[string]map[string]any, error)

	// NormalizeType converts a type alias to its canonical form for this database.
	// For example, PostgreSQL: "INT4" -> "INTEGER", "VARCHAR" -> "CHARACTER VARYING"
	// Returns the input unchanged if no mapping exists.
	NormalizeType(typeName string) string

	// GetColumnTypes returns a map of column names to their types for a table.
	// Used for type conversion during CRUD operations.
	GetColumnTypes(db *gorm.DB, schema, tableName string) (map[string]string, error)

	// GetLastInsertID returns the most recently auto-generated ID after an INSERT.
	// This is used by the mock data generator to track PKs for FK references.
	// Returns 0 if the database doesn't support this or no ID was generated.
	GetLastInsertID(db *gorm.DB) (int64, error)

	// GetMaxBulkInsertParameters returns the maximum number of parameters supported
	// for bulk insert operations. Used to calculate appropriate batch sizes.
	// Default: 65535 (PostgreSQL/MySQL limit). Override for databases with lower limits.
	GetMaxBulkInsertParameters() int
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

		for rows.Next() {
			tableName, attributes := p.GetTableNameAndAttributes(rows)
			if attributes == nil && tableName == "" {
				continue
			}
			storageUnits = append(storageUnits, engine.StorageUnit{
				Name:       tableName,
				Attributes: attributes,
			})
		}

		return storageUnits, nil
	})
}

func (p *GormPlugin) StorageUnitExists(config *engine.PluginConfig, schema string, storageUnit string) (bool, error) {
	return plugins.WithConnection(config, p.DB, func(db *gorm.DB) (bool, error) {
		var exists bool
		err := db.Raw(p.GetStorageUnitExistsQuery(), schema, storageUnit).Scan(&exists).Error
		if err != nil {
			return false, err
		}
		return exists, nil
	})
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
		schemaNames := make([]string, len(schemas))
		for i, schema := range schemas {
			switch v := schema.(type) {
			case string:
				schemaNames[i] = v
			case []byte:
				schemaNames[i] = string(v)
			default:
				schemaNames[i] = fmt.Sprintf("%v", v)
			}
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

func (p *GormPlugin) GetRowCount(config *engine.PluginConfig, schema string, storageUnit string, where *model.WhereCondition) (int64, error) {
	return plugins.WithConnection(config, p.DB, func(db *gorm.DB) (int64, error) {
		var columnTypes map[string]string
		if where != nil {
			columnTypes, _ = p.GormPluginFunctions.GetColumnTypes(db, schema, storageUnit)
		}

		builder := p.GormPluginFunctions.CreateSQLBuilder(db)
		fullTable := builder.BuildFullTableName(schema, storageUnit)

		// codeql[go/sql-injection]: table name validated by StorageUnitExists before reaching this code
		query := db.Table(fullTable)
		query, err := p.ApplyWhereConditions(query, where, columnTypes)
		if err != nil {
			return 0, err
		}

		var count int64
		if err := query.Count(&count).Error; err != nil {
			return 0, err
		}
		return count, nil
	})
}

func (p *GormPlugin) GetColumnsForTable(config *engine.PluginConfig, schema string, storageUnit string) ([]engine.Column, error) {
	return plugins.WithConnection(config, p.DB, func(db *gorm.DB) ([]engine.Column, error) {
		migrator := NewMigratorHelper(db, p.GormPluginFunctions)
		var fullTableName string
		if schema != "" && p.Type != engine.DatabaseType_Sqlite3 {
			fullTableName = schema + "." + storageUnit
		} else {
			fullTableName = storageUnit
		}

		columns, err := migrator.GetOrderedColumns(fullTableName)
		if err != nil {
			log.Logger.WithError(err).Error(fmt.Sprintf("Failed to get columns for table %s.%s", schema, storageUnit))
			return nil, err
		}

		fkRelationships, err := p.GetForeignKeyRelationships(config, schema, storageUnit)
		if err != nil {
			log.Logger.WithError(err).Warn(fmt.Sprintf("Failed to get foreign key relationships for table %s.%s", schema, storageUnit))
			fkRelationships = make(map[string]*engine.ForeignKeyRelationship)
		}

		primaryKeys, err := p.GetPrimaryKeyColumns(db, schema, storageUnit)
		if err != nil {
			log.Logger.WithError(err).Warn(fmt.Sprintf("Failed to get primary keys for table %s.%s", schema, storageUnit))
			primaryKeys = []string{}
		}

		// Enrich columns with primary key and foreign key information
		// Note: IsAutoIncrement is already set by GORM's native detection in GetOrderedColumns
		for i := range columns {
			// Check if column is a primary key
			columns[i].IsPrimary = slices.Contains(primaryKeys, columns[i].Name)

			// Check if column is a foreign key
			if fk, exists := fkRelationships[columns[i].Name]; exists {
				columns[i].IsForeignKey = true
				columns[i].ReferencedTable = &fk.ReferencedTable
				columns[i].ReferencedColumn = &fk.ReferencedColumn
			}
		}

		return columns, nil
	})
}

// SQLite-specific row retrieval is implemented in the sqlite3 plugin override.
func (p *GormPlugin) getGenericRows(db *gorm.DB, schema, storageUnit string, where *model.WhereCondition, sort []*model.SortCondition, pageSize, pageOffset int) (*engine.GetRowsResult, error) {
	var columnTypes map[string]string
	if where != nil {
		columnTypes, _ = p.GormPluginFunctions.GetColumnTypes(db, schema, storageUnit)
	}

	builder := p.GormPluginFunctions.CreateSQLBuilder(db)
	fullTable := builder.BuildFullTableName(schema, storageUnit)

	// Parallel count query improves performance for large tables
	var totalCount int64
	countDone := make(chan error, 1)
	go func() {
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

	// codeql[go/sql-injection]: table name validated by StorageUnitExists before reaching this code
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

	// Only apply pagination if pageSize > 0
	// pageSize <= 0 means fetch all rows
	if pageSize > 0 {
		query = query.Limit(pageSize).Offset(pageOffset)
	}

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

	// Wait for count query to complete and set TotalCount
	if countErr := <-countDone; countErr != nil {
		log.Logger.WithError(countErr).Warn(fmt.Sprintf("Failed to get row count for table %s.%s", schema, storageUnit))
		// Don't fail the whole operation if count fails
	} else {
		result.TotalCount = totalCount
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
				vals := make([]any, 0, len(parts))
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

	// TODO: Extract database-specific type handling into individual plugins
	for _, col := range columns {
		if colType, exists := typeMap[col]; exists {
			colTypeName := colType.DatabaseTypeName()
			// TODO: BIG EDGE CASE - PostgreSQL array types use underscore prefix (e.g., _int4 for int[])
			// Move to PostgreSQL plugin's GetCustomColumnTypeName once all edge cases are tested
			if p.Type == engine.DatabaseType_Postgres && strings.HasPrefix(colTypeName, "_") {
				colTypeName = strings.Replace(colTypeName, "_", "[]", 1)
			}
			if customName := p.GormPluginFunctions.GetCustomColumnTypeName(col, colTypeName); customName != "" {
				colTypeName = customName
			}

			column := engine.Column{Name: col, Type: colTypeName}
			baseTypeName := strings.ToUpper(colType.DatabaseTypeName())

			// Only extract length for types where it's user-specifiable
			if typesWithLength[baseTypeName] {
				if length, ok := colType.Length(); ok && length > 0 {
					l := int(length)
					column.Length = &l
					// Include length in type name for display
					colTypeName = fmt.Sprintf("%s(%d)", colTypeName, length)
					column.Type = colTypeName
				}
			}

			// Only extract precision/scale for decimal-like types
			if typesWithPrecision[baseTypeName] {
				if precision, scale, ok := colType.DecimalSize(); ok && precision > 0 {
					prec := int(precision)
					column.Precision = &prec
					if scale > 0 {
						s := int(scale)
						column.Scale = &s
						colTypeName = fmt.Sprintf("%s(%d,%d)", colType.DatabaseTypeName(), precision, scale)
					} else {
						colTypeName = fmt.Sprintf("%s(%d)", colType.DatabaseTypeName(), precision)
					}
					column.Type = colTypeName
				}
			}

			result.Columns = append(result.Columns, column)
		}
	}

	for rows.Next() {
		columnPointers := make([]any, len(columns))
		row := make([]string, len(columns))

		for i, col := range columns {
			colType := typeMap[col]
			typeName := colType.DatabaseTypeName()

			if p.GormPluginFunctions.ShouldHandleColumnType(typeName) {
				columnPointers[i] = p.GormPluginFunctions.GetColumnScanner(typeName)
			} else {
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

			if p.GormPluginFunctions.ShouldHandleColumnType(typeName) {
				value, err := p.GormPluginFunctions.FormatColumnValue(typeName, colPtr)
				if err != nil {
					row[i] = "ERROR: " + err.Error()
				} else {
					row[i] = value
				}
			} else {
				switch typeName {
				case "VARBINARY", "BINARY", "IMAGE", "BYTEA", "BLOB":
					rawBytes := colPtr.(*sql.RawBytes)
					if rawBytes == nil || len(*rawBytes) == 0 {
						row[i] = ""
					} else {
						row[i] = "0x" + hex.EncodeToString(*rawBytes)
					}
				// TODO: Geometry types need more testing before production use
				case "GEOMETRY", "POINT", "LINESTRING", "POLYGON", "GEOGRAPHY",
					"MULTIPOINT", "MULTILINESTRING", "MULTIPOLYGON":
					rawBytes := colPtr.(*sql.RawBytes)
					if rawBytes == nil || len(*rawBytes) == 0 {
						row[i] = ""
					} else if formatted := p.GormPluginFunctions.FormatGeometryValue(*rawBytes, typeName); formatted != "" {
						row[i] = formatted
					} else {
						row[i] = "0x" + hex.EncodeToString(*rawBytes)
					}
				case "TIME":
					// TIME columns are returned as full datetime strings with zero date (e.g., "0001-01-01T12:00:00Z")
					// Extract just the time portion for display
					val := colPtr.(*sql.NullString)
					if val.Valid {
						row[i] = formatTimeOnly(val.String)
					} else {
						row[i] = ""
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

// TODO: Extract into base GormPlugin if needed by other database plugins
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
func (p *GormPlugin) GetColumnScanner(columnType string) any {
	return nil
}

// FormatColumnValue returns empty string by default
func (p *GormPlugin) FormatColumnValue(columnType string, scanner any) (string, error) {
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

// formatTimeOnly extracts just the time portion from a datetime string.
// Database drivers return TIME columns as full datetime with zero date (e.g., "0001-01-01T12:00:00Z").
// This function extracts just the time portion for cleaner display.
func formatTimeOnly(value string) string {
	c := carbon.Parse(value)
	if c.Error != nil || c.IsInvalid() {
		// If carbon can't parse it, return as-is
		return value
	}

	// Check if it has sub-second precision
	if c.Nanosecond() > 0 {
		return c.ToTimeMilliString()
	}
	return c.ToTimeString()
}

// HandleCustomDataType returns false by default (no custom handling)
func (p *GormPlugin) HandleCustomDataType(value string, columnType string, isNullable bool) (any, bool, error) {
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

// GetForeignKeyRelationships returns foreign key relationships for a table (default empty implementation)
func (p *GormPlugin) GetForeignKeyRelationships(config *engine.PluginConfig, schema string, storageUnit string) (map[string]*engine.ForeignKeyRelationship, error) {
	return make(map[string]*engine.ForeignKeyRelationship), nil
}

// QueryForeignKeyRelationships executes a foreign key query and returns the relationships map.
// This is a helper for SQL plugins that query system catalogs for FK information.
// The query must return exactly 3 columns: column_name, referenced_table, referenced_column.
func (p *GormPlugin) QueryForeignKeyRelationships(config *engine.PluginConfig, query string, params ...any) (map[string]*engine.ForeignKeyRelationship, error) {
	return plugins.WithConnection(config, p.DB, func(db *gorm.DB) (map[string]*engine.ForeignKeyRelationship, error) {
		rows, err := db.Raw(query, params...).Rows()
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		relationships := make(map[string]*engine.ForeignKeyRelationship)
		for rows.Next() {
			var columnName, referencedTable, referencedColumn string
			if err := rows.Scan(&columnName, &referencedTable, &referencedColumn); err != nil {
				log.Logger.WithError(err).Error("Failed to scan foreign key relationship")
				continue
			}
			relationships[columnName] = &engine.ForeignKeyRelationship{
				ColumnName:       columnName,
				ReferencedTable:  referencedTable,
				ReferencedColumn: referencedColumn,
			}
		}
		return relationships, nil
	})
}

// QueryComputedColumns executes a query and returns a set of computed column names.
// This is a helper for SQL plugins that query system catalogs for generated/computed columns.
// The query must return exactly 1 column: column_name.
func (p *GormPlugin) QueryComputedColumns(config *engine.PluginConfig, query string, params ...any) (map[string]bool, error) {
	return plugins.WithConnection(config, p.DB, func(db *gorm.DB) (map[string]bool, error) {
		rows, err := db.Raw(query, params...).Rows()
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		computed := make(map[string]bool)
		for rows.Next() {
			var columnName string
			if err := rows.Scan(&columnName); err != nil {
				continue
			}
			computed[columnName] = true
		}
		return computed, nil
	})
}

// NormalizeType returns the type unchanged by default.
// Database plugins should override this to normalize aliases to canonical types.
func (p *GormPlugin) NormalizeType(typeName string) string {
	// Default: strip length and return uppercase base type
	spec := common.ParseTypeSpec(typeName)
	return common.FormatTypeSpec(spec)
}

// GetMaxBulkInsertParameters returns the default limit of 65535 parameters.
// PostgreSQL and MySQL both support this limit. Override for databases with
// lower limits (e.g., SQLite: 999, MSSQL: 2100, Oracle: 1000).
func (p *GormPlugin) GetMaxBulkInsertParameters() int {
	return 65535
}

// GetDatabaseMetadata returns nil by default.
// Database plugins should override this to provide metadata for frontend configuration.
func (p *GormPlugin) GetDatabaseMetadata() *engine.DatabaseMetadata {
	return nil
}
