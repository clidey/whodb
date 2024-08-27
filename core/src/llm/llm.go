package llm

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/env"
)

const (
	chatgptAPIEndpoint = "https://api.openai.com/v1"
)

type LLMType string

const (
	Ollama_LLMType  LLMType = "Ollama"
	ChatGPT_LLMType LLMType = "ChatGPT"
)

type LLMModel string

const (
	Llama3_LLMModel LLMModel = "Llama3"
	GPT3_5_LLMModel LLMModel = "gpt-3.5-turbo"
	GPT4_LLMModel   LLMModel = "gpt-4"
)

type LLMClient struct {
	Type   LLMType
	APIKey string
}

func (c *LLMClient) Complete(prompt string, model LLMModel, receiverChan *chan string) (*string, error) {
	var requestBody []byte
	var err error
	var url string
	var headers map[string]string

	switch c.Type {
	case Ollama_LLMType:
		requestBody, err = json.Marshal(map[string]interface{}{
			"model":  string(model),
			"prompt": prompt,
		})
		if err != nil {
			return nil, err
		}
		url = fmt.Sprintf("%v/generate", getOllamaEndpoint())

	case ChatGPT_LLMType:
		requestBody, err = json.Marshal(map[string]interface{}{
			"model":    string(model),
			"messages": []map[string]string{{"role": "user", "content": prompt}},
			"stream":   receiverChan != nil,
		})
		if err != nil {
			return nil, err
		}
		url = fmt.Sprintf("%v/chat/completions", chatgptAPIEndpoint)
		headers = map[string]string{
			"Authorization": fmt.Sprintf("Bearer %s", c.APIKey),
			"Content-Type":  "application/json",
		}
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, err
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, errors.New(string(body))
	}

	responseBuilder := strings.Builder{}

	switch c.Type {
	case Ollama_LLMType:
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			var completionResponse struct {
				Response string `json:"response"`
				Done     bool   `json:"done"`
			}
			err := json.Unmarshal([]byte(line), &completionResponse)
			if err != nil {
				return nil, err
			}
			if receiverChan != nil {
				*receiverChan <- completionResponse.Response
			}
			if _, err := responseBuilder.WriteString(completionResponse.Response); err != nil {
				return nil, err
			}
			if completionResponse.Done {
				response := responseBuilder.String()
				return &response, nil
			}
		}
		return nil, scanner.Err()

	case ChatGPT_LLMType:
		if receiverChan != nil {
			scanner := bufio.NewScanner(resp.Body)
			for scanner.Scan() {
				line := scanner.Text()
				if line == "" {
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

				err := json.Unmarshal([]byte(line), &completionResponse)
				if err != nil {
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
			return nil, scanner.Err()
		} else {
			var completionResponse struct {
				Choices []struct {
					Message struct {
						Content string `json:"content"`
					} `json:"message"`
				} `json:"choices"`
			}

			if err := json.NewDecoder(resp.Body).Decode(&completionResponse); err != nil {
				return nil, err
			}

			if len(completionResponse.Choices) > 0 {
				response := completionResponse.Choices[0].Message.Content
				return &response, nil
			}

			return nil, errors.New("no completion response received")
		}
	}

	return nil, errors.New("unsupported LLM type")
}

func (c *LLMClient) GetSupportedModels() ([]string, error) {
	var url string
	var headers map[string]string
	switch c.Type {
	case Ollama_LLMType:
		url = fmt.Sprintf("%v/tags", getOllamaEndpoint())
	case ChatGPT_LLMType:
		url = fmt.Sprintf("%v/models", chatgptAPIEndpoint)
		headers = map[string]string{
			"Authorization": fmt.Sprintf("Bearer %s", c.APIKey),
			"Content-Type":  "application/json",
		}
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, errors.New(string(body))
	}

	if c.Type == Ollama_LLMType {
		var modelsResp struct {
			Models []struct {
				Name string `json:"model"`
			} `json:"models"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
			return nil, err
		}

		models := []string{}
		for _, model := range modelsResp.Models {
			models = append(models, model.Name)
		}

		return models, nil
	} else if c.Type == ChatGPT_LLMType {
		var modelsResp struct {
			Models []struct {
				Name string `json:"id"`
			} `json:"data"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
			return nil, err
		}

		models := []string{}
		for _, model := range modelsResp.Models {
			models = append(models, model.Name)
		}

		return models, nil
	}
	return []string{}, nil
}

var llmInstance map[LLMType]*LLMClient

func Instance(config *engine.PluginConfig) *LLMClient {
	if llmInstance == nil {
		llmInstance = make(map[LLMType]*LLMClient)
	}

	llmType := LLMType(config.ExternalModel.Type)

	if _, ok := llmInstance[llmType]; ok {
		return llmInstance[llmType]
	}
	instance := &LLMClient{
		Type:   llmType,
		APIKey: config.ExternalModel.Token,
	}
	llmInstance[llmType] = instance
	return instance
}

func getOllamaEndpoint() string {
	host := "localhost"
	port := "11434"

	if common.IsRunningInsideDocker() {
		host = "host.docker.internal"
	}

	if env.OllamaHost != "" {
		host = env.OllamaHost
	}
	if env.OllamaPort != "" {
		port = env.OllamaPort
	}

	return fmt.Sprintf("http://%v:%v/api", host, port)
}
