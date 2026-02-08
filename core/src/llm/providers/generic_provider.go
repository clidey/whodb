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
	"fmt"
)

// GenericProvider implements the AIProvider interface for any OpenAI-compatible API.
// This enables support for Mistral, Cohere, Google Gemini, and other providers
// that implement the OpenAI API specification.
type GenericProvider struct {
	providerId string
	name       string
	models     []string
	clientType string
}

// NewGenericProvider creates a new generic provider instance.
func NewGenericProvider(providerId, name string, models []string, clientType string) *GenericProvider {
	if clientType == "" {
		clientType = "openai-generic" // Default to OpenAI-compatible
	}
	return &GenericProvider{
		providerId: providerId,
		name:       name,
		models:     models,
		clientType: clientType,
	}
}

// GetType returns the provider type (unique per generic provider instance).
func (p *GenericProvider) GetType() LLMType {
	return LLMType(p.providerId)
}

// GetName returns the provider name.
func (p *GenericProvider) GetName() string {
	return p.name
}

// RequiresAPIKey returns true as most generic providers require an API key.
// Providers that don't need keys can leave APIKey empty in config.
func (p *GenericProvider) RequiresAPIKey() bool {
	return false // Allow optional API key for flexibility
}

// GetDefaultEndpoint returns an empty string as generic providers must specify their endpoint.
func (p *GenericProvider) GetDefaultEndpoint() string {
	return "" // No default - must be configured
}

// ValidateConfig validates the provider configuration.
func (p *GenericProvider) ValidateConfig(config *ProviderConfig) error {
	if config.Endpoint == "" {
		return fmt.Errorf("endpoint is required for generic provider %s", p.name)
	}
	return nil
}

// GetSupportedModels returns the pre-configured list of models.
// Generic providers don't support dynamic model discovery - models must be specified in config.
func (p *GenericProvider) GetSupportedModels(config *ProviderConfig) ([]string, error) {
	return p.models, nil
}

// GetBAMLClientType returns the BAML client type for this provider.
func (p *GenericProvider) GetBAMLClientType() string {
	return p.clientType
}

// CreateBAMLClientOptions creates BAML client options for the generic provider.
func (p *GenericProvider) CreateBAMLClientOptions(config *ProviderConfig, model string) (map[string]any, error) {
	if err := p.ValidateConfig(config); err != nil {
		return nil, err
	}

	opts := map[string]any{
		"base_url":           config.Endpoint,
		"model":              model,
		"default_role":       "user",
		"request_timeout_ms": 60000,
	}

	// Only include api_key if provided
	if config.APIKey != "" {
		opts["api_key"] = config.APIKey
	}

	return opts, nil
}
