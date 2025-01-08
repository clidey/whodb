package llm

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)


func prepareChatGPTRequest(c *LLMClient, prompt string, model LLMModel, receiverChan *chan string) (string, []byte, map[string]string, error) {
	requestBody, err := json.Marshal(map[string]interface{}{
		"model":    string(model),
		"messages": []map[string]string{{"role": "user", "content": prompt}},
		"stream":   receiverChan != nil,
	})
	if err != nil {
		return "", nil, nil, err
	}
	url := fmt.Sprintf("%v/chat/completions", getOpenAICompatibleBaseURL())
	headers := map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", c.APIKey),
		"Content-Type":  "application/json",
	}
	return url, requestBody, headers, nil
}

func prepareChatGPTModelsRequest(apiKey string) (string, map[string]string) {
	url := fmt.Sprintf("%v/models", getOpenAICompatibleBaseURL())
	headers := map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", apiKey),
		"Content-Type":  "application/json",
	}
	return url, headers
}

func parseChatGPTResponse(body io.ReadCloser, receiverChan *chan string, responseBuilder *strings.Builder) (*string, error) {
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

			if strings.TrimSpace(line) == "" {
				continue
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

	return nil, nil
}

func parseChatGPTModelsResponse(body io.ReadCloser) ([]string, error) {
	defer body.Close()

	var modelsResp struct {
		Models []struct {
			Name string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(body).Decode(&modelsResp); err != nil {
		return nil, err
	}

	models := []string{}
	for _, model := range modelsResp.Models {
		if strings.HasPrefix(model.Name, "gpt-") {
			models = append(models, model.Name)
		}
	}
	return models, nil
}
