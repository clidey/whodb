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

package settings

import (
	"github.com/clidey/whodb/core/src/common/config"
	"github.com/clidey/whodb/core/src/env"
	"github.com/clidey/whodb/core/src/log"
)

// azureSection is the structure stored in the "azure" section of config.json.
type azureSection struct {
	Providers []persistedAzureProviderConfig `json:"providers"`
}

// persistedAzureProviderConfig is the on-disk format for Azure provider configs.
// ClientSecret is intentionally excluded — it is never persisted.
type persistedAzureProviderConfig struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	SubscriptionID     string `json:"subscriptionId"`
	TenantID           string `json:"tenantId,omitempty"`
	ClientID           string `json:"clientId,omitempty"`
	AuthMethod         string `json:"authMethod"`
	ResourceGroup      string `json:"resourceGroup,omitempty"`
	DiscoverPostgreSQL bool   `json:"discoverPostgreSQL"`
	DiscoverMySQL      bool   `json:"discoverMySQL"`
	DiscoverRedis      bool   `json:"discoverRedis"`
	DiscoverCosmosDB   bool   `json:"discoverCosmosDB"`
}

// saveAzureProvidersToFile persists the current Azure provider configs.
func saveAzureProvidersToFile() error {
	azureProvidersMu.RLock()
	providersList := make([]persistedAzureProviderConfig, 0, len(azureProviders))
	for _, state := range azureProviders {
		providersList = append(providersList, persistedAzureProviderConfig{
			ID:                 state.Config.ID,
			Name:               state.Config.Name,
			SubscriptionID:     state.Config.SubscriptionID,
			TenantID:           state.Config.TenantID,
			ClientID:           state.Config.ClientID,
			AuthMethod:         state.Config.AuthMethod,
			ResourceGroup:      state.Config.ResourceGroup,
			DiscoverPostgreSQL: state.Config.DiscoverPostgreSQL,
			DiscoverMySQL:      state.Config.DiscoverMySQL,
			DiscoverRedis:      state.Config.DiscoverRedis,
			DiscoverCosmosDB:   state.Config.DiscoverCosmosDB,
		})
	}
	azureProvidersMu.RUnlock()

	section := azureSection{Providers: providersList}
	opts := getConfigOptions()

	if err := config.WriteSection(config.SectionAzure, section, opts); err != nil {
		log.Warnf("Failed to save Azure providers: %v", err)
		return err
	}

	log.Debugf("Saved %d Azure provider(s) to config", len(providersList))
	return nil
}

// LoadAzureProvidersFromFile loads persisted Azure provider configs from config.json.
// This should be called during server startup.
func LoadAzureProvidersFromFile() error {
	if !env.IsAzureProviderEnabled {
		return nil
	}
	opts := getConfigOptions()

	var section azureSection
	if err := config.ReadSection(config.SectionAzure, &section, opts); err != nil {
		log.Warnf("Failed to read Azure providers from config: %v", err)
		return err
	}

	if len(section.Providers) == 0 {
		return nil
	}

	// Prevent circular saves during load
	skipAzurePersist = true
	defer func() { skipAzurePersist = false }()

	loadedCount := 0
	for _, cfg := range section.Providers {
		providerCfg := &AzureProviderConfig{
			ID:                 cfg.ID,
			Name:               cfg.Name,
			SubscriptionID:     cfg.SubscriptionID,
			TenantID:           cfg.TenantID,
			ClientID:           cfg.ClientID,
			AuthMethod:         cfg.AuthMethod,
			ResourceGroup:      cfg.ResourceGroup,
			DiscoverPostgreSQL: cfg.DiscoverPostgreSQL,
			DiscoverMySQL:      cfg.DiscoverMySQL,
			DiscoverRedis:      cfg.DiscoverRedis,
			DiscoverCosmosDB:   cfg.DiscoverCosmosDB,
		}

		// Check if already registered (e.g., from env var)
		azureProvidersMu.RLock()
		_, exists := azureProviders[cfg.ID]
		azureProvidersMu.RUnlock()

		if exists {
			log.Debugf("Skipping persisted Azure provider %s - already registered", cfg.ID)
			continue
		}

		// No client secret available from disk — for service principal auth,
		// the secret must come from the WHODB_AZURE_PROVIDER env var.
		_, err := AddAzureProvider(providerCfg, "")
		if err != nil {
			log.Warnf("Failed to load persisted Azure provider %s: %v", cfg.Name, err)
			continue
		}

		loadedCount++
	}

	if loadedCount > 0 {
		log.Debugf("Loaded %d Azure provider(s) from config", loadedCount)
	}

	return nil
}
