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
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/clidey/whodb/core/src/log"
)

type LLMType string

const (
	Ollama_LLMType           LLMType = "Ollama"
	ChatGPT_LLMType          LLMType = "ChatGPT"
	Anthropic_LLMType        LLMType = "Anthropic"
	OpenAICompatible_LLMType LLMType = "OpenAI-Compatible"
)

type LLMModel string

type LLMClient struct {
	Config *ProviderConfig // Provider configuration is now required
}

func (c *LLMClient) Complete(prompt string, model LLMModel, receiverChan *chan string) (*string, error) {
	// Always use provider config
	if c.Config == nil {
		return nil, errors.New("provider configuration is required")
	}

	return c.CompleteWithConfig(prompt, string(model), receiverChan)
}

func (c *LLMClient) GetSupportedModels() ([]string, error) {
	// Always use provider config
	if c.Config == nil {
		return nil, errors.New("provider configuration is required")
	}

	var url string
	var headers map[string]string

	switch c.Config.Type {
	case Ollama_LLMType:
		url = fmt.Sprintf("%v/tags", c.Config.BaseURL)
		headers = nil
	case ChatGPT_LLMType:
		url = fmt.Sprintf("%v/models", c.Config.BaseURL)
		headers = map[string]string{
			"Authorization": fmt.Sprintf("Bearer %s", c.Config.APIKey),
			"Content-Type":  "application/json",
		}
	case Anthropic_LLMType:
		return getAnthropicModels(c.Config.APIKey)
	case OpenAICompatible_LLMType:
		return getOpenAICompatibleModelsForConfig(c.Config)
	default:
		return nil, errors.New("unsupported LLM type")
	}

	resp, err := sendHTTPRequest("GET", url, nil, headers)
	if err != nil {
		log.Logger.WithError(err).Errorf("Failed to fetch models from %s LLM service at %s", c.Config.Type, url)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Logger.WithError(err).Errorf("Failed to read models error response body from %s LLM service (status: %d)", c.Config.Type, resp.StatusCode)
			return nil, err
		}
		log.Logger.Errorf("%s LLM service models endpoint returned non-OK status: %d, body: %s", c.Config.Type, resp.StatusCode, string(body))
		return nil, errors.New(string(body))
	}

	return c.parseModelsResponse(resp.Body)
}

func (c *LLMClient) parseResponse(body io.ReadCloser, receiverChan *chan string) (*string, error) {
	if c.Config == nil {
		return nil, errors.New("provider configuration is required")
	}

	responseBuilder := strings.Builder{}
	switch c.Config.Type {
	case Ollama_LLMType:
		return parseOllamaResponse(body, receiverChan, &responseBuilder)
	case ChatGPT_LLMType:
		return parseChatGPTResponse(body, receiverChan, &responseBuilder)
	case Anthropic_LLMType:
		return parseAnthropicResponse(body, receiverChan, &responseBuilder)
	case OpenAICompatible_LLMType:
		return parseChatGPTResponse(body, receiverChan, &responseBuilder)
	default:
		return nil, errors.New("unsupported LLM type")
	}
}

func (c *LLMClient) parseModelsResponse(body io.ReadCloser) ([]string, error) {
	if c.Config == nil {
		return nil, errors.New("provider configuration is required")
	}

	switch c.Config.Type {
	case Ollama_LLMType:
		return parseOllamaModelsResponse(body)
	case ChatGPT_LLMType:
		return parseChatGPTModelsResponse(body)
	default:
		return nil, errors.New("unsupported LLM type")
	}
}

// CompleteWithConfig performs completion using provider configuration
func (c *LLMClient) CompleteWithConfig(prompt string, modelOverride string, receiverChan *chan string) (*string, error) {
	if c.Config == nil {
		return nil, errors.New("provider configuration is required")
	}

	if err := c.Config.Validate(); err != nil {
		return nil, err
	}

	// Get provider settings
	settings, err := c.Config.GetSettings()
	if err != nil {
		return nil, err
	}

	// Use model override if provided, otherwise use settings model
	model := modelOverride
	if model == "" {
		model = settings.Model
	}

	var url string
	var headers map[string]string
	var requestBody []byte

	switch c.Config.Type {
	case Ollama_LLMType:
		url, requestBody, headers, err = prepareOllamaRequestWithConfig(c.Config, prompt, model, settings, receiverChan)
	case ChatGPT_LLMType:
		url, requestBody, headers, err = prepareChatGPTRequestWithConfig(c.Config, prompt, model, settings, receiverChan)
	case Anthropic_LLMType:
		url, requestBody, headers, err = prepareAnthropicRequestWithConfig(c.Config, prompt, model, settings, receiverChan)
	case OpenAICompatible_LLMType:
		url, requestBody, headers, err = prepareOpenAICompatibleRequestWithConfig(c.Config, prompt, model, settings, receiverChan)
	default:
		return nil, errors.New("unsupported LLM type: " + string(c.Config.Type))
	}

	if err != nil {
		log.Logger.WithError(err).Errorf("Failed to prepare %s LLM request for model %s", c.Config.Type, model)
		return nil, err
	}

	resp, err := sendHTTPRequest("POST", url, requestBody, headers)
	if err != nil {
		log.Logger.WithError(err).Errorf("Failed to send HTTP request to %s LLM service at %s", c.Config.Type, url)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Logger.WithError(err).Errorf("Failed to read error response body from %s LLM service (status: %d)", c.Config.Type, resp.StatusCode)
			return nil, err
		}
		log.Logger.Errorf("%s LLM service returned non-OK status: %d, body: %s", c.Config.Type, resp.StatusCode, string(body))
		return nil, errors.New(string(body))
	}

	return c.parseResponse(resp.Body, receiverChan)
}
