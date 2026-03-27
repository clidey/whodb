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
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/migrator"
)

// DuckDBMigrator wraps GORM's base migrator with DuckDB-specific overrides.
type DuckDBMigrator struct {
	migrator.Migrator
}

// ColumnTypes queries information_schema.columns for accurate column metadata.
// The value may be a schema-qualified name like "main.orders" from FormTableName.
func (m DuckDBMigrator) ColumnTypes(value any) ([]gorm.ColumnType, error) {
	var resolved string
	switch v := value.(type) {
	case string:
		resolved = v
	default:
		stmt := &gorm.Statement{DB: m.DB}
		if err := stmt.Parse(value); err != nil {
			return nil, err
		}
		resolved = stmt.Table
	}

	// Parse schema.table if qualified, otherwise default to "main"
	schemaName := "main"
	tableName := resolved
	if parts := strings.SplitN(resolved, ".", 2); len(parts) == 2 {
		schemaName = parts[0]
		tableName = parts[1]
	}

	query := `
		SELECT
			c.column_name,
			c.data_type,
			c.is_nullable,
			c.column_default,
			c.character_maximum_length,
			c.numeric_precision,
			c.numeric_scale,
			CASE WHEN tc.constraint_type = 'PRIMARY KEY' THEN 'YES' ELSE 'NO' END AS is_primary,
			CASE WHEN c.column_default LIKE 'nextval%%' OR c.is_identity = 'YES' THEN 'YES' ELSE 'NO' END AS is_autoincrement
		FROM information_schema.columns c
		LEFT JOIN information_schema.key_column_usage kcu
			ON c.table_schema = kcu.table_schema AND c.table_name = kcu.table_name AND c.column_name = kcu.column_name
		LEFT JOIN information_schema.table_constraints tc
			ON kcu.constraint_name = tc.constraint_name AND kcu.table_schema = tc.table_schema AND tc.constraint_type = 'PRIMARY KEY'
		WHERE c.table_schema = ? AND c.table_name = ?
		ORDER BY c.ordinal_position
	`

	rows, err := m.DB.Raw(query, schemaName, tableName).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columnTypes []gorm.ColumnType
	for rows.Next() {
		var (
			name            string
			dataType        string
			nullable        string
			columnDefault   sql.NullString
			charMaxLen      sql.NullInt64
			numericPrec     sql.NullInt64
			numericScale    sql.NullInt64
			isPrimary       string
			isAutoIncrement string
		)

		if err := rows.Scan(&name, &dataType, &nullable, &columnDefault, &charMaxLen, &numericPrec, &numericScale, &isPrimary, &isAutoIncrement); err != nil {
			return nil, err
		}

		// Ensure all NullInt64 values are always Valid (default 0) to prevent
		// nil pointer panics in gorm's migrator.ColumnType.Length() and
		// DecimalSize() which fall through to SQLColumnType when not valid.
		if !charMaxLen.Valid {
			charMaxLen = sql.NullInt64{Int64: 0, Valid: true}
		}
		if !numericPrec.Valid {
			numericPrec = sql.NullInt64{Int64: 0, Valid: true}
		}
		if !numericScale.Valid {
			numericScale = sql.NullInt64{Int64: 0, Valid: true}
		}

		ct := migrator.ColumnType{
			NameValue:          sql.NullString{String: name, Valid: true},
			DataTypeValue:      sql.NullString{String: dataType, Valid: true},
			ColumnTypeValue:    sql.NullString{String: dataType, Valid: true},
			NullableValue:      sql.NullBool{Bool: nullable == "YES", Valid: true},
			PrimaryKeyValue:    sql.NullBool{Bool: isPrimary == "YES", Valid: true},
			AutoIncrementValue: sql.NullBool{Bool: isAutoIncrement == "YES", Valid: true},
			DefaultValueValue:  columnDefault,
			LengthValue:        charMaxLen,
			DecimalSizeValue:   numericPrec,
			ScaleValue:         numericScale,
		}
		columnTypes = append(columnTypes, ct)
	}

	return columnTypes, rows.Err()
}
