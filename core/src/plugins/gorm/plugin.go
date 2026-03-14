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
	"slices"
	"strconv"
	"strings"

	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
	"github.com/clidey/whodb/core/src/plugins"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

// FormTableName returns the qualified table name for a given schema and storage unit.
func (p *GormPlugin) FormTableName(schema string, storageUnit string) string {
	if schema == "" {
		return storageUnit
	}
	return schema + "." + storageUnit
}

type GormPluginFunctions interface {
	// these below are meant to be generic-ish implementations by the base gorm plugin
	ParseConnectionConfig(config *engine.PluginConfig) (*ConnectionInput, error)

	ConvertStringValue(value, columnType string, isNullable bool) (any, error)
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

	// GetForeignKeyRelationships returns foreign key relationships for a table
	GetForeignKeyRelationships(config *engine.PluginConfig, schema string, storageUnit string) (map[string]*engine.ForeignKeyRelationship, error)

	// GetDatabaseType returns the database type
	GetDatabaseType() engine.DatabaseType

	// GetColumnConstraints retrieves column constraints for a table
	GetColumnConstraints(config *engine.PluginConfig, schema string, storageUnit string) (map[string]map[string]any, error)

	// NormalizeType converts a type alias to its canonical form for this database.
	// For example, PostgreSQL: "INT4" -> "INTEGER", "VARCHAR" -> "CHARACTER VARYING"
	// Returns the input unchanged if no mapping exists.
	NormalizeType(typeName string) string

	// GetColumnTypes returns a map of column names to their type info (type + nullability).
	// Used for type conversion during CRUD operations.
	GetColumnTypes(db *gorm.DB, schema, tableName string) (map[string]ColumnTypeInfo, error)

	// GetLastInsertID returns the most recently auto-generated ID after an INSERT.
	// This is used by the mock data generator to track PKs for FK references.
	// Returns 0 if the database doesn't support this or no ID was generated.
	GetLastInsertID(db *gorm.DB) (int64, error)

	// GetMaxBulkInsertParameters returns the maximum number of parameters supported
	// for bulk insert operations. Used to calculate appropriate batch sizes.
	// Default: 65535 (PostgreSQL/MySQL limit). Override for databases with lower limits.
	GetMaxBulkInsertParameters() int

	// BuildSkipConflictClause returns an OnConflict clause that skips duplicate rows
	// during append-mode imports. Dialect-specific because Postgres uses DO NOTHING
	// while MySQL needs identity assignments (pk = pk) since GORM can't generate the
	// fallback without schema info when using .Table() with map records.
	BuildSkipConflictClause(pkColumns []string) clause.OnConflict
}

