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

package sqlite3

import (
	"database/sql/driver"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

// DateTimeString is a custom type that stores datetime values as plain strings
// without any parsing or formatting. This is needed because SQLite stores
// datetime values as TEXT and we want to preserve the exact format.
type DateTimeString string

// Scan implements sql.Scanner interface to read datetime values as strings
func (ds *DateTimeString) Scan(value any) error {
	if value == nil {
		*ds = ""
		return nil
	}

	switch v := value.(type) {
	case string:
		*ds = DateTimeString(v)
	case []byte:
		*ds = DateTimeString(v)
	case driver.Value:
		// Handle case where the driver returns a driver.Value
		str := fmt.Sprintf("%v", v)
		*ds = DateTimeString(str)
	default:
		// Handle time.Time from go-sqlite3's datetime auto-parsing.
		// The driver parses datetime strings into time.Time before our scanner sees them,
		// so we format back without the "+0000 UTC" suffix that fmt.Sprintf produces.
		// Zero time (from failed parse of non-datetime text) is treated as empty.
		if t, ok := value.(time.Time); ok {
			if t.IsZero() {
				*ds = ""
			} else if t.Nanosecond() > 0 {
				*ds = DateTimeString(t.Format("2006-01-02 15:04:05.999999999"))
			} else {
				*ds = DateTimeString(t.Format("2006-01-02 15:04:05"))
			}
		} else {
			str := fmt.Sprintf("%v", value)
			*ds = DateTimeString(str)
		}
	}
	return nil
}

// Value implements driver.Valuer interface to write datetime values as strings
func (ds DateTimeString) Value() (driver.Value, error) {
	return string(ds), nil
}

// ConvertStringValue overrides the base GORM implementation to preserve datetime strings
func (p *Sqlite3Plugin) ConvertStringValue(value, columnType string, isNullable bool) (any, error) {
	// For datetime types, preserve the original string value
	normalizedType := strings.ToUpper(columnType)
	if normalizedType == "DATE" || normalizedType == "DATETIME" || normalizedType == "TIMESTAMP" {
		return value, nil
	}
	// For non-datetime types, delegate to the base GORM implementation
	return p.GormPlugin.ConvertStringValue(value, columnType, isNullable)
}

// GetPrimaryKeyColumns overrides the base implementation because SQLite's
// primary key query takes only a table name parameter (no schema).
func (p *Sqlite3Plugin) GetPrimaryKeyColumns(db *gorm.DB, schema string, tableName string) ([]string, error) {
	query := p.GetPrimaryKeyColQuery()
	if query == "" {
		return nil, nil
	}

	rows, err := db.Raw(query, tableName).Rows()
	if err != nil {
		return nil, nil
	}
	defer rows.Close()

	var primaryKeys []string
	for rows.Next() {
		var columnName string
		if err := rows.Scan(&columnName); err != nil {
			continue
		}
		primaryKeys = append(primaryKeys, columnName)
	}
	return primaryKeys, nil
}

func (p *Sqlite3Plugin) GetPrimaryKeyColQuery() string {
	return `
		SELECT p.name AS pk_column
		FROM sqlite_master m,
			 pragma_table_info(m.name) p
		WHERE m.type = 'table'
		  AND m.name = ?
		  AND m.name NOT LIKE 'sqlite_%'
		  AND p.pk > 0
		ORDER BY m.name, p.pk;`
}

// Identifier quoting handled by GORM Dialector
