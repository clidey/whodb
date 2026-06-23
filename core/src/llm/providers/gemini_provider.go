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
	"net/url"
	"strings"

	"github.com/clidey/whodb/core/src/env"
	"github.com/clidey/whodb/core/src/log"
)

const (
	// Gemini_LLMType is the provider type for Google Gemini API / AI Studio.
	Gemini_LLMType LLMType = "Gemini"
)

// GeminiProvider implements the AIProvider interface for Google Gemini API.
type GeminiProvider struct{}

// NewGeminiProvider creates a new Gemini provider instance.
func NewGeminiProvider() *GeminiProvider {
	return &GeminiProvider{}
}

// GetType returns the provider type.
func (p *GeminiProvider) GetType() LLMType {
	return Gemini_LLMType
}

// GetProtocol returns "openai" because this provider uses Gemini's OpenAI-compatible endpoint.
func (p *GeminiProvider) GetProtocol() string {
	return "openai"
}

// GetDefaultEndpoint returns the default Gemini OpenAI-compatible API endpoint.
func (p *GeminiProvider) GetDefaultEndpoint() string {
	return env.GetGeminiEndpoint()
}

// ValidateConfig validates the provider configuration.
func (p *GeminiProvider) ValidateConfig(config *ProviderConfig) error {
	if config.APIKey == "" {
		return errors.New("API key is required for Gemini")
	}
	if config.Endpoint == "" {
		config.Endpoint = p.GetDefaultEndpoint()
	}
	return nil
}

// GetSupportedModels fetches Gemini models that support generateContent.
func (p *GeminiProvider) GetSupportedModels(config *ProviderConfig) ([]string, error) {
	if err := p.ValidateConfig(config); err != nil {
		return nil, err
	}

	models := []string{}
	seen := map[string]struct{}{}
	pageToken := ""
	for {
		modelsURL, err := geminiModelsURL(config.Endpoint, config.APIKey, pageToken)
		if err != nil {
			return nil, err
		}

		resp, err := sendHTTPRequest("GET", modelsURL, nil, map[string]string{
			"Content-Type": "application/json",
		})
		if err != nil {
			log.WithError(err).Errorf("Failed to fetch models from Gemini at %s", modelsURL)
			return nil, err
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			log.Errorf("Gemini models endpoint returned non-OK status: %d, body: %s", resp.StatusCode, string(body))
			return nil, fmt.Errorf("failed to fetch models: %s", string(body))
		}

		var modelsResp struct {
			Models []struct {
				Name                       string   `json:"name"`
				BaseModelID                string   `json:"baseModelId"`
				SupportedGenerationMethods []string `json:"supportedGenerationMethods"`
			} `json:"models"`
			NextPageToken string `json:"nextPageToken"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
			_ = resp.Body.Close()
			log.WithError(err).Error("Failed to decode Gemini models response")
			return nil, err
		}
		_ = resp.Body.Close()

		for _, model := range modelsResp.Models {
			if !supportsGeminiGenerateContent(model.SupportedGenerationMethods) {
				continue
			}
			name := model.BaseModelID
			if name == "" {
				name = strings.TrimPrefix(model.Name, "models/")
			}
			if name == "" {
				continue
			}
			if _, ok := seen[name]; ok {
				continue
			}
			models = append(models, name)
			seen[name] = struct{}{}
		}

		if modelsResp.NextPageToken == "" {
			break
		}
		pageToken = modelsResp.NextPageToken
	}
	return models, nil
}

// CreateBAMLClient creates BAML client type and options for Gemini.
func (p *GeminiProvider) CreateBAMLClient(config *ProviderConfig, model string) (string, map[string]any, error) {
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

func geminiModelsURL(endpoint string, apiKey string, pageToken string) (string, error) {
	base := strings.TrimRight(endpoint, "/")
	base = strings.TrimSuffix(base, "/openai")
	modelsURL, err := url.Parse(base + "/models")
	if err != nil {
		return "", err
	}
	query := modelsURL.Query()
	query.Set("key", apiKey)
	query.Set("pageSize", "1000")
	if pageToken != "" {
		query.Set("pageToken", pageToken)
	}
	modelsURL.RawQuery = query.Encode()
	return modelsURL.String(), nil
}

func supportsGeminiGenerateContent(methods []string) bool {
	for _, method := range methods {
		if method == "generateContent" {
			return true
		}
	}
	return false
}
