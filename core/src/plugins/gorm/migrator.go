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
	"fmt"
	"strings"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
	"gorm.io/gorm"
)

// MigratorHelper provides schema operations using GORM's Migrator interface
// NOTE: Most methods are not yet used but are prepared for future schema modification features
type MigratorHelper struct {
	db       *gorm.DB
	plugin   GormPluginFunctions
	migrator gorm.Migrator
}

// NewMigratorHelper creates a new migrator helper
func NewMigratorHelper(db *gorm.DB, plugin GormPluginFunctions) *MigratorHelper {
	return &MigratorHelper{
		db:       db,
		plugin:   plugin,
		migrator: db.Migrator(),
	}
}

// TableExists checks if a table exists using Migrator
func (m *MigratorHelper) TableExists(tableName string) bool {
	return m.migrator.HasTable(tableName)
}

// DropTable drops a table using Migrator // todo: NOT YET SUPPORTED
func (m *MigratorHelper) DropTable(tableName string) error {
	return m.migrator.DropTable(tableName)
}

// RenameTable renames a table using Migrator // todo: NOT YET SUPPORTED
func (m *MigratorHelper) RenameTable(oldName, newName string) error {
	return m.migrator.RenameTable(oldName, newName)
}

// GetIndexes gets indexes for a table using Migrator //todo: NOT YET SUPPORTED
func (m *MigratorHelper) GetIndexes(tableName string) ([]gorm.Index, error) {
	return m.migrator.GetIndexes(tableName)
}

// CreateIndex creates an index using Migrator. todo: NOT YET SUPPORTED
func (m *MigratorHelper) CreateIndex(tableName string, indexName string, columns ...string) error {
	// Use GORM's migrator to check if index exists
	if m.migrator.HasIndex(tableName, indexName) {
		return fmt.Errorf("index %s already exists", indexName)
	}

	// Since we don't have typed structs, we can't use migrator.CreateIndex directly
	// DDL statements don't support placeholders for identifiers
	// So we must use the plugin's escape function for safety

	// Build escaped column list
	escapedColumns := make([]string, len(columns))
	for i, col := range columns {
		escapedColumns[i] = m.plugin.EscapeIdentifier(col)
	}
	columnList := strings.Join(escapedColumns, ", ")

	// Build the CREATE INDEX statement with escaped identifiers
	query := "CREATE INDEX " + m.plugin.EscapeIdentifier(indexName) +
		" ON " + m.plugin.EscapeIdentifier(tableName) +
		" (" + columnList + ")"

	return m.db.Exec(query).Error
}

// DropIndex drops an index using Migrator todo: NOT YET SUPPORTED
func (m *MigratorHelper) DropIndex(tableName string, indexName string) error {
	return m.migrator.DropIndex(tableName, indexName)
}

// GetColumnTypes gets column types using Migrator's ColumnTypes
func (m *MigratorHelper) GetColumnTypes(tableName string) (map[string]string, error) {
	columnTypes := make(map[string]string)

	// Try to use GORM's Migrator ColumnTypes method
	types, err := m.migrator.ColumnTypes(tableName)
	if err != nil {
		// Fall back to raw SQL if Migrator doesn't work
		return m.getColumnTypesRaw(tableName)
	}

	for _, col := range types {
		columnTypes[col.Name()] = strings.ToUpper(col.DatabaseTypeName())
	}

	return columnTypes, nil
}

