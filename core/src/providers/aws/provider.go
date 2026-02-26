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

// Package aws implements the AWS connection provider for WhoDB.
//
// This provider discovers database resources from an AWS account (RDS, ElastiCache,
// DocumentDB) and generates credentials that existing WhoDB plugins understand.
// The provider acts as a "credential factory" - it finds AWS databases and produces
// standard engine.Credentials that MySQL, PostgreSQL, Redis, and MongoDB plugins
// can use without modification.
//
// Supported AWS services (CE):
//   - RDS: MySQL, PostgreSQL, MariaDB instances
//   - ElastiCache: Redis clusters
//   - DocumentDB: MongoDB-compatible clusters
//
// Authentication methods:
//   - default: AWS SDK default credential chain (env → shared config → IAM role)
//   - profile: Named AWS profile from ~/.aws/credentials
//
// # Extension Points for EE
//
// The package provides extension points for Enterprise Edition to add support
// for additional database types without modifying CE code:
//   - SetRDSEngineMapper: Add mappings for Oracle, SQL Server RDS engines
//   - SetDiscoveryExtension: Add discovery for DynamoDB
package aws

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/docdb"
	"github.com/aws/aws-sdk-go-v2/service/elasticache"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"golang.org/x/sync/errgroup"

	awsinfra "github.com/clidey/whodb/core/src/aws"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
	"github.com/clidey/whodb/core/src/providers"
)

// DiscoveryExtension discovers additional database connections.
type DiscoveryExtension func(ctx context.Context, p *Provider) ([]providers.DiscoveredConnection, error)

// discoveryExtensions allows EE to register additional discovery functions.
var discoveryExtensions []DiscoveryExtension

// SetDiscoveryExtension registers an additional discovery function.
func SetDiscoveryExtension(ext DiscoveryExtension) {
	discoveryExtensions = append(discoveryExtensions, ext)
}

// GetAWSConfig returns the provider's AWS configuration.
// Used by EE extensions that need direct AWS SDK access.
func (p *Provider) GetAWSConfig() aws.Config {
	return p.awsConfig
}

// Config holds the configuration for an AWS provider instance.
type Config struct {
	// ID uniquely identifies this provider instance.
	ID string

	// Name is a human-readable name (e.g., "Production AWS", "Dev Account").
	Name string

	// Region is the AWS region to discover resources in.
	Region string

	// AuthMethod determines how to authenticate with AWS.
	AuthMethod awsinfra.AuthMethod

	// ProfileName for profile auth.
	ProfileName string

	// DiscoverRDS enables RDS instance discovery.
	DiscoverRDS bool

	// DiscoverElastiCache enables ElastiCache cluster discovery.
	DiscoverElastiCache bool

	// DiscoverDocumentDB enables DocumentDB cluster discovery.
	DiscoverDocumentDB bool
}

