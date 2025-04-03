package common

import (
	"encoding/json"
	"fmt"
	"github.com/clidey/whodb/core/src/engine"
	"strings"
)

const RawSQLQueryPrompt = `You are a %v SQL query expert. You have access to the following information:
Schema: %v
Tables and Fields:
%v
Instructions:
Based on the user's input, generate a explanation response with a valid SQL query that will retrieve the required data or execute an action from the database.

Previous Conversation:
%v

User Prompt:
%v

System Prompt:
Generate the SQL query inside ` + "```sql" + ` that corresponds to the user's request. Important note: if you generate multiple queries, provide multiple SQL queries in the SEPARATE quotes.
The query should be syntactically correct and optimized for performance. Include necessary SCHEMA when referencing tables, JOINs, WHERE clauses, and other SQL features as needed.
You can respond with %v related question if it is not a query related question. Speak to the user as "you".`

type RawExecutePlugin interface {
	RawExecute(config *engine.PluginConfig, query string) (*engine.GetRowsResult, error)
}

func SQLChat(response string, config *engine.PluginConfig, plugin RawExecutePlugin) ([]*engine.ChatMessage, error) {
	response = strings.Split(response, "```json")[1]
	response = strings.Split(response, "```")[0]

	var parsedResponses []map[string]interface{}
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
				case "get":
					message.Type = "sql:get"
				case "insert":
					message.Type = "sql:insert"
				case "update":
					message.Type = "sql:update"
				case "delete":
					message.Type = "sql:delete"
				case "line-chart":
					message.Type = "sql:line-chart"
				case "pie-chart":
					message.Type = "sql:pie-chart"
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
