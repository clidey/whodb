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
	Type      LLMType
	APIKey    string
	ProfileId string
}

func (c *LLMClient) Complete(prompt string, model LLMModel, receiverChan *chan string) (*string, error) {
	// Validate API key for services that require it
	if err := c.validateAPIKey(); err != nil {
		return nil, err
	}

	var url string
	var headers map[string]string
	var requestBody []byte
	var err error

	switch c.Type {
	case Ollama_LLMType:
		url, requestBody, headers, err = prepareOllamaRequest(prompt, model)
	case ChatGPT_LLMType:
		url, requestBody, headers, err = prepareChatGPTRequest(c, prompt, model, receiverChan, false)
	case Anthropic_LLMType:
		url, requestBody, headers, err = prepareAnthropicRequest(c, prompt, model)
	case OpenAICompatible_LLMType:
		url, requestBody, headers, err = prepareChatGPTRequest(c, prompt, model, receiverChan, true)
	default:
		return nil, errors.New("unsupported LLM type")
	}

	if err != nil {
		log.Logger.WithError(err).Errorf("Failed to prepare %s LLM request for model %s", c.Type, model)
		return nil, err
	}

	resp, err := sendHTTPRequest("POST", url, requestBody, headers)
	if err != nil {
		log.Logger.WithError(err).Errorf("Failed to send HTTP request to %s LLM service at %s", c.Type, url)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Logger.WithError(err).Errorf("Failed to read error response body from %s LLM service (status: %d)", c.Type, resp.StatusCode)
			return nil, err
		}
		log.Logger.Errorf("%s LLM service returned non-OK status: %d, body: %s", c.Type, resp.StatusCode, string(body))
		return nil, errors.New(string(body))
	}

	return c.parseResponse(resp.Body, receiverChan)
}

func (c *LLMClient) GetSupportedModels() ([]string, error) {
	// Validate API key for services that require it
	if err := c.validateAPIKey(); err != nil {
		return nil, err
	}

	var url string
	var headers map[string]string

	switch c.Type {
	case Ollama_LLMType:
		url, headers = prepareOllamaModelsRequest()
	case ChatGPT_LLMType:
		url, headers = prepareChatGPTModelsRequest(c.APIKey)
	case Anthropic_LLMType:
		return getAnthropicModels(c.APIKey)
	case OpenAICompatible_LLMType:
		return getOpenAICompatibleModels()
	default:
		return nil, errors.New("unsupported LLM type")
	}

	resp, err := sendHTTPRequest("GET", url, nil, headers)
	if err != nil {
		log.Logger.WithError(err).Errorf("Failed to fetch models from %s LLM service at %s", c.Type, url)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Logger.WithError(err).Errorf("Failed to read models error response body from %s LLM service (status: %d)", c.Type, resp.StatusCode)
			return nil, err
		}
		log.Logger.Errorf("%s LLM service models endpoint returned non-OK status: %d, body: %s", c.Type, resp.StatusCode, string(body))
		return nil, errors.New(string(body))
	}

	return c.parseModelsResponse(resp.Body)
}

func (c *LLMClient) parseResponse(body io.ReadCloser, receiverChan *chan string) (*string, error) {
	responseBuilder := strings.Builder{}
	switch c.Type {
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
	switch c.Type {
	case Ollama_LLMType:
		return parseOllamaModelsResponse(body)
	case ChatGPT_LLMType:
		return parseChatGPTModelsResponse(body)
	default:
		return nil, errors.New("unsupported LLM type")
	}
}

// validateAPIKey checks if API key is present for services that require it
func (c *LLMClient) validateAPIKey() error {
	requiresAPIKey := c.Type == ChatGPT_LLMType || c.Type == Anthropic_LLMType
	if requiresAPIKey && strings.TrimSpace(c.APIKey) == "" {
		return errors.New("API key is required for " + string(c.Type))
	}
	return nil
}
