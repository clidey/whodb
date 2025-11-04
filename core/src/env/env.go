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
	"github.com/clidey/whodb/core/src/types"
)

var IsDevelopment = os.Getenv("ENVIRONMENT") == "dev"
var IsEnterpriseEdition = false // Set to true by EE build

// GetIsDesktopMode returns true if running in desktop mode.
// This is a function (not a variable) so it reads the env var each time,
// allowing the desktop app to set WHODB_DESKTOP after package initialization.
func GetIsDesktopMode() bool {
	return os.Getenv("WHODB_DESKTOP") == "true"
}

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
var DisableMockDataGeneration = os.Getenv("WHODB_DISABLE_MOCK_DATA_GENERATION")

type ChatProvider struct {
	Type       string
	APIKey     string
	Endpoint   string
	ProviderId string
}

// TODO: need to make this more dynamic so users can configure more than one key for each provider
func GetConfiguredChatProviders() []ChatProvider {
	var providers []ChatProvider

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

func GetDefaultDatabaseCredentials(databaseType string) []types.DatabaseCredentials {
	uppercaseDatabaseType := strings.ToUpper(databaseType)
	credEnvVar := fmt.Sprintf("WHODB_%s", uppercaseDatabaseType)
	credEnvValue := os.Getenv(credEnvVar)

	if credEnvValue == "" {
		return findAllDatabaseCredentials(databaseType)
	}

	var creds []types.DatabaseCredentials
	err := json.Unmarshal([]byte(credEnvValue), &creds)
	if err != nil {
		log.Logger.Error("🔴 [Database Error] Failed to parse database credentials from environment variable! Error: ", err)
		return nil
	}

	return creds
}

func findAllDatabaseCredentials(databaseType string) []types.DatabaseCredentials {
	uppercaseDatabaseType := strings.ToUpper(databaseType)
	i := 1
	var profiles []types.DatabaseCredentials

	for {
		databaseProfile := os.Getenv(fmt.Sprintf("WHODB_%s_%d", uppercaseDatabaseType, i))
		if databaseProfile == "" {
			break
		}

		var creds types.DatabaseCredentials
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
}

func IsMockDataGenerationAllowed(tableName string) bool {
	if DisableMockDataGeneration == "" {
		return true
	}

	// If all tables are disabled
	if DisableMockDataGeneration == "*" {
		return false
	}

	disabledTables := strings.Split(DisableMockDataGeneration, ",")
	for _, disabled := range disabledTables {
		if strings.TrimSpace(disabled) == tableName {
			return false
		}
	}

	return true
}

func GetMockDataGenerationMaxRowCount() int {
	return 200
}
