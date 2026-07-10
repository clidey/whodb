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

// Package envconfig provides configuration-loading functions that depend on
// both the env and log packages. It exists to break the circular dependency
// that would occur if env imported log directly.
package envconfig

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/env"
	"github.com/clidey/whodb/core/src/log"
	"github.com/clidey/whodb/core/src/migrate"
	"github.com/clidey/whodb/core/src/security"
	"github.com/clidey/whodb/core/src/types"
)

// GetDefaultDatabaseCredentials reads database credentials from environment
// variables. It first checks WHODB_<TYPE> for a JSON array, then falls back
// to numbered WHODB_<TYPE>_1, WHODB_<TYPE>_2, etc.
func GetDefaultDatabaseCredentials(databaseType string) []types.DatabaseCredentials {
	uppercaseDatabaseType := strings.ToUpper(databaseType)
	credEnvVar := "WHODB_" + uppercaseDatabaseType
	credEnvValue := os.Getenv(credEnvVar)

	if credEnvValue == "" {
		return findAllDatabaseCredentials(databaseType)
	}

	var creds []types.DatabaseCredentials
	err := json.Unmarshal([]byte(credEnvValue), &creds)
	if err != nil {
		log.Error("🔴 [Database Error] Failed to parse database credentials from environment variable! Error: ", err)
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
			log.Error("Unable to parse database credential: ", err)
			break
		}

		profiles = append(profiles, creds)
		i++
	}

	return profiles
}

// GetAWSProvidersFromEnv parses the WHODB_AWS_PROVIDER environment variable
// into a slice of AWS provider configurations, applying defaults.
func GetAWSProvidersFromEnv() ([]env.AWSProviderEnvConfig, error) {
	val := os.Getenv("WHODB_AWS_PROVIDER")
	if val == "" {
		return nil, nil
	}

	var configs []env.AWSProviderEnvConfig
	if err := json.Unmarshal([]byte(val), &configs); err != nil {
		log.Error("[AWS Provider] Failed to parse WHODB_AWS_PROVIDER: ", err)
		return nil, err
	}

	// Apply defaults
	for i := range configs {
		if configs[i].DiscoverRDS == nil {
			configs[i].DiscoverRDS = new(true)
		}
		if configs[i].DiscoverElastiCache == nil {
			configs[i].DiscoverElastiCache = new(true)
		}
		if configs[i].DiscoverDocumentDB == nil {
			configs[i].DiscoverDocumentDB = new(true)
		}
	}

	return configs, nil
}

// GetAzureProvidersFromEnv parses the WHODB_AZURE_PROVIDER environment variable
// into a slice of Azure provider configurations, applying defaults.
func GetAzureProvidersFromEnv() ([]env.AzureProviderEnvConfig, error) {
	val := os.Getenv("WHODB_AZURE_PROVIDER")
	if val == "" {
		return nil, nil
	}

	var configs []env.AzureProviderEnvConfig
	if err := json.Unmarshal([]byte(val), &configs); err != nil {
		log.Error("[Azure Provider] Failed to parse WHODB_AZURE_PROVIDER: ", err)
		return nil, err
	}

	// Apply defaults — all discovery flags default to true
	for i := range configs {
		if configs[i].DiscoverPostgreSQL == nil {
			configs[i].DiscoverPostgreSQL = new(true)
		}
		if configs[i].DiscoverMySQL == nil {
			configs[i].DiscoverMySQL = new(true)
		}
		if configs[i].DiscoverRedis == nil {
			configs[i].DiscoverRedis = new(true)
		}
		if configs[i].DiscoverCosmosDB == nil {
			configs[i].DiscoverCosmosDB = new(true)
		}
	}

	return configs, nil
}

