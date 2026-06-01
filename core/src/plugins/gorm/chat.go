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

package gorm_plugin

import (
	ctx "context"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/clidey/whodb/core/src/bamlconfig"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
	"github.com/clidey/whodb/core/src/plugins"
	"github.com/clidey/whodb/core/src/source"
)

func (p *GormPlugin) buildChatTableContext(config *engine.PluginConfig, db *gorm.DB, schema string) (string, error) {
	// Get table names from table info query
	rows, err := db.Raw(p.GetTableInfoQuery(), schema).Rows()
	if err != nil {
		log.WithError(err).Error("Failed to get tables for chat operation in schema: " + schema)
		return "", err
	}
	defer rows.Close()

	tableDetails := strings.Builder{}

	for rows.Next() {
		tableName, _ := p.GetTableNameAndAttributes(rows)
		if tableName == "" {
			continue
		}

		fmt.Fprintf(&tableDetails, "table: %v\n", tableName)

		// Use the plugin column lookup so database-specific overrides
		// (QuestDB, ClickHouse, SQLite, etc.) are reused for chat context too.
		orderedColumns, err := p.PluginFunctions.GetColumnsForTable(config, schema, tableName)
		if err != nil {
			log.WithError(err).Warnf("Failed to get column types for table %s in chat", p.FormTableName(schema, tableName))
			continue
		}

		for _, col := range orderedColumns {
			fmt.Fprintf(&tableDetails, "- %v (%v)\n", col.Name, col.Type)
		}
	}

	return tableDetails.String(), nil
}

func (p *GormPlugin) Chat(config *engine.PluginConfig, schema string, previousConversation string, query string) ([]*engine.ChatMessage, error) {
	return plugins.WithConnection(config, p.DB, func(db *gorm.DB) ([]*engine.ChatMessage, error) {
		tableContext, err := p.buildChatTableContext(config, db, schema)
		if err != nil {
			return nil, err
		}

		// Use BAML for structured SQL query generation
		callCtx := ctx.Background()

		return bamlconfig.ExecuteChatQuery(
			callCtx,
			string(p.Type),
			schema,
			tableContext,
			previousConversation,
			query,
			config.ExternalModel,
			bamlconfig.ChatQueryExecutorFunc(func(_ ctx.Context, query string, params ...any) (*source.RowsResult, error) {
				return p.RawExecute(config, query, params...)
			}),
		)
	})
}
