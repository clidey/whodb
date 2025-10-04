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


func prepareAnthropicRequestWithConfig(config *ProviderConfig, prompt string, model string, settings *ProviderSettings, receiverChan *chan string) (string, []byte, map[string]string, error) {
	maxTokens := 4096 // conservative default for unknown models

	// Use settings max tokens if provided
	if settings.MaxTokens != nil {
		maxTokens = *settings.MaxTokens
	} else {
		// Auto-detect based on model
		switch model {
		case "claude-3-7-sonnet-20250219", "claude-sonnet-4-20250514":
			maxTokens = 64000
		case "claude-opus-4-20250514":
			maxTokens = 32000
		case "claude-3-5-sonnet-20241022", "claude-3-5-sonnet-20240620", "claude-3-5-opus-20241022", "claude-3-5-haiku-20241022":
			maxTokens = 8192
		case "claude-3-opus-20240229", "claude-3-haiku-20240307":
			maxTokens = 4096
		}
	}

	// Build request body with settings
	requestData := map[string]interface{}{
		"model":      model,
		"max_tokens": maxTokens,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}

	// Apply provider-specific settings
	if settings.Temperature != nil {
		requestData["temperature"] = *settings.Temperature
	}
	if settings.TopP != nil {
		requestData["top_p"] = *settings.TopP
	}
	if settings.TopK != nil {
		requestData["top_k"] = *settings.TopK
	}

	// Note: Anthropic doesn't support streaming in the same way as OpenAI
	// Stream support would require different handling

	requestBody, err := json.Marshal(requestData)
	if err != nil {
		log.Logger.WithError(err).Errorf("Failed to marshal Anthropic request body for model %s", model)
		return "", nil, nil, err
	}

	url := fmt.Sprintf("%v/messages", config.BaseURL)

	headers := map[string]string{
		"x-api-key":         config.APIKey,
		"anthropic-version": "2023-06-01",
		"Content-Type":      "application/json",
	}

	return url, requestBody, headers, nil
}

func getAnthropicModels(_ string) ([]string, error) {
	models := []string{
		"claude-opus-4-20250514",
		"claude-sonnet-4-20250514",
		"claude-3-7-sonnet-20250219",
		"claude-3-5-sonnet-20241022",
		"claude-3-5-sonnet-20240620",
		"claude-3-5-opus-20241022",
		"claude-3-5-haiku-20241022",
		"claude-3-opus-20240229",
		"claude-3-haiku-20240307",
	}
	return models, nil
}

func parseAnthropicResponse(body io.ReadCloser, receiverChan *chan string, responseBuilder *strings.Builder) (*string, error) {
	reader := bufio.NewReader(body)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				log.Logger.WithError(err).Error("Failed to read line from Anthropic response")
				return nil, err
			}
		}

		var anthropicResponse struct {
			Content []struct {
				Text string `json:"text"`
				Type string `json:"type"`
			} `json:"content"`
			StopReason string `json:"stop_reason"`
			Usage      struct {
				InputTokens  int `json:"input_tokens"`
				OutputTokens int `json:"output_tokens"`
			} `json:"usage"`
			Role         string  `json:"role"`
			Model        string  `json:"model"`
			ID           string  `json:"id"`
			Type         string  `json:"type"`
			StopSequence *string `json:"stop_sequence,omitempty"`
		}

		if err := json.Unmarshal([]byte(line), &anthropicResponse); err != nil {
			log.Logger.WithError(err).Errorf("Failed to unmarshal Anthropic response line: %s", line)
			return nil, err
		}

		for _, content := range anthropicResponse.Content {
			if receiverChan != nil {
				*receiverChan <- content.Text
			}
			if _, err := responseBuilder.WriteString(content.Text); err != nil {
				log.Logger.WithError(err).Error("Failed to write to Anthropic response builder")
				return nil, err
			}
		}

		if anthropicResponse.StopReason == "end_turn" {
			response := responseBuilder.String()
			return &response, nil
		}
	}
}
