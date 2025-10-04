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
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/clidey/whodb/core/src/log"
)


func prepareOllamaRequestWithConfig(config *ProviderConfig, prompt string, model string, settings *ProviderSettings, receiverChan *chan string) (string, []byte, map[string]string, error) {
	// Build request body with settings
	requestData := map[string]interface{}{
		"model":  model,
		"prompt": prompt,
		"stream": receiverChan != nil,
	}

	// Build options object for Ollama-specific settings
	options := make(map[string]interface{})

	// Apply provider-specific settings
	if settings.Temperature != nil {
		options["temperature"] = *settings.Temperature
	}
	if settings.TopP != nil {
		options["top_p"] = *settings.TopP
	}
	if settings.TopK != nil {
		options["top_k"] = *settings.TopK
	}
	if settings.RepeatPenalty != nil {
		options["repeat_penalty"] = *settings.RepeatPenalty
	}

	// Add options to request if any were set
	if len(options) > 0 {
		requestData["options"] = options
	}

	requestBody, err := json.Marshal(requestData)
	if err != nil {
		log.Logger.WithError(err).Errorf("Failed to marshal Ollama request body for model %s", model)
		return "", nil, nil, err
	}

	url := fmt.Sprintf("%v/generate", config.BaseURL)
	return url, requestBody, nil, nil
}


func parseOllamaResponse(body io.ReadCloser, receiverChan *chan string, responseBuilder *strings.Builder) (*string, error) {
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

func parseOllamaModelsResponse(body io.ReadCloser) ([]string, error) {

	var modelsResp struct {
		Models []struct {
			Name string `json:"model"`
		} `json:"models"`
	}
	if err := json.NewDecoder(body).Decode(&modelsResp); err != nil {
		log.Logger.WithError(err).Error("Failed to decode Ollama models response")
		return nil, err
	}

	models := []string{}
	for _, model := range modelsResp.Models {
		models = append(models, model.Name)
	}
	return models, nil
}
