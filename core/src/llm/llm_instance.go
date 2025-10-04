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
    "sync"

    "github.com/clidey/whodb/core/src/engine"
)

var (
    llmInstance map[string]*LLMClient
    llmMu       sync.Mutex
)

func Instance(config *engine.PluginConfig) *LLMClient {
    llmMu.Lock()
    if llmInstance == nil {
        llmInstance = make(map[string]*LLMClient)
    }
    llmMu.Unlock()

	// Always use provider system
	if config.ExternalModel == nil || config.ExternalModel.ProviderId == "" {
		// Return a default Ollama instance if no provider specified
		registry := GetProviderRegistry()
		providers := registry.GetProvidersByType(Ollama_LLMType)
		if len(providers) > 0 {
			return &LLMClient{
				Config: providers[0],
			}
		}
		// Return empty client if no providers available
		return &LLMClient{}
	}

	return InstanceWithProvider(config.ExternalModel.ProviderId)
}

// InstanceWithProvider creates an LLMClient using a provider configuration
func InstanceWithProvider(providerId string) *LLMClient {
    llmMu.Lock()
    if llmInstance == nil {
        llmInstance = make(map[string]*LLMClient)
    }

	// Use provider ID as cache key
	cacheKey := "provider:" + providerId

    if instance, ok := llmInstance[cacheKey]; ok {
        llmMu.Unlock()
        return instance
    }

	// Get provider from registry
    registry := GetProviderRegistry()
    providerConfig, err := registry.GetProvider(providerId)
    if err != nil {
        // Return empty client if provider not found
        llmMu.Unlock()
        return &LLMClient{}
    }

    instance := &LLMClient{
        Config: providerConfig,
    }
    llmInstance[cacheKey] = instance
    llmMu.Unlock()
    return instance
}
