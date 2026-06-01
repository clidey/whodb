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

// Package gcp implements the GCP connection provider for WhoDB.
//
// This provider discovers database resources from a GCP project (Cloud SQL, AlloyDB,
// Memorystore) and generates credentials that existing WhoDB plugins understand.
// The provider acts as a "credential factory" — it finds GCP databases and produces
// standard engine.Credentials that MySQL, PostgreSQL, and Redis plugins can use
// without modification.
//
// Supported GCP services (CE):
//   - Cloud SQL: MySQL, PostgreSQL instances
//   - AlloyDB: PostgreSQL-compatible clusters
//   - Memorystore: Redis instances
//
// Authentication methods:
//   - default: Application Default Credentials (GOOGLE_APPLICATION_CREDENTIALS → gcloud → metadata server)
//   - service-account-key: Explicit JSON key file path
//
// # Extension Points
//
// The package provides extension points to add support
// for additional database types without modifying CE code:
//   - SetCloudSQLEngineMapper: Add mappings for additional Cloud SQL engines (e.g., SQL Server)
//   - SetDiscoveryExtension: Add discovery for additional services (e.g., BigQuery, Spanner)
package gcp

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
	"google.golang.org/api/cloudresourcemanager/v3"
	"google.golang.org/api/option"
	sqladmin "google.golang.org/api/sqladmin/v1beta4"

	alloydb "cloud.google.com/go/alloydb/apiv1"
	memcache "cloud.google.com/go/memcache/apiv1"
	redis "cloud.google.com/go/redis/apiv1"

	gcpinfra "github.com/clidey/whodb/core/src/gcp"
	"github.com/clidey/whodb/core/src/log"
	"github.com/clidey/whodb/core/src/providers"
)

// maxPaginationPages is a safety limit to prevent infinite loops if GCP pagination is broken.
const maxPaginationPages = 1000

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

// Config holds the configuration for a GCP provider instance.
type Config struct {
	// ID uniquely identifies this provider instance.
	ID string

	// Name is a human-readable name (e.g., "Production GCP", "Dev Project").
	Name string

	// ProjectID is the GCP project to discover resources in.
	ProjectID string

	// Region is the GCP region to discover resources in.
	Region string

	// AuthMethod determines how to authenticate with GCP.
	AuthMethod gcpinfra.AuthMethod

	// ServiceAccountKeyPath for service account key auth.
	ServiceAccountKeyPath string

	// DiscoverCloudSQL enables Cloud SQL instance discovery.
	DiscoverCloudSQL bool

	// DiscoverAlloyDB enables AlloyDB cluster discovery.
	DiscoverAlloyDB bool

	// DiscoverMemorystore enables Memorystore Redis discovery.
	DiscoverMemorystore bool

	// DiscoveryTimeout overrides the default per-service discovery timeout.
	DiscoveryTimeout time.Duration
}

// String returns a safe string representation that excludes sensitive credentials.
func (c *Config) String() string {
	return fmt.Sprintf("Config{ID:%s, Name:%s, ProjectID:%s, Region:%s, AuthMethod:%s, DiscoverCloudSQL:%t, DiscoverAlloyDB:%t, DiscoverMemorystore:%t}",
		c.ID, c.Name, c.ProjectID, c.Region, c.AuthMethod, c.DiscoverCloudSQL, c.DiscoverAlloyDB, c.DiscoverMemorystore)
}

// Validate checks that the provider configuration has required fields.
func (c *Config) Validate() error {
	if c.ProjectID == "" {
		return errors.New("gcp: project ID is required")
	}
	if c.Region == "" {
		return errors.New("gcp: region is required")
	}
	if !c.DiscoverCloudSQL && !c.DiscoverAlloyDB && !c.DiscoverMemorystore {
		log.Warnf("GCP provider %s: no discovery flags enabled", c.ID)
	}
	return nil
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig(id, name, projectID, region string) *Config {
	return &Config{
		ID:                  id,
		Name:                name,
		ProjectID:           projectID,
		Region:              region,
		AuthMethod:          gcpinfra.AuthMethodDefault,
		DiscoverCloudSQL:    true,
		DiscoverAlloyDB:     true,
		DiscoverMemorystore: true,
	}
}

// Provider implements providers.ConnectionProvider for GCP.
type Provider struct {
	config *Config

	clientOpts []option.ClientOption

	sqladminService   *sqladmin.Service
	alloydbClient     *alloydb.AlloyDBAdminClient
	memorystoreClient *redis.CloudRedisClient
	memcachedClient   *memcache.CloudMemcacheClient

	initMu      sync.Mutex
	initialized bool
}

// New creates a new GCP provider with the given configuration.
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
	return providers.ProviderTypeGCP
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

	var opts []option.ClientOption
	if p.config.AuthMethod == gcpinfra.AuthMethodServiceAccountKey && p.config.ServiceAccountKeyPath != "" {
		opts = append(opts, option.WithCredentialsFile(p.config.ServiceAccountKeyPath))
	}

	p.clientOpts = opts

	if p.config.DiscoverCloudSQL {
		svc, err := sqladmin.NewService(ctx, opts...)
		if err != nil {
			return fmt.Errorf("failed to create Cloud SQL client: %w", gcpinfra.HandleGCPError(err))
		}
		p.sqladminService = svc
	}

	if p.config.DiscoverAlloyDB {
		client, err := alloydb.NewAlloyDBAdminClient(ctx, opts...)
		if err != nil {
			return fmt.Errorf("failed to create AlloyDB client: %w", gcpinfra.HandleGCPError(err))
		}
		p.alloydbClient = client
	}

	if p.config.DiscoverMemorystore {
		client, err := redis.NewCloudRedisClient(ctx, opts...)
		if err != nil {
			return fmt.Errorf("failed to create Memorystore client: %w", gcpinfra.HandleGCPError(err))
		}
		p.memorystoreClient = client

		mcClient, err := memcache.NewCloudMemcacheClient(ctx, opts...)
		if err != nil {
			log.Warnf("GCP Provider: failed to create Memcached client (non-fatal): %v", err)
		} else {
			p.memcachedClient = mcClient
		}
	}

	p.initialized = true
	return nil
}

