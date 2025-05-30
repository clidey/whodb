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
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/clidey/whodb/core/src/env"
)

func prepareOpenAICompatibleRequest(c *LLMClient, prompt string, model LLMModel, receiverChan *chan string) (string, []byte, map[string]string, error) {
	requestBody, err := json.Marshal(map[string]interface{}{
		"model":    string(model),
		"messages": []map[string]string{{"role": "user", "content": prompt}},
		"stream":   receiverChan != nil,
	})
	if err != nil {
		return "", nil, nil, err
	}
	url := fmt.Sprintf("%v/chat/completions", env.OpenAICompatibleEndpoint)
	headers := map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", c.APIKey),
		"Content-Type":  "application/json",
	}
	return url, requestBody, headers, nil
}

func parseOpenAICompatibleResponse(body io.ReadCloser, receiverChan *chan string, responseBuilder *strings.Builder) (*string, error) {
	defer body.Close()

	if receiverChan != nil {
		reader := bufio.NewReader(body)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					break
				}
				return nil, err
			}

			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			// Handle SSE format: strip "data: " prefix
			if strings.HasPrefix(line, "data: ") {
				line = strings.TrimPrefix(line, "data: ")
			}

			// Handle SSE control messages
			if line == "[DONE]" {
				// Send final accumulated response before terminating
				if responseBuilder.Len() > 0 {
					response := responseBuilder.String()
					return &response, nil
				}
				break
			}

			var completionResponse struct {
				Choices []struct {
					Delta struct {
						Content string `json:"content"`
					} `json:"delta"`
				} `json:"choices"`
				FinishReason string `json:"finish_reason"`
			}

			if err := json.Unmarshal([]byte(line), &completionResponse); err != nil {
				return nil, err
			}

			if len(completionResponse.Choices) > 0 {
				content := completionResponse.Choices[0].Delta.Content
				if content != "" {
					*receiverChan <- content
					responseBuilder.WriteString(content)
				}
				if completionResponse.FinishReason == "stop" {
					response := responseBuilder.String()
					return &response, nil
				}
			}
		}
	} else {
		var completionResponse struct {
			Choices []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			} `json:"choices"`
		}

		if err := json.NewDecoder(body).Decode(&completionResponse); err != nil {
			return nil, err
		}

		if len(completionResponse.Choices) > 0 {
			response := completionResponse.Choices[0].Message.Content
			return &response, nil
		}

		return nil, errors.New("no completion response received")
	}

	// Return accumulated response if available
	if responseBuilder.Len() > 0 {
		response := responseBuilder.String()
		return &response, nil
	}

	return nil, nil
}

func getOpenAICompatibleModels(apiKey string) ([]string, error) {
	if len(env.CustomModels) > 0 {
		return env.CustomModels, nil
	}
	return []string{}, nil
}