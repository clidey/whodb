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

	awsinfra "github.com/clidey/whodb/core/src/aws"
	"github.com/clidey/whodb/core/src/env"
	"github.com/clidey/whodb/core/src/log"
	"github.com/clidey/whodb/core/src/providers"
	awsprovider "github.com/clidey/whodb/core/src/providers/aws"
)

var (
	// ErrProviderNotFound indicates the provider doesn't exist.
	ErrProviderNotFound = errors.New("aws provider not found")

	// ErrProviderAlreadyExists indicates a provider with this ID exists.
	ErrProviderAlreadyExists = errors.New("aws provider already exists")
)

// AWSProviderConfig holds the configuration for an AWS provider.
// This struct is used for persistence and management.
// SECURITY: Sensitive fields use json:"-" to prevent accidental serialization.
// The String() method is implemented to prevent accidental logging of credentials.
type AWSProviderConfig struct {
	ID                  string `json:"id"`
	Name                string `json:"name"`
	Region              string `json:"region"`
	AuthMethod          string `json:"authMethod"`
	AccessKeyID         string `json:"-"`
	SecretAccessKey     string `json:"-"`
	SessionToken        string `json:"-"`
	ProfileName         string `json:"profileName,omitempty"`
	DiscoverRDS         bool   `json:"discoverRDS"`
	DiscoverElastiCache bool   `json:"discoverElastiCache"`
	DiscoverDocumentDB  bool   `json:"discoverDocumentDB"`
	DBUsername          string `json:"dbUsername,omitempty"`
}

func (c *AWSProviderConfig) String() string {
	return fmt.Sprintf("AWSProviderConfig{ID:%s, Name:%s, Region:%s, AuthMethod:%s, ProfileName:%s, DiscoverRDS:%t, DiscoverElastiCache:%t, DiscoverDocumentDB:%t}",
		c.ID, c.Name, c.Region, c.AuthMethod, c.ProfileName, c.DiscoverRDS, c.DiscoverElastiCache, c.DiscoverDocumentDB)
}

// AWSProviderState holds the runtime state of an AWS provider including
// its configuration, provider instance, status, and discovery statistics.
type AWSProviderState struct {
	Config          *AWSProviderConfig
	Provider        *awsprovider.Provider
	Status          string
	LastDiscoveryAt *time.Time
	DiscoveredCount int
	Error           string
}

var (
	awsProvidersMu sync.RWMutex
	awsProviders   = make(map[string]*AWSProviderState)
	skipPersist    bool
)

// GetAWSProviders returns all configured AWS provider states.
func GetAWSProviders() []*AWSProviderState {
	awsProvidersMu.RLock()
	defer awsProvidersMu.RUnlock()

	result := make([]*AWSProviderState, 0, len(awsProviders))
	for _, state := range awsProviders {
		result = append(result, state)
	}
	return result
}

// GetAWSProvider retrieves an AWS provider state by ID.
// Returns ErrProviderNotFound if the provider doesn't exist.
func GetAWSProvider(id string) (*AWSProviderState, error) {
	awsProvidersMu.RLock()
	defer awsProvidersMu.RUnlock()

	state, exists := awsProviders[id]
	if !exists {
		return nil, ErrProviderNotFound
	}
	return state, nil
}

// AddAWSProvider creates and registers a new AWS provider with the given configuration.
// Returns ErrProviderAlreadyExists if a provider with the same ID already exists.
func AddAWSProvider(cfg *AWSProviderConfig) (*AWSProviderState, error) {
	awsProvidersMu.Lock()
	defer awsProvidersMu.Unlock()

	if _, exists := awsProviders[cfg.ID]; exists {
		return nil, ErrProviderAlreadyExists
	}

	providerCfg := configToProviderConfig(cfg)
	provider, err := awsprovider.New(providerCfg)
	if err != nil {
		return nil, err
	}

	registry := providers.GetDefaultRegistry()
	if err := registry.Register(provider); err != nil {
		return nil, err
	}

	state := &AWSProviderState{
		Config:   cfg,
		Provider: provider,
		Status:   "Connected",
	}

	awsProviders[cfg.ID] = state

	if !skipPersist {
		go func() {
			if err := saveProvidersToFile(); err != nil {
				log.Logger.Warnf("Failed to persist provider after add: %v", err)
			}
		}()
	}

	return state, nil
}

