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

package graph

import (
	"context"
	"fmt"
	"time"

	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/src/aws"
	"github.com/clidey/whodb/core/src/env"
	"github.com/clidey/whodb/core/src/log"
	"github.com/clidey/whodb/core/src/providers"
	awsprovider "github.com/clidey/whodb/core/src/providers/aws"
	"github.com/clidey/whodb/core/src/settings"
)

// AddAWSProvider is the resolver for the AddAWSProvider field.
func (r *mutationResolver) AddAWSProvider(ctx context.Context, input model.AWSProviderInput) (*model.AWSProvider, error) {
	if !env.IsAWSProviderEnabled {
		return nil, aws.ErrAWSProviderDisabled
	}

	id := settings.GenerateProviderID(input.Name, input.Region)

	// Derive auth method: if a profile name is provided, use "profile"; otherwise "default"
	authMethod := "default"
	profileName := ""
	if input.ProfileName != nil && *input.ProfileName != "" {
		authMethod = "profile"
		profileName = *input.ProfileName
	}

	discoverRDS := true
	if input.DiscoverRds != nil {
		discoverRDS = *input.DiscoverRds
	}

	discoverElastiCache := true
	if input.DiscoverElastiCache != nil {
		discoverElastiCache = *input.DiscoverElastiCache
	}

	discoverDocumentDB := true
	if input.DiscoverDocumentDb != nil {
		discoverDocumentDB = *input.DiscoverDocumentDb
	}

	cfg := &settings.AWSProviderConfig{
		ID:                  id,
		Name:                input.Name,
		Region:              input.Region,
		AuthMethod:          authMethod,
		ProfileName:         profileName,
		DiscoverRDS:         discoverRDS,
		DiscoverElastiCache: discoverElastiCache,
		DiscoverDocumentDB:  discoverDocumentDB,
	}

	state, err := settings.AddAWSProvider(cfg)
	if err != nil {
		return nil, err
	}

	return stateToAWSProvider(state), nil
}

// UpdateAWSProvider is the resolver for the UpdateAWSProvider field.
func (r *mutationResolver) UpdateAWSProvider(ctx context.Context, id string, input model.AWSProviderInput) (*model.AWSProvider, error) {
	if !env.IsAWSProviderEnabled {
		return nil, aws.ErrAWSProviderDisabled
	}

	existing, err := settings.GetAWSProvider(id)
	if err != nil {
		return nil, err
	}

	// Derive auth method from profile name presence
	authMethod := "default"
	profileName := existing.Config.ProfileName
	if input.ProfileName != nil {
		profileName = *input.ProfileName
	}
	if profileName != "" {
		authMethod = "profile"
	}

	discoverRDS := existing.Config.DiscoverRDS
	if input.DiscoverRds != nil {
		discoverRDS = *input.DiscoverRds
	}

	discoverElastiCache := existing.Config.DiscoverElastiCache
	if input.DiscoverElastiCache != nil {
		discoverElastiCache = *input.DiscoverElastiCache
	}

	discoverDocumentDB := existing.Config.DiscoverDocumentDB
	if input.DiscoverDocumentDb != nil {
		discoverDocumentDB = *input.DiscoverDocumentDb
	}

	cfg := &settings.AWSProviderConfig{
		ID:                  id,
		Name:                input.Name,
		Region:              input.Region,
		AuthMethod:          authMethod,
		ProfileName:         profileName,
		DiscoverRDS:         discoverRDS,
		DiscoverElastiCache: discoverElastiCache,
		DiscoverDocumentDB:  discoverDocumentDB,
	}

	state, err := settings.UpdateAWSProvider(id, cfg)
	if err != nil {
		return nil, err
	}

	return stateToAWSProvider(state), nil
}

