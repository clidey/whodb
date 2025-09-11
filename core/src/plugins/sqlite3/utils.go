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

package sqlite3

import (
	"database/sql/driver"
	"fmt"
	"strings"
)

// DateTimeString is a custom type that stores datetime values as plain strings
// without any parsing or formatting. This is needed because SQLite stores
// datetime values as TEXT and we want to preserve the exact format.
type DateTimeString string

// Scan implements sql.Scanner interface to read datetime values as strings
func (ds *DateTimeString) Scan(value interface{}) error {
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
		// For any other type (including time.Time), convert to string
		// This handles the case where the SQLite driver has already parsed the datetime
		str := fmt.Sprintf("%v", value)
		*ds = DateTimeString(str)
	}
	return nil
}

// Value implements driver.Valuer interface to write datetime values as strings
func (ds DateTimeString) Value() (driver.Value, error) {
	return string(ds), nil
}

func (p *Sqlite3Plugin) ConvertStringValueDuringMap(value, columnType string) (interface{}, error) {
	return value, nil
}

// ConvertStringValue overrides the base GORM implementation to preserve datetime strings
func (p *Sqlite3Plugin) ConvertStringValue(value, columnType string) (interface{}, error) {
	// For datetime types, preserve the original string value
	normalizedType := strings.ToUpper(columnType)
	if normalizedType == "DATE" || normalizedType == "DATETIME" || normalizedType == "TIMESTAMP" {
		return value, nil
	}
	// For non-datetime types, delegate to the base GORM implementation
	return p.GormPlugin.ConvertStringValue(value, columnType)
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

func (p *Sqlite3Plugin) GetColTypeQuery() string {
	// Order by cid to preserve column definition order
	return `SELECT name AS column_name, type AS data_type FROM pragma_table_info(?) ORDER BY cid`
}

// Identifier quoting handled by GORM Dialector
