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

// Complete sends a completion request to the generic provider.
// Uses OpenAI-compatible API format.
func (p *GenericProvider) Complete(config *ProviderConfig, prompt string, model LLMModel, receiverChan *chan string) (*string, error) {
	if err := p.ValidateConfig(config); err != nil {
		return nil, err
	}

	requestBody, err := json.Marshal(map[string]any{
		"model":    string(model),
		"messages": []map[string]string{{"role": "user", "content": prompt}},
		"stream":   receiverChan != nil,
	})
	if err != nil {
		log.Logger.WithError(err).Errorf("Failed to marshal %s request body for model %s", p.name, model)
		return nil, err
	}

	// Generic providers should use OpenAI-compatible /chat/completions endpoint
	url := fmt.Sprintf("%s/chat/completions", strings.TrimSuffix(config.Endpoint, "/"))
	headers := map[string]string{
		"Content-Type": "application/json",
	}

	// Add authorization header if API key is provided
	if config.APIKey != "" {
		headers["Authorization"] = fmt.Sprintf("Bearer %s", config.APIKey)
	}

	resp, err := sendHTTPRequest("POST", url, requestBody, headers)
	if err != nil {
		log.Logger.WithError(err).Errorf("Failed to send HTTP request to %s at %s", p.name, url)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Logger.Errorf("%s returned non-OK status: %d, body: %s", p.name, resp.StatusCode, string(body))
		return nil, fmt.Errorf("%s error: %s", p.name, string(body))
	}

	return p.parseResponse(resp.Body, receiverChan)
}

// parseResponse parses OpenAI-compatible API responses (streaming or non-streaming).
func (p *GenericProvider) parseResponse(body io.ReadCloser, receiverChan *chan string) (*string, error) {
	responseBuilder := strings.Builder{}

	if receiverChan != nil {
		// Streaming response
		reader := bufio.NewReader(body)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					break
				}
				log.Logger.WithError(err).Errorf("Failed to read line from %s streaming response", p.name)
				return nil, err
			}

			if strings.TrimSpace(line) == "" {
				continue
			}

			// Skip SSE data: prefix if present
			line = strings.TrimPrefix(line, "data: ")

			// Skip [DONE] message
			if strings.TrimSpace(line) == "[DONE]" {
				continue
			}

			var completionResponse struct {
				Choices []struct {
					Delta struct {
						Content string `json:"content"`
					} `json:"delta"`
				} `json:"choices"`
				FinishReason string `json:"finish_reason"`
			}

			if err := json.Unmarshal([]byte(line), &completionResponse); err != nil {
				log.Logger.WithError(err).Errorf("Failed to unmarshal %s streaming response line: %s", p.name, line)
				continue // Skip malformed lines instead of failing
			}

			if len(completionResponse.Choices) > 0 {
				content := completionResponse.Choices[0].Delta.Content
				if content != "" {
					*receiverChan <- content
					responseBuilder.WriteString(content)
				}
				if completionResponse.FinishReason == "stop" {
					response := responseBuilder.String()
					return &response, nil
				}
			}
		}

		// Return accumulated response if loop ends without explicit stop
		if responseBuilder.Len() > 0 {
			response := responseBuilder.String()
			return &response, nil
		}
	} else {
		// Non-streaming response
		var completionResponse struct {
			Choices []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			} `json:"choices"`
		}

		if err := json.NewDecoder(body).Decode(&completionResponse); err != nil {
			log.Logger.WithError(err).Errorf("Failed to decode %s non-streaming response", p.name)
			return nil, err
		}

		if len(completionResponse.Choices) > 0 {
			response := completionResponse.Choices[0].Message.Content
			return &response, nil
		}

		return nil, errors.New("no completion response received from " + p.name)
	}

	return nil, nil
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
