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
)

const anthropicEndpoint = "https://api.anthropic.com/v1"

func prepareAnthropicRequest(c *LLMClient, prompt string, model LLMModel) (string, []byte, map[string]string, error) {
	maxTokens := 64000 // this is for claude-3-7-sonnet-20250219
	modelName := string(model)

	switch modelName {
	case "claude-3-5-sonnet-20241022", "claude-3-5-sonnet-20240620", "claude-3-5-haiku-20241022":
		maxTokens = 8192
	case "claude-3-opus-20240229", "claude-3-haiku-20240307":
		maxTokens = 4096
	}

	requestBody, err := json.Marshal(map[string]interface{}{
		"model":      modelName,
		"max_tokens": maxTokens,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	})
	if err != nil {
		return "", nil, nil, err
	}

	url := fmt.Sprintf("%v/messages", anthropicEndpoint)

	headers := map[string]string{
		"x-api-key":         c.APIKey,
		"anthropic-version": "2023-06-01",
		"content-type":      "application/json",
	}

	return url, requestBody, headers, nil
}

func getAnthropicModels(_ string) ([]string, error) {
	models := []string{
		//"claude-3-7-sonnet-latest",
		//"claude-3-5-haiku-latest",
		//"claude-3-5-sonnet-latest",
		//"claude-3-opus-latest",
		"claude-3-7-sonnet-20250219",
		"claude-3-5-sonnet-20241022",
		"claude-3-5-sonnet-20240620",
		"claude-3-5-haiku-20241022",
		"claude-3-haiku-20240307",
		"claude-3-opus-20240229",
	}
	return models, nil
}

func parseAnthropicResponse(body io.ReadCloser, receiverChan *chan string, responseBuilder *strings.Builder) (*string, error) {
	defer body.Close()
	reader := bufio.NewReader(body)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
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
			return nil, err
		}

		for _, content := range anthropicResponse.Content {
			if receiverChan != nil {
				*receiverChan <- content.Text
			}
			if _, err := responseBuilder.WriteString(content.Text); err != nil {
				return nil, err
			}
		}

		if anthropicResponse.StopReason == "end_turn" {
			response := responseBuilder.String()
			return &response, nil
		}
	}
}