// TestAWSCredentials is the resolver for the TestAWSCredentials field.
// Tests AWS credentials without creating/persisting a provider.
func (r *mutationResolver) TestAWSCredentials(ctx context.Context, input model.AWSProviderInput) (model.CloudProviderStatus, error) {
	if !env.IsAWSProviderEnabled {
		return model.CloudProviderStatusError, aws.ErrAWSProviderDisabled
	}

	authMethod := "default"
	profileName := ""
	if input.ProfileName != nil && *input.ProfileName != "" {
		authMethod = "profile"
		profileName = *input.ProfileName
	}

	log.Infof("TestAWSCredentials: testing region=%s, authMethod=%s, profileName=%s", input.Region, authMethod, profileName)

	cfg := &awsprovider.Config{
		ID:          "test-temp",
		Name:        input.Name,
		Region:      input.Region,
		AuthMethod:  aws.AuthMethod(authMethod),
		ProfileName: profileName,
	}

	provider, err := awsprovider.New(cfg)
	if err != nil {
		log.Errorf("TestAWSCredentials: failed to create provider: %v", err)
		return model.CloudProviderStatusError, err
	}

	testCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := provider.TestConnection(testCtx); err != nil {
		log.Warnf("TestAWSCredentials: connection test failed: %v", err)
		return model.CloudProviderStatusError, err
	}
	log.Infof("TestAWSCredentials: connection successful for region=%s", input.Region)
	return model.CloudProviderStatusConnected, nil
}

// RemoveCloudProvider is the resolver for the RemoveCloudProvider field.
func (r *mutationResolver) RemoveCloudProvider(ctx context.Context, id string) (*model.StatusResponse, error) {
	if !env.IsAWSProviderEnabled {
		return &model.StatusResponse{Status: false}, aws.ErrAWSProviderDisabled
	}

	// Router pattern: when adding GCP, dispatch based on settings.GetProviderType(id).
	// See .claude/docs/cloud-providers.md for full example.
	err := settings.RemoveAWSProvider(id)
	if err != nil {
		return &model.StatusResponse{Status: false}, err
	}
	return &model.StatusResponse{Status: true}, nil
}

// TestCloudProvider is the resolver for the TestCloudProvider field.
func (r *mutationResolver) TestCloudProvider(ctx context.Context, id string) (model.CloudProviderStatus, error) {
	if !env.IsAWSProviderEnabled {
		return model.CloudProviderStatusError, aws.ErrAWSProviderDisabled
	}

	status, err := settings.TestAWSProvider(id)
	if err != nil {
		return model.CloudProviderStatusError, err
	}
	return mapCloudProviderStatus(status), nil
}

// RefreshCloudProvider is the resolver for the RefreshCloudProvider field.
func (r *mutationResolver) RefreshCloudProvider(ctx context.Context, id string) (*model.AWSProvider, error) {
	if !env.IsAWSProviderEnabled {
		return nil, aws.ErrAWSProviderDisabled
	}

	state, err := settings.RefreshAWSProvider(id)
	if err != nil {
		return nil, err
	}
	return stateToAWSProvider(state), nil
}

// GenerateRDSAuthToken is the resolver for the GenerateRDSAuthToken field.
// Generates a short-lived IAM auth token for RDS database authentication.
func (r *mutationResolver) GenerateRDSAuthToken(ctx context.Context, providerID string, endpoint string, port int, region string, username string) (string, error) {
	if !env.IsAWSProviderEnabled {
		return "", aws.ErrAWSProviderDisabled
	}

	log.Infof("GenerateRDSAuthToken: providerID=%s, endpoint=%s, port=%d, region=%s, username=%s", providerID, endpoint, port, region, username)

	registry := providers.GetDefaultRegistry()
	provider, err := registry.Get(providerID)
	if err != nil {
		log.Errorf("GenerateRDSAuthToken: provider not found: %v", err)
		return "", err
	}

	awsProvider, ok := provider.(*awsprovider.Provider)
	if !ok {
		log.Errorf("GenerateRDSAuthToken: provider %s is not an AWS provider", providerID)
		return "", fmt.Errorf("provider %s is not an AWS provider", providerID)
	}

	tokenCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	token, err := aws.GenerateRDSAuthToken(tokenCtx, awsProvider.GetAWSConfig(), endpoint, port, region, username)
	if err != nil {
		log.Errorf("GenerateRDSAuthToken: failed: %v", err)
		return "", err
	}
	log.Infof("GenerateRDSAuthToken: token generated successfully (length=%d)", len(token))
	return token, nil
}