// UpdateAWSProvider updates an existing AWS provider with new configuration.
// The provider is unregistered and re-created with the new settings.
// Returns ErrProviderNotFound if the provider doesn't exist.
func UpdateAWSProvider(id string, cfg *AWSProviderConfig) (*AWSProviderState, error) {
	awsProvidersMu.Lock()
	defer awsProvidersMu.Unlock()

	oldState, exists := awsProviders[id]
	if !exists {
		return nil, ErrProviderNotFound
	}

	registry := providers.GetDefaultRegistry()
	if err := registry.Unregister(id); err != nil {
		log.Logger.Warnf("Failed to unregister provider %s during update: %v", id, err)
	}

	cfg.ID = id
	providerCfg := configToProviderConfig(cfg)
	provider, err := awsprovider.New(providerCfg)
	if err != nil {
		_ = registry.Register(oldState.Provider)
		return nil, err
	}

	if err := registry.Register(provider); err != nil {
		return nil, err
	}

	state := &AWSProviderState{
		Config:   cfg,
		Provider: provider,
		Status:   "Connected",
	}

	awsProviders[id] = state

	go func() {
		if err := saveProvidersToFile(); err != nil {
			log.Logger.Warnf("Failed to persist provider after update: %v", err)
		}
	}()

	return state, nil
}

// RemoveAWSProvider removes an AWS provider by ID.
// Returns ErrProviderNotFound if the provider doesn't exist.
func RemoveAWSProvider(id string) error {
	awsProvidersMu.Lock()
	defer awsProvidersMu.Unlock()

	if _, exists := awsProviders[id]; !exists {
		return ErrProviderNotFound
	}

	registry := providers.GetDefaultRegistry()
	if err := registry.Unregister(id); err != nil {
		log.Logger.Warnf("Failed to unregister provider %s during removal: %v", id, err)
	}

	delete(awsProviders, id)

	go func() {
		if err := saveProvidersToFile(); err != nil {
			log.Logger.Warnf("Failed to persist provider after remove: %v", err)
		}
	}()

	return nil
}

