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

package postgres

import (
	"database/sql"
	"fmt"
	"regexp"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/migrator"

	gorm_plugin "github.com/clidey/whodb/core/src/plugins/gorm"
)

var (
	quotedAutoIncrementPattern = regexp.MustCompile(`^nextval\('"?[^']+seq"?'::regclass\)$`)
	quotedDefaultValuePattern  = regexp.MustCompile(`^(.*?)(?:::.*)?$`)
)

// QuotedColumnTypes returns complete PostgreSQL metadata for identifiers that GORM's migrator cannot safely quote.
func (p *PostgresPlugin) QuotedColumnTypes(db *gorm.DB, schema, table string) ([]gorm.ColumnType, error) {
	rawTypes, err := gorm_plugin.ReadSQLColumnTypes(p.CreateSQLBuilder(db).GetTableQuery(schema, table))
	if err != nil {
		return nil, err
	}
	rows, err := db.Raw(`
		SELECT c.column_name,
		       c.is_nullable = 'YES',
		       c.udt_name,
		       c.character_maximum_length,
		       c.numeric_precision,
		       c.numeric_scale,
		       c.datetime_precision,
		       c.column_default,
		       c.identity_increment,
		       pg_catalog.format_type(a.atttypid, a.atttypmod),
		       a.attlen,
		       pg_catalog.col_description(t.oid, a.attnum),
		       EXISTS (
		           SELECT 1 FROM pg_catalog.pg_constraint con
		           WHERE con.conrelid = t.oid AND con.contype = 'p' AND a.attnum = ANY(con.conkey)
		       ),
		       EXISTS (
		           SELECT 1 FROM pg_catalog.pg_constraint con
		           WHERE con.conrelid = t.oid AND con.contype = 'u'
		             AND array_length(con.conkey, 1) = 1 AND a.attnum = ANY(con.conkey)
		       )
		FROM information_schema.columns c
		JOIN pg_catalog.pg_namespace n ON n.nspname = c.table_schema
		JOIN pg_catalog.pg_class t ON t.relnamespace = n.oid AND t.relname = c.table_name
		JOIN pg_catalog.pg_attribute a ON a.attrelid = t.oid AND a.attname = c.column_name
		WHERE c.table_catalog = current_database()
		  AND c.table_schema = ?
		  AND c.table_name = ?
		ORDER BY c.ordinal_position
	`, schema, table).Rows()
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var types []gorm.ColumnType
	for rows.Next() {
		var (
			name, dataType, formattedType string
			nullable                      bool
			length, precision, scale      sql.NullInt64
			datetimePrecision             sql.NullInt64
			defaultValue, identity        sql.NullString
			typeLength                    sql.NullInt64
			comment                       sql.NullString
			primary, unique               bool
		)
		if err := rows.Scan(
			&name,
			&nullable,
			&dataType,
			&length,
			&precision,
			&scale,
			&datetimePrecision,
			&defaultValue,
			&identity,
			&formattedType,
			&typeLength,
			&comment,
			&primary,
			&unique,
		); err != nil {
			return nil, err
		}
		if strings.HasPrefix(dataType, "_") {
			dataType = formattedType
		}
		if datetimePrecision.Valid {
			precision = datetimePrecision
		}
		if typeLength.Valid && typeLength.Int64 > 0 {
			length = typeLength
		}
		autoIncrement := identity.Valid || (defaultValue.Valid && quotedAutoIncrementPattern.MatchString(defaultValue.String))
		if autoIncrement {
			defaultValue = sql.NullString{}
		} else if defaultValue.Valid {
			defaultValue.String = strings.Trim(quotedDefaultValuePattern.ReplaceAllString(defaultValue.String, "$1"), "'")
		}
		types = append(types, &migrator.ColumnType{
			NameValue:          sql.NullString{String: name, Valid: true},
			DataTypeValue:      sql.NullString{String: dataType, Valid: true},
			ColumnTypeValue:    sql.NullString{String: formattedType, Valid: true},
			NullableValue:      sql.NullBool{Bool: nullable, Valid: true},
			PrimaryKeyValue:    sql.NullBool{Bool: primary, Valid: true},
			UniqueValue:        sql.NullBool{Bool: unique, Valid: true},
			AutoIncrementValue: sql.NullBool{Bool: autoIncrement, Valid: true},
			DefaultValueValue:  defaultValue,
			CommentValue:       comment,
			LengthValue:        length,
			DecimalSizeValue:   precision,
			ScaleValue:         scale,
			SQLColumnType:      rawTypes[name],
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read PostgreSQL column metadata: %w", err)
	}
	return types, nil
}
