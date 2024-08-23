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
	"github.com/clidey/whodb/core/src/env"
)

type completionRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

type completionResponse struct {
	Model     string `json:"model"`
	CreatedAt string `json:"created_at"`
	Response  string `json:"response"`
	Done      bool   `json:"done"`
}

type LLMType string

const (
	Ollama_LLMType LLMType = "Ollama"
)

type LLMModel string

const (
	Llama3_LLMModel LLMModel = "Llama3"
)

type LLMClient struct {
	Type LLMType
}

func (c *LLMClient) Complete(prompt string, model LLMModel, receiverChan *chan string) (*string, error) {
	requestBody, err := json.Marshal(completionRequest{
		Model:  string(model),
		Prompt: prompt,
	})

	if err != nil {
		return nil, err
	}

	var url string
	switch c.Type {
	case Ollama_LLMType:
		url = fmt.Sprintf("%v/generate", getOllamaEndpoint())
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, errors.New(string(body))
	}

	responseBuilder := strings.Builder{}
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		var completionResponse completionResponse
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
}

func (c *LLMClient) GetSupportedModels() ([]string, error) {
	var url string
	switch c.Type {
	case Ollama_LLMType:
		url = fmt.Sprintf("%v/tags", getOllamaEndpoint())
	}

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, errors.New(string(body))
	}

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
}

var llmInstance map[LLMType]*LLMClient

func Instance(llmType LLMType) *LLMClient {
	if llmInstance == nil {
		llmInstance = make(map[LLMType]*LLMClient)
	}

	if _, ok := llmInstance[llmType]; ok {
		return llmInstance[llmType]
	}
	instance := &LLMClient{
		Type: llmType,
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
