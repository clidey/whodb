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

	if instance, ok := llmInstance[llmType]; ok {
		// Update the API key if it has changed
		instance.APIKey = config.ExternalModel.Token
		return instance
	}
	instance := &LLMClient{
		Type:   llmType,
		APIKey: config.ExternalModel.Token,
	}
	llmInstance[llmType] = instance
	return instance
}
