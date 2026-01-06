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

package providers

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/clidey/whodb/core/src/log"
)

const (
	Ollama_LLMType LLMType = "Ollama"
)

// OllamaProvider implements the AIProvider interface for local Ollama instances.
type OllamaProvider struct{}

// NewOllamaProvider creates a new Ollama provider instance.
func NewOllamaProvider() *OllamaProvider {
	return &OllamaProvider{}
}

// GetType returns the provider type.
func (p *OllamaProvider) GetType() LLMType {
	return Ollama_LLMType
}

// GetName returns the provider name.
func (p *OllamaProvider) GetName() string {
	return "Ollama"
}

// RequiresAPIKey returns false as Ollama doesn't require an API key.
func (p *OllamaProvider) RequiresAPIKey() bool {
	return false
}

// GetDefaultEndpoint returns the default Ollama API endpoint.
// This will be overridden by the endpoint from config which considers WHODB_OLLAMA_HOST/PORT.
func (p *OllamaProvider) GetDefaultEndpoint() string {
	return "http://localhost:11434/api"
}

// ValidateConfig validates the provider configuration.
func (p *OllamaProvider) ValidateConfig(config *ProviderConfig) error {
	if config.Endpoint == "" {
		config.Endpoint = p.GetDefaultEndpoint()
	}
	return nil
}

// GetSupportedModels fetches the list of available models from Ollama.
func (p *OllamaProvider) GetSupportedModels(config *ProviderConfig) ([]string, error) {
	if err := p.ValidateConfig(config); err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/tags", config.Endpoint)

	resp, err := sendHTTPRequest("GET", url, nil, nil)
	if err != nil {
		log.Logger.WithError(err).Errorf("Failed to fetch models from Ollama at %s", url)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Logger.Errorf("Ollama models endpoint returned non-OK status: %d, body: %s", resp.StatusCode, string(body))
		return nil, fmt.Errorf("failed to fetch models: %s", string(body))
	}

	var modelsResp struct {
		Models []struct {
			Name string `json:"model"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		log.Logger.WithError(err).Error("Failed to decode Ollama models response")
		return nil, err
	}

	models := []string{}
	for _, model := range modelsResp.Models {
		models = append(models, model.Name)
	}
	return models, nil
}

// Complete sends a completion request to Ollama.
func (p *OllamaProvider) Complete(config *ProviderConfig, prompt string, model LLMModel, receiverChan *chan string) (*string, error) {
	if err := p.ValidateConfig(config); err != nil {
		return nil, err
	}

	requestBody, err := json.Marshal(map[string]interface{}{
		"model":  string(model),
		"prompt": prompt,
	})
	if err != nil {
		log.Logger.WithError(err).Errorf("Failed to marshal Ollama request body for model %s", model)
		return nil, err
	}

	url := fmt.Sprintf("%s/generate", config.Endpoint)

	resp, err := sendHTTPRequest("POST", url, requestBody, nil)
	if err != nil {
		log.Logger.WithError(err).Errorf("Failed to send HTTP request to Ollama at %s", url)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Logger.Errorf("Ollama returned non-OK status: %d, body: %s", resp.StatusCode, string(body))
		return nil, fmt.Errorf("ollama error: %s", string(body))
	}

	return p.parseResponse(resp.Body, receiverChan)
}

// parseResponse parses the Ollama API response (always streaming).
func (p *OllamaProvider) parseResponse(body io.ReadCloser, receiverChan *chan string) (*string, error) {
	responseBuilder := strings.Builder{}
	reader := bufio.NewReader(body)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Logger.WithError(err).Error("Failed to read line from Ollama streaming response")
			return nil, err
		}

		var completionResponse struct {
			Response string `json:"response"`
			Done     bool   `json:"done"`
		}
		if err := json.Unmarshal([]byte(line), &completionResponse); err != nil {
			log.Logger.WithError(err).Errorf("Failed to unmarshal Ollama response line: %s", line)
			return nil, err
		}

		if receiverChan != nil {
			*receiverChan <- completionResponse.Response
		}

		if _, err := responseBuilder.WriteString(completionResponse.Response); err != nil {
			log.Logger.WithError(err).Error("Failed to write to Ollama response builder")
			return nil, err
		}

		if completionResponse.Done {
			response := responseBuilder.String()
			return &response, nil
		}
	}

	return nil, nil
}

// GetBAMLClientType returns the BAML client type for Ollama.
// Ollama uses the openai-generic client type as it's OpenAI-compatible.
func (p *OllamaProvider) GetBAMLClientType() string {
	return "openai-generic"
}

// CreateBAMLClientOptions creates BAML client options for Ollama.
func (p *OllamaProvider) CreateBAMLClientOptions(config *ProviderConfig, model string) (map[string]interface{}, error) {
	if err := p.ValidateConfig(config); err != nil {
		return nil, err
	}

	// Ollama expects base_url without /api suffix for OpenAI-compatible endpoint
	baseURL := strings.TrimSuffix(config.Endpoint, "/api") + "/v1"

	return map[string]interface{}{
		"base_url":           baseURL,
		"model":              model,
		"default_role":       "user",
		"request_timeout_ms": 60000,
	}, nil
}