// discoveryResult holds the outcome of a single service discovery.
type discoveryResult struct {
	conns []providers.DiscoveredConnection
	err   error
	name  string
}

// DiscoverConnections implements providers.ConnectionProvider.
// Discovery runs in parallel across all enabled services for faster results.
func (p *Provider) DiscoverConnections(ctx context.Context) ([]providers.DiscoveredConnection, error) {
	log.Infof("GCP Provider DiscoverConnections called for id=%s, projectID=%s, region=%s, authMethod=%s",
		p.config.ID, p.config.ProjectID, p.config.Region, p.config.AuthMethod)

	if err := p.initialize(ctx); err != nil {
		log.Errorf("GCP Provider initialize failed: %v", err)
		return nil, err
	}
	log.Infof("GCP Provider initialized successfully")

	taskCount := len(discoveryExtensions)
	if p.config.DiscoverCloudSQL && p.sqladminService != nil {
		taskCount++
	}
	if p.config.DiscoverAlloyDB && p.alloydbClient != nil {
		taskCount++
	}
	if p.config.DiscoverMemorystore && p.memorystoreClient != nil {
		taskCount++
	}
	if p.config.DiscoverMemorystore && p.memcachedClient != nil {
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

	if p.config.DiscoverCloudSQL && p.sqladminService != nil {
		g.Go(func() error {
			svcCtx, cancel := context.WithTimeout(gctx, timeout)
			defer cancel()
			conns, err := p.discoverCloudSQL(svcCtx)
			results <- discoveryResult{conns, err, "Cloud SQL"}
			return nil
		})
	}

	if p.config.DiscoverAlloyDB && p.alloydbClient != nil {
		g.Go(func() error {
			svcCtx, cancel := context.WithTimeout(gctx, timeout)
			defer cancel()
			conns, err := p.discoverAlloyDB(svcCtx)
			results <- discoveryResult{conns, err, "AlloyDB"}
			return nil
		})
	}

	if p.config.DiscoverMemorystore && p.memorystoreClient != nil {
		g.Go(func() error {
			svcCtx, cancel := context.WithTimeout(gctx, timeout)
			defer cancel()
			conns, err := p.discoverMemorystore(svcCtx)
			results <- discoveryResult{conns, err, "Memorystore"}
			return nil
		})
	}

	if p.config.DiscoverMemorystore && p.memcachedClient != nil {
		g.Go(func() error {
			svcCtx, cancel := context.WithTimeout(gctx, timeout)
			defer cancel()
			conns, err := p.discoverMemcached(svcCtx)
			results <- discoveryResult{conns, err, "Memcached"}
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
			log.Errorf("GCP Provider: %s discovery failed: %v", r.name, r.err)
			allErrs = append(allErrs, fmt.Errorf("%s: %w", r.name, r.err))
		} else {
			log.Infof("GCP Provider: %s discovery found %d resources", r.name, len(r.conns))
			allConns = append(allConns, r.conns...)
		}
	}

	if len(allErrs) > 0 {
		return allConns, errors.Join(allErrs...)
	}
	return allConns, nil
}

// TestConnection implements providers.ConnectionProvider.
// Uses Cloud Resource Manager to verify project access.
func (p *Provider) TestConnection(ctx context.Context) error {
	if err := p.initialize(ctx); err != nil {
		return err
	}

	crmService, err := cloudresourcemanager.NewService(ctx, p.clientOpts...)
	if err != nil {
		return gcpinfra.HandleGCPError(err)
	}

	validateCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	projectName := "projects/" + p.config.ProjectID
	_, err = crmService.Projects.Get(projectName).Context(validateCtx).Do()
	if err != nil {
		return gcpinfra.HandleGCPError(err)
	}
	return nil
}

// RefreshConnection implements providers.ConnectionProvider.
// This is intentionally a no-op — the GCP SDK handles credential refresh internally.
func (p *Provider) RefreshConnection(ctx context.Context, connectionID string) (bool, error) {
	return false, nil
}

// Close implements providers.ConnectionProvider.
func (p *Provider) Close(ctx context.Context) error {
	p.initMu.Lock()
	defer p.initMu.Unlock()

	if p.alloydbClient != nil {
		_ = p.alloydbClient.Close()
	}
	if p.memorystoreClient != nil {
		_ = p.memorystoreClient.Close()
	}
	if p.memcachedClient != nil {
		_ = p.memcachedClient.Close()
	}
	return nil
}

// GetConfig returns the provider's configuration.
func (p *Provider) GetConfig() *Config {
	return p.config
}

// GetClientOpts returns the provider's client options.
// Used by extensions that need direct GCP SDK access.
func (p *Provider) GetClientOpts() []option.ClientOption {
	return p.clientOpts
}

// connectionID generates a unique connection ID.
func (p *Provider) connectionID(resourceID string) string {
	return fmt.Sprintf("%s/%s", p.config.ID, resourceID)
}
