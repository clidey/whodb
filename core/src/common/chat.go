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

package common

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/clidey/whodb/core/src/engine"
)

const RawSQLQueryPrompt = `You are a %v SQL query expert with advanced analytical capabilities. You have access to the following database details:

Schema:  
%v  

Tables and Fields:  
%v  

### Instructions:
Based on the user's input, generate a structured response in JSON array format in one line that includes the following fields:
- **type**: ` + "`\"sql\"`" + ` for SQL queries, ` + "`\"message\"`" + ` for textual responses.
- **operation**: 
  - ` + "`\"get\"`" + ` for SELECT queries.
  - ` + "`\"insert\"`" + ` for INSERT queries.
  - ` + "`\"update\"`" + ` for UPDATE queries.
  - ` + "`\"delete\"`" + ` for DELETE queries.
  - ` + "`\"text\"`" + ` for general text responses.

- **text**: The actual SQL query or response text (response should not contain data - always return a query for data).

### Query Categorization:
- **GET (Retrieve Data):** Execute SELECT queries.
- **INSERT (Insert Data):** Execute INSERT queries.
- **UPDATE (Modify Data):** Execute UPDATE queries with safe WHERE clauses.
- **DELETE (Remove Data):** Execute DELETE queries while ensuring responsible constraints.

### Rules
- Ensure that the JSON array is valid and not formatted - return as a single line inside ` + "```json " + `wrappers
- If multiple jsons are return - return them with separate ` + "```json" + `
- Do not stringify the JSON
- If the query is going to be too large or unpredictable, convey that to the user
- If the query does not make sense as one query, split it into multiple queries
- SQL generated should be valid
- When referencing tables in the SQL query, always include the schema
- Include your explanation as text, if needed.
- Before proceeding with sensitive actions like delete, prompt the user to confirm. Do not provide any valid SQL queries until confirmed.

### Context Consideration:
Previous Conversation:  
%v

### New User Prompt:  
%v

### Expected Response
` + "```" + `json
[{"type":"","operation":"","text":""},...]` + "```"

type RawExecutePlugin interface {
	RawExecute(config *engine.PluginConfig, query string) (*engine.GetRowsResult, error)
}

func SQLChat(response string, config *engine.PluginConfig, plugin RawExecutePlugin) ([]*engine.ChatMessage, error) {
	if !strings.Contains(response, "```json") {
		return nil, errors.New("please try again as there was a problem in processing")
	}
	response = strings.Split(response, "```json")[1]
	response = strings.Split(response, "```")[0]

	var parsedResponses []map[string]any
	err := json.Unmarshal([]byte(response), &parsedResponses)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response JSON: %v", err)
	}

	var chatMessages []*engine.ChatMessage

	for _, chat := range parsedResponses {
		chatType, _ := chat["type"].(string)
		operation, _ := chat["operation"].(string)
		text, _ := chat["text"].(string)

		message := &engine.ChatMessage{
			Type: chatType,
			Text: text,
		}

		if chatType == "sql" {
			result, execErr := plugin.RawExecute(config, text)
			if execErr != nil {
				message.Type = "error"
				message.Text = execErr.Error()
			} else {
				switch operation {
				case "":
				case "get":
					message.Type = "sql:get"
				case "insert":
					message.Type = "sql:insert"
				case "update":
					message.Type = "sql:update"
				case "delete":
					message.Type = "sql:delete"
				default:
					message.Type = "sql"
				}
			}

			message.Result = result
		}

		chatMessages = append(chatMessages, message)
	}

	return chatMessages, nil
}
