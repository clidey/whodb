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
	"fmt"
	"io"
	"net/http"

	"github.com/clidey/whodb/core/src/log"
)

const (
	LMStudio_LLMType LLMType = "LMStudio"
)

// LMStudioProvider implements the AIProvider interface for local LM Studio instances.
type LMStudioProvider struct{}

// NewLMStudioProvider creates a new LM Studio provider instance.
func NewLMStudioProvider() *LMStudioProvider {
	return &LMStudioProvider{}
}

// GetType returns the provider type.
func (p *LMStudioProvider) GetType() LLMType {
	return LMStudio_LLMType
}

// GetProtocol returns "openai" — LM Studio uses the OpenAI-compatible streaming protocol.
func (p *LMStudioProvider) GetProtocol() string {
	return "openai"
}

// GetDefaultEndpoint returns the default LM Studio API endpoint.
func (p *LMStudioProvider) GetDefaultEndpoint() string {
	return "http://localhost:1234/v1"
}

// ValidateConfig validates the provider configuration.
func (p *LMStudioProvider) ValidateConfig(config *ProviderConfig) error {
	if config.Endpoint == "" {
		config.Endpoint = p.GetDefaultEndpoint()
	}
	return nil
}

// GetSupportedModels fetches models from LM Studio's OpenAI-compatible /v1/models endpoint.
// Unlike the OpenAI provider, we do NOT filter by model name prefix —
// LM Studio serves user-loaded models with arbitrary names.
func (p *LMStudioProvider) GetSupportedModels(config *ProviderConfig) ([]string, error) {
	if err := p.ValidateConfig(config); err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/models", config.Endpoint)

	headers := map[string]string{
		"Content-Type": "application/json",
	}
	if config.APIKey != "" {
		headers["Authorization"] = fmt.Sprintf("Bearer %s", config.APIKey)
	}

	resp, err := sendHTTPRequest("GET", url, nil, headers)
	if err != nil {
		log.WithError(err).Errorf("Failed to fetch models from LM Studio at %s", url)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Errorf("LM Studio models endpoint returned non-OK status: %d, body: %s", resp.StatusCode, string(body))
		return nil, fmt.Errorf("failed to fetch models: %s", string(body))
	}

	var modelsResp struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		log.WithError(err).Error("Failed to decode LM Studio models response")
		return nil, err
	}

	models := []string{}
	for _, model := range modelsResp.Data {
		if model.ID != "" {
			models = append(models, model.ID)
		}
	}
	return models, nil
}

// CreateBAMLClient creates BAML client type and options for LM Studio.
// LM Studio uses the openai-generic client type as it's OpenAI-compatible.
func (p *LMStudioProvider) CreateBAMLClient(config *ProviderConfig, model string) (string, map[string]any, error) {
	if err := p.ValidateConfig(config); err != nil {
		return "", nil, err
	}

	opts := map[string]any{
		"base_url":           config.Endpoint,
		"model":              model,
		"default_role":       "user",
		"request_timeout_ms": 60000,
	}

	if config.APIKey != "" {
		opts["api_key"] = config.APIKey
	}

	return "openai-generic", opts, nil
}
