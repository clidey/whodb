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
	"fmt"
	"strings"

	"github.com/clidey/whodb/core/src/plugins"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SQLBuilderInterface defines the methods that can be overridden by database-specific implementations
type SQLBuilderInterface interface {
	QuoteIdentifier(identifier string) string
	BuildFullTableName(schema, table string) string
	GetTableQuery(schema, table string) *gorm.DB
	SelectQuery(schema, table string, columns []string, conditions map[string]any) *gorm.DB
	BuildOrderBy(query *gorm.DB, sortList []plugins.Sort) *gorm.DB
	CreateTableQuery(schema, table string, columns []ColumnDef) string
	CreateTableQueryWithSuffix(schema, table string, columns []ColumnDef, suffix string) string
	InsertRow(schema, table string, data map[string]any) error
	UpdateQuery(schema, table string, updates map[string]any, conditions map[string]any) *gorm.DB
	DeleteQuery(schema, table string, conditions map[string]any) *gorm.DB
	CountQuery(schema, table string) (int64, error)
}

// SQLBuilder provides SQL query building functionality
// This builder prioritizes GORM's native methods for all operations where possible.
type SQLBuilder struct {
	db     *gorm.DB
	plugin GormPluginFunctions
	self   SQLBuilderInterface // Reference to the actual implementation
}

// NewSQLBuilder creates a new SQL builder
func NewSQLBuilder(db *gorm.DB, plugin GormPluginFunctions) *SQLBuilder {
	sb := &SQLBuilder{
		db:     db,
		plugin: plugin,
	}
	sb.self = sb // Default to self, will be overridden if wrapped
	return sb
}

// SetSelf sets the reference to the actual implementation (used by subclasses)
func (sb *SQLBuilder) SetSelf(self SQLBuilderInterface) {
	sb.self = self
}

// GetDB returns the underlying GORM DB instance
func (sb *SQLBuilder) GetDB() *gorm.DB {
	return sb.db
}

// QuoteIdentifier quotes an identifier (column/table name) ONLY for DDL operations
// For DML operations (SELECT, INSERT, UPDATE, DELETE), use GORM's native methods
// which handle escaping automatically
func (sb *SQLBuilder) QuoteIdentifier(identifier string) string {
	// Prefer GORM dialect quoting when available
	if sb.db != nil && sb.db.Dialector != nil {
		var b strings.Builder
		sb.db.Dialector.QuoteTo(&b, identifier)
		return b.String()
	}
	// If no dialect is available (should be rare), return as-is
	return identifier
}

// BuildFullTableName builds a fully qualified table name for GORM operations
// GORM's dialector handles identifier quoting internally via QuoteTo method.
func (sb *SQLBuilder) BuildFullTableName(schema, table string) string {
	if table == "" {
		return ""
	}
	if schema == "" {
		return table
	}
	return schema + "." + table
}

// GetTableQuery creates a GORM query with the appropriate table reference
// This method can be overridden by database-specific implementations
func (sb *SQLBuilder) GetTableQuery(schema, table string) *gorm.DB {
	fullTableName := sb.BuildFullTableName(schema, table)
	return sb.db.Table(fullTableName)
}

// SelectQuery builds a SELECT query using GORM's query builder
// GORM handles all escaping automatically
func (sb *SQLBuilder) SelectQuery(schema, table string, columns []string, conditions map[string]any) *gorm.DB {
	query := sb.self.GetTableQuery(schema, table)

	// GORM handles column name escaping automatically
	if len(columns) > 0 {
		query = query.Select(columns)
	}

	// Add WHERE conditions using GORM's native support
	if len(conditions) > 0 {
		query = query.Where(conditions)
	}

	return query
}

// BuildOrderBy builds ORDER BY clause using GORM's Order method
// GORM handles column name escaping automatically
func (sb *SQLBuilder) BuildOrderBy(query *gorm.DB, sortList []plugins.Sort) *gorm.DB {
	for _, sort := range sortList {
		query = query.Order(clause.OrderByColumn{
			Column: clause.Column{Name: sort.Column},
			Desc:   sort.Direction == plugins.Down,
		})
	}
	return query
}

