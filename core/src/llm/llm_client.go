// Licensed to Clidey Limited under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Clidey Limited licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package llm

import (
	"errors"
	"io"
	"net/http"
	"strings"
)

type LLMType string

const (
	Ollama_LLMType    LLMType = "Ollama"
	ChatGPT_LLMType   LLMType = "ChatGPT"
	Anthropic_LLMType LLMType = "Anthropic"
)

type LLMModel string

type LLMClient struct {
	Type   LLMType
	APIKey string
}

func (c *LLMClient) Complete(prompt string, model LLMModel, receiverChan *chan string) (*string, error) {
	var url string
	var headers map[string]string
	var requestBody []byte
	var err error

	switch c.Type {
	case Ollama_LLMType:
		url, requestBody, headers, err = prepareOllamaRequest(prompt, model)
	case ChatGPT_LLMType:
		url, requestBody, headers, err = prepareChatGPTRequest(c, prompt, model, receiverChan)
	case Anthropic_LLMType:
		url, requestBody, headers, err = prepareAnthropicRequest(c, prompt, model)
	default:
		return nil, errors.New("unsupported LLM type")
	}

	if err != nil {
		return nil, err
	}

	resp, err := sendHTTPRequest("POST", url, requestBody, headers)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, errors.New(string(body))
	}

	return c.parseResponse(resp.Body, receiverChan)
}

func (c *LLMClient) GetSupportedModels() ([]string, error) {
	var url string
	var headers map[string]string

	switch c.Type {
	case Ollama_LLMType:
		url, headers = prepareOllamaModelsRequest()
	case ChatGPT_LLMType:
		url, headers = prepareChatGPTModelsRequest(c.APIKey)
	case Anthropic_LLMType:
		return getAnthropicModels(c.APIKey)
	default:
		return nil, errors.New("unsupported LLM type")
	}

	resp, err := sendHTTPRequest("GET", url, nil, headers)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
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
