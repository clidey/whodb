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

// gcpSection is the structure stored in the "gcp" section of config.json.
type gcpSection struct {
	Providers []persistedGCPProviderConfig `json:"providers"`
}

// persistedGCPProviderConfig is the on-disk format for GCP provider configs.
type persistedGCPProviderConfig struct {
	ID                    string `json:"id"`
	Name                  string `json:"name"`
	ProjectID             string `json:"projectId"`
	Region                string `json:"region"`
	AuthMethod            string `json:"authMethod"`
	ServiceAccountKeyPath string `json:"serviceAccountKeyPath,omitempty"`
	DiscoverCloudSQL      bool   `json:"discoverCloudSQL"`
	DiscoverAlloyDB       bool   `json:"discoverAlloyDB"`
	DiscoverMemorystore   bool   `json:"discoverMemorystore"`
}

// saveGCPProvidersToFile persists the current GCP provider configs to the unified config file.
func saveGCPProvidersToFile() error {
	gcpProvidersMu.RLock()
	providersList := make([]persistedGCPProviderConfig, 0, len(gcpProviders))
	for _, state := range gcpProviders {
		providersList = append(providersList, persistedGCPProviderConfig{
			ID:                    state.Config.ID,
			Name:                  state.Config.Name,
			ProjectID:             state.Config.ProjectID,
			Region:                state.Config.Region,
			AuthMethod:            state.Config.AuthMethod,
			ServiceAccountKeyPath: state.Config.ServiceAccountKeyPath,
			DiscoverCloudSQL:      state.Config.DiscoverCloudSQL,
			DiscoverAlloyDB:       state.Config.DiscoverAlloyDB,
			DiscoverMemorystore:   state.Config.DiscoverMemorystore,
		})
	}
	gcpProvidersMu.RUnlock()

	section := gcpSection{Providers: providersList}
	opts := getConfigOptions()

	if err := config.WriteSection(config.SectionGCP, section, opts); err != nil {
		log.Warnf("Failed to save GCP providers: %v", err)
		return err
	}

	log.Debugf("Saved %d GCP provider(s) to config", len(providersList))
	return nil
}

// LoadGCPProvidersFromFile loads persisted GCP provider configs from the unified config file.
func LoadGCPProvidersFromFile() error {
	if !env.IsGCPProviderEnabled {
		return nil
	}
	opts := getConfigOptions()

	var section gcpSection
	if err := config.ReadSection(config.SectionGCP, &section, opts); err != nil {
		log.Warnf("Failed to read GCP providers from config: %v", err)
		return err
	}

	if len(section.Providers) == 0 {
		return nil
	}

	// Prevent circular saves during load
	gcpSkipPersist = true
	defer func() { gcpSkipPersist = false }()

	loadedCount := 0
	for _, cfg := range section.Providers {
		providerCfg := &GCPProviderConfig{
			ID:                    cfg.ID,
			Name:                  cfg.Name,
			ProjectID:             cfg.ProjectID,
			Region:                cfg.Region,
			AuthMethod:            cfg.AuthMethod,
			ServiceAccountKeyPath: cfg.ServiceAccountKeyPath,
			DiscoverCloudSQL:      cfg.DiscoverCloudSQL,
			DiscoverAlloyDB:       cfg.DiscoverAlloyDB,
			DiscoverMemorystore:   cfg.DiscoverMemorystore,
		}

		// Check if this provider already exists (e.g., from env var)
		gcpProvidersMu.RLock()
		_, exists := gcpProviders[cfg.ID]
		gcpProvidersMu.RUnlock()

		if exists {
			log.Debugf("Skipping persisted GCP provider %s - already registered", cfg.ID)
			continue
		}

		_, err := AddGCPProvider(providerCfg)
		if err != nil {
			log.Warnf("Failed to load persisted GCP provider %s: %v", cfg.Name, err)
			continue
		}

		loadedCount++
	}

	if loadedCount > 0 {
		log.Debugf("Loaded %d GCP provider(s) from config", loadedCount)
	}

	return nil
}
