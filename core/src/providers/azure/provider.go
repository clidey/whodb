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

// Package azure implements the Azure connection provider for WhoDB.
//
// This provider discovers database resources from an Azure subscription
// (PostgreSQL, MySQL, Redis, Cosmos DB for MongoDB) and generates credentials
// that existing WhoDB plugins understand.
//
// Supported Azure services (CE):
//   - Azure Database for PostgreSQL (Flexible Server)
//   - Azure Database for MySQL (Flexible Server)
//   - Azure Cache for Redis
//   - Azure Cosmos DB for MongoDB
//
// Authentication methods:
//   - default: Azure SDK default credential chain (env → managed identity → Azure CLI)
//   - service-principal: Explicit Tenant ID, Client ID, Client Secret
//
// # Extension Points
//
// The package provides extension points to add support
// for additional database types without modifying CE code:
//   - SetAzureSQLEngineMapper: Add mappings for additional Azure SQL engines
//   - SetDiscoveryExtension: Add discovery for additional services
package azure

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/cosmos/armcosmos/v3"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/mysql/armmysqlflexibleservers"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/postgresql/armpostgresqlflexibleservers"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/redis/armredis"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armsubscriptions"
	"golang.org/x/sync/errgroup"

	azureinfra "github.com/clidey/whodb/core/src/azure"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
	"github.com/clidey/whodb/core/src/providers"
)

// discoveryTimeout is the per-service timeout for individual discovery operations.
const discoveryTimeout = 60 * time.Second

// DiscoveryExtension discovers additional database connections.
type DiscoveryExtension func(ctx context.Context, p *Provider) ([]providers.DiscoveredConnection, error)

// discoveryExtensions allows extensions to register additional discovery functions.
var discoveryExtensions []DiscoveryExtension

// SetDiscoveryExtension registers an additional discovery function.
func SetDiscoveryExtension(ext DiscoveryExtension) {
	discoveryExtensions = append(discoveryExtensions, ext)
}

// AzureSQLEngineMapper maps an Azure SQL engine type to a WhoDB DatabaseType.
// Returns false if the engine is not recognized by this mapper.
var azureSQLEngineMapper func(string) (engine.DatabaseType, bool) //nolint:unused

// SetAzureSQLEngineMapper registers a mapper for additional Azure SQL engine types.
func SetAzureSQLEngineMapper(fn func(string) (engine.DatabaseType, bool)) {
	azureSQLEngineMapper = fn
}

// Config holds the configuration for an Azure provider instance.
type Config struct {
	// ID uniquely identifies this provider instance.
	ID string

	// Name is a human-readable name (e.g., "Production Azure", "Dev Subscription").
	Name string

	// SubscriptionID is the Azure subscription to discover resources in.
	SubscriptionID string

	// TenantID for service principal auth.
	TenantID string

	// ClientID for service principal auth.
	ClientID string

	// ClientSecret for service principal auth. Never logged.
	ClientSecret string

	// AuthMethod determines how to authenticate with Azure.
	AuthMethod azureinfra.AuthMethod

	// ResourceGroup optionally limits discovery to a single resource group.
	ResourceGroup string

	// DiscoverPostgreSQL enables Azure Database for PostgreSQL discovery.
	DiscoverPostgreSQL bool

	// DiscoverMySQL enables Azure Database for MySQL discovery.
	DiscoverMySQL bool

	// DiscoverRedis enables Azure Cache for Redis discovery.
	DiscoverRedis bool

	// DiscoverCosmosDB enables Azure Cosmos DB for MongoDB discovery.
	DiscoverCosmosDB bool

	// DiscoveryTimeout overrides the default per-service discovery timeout.
	DiscoveryTimeout time.Duration
}

// String returns a safe string representation that excludes sensitive credentials.
func (c *Config) String() string {
	return fmt.Sprintf("Config{ID:%s, Name:%s, SubscriptionID:%s, AuthMethod:%s, ResourceGroup:%s, DiscoverPostgreSQL:%t, DiscoverMySQL:%t, DiscoverRedis:%t, DiscoverCosmosDB:%t}",
		c.ID, c.Name, c.SubscriptionID, c.AuthMethod, c.ResourceGroup, c.DiscoverPostgreSQL, c.DiscoverMySQL, c.DiscoverRedis, c.DiscoverCosmosDB)
}

