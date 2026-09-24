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

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
	"github.com/clidey/whodb/core/src/sqlident"
)

// YugabyteDBPlugin reuses the PostgreSQL runtime while using YugabyteDB-safe
// catalog queries for metadata paths that differ from upstream PostgreSQL.
type YugabyteDBPlugin struct {
	PostgresPlugin
}

// GetTableInfoQuery returns a YugabyteDB-compatible table info query.
func (p *YugabyteDBPlugin) GetTableInfoQuery() string {
	return `
		SELECT
			t.table_name,
			t.table_type
		FROM
			information_schema.tables t
		WHERE
			t.table_schema = ?
			AND t.table_type IN ('BASE TABLE', 'VIEW')
		ORDER BY
			t.table_name;
	`
}

// GetTableNameAndAttributes parses YugabyteDB table info rows.
func (p *YugabyteDBPlugin) GetTableNameAndAttributes(rows *sql.Rows) (string, []engine.Record) {
	var tableName, tableType string
	if err := rows.Scan(&tableName, &tableType); err != nil {
		log.WithError(err).Error("Failed to scan YugabyteDB table info row data")
		return "", nil
	}

	return tableName, []engine.Record{
		{Key: "Type", Value: tableType},
	}
}

// GetPrimaryKeyColQuery returns primary key columns through information_schema.
func (p *YugabyteDBPlugin) GetPrimaryKeyColQuery() string {
	return `
		SELECT
			kcu.column_name
		FROM
			information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
			ON tc.constraint_name = kcu.constraint_name
			AND tc.table_schema = kcu.table_schema
			AND tc.table_name = kcu.table_name
		WHERE
			tc.constraint_type = 'PRIMARY KEY'
			AND tc.table_schema = ?
			AND tc.table_name = ?
		ORDER BY
			kcu.ordinal_position;
	`
}

// NewYugabyteDBPlugin creates a YugabyteDB plugin that reuses the PostgreSQL
// connection and query runtime with YugabyteDB-specific metadata hooks.
func NewYugabyteDBPlugin() *engine.Plugin {
	yugabyteDBPlugin := &YugabyteDBPlugin{}
	yugabyteDBPlugin.ConfigureIdentifierQuoting(sqlident.DoubleQuote)
	yugabyteDBPlugin.Type = engine.DatabaseType_YugabyteDB
	yugabyteDBPlugin.PluginFunctions = yugabyteDBPlugin
	yugabyteDBPlugin.GormPluginFunctions = yugabyteDBPlugin
	return &yugabyteDBPlugin.Plugin
}

func init() {
	engine.RegisterPlugin(NewYugabyteDBPlugin())
}