func (p *GormPlugin) GetStorageUnits(config *engine.PluginConfig, schema string) ([]engine.StorageUnit, error) {
	return plugins.WithConnection(config, p.DB, func(db *gorm.DB) ([]engine.StorageUnit, error) {
		var storageUnits []engine.StorageUnit
		rows, err := db.Raw(p.GetTableInfoQuery(), schema).Rows()
		if err != nil {
			log.WithError(err).Error("Failed to execute table info query for schema: " + schema)
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
		var columnTypes map[string]ColumnTypeInfo
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
		fullTableName := p.FormTableName(schema, storageUnit)
		columns, err := migrator.GetOrderedColumns(fullTableName)
		if err != nil {
			log.WithError(err).Error(fmt.Sprintf("Failed to get columns for table %s.%s", schema, storageUnit))
			return nil, err
		}

		fkRelationships, err := p.GormPluginFunctions.GetForeignKeyRelationships(config, schema, storageUnit)
		if err != nil {
			log.WithError(err).Warn(fmt.Sprintf("Failed to get foreign key relationships for table %s.%s", schema, storageUnit))
			fkRelationships = make(map[string]*engine.ForeignKeyRelationship)
		}

		primaryKeys, err := p.GetPrimaryKeyColumns(db, schema, storageUnit)
		if err != nil {
			log.WithError(err).Warn(fmt.Sprintf("Failed to get primary keys for table %s.%s", schema, storageUnit))
			primaryKeys = []string{}
		}

		log.Debugf("[FK DEBUG] GetColumns for %s.%s: fkCount=%d, pkCount=%d, pks=%v", schema, storageUnit, len(fkRelationships), len(primaryKeys), primaryKeys)

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
	var columnTypes map[string]ColumnTypeInfo
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
		log.WithError(err).Error(fmt.Sprintf("Failed to apply where conditions for table %s.%s", schema, storageUnit))
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
		log.WithError(err).Error(fmt.Sprintf("Failed to execute generic rows query for table %s.%s", schema, storageUnit))
		return nil, err
	}
	defer rows.Close()

	result, err := p.GormPluginFunctions.ConvertRawToRows(rows)
	if err != nil {
		log.WithError(err).Error(fmt.Sprintf("Failed to convert raw rows for table %s.%s", schema, storageUnit))
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
		log.WithError(countErr).Warn(fmt.Sprintf("Failed to get row count for table %s.%s", schema, storageUnit))
		// Don't fail the whole operation if count fails
	} else {
		result.TotalCount = totalCount
	}

	return result, nil
}

func (p *GormPlugin) ApplyWhereConditions(query *gorm.DB, condition *model.WhereCondition, columnTypes map[string]ColumnTypeInfo) (*gorm.DB, error) {
	if condition == nil {
		return query, nil
	}

	switch condition.Type {
	case model.WhereConditionTypeAtomic:
		return p.applyAtomicCondition(query, condition.Atomic, columnTypes)
	case model.WhereConditionTypeAnd:
		return p.applyAndConditions(query, condition.And, columnTypes)
	case model.WhereConditionTypeOr:
		return p.applyOrConditions(query, condition.Or, columnTypes)
	}

	return query, nil
}

// applyAtomicCondition handles a single atomic WHERE condition (e.g., column = value).
func (p *GormPlugin) applyAtomicCondition(query *gorm.DB, atomic *model.AtomicWhereCondition, columnTypes map[string]ColumnTypeInfo) (*gorm.DB, error) {
	if atomic == nil {
		return query, nil
	}

	// Use actual column type from database if available
	columnType := atomic.ColumnType
	isNullable := false
	if columnTypes != nil {
		if colInfo, exists := columnTypes[atomic.Key]; exists && colInfo.Type != "" {
			columnType = colInfo.Type
			isNullable = colInfo.IsNullable
		}
	}

	operator, ok := p.GetSupportedOperators()[atomic.Operator]
	if !ok {
		return nil, fmt.Errorf("invalid SQL operator: %s", atomic.Operator)
	}

	builder := p.GormPluginFunctions.CreateSQLBuilder(query.Session(&gorm.Session{NewDB: true}))
	col := builder.QuoteIdentifier(atomic.Key)

	switch operator {
	case "IS NULL", "IS NOT NULL":
		return query.Where(col + " " + operator), nil
	case "IN", "NOT IN":
		return p.applyInCondition(query, col, operator, atomic.Value, columnType, isNullable)
	case "BETWEEN", "NOT BETWEEN":
		return p.applyBetweenCondition(query, col, operator, atomic.Value, columnType, isNullable)
	default:
		return p.applySingleValueCondition(query, col, operator, atomic.Value, columnType, isNullable)
	}
}

// applyInCondition handles IN and NOT IN operators with comma-separated values.
func (p *GormPlugin) applyInCondition(query *gorm.DB, col, operator, rawValue, columnType string, isNullable bool) (*gorm.DB, error) {
	raw := strings.TrimSpace(rawValue)
	if raw == "" {
		// empty IN list should return no rows; use a false predicate
		return query.Where("1 = 0"), nil
	}
	parts := strings.Split(raw, ",")
	vals := make([]any, 0, len(parts))
	for _, part := range parts {
		v := strings.TrimSpace(part)
		cv, err := p.GormPluginFunctions.ConvertStringValue(v, columnType, isNullable)
		if err != nil {
			log.WithError(err).Error(fmt.Sprintf("Failed to convert IN value '%s' for column type '%s'", v, columnType))
			return nil, err
		}
		vals = append(vals, cv)
	}
	return query.Where(col+" "+operator+" ?", vals), nil
}

// applyBetweenCondition handles BETWEEN and NOT BETWEEN operators with "min,max" values.
func (p *GormPlugin) applyBetweenCondition(query *gorm.DB, col, operator, rawValue, columnType string, isNullable bool) (*gorm.DB, error) {
	raw := strings.TrimSpace(rawValue)
	parts := strings.Split(raw, ",")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid BETWEEN value; expected 'min,max'")
	}
	v1, err := p.GormPluginFunctions.ConvertStringValue(strings.TrimSpace(parts[0]), columnType, isNullable)
	if err != nil {
		log.WithError(err).Error("Failed to convert BETWEEN min value")
		return nil, err
	}
	v2, err := p.GormPluginFunctions.ConvertStringValue(strings.TrimSpace(parts[1]), columnType, isNullable)
	if err != nil {
		log.WithError(err).Error("Failed to convert BETWEEN max value")
		return nil, err
	}
	if operator == "BETWEEN" {
		return query.Where(col+" BETWEEN ? AND ?", v1, v2), nil
	}
	return query.Where(col+" NOT BETWEEN ? AND ?", v1, v2), nil
}

// applySingleValueCondition handles operators that compare against a single value (=, <, >, LIKE, etc.).
func (p *GormPlugin) applySingleValueCondition(query *gorm.DB, col, operator, rawValue, columnType string, isNullable bool) (*gorm.DB, error) {
	value, err := p.GormPluginFunctions.ConvertStringValue(rawValue, columnType, isNullable)
	if err != nil {
		log.WithError(err).Error(fmt.Sprintf("Failed to convert string value '%s' for column type '%s'", rawValue, columnType))
		return nil, err
	}
	return query.Where(col+" "+operator+" ?", value), nil
}

// applyAndConditions applies all children as AND-combined WHERE clauses.
func (p *GormPlugin) applyAndConditions(query *gorm.DB, and *model.OperationWhereCondition, columnTypes map[string]ColumnTypeInfo) (*gorm.DB, error) {
	if and == nil {
		return query, nil
	}
	for _, child := range and.Children {
		var err error
		query, err = p.ApplyWhereConditions(query, child, columnTypes)
		if err != nil {
			log.WithError(err).Error("Failed to apply AND where condition")
			return nil, err
		}
	}
	return query, nil
}

// applyOrConditions applies all children as OR-combined WHERE clauses.
func (p *GormPlugin) applyOrConditions(query *gorm.DB, or *model.OperationWhereCondition, columnTypes map[string]ColumnTypeInfo) (*gorm.DB, error) {
	if or == nil {
		return query, nil
	}
	orQueries := query
	for _, child := range or.Children {
		childQuery, err := p.ApplyWhereConditions(query, child, columnTypes)
		if err != nil {
			log.WithError(err).Error("Failed to apply OR where condition")
			return nil, err
		}
		orQueries = orQueries.Or(childQuery)
	}
	return orQueries, nil
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

// ExecuteRawSQL handles raw SQL execution with optional multi-statement support.
// openMultiStatementDB is called when config.MultiStatement is true to get a DB connection
// that supports multiple statements. Pass nil if multi-statement is not supported.
func (p *GormPlugin) ExecuteRawSQL(config *engine.PluginConfig, openMultiStatementDB func(*engine.PluginConfig) (*gorm.DB, error), query string, params ...any) (*engine.GetRowsResult, error) {
	multiStatement := config != nil && config.MultiStatement
	dbFunc := p.DB
	if multiStatement && openMultiStatementDB != nil {
		dbFunc = openMultiStatementDB
	}

	return plugins.WithConnection(config, dbFunc, func(db *gorm.DB) (*engine.GetRowsResult, error) {
		if multiStatement {
			sqlDB, err := db.DB()
			if err != nil {
				return nil, err
			}
			_, err = sqlDB.Exec(query)
			if err != nil {
				return nil, err
			}
			return &engine.GetRowsResult{
				Columns: []engine.Column{},
				Rows:    [][]string{},
			}, nil
		}

		rows, err := db.Raw(query, params...).Rows()
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		return p.ConvertRawToRows(rows)
	})
}

// GetForeignKeyRelationships returns foreign key relationships for a table (default empty implementation)
func (p *GormPlugin) GetForeignKeyRelationships(config *engine.PluginConfig, schema string, storageUnit string) (map[string]*engine.ForeignKeyRelationship, error) {
	return make(map[string]*engine.ForeignKeyRelationship), nil
}

// QueryForeignKeyRelationships executes a foreign key query and returns the relationships map.
// This is a helper for SQL plugins that query system catalogs for FK information.
// The query must return exactly 3 columns: column_name, referenced_table, referenced_column.
func (p *GormPlugin) QueryForeignKeyRelationships(config *engine.PluginConfig, query string, params ...any) (map[string]*engine.ForeignKeyRelationship, error) {
	log.Debugf("[FK DEBUG] QueryForeignKeyRelationships called with params: %v", params)
	return plugins.WithConnection(config, p.DB, func(db *gorm.DB) (map[string]*engine.ForeignKeyRelationship, error) {
		rows, err := db.Raw(query, params...).Rows()
		if err != nil {
			log.Debugf("[FK DEBUG] ERROR: Raw query error: %v", err)
			return nil, err
		}
		defer rows.Close()

		relationships := make(map[string]*engine.ForeignKeyRelationship)
		for rows.Next() {
			var columnName, referencedTable, referencedColumn string
			if err := rows.Scan(&columnName, &referencedTable, &referencedColumn); err != nil {
				log.WithError(err).Error("Failed to scan foreign key relationship")
				continue
			}
			relationships[columnName] = &engine.ForeignKeyRelationship{
				ColumnName:       columnName,
				ReferencedTable:  referencedTable,
				ReferencedColumn: referencedColumn,
			}
		}
		log.Debugf("[FK DEBUG] Found %d FK relationships for params %v", len(relationships), params)
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
func (p *GormPlugin) GetMaxBulkInsertParameters() int {
	return 65535
}

// BuildSkipConflictClause returns ON CONFLICT (pk) DO NOTHING — works for Postgres, SQLite, ClickHouse.
// MySQL/MariaDB plugins override this with identity assignments.
func (p *GormPlugin) BuildSkipConflictClause(pkColumns []string) clause.OnConflict {
	conflictCols := make([]clause.Column, len(pkColumns))
	for i, col := range pkColumns {
		conflictCols[i] = clause.Column{Name: col}
	}
	return clause.OnConflict{
		Columns:   conflictCols,
		DoNothing: true,
	}
}

// MarkGeneratedColumns is a no-op base implementation.
// Database plugins should override this to detect auto-increment and computed columns.
func (p *GormPlugin) MarkGeneratedColumns(config *engine.PluginConfig, schema string, storageUnit string, columns []engine.Column) error {
	return nil
}

// GetDatabaseMetadata returns nil by default.
// Database plugins should override this to provide metadata for frontend configuration.
func (p *GormPlugin) GetDatabaseMetadata() *engine.DatabaseMetadata {
	return nil
}
