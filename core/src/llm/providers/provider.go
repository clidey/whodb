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

package providers

import (
	"errors"
	"fmt"
)

// LLMType represents the type of LLM provider.
type LLMType string

// LLMModel represents a specific model within a provider.
type LLMModel string

// AIProvider defines the interface that all LLM providers must implement.
// This follows WhoDB's plugin architecture pattern (similar to database plugins).
type AIProvider interface {
	// Metadata methods
	GetType() LLMType
	GetName() string
	RequiresAPIKey() bool

	// Configuration methods
	GetDefaultEndpoint() string
	ValidateConfig(config *ProviderConfig) error

	// Core operations
	GetSupportedModels(config *ProviderConfig) ([]string, error)
	Complete(config *ProviderConfig, prompt string, model LLMModel, receiverChan *chan string) (*string, error)

	// BAML integration methods
	GetBAMLClientType() string
	CreateBAMLClientOptions(config *ProviderConfig, model string) (map[string]any, error)
}

// ProviderConfig holds configuration for a specific provider instance.
type ProviderConfig struct {
	Type       LLMType
	APIKey     string
	Endpoint   string
	ProviderId string // e.g., "openai-1", "generic-mistral"
	Metadata   map[string]string
}

// ProviderRegistry manages all registered AI providers.
// Uses a map for O(1) lookup by provider type.
type ProviderRegistry struct {
	providers map[LLMType]AIProvider
}

// NewProviderRegistry creates a new provider registry.
func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{
		providers: make(map[LLMType]AIProvider),
	}
}

// Register adds a provider to the registry.
// If a provider with the same type already exists, it will be replaced.
func (r *ProviderRegistry) Register(provider AIProvider) {
	r.providers[provider.GetType()] = provider
}

// Get retrieves a provider by type.
// Returns an error if the provider type is not registered.
func (r *ProviderRegistry) Get(providerType LLMType) (AIProvider, error) {
	if provider, ok := r.providers[providerType]; ok {
		return provider, nil
	}
	return nil, fmt.Errorf("unsupported provider type: %s", providerType)
}

// Has checks if a provider type is registered.
func (r *ProviderRegistry) Has(providerType LLMType) bool {
	_, ok := r.providers[providerType]
	return ok
}

// List returns all registered provider types.
func (r *ProviderRegistry) List() []LLMType {
	types := make([]LLMType, 0, len(r.providers))
	for providerType := range r.providers {
		types = append(types, providerType)
	}
	return types
}

// Module-level registry instance
var providerRegistry *ProviderRegistry

// init initializes the provider registry and registers built-in providers.
// This is called automatically when the package is imported.
func init() {
	providerRegistry = NewProviderRegistry()
}

// GetProvider retrieves a provider from the global registry.
// Returns an error if the provider type is not registered.
func GetProvider(providerType LLMType) (AIProvider, error) {
	if providerRegistry == nil {
		return nil, errors.New("provider registry not initialized")
	}
	return providerRegistry.Get(providerType)
}

// RegisterProvider registers a provider in the global registry.
func RegisterProvider(provider AIProvider) {
	if providerRegistry == nil {
		providerRegistry = NewProviderRegistry()
	}
	providerRegistry.Register(provider)
}

// HasProvider checks if a provider type is registered in the global registry.
func HasProvider(providerType LLMType) bool {
	if providerRegistry == nil {
		return false
	}
	return providerRegistry.Has(providerType)
}

// ListProviders returns all registered provider types from the global registry.
func ListProviders() []LLMType {
	if providerRegistry == nil {
		return []LLMType{}
	}
	return providerRegistry.List()
}