// getColumnTypesRaw falls back to raw SQL for column types
func (m *MigratorHelper) getColumnTypesRaw(tableName string) (map[string]string, error) {
	columnTypes := make(map[string]string)

	// Extract schema and table name
	parts := strings.Split(tableName, ".")
	var schema, table string
	if len(parts) == 2 {
		schema = parts[0]
		table = parts[1]
	} else {
		table = tableName
	}

	query := m.plugin.GetColTypeQuery()

	var rows *sql.Rows
	var err error

	if m.plugin.GetDatabaseType() == engine.DatabaseType_Sqlite3 {
		rows, err = m.db.Raw(query, table).Rows()
	} else {
		rows, err = m.db.Raw(query, schema, table).Rows()
	}

	if err != nil {
		log.Logger.WithError(err).WithField("table", tableName).Error("Failed to execute column types query")
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var columnName, dataType string
		if err := rows.Scan(&columnName, &dataType); err != nil {
			log.Logger.WithError(err).WithField("table", tableName).Error("Failed to scan column type data")
			return nil, err
		}
		columnTypes[columnName] = dataType
	}

	if err := rows.Err(); err != nil {
		log.Logger.WithError(err).WithField("table", tableName).Error("Row iteration error while getting column types")
		return nil, err
	}

	return columnTypes, nil
}

// GetPrimaryKeys gets primary key columns using Migrator
func (m *MigratorHelper) GetPrimaryKeys(tableName string) ([]string, error) {
	var primaryKeys []string

	// Try to get column types which includes primary key info
	columnTypes, err := m.migrator.ColumnTypes(tableName)
	if err != nil {
		// Fall back to raw SQL if Migrator doesn't work
		return m.getPrimaryKeysRaw(tableName)
	}

	for _, col := range columnTypes {
		isPrimary, ok := col.PrimaryKey()
		if ok && isPrimary {
			primaryKeys = append(primaryKeys, col.Name())
		}
	}

	if len(primaryKeys) == 0 {
		// If no primary keys found via Migrator, fall back to raw SQL
		return m.getPrimaryKeysRaw(tableName)
	}

	return primaryKeys, nil
}

// getPrimaryKeysRaw falls back to raw SQL for primary keys
func (m *MigratorHelper) getPrimaryKeysRaw(tableName string) ([]string, error) {
	// Extract schema and table name
	parts := strings.Split(tableName, ".")
	var schema, table string
	if len(parts) == 2 {
		schema = parts[0]
		table = parts[1]
	} else {
		table = tableName
	}

	return m.plugin.GetPrimaryKeyColumns(m.db, schema, table)
}

// HasColumn checks if a column exists using Migrator // todo: NOT YET SUPPORTED
func (m *MigratorHelper) HasColumn(tableName string, columnName string) bool {
	// Since we don't have typed structs, we need to pass the table name directly
	return m.migrator.HasColumn(tableName, columnName)
}

// AddColumn adds a column to a table todo: NOT YET SUPPORTED
func (m *MigratorHelper) AddColumn(tableName string, columnName string, columnType string) error {
	// DDL statements don't support placeholders for identifiers
	// We must use escaped identifiers for safety
	query := "ALTER TABLE " + m.plugin.EscapeIdentifier(tableName) +
		" ADD COLUMN " + m.plugin.EscapeIdentifier(columnName) +
		" " + columnType
	return m.db.Exec(query).Error
}

// DropColumn drops a column from a table todo: NOT YET SUPPORTED
func (m *MigratorHelper) DropColumn(tableName string, columnName string) error {
	// DDL statements don't support placeholders for identifiers
	query := "ALTER TABLE " + m.plugin.EscapeIdentifier(tableName) +
		" DROP COLUMN " + m.plugin.EscapeIdentifier(columnName)
	return m.db.Exec(query).Error
}

