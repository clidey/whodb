/*
 * Copyright 2025 Clidey, Inc.
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

package llm

import (
	"github.com/clidey/whodb/core/src/env"
	"github.com/clidey/whodb/core/src/llm/providers"
	"github.com/clidey/whodb/core/src/log"
)

func init() {
	// Register built-in providers
	providers.RegisterProvider(providers.NewOpenAIProvider())
	providers.RegisterProvider(providers.NewAnthropicProvider())
	providers.RegisterProvider(providers.NewOllamaProvider())
}

// RegisterGenericProviders registers generic AI providers from environment configuration.
// This is called by the env package after parsing generic provider configs and by EE providers.
// It registers the provider with both the LLM provider system (for backend operations)
// and the env.GenericProviders list (for frontend display).
func RegisterGenericProviders(name string, providerId string, models []string, clientType string, baseURL string, apiKey string) {
	provider := providers.NewGenericProvider(providerId, name, models, clientType)
	providers.RegisterProvider(provider)

	// Also add to env.GenericProviders so it shows up in the frontend
	env.AddGenericProvider(env.GenericProviderConfig{
		ProviderId: providerId,
		Name:       name,
		ClientType: clientType,
		BaseURL:    baseURL,
		APIKey:     apiKey,
		Models:     models,
	})
}

// Type aliases for backward compatibility with llm package
type LLMType = providers.LLMType
type LLMModel = providers.LLMModel

const (
	Ollama_LLMType    = providers.Ollama_LLMType
	OpenAI_LLMType    = providers.OpenAI_LLMType
	Anthropic_LLMType = providers.Anthropic_LLMType
	// Deprecated: Use OpenAI_LLMType instead
	ChatGPT_LLMType LLMType = "ChatGPT"
)

// NormalizeLLMType normalizes LLM type strings for backward compatibility.
// Maps deprecated "ChatGPT" to "OpenAI".
func NormalizeLLMType(typeStr string) LLMType {
	if typeStr == "ChatGPT" {
		log.Logger.Warn("'ChatGPT' provider name is deprecated, use 'OpenAI' instead")
		return OpenAI_LLMType
	}
	return LLMType(typeStr)
}

type LLMClient struct {
	Type      LLMType
	APIKey    string
	ProfileId string
}

func (c *LLMClient) Complete(prompt string, model LLMModel, receiverChan *chan string) (*string, error) {
	// Normalize type for backward compatibility
	normalizedType := NormalizeLLMType(string(c.Type))

	// Get provider from registry
	provider, err := providers.GetProvider(normalizedType)
	if err != nil {
		log.Logger.WithError(err).Errorf("Provider not found for type: %s", normalizedType)
		return nil, err
	}

	// Build provider config with endpoint from environment
	config := &providers.ProviderConfig{
		Type:     normalizedType,
		APIKey:   c.APIKey,
		Endpoint: getEndpointForProvider(normalizedType),
	}

	// Use provider to complete the request
	return provider.Complete(config, prompt, model, receiverChan)
}

// getEndpointForProvider returns the appropriate endpoint for a provider type
func getEndpointForProvider(providerType LLMType) string {
	switch providerType {
	case OpenAI_LLMType:
		return env.GetOpenAIEndpoint()
	case Anthropic_LLMType:
		return env.GetAnthropicEndpoint()
	case Ollama_LLMType:
		return env.GetOllamaEndpoint()
	default:
		// For generic providers, look up endpoint from environment configuration
		for _, provider := range env.GenericProviders {
			if providers.LLMType(provider.ProviderId) == providerType {
				return provider.BaseURL
			}
		}
		log.Logger.Warnf("No endpoint found for provider type: %s", providerType)
		return ""
	}
}

func (c *LLMClient) GetSupportedModels() ([]string, error) {
	// Normalize type for backward compatibility
	normalizedType := NormalizeLLMType(string(c.Type))

	// Get provider from registry
	provider, err := providers.GetProvider(normalizedType)
	if err != nil {
		log.Logger.WithError(err).Errorf("Provider not found for type: %s", normalizedType)
		return nil, err
	}

	// Build provider config with endpoint from environment
	config := &providers.ProviderConfig{
		Type:     normalizedType,
		APIKey:   c.APIKey,
		Endpoint: getEndpointForProvider(normalizedType),
	}

	// Use provider to get supported models
	return provider.GetSupportedModels(config)
}

