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
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/clidey/whodb/core/src/env"
	"github.com/clidey/whodb/core/src/envconfig"
	gcpinfra "github.com/clidey/whodb/core/src/gcp"
	"github.com/clidey/whodb/core/src/log"
	"github.com/clidey/whodb/core/src/providers"
	gcpprovider "github.com/clidey/whodb/core/src/providers/gcp"
)

var (
	// ErrGCPProviderNotFound indicates the GCP provider doesn't exist.
	ErrGCPProviderNotFound = errors.New("gcp provider not found")

	// ErrGCPProviderAlreadyExists indicates a GCP provider with this ID exists.
	ErrGCPProviderAlreadyExists = errors.New("gcp provider already exists")
)

// GCPProviderConfig holds the configuration for a GCP provider.
// All fields are non-sensitive — no credentials are stored.
type GCPProviderConfig struct {
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

// GCPProviderState holds the runtime state of a GCP provider including
// its configuration, provider instance, status, and discovery statistics.
type GCPProviderState struct {
	Config          *GCPProviderConfig
	Provider        *gcpprovider.Provider
	Status          string
	LastDiscoveryAt *time.Time
	DiscoveredCount int
	Error           string
}

var (
	gcpProvidersMu sync.RWMutex
	gcpProviders   = make(map[string]*GCPProviderState)
	gcpSkipPersist bool
)

// GetGCPProviders returns all configured GCP provider states.
func GetGCPProviders() []*GCPProviderState {
	gcpProvidersMu.RLock()
	defer gcpProvidersMu.RUnlock()

	result := make([]*GCPProviderState, 0, len(gcpProviders))
	for _, state := range gcpProviders {
		result = append(result, state)
	}
	return result
}

// GetGCPProvider retrieves a GCP provider state by ID.
func GetGCPProvider(id string) (*GCPProviderState, error) {
	gcpProvidersMu.RLock()
	defer gcpProvidersMu.RUnlock()

	state, exists := gcpProviders[id]
	if !exists {
		return nil, ErrGCPProviderNotFound
	}
	return state, nil
}

// AddGCPProvider creates and registers a new GCP provider with the given configuration.
func AddGCPProvider(cfg *GCPProviderConfig) (*GCPProviderState, error) {
	gcpProvidersMu.Lock()

	if _, exists := gcpProviders[cfg.ID]; exists {
		gcpProvidersMu.Unlock()
		return nil, ErrGCPProviderAlreadyExists
	}

	providerCfg := configToGCPProviderConfig(cfg)
	provider, err := gcpprovider.New(providerCfg)
	if err != nil {
		gcpProvidersMu.Unlock()
		return nil, err
	}

	registry := providers.GetDefaultRegistry()
	if err := registry.Register(provider); err != nil {
		gcpProvidersMu.Unlock()
		return nil, err
	}

	state := &GCPProviderState{
		Config:   cfg,
		Provider: provider,
		Status:   "Connected",
	}

	gcpProviders[cfg.ID] = state
	gcpProvidersMu.Unlock()

	if !gcpSkipPersist {
		if err := saveGCPProvidersToFile(); err != nil {
			log.Warnf("Failed to persist GCP provider after add: %v", err)
		}
	}

	return state, nil
}

// UpdateGCPProvider updates an existing GCP provider with new configuration.
func UpdateGCPProvider(id string, cfg *GCPProviderConfig) (*GCPProviderState, error) {
	cfg.ID = id
	providerCfg := configToGCPProviderConfig(cfg)
	provider, err := gcpprovider.New(providerCfg)
	if err != nil {
		return nil, err
	}

	gcpProvidersMu.Lock()
	if _, exists := gcpProviders[id]; !exists {
		gcpProvidersMu.Unlock()
		return nil, ErrGCPProviderNotFound
	}

	registry := providers.GetDefaultRegistry()
	if err := registry.Unregister(id); err != nil {
		log.Warnf("Failed to unregister GCP provider %s during update: %v", id, err)
	}

	if err := registry.Register(provider); err != nil {
		gcpProvidersMu.Unlock()
		return nil, err
	}

	state := &GCPProviderState{
		Config:   cfg,
		Provider: provider,
		Status:   "Connected",
	}

	gcpProviders[id] = state
	gcpProvidersMu.Unlock()

	if !gcpSkipPersist {
		if err := saveGCPProvidersToFile(); err != nil {
			log.Warnf("Failed to persist GCP provider after update: %v", err)
		}
	}

	return state, nil
}

// RemoveGCPProvider removes a GCP provider by ID.
func RemoveGCPProvider(id string) error {
	gcpProvidersMu.Lock()

	if _, exists := gcpProviders[id]; !exists {
		gcpProvidersMu.Unlock()
		return ErrGCPProviderNotFound
	}

	registry := providers.GetDefaultRegistry()
	if err := registry.Unregister(id); err != nil {
		log.Warnf("Failed to unregister GCP provider %s during removal: %v", id, err)
	}

	delete(gcpProviders, id)
	gcpProvidersMu.Unlock()

	if !gcpSkipPersist {
		if err := saveGCPProvidersToFile(); err != nil {
			log.Warnf("Failed to persist GCP provider after remove: %v", err)
		}
	}

	return nil
}

// TestGCPProvider tests the GCP connection for a provider and updates its status.
func TestGCPProvider(id string) (string, error) {
	gcpProvidersMu.RLock()
	state, exists := gcpProviders[id]
	if !exists {
		gcpProvidersMu.RUnlock()
		return "Disconnected", ErrGCPProviderNotFound
	}
	provider := state.Provider
	gcpProvidersMu.RUnlock()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	testErr := provider.TestConnection(ctx)

	gcpProvidersMu.Lock()
	state, exists = gcpProviders[id]
	if !exists {
		gcpProvidersMu.Unlock()
		return "Disconnected", ErrGCPProviderNotFound
	}

	if testErr != nil {
		state.Status = "Error"
		state.Error = testErr.Error()
		gcpProvidersMu.Unlock()
		return "Error", testErr
	}

	state.Status = "Connected"
	state.Error = ""
	gcpProvidersMu.Unlock()
	return "Connected", nil
}

// RefreshGCPProvider triggers a re-discovery of connections for the specified provider.
func RefreshGCPProvider(id string) (*GCPProviderState, error) {
	log.Debugf("RefreshGCPProvider called for id=%s", id)

	gcpProvidersMu.Lock()
	state, exists := gcpProviders[id]
	if !exists {
		gcpProvidersMu.Unlock()
		log.Warnf("RefreshGCPProvider: provider not found id=%s", id)
		return nil, ErrGCPProviderNotFound
	}
	state.Status = "Discovering"
	log.Debugf("RefreshGCPProvider: provider found, id=%s, projectID=%s, region=%s, authMethod=%s",
		state.Config.ID, state.Config.ProjectID, state.Config.Region, state.Config.AuthMethod)
	gcpProvidersMu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	registry := providers.GetDefaultRegistry()
	log.Debugf("RefreshGCPProvider: calling registry.RefreshDiscovery for id=%s", id)
	conns, err := registry.RefreshDiscovery(ctx, id)
	log.Debugf("RefreshGCPProvider: registry.RefreshDiscovery returned %d connections, err=%v", len(conns), err)

	// Run connectivity checks on refresh
	if len(conns) > 0 {
		gcpprovider.CheckConnectivity(conns)
	}

	gcpProvidersMu.Lock()
	defer gcpProvidersMu.Unlock()

	state, exists = gcpProviders[id]
	if !exists {
		return nil, ErrGCPProviderNotFound
	}

	now := time.Now()
	state.LastDiscoveryAt = &now
	state.DiscoveredCount = len(conns)

	if err != nil {
		if len(conns) > 0 {
			state.Status = "Connected"
			state.Error = err.Error()
			return state, nil
		}
		state.Status = "Error"
		state.Error = err.Error()
		return state, err
	}

	state.Status = "Connected"
	state.Error = ""
	return state, nil
}

func configToGCPProviderConfig(cfg *GCPProviderConfig) *gcpprovider.Config {
	return &gcpprovider.Config{
		ID:                    cfg.ID,
		Name:                  cfg.Name,
		ProjectID:             cfg.ProjectID,
		Region:                cfg.Region,
		AuthMethod:            gcpinfra.AuthMethod(cfg.AuthMethod),
		ServiceAccountKeyPath: cfg.ServiceAccountKeyPath,
		DiscoverCloudSQL:      cfg.DiscoverCloudSQL,
		DiscoverAlloyDB:       cfg.DiscoverAlloyDB,
		DiscoverMemorystore:   cfg.DiscoverMemorystore,
	}
}

// GenerateGCPProviderID creates a unique provider ID from a name and region.
func GenerateGCPProviderID(name, region string) string {
	return "gcp-" + sanitizeID(name) + "-" + region
}

// GetProviderType determines the provider type for a given provider ID.
// This is used by generic resolvers to dispatch to the correct provider.
func GetProviderType(id string) providers.ProviderType {
	awsProvidersMu.RLock()
	if _, exists := awsProviders[id]; exists {
		awsProvidersMu.RUnlock()
		return providers.ProviderTypeAWS
	}
	awsProvidersMu.RUnlock()

	gcpProvidersMu.RLock()
	if _, exists := gcpProviders[id]; exists {
		gcpProvidersMu.RUnlock()
		return providers.ProviderTypeGCP
	}
	gcpProvidersMu.RUnlock()

	return ""
}

// InitGCPProvidersFromEnv initializes GCP providers from the WHODB_GCP_PROVIDER
// environment variable.
func InitGCPProvidersFromEnv() error {
	if !env.IsGCPProviderEnabled {
		return nil
	}

	envConfigs, err := envconfig.GetGCPProvidersFromEnv()
	if err != nil {
		return err
	}

	if len(envConfigs) == 0 {
		return nil
	}

	for i, envCfg := range envConfigs {
		name := envCfg.Name
		if name == "" {
			name = fmt.Sprintf("GCP-%d", i+1)
		}
		id := GenerateGCPProviderID(name, envCfg.Region)

		authMethod := "default"
		if envCfg.ServiceAccountKeyPath != "" {
			authMethod = "service-account-key"
		}

		discoverCloudSQL := true
		if envCfg.DiscoverCloudSQL != nil {
			discoverCloudSQL = *envCfg.DiscoverCloudSQL
		}

		discoverAlloyDB := true
		if envCfg.DiscoverAlloyDB != nil {
			discoverAlloyDB = *envCfg.DiscoverAlloyDB
		}

		discoverMemorystore := true
		if envCfg.DiscoverMemorystore != nil {
			discoverMemorystore = *envCfg.DiscoverMemorystore
		}

		cfg := &GCPProviderConfig{
			ID:                    id,
			Name:                  name,
			ProjectID:             envCfg.ProjectID,
			Region:                envCfg.Region,
			AuthMethod:            authMethod,
			ServiceAccountKeyPath: envCfg.ServiceAccountKeyPath,
			DiscoverCloudSQL:      discoverCloudSQL,
			DiscoverAlloyDB:       discoverAlloyDB,
			DiscoverMemorystore:   discoverMemorystore,
		}

		if _, err := AddGCPProvider(cfg); err != nil {
			log.Warnf("Failed to initialize GCP provider %s: %v", name, err)
		} else {
			log.Debugf("Initialized GCP provider: %s (%s/%s)", name, envCfg.ProjectID, envCfg.Region)
		}
	}

	return nil
}