// RenameColumn renames a column in a table todo: NOT YET SUPPORTED
func (m *MigratorHelper) RenameColumn(tableName string, oldName, newName string, columnType string) error {
	// Implementation varies by database
	var query string
	escapedTable := m.plugin.EscapeIdentifier(tableName)
	escapedOld := m.plugin.EscapeIdentifier(oldName)
	escapedNew := m.plugin.EscapeIdentifier(newName)

	switch m.plugin.GetDatabaseType() {
	case engine.DatabaseType_Postgres:
		query = "ALTER TABLE " + escapedTable +
			" RENAME COLUMN " + escapedOld +
			" TO " + escapedNew
	case engine.DatabaseType_MySQL, engine.DatabaseType_MariaDB:
		// MySQL needs the column type for rename
		query = "ALTER TABLE " + escapedTable +
			" CHANGE " + escapedOld +
			" " + escapedNew +
			" " + columnType
	case engine.DatabaseType_Sqlite3:
		// SQLite doesn't support direct column rename before version 3.25.0
		// Would need to recreate table - too complex for this helper
		return fmt.Errorf("column rename not supported for SQLite in this helper")
	default:
		query = "ALTER TABLE " + escapedTable +
			" RENAME COLUMN " + escapedOld +
			" TO " + escapedNew
	}

	return m.db.Exec(query).Error
}

// GetConstraints gets table constraints using Migrator
func (m *MigratorHelper) GetConstraints(tableName string) (map[string][]gorm.ColumnType, error) {
	// GORM's Migrator doesn't directly expose constraints
	// We can get column types which include some constraint info
	columnTypes, err := m.migrator.ColumnTypes(tableName)
	if err != nil {
		return nil, err
	}

	constraints := make(map[string][]gorm.ColumnType)
	for _, col := range columnTypes {
		// Check for various constraints
		if primary, ok := col.PrimaryKey(); ok && primary {
			constraints["PRIMARY"] = append(constraints["PRIMARY"], col)
		}
		if unique, ok := col.Unique(); ok && unique {
			constraints["UNIQUE"] = append(constraints["UNIQUE"], col)
		}
		if nullable, ok := col.Nullable(); ok && !nullable {
			constraints["NOT_NULL"] = append(constraints["NOT_NULL"], col)
		}
	}

	return constraints, nil
}

// CreateConstraint creates a constraint on a table // todo: NOT YET SUPPORTED
func (m *MigratorHelper) CreateConstraint(tableName string, constraintName string, constraintType string, columns ...string) error {
	// Build escaped column list
	escapedColumns := make([]string, len(columns))
	for i, col := range columns {
		escapedColumns[i] = m.plugin.EscapeIdentifier(col)
	}
	columnList := strings.Join(escapedColumns, ", ")

	escapedTable := m.plugin.EscapeIdentifier(tableName)
	escapedConstraint := m.plugin.EscapeIdentifier(constraintName)

	var query string
	switch strings.ToUpper(constraintType) {
	case "UNIQUE":
		query = "ALTER TABLE " + escapedTable +
			" ADD CONSTRAINT " + escapedConstraint +
			" UNIQUE (" + columnList + ")"
	case "CHECK":
		// Check constraints need the condition, not just columns
		return fmt.Errorf("CHECK constraints require a condition, not just columns")
	case "FOREIGN KEY", "FOREIGN_KEY":
		// Foreign keys need reference info
		return fmt.Errorf("FOREIGN KEY constraints require reference table and columns")
	default:
		return fmt.Errorf("unsupported constraint type: %s", constraintType)
	}

	return m.db.Exec(query).Error
}

// DropConstraint drops a constraint from a table // todo: NOT YET SUPPORTED
func (m *MigratorHelper) DropConstraint(tableName string, constraintName string) error {
	escapedTable := m.plugin.EscapeIdentifier(tableName)
	escapedConstraint := m.plugin.EscapeIdentifier(constraintName)

	var query string
	switch m.plugin.GetDatabaseType() {
	case engine.DatabaseType_MySQL, engine.DatabaseType_MariaDB:
		// MySQL uses different syntax for different constraint types
		// For simplicity, try dropping as index first, then as foreign key
		query = "ALTER TABLE " + escapedTable +
			" DROP INDEX " + escapedConstraint
	default:
		query = "ALTER TABLE " + escapedTable +
			" DROP CONSTRAINT " + escapedConstraint
	}

	return m.db.Exec(query).Error
}