// CreateTableQuery builds a CREATE TABLE statement for DDL operations
// DDL requires manual SQL building as GORM doesn't support dynamic table creation
func (sb *SQLBuilder) CreateTableQuery(schema, table string, columns []ColumnDef) string {
	columnDefs := make([]string, len(columns))
	var primaryKeys []string

	for i, col := range columns {
		def := sb.QuoteIdentifier(col.Name) + " " + col.Type

		if col.Primary {
			primaryKeys = append(primaryKeys, sb.QuoteIdentifier(col.Name))
		}

		if !col.Nullable && !col.Primary {
			def += " NOT NULL"
		}

		columnDefs[i] = def
	}

	// Add primary key constraint if there are primary keys
	if len(primaryKeys) > 0 {
		columnDefs = append(columnDefs, fmt.Sprintf("PRIMARY KEY (%s)", strings.Join(primaryKeys, ", ")))
	}

	// For DDL, we need proper identifier escaping
	var fullTableName string
	if schema == "" {
		fullTableName = sb.QuoteIdentifier(table)
	} else {
		fullTableName = sb.QuoteIdentifier(schema) + "." + sb.QuoteIdentifier(table)
	}
	return fmt.Sprintf("CREATE TABLE %s (%s)", fullTableName, strings.Join(columnDefs, ", "))
}

// CreateTableQueryWithSuffix builds a CREATE TABLE statement with a suffix (for ClickHouse ENGINE, etc)
func (sb *SQLBuilder) CreateTableQueryWithSuffix(schema, table string, columns []ColumnDef, suffix string) string {
	baseQuery := sb.CreateTableQuery(schema, table, columns)
	if suffix != "" {
		return baseQuery + " " + suffix
	}
	return baseQuery
}

// InsertRow inserts a row using GORM's Create method
// GORM handles all escaping automatically
func (sb *SQLBuilder) InsertRow(schema, table string, data map[string]any) error {
	if table == "" {
		return fmt.Errorf("table name cannot be empty when inserting row")
	}

	// Let GORM handle the table name formatting
	// GORM's dialect will properly escape and format the table name
	tableName := table
	if schema != "" {
		// For databases that support schemas, use schema.table format
		// GORM will handle this appropriately for each dialect
		tableName = schema + "." + table
	}

	result := sb.db.Table(tableName).Create(data)
	if result.Error != nil {
		return result.Error
	}

	return nil
}

// UpdateQuery builds an UPDATE query using GORM's Update methods
// GORM handles all escaping automatically
func (sb *SQLBuilder) UpdateQuery(schema, table string, updates map[string]any, conditions map[string]any) *gorm.DB {
	// Let GORM handle the table name formatting
	tableName := table
	if schema != "" {
		tableName = schema + "." + table
	}

	query := sb.db.Table(tableName)

	// Add WHERE conditions using GORM's native support
	if len(conditions) > 0 {
		query = query.Where(conditions)
	}

	// Add updates - GORM handles column escaping
	return query.Updates(updates)
}

// DeleteQuery builds a DELETE query using GORM's Delete method
// GORM handles all escaping automatically
func (sb *SQLBuilder) DeleteQuery(schema, table string, conditions map[string]any) *gorm.DB {
	// Let GORM handle the table name formatting
	tableName := table
	if schema != "" {
		tableName = schema + "." + table
	}

	query := sb.db.Table(tableName)

	// Add WHERE conditions using GORM's native support
	if len(conditions) > 0 {
		query = query.Where(conditions)
	}

	return query.Delete(nil)
}

// CountQuery builds a COUNT query using GORM's Count method
// GORM handles all escaping automatically
func (sb *SQLBuilder) CountQuery(schema, table string) (int64, error) {
	var count int64
	// Let GORM handle the table name formatting
	tableName := table
	if schema != "" {
		tableName = schema + "." + table
	}

	err := sb.db.Table(tableName).Count(&count).Error
	return count, err
}

// ColumnDef represents a column definition for CREATE TABLE
type ColumnDef struct {
	Name     string
	Type     string
	Primary  bool
	Nullable bool
	NotNull  bool   // Explicit NOT NULL flag (opposite of Nullable)
	Extra    string // Additional column modifiers (e.g., AUTO_INCREMENT, DEFAULT)
}