// GetGCPProvidersFromEnv parses the WHODB_GCP_PROVIDER environment variable
// into a slice of GCP provider configurations, applying defaults.
func GetGCPProvidersFromEnv() ([]env.GCPProviderEnvConfig, error) {
	val := os.Getenv("WHODB_GCP_PROVIDER")
	if val == "" {
		return nil, nil
	}

	var configs []env.GCPProviderEnvConfig
	if err := json.Unmarshal([]byte(val), &configs); err != nil {
		log.Error("[GCP Provider] Failed to parse WHODB_GCP_PROVIDER: ", err)
		return nil, err
	}

	// Apply defaults
	for i := range configs {
		if configs[i].DiscoverCloudSQL == nil {
			configs[i].DiscoverCloudSQL = new(true)
		}
		if configs[i].DiscoverAlloyDB == nil {
			configs[i].DiscoverAlloyDB = new(true)
		}
		if configs[i].DiscoverMemorystore == nil {
			configs[i].DiscoverMemorystore = new(true)
		}
	}

	return configs, nil
}

// ParseGenericProviders reads WHODB_AI_GENERIC_* environment variables to
// discover generic AI provider configurations.
//
// Environment variable format:
//
//	WHODB_AI_GENERIC_<ID>_NAME="Provider Display Name"
//	WHODB_AI_GENERIC_<ID>_TYPE="openai-generic"
//	WHODB_AI_GENERIC_<ID>_BASE_URL="https://api.example.com/v1"
//	WHODB_AI_GENERIC_<ID>_API_KEY="sk-..."
//	WHODB_AI_GENERIC_<ID>_MODELS="model-1,model-2,model-3"
//	WHODB_AI_GENERIC_<ID>_ICON="https://example.com/icon.svg"
func ParseGenericProviders() []env.GenericProviderConfig {
	var providers []env.GenericProviderConfig
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
		icon := os.Getenv(prefix + "ICON")

		// Validate required fields
		if baseURL == "" || modelsStr == "" {
			log.Warnf("Incomplete generic provider config for %s, skipping (missing base_url or models)", providerID)
			continue
		}

		// Parse models
		models := common.FilterList(strings.Split(modelsStr, ","), func(item string) bool {
			return strings.TrimSpace(item) != ""
		})

		if len(models) == 0 {
			log.Warnf("No models specified for generic provider %s, skipping", providerID)
			continue
		}

		// Default values
		if name == "" {
			name = providerID
		}
		if clientType == "" {
			clientType = "openai-generic" // Default to OpenAI-compatible
		}

		providers = append(providers, env.GenericProviderConfig{
			ProviderId: strings.ToLower(providerID),
			Name:       name,
			ClientType: clientType,
			BaseURL:    baseURL,
			APIKey:     apiKey,
			Models:     models,
			Icon:       icon,
		})

		processed[providerID] = true
	}

	if len(providers) > 0 {
		log.Debugf("Discovered %d generic AI provider(s)", len(providers))
		for _, provider := range providers {
			log.Debugf("  - %s (%s) with %d model(s)", provider.Name, provider.ProviderId, len(provider.Models))
		}
	}

	return providers
}

