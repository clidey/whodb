package llm

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
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

const ollamaLocalEndpoint = "http://localhost:11434/api/generate"

type LLMType string

const (
	Ollama_LLMType LLMType = "Ollama"
)

type LLMModel string

const (
	Llama3_LLMModel LLMModel = "Llama3"
)

type LLMClient struct {
	Type  LLMType
	Model LLMModel
}

func (c *LLMClient) Complete(prompt string, receiverChan chan string) error {
	requestBody, err := json.Marshal(completionRequest{
		Model:  string(c.Model),
		Prompt: prompt,
	})

	if err != nil {
		return err
	}

	var url string
	switch c.Type {
	case Ollama_LLMType:
		url = ollamaLocalEndpoint
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return errors.New(string(body))
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		var completionResponse completionResponse
		err := json.Unmarshal([]byte(line), &completionResponse)
		if err != nil {
			return err
		}
		receiverChan <- completionResponse.Response
		if completionResponse.Done {
			return nil
		}
	}

	return scanner.Err()
}

func CreateLLMClient(llmType LLMType, model LLMModel) *LLMClient {
	return &LLMClient{
		Type:  llmType,
		Model: model,
	}
}
