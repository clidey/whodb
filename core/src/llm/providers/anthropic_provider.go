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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/clidey/whodb/core/src/log"
)

const (
	Anthropic_LLMType LLMType = "Anthropic"
)

// AnthropicProvider implements the AIProvider interface for Anthropic's Claude API.
type AnthropicProvider struct{}

// NewAnthropicProvider creates a new Anthropic provider instance.
func NewAnthropicProvider() *AnthropicProvider {
	return &AnthropicProvider{}
}

// GetType returns the provider type.
func (p *AnthropicProvider) GetType() LLMType {
	return Anthropic_LLMType
}

// GetName returns the provider name.
func (p *AnthropicProvider) GetName() string {
	return "Anthropic"
}

// RequiresAPIKey returns true as Anthropic requires an API key.
func (p *AnthropicProvider) RequiresAPIKey() bool {
	return true
}

// GetDefaultEndpoint returns the default Anthropic API endpoint.
func (p *AnthropicProvider) GetDefaultEndpoint() string {
	return "https://api.anthropic.com/v1"
}

// ValidateConfig validates the provider configuration.
func (p *AnthropicProvider) ValidateConfig(config *ProviderConfig) error {
	if config.APIKey == "" {
		return errors.New("API key is required for Anthropic")
	}
	if config.Endpoint == "" {
		config.Endpoint = p.GetDefaultEndpoint()
	}
	return nil
}

// GetSupportedModels fetches the list of available models from Anthropic.
func (p *AnthropicProvider) GetSupportedModels(config *ProviderConfig) ([]string, error) {
	if err := p.ValidateConfig(config); err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/models", config.Endpoint)
	headers := map[string]string{
		"x-api-key":         config.APIKey,
		"anthropic-version": "2023-06-01",
		"Content-Type":      "application/json",
	}

	resp, err := sendHTTPRequest("GET", url, nil, headers)
	if err != nil {
		log.WithError(err).Errorf("Failed to fetch models from Anthropic at %s", url)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Errorf("Anthropic models endpoint returned non-OK status: %d, body: %s", resp.StatusCode, string(body))
		return nil, fmt.Errorf("failed to fetch models: %s", string(body))
	}

	var modelsResp struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		log.WithError(err).Error("Failed to decode Anthropic models response")
		return nil, err
	}

	// Filter to only include claude-* models (all Claude models are chat-compatible)
	var models []string
	for _, model := range modelsResp.Data {
		if strings.HasPrefix(model.ID, "claude-") {
			models = append(models, model.ID)
		}
	}
	return models, nil
}

// GetBAMLClientType returns the BAML client type for Anthropic.
func (p *AnthropicProvider) GetBAMLClientType() string {
	return "anthropic"
}

// CreateBAMLClientOptions creates BAML client options for Anthropic.
func (p *AnthropicProvider) CreateBAMLClientOptions(config *ProviderConfig, model string) (map[string]any, error) {
	if err := p.ValidateConfig(config); err != nil {
		return nil, err
	}

	return map[string]any{
		"model":   model,
		"api_key": config.APIKey,
	}, nil
}
