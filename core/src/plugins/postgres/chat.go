// Licensed to Clidey Limited under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Clidey Limited licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package postgres

import (
	"fmt"
	"strings"

	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/llm"
)

func (p *PostgresPlugin) Chat(config *engine.PluginConfig, schema string, model string, previousConversation string, query string) ([]*engine.ChatMessage, error) {
	db, err := DB(config)
	if err != nil {
		return nil, err
	}
	sqlDb, err := db.DB()
	if err != nil {
		return nil, err
	}
	defer sqlDb.Close()

	tableFields, err := getTableSchema(db, schema)
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

	completeQuery := fmt.Sprintf(common.RawSQLQueryPrompt, "Postgres", schema, context, previousConversation, query, "Postgres")

	response, err := llm.Instance(config).Complete(completeQuery, llm.LLMModel(model), nil)
	if err != nil {
		return nil, err
	}

	chats := common.ExtractCodeFromResponse(*response)
	chatMessages := []*engine.ChatMessage{}
	for _, chat := range chats {
		var result *engine.GetRowsResult
		chatType := "message"
		if chat.Type == "sql" {
			rowResult, err := p.RawExecute(config, chat.Text)
			if err != nil {
				return nil, err
			}
			chatType = "sql"
			result = rowResult
		}
		chatMessages = append(chatMessages, &engine.ChatMessage{
			Type:   chatType,
			Result: result,
			Text:   chat.Text,
		})
	}

	return chatMessages, nil
}
