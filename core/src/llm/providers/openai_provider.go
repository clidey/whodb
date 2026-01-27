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
	OpenAI_LLMType LLMType = "OpenAI"
)

// OpenAIProvider implements the AIProvider interface for OpenAI's API.
type OpenAIProvider struct{}

// NewOpenAIProvider creates a new OpenAI provider instance.
func NewOpenAIProvider() *OpenAIProvider {
	return &OpenAIProvider{}
}

// GetType returns the provider type.
func (p *OpenAIProvider) GetType() LLMType {
	return OpenAI_LLMType
}

// GetName returns the provider name.
func (p *OpenAIProvider) GetName() string {
	return "OpenAI"
}

// RequiresAPIKey returns true as OpenAI requires an API key.
func (p *OpenAIProvider) RequiresAPIKey() bool {
	return true
}

// GetDefaultEndpoint returns the default OpenAI API endpoint.
func (p *OpenAIProvider) GetDefaultEndpoint() string {
	return "https://api.openai.com/v1"
}

// ValidateConfig validates the provider configuration.
func (p *OpenAIProvider) ValidateConfig(config *ProviderConfig) error {
	if config.APIKey == "" {
		return errors.New("API key is required for OpenAI")
	}
	if config.Endpoint == "" {
		config.Endpoint = p.GetDefaultEndpoint()
	}
	return nil
}

// GetSupportedModels fetches the list of available models from OpenAI.
func (p *OpenAIProvider) GetSupportedModels(config *ProviderConfig) ([]string, error) {
	if err := p.ValidateConfig(config); err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/models", config.Endpoint)
	headers := map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", config.APIKey),
		"Content-Type":  "application/json",
	}

	resp, err := sendHTTPRequest("GET", url, nil, headers)
	if err != nil {
		log.Logger.WithError(err).Errorf("Failed to fetch models from OpenAI at %s", url)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Logger.Errorf("OpenAI models endpoint returned non-OK status: %d, body: %s", resp.StatusCode, string(body))
		return nil, fmt.Errorf("failed to fetch models: %s", string(body))
	}

	var modelsResp struct {
		Models []struct {
			Name string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		log.Logger.WithError(err).Error("Failed to decode OpenAI models response")
		return nil, err
	}

	// Filter to only include gpt-* models
	models := []string{}
	for _, model := range modelsResp.Models {
		if strings.HasPrefix(model.Name, "gpt-") {
			models = append(models, model.Name)
		}
	}
	return models, nil
}

// Complete sends a completion request to OpenAI.
func (p *OpenAIProvider) Complete(config *ProviderConfig, prompt string, model LLMModel, receiverChan *chan string) (*string, error) {
	if err := p.ValidateConfig(config); err != nil {
		return nil, err
	}

	requestBody, err := json.Marshal(map[string]any{
		"model":    string(model),
		"messages": []map[string]string{{"role": "user", "content": prompt}},
		"stream":   receiverChan != nil,
	})
	if err != nil {
		log.Logger.WithError(err).Errorf("Failed to marshal OpenAI request body for model %s", model)
		return nil, err
	}

	url := fmt.Sprintf("%s/chat/completions", config.Endpoint)
	headers := map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", config.APIKey),
		"Content-Type":  "application/json",
	}

	resp, err := sendHTTPRequest("POST", url, requestBody, headers)
	if err != nil {
		log.Logger.WithError(err).Errorf("Failed to send HTTP request to OpenAI at %s", url)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Logger.Errorf("OpenAI returned non-OK status: %d, body: %s", resp.StatusCode, string(body))
		return nil, errors.New(string(body))
	}

	return p.ParseResponse(resp.Body, receiverChan)
}

// ParseResponse parses the OpenAI API response (streaming or non-streaming).
// Exported for testing.
func (p *OpenAIProvider) ParseResponse(body io.ReadCloser, receiverChan *chan string) (*string, error) {
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
				log.Logger.WithError(err).Error("Failed to read line from OpenAI streaming response")
				return nil, err
			}

			if strings.TrimSpace(line) == "" {
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
				log.Logger.WithError(err).Errorf("Failed to unmarshal OpenAI streaming response line: %s", line)
				return nil, err
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
			log.Logger.WithError(err).Error("Failed to decode OpenAI non-streaming response")
			return nil, err
		}

		if len(completionResponse.Choices) > 0 {
			response := completionResponse.Choices[0].Message.Content
			return &response, nil
		}

		return nil, errors.New("no completion response received from OpenAI")
	}

	return nil, nil
}

// GetBAMLClientType returns the BAML client type for OpenAI.
func (p *OpenAIProvider) GetBAMLClientType() string {
	return "openai"
}

// CreateBAMLClientOptions creates BAML client options for OpenAI.
func (p *OpenAIProvider) CreateBAMLClientOptions(config *ProviderConfig, model string) (map[string]any, error) {
	if err := p.ValidateConfig(config); err != nil {
		return nil, err
	}

	return map[string]any{
		"model":   model,
		"api_key": config.APIKey,
	}, nil
}

// sendHTTPRequest is a helper function to send HTTP requests.
// This is duplicated from http_client.go to avoid circular dependencies.
// TODO: Consider refactoring to a shared utility package.
func sendHTTPRequest(method, url string, body []byte, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequest(method, url, strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{}
	return client.Do(req)
}
