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
	"encoding/json"
	"errors"
)

// ProviderConfig represents a configured AI provider instance
type ProviderConfig struct {
	ID                   string                 `json:"id"`
	Name                 string                 `json:"name"`
	Type                 LLMType                `json:"type"`
	BaseURL              string                 `json:"baseURL,omitempty"`
	APIKey               string                 `json:"apiKey,omitempty"`
	Settings             map[string]interface{} `json:"settings,omitempty"`
	IsEnvironmentDefined bool                   `json:"isEnvironmentDefined"`
	IsUserDefined        bool                   `json:"isUserDefined"`
}

// ProviderSettings defines common settings across providers
type ProviderSettings struct {
    Temperature      *float32 `json:"temperature,omitempty"`
    MaxTokens        *int     `json:"max_tokens,omitempty"`
    TopP             *float32 `json:"top_p,omitempty"`
    TopK             *int     `json:"top_k,omitempty"`
    FrequencyPenalty *float32 `json:"frequency_penalty,omitempty"`
    PresencePenalty  *float32 `json:"presence_penalty,omitempty"`
    RepeatPenalty    *float32 `json:"repeat_penalty,omitempty"`
    // Model can be overridden per request
    Model string `json:"model,omitempty"`
}

// GetSettings parses the settings map into a ProviderSettings struct
func (p *ProviderConfig) GetSettings() (*ProviderSettings, error) {
	if p.Settings == nil {
		return &ProviderSettings{}, nil
	}

	// Marshal to JSON then unmarshal to struct
	data, err := json.Marshal(p.Settings)
	if err != nil {
		return nil, err
	}

	var settings ProviderSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, err
	}

	return &settings, nil
}

// ValidateProvider checks if the provider configuration is valid
func (p *ProviderConfig) Validate() error {
	if p.ID == "" {
		return errors.New("provider ID is required")
	}
	if p.Name == "" {
		return errors.New("provider name is required")
	}
	if p.Type == "" {
		return errors.New("provider type is required")
	}

	// Validate type-specific requirements
	switch p.Type {
	case ChatGPT_LLMType, Anthropic_LLMType:
		if p.APIKey == "" && !p.IsEnvironmentDefined {
			return errors.New("API key is required for " + string(p.Type))
		}
	case OpenAICompatible_LLMType:
		if p.BaseURL == "" {
			return errors.New("base URL is required for OpenAI-compatible providers")
		}
		if p.APIKey == "" && !p.IsEnvironmentDefined {
			return errors.New("API key is required for OpenAI-compatible providers")
		}
	case Ollama_LLMType:
		// Ollama doesn't require API key
	default:
		// Future providers like AWS Bedrock, GCP Vertex
	}

	return nil
}

// ApplyDefaults applies default settings based on provider type
func (p *ProviderConfig) ApplyDefaults() {
	if p.Settings == nil {
		p.Settings = make(map[string]interface{})
	}

	// Apply type-specific defaults
	switch p.Type {
	case ChatGPT_LLMType, OpenAICompatible_LLMType:
		if p.BaseURL == "" && p.Type == ChatGPT_LLMType {
			p.BaseURL = "https://api.openai.com/v1"
		}
		// Set default temperature if not specified
		if _, ok := p.Settings["temperature"]; !ok {
			p.Settings["temperature"] = 0.7
		}
		if _, ok := p.Settings["max_tokens"]; !ok {
			p.Settings["max_tokens"] = 2048
		}
	case Anthropic_LLMType:
		if p.BaseURL == "" {
			p.BaseURL = "https://api.anthropic.com/v1"
		}
		if _, ok := p.Settings["temperature"]; !ok {
			p.Settings["temperature"] = 0.7
		}
		if _, ok := p.Settings["max_tokens"]; !ok {
			p.Settings["max_tokens"] = 4096
		}
	case Ollama_LLMType:
		if p.BaseURL == "" {
			p.BaseURL = "http://localhost:11434"
		}
		if _, ok := p.Settings["temperature"]; !ok {
			p.Settings["temperature"] = 0.7
		}
	}
}

// Clone creates a deep copy of the provider config
func (p *ProviderConfig) Clone() *ProviderConfig {
	clone := &ProviderConfig{
		ID:                   p.ID,
		Name:                 p.Name,
		Type:                 p.Type,
		BaseURL:              p.BaseURL,
		APIKey:               p.APIKey,
		IsEnvironmentDefined: p.IsEnvironmentDefined,
		IsUserDefined:        p.IsUserDefined,
	}

	if p.Settings != nil {
		clone.Settings = make(map[string]interface{})
		for k, v := range p.Settings {
			clone.Settings[k] = v
		}
	}

	return clone
}