// String returns a safe string representation that excludes sensitive credentials.
// This prevents accidental logging of AccessKeyID, SecretAccessKey, and SessionToken.
func (c *Config) String() string {
	return fmt.Sprintf("Config{ID:%s, Name:%s, Region:%s, AuthMethod:%s, ProfileName:%s, DiscoverRDS:%t, DiscoverElastiCache:%t, DiscoverDocumentDB:%t}",
		c.ID, c.Name, c.Region, c.AuthMethod, c.ProfileName, c.DiscoverRDS, c.DiscoverElastiCache, c.DiscoverDocumentDB)
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig(id, name, region string) *Config {
	return &Config{
		ID:                  id,
		Name:                name,
		Region:              region,
		AuthMethod:          awsinfra.AuthMethodDefault,
		DiscoverRDS:         true,
		DiscoverElastiCache: true,
		DiscoverDocumentDB:  true,
	}
}

// Provider implements providers.ConnectionProvider for AWS.
type Provider struct {
	config    *Config
	awsConfig aws.Config

	rdsClient         *rds.Client
	elasticacheClient *elasticache.Client
	docdbClient       *docdb.Client

	initOnce sync.Once
	initErr  error
}

// New creates a new AWS provider with the given configuration.
func New(config *Config) (*Provider, error) {
	if config == nil {
		return nil, errors.New("config is required")
	}
	if config.ID == "" {
		return nil, errors.New("provider ID is required")
	}
	if config.Region == "" {
		return nil, errors.New("region is required")
	}

	return &Provider{
		config: config,
	}, nil
}

// Type implements providers.ConnectionProvider.
func (p *Provider) Type() providers.ProviderType {
	return providers.ProviderTypeAWS
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
	p.initOnce.Do(func() {
		creds := p.buildInternalCredentials()

		cfg, err := awsinfra.LoadAWSConfig(ctx, creds)
		if err != nil {
			p.initErr = fmt.Errorf("failed to load AWS config: %w", err)
			return
		}

		p.awsConfig = cfg

		if p.config.DiscoverRDS {
			p.rdsClient = rds.NewFromConfig(cfg)
		}
		if p.config.DiscoverElastiCache {
			p.elasticacheClient = elasticache.NewFromConfig(cfg)
		}
		if p.config.DiscoverDocumentDB {
			p.docdbClient = docdb.NewFromConfig(cfg)
		}
	})
	return p.initErr
}

func (p *Provider) buildInternalCredentials() *engine.Credentials {
	creds := &engine.Credentials{
		Hostname: p.config.Region,
	}

	var advanced []engine.Record
	advanced = append(advanced, engine.Record{
		Key:   awsinfra.AdvancedKeyAuthMethod,
		Value: string(p.config.AuthMethod),
	})

	if p.config.ProfileName != "" {
		advanced = append(advanced, engine.Record{
			Key:   awsinfra.AdvancedKeyProfileName,
			Value: p.config.ProfileName,
		})
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
// Discovery runs in parallel across all enabled services for faster results.
func (p *Provider) DiscoverConnections(ctx context.Context) ([]providers.DiscoveredConnection, error) {
	log.Infof("AWS Provider DiscoverConnections called for id=%s, region=%s, authMethod=%s, profileName=%s",
		p.config.ID, p.config.Region, p.config.AuthMethod, p.config.ProfileName)

	if err := p.initialize(ctx); err != nil {
		log.Errorf("AWS Provider initialize failed: %v", err)
		return nil, err
	}
	log.Infof("AWS Provider initialized successfully, rdsClient=%v", p.rdsClient != nil)

	// Count how many discovery tasks we'll run
	taskCount := len(discoveryExtensions)
	if p.config.DiscoverRDS && p.rdsClient != nil {
		taskCount++
	}
	if p.config.DiscoverElastiCache && p.elasticacheClient != nil {
		taskCount++
	}
	if p.config.DiscoverDocumentDB && p.docdbClient != nil {
		taskCount++
	}

	if taskCount == 0 {
		return nil, nil
	}

	results := make(chan discoveryResult, taskCount)
	g, gctx := errgroup.WithContext(ctx)

	if p.config.DiscoverRDS && p.rdsClient != nil {
		g.Go(func() error {
			conns, err := p.discoverRDS(gctx)
			results <- discoveryResult{conns, err, "RDS"}
			return nil
		})
	}

	if p.config.DiscoverElastiCache && p.elasticacheClient != nil {
		g.Go(func() error {
			conns, err := p.discoverElastiCache(gctx)
			results <- discoveryResult{conns, err, "ElastiCache"}
			return nil
		})
	}

	if p.config.DiscoverDocumentDB && p.docdbClient != nil {
		g.Go(func() error {
			conns, err := p.discoverDocumentDB(gctx)
			results <- discoveryResult{conns, err, "DocumentDB"}
			return nil
		})
	}

	for _, ext := range discoveryExtensions {
		ext := ext
		g.Go(func() error {
			conns, err := ext(gctx, p)
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
			log.Errorf("AWS Provider: %s discovery failed: %v", r.name, r.err)
			allErrs = append(allErrs, fmt.Errorf("%s: %w", r.name, r.err))
		} else {
			log.Infof("AWS Provider: %s discovery found %d resources", r.name, len(r.conns))
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

	stsClient := sts.NewFromConfig(p.awsConfig)
	validateCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	_, err := stsClient.GetCallerIdentity(validateCtx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return awsinfra.HandleAWSError(err)
	}
	return nil
}

// RefreshConnection implements providers.ConnectionProvider.
// This is intentionally a no-op — the AWS SDK handles credential refresh internally
// through the configured credential chain (env, shared config, IAM role).
func (p *Provider) RefreshConnection(ctx context.Context, connectionID string) (bool, error) {
	return false, nil
}

// Close implements providers.ConnectionProvider.
// AWS SDK clients don't require explicit cleanup.
func (p *Provider) Close(ctx context.Context) error {
	return nil
}

// GetConfig returns the provider's configuration.
func (p *Provider) GetConfig() *Config {
	return p.config
}

// connectionID generates a unique connection ID.
func (p *Provider) connectionID(resourceID string) string {
	return fmt.Sprintf("%s/%s", p.config.ID, resourceID)
}