// Validate checks that the provider configuration has required fields.
func (c *Config) Validate() error {
	if c.SubscriptionID == "" {
		return errors.New("azure: subscription ID is required")
	}
	if !c.DiscoverPostgreSQL && !c.DiscoverMySQL && !c.DiscoverRedis && !c.DiscoverCosmosDB {
		log.Warnf("Azure provider %s: no discovery flags enabled", c.ID)
	}
	return nil
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig(id, name, subscriptionID string) *Config {
	return &Config{
		ID:                 id,
		Name:               name,
		SubscriptionID:     subscriptionID,
		AuthMethod:         azureinfra.AuthMethodDefault,
		DiscoverPostgreSQL: true,
		DiscoverMySQL:      true,
		DiscoverRedis:      true,
		DiscoverCosmosDB:   true,
	}
}

// Provider implements providers.ConnectionProvider for Azure.
type Provider struct {
	config     *Config
	credential azcore.TokenCredential

	postgresClient *armpostgresqlflexibleservers.ServersClient
	mysqlClient    *armmysqlflexibleservers.ServersClient
	redisClient    *armredis.Client
	cosmosClient   *armcosmos.DatabaseAccountsClient

	initMu      sync.Mutex
	initialized bool
}

// New creates a new Azure provider with the given configuration.
func New(config *Config) (*Provider, error) {
	if config == nil {
		return nil, errors.New("config is required")
	}
	if config.ID == "" {
		return nil, errors.New("provider ID is required")
	}
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &Provider{
		config: config,
	}, nil
}

// Type implements providers.ConnectionProvider.
func (p *Provider) Type() providers.ProviderType {
	return providers.ProviderTypeAzure
}

// ID implements providers.ConnectionProvider.
func (p *Provider) ID() string {
	return p.config.ID
}

// Name implements providers.ConnectionProvider.
func (p *Provider) Name() string {
	return p.config.Name
}

func (p *Provider) initialize(ctx context.Context) error {
	p.initMu.Lock()
	defer p.initMu.Unlock()

	if p.initialized {
		return nil
	}

	creds := p.buildInternalCredentials()
	credential, _, err := azureinfra.LoadAzureCredential(creds)
	if err != nil {
		return fmt.Errorf("failed to load Azure credential: %w", err)
	}
	p.credential = credential

	if p.config.DiscoverPostgreSQL {
		client, err := armpostgresqlflexibleservers.NewServersClient(p.config.SubscriptionID, credential, nil)
		if err != nil {
			return fmt.Errorf("failed to create PostgreSQL client: %w", azureinfra.HandleAzureError(err))
		}
		p.postgresClient = client
	}

	if p.config.DiscoverMySQL {
		client, err := armmysqlflexibleservers.NewServersClient(p.config.SubscriptionID, credential, nil)
		if err != nil {
			return fmt.Errorf("failed to create MySQL client: %w", azureinfra.HandleAzureError(err))
		}
		p.mysqlClient = client
	}

	if p.config.DiscoverRedis {
		client, err := armredis.NewClient(p.config.SubscriptionID, credential, nil)
		if err != nil {
			return fmt.Errorf("failed to create Redis client: %w", azureinfra.HandleAzureError(err))
		}
		p.redisClient = client
	}

	if p.config.DiscoverCosmosDB {
		client, err := armcosmos.NewDatabaseAccountsClient(p.config.SubscriptionID, credential, nil)
		if err != nil {
			return fmt.Errorf("failed to create Cosmos DB client: %w", azureinfra.HandleAzureError(err))
		}
		p.cosmosClient = client
	}

	p.initialized = true
	return nil
}

func (p *Provider) buildInternalCredentials() *engine.Credentials {
	creds := &engine.Credentials{
		Hostname: p.config.SubscriptionID,
	}

	var advanced []engine.Record
	advanced = append(advanced, engine.Record{
		Key:   azureinfra.AdvancedKeyAuthMethod,
		Value: string(p.config.AuthMethod),
	})

	if p.config.TenantID != "" {
		advanced = append(advanced, engine.Record{Key: azureinfra.AdvancedKeyTenantID, Value: p.config.TenantID})
	}
	if p.config.ClientID != "" {
		advanced = append(advanced, engine.Record{Key: azureinfra.AdvancedKeyClientID, Value: p.config.ClientID})
	}
	if p.config.ClientSecret != "" {
		advanced = append(advanced, engine.Record{Key: azureinfra.AdvancedKeyClientSecret, Value: p.config.ClientSecret})
	}
	if p.config.ResourceGroup != "" {
		advanced = append(advanced, engine.Record{Key: azureinfra.AdvancedKeyResourceGroup, Value: p.config.ResourceGroup})
	}

	creds.Advanced = advanced
	return creds
}

// discoveryResult holds the outcome of a single service discovery.
type discoveryResult struct {
	conns []providers.DiscoveredConnection
	err   error
	name  string
}

// DiscoverConnections implements providers.ConnectionProvider.
// Discovery runs in parallel across all enabled services.
func (p *Provider) DiscoverConnections(ctx context.Context) ([]providers.DiscoveredConnection, error) {
	log.Infof("Azure Provider DiscoverConnections called for id=%s, subscription=%s, authMethod=%s",
		p.config.ID, p.config.SubscriptionID, p.config.AuthMethod)

	if err := p.initialize(ctx); err != nil {
		log.Errorf("Azure Provider initialize failed: %v", err)
		return nil, err
	}

	taskCount := len(discoveryExtensions)
	if p.config.DiscoverPostgreSQL && p.postgresClient != nil {
		taskCount++
	}
	if p.config.DiscoverMySQL && p.mysqlClient != nil {
		taskCount++
	}
	if p.config.DiscoverRedis && p.redisClient != nil {
		taskCount++
	}
	if p.config.DiscoverCosmosDB && p.cosmosClient != nil {
		taskCount++
	}

	if taskCount == 0 {
		return nil, nil
	}

	timeout := discoveryTimeout
	if p.config.DiscoveryTimeout > 0 {
		timeout = p.config.DiscoveryTimeout
	}

	results := make(chan discoveryResult, taskCount)
	g, gctx := errgroup.WithContext(ctx)

	if p.config.DiscoverPostgreSQL && p.postgresClient != nil {
		g.Go(func() error {
			svcCtx, cancel := context.WithTimeout(gctx, timeout)
			defer cancel()
			conns, err := p.discoverPostgreSQL(svcCtx)
			results <- discoveryResult{conns, err, "PostgreSQL"}
			return nil
		})
	}

	if p.config.DiscoverMySQL && p.mysqlClient != nil {
		g.Go(func() error {
			svcCtx, cancel := context.WithTimeout(gctx, timeout)
			defer cancel()
			conns, err := p.discoverMySQL(svcCtx)
			results <- discoveryResult{conns, err, "MySQL"}
			return nil
		})
	}

	if p.config.DiscoverRedis && p.redisClient != nil {
		g.Go(func() error {
			svcCtx, cancel := context.WithTimeout(gctx, timeout)
			defer cancel()
			conns, err := p.discoverRedis(svcCtx)
			results <- discoveryResult{conns, err, "Redis"}
			return nil
		})
	}

	if p.config.DiscoverCosmosDB && p.cosmosClient != nil {
		g.Go(func() error {
			svcCtx, cancel := context.WithTimeout(gctx, timeout)
			defer cancel()
			conns, err := p.discoverCosmosDB(svcCtx)
			results <- discoveryResult{conns, err, "CosmosDB"}
			return nil
		})
	}

	for _, ext := range discoveryExtensions {
		g.Go(func() error {
			svcCtx, cancel := context.WithTimeout(gctx, timeout)
			defer cancel()
			conns, err := ext(svcCtx, p)
			results <- discoveryResult{conns, err, "extension"}
			return nil
		})
	}

	_ = g.Wait()
	close(results)

	var allConns []providers.DiscoveredConnection
	var allErrs []error
	for r := range results {
		if r.err != nil {
			log.Errorf("Azure Provider: %s discovery failed: %v", r.name, r.err)
			allErrs = append(allErrs, fmt.Errorf("%s: %w", r.name, r.err))
		} else {
			log.Infof("Azure Provider: %s discovery found %d resources", r.name, len(r.conns))
			allConns = append(allConns, r.conns...)
		}
	}

	if len(allErrs) > 0 {
		return allConns, errors.Join(allErrs...)
	}
	return allConns, nil
}

// TestConnection implements providers.ConnectionProvider.
func (p *Provider) TestConnection(ctx context.Context) error {
	if err := p.initialize(ctx); err != nil {
		return err
	}

	// Verify credential by listing subscriptions and checking ours is accessible
	subsClient, err := armsubscriptions.NewClient(p.credential, nil)
	if err != nil {
		return azureinfra.HandleAzureError(err)
	}

	validateCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	_, err = subsClient.Get(validateCtx, p.config.SubscriptionID, nil)
	if err != nil {
		return azureinfra.HandleAzureError(err)
	}
	return nil
}

// RefreshConnection implements providers.ConnectionProvider.
// Azure SDK handles credential refresh internally.
func (p *Provider) RefreshConnection(ctx context.Context, connectionID string) (bool, error) {
	return false, nil
}

// Close implements providers.ConnectionProvider.
// Azure SDK clients don't require explicit cleanup.
func (p *Provider) Close(ctx context.Context) error {
	return nil
}

// GetConfig returns the provider's configuration.
func (p *Provider) GetConfig() *Config {
	return p.config
}

// GetAzureCredential returns the provider's Azure credential.
// Used by extensions that need direct Azure SDK access.
func (p *Provider) GetAzureCredential() azcore.TokenCredential {
	return p.credential
}

// GetSubscriptionID returns the provider's subscription ID.
// Used by extensions that need to create additional ARM clients.
func (p *Provider) GetSubscriptionID() string {
	return p.config.SubscriptionID
}

// GetResourceGroup returns the optional resource group filter.
func (p *Provider) GetResourceGroup() string {
	return p.config.ResourceGroup
}

// connectionID generates a unique connection ID.
func (p *Provider) connectionID(resourceID string) string {
	return fmt.Sprintf("%s/%s", p.config.ID, resourceID)
}

// extractResourceGroup extracts the resource group name from an Azure resource ID.
// Azure resource IDs follow the pattern: /subscriptions/{sub}/resourceGroups/{rg}/providers/...
func extractResourceGroup(resourceID string) string {
	if resourceID == "" {
		return ""
	}
	const prefix = "/resourcegroups/"
	lower := toLower(resourceID)
	idx := indexOf(lower, prefix)
	if idx == -1 {
		return ""
	}
	rest := resourceID[idx+len(prefix):]
	end := indexOf(rest, "/")
	if end == -1 {
		return rest
	}
	return rest[:end]
}

func toLower(s string) string {
	b := make([]byte, len(s))
	for i := range s {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			b[i] = c + 32
		} else {
			b[i] = c
		}
	}
	return string(b)
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
