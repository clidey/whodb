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
	"context"
	"errors"
	"sync"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
)

var (
	// ErrProviderNotFound indicates the requested provider doesn't exist.
	ErrProviderNotFound = errors.New("provider not found")

	// ErrProviderAlreadyExists indicates a provider with this ID already exists.
	ErrProviderAlreadyExists = errors.New("provider already exists")

	// ErrConnectionNotFound indicates the requested connection doesn't exist.
	ErrConnectionNotFound = errors.New("connection not found")
)

// Registry manages connection providers and aggregates their discovered connections.
// It is safe for concurrent use.
type Registry struct {
	mu        sync.RWMutex
	providers map[string]ConnectionProvider

	// discoveryCache caches discovered connections by provider ID.
	// This avoids repeated expensive API calls.
	cacheMu   sync.RWMutex
	connCache map[string][]DiscoveredConnection
}

// NewRegistry creates a new provider registry.
func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]ConnectionProvider),
		connCache: make(map[string][]DiscoveredConnection),
	}
}

// Register adds a provider to the registry.
// Returns ErrProviderAlreadyExists if a provider with this ID already exists.
func (r *Registry) Register(provider ConnectionProvider) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	id := provider.ID()
	if _, exists := r.providers[id]; exists {
		return ErrProviderAlreadyExists
	}

	r.providers[id] = provider
	return nil
}

// Unregister removes a provider from the registry.
// Also clears any cached connections for this provider.
func (r *Registry) Unregister(providerID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	provider, exists := r.providers[providerID]
	if !exists {
		return ErrProviderNotFound
	}

	if err := provider.Close(context.Background()); err != nil {
		log.Logger.Warnf("Provider %s close failed during unregister: %v", providerID, err)
	}

	delete(r.providers, providerID)

	r.cacheMu.Lock()
	delete(r.connCache, providerID)
	r.cacheMu.Unlock()

	return nil
}

// Get returns a provider by ID.
func (r *Registry) Get(providerID string) (ConnectionProvider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	provider, exists := r.providers[providerID]
	if !exists {
		return nil, ErrProviderNotFound
	}
	return provider, nil
}

// GetByType returns all providers of a given type.
func (r *Registry) GetByType(providerType ProviderType) []ConnectionProvider {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []ConnectionProvider
	for _, p := range r.providers {
		if p.Type() == providerType {
			result = append(result, p)
		}
	}
	return result
}

// List returns all registered providers.
func (r *Registry) List() []ConnectionProvider {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]ConnectionProvider, 0, len(r.providers))
	for _, p := range r.providers {
		result = append(result, p)
	}
	return result
}

// DiscoverAll discovers connections from all registered providers.
// Results are cached. Use RefreshDiscovery to force a refresh.
func (r *Registry) DiscoverAll(ctx context.Context) ([]DiscoveredConnection, error) {
	r.mu.RLock()
	providers := make([]ConnectionProvider, 0, len(r.providers))
	for _, p := range r.providers {
		providers = append(providers, p)
	}
	r.mu.RUnlock()

	log.Logger.Infof("DiscoverAll: found %d registered providers", len(providers))

	var allConns []DiscoveredConnection
	var errs []error

	for _, p := range providers {
		providerID := p.ID()

		r.cacheMu.RLock()
		cached, hasCached := r.connCache[providerID]
		r.cacheMu.RUnlock()

		if hasCached {
			log.Logger.Infof("DiscoverAll: using %d cached connections for provider %s", len(cached), providerID)
			allConns = append(allConns, cached...)
			continue
		}

		log.Logger.Infof("DiscoverAll: no cache for provider %s, calling DiscoverConnections", providerID)

		conns, err := p.DiscoverConnections(ctx)

		if len(conns) > 0 {
			log.Logger.Infof("DiscoverAll: got %d connections from provider %s", len(conns), providerID)
			r.cacheMu.Lock()
			r.connCache[providerID] = conns
			r.cacheMu.Unlock()
			allConns = append(allConns, conns...)
		}

		if err != nil {
			log.Logger.Warnf("DiscoverAll: error from provider %s: %v", providerID, err)
			errs = append(errs, err)
		}
	}

	log.Logger.Infof("DiscoverAll: returning %d total connections", len(allConns))

	if len(errs) > 0 {
		return allConns, errors.Join(errs...)
	}
	return allConns, nil
}

// RefreshDiscovery clears the cache for a provider and re-discovers connections.
// If providerID is empty, refreshes all providers.
func (r *Registry) RefreshDiscovery(ctx context.Context, providerID string) ([]DiscoveredConnection, error) {
	if providerID == "" {
		r.cacheMu.Lock()
		r.connCache = make(map[string][]DiscoveredConnection)
		r.cacheMu.Unlock()
		return r.DiscoverAll(ctx)
	}

	r.cacheMu.Lock()
	delete(r.connCache, providerID)
	r.cacheMu.Unlock()

	provider, err := r.Get(providerID)
	if err != nil {
		return nil, err
	}

	conns, err := provider.DiscoverConnections(ctx)
	if err != nil {
		if len(conns) > 0 {
			r.cacheMu.Lock()
			r.connCache[providerID] = conns
			r.cacheMu.Unlock()
		}
		return conns, err
	}

	r.cacheMu.Lock()
	r.connCache[providerID] = conns
	r.cacheMu.Unlock()

	return conns, nil
}

// FilterByDatabaseType returns connections that match the given database type.
func (r *Registry) FilterByDatabaseType(ctx context.Context, dbType engine.DatabaseType) ([]DiscoveredConnection, error) {
	all, err := r.DiscoverAll(ctx)
	if err != nil {
		return nil, err
	}

	var result []DiscoveredConnection
	for _, c := range all {
		if c.DatabaseType == dbType {
			result = append(result, c)
		}
	}
	return result, nil
}

// FilterByProvider returns connections from a specific provider.
func (r *Registry) FilterByProvider(ctx context.Context, providerID string) ([]DiscoveredConnection, error) {
	all, err := r.DiscoverAll(ctx)
	if err != nil {
		return nil, err
	}

	var result []DiscoveredConnection
	for _, c := range all {
		if c.ProviderID == providerID {
			result = append(result, c)
		}
	}
	return result, nil
}

// FilterAvailable returns only connections that are currently available.
func (r *Registry) FilterAvailable(ctx context.Context) ([]DiscoveredConnection, error) {
	all, err := r.DiscoverAll(ctx)
	if err != nil {
		return nil, err
	}

	var result []DiscoveredConnection
	for _, c := range all {
		if c.Status.IsAvailable() {
			result = append(result, c)
		}
	}
	return result, nil
}

// Close closes all registered providers and clears the registry.
func (r *Registry) Close(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var errs []error
	for _, p := range r.providers {
		if err := p.Close(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	r.providers = make(map[string]ConnectionProvider)

	r.cacheMu.Lock()
	r.connCache = make(map[string][]DiscoveredConnection)
	r.cacheMu.Unlock()

	return errors.Join(errs...)
}

// DefaultRegistry is the global provider registry.
// It is initialized lazily on first access.
var (
	defaultRegistry     *Registry
	defaultRegistryOnce sync.Once
)

// GetDefaultRegistry returns the global provider registry.
func GetDefaultRegistry() *Registry {
	defaultRegistryOnce.Do(func() {
		defaultRegistry = NewRegistry()
	})
	return defaultRegistry
}
