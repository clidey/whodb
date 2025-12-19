// Copyright 2025 Clidey, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gorm_plugin

import (
	"fmt"
	"strings"

	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/llm"
	"github.com/clidey/whodb/core/src/log"
	"github.com/clidey/whodb/core/src/plugins"
	"gorm.io/gorm"
)

func (p *GormPlugin) Chat(config *engine.PluginConfig, schema string, model string, previousConversation string, query string) ([]*engine.ChatMessage, error) {
	return plugins.WithConnection(config, p.DB, func(db *gorm.DB) ([]*engine.ChatMessage, error) {
		// Get table names from table info query
		rows, err := db.Raw(p.GetTableInfoQuery(), schema).Rows()
		if err != nil {
			log.Logger.WithError(err).Error(fmt.Sprintf("Failed to get tables for chat operation in schema: %s", schema))
			return nil, err
		}
		defer rows.Close()

		helper := NewMigratorHelper(db, p.GormPluginFunctions)
		tableDetails := strings.Builder{}

		for rows.Next() {
			tableName, _ := p.GetTableNameAndAttributes(rows)
			if tableName == "" {
				continue
			}

			// Use GORM migrator to get column types with length info (preserves column order)
			fullTableName := p.FormTableName(schema, tableName)
			orderedColumns, err := helper.GetOrderedColumns(fullTableName)
			if err != nil {
				log.Logger.WithError(err).Warnf("Failed to get column types for table %s in chat", fullTableName)
				continue
			}

			tableDetails.WriteString(fmt.Sprintf("table: %v\n", tableName))
			for _, col := range orderedColumns {
				tableDetails.WriteString(fmt.Sprintf("- %v (%v)\n", col.Name, col.Type))
			}
		}

		context := tableDetails.String()

		completeQuery := fmt.Sprintf(common.RawSQLQueryPrompt, p.Type, schema, context, previousConversation, query)

		response, err := llm.Instance(config).Complete(completeQuery, llm.LLMModel(model), nil)
		if err != nil {
			log.Logger.WithError(err).Error(fmt.Sprintf("Failed to complete LLM query using model %s for schema %s", model, schema))
			return nil, err
		}

		return common.SQLChat(*response, config, p)
	})
}
