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

import (
	"errors"
)

// OpenAICompatibleProvider implements first-class providers that expose an OpenAI-compatible API.
type OpenAICompatibleProvider struct {
	providerType    LLMType
	defaultEndpoint string
	modelsEndpoint  string
}

// NewOpenAICompatibleProvider creates a provider backed by the OpenAI-compatible adapter.
func NewOpenAICompatibleProvider(providerType LLMType, defaultEndpoint string) *OpenAICompatibleProvider {
	return NewOpenAICompatibleProviderWithModelsEndpoint(providerType, defaultEndpoint, "")
}

// NewOpenAICompatibleProviderWithModelsEndpoint creates a provider with a custom model discovery endpoint.
func NewOpenAICompatibleProviderWithModelsEndpoint(providerType LLMType, defaultEndpoint string, modelsEndpoint string) *OpenAICompatibleProvider {
	return &OpenAICompatibleProvider{
		providerType:    providerType,
		defaultEndpoint: defaultEndpoint,
		modelsEndpoint:  modelsEndpoint,
	}
}

// GetType returns the provider type.
func (p *OpenAICompatibleProvider) GetType() LLMType {
	return p.providerType
}

// GetProtocol returns "openai" because this provider uses the OpenAI-compatible protocol.
func (p *OpenAICompatibleProvider) GetProtocol() string {
	return "openai"
}

// GetDefaultEndpoint returns the provider's default OpenAI-compatible endpoint.
func (p *OpenAICompatibleProvider) GetDefaultEndpoint() string {
	return p.defaultEndpoint
}

// ValidateConfig validates the provider configuration.
func (p *OpenAICompatibleProvider) ValidateConfig(config *ProviderConfig) error {
	if config.APIKey == "" {
		return errors.New("API key is required")
	}
	if config.Endpoint == "" {
		config.Endpoint = p.GetDefaultEndpoint()
	}
	return nil
}

// GetSupportedModels fetches models from the provider's OpenAI-compatible /models endpoint.
func (p *OpenAICompatibleProvider) GetSupportedModels(config *ProviderConfig) ([]string, error) {
	if err := p.ValidateConfig(config); err != nil {
		return nil, err
	}
	if p.modelsEndpoint != "" && config.Endpoint == p.GetDefaultEndpoint() {
		return fetchOpenAICompatibleModelsFromURL(p.modelsEndpoint, config.APIKey, string(p.providerType))
	}
	return fetchOpenAICompatibleModels(config.Endpoint, config.APIKey, string(p.providerType))
}

// CreateBAMLClient creates BAML client type and options for the provider.
func (p *OpenAICompatibleProvider) CreateBAMLClient(config *ProviderConfig, model string) (string, map[string]any, error) {
	if err := p.ValidateConfig(config); err != nil {
		return "", nil, err
	}

	return "openai-generic", map[string]any{
		"base_url":     config.Endpoint,
		"api_key":      config.APIKey,
		"model":        model,
		"default_role": "user",
	}, nil
}
