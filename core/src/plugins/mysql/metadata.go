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

package mysql

import (
	"database/sql"
	"fmt"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/migrator"

	gorm_plugin "github.com/clidey/whodb/core/src/plugins/gorm"
)

type mysqlColumnMetadata struct {
	name            string
	dataType        string
	isNullable      bool
	isPrimary       bool
	isAutoIncrement bool
	isUnique        bool
	columnType      string
	defaultValue    sql.NullString
	comment         sql.NullString
	length          sql.NullInt64
	precision       sql.NullInt64
	scale           sql.NullInt64
}

// QuotedColumnTypes returns complete MySQL metadata for identifiers that GORM's migrator cannot safely quote.
func (p *MySQLPlugin) QuotedColumnTypes(db *gorm.DB, schema, tableName string) ([]gorm.ColumnType, error) {
	rawTypes, err := gorm_plugin.ReadSQLColumnTypes(p.CreateSQLBuilder(db).GetTableQuery(schema, tableName))
	if err != nil {
		return nil, err
	}
	metadata, err := p.readColumnMetadata(db, schema, tableName)
	if err != nil {
		return nil, err
	}
	types := make([]gorm.ColumnType, 0, len(metadata))
	for i := range metadata {
		column := &metadata[i]
		types = append(types, migrator.ColumnType{
			NameValue:          sql.NullString{String: column.name, Valid: true},
			DataTypeValue:      sql.NullString{String: column.dataType, Valid: true},
			ColumnTypeValue:    sql.NullString{String: column.columnType, Valid: true},
			NullableValue:      sql.NullBool{Bool: column.isNullable, Valid: true},
			PrimaryKeyValue:    sql.NullBool{Bool: column.isPrimary, Valid: true},
			UniqueValue:        sql.NullBool{Bool: column.isUnique, Valid: true},
			AutoIncrementValue: sql.NullBool{Bool: column.isAutoIncrement, Valid: column.isAutoIncrement},
			DefaultValueValue:  column.defaultValue,
			CommentValue:       column.comment,
			LengthValue:        column.length,
			DecimalSizeValue:   column.precision,
			ScaleValue:         column.scale,
			SQLColumnType:      rawTypes[column.name],
		})
	}
	return types, nil
}

func (p *MySQLPlugin) readColumnMetadata(db *gorm.DB, schema, tableName string) ([]mysqlColumnMetadata, error) {
	rows, err := db.Raw(`
		SELECT COLUMN_NAME, COLUMN_DEFAULT, DATA_TYPE, COLUMN_TYPE, IS_NULLABLE,
		       COLUMN_KEY, EXTRA, COLUMN_COMMENT, CHARACTER_MAXIMUM_LENGTH,
		       NUMERIC_PRECISION, NUMERIC_SCALE, DATETIME_PRECISION
		FROM INFORMATION_SCHEMA.COLUMNS
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
		ORDER BY ORDINAL_POSITION
	`, schema, tableName).Rows()
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var columns []mysqlColumnMetadata
	for rows.Next() {
		var column mysqlColumnMetadata
		var nullable, key, extra string
		var datetimePrecision sql.NullInt64
		if err := rows.Scan(
			&column.name,
			&column.defaultValue,
			&column.dataType,
			&column.columnType,
			&nullable,
			&key,
			&extra,
			&column.comment,
			&column.length,
			&column.precision,
			&column.scale,
			&datetimePrecision,
		); err != nil {
			return nil, err
		}
		column.isNullable = strings.EqualFold(nullable, "YES")
		column.isPrimary = key == "PRI"
		column.isUnique = key == "UNI"
		column.isAutoIncrement = strings.Contains(strings.ToLower(extra), "auto_increment")
		if datetimePrecision.Valid {
			column.precision = datetimePrecision
		}
		column.defaultValue.String = trimMySQLDefault(column.defaultValue.String)
		columns = append(columns, column)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read MySQL column metadata: %w", err)
	}
	return columns, nil
}

func trimMySQLDefault(value string) string {
	for (len(value) >= 3 && value[0] == '\'' && value[len(value)-1] == '\'' && value[len(value)-2] != '\\') ||
		(len(value) == 2 && value == "''") {
		value = value[1 : len(value)-1]
	}
	return value
}