// CloudProviders is the resolver for the CloudProviders field.
func (r *queryResolver) CloudProviders(ctx context.Context) ([]*model.AWSProvider, error) {
	if !env.IsAWSProviderEnabled {
		return []*model.AWSProvider{}, nil
	}

	// Router pattern: when adding GCP, append settings.GetGCPProviders() results here.
	// See .claude/docs/cloud-providers.md for full example.
	states := settings.GetAWSProviders()
	result := make([]*model.AWSProvider, 0, len(states))
	for _, state := range states {
		result = append(result, stateToAWSProvider(state))
	}
	return result, nil
}

// CloudProvider is the resolver for the CloudProvider field.
func (r *queryResolver) CloudProvider(ctx context.Context, id string) (*model.AWSProvider, error) {
	if !env.IsAWSProviderEnabled {
		return nil, nil
	}

	state, err := settings.GetAWSProvider(id)
	if err != nil {
		return nil, err
	}
	return stateToAWSProvider(state), nil
}

// DiscoveredConnections is the resolver for the DiscoveredConnections field.
func (r *queryResolver) DiscoveredConnections(ctx context.Context) ([]*model.DiscoveredConnection, error) {
	if !env.IsAWSProviderEnabled {
		return []*model.DiscoveredConnection{}, nil
	}

	registry := providers.GetDefaultRegistry()
	conns, err := registry.DiscoverAll(ctx)

	result := make([]*model.DiscoveredConnection, 0, len(conns))
	for _, conn := range conns {
		result = append(result, discoveredConnectionToModel(&conn))
	}

	// Return partial results with error so UI can show a warning
	if err != nil {
		log.Warn("Error discovering connections: ", err)
		return result, err
	}
	return result, nil
}

// ProviderConnections is the resolver for the ProviderConnections field.
func (r *queryResolver) ProviderConnections(ctx context.Context, providerID string) ([]*model.DiscoveredConnection, error) {
	if !env.IsAWSProviderEnabled {
		return []*model.DiscoveredConnection{}, nil
	}

	registry := providers.GetDefaultRegistry()
	conns, err := registry.FilterByProvider(ctx, providerID)
	if err != nil {
		return nil, err
	}

	result := make([]*model.DiscoveredConnection, 0, len(conns))
	for _, conn := range conns {
		result = append(result, discoveredConnectionToModel(&conn))
	}
	return result, nil
}

// LocalAWSProfiles is the resolver for the LocalAWSProfiles field.
func (r *queryResolver) LocalAWSProfiles(ctx context.Context) ([]*model.LocalAWSProfile, error) {
	if !env.IsAWSProviderEnabled {
		return []*model.LocalAWSProfile{}, nil
	}

	localProfiles, err := aws.DiscoverLocalProfiles()
	if err != nil {
		return nil, err
	}

	result := make([]*model.LocalAWSProfile, len(localProfiles))
	for i, profile := range localProfiles {
		var region *string
		if profile.Region != "" {
			region = &profile.Region
		}
		result[i] = &model.LocalAWSProfile{
			Name:      profile.Name,
			Region:    region,
			Source:    profile.Source,
			AuthType:  profile.AuthType,
			IsDefault: profile.IsDefault,
		}
	}
	return result, nil
}

// AWSRegions is the resolver for the AWSRegions field.
func (r *queryResolver) AWSRegions(ctx context.Context) ([]*model.AWSRegion, error) {
	regions := aws.GetRegions()
	result := make([]*model.AWSRegion, len(regions))
	for i, region := range regions {
		result[i] = &model.AWSRegion{
			ID:          region.ID,
			Description: region.Description,
			Partition:   region.Partition,
		}
	}
	return result, nil
}
