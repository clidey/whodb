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
	"fmt"
	gorm_plugin "github.com/clidey/whodb/core/src/plugins/gorm"
	"gorm.io/gorm"
)

// SQLiteSQLBuilder embeds the generic SQLBuilder and overrides methods for SQLite-specific syntax.
type SQLiteSQLBuilder struct {
	*gorm_plugin.SQLBuilder
}

// NewSQLiteSQLBuilder creates a new SQL builder for SQLite.
func NewSQLiteSQLBuilder(db *gorm.DB, plugin gorm_plugin.GormPluginFunctions) *SQLiteSQLBuilder {
	baseBuilder := gorm_plugin.NewSQLBuilder(db, plugin)
	builder := &SQLiteSQLBuilder{
		SQLBuilder: baseBuilder,
	}
	baseBuilder.SetSelf(builder)
	return builder
}

// PragmaQuery builds a SQLite PRAGMA query.
func (sb *SQLiteSQLBuilder) PragmaQuery(pragma, table string) (string, error) {
	allowedPragmas := map[string]bool{
		"table_info":       true,
		"index_list":       true,
		"index_info":       true,
		"foreign_key_list": true,
		"table_list":       true,
	}

	if !allowedPragmas[pragma] {
		return "", fmt.Errorf("disallowed pragma: %s", pragma)
	}

	return fmt.Sprintf("PRAGMA %s(%s)", pragma, sb.QuoteIdentifier(table)), nil
}
