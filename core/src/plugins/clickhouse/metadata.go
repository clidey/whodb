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

package clickhouse

import (
	"database/sql"
	"fmt"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/migrator"

	gorm_plugin "github.com/clidey/whodb/core/src/plugins/gorm"
)

// QuotedColumnTypes returns complete ClickHouse metadata for identifiers that require quoting.
func (p *ClickHousePlugin) QuotedColumnTypes(db *gorm.DB, schema, table string) ([]gorm.ColumnType, error) {
	if schema == "" {
		if err := db.Raw("SELECT currentDatabase()").Scan(&schema).Error; err != nil {
			return nil, err
		}
	}
	rawTypes, err := gorm_plugin.ReadSQLColumnTypes(p.CreateSQLBuilder(db).GetTableQuery(schema, table))
	if err != nil {
		return nil, err
	}
	rows, err := db.Raw(`
		SELECT name, type, default_expression, comment, is_in_primary_key,
		       character_octet_length, numeric_precision, numeric_scale, datetime_precision
		FROM system.columns
		WHERE database = ? AND table = ?
		ORDER BY position
	`, schema, table).Rows()
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var types []gorm.ColumnType
	for rows.Next() {
		var (
			name, dataType string
			defaultValue   sql.NullString
			comment        sql.NullString
			isPrimary      bool
			length         sql.NullInt64
			precision      sql.NullInt64
			scale          sql.NullInt64
			datetime       sql.NullInt64
		)
		if err := rows.Scan(&name, &dataType, &defaultValue, &comment, &isPrimary, &length, &precision, &scale, &datetime); err != nil {
			return nil, err
		}
		if defaultValue.Valid {
			defaultValue.String = strings.Trim(defaultValue.String, "'")
		}
		if datetime.Valid {
			precision = datetime
		}
		types = append(types, migrator.ColumnType{
			NameValue:         sql.NullString{String: name, Valid: true},
			DataTypeValue:     sql.NullString{String: dataType, Valid: true},
			ColumnTypeValue:   sql.NullString{String: dataType, Valid: true},
			PrimaryKeyValue:   sql.NullBool{Bool: isPrimary, Valid: true},
			DefaultValueValue: defaultValue,
			CommentValue:      comment,
			LengthValue:       length,
			DecimalSizeValue:  precision,
			ScaleValue:        scale,
			SQLColumnType:     rawTypes[name],
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read ClickHouse column metadata: %w", err)
	}
	return types, nil
}
