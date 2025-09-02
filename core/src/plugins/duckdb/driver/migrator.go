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

package duckdb

import (
	"gorm.io/gorm/migrator"
)

type Migrator struct {
	migrator.Migrator
}

func (m Migrator) HasTable(name string) bool {
	var count int64
	m.DB.Raw("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'main' AND table_name = ?", name).Row().Scan(&count)
	return count > 0
}

func (m Migrator) HasColumn(tableName, columnName string) bool {
	var count int64
	m.DB.Raw("SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = 'main' AND table_name = ? AND column_name = ?", tableName, columnName).Row().Scan(&count)
	return count > 0
}

func (m Migrator) HasIndex(tableName, indexName string) bool {
	// DuckDB doesn't have a standard way to check for indexes in information_schema
	// For now, return false to let GORM handle it
	return false
}

func (m Migrator) HasConstraint(tableName, constraintName string) bool {
	var count int64
	m.DB.Raw("SELECT COUNT(*) FROM information_schema.table_constraints WHERE table_schema = 'main' AND table_name = ? AND constraint_name = ?", tableName, constraintName).Row().Scan(&count)
	return count > 0
}

func (m Migrator) CurrentDatabase() (name string) {
	m.DB.Raw("SELECT current_schema()").Row().Scan(&name)
	return
}