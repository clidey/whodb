/*
 * Copyright 2026 Clidey, Inc.
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
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/clidey/whodb/core/src/common"
)

var IsDevelopment = os.Getenv("ENVIRONMENT") == "dev"
var IsEnterpriseEdition = false // Set to true by EE build

// GetIsDesktopMode returns true if running in desktop mode.
// This is a function (not a variable) so it reads the env var each time,
// allowing the desktop app to set WHODB_DESKTOP after package initialization.
func GetIsDesktopMode() bool {
	return os.Getenv("WHODB_DESKTOP") == "true"
}

// GetIsCLIMode returns true if running as the CLI/TUI application.
func GetIsCLIMode() bool {
	return os.Getenv("WHODB_CLI") == "true"
}

// GetIsLocalMode returns true if running locally (desktop or CLI) where
// full filesystem access is expected, as opposed to server mode.
func GetIsLocalMode() bool {
	return GetIsDesktopMode() || GetIsCLIMode()
}

var Tokens = common.FilterList(strings.Split(os.Getenv("WHODB_TOKENS"), ","), func(item string) bool {
	return item != ""
})
var IsAPIGatewayEnabled = len(Tokens) > 0
var OllamaHost = os.Getenv("WHODB_OLLAMA_HOST")
var OllamaPort = os.Getenv("WHODB_OLLAMA_PORT")

var AnthropicAPIKey = os.Getenv("WHODB_ANTHROPIC_API_KEY")
var AnthropicEndpoint = os.Getenv("WHODB_ANTHROPIC_ENDPOINT")
var AnthropicName = os.Getenv("WHODB_ANTHROPIC_NAME")

var OpenAIAPIKey = os.Getenv("WHODB_OPENAI_API_KEY")
var OpenAIEndpoint = os.Getenv("WHODB_OPENAI_ENDPOINT")
var OpenAIName = os.Getenv("WHODB_OPENAI_NAME")

var OllamaName = os.Getenv("WHODB_OLLAMA_NAME")

var OpenAICompatibleEndpoint = os.Getenv("WHODB_OPENAI_COMPATIBLE_ENDPOINT")
var OpenAICompatibleAPIKey = os.Getenv("WHODB_OPENAI_COMPATIBLE_API_KEY")

var CustomModels = common.FilterList(strings.Split(os.Getenv("WHODB_CUSTOM_MODELS"), ","), func(item string) bool {
	return strings.TrimSpace(item) != ""
})

var AllowedOrigins = common.FilterList(strings.Split(os.Getenv("WHODB_ALLOWED_ORIGINS"), ","), func(item string) bool {
	return item != ""
})

var LogLevel = os.Getenv("WHODB_LOG_LEVEL")

var AccessLogFile = os.Getenv("WHODB_ACCESS_LOG_FILE") // where to store the http access logs
var LogFile = os.Getenv("WHODB_LOG_FILE")              // where to store all other non-http logs
var LogFormat = os.Getenv("WHODB_LOG_FORMAT")          // only option right now is "json". leave blank for default format

// Default log paths used when the AccessLogFile and LogFile vars are set to "default".
const DefaultLogDir = "/var/log/whodb"
const DefaultLogFile = DefaultLogDir + "/whodb.log"
const DefaultAccessLogFile = DefaultLogDir + "/whodb.access.log"

var DisableMockDataGeneration = os.Getenv("WHODB_DISABLE_MOCK_DATA_GENERATION")

var ApplicationEnvironment = os.Getenv("WHODB_APPLICATION_ENVIRONMENT")

var ApplicationVersion string

var PosthogAPIKey = "phc_hbXcCoPTdxm5ADL8PmLSYTIUvS6oRWFM2JAK8SMbfnH"
var PosthogHost = "https://us.i.posthog.com"

// IsAWSProviderEnabled controls whether AWS provider functionality is available.
// disabled by default for now until official release
var IsAWSProviderEnabled = os.Getenv("WHODB_ENABLE_AWS_PROVIDER") == "true"

// DisableCredentialForm controls whether the credential form is disabled.
var DisableCredentialForm = os.Getenv("WHODB_DISABLE_CREDENTIAL_FORM") == "true"

// MaxPageSize is the maximum number of rows that can be requested in a single
// page via the Row resolver. Configurable via WHODB_MAX_PAGE_SIZE (default 10000).
var MaxPageSize = getMaxPageSize()

type ChatProvider struct {
	Type       string
	Name       string // Display name/alias for the provider
	APIKey     string
	Endpoint   string
	ProviderId string
	ClientType string // BAML client type (openai-generic, anthropic, aws-bedrock) - only for generic providers
	IsGeneric  bool   // True for generic/custom providers, false for built-in providers
}

// GenericProviderConfig holds configuration for a generic AI provider.
type GenericProviderConfig struct {
	ProviderId string
	Name       string
	ClientType string // "openai-generic", "anthropic", etc.
	BaseURL    string
	APIKey     string
	Models     []string
}

var GenericProviders []GenericProviderConfig

// AddGenericProvider adds a generic provider to the GenericProviders list.
// This is used by EE providers and other dynamic provider registration systems.
func AddGenericProvider(config GenericProviderConfig) {
	GenericProviders = append(GenericProviders, config)
}

func GetConfiguredChatProviders() []ChatProvider {
	var providers []ChatProvider

	if len(OpenAIAPIKey) > 0 {
		name := OpenAIName
		if name == "" {
			name = "OpenAI"
		}
		providers = append(providers, ChatProvider{
			Type:       "OpenAI",
			Name:       name,
			APIKey:     OpenAIAPIKey,
			Endpoint:   GetOpenAIEndpoint(),
			ProviderId: "openai-1",
		})
	}

	if len(AnthropicAPIKey) > 0 {
		name := AnthropicName
		if name == "" {
			name = "Anthropic"
		}
		providers = append(providers, ChatProvider{
			Type:       "Anthropic",
			Name:       name,
			APIKey:     AnthropicAPIKey,
			Endpoint:   GetAnthropicEndpoint(),
			ProviderId: "anthropic-1",
		})
	}

	if len(OpenAICompatibleAPIKey) > 0 && len(OpenAICompatibleEndpoint) > 0 && len(CustomModels) > 0 {
		providers = append(providers, ChatProvider{
			Type:       "OpenAI-Compatible",
			Name:       "OpenAI-Compatible",
			APIKey:     OpenAICompatibleAPIKey,
			Endpoint:   GetOpenAICompatibleEndpoint(),
			ProviderId: "openai-compatible-1",
		})
	}

	// Add all generic providers
	for _, genericProvider := range GenericProviders {
		providers = append(providers, ChatProvider{
			Type:       genericProvider.ProviderId, // Use provider ID as type
			Name:       genericProvider.Name,       // Display name
			APIKey:     genericProvider.APIKey,
			Endpoint:   genericProvider.BaseURL,
			ProviderId: genericProvider.ProviderId,
			ClientType: genericProvider.ClientType, // BAML client type
			IsGeneric:  true,                       // Mark as generic provider
		})
	}

	name := OllamaName
	if name == "" {
		name = "Ollama"
	}
	providers = append(providers, ChatProvider{
		Type:       "Ollama",
		Name:       name,
		APIKey:     "",
		Endpoint:   GetOllamaEndpoint(),
		ProviderId: "ollama-1",
	})

	return providers
}

func GetOllamaEndpoint() string {
	host, port := GetOllamaHost()
	return fmt.Sprintf("http://%s:%s/api", host, port)
}

// GetOllamaHost returns the resolved Ollama host and port, accounting for
// Docker/WSL2 environments and WHODB_OLLAMA_HOST/PORT overrides.
func GetOllamaHost() (string, string) {
	host := "localhost"
	port := "11434"

	if common.IsRunningInsideDocker() {
		host = "host.docker.internal"
	} else if common.IsRunningInsideWSL2() {
		if wslHost := common.GetWSL2WindowsHost(); wslHost != "" {
			host = wslHost
		}
	}

	if OllamaHost != "" {
		host = OllamaHost
	}
	if OllamaPort != "" {
		port = OllamaPort
	}

	return host, port
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

func getMaxPageSize() int {
	val := os.Getenv("WHODB_MAX_PAGE_SIZE")
	if val == "" {
		return 10000
	}
	n, err := strconv.Atoi(val)
	if err != nil || n <= 0 {
		return 10000
	}
	return n
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
