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
	"sync"

	"github.com/clidey/whodb/core/src/log"
)

// responsesAPICache caches whether a given endpoint supports the Responses API.
// Keyed by endpoint URL, value is bool (true = supports Responses API).
var responsesAPICache sync.Map

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

// GetProtocol returns "openai" — the streaming protocol family.
func (p *OpenAIProvider) GetProtocol() string { return "openai" }

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
		log.WithError(err).Errorf("Failed to fetch models from OpenAI at %s", url)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Errorf("OpenAI models endpoint returned non-OK status: %d, body: %s", resp.StatusCode, string(body))
		return nil, fmt.Errorf("failed to fetch models: %s", string(body))
	}

	var modelsResp struct {
		Models []struct {
			Name string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		log.WithError(err).Error("Failed to decode OpenAI models response")
		return nil, err
	}

	// Filter out non-chat models using a blocklist approach.
	// New model families pass through automatically.
	var models []string
	for _, model := range modelsResp.Models {
		name := model.Name
		if isNonChatModel(name) {
			continue
		}
		models = append(models, name)
	}
	return models, nil
}

// CreateBAMLClient creates BAML client type and options for OpenAI.
// It probes the endpoint to determine whether the Responses API is available,
// falling back to Chat Completions if not. The probe result is cached per endpoint.
func (p *OpenAIProvider) CreateBAMLClient(config *ProviderConfig, model string) (string, map[string]any, error) {
	endpoint := config.Endpoint
	if endpoint == "" {
		endpoint = p.GetDefaultEndpoint()
	}

	clientType := "openai"
	if p.supportsResponsesAPI(endpoint, config.APIKey) {
		clientType = "openai-responses"
	}

	opts := map[string]any{"model": model}
	if config.APIKey != "" {
		opts["api_key"] = config.APIKey
	}
	if config.Endpoint != "" && config.Endpoint != p.GetDefaultEndpoint() {
		opts["base_url"] = config.Endpoint
	}
	return clientType, opts, nil
}

// supportsResponsesAPI probes the endpoint to check for Responses API support.
// Results are cached in responsesAPICache (keyed by endpoint URL).
//
// Logic: POST to {endpoint}/responses with empty JSON body.
//   - 404/405 → endpoint not found → Chat Completions only (cached)
//   - Any other status (400, 401, 422, etc.) → endpoint exists → Responses API available (cached)
//   - Network error → safe fallback to Chat Completions (not cached, will retry next call)
func (p *OpenAIProvider) supportsResponsesAPI(endpoint, apiKey string) bool {
	if cached, ok := responsesAPICache.Load(endpoint); ok {
		return cached.(bool)
	}

	probeURL := endpoint + "/responses"
	headers := map[string]string{
		"Content-Type": "application/json",
	}
	if apiKey != "" {
		headers["Authorization"] = "Bearer " + apiKey
	}

	resp, err := sendHTTPRequest("POST", probeURL, []byte("{}"), headers)
	if err != nil {
		log.Debugf("Responses API probe failed for %s (network error), falling back to Chat Completions", endpoint)
		return false
	}
	defer resp.Body.Close()
	io.ReadAll(resp.Body)

	supports := resp.StatusCode != http.StatusNotFound && resp.StatusCode != http.StatusMethodNotAllowed
	if supports {
		log.Debugf("Responses API available at %s (probe status: %d)", endpoint, resp.StatusCode)
	} else {
		log.Debugf("Responses API not available at %s (probe status: %d), using Chat Completions", endpoint, resp.StatusCode)
	}

	responsesAPICache.Store(endpoint, supports)
	return supports
}

// isNonChatModel returns true for model IDs that are not chat-compatible.
// Uses a blocklist so new chat model families pass through automatically.
func isNonChatModel(name string) bool {
	// Non-chat model families (prefix match)
	nonChatPrefixes := []string{
		"dall-e-",          // image generation
		"whisper-",         // speech-to-text
		"tts-",             // text-to-speech
		"text-embedding-",  // embeddings
		"text-moderation-", // moderation
		"gpt-image-",       // image generation
		"babbage-",         // legacy completions
		"davinci-",         // legacy completions
	}
	for _, prefix := range nonChatPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}

	// Non-chat model suffixes/infixes
	nonChatPatterns := []string{"-instruct", "-base", "-codex"}
	for _, pattern := range nonChatPatterns {
		if strings.Contains(name, pattern) {
			return true
		}
	}

	// Non-chat model infixes (partial name matches)
	if strings.Contains(name, "-transcribe") ||
		strings.Contains(name, "-realtime") {
		return true
	}

	return false
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
