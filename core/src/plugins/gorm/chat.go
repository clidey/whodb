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
	"github.com/clidey/whodb/core/src/plugins"
	"gorm.io/gorm"
)

func (p *GormPlugin) Chat(config *engine.PluginConfig, schema string, model string, previousConversation string, query string) ([]*engine.ChatMessage, error) {
	return plugins.WithConnection(config, p.DB, func(db *gorm.DB) ([]*engine.ChatMessage, error) {
		tableFields, err := p.GetTableSchema(db, schema)
		if err != nil {
			return nil, err
		}

		tableDetails := strings.Builder{}
		for tableName, fields := range tableFields {
			tableDetails.WriteString(fmt.Sprintf("table: %v\n", tableName))
			for _, field := range fields {
				tableDetails.WriteString(fmt.Sprintf("- %v (%v)\n", field.Key, field.Value))
			}
		}

		context := tableDetails.String()

		completeQuery := fmt.Sprintf(common.RawSQLQueryPrompt, p.Type, schema, context, previousConversation, query)

		response, err := llm.Instance(config).Complete(completeQuery, llm.LLMModel(model), nil)
		if err != nil {
			return nil, err
		}

		return common.SQLChat(*response, config, p)
	})
}
