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

package llm

import (
	"errors"
	"sync"

	"github.com/clidey/whodb/core/src/env"
	"github.com/google/uuid"
)

var (
	providerRegistry *ProviderRegistry
	registryOnce     sync.Once
)

// ProviderRegistry manages all AI provider configurations
type ProviderRegistry struct {
	providers map[string]*ProviderConfig
	mu        sync.RWMutex
}

// GetProviderRegistry returns the singleton provider registry
func GetProviderRegistry() *ProviderRegistry {
	registryOnce.Do(func() {
		providerRegistry = &ProviderRegistry{
			providers: make(map[string]*ProviderConfig),
		}
		providerRegistry.initializeFromEnvironment()
	})
	return providerRegistry
}

// initializeFromEnvironment loads providers from environment variables
func (r *ProviderRegistry) initializeFromEnvironment() {
	envProviders := env.GetConfiguredChatProviders()

	for _, provider := range envProviders {
        config := &ProviderConfig{
            ID:                   provider.ProviderId,
            Name:                 string(provider.Type) + " (Environment)",
            Type:                 LLMType(provider.Type),
            APIKey:               provider.APIKey,
            BaseURL:              provider.Endpoint,
            IsEnvironmentDefined: true,
            IsUserDefined:        false,
        }

		// Apply defaults
		config.ApplyDefaults()

        // BaseURL already set from environment provider definition

		r.providers[config.ID] = config
	}
}

// GetProvider returns a provider by ID
func (r *ProviderRegistry) GetProvider(id string) (*ProviderConfig, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	provider, exists := r.providers[id]
	if !exists {
		return nil, errors.New("provider not found: " + id)
	}

	return provider.Clone(), nil
}

// GetAllProviders returns all registered providers
func (r *ProviderRegistry) GetAllProviders() []*ProviderConfig {
	r.mu.RLock()
	defer r.mu.RUnlock()

	providers := make([]*ProviderConfig, 0, len(r.providers))
	for _, provider := range r.providers {
		providers = append(providers, provider.Clone())
	}

	return providers
}

// CreateProvider creates a new provider configuration
func (r *ProviderRegistry) CreateProvider(config *ProviderConfig) (*ProviderConfig, error) {
	if config.ID == "" {
		config.ID = uuid.New().String()
	}

	config.IsUserDefined = true
	config.ApplyDefaults()

	if err := config.Validate(); err != nil {
		return nil, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.providers[config.ID]; exists {
		return nil, errors.New("provider with ID already exists: " + config.ID)
	}

	r.providers[config.ID] = config.Clone()

    // Note: persistence of providers is not implemented yet

	return config.Clone(), nil
}

// UpdateProvider updates an existing provider configuration
func (r *ProviderRegistry) UpdateProvider(id string, config *ProviderConfig) (*ProviderConfig, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, exists := r.providers[id]
	if !exists {
		return nil, errors.New("provider not found: " + id)
	}

	// Don't allow updating environment-defined providers
	if existing.IsEnvironmentDefined && !existing.IsUserDefined {
		return nil, errors.New("cannot update environment-defined provider")
	}

	// Preserve the ID
	config.ID = id
	config.IsUserDefined = true
	config.ApplyDefaults()

	if err := config.Validate(); err != nil {
		return nil, err
	}

	r.providers[id] = config.Clone()

    // Note: persistence of providers is not implemented yet

	return config.Clone(), nil
}

// DeleteProvider removes a provider configuration
func (r *ProviderRegistry) DeleteProvider(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, exists := r.providers[id]
	if !exists {
		return errors.New("provider not found: " + id)
	}

	// Don't allow deleting environment-defined providers
	if existing.IsEnvironmentDefined && !existing.IsUserDefined {
		return errors.New("cannot delete environment-defined provider")
	}

	delete(r.providers, id)

    // Note: persistence of providers is not implemented yet

	return nil
}

// GetProvidersByType returns all providers of a specific type
func (r *ProviderRegistry) GetProvidersByType(llmType LLMType) []*ProviderConfig {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var providers []*ProviderConfig
	for _, provider := range r.providers {
		if provider.Type == llmType {
			providers = append(providers, provider.Clone())
		}
	}

	return providers
}

// Placeholder persistence helpers were removed until backend storage is added.
