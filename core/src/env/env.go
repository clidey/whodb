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

package env

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/log"
)

var IsDevelopment = os.Getenv("ENVIRONMENT") == "dev"
var IsEnterpriseEdition = false // Set to true by EE build
var Tokens = common.FilterList(strings.Split(os.Getenv("WHODB_TOKENS"), ","), func(item string) bool {
	return item != ""
})
var IsAPIGatewayEnabled = len(Tokens) > 0
var OllamaHost = os.Getenv("WHODB_OLLAMA_HOST")
var OllamaPort = os.Getenv("WHODB_OLLAMA_PORT")

var AnthropicAPIKey = os.Getenv("WHODB_ANTHROPIC_API_KEY")
var AnthropicEndpoint = os.Getenv("WHODB_ANTHROPIC_ENDPOINT")

var OpenAIAPIKey = os.Getenv("WHODB_OPENAI_API_KEY")
var OpenAIEndpoint = os.Getenv("WHODB_OPENAI_ENDPOINT")

var OpenAICompatibleEndpoint = os.Getenv("WHODB_OPENAI_COMPATIBLE_ENDPOINT")
var OpenAICompatibleAPIKey = os.Getenv("WHODB_OPENAI_COMPATIBLE_API_KEY")

// var OpenAICompatibleLabel = os.Getenv("WHODB_OPENAI_COMPATIBLE_LABEL")

var CustomModels = common.FilterList(strings.Split(os.Getenv("WHODB_CUSTOM_MODELS"), ","), func(item string) bool {
	return strings.TrimSpace(item) != ""
})

var AllowedOrigins = common.FilterList(strings.Split(os.Getenv("WHODB_ALLOWED_ORIGINS"), ","), func(item string) bool {
	return item != ""
})

var LogLevel = getLogLevel()
var EnableMockDataGeneration = os.Getenv("WHODB_ENABLE_MOCK_DATA_GENERATION")

type ChatProvider struct {
	Type       string
	APIKey     string
	Endpoint   string
	ProviderId string
}

// TODO: need to make this more dynamic so users can configure more than one key for each provider
func GetConfiguredChatProviders() []ChatProvider {
	providers := []ChatProvider{}

	if len(OpenAIAPIKey) > 0 {
		providers = append(providers, ChatProvider{
			Type:       "ChatGPT",
			APIKey:     OpenAIAPIKey,
			Endpoint:   GetOpenAIEndpoint(),
			ProviderId: "chatgpt-1",
		})
	}

	if len(AnthropicAPIKey) > 0 {
		providers = append(providers, ChatProvider{
			Type:       "Anthropic",
			APIKey:     AnthropicAPIKey,
			Endpoint:   GetAnthropicEndpoint(),
			ProviderId: "anthropic-1",
		})
	}

	if len(OpenAICompatibleAPIKey) > 0 && len(OpenAICompatibleEndpoint) > 0 && len(CustomModels) > 0 {
		providers = append(providers, ChatProvider{
			Type:       "OpenAI-Compatible",
			APIKey:     OpenAICompatibleAPIKey,
			Endpoint:   GetOpenAICompatibleEndpoint(),
			ProviderId: "openai-compatible-1",
		})
	}

	providers = append(providers, ChatProvider{
		Type:       "Ollama",
		APIKey:     "",
		Endpoint:   GetOllamaEndpoint(),
		ProviderId: "ollama-1",
	})

	return providers
}

func GetOllamaEndpoint() string {
	host := "localhost"
	port := "11434"

	if common.IsRunningInsideDocker() {
		host = "host.docker.internal"
	}

	if OllamaHost != "" {
		host = OllamaHost
	}
	if OllamaPort != "" {
		port = OllamaPort
	}

	return fmt.Sprintf("http://%v:%v/api", host, port)
}

func GetAnthropicEndpoint() string {
	if AnthropicEndpoint != "" {
		return AnthropicEndpoint
	}
	return "https://api.anthropic.com/v1"
}

func GetOpenAIEndpoint() string {
	if OpenAIEndpoint != "" {
		return OpenAIEndpoint
	}
	return "https://api.openai.com/v1"
}

func GetOpenAICompatibleEndpoint() string {
	if OpenAICompatibleEndpoint != "" && OpenAICompatibleAPIKey != "" && len(CustomModels) > 0 {
		return OpenAICompatibleEndpoint
	}
	return "https://api.openai.com/v1"
}

func GetClideyQuickContainerImage() string {
	image := os.Getenv("CLIDEY_QUICK_CONTAINER_IMAGE")
	if len(image) == 0 {
		return ""
	}
	splitImage := strings.Split(image, ":")
	if len(splitImage) != 2 {
		return ""
	}
	return splitImage[1]
}

type DatabaseCredentials struct {
	Alias    string            `json:"alias"`
	Hostname string            `json:"host"`
	Username string            `json:"user"`
	Password string            `json:"password"`
	Database string            `json:"database"`
	Port     string            `json:"port"`
	Config   map[string]string `json:"config"`

	IsProfile bool
	Type      string
}

func GetDefaultDatabaseCredentials(databaseType string) []DatabaseCredentials {
	uppercaseDatabaseType := strings.ToUpper(databaseType)
	credEnvVar := fmt.Sprintf("WHODB_%s", uppercaseDatabaseType)
	credEnvValue := os.Getenv(credEnvVar)

	if credEnvValue == "" {
		return findAllDatabaseCredentials(databaseType)
	}

	var creds []DatabaseCredentials
	err := json.Unmarshal([]byte(credEnvValue), &creds)
	if err != nil {
		log.Logger.Error("🔴 [Database Error] Failed to parse database credentials from environment variable! Error: ", err)
		return nil
	}

	return creds
}

func findAllDatabaseCredentials(databaseType string) []DatabaseCredentials {
	uppercaseDatabaseType := strings.ToUpper(databaseType)
	i := 1
	profiles := []DatabaseCredentials{}

	for {
		databaseProfile := os.Getenv(fmt.Sprintf("WHODB_%s_%d", uppercaseDatabaseType, i))
		if databaseProfile == "" {
			break
		}

		var creds DatabaseCredentials
		err := json.Unmarshal([]byte(databaseProfile), &creds)
		if err != nil {
			log.Logger.Error("Unable to parse database credential: ", err)
			break
		}

		profiles = append(profiles, creds)
		i++
	}

	return profiles
}

func getLogLevel() string {
	level := os.Getenv("WHODB_LOG_LEVEL")
	switch level {
	case "info", "INFO", "Info":
		return "info"
	case "warning", "WARNING", "Warning", "warn", "WARN", "Warn":
		return "warning"
	case "error", "ERROR", "Error":
		return "error"
	default:
		return "info" // Default to info level
	}

  func IsMockDataGenerationAllowed(tableName string) bool {
	if EnableMockDataGeneration == "" {
		return false
	}

	if EnableMockDataGeneration == "*" {
		return true
	}

	allowedTables := strings.SplitSeq(EnableMockDataGeneration, ",")
	for allowed := range allowedTables {
		if strings.TrimSpace(allowed) == tableName {
			return true
		}
	}

	return false
}

func GetMockDataGenerationMaxRowCount() int {
	return 200
}
