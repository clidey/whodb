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
	"github.com/clidey/whodb/core/src/common/datadir"
	"github.com/clidey/whodb/core/src/env"
	"github.com/clidey/whodb/core/src/log"
)

// awsSection is the structure stored in the "aws" section of config.json.
type awsSection struct {
	Providers []persistedProviderConfig `json:"providers"`
}

// persistedProviderConfig is the on-disk format for provider configs.
type persistedProviderConfig struct {
	ID                  string `json:"id"`
	Name                string `json:"name"`
	Region              string `json:"region"`
	AuthMethod          string `json:"authMethod"`
	ProfileName         string `json:"profileName,omitempty"`
	DiscoverRDS         bool   `json:"discoverRDS"`
	DiscoverElastiCache bool   `json:"discoverElastiCache"`
	DiscoverDocumentDB  bool   `json:"discoverDocumentDB"`
}

// getConfigOptions returns the datadir options for this environment.
func getConfigOptions() datadir.Options {
	return datadir.Options{
		AppName:           "whodb",
		EnterpriseEdition: env.IsEnterpriseEdition,
		Development:       env.IsDevelopment,
	}
}

// saveProvidersToFile persists the current provider configs to the unified config file.
// This is called automatically after add/update/remove operations.
func saveProvidersToFile() error {
	awsProvidersMu.RLock()
	providers := make([]persistedProviderConfig, 0, len(awsProviders))
	for _, state := range awsProviders {
		providers = append(providers, persistedProviderConfig{
			ID:                  state.Config.ID,
			Name:                state.Config.Name,
			Region:              state.Config.Region,
			AuthMethod:          state.Config.AuthMethod,
			ProfileName:         state.Config.ProfileName,
			DiscoverRDS:         state.Config.DiscoverRDS,
			DiscoverElastiCache: state.Config.DiscoverElastiCache,
			DiscoverDocumentDB:  state.Config.DiscoverDocumentDB,
		})
	}
	awsProvidersMu.RUnlock()

	section := awsSection{Providers: providers}
	opts := getConfigOptions()

	if err := config.WriteSection(config.SectionAWS, section, opts); err != nil {
		log.Warnf("Failed to save AWS providers: %v", err)
		return err
	}

	log.Debugf("Saved %d AWS provider(s) to config", len(providers))
	return nil
}

// LoadProvidersFromFile loads persisted provider configs from the unified config file.
// This should be called during server startup.
func LoadProvidersFromFile() error {
	if !env.IsAWSProviderEnabled {
		return nil
	}
	opts := getConfigOptions()

	var section awsSection
	if err := config.ReadSection(config.SectionAWS, &section, opts); err != nil {
		log.Warnf("Failed to read AWS providers from config: %v", err)
		return err
	}

	if len(section.Providers) == 0 {
		return nil
	}

	// Prevent circular saves during load
	skipPersist = true
	defer func() { skipPersist = false }()

	loadedCount := 0
	for _, cfg := range section.Providers {
		// Convert to AWSProviderConfig
		providerCfg := &AWSProviderConfig{
			ID:                  cfg.ID,
			Name:                cfg.Name,
			Region:              cfg.Region,
			AuthMethod:          cfg.AuthMethod,
			ProfileName:         cfg.ProfileName,
			DiscoverRDS:         cfg.DiscoverRDS,
			DiscoverElastiCache: cfg.DiscoverElastiCache,
			DiscoverDocumentDB:  cfg.DiscoverDocumentDB,
		}

		// Check if this provider already exists (e.g., from env var)
		awsProvidersMu.RLock()
		_, exists := awsProviders[cfg.ID]
		awsProvidersMu.RUnlock()

		if exists {
			log.Debugf("Skipping persisted provider %s - already registered", cfg.ID)
			continue
		}

		// Add the provider
		_, err := AddAWSProvider(providerCfg)
		if err != nil {
			log.Warnf("Failed to load persisted provider %s: %v", cfg.Name, err)
			continue
		}

		loadedCount++
	}

	if loadedCount > 0 {
		log.Infof("Loaded %d AWS provider(s) from config", loadedCount)
	}

	return nil
}
