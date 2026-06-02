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

	azureinfra "github.com/clidey/whodb/core/src/azure"
	"github.com/clidey/whodb/core/src/env"
	"github.com/clidey/whodb/core/src/envconfig"
	"github.com/clidey/whodb/core/src/log"
	"github.com/clidey/whodb/core/src/providers"
	azureprovider "github.com/clidey/whodb/core/src/providers/azure"
)

var (
	// ErrAzureProviderNotFound indicates the Azure provider doesn't exist.
	ErrAzureProviderNotFound = errors.New("azure provider not found")

	// ErrAzureProviderAlreadyExists indicates an Azure provider with this ID exists.
	ErrAzureProviderAlreadyExists = errors.New("azure provider already exists")
)

// AzureProviderConfig holds the configuration for an Azure provider.
// ClientSecret is intentionally omitted — it is never persisted to disk.
type AzureProviderConfig struct {
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

// AzureProviderState holds the runtime state of an Azure provider.
type AzureProviderState struct {
	Config          *AzureProviderConfig
	Provider        *azureprovider.Provider
	Status          string
	LastDiscoveryAt *time.Time
	DiscoveredCount int
	Error           string
}

var (
	azureProvidersMu sync.RWMutex
	azureProviders   = make(map[string]*AzureProviderState)
	skipAzurePersist bool
)

// GetAzureProviders returns all configured Azure provider states.
func GetAzureProviders() []*AzureProviderState {
	azureProvidersMu.RLock()
	defer azureProvidersMu.RUnlock()

	result := make([]*AzureProviderState, 0, len(azureProviders))
	for _, state := range azureProviders {
		result = append(result, state)
	}
	return result
}

// GetAzureProvider retrieves an Azure provider state by ID.
func GetAzureProvider(id string) (*AzureProviderState, error) {
	azureProvidersMu.RLock()
	defer azureProvidersMu.RUnlock()

	state, exists := azureProviders[id]
	if !exists {
		return nil, ErrAzureProviderNotFound
	}
	return state, nil
}

// AddAzureProvider creates and registers a new Azure provider.
func AddAzureProvider(cfg *AzureProviderConfig, clientSecret string) (*AzureProviderState, error) {
	azureProvidersMu.Lock()

	if _, exists := azureProviders[cfg.ID]; exists {
		azureProvidersMu.Unlock()
		return nil, ErrAzureProviderAlreadyExists
	}

	providerCfg := configToAzureProviderConfig(cfg, clientSecret)
	provider, err := azureprovider.New(providerCfg)
	if err != nil {
		azureProvidersMu.Unlock()
		return nil, err
	}

	registry := providers.GetDefaultRegistry()
	if err := registry.Register(provider); err != nil {
		azureProvidersMu.Unlock()
		return nil, err
	}

	state := &AzureProviderState{
		Config:   cfg,
		Provider: provider,
		Status:   "Connected",
	}

	azureProviders[cfg.ID] = state
	azureProvidersMu.Unlock()

	if !skipAzurePersist {
		if err := saveAzureProvidersToFile(); err != nil {
			log.Warnf("Failed to persist Azure provider after add: %v", err)
		}
	}

	return state, nil
}

// UpdateAzureProvider updates an existing Azure provider with new configuration.
func UpdateAzureProvider(id string, cfg *AzureProviderConfig, clientSecret string) (*AzureProviderState, error) {
	cfg.ID = id
	providerCfg := configToAzureProviderConfig(cfg, clientSecret)
	provider, err := azureprovider.New(providerCfg)
	if err != nil {
		return nil, err
	}

	azureProvidersMu.Lock()
	if _, exists := azureProviders[id]; !exists {
		azureProvidersMu.Unlock()
		return nil, ErrAzureProviderNotFound
	}

	registry := providers.GetDefaultRegistry()
	if err := registry.Unregister(id); err != nil {
		log.Warnf("Failed to unregister Azure provider %s during update: %v", id, err)
	}

	if err := registry.Register(provider); err != nil {
		azureProvidersMu.Unlock()
		return nil, err
	}

	state := &AzureProviderState{
		Config:   cfg,
		Provider: provider,
		Status:   "Connected",
	}

	azureProviders[id] = state
	azureProvidersMu.Unlock()

	if !skipAzurePersist {
		if err := saveAzureProvidersToFile(); err != nil {
			log.Warnf("Failed to persist Azure provider after update: %v", err)
		}
	}

	return state, nil
}

// RemoveAzureProvider removes an Azure provider by ID.
func RemoveAzureProvider(id string) error {
	azureProvidersMu.Lock()

	if _, exists := azureProviders[id]; !exists {
		azureProvidersMu.Unlock()
		return ErrAzureProviderNotFound
	}

	registry := providers.GetDefaultRegistry()
	if err := registry.Unregister(id); err != nil {
		log.Warnf("Failed to unregister Azure provider %s during removal: %v", id, err)
	}

	delete(azureProviders, id)
	azureProvidersMu.Unlock()

	if !skipAzurePersist {
		if err := saveAzureProvidersToFile(); err != nil {
			log.Warnf("Failed to persist Azure provider after remove: %v", err)
		}
	}

	return nil
}

// TestAzureProvider tests the Azure connection for a provider and updates its status.
func TestAzureProvider(id string) (string, error) {
	azureProvidersMu.RLock()
	state, exists := azureProviders[id]
	if !exists {
		azureProvidersMu.RUnlock()
		return "Disconnected", ErrAzureProviderNotFound
	}
	provider := state.Provider
	azureProvidersMu.RUnlock()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	testErr := provider.TestConnection(ctx)

	azureProvidersMu.Lock()
	state, exists = azureProviders[id]
	if !exists {
		azureProvidersMu.Unlock()
		return "Disconnected", ErrAzureProviderNotFound
	}

	if testErr != nil {
		state.Status = "Error"
		state.Error = testErr.Error()
		azureProvidersMu.Unlock()
		return "Error", testErr
	}

	state.Status = "Connected"
	state.Error = ""
	azureProvidersMu.Unlock()
	return "Connected", nil
}

// RefreshAzureProvider triggers a re-discovery of connections for the specified provider.
func RefreshAzureProvider(id string) (*AzureProviderState, error) {
	log.Infof("RefreshAzureProvider called for id=%s", id)

	azureProvidersMu.Lock()
	state, exists := azureProviders[id]
	if !exists {
		azureProvidersMu.Unlock()
		return nil, ErrAzureProviderNotFound
	}
	state.Status = "Discovering"
	azureProvidersMu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	registry := providers.GetDefaultRegistry()
	conns, err := registry.RefreshDiscovery(ctx, id)

	if len(conns) > 0 {
		azureprovider.CheckConnectivity(conns)
	}

	azureProvidersMu.Lock()
	defer azureProvidersMu.Unlock()

	state, exists = azureProviders[id]
	if !exists {
		return nil, ErrAzureProviderNotFound
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

func configToAzureProviderConfig(cfg *AzureProviderConfig, clientSecret string) *azureprovider.Config {
	return &azureprovider.Config{
		ID:                 cfg.ID,
		Name:               cfg.Name,
		SubscriptionID:     cfg.SubscriptionID,
		TenantID:           cfg.TenantID,
		ClientID:           cfg.ClientID,
		ClientSecret:       clientSecret,
		AuthMethod:         azureinfra.AuthMethod(cfg.AuthMethod),
		ResourceGroup:      cfg.ResourceGroup,
		DiscoverPostgreSQL: cfg.DiscoverPostgreSQL,
		DiscoverMySQL:      cfg.DiscoverMySQL,
		DiscoverRedis:      cfg.DiscoverRedis,
		DiscoverCosmosDB:   cfg.DiscoverCosmosDB,
	}
}

// GenerateAzureProviderID creates a unique provider ID from a name and subscription ID.
func GenerateAzureProviderID(name, subscriptionID string) string {
	return "azure-" + sanitizeID(name) + "-" + subscriptionID
}

// InitAzureProvidersFromEnv initializes Azure providers from the WHODB_AZURE_PROVIDER
// environment variable.
func InitAzureProvidersFromEnv() error {
	if !env.IsAzureProviderEnabled {
		return nil
	}

	envConfigs, err := envconfig.GetAzureProvidersFromEnv()
	if err != nil {
		return err
	}

	if len(envConfigs) == 0 {
		return nil
	}

	for i := range envConfigs {
		envCfg := &envConfigs[i]
		name := envCfg.Name
		if name == "" {
			name = fmt.Sprintf("Azure-%d", i+1)
		}
		id := GenerateAzureProviderID(name, envCfg.SubscriptionID)

		authMethod := "default"
		if envCfg.AuthMethod != "" {
			authMethod = envCfg.AuthMethod
		} else if envCfg.ClientID != "" {
			authMethod = "service-principal"
		}

		discoverPostgreSQL := true
		if envCfg.DiscoverPostgreSQL != nil {
			discoverPostgreSQL = *envCfg.DiscoverPostgreSQL
		}
		discoverMySQL := true
		if envCfg.DiscoverMySQL != nil {
			discoverMySQL = *envCfg.DiscoverMySQL
		}
		discoverRedis := true
		if envCfg.DiscoverRedis != nil {
			discoverRedis = *envCfg.DiscoverRedis
		}
		discoverCosmosDB := true
		if envCfg.DiscoverCosmosDB != nil {
			discoverCosmosDB = *envCfg.DiscoverCosmosDB
		}

		cfg := &AzureProviderConfig{
			ID:                 id,
			Name:               name,
			SubscriptionID:     envCfg.SubscriptionID,
			TenantID:           envCfg.TenantID,
			ClientID:           envCfg.ClientID,
			AuthMethod:         authMethod,
			ResourceGroup:      envCfg.ResourceGroup,
			DiscoverPostgreSQL: discoverPostgreSQL,
			DiscoverMySQL:      discoverMySQL,
			DiscoverRedis:      discoverRedis,
			DiscoverCosmosDB:   discoverCosmosDB,
		}

		if _, err := AddAzureProvider(cfg, envCfg.ClientSecret); err != nil {
			log.Warnf("Failed to initialize Azure provider %s: %v", name, err)
		} else {
			log.Infof("Initialized Azure provider: %s (%s)", name, envCfg.SubscriptionID)
		}
	}

	return nil
}
