package gorm_plugin

import (
	"fmt"
	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/llm"
	"github.com/clidey/whodb/core/src/plugins"
	"gorm.io/gorm"
	"strings"
)

func (p *GormPlugin) Chat(config *engine.PluginConfig, schema string, model string, previousConversation string, query string) ([]*engine.ChatMessage, error) {
	return plugins.WithConnection[[]*engine.ChatMessage](config, p.DB, func(db *gorm.DB) ([]*engine.ChatMessage, error) {
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
