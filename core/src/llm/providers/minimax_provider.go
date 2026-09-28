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

package providers

import "strings"

const (
	MiniMax_LLMType LLMType = "MiniMax"
)

// MiniMaxDefaultEndpoint is the global OpenAI-compatible base URL for MiniMax.
// Deployments in mainland China can point WHODB_MINIMAX_ENDPOINT at
// https://api.minimaxi.com/v1 instead; both regions speak the same protocol.
const MiniMaxDefaultEndpoint = "https://api.minimax.io/v1"

type MiniMaxProvider struct {
	*OpenAICompatibleProvider
}

// NewMiniMaxProvider creates a first-class MiniMax provider backed by the
// OpenAI-compatible adapter.
func NewMiniMaxProvider() *MiniMaxProvider {
	return &MiniMaxProvider{
		OpenAICompatibleProvider: NewOpenAICompatibleProvider(MiniMax_LLMType, MiniMaxDefaultEndpoint),
	}
}

// GetSupportedModels filters the shared model catalog to MiniMax chat models.
func (p *MiniMaxProvider) GetSupportedModels(config *ProviderConfig) ([]string, error) {
	models, err := p.OpenAICompatibleProvider.GetSupportedModels(config)
	if err != nil {
		return nil, err
	}

	chatModels := make([]string, 0, len(models))
	for _, model := range models {
		if strings.HasPrefix(model, "MiniMax-M3") || strings.HasPrefix(model, "MiniMax-M2.7") {
			chatModels = append(chatModels, model)
		}
	}
	return chatModels, nil
}
