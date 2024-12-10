package llm

import (
	"fmt"
    "strings"
    "os"
	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/env"
)

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

func getOpenAICompatibleBaseURL() string {
	defaultBaseURL := "https://api.openai.com/v1"
	baseURL := os.Getenv("OPENAI_BASE_URL")
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return baseURL
}

func getCustomModels() ([]string, error) {
    modelsStr := os.Getenv("CUSTOM_MODELS")
	if modelsStr == "" {
		return []string{}, nil
	}

	models := strings.Split(modelsStr, ",")

	for i := range models {
		models[i] = strings.TrimSpace(models[i])
	}
    return models, nil
}

func ShouldUseCustomModels() bool {
	useCustomModels := os.Getenv("USE_CUSTOM_MODELS")
	return useCustomModels == "1"
}