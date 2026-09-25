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

package mysql

import (
	"gorm.io/gorm"

	gorm_plugin "github.com/clidey/whodb/core/src/plugins/gorm"
)

// MySQLSQLBuilder extends the base SQLBuilder with MySQL-specific behavior
type MySQLSQLBuilder struct {
	*gorm_plugin.SQLBuilder
}

// NewMySQLSQLBuilder creates a new MySQL-specific SQL builder
func NewMySQLSQLBuilder(db *gorm.DB, plugin gorm_plugin.GormPluginFunctions) gorm_plugin.SQLBuilderInterface {
	msb := &MySQLSQLBuilder{
		SQLBuilder: gorm_plugin.NewSQLBuilder(db, plugin),
	}
	msb.SetSelf(msb) // Set self reference to enable polymorphic calls
	return msb
}

// QualifiedTableName renders a table in the current MySQL database.
func (msb *MySQLSQLBuilder) QualifiedTableName(_ string, table string) (string, error) {
	return msb.SQLBuilder.QualifiedTableName("", table)
}