// GetConfiguredChatProviders returns all configured AI chat providers from
// environment variables and dynamically registered generic providers.
func GetConfiguredChatProviders() []env.ChatProvider {
	var providers []env.ChatProvider

	if len(env.OpenAIAPIKey) > 0 {
		name := env.OpenAIName
		if name == "" {
			name = "OpenAI"
		}
		providers = append(providers, env.ChatProvider{
			Type:       "OpenAI",
			Name:       name,
			APIKey:     env.OpenAIAPIKey,
			Endpoint:   env.GetOpenAIEndpoint(),
			ProviderId: "openai-1",
		})
	}

	if len(env.AnthropicAPIKey) > 0 {
		name := env.AnthropicName
		if name == "" {
			name = "Anthropic"
		}
		providers = append(providers, env.ChatProvider{
			Type:       "Anthropic",
			Name:       name,
			APIKey:     env.AnthropicAPIKey,
			Endpoint:   env.GetAnthropicEndpoint(),
			ProviderId: "anthropic-1",
		})
	}

	// Flag if legacy OpenAI-Compatible env vars are still set
	if os.Getenv("WHODB_OPENAI_COMPATIBLE_ENDPOINT") != "" || os.Getenv("WHODB_OPENAI_COMPATIBLE_API_KEY") != "" || os.Getenv("WHODB_CUSTOM_MODELS") != "" {
		migrate.DeprecatedOpenAICompatibleEnv = true
	}

	// Add all generic providers
	for _, genericProvider := range env.GenericProviders {
		providers = append(providers, env.ChatProvider{
			Type:       genericProvider.ProviderId,
			Name:       genericProvider.Name,
			APIKey:     genericProvider.APIKey,
			Endpoint:   common.ResolveLocalURL(genericProvider.BaseURL),
			ProviderId: genericProvider.ProviderId,
			ClientType: genericProvider.ClientType,
			IsGeneric:  true,
			Icon:       genericProvider.Icon,
		})
	}

	// Only show local providers (LM Studio, Ollama) when explicitly configured
	// or when not running in hosted EE mode.
	showLocalProviders := !env.IsEnterpriseEdition || env.LMStudioBaseURL != "" || env.LMStudioAPIKey != ""
	if showLocalProviders {
		name := env.LMStudioName
		if name == "" {
			name = "LM Studio"
		}
		providers = append(providers, env.ChatProvider{
			Type:       "LMStudio",
			Name:       name,
			APIKey:     env.LMStudioAPIKey,
			Endpoint:   env.GetLMStudioEndpoint(),
			ProviderId: "lmstudio-1",
		})
	}

	showOllama := !env.IsEnterpriseEdition || env.OllamaHost != ""
	if showOllama {
		name := env.OllamaName
		if name == "" {
			name = "Ollama"
		}
		providers = append(providers, env.ChatProvider{
			Type:       "Ollama",
			Name:       name,
			APIKey:     "",
			Endpoint:   env.GetOllamaEndpoint(),
			ProviderId: "ollama-1",
		})
	}

	return providers
}

// SupplementalCredentialResolver resolves credentials for a provider ID not found in env config.
// EE registers one at boot to look up platform-managed providers from the database.
type SupplementalCredentialResolver func(providerId string) (token string, endpoint string, found bool)

var supplementalResolver SupplementalCredentialResolver

// RegisterSupplementalCredentialResolver allows EE to register a callback that resolves
// credentials from the platform database when the provider is not in env config.
func RegisterSupplementalCredentialResolver(fn SupplementalCredentialResolver) {
	supplementalResolver = fn
}

// ResolveProviderCredentials looks up a provider by ID and resolves credentials.
// Request-level values take precedence over environment-configured values.
func ResolveProviderCredentials(providerId, requestToken, requestEndpoint, requestModelType string) env.ResolvedCredentials {
	// In hosted/production deployments, never let a client choose the outbound
	// endpoint for a server-configured provider: that would be an SSRF vector
	// (and could exfiltrate the shared provider API key). Drop the per-request
	// endpoint override and rely only on server-configured endpoints.
	if security.EgressRestricted() {
		requestEndpoint = ""
	}

	result := env.ResolvedCredentials{
		ModelType: requestModelType,
		Token:     requestToken,
		Endpoint:  requestEndpoint,
	}
	if providerId == "" {
		return result
	}
	// A server-configured API key must never be paired with a client-supplied
	// endpoint: that would send the shared secret to an attacker-chosen host.
	// When falling back to the server's key, also use the server's endpoint.
	clientSuppliedEndpoint := requestEndpoint != ""
	for _, provider := range GetConfiguredChatProviders() {
		if provider.ProviderId != providerId {
			continue
		}
		if result.Token == "" {
			result.Token = provider.APIKey
			if clientSuppliedEndpoint {
				result.Endpoint = provider.Endpoint
			}
		}
		if result.Endpoint == "" {
			result.Endpoint = provider.Endpoint
		}
		return result
	}
	if result.Token == "" && supplementalResolver != nil {
		token, endpoint, found := supplementalResolver(providerId)
		if found {
			result.Token = token
			// As above, don't pair a server-resolved key with a client endpoint.
			if result.Endpoint == "" || clientSuppliedEndpoint {
				result.Endpoint = endpoint
			}
		}
	}
	return result
}
