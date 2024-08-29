package llm

import (
	"github.com/clidey/whodb/core/src/engine"
)

var llmInstance map[LLMType]*LLMClient

func Instance(config *engine.PluginConfig) *LLMClient {
	if llmInstance == nil {
		llmInstance = make(map[LLMType]*LLMClient)
	}

	llmType := LLMType(config.ExternalModel.Type)

	if _, ok := llmInstance[llmType]; ok {
		return llmInstance[llmType]
	}
	instance := &LLMClient{
		Type:   llmType,
		APIKey: config.ExternalModel.Token,
	}
	llmInstance[llmType] = instance
	return instance
}
