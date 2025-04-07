// Copyright 2025 Clidey, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package llm

import (
	"fmt"
	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/env"
	"os"
	"strings"
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