// TestAWSProvider tests the AWS connection for a provider and updates its status.
// Returns the new status ("Connected" or "Error") and any error encountered.
func TestAWSProvider(id string) (string, error) {
	awsProvidersMu.Lock()
	defer awsProvidersMu.Unlock()

	state, exists := awsProviders[id]
	if !exists {
		return "Disconnected", ErrProviderNotFound
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := state.Provider.TestConnection(ctx); err != nil {
		state.Status = "Error"
		state.Error = err.Error()
		return "Error", err
	}

	state.Status = "Connected"
	state.Error = ""
	return "Connected", nil
}

// RefreshAWSProvider triggers a re-discovery of connections for the specified provider.
// Updates the provider's status, discovered count, and last discovery time.
func RefreshAWSProvider(id string) (*AWSProviderState, error) {
	log.Logger.Infof("RefreshAWSProvider called for id=%s", id)

	awsProvidersMu.Lock()
	state, exists := awsProviders[id]
	if !exists {
		awsProvidersMu.Unlock()
		log.Logger.Warnf("RefreshAWSProvider: provider not found id=%s", id)
		return nil, ErrProviderNotFound
	}
	state.Status = "Discovering"
	log.Logger.Infof("RefreshAWSProvider: provider found, id=%s, region=%s, authMethod=%s",
		state.Config.ID, state.Config.Region, state.Config.AuthMethod)
	awsProvidersMu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	registry := providers.GetDefaultRegistry()
	log.Logger.Infof("RefreshAWSProvider: calling registry.RefreshDiscovery for id=%s", id)
	conns, err := registry.RefreshDiscovery(ctx, id)
	log.Logger.Infof("RefreshAWSProvider: registry.RefreshDiscovery returned %d connections, err=%v", len(conns), err)

	awsProvidersMu.Lock()
	defer awsProvidersMu.Unlock()

	state, exists = awsProviders[id]
	if !exists {
		return nil, ErrProviderNotFound
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

func configToProviderConfig(cfg *AWSProviderConfig) *awsprovider.Config {
	return &awsprovider.Config{
		ID:                  cfg.ID,
		Name:                cfg.Name,
		Region:              cfg.Region,
		AuthMethod:          awsinfra.AuthMethod(cfg.AuthMethod),
		AccessKeyID:         cfg.AccessKeyID,
		SecretAccessKey:     cfg.SecretAccessKey,
		SessionToken:        cfg.SessionToken,
		ProfileName:         cfg.ProfileName,
		DiscoverRDS:         cfg.DiscoverRDS,
		DiscoverElastiCache: cfg.DiscoverElastiCache,
		DiscoverDocumentDB:  cfg.DiscoverDocumentDB,
		DBUsername:          cfg.DBUsername,
	}
}

// GenerateProviderID creates a unique provider ID from a name and region.
// The format is "aws-{sanitized-name}-{region}".
func GenerateProviderID(name, region string) string {
	return "aws-" + sanitizeID(name) + "-" + region
}

// InitAWSProvidersFromEnv initializes AWS providers from the WHODB_AWS_PROVIDER
// environment variable. Each provider configuration is parsed and registered.
func InitAWSProvidersFromEnv() error {
	if !env.IsAWSProviderEnabled {
		return nil
	}

	envConfigs, err := env.GetAWSProvidersFromEnv()
	if err != nil {
		return err
	}

	if len(envConfigs) == 0 {
		return nil
	}

	for i, envCfg := range envConfigs {
		name := envCfg.Name
		if name == "" {
			name = fmt.Sprintf("AWS-%d", i+1)
		}
		id := GenerateProviderID(name, envCfg.Region)

		authMethod := envCfg.Auth
		if authMethod == "" {
			authMethod = "default"
		}

		discoverRDS := true
		if envCfg.DiscoverRDS != nil {
			discoverRDS = *envCfg.DiscoverRDS
		}

		discoverElastiCache := true
		if envCfg.DiscoverElastiCache != nil {
			discoverElastiCache = *envCfg.DiscoverElastiCache
		}

		discoverDocumentDB := true
		if envCfg.DiscoverDocumentDB != nil {
			discoverDocumentDB = *envCfg.DiscoverDocumentDB
		}

		cfg := &AWSProviderConfig{
			ID:                  id,
			Name:                name,
			Region:              envCfg.Region,
			AuthMethod:          authMethod,
			AccessKeyID:         envCfg.AccessKeyID,
			SecretAccessKey:     envCfg.SecretAccessKey,
			SessionToken:        envCfg.SessionToken,
			ProfileName:         envCfg.ProfileName,
			DiscoverRDS:         discoverRDS,
			DiscoverElastiCache: discoverElastiCache,
			DiscoverDocumentDB:  discoverDocumentDB,
			DBUsername:          envCfg.DBUsername,
		}

		if _, err := AddAWSProvider(cfg); err != nil {
			log.Logger.Warnf("Failed to initialize AWS provider %s: %v", name, err)
		} else {
			log.Logger.Infof("Initialized AWS provider: %s (%s)", name, envCfg.Region)
		}
	}

	return nil
}

func sanitizeID(s string) string {
	result := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_' {
			result = append(result, c)
		}
	}
	return string(result)
}
