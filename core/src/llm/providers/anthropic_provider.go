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
	"bufio"
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
		log.Logger.WithError(err).Errorf("Failed to fetch models from Anthropic at %s", url)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Logger.Errorf("Anthropic models endpoint returned non-OK status: %d, body: %s", resp.StatusCode, string(body))
		return nil, fmt.Errorf("failed to fetch models: %s", string(body))
	}

	var modelsResp struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		log.Logger.WithError(err).Error("Failed to decode Anthropic models response")
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

// Complete sends a completion request to Anthropic.
func (p *AnthropicProvider) Complete(config *ProviderConfig, prompt string, model LLMModel, receiverChan *chan string) (*string, error) {
	if err := p.ValidateConfig(config); err != nil {
		return nil, err
	}

	// Determine max_tokens based on model
	maxTokens := p.getMaxTokensForModel(string(model))

	requestBody, err := json.Marshal(map[string]any{
		"model":      string(model),
		"max_tokens": maxTokens,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	})
	if err != nil {
		log.Logger.WithError(err).Errorf("Failed to marshal Anthropic request body for model %s", model)
		return nil, err
	}

	url := fmt.Sprintf("%s/messages", config.Endpoint)
	headers := map[string]string{
		"x-api-key":         config.APIKey,
		"anthropic-version": "2023-06-01",
		"content-type":      "application/json",
	}

	resp, err := sendHTTPRequest("POST", url, requestBody, headers)
	if err != nil {
		log.Logger.WithError(err).Errorf("Failed to send HTTP request to Anthropic at %s", url)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Logger.Errorf("Anthropic returned non-OK status: %d, body: %s", resp.StatusCode, string(body))
		return nil, errors.New(string(body))
	}

	return p.ParseResponse(resp.Body, receiverChan)
}

// getMaxTokensForModel returns the appropriate max_tokens value for each model.
func (p *AnthropicProvider) getMaxTokensForModel(model string) int {
	switch model {
	case "claude-3-7-sonnet-20250219", "claude-sonnet-4-20250514":
		return 64000
	case "claude-opus-4-20250514":
		return 32000
	case "claude-3-5-sonnet-20241022", "claude-3-5-sonnet-20240620", "claude-3-5-opus-20241022", "claude-3-5-haiku-20241022":
		return 8192
	case "claude-3-opus-20240229", "claude-3-haiku-20240307":
		return 4096
	default:
		return 4096 // Conservative default for unknown models
	}
}

// ParseResponse parses the Anthropic API response.
func (p *AnthropicProvider) ParseResponse(body io.ReadCloser, receiverChan *chan string) (*string, error) {
	responseBuilder := strings.Builder{}
	reader := bufio.NewReader(body)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				log.Logger.WithError(err).Error("Failed to read line from Anthropic response")
				return nil, err
			}
		}

		var anthropicResponse struct {
			Content []struct {
				Text string `json:"text"`
				Type string `json:"type"`
			} `json:"content"`
			StopReason string `json:"stop_reason"`
			Usage      struct {
				InputTokens  int `json:"input_tokens"`
				OutputTokens int `json:"output_tokens"`
			} `json:"usage"`
			Role         string  `json:"role"`
			Model        string  `json:"model"`
			ID           string  `json:"id"`
			Type         string  `json:"type"`
			StopSequence *string `json:"stop_sequence,omitempty"`
		}

		if err := json.Unmarshal([]byte(line), &anthropicResponse); err != nil {
			log.Logger.WithError(err).Errorf("Failed to unmarshal Anthropic response line: %s", line)
			return nil, err
		}

		for _, content := range anthropicResponse.Content {
			if receiverChan != nil {
				*receiverChan <- content.Text
			}
			if _, err := responseBuilder.WriteString(content.Text); err != nil {
				log.Logger.WithError(err).Error("Failed to write to Anthropic response builder")
				return nil, err
			}
		}

		if anthropicResponse.StopReason == "end_turn" {
			response := responseBuilder.String()
			return &response, nil
		}
	}
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
