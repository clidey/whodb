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
var AnthropicName = os.Getenv("WHODB_ANTHROPIC_NAME")

var OpenAIAPIKey = os.Getenv("WHODB_OPENAI_API_KEY")
var OpenAIEndpoint = os.Getenv("WHODB_OPENAI_ENDPOINT")
var OpenAIName = os.Getenv("WHODB_OPENAI_NAME")

var OllamaName = os.Getenv("WHODB_OLLAMA_NAME")

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

var ApplicationEnvironment = os.Getenv("WHODB_APPLICATION_ENVIRONMENT")

var ApplicationVersion string

var PosthogAPIKey = "phc_hbXcCoPTdxm5ADL8PmLSYTIUvS6oRWFM2JAK8SMbfnH"
var PosthogHost = "https://us.i.posthog.com"

// IsAWSProviderEnabled controls whether AWS provider functionality is available.
// disabled by default for now until official release
var IsAWSProviderEnabled = os.Getenv("WHODB_ENABLE_AWS_PROVIDER") == "true"

// DisableCredentialForm controls whether the credential form is disabled.
var DisableCredentialForm = os.Getenv("WHODB_DISABLE_CREDENTIAL_FORM") == "true"

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

func GetAWSProvidersFromEnv() ([]AWSProviderEnvConfig, error) {
	val := os.Getenv("WHODB_AWS_PROVIDER")
	if val == "" {
		return nil, nil
	}

	var configs []AWSProviderEnvConfig
	if err := json.Unmarshal([]byte(val), &configs); err != nil {
		log.Logger.Error("[AWS Provider] Failed to parse WHODB_AWS_PROVIDER: ", err)
		return nil, err
	}

	// Apply defaults
	for i := range configs {
		if configs[i].Auth == "" {
			configs[i].Auth = "default"
		}
		if configs[i].DiscoverRDS == nil {
			t := true
			configs[i].DiscoverRDS = &t
		}
		if configs[i].DiscoverElastiCache == nil {
			t := true
			configs[i].DiscoverElastiCache = &t
		}
		if configs[i].DiscoverDocumentDB == nil {
			t := true
			configs[i].DiscoverDocumentDB = &t
		}
	}

	return configs, nil
}

// parseGenericProviders reads environment variables to discover generic AI provider configurations.
// Environment variable format:
// WHODB_AI_GENERIC_<ID>_NAME="Provider Display Name"
// WHODB_AI_GENERIC_<ID>_TYPE="openai-generic"
// WHODB_AI_GENERIC_<ID>_BASE_URL="https://api.example.com/v1"
// WHODB_AI_GENERIC_<ID>_API_KEY="sk-..."
// WHODB_AI_GENERIC_<ID>_MODELS="model-1,model-2,model-3"
func parseGenericProviders() []GenericProviderConfig {
	var providers []GenericProviderConfig
	processed := make(map[string]bool)

	// Iterate through all environment variables to find WHODB_AI_GENERIC_* patterns
	for _, envVar := range os.Environ() {
		if !strings.HasPrefix(envVar, "WHODB_AI_GENERIC_") {
			continue
		}

		parts := strings.SplitN(envVar, "=", 2)
		key := parts[0]

		// Extract provider ID from key (e.g., "MISTRAL" from "WHODB_AI_GENERIC_MISTRAL_NAME")
		keyParts := strings.Split(key, "_")
		if len(keyParts) < 5 {
			continue
		}

		// ID is everything between WHODB_AI_GENERIC_ and the final field
		idParts := keyParts[3 : len(keyParts)-1]
		providerID := strings.Join(idParts, "_")

		if processed[providerID] {
			continue
		}

		// Read all fields for this provider
		prefix := fmt.Sprintf("WHODB_AI_GENERIC_%s_", providerID)
		name := os.Getenv(prefix + "NAME")
		clientType := os.Getenv(prefix + "TYPE")
		baseURL := os.Getenv(prefix + "BASE_URL")
		apiKey := os.Getenv(prefix + "API_KEY")
		modelsStr := os.Getenv(prefix + "MODELS")

		// Validate required fields
		if baseURL == "" || modelsStr == "" {
			log.Logger.Warnf("Incomplete generic provider config for %s, skipping (missing base_url or models)", providerID)
			continue
		}

		// Parse models
		models := common.FilterList(strings.Split(modelsStr, ","), func(item string) bool {
			return strings.TrimSpace(item) != ""
		})

		if len(models) == 0 {
			log.Logger.Warnf("No models specified for generic provider %s, skipping", providerID)
			continue
		}

		// Default values
		if name == "" {
			name = providerID
		}
		if clientType == "" {
			clientType = "openai-generic" // Default to OpenAI-compatible
		}

		providers = append(providers, GenericProviderConfig{
			ProviderId: strings.ToLower(providerID),
			Name:       name,
			ClientType: clientType,
			BaseURL:    baseURL,
			APIKey:     apiKey,
			Models:     models,
		})

		processed[providerID] = true
	}

	if len(providers) > 0 {
		log.Logger.Infof("Discovered %d generic AI provider(s)", len(providers))
		for _, provider := range providers {
			log.Logger.Infof("  - %s (%s) with %d model(s)", provider.Name, provider.ProviderId, len(provider.Models))
		}
	}

	return providers
}

func init() {
	// Parse generic providers at initialization
	// They will be registered with the LLM provider registry by src.InitializeEngine()
	GenericProviders = parseGenericProviders()
}
