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
	"github.com/clidey/whodb/core/src/types"
)

// GetDefaultDatabaseCredentials reads database credentials from environment
// variables. It first checks WHODB_<TYPE> for a JSON array, then falls back
// to numbered WHODB_<TYPE>_1, WHODB_<TYPE>_2, etc.
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
		})

		processed[providerID] = true
	}

	if len(providers) > 0 {
		log.Infof("Discovered %d generic AI provider(s)", len(providers))
		for _, provider := range providers {
			log.Infof("  - %s (%s) with %d model(s)", provider.Name, provider.ProviderId, len(provider.Models))
		}
	}

	return providers
}
