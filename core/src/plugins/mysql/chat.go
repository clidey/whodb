package mysql

import (
	"fmt"
	"strings"

	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/llm"
)

func (p *MySQLPlugin) Chat(config *engine.PluginConfig, schema string, model string, previousConversation string, query string) ([]*engine.ChatMessage, error) {
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

	completeQuery := fmt.Sprintf(common.RawSQLQueryPrompt, "MySQL", schema, context, previousConversation, query, "MySQL")

	response, err := llm.Instance(llm.Ollama_LLMType).Complete(completeQuery, llm.LLMModel(model), nil)
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
