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
	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/src/providers"
	"github.com/clidey/whodb/core/src/settings"
	"github.com/clidey/whodb/core/src/source"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct{}

// MapColumnsToModel converts source columns to GraphQL model columns.
func MapColumnsToModel(columnsResult []source.Column) []*model.Column {
	var columns []*model.Column
	for _, column := range columnsResult {
		columns = append(columns, &model.Column{
			Type:             column.Type,
			Name:             column.Name,
			IsPrimary:        column.IsPrimary,
			IsForeignKey:     column.IsForeignKey,
			ReferencedTable:  column.ReferencedTable,
			ReferencedColumn: column.ReferencedColumn,
			Length:           column.Length,
			Precision:        column.Precision,
			Scale:            column.Scale,
		})
	}
	return columns
}

// stateToAWSProvider converts settings.AWSProviderState to the GraphQL model.
func stateToAWSProvider(state *settings.AWSProviderState) *model.AWSProvider {
	var profileName *string
	if state.Config.ProfileName != "" {
		profileName = &state.Config.ProfileName
	}

	var lastDiscoveryAt *string
	if state.LastDiscoveryAt != nil {
		t := state.LastDiscoveryAt.Format("2006-01-02T15:04:05Z")
		lastDiscoveryAt = &t
	}

	var errorStr *string
	if state.Error != "" {
		errorStr = &state.Error
	}

	return &model.AWSProvider{
		ID:                  state.Config.ID,
		ProviderType:        model.CloudProviderTypeAWS,
		Name:                state.Config.Name,
		Region:              state.Config.Region,
		ProfileName:         profileName,
		DiscoverRds:         state.Config.DiscoverRDS,
		DiscoverElastiCache: state.Config.DiscoverElastiCache,
		DiscoverDocumentDb:  state.Config.DiscoverDocumentDB,
		DiscoverS3:          state.Config.DiscoverS3,
		Status:              mapCloudProviderStatus(state.Status),
		LastDiscoveryAt:     lastDiscoveryAt,
		DiscoveredCount:     state.DiscoveredCount,
		Error:               errorStr,
	}
}

// stateToGCPProvider converts settings.GCPProviderState to the GraphQL model.
func stateToGCPProvider(state *settings.GCPProviderState) *model.GCPProvider {
	var serviceAccountKeyPath *string
	if state.Config.ServiceAccountKeyPath != "" {
		serviceAccountKeyPath = &state.Config.ServiceAccountKeyPath
	}

	var lastDiscoveryAt *string
	if state.LastDiscoveryAt != nil {
		t := state.LastDiscoveryAt.Format("2006-01-02T15:04:05Z")
		lastDiscoveryAt = &t
	}

	var errorStr *string
	if state.Error != "" {
		errorStr = &state.Error
	}

	return &model.GCPProvider{
		ID:                    state.Config.ID,
		ProviderType:          model.CloudProviderTypeGCP,
		Name:                  state.Config.Name,
		Region:                state.Config.Region,
		ProjectID:             state.Config.ProjectID,
		ServiceAccountKeyPath: serviceAccountKeyPath,
		DiscoverCloudSQL:      state.Config.DiscoverCloudSQL,
		DiscoverAlloyDb:       state.Config.DiscoverAlloyDB,
		DiscoverMemorystore:   state.Config.DiscoverMemorystore,
		Status:                mapCloudProviderStatus(state.Status),
		LastDiscoveryAt:       lastDiscoveryAt,
		DiscoveredCount:       state.DiscoveredCount,
		Error:                 errorStr,
	}
}

// mapCloudProviderStatus converts a status string to the GraphQL enum.
func mapCloudProviderStatus(status string) model.CloudProviderStatus {
	switch status {
	case "Connected":
		return model.CloudProviderStatusConnected
	case "Discovering":
		return model.CloudProviderStatusDiscovering
	case "Error":
		return model.CloudProviderStatusError
	default:
		return model.CloudProviderStatusDisconnected
	}
}

// mapProviderTypeToModel converts providers.ProviderType to the GraphQL enum.
func mapProviderTypeToModel(pt providers.ProviderType) model.CloudProviderType {
	switch pt {
	case providers.ProviderTypeAWS:
		return model.CloudProviderTypeAWS
	case providers.ProviderTypeAzure:
		return model.CloudProviderTypeAzure
	case providers.ProviderTypeGCP:
		return model.CloudProviderTypeGCP
	default:
		return model.CloudProviderTypeAWS
	}
}

// mapConnectionStatusToModel converts providers.ConnectionStatus to the GraphQL enum.
func mapConnectionStatusToModel(status providers.ConnectionStatus) model.ConnectionStatus {
	switch status {
	case providers.ConnectionStatusAvailable:
		return model.ConnectionStatusAvailable
	case providers.ConnectionStatusStarting:
		return model.ConnectionStatusStarting
	case providers.ConnectionStatusStopped:
		return model.ConnectionStatusStopped
	case providers.ConnectionStatusDeleting:
		return model.ConnectionStatusDeleting
	case providers.ConnectionStatusFailed:
		return model.ConnectionStatusFailed
	default:
		return model.ConnectionStatusUnknown
	}
}

// discoveredConnectionToModel converts providers.DiscoveredConnection to the GraphQL model.
// Connection metadata (endpoint, port, TLS settings) is exposed to the frontend for
// prefilling the login form, allowing users to modify values before connecting.
func discoveredConnectionToModel(conn *providers.DiscoveredConnection) *model.DiscoveredConnection {
	var region *string
	if conn.Region != "" {
		region = &conn.Region
	}

	// Expose metadata needed for UI prefill and connection decisions.
	// - endpoint: database hostname for connection prefill
	// - port: database port for connection prefill
	// - transitEncryption: TLS setting for ElastiCache/Redis
	// - serverless: indicates serverless deployment (affects UI hints)
	// - iamAuthEnabled: determines if password is optional for RDS
	// - authTokenEnabled: Redis AUTH token hint
	allowedMetadataKeys := map[string]bool{
		"endpoint":          true,
		"port":              true,
		"databaseName":      true,
		"transitEncryption": true,
		"serverless":        true,
		"iamAuthEnabled":    true,
		"authTokenEnabled":  true,
		"iamAuthSupported":  true,
		"endpointType":      true,
		"clusterIdentifier": true,
		"proxyName":         true,
		"requireTLS":        true,
		"connectivity":      true,
		"service":           true,
		"region":            true,
		"bucket":            true,
		"authMethod":        true,
		"profileName":       true,
		// Azure-specific metadata
		"location":         true,
		"resourceGroup":    true,
		"sku":              true,
		"version":          true,
		"enableNonSslPort": true,
		"nonSslPort":       true,
		"kind":             true,
		"connectionName":   true,
		"databaseVersion":  true,
		"tier":             true,
		"instanceType":     true,
		"clusterName":      true,
		"redisVersion":     true,
		"projectId":        true,
	}

	var metadata []*model.Record
	for k, v := range conn.Metadata {
		if allowedMetadataKeys[k] {
			metadata = append(metadata, &model.Record{Key: k, Value: v})
		}
	}

	return &model.DiscoveredConnection{
		ID:           conn.ID,
		ProviderType: mapProviderTypeToModel(conn.ProviderType),
		ProviderID:   conn.ProviderID,
		Name:         conn.Name,
		SourceType:   string(conn.DatabaseType),
		Region:       region,
		Status:       mapConnectionStatusToModel(conn.Status),
		Metadata:     metadata,
	}
}

// stateToAzureProvider converts settings.AzureProviderState to the GraphQL model.
func stateToAzureProvider(state *settings.AzureProviderState) *model.AzureProvider {
	var tenantID *string
	if state.Config.TenantID != "" {
		tenantID = &state.Config.TenantID
	}

	var resourceGroup *string
	if state.Config.ResourceGroup != "" {
		resourceGroup = &state.Config.ResourceGroup
	}

	var lastDiscoveryAt *string
	if state.LastDiscoveryAt != nil {
		t := state.LastDiscoveryAt.Format("2006-01-02T15:04:05Z")
		lastDiscoveryAt = &t
	}

	var errorStr *string
	if state.Error != "" {
		errorStr = &state.Error
	}

	return &model.AzureProvider{
		ID:                 state.Config.ID,
		ProviderType:       model.CloudProviderTypeAzure,
		Name:               state.Config.Name,
		Region:             state.Config.SubscriptionID,
		SubscriptionID:     state.Config.SubscriptionID,
		TenantID:           tenantID,
		ResourceGroup:      resourceGroup,
		DiscoverPostgreSQL: state.Config.DiscoverPostgreSQL,
		DiscoverMySQL:      state.Config.DiscoverMySQL,
		DiscoverRedis:      state.Config.DiscoverRedis,
		DiscoverCosmosDb:   state.Config.DiscoverCosmosDB,
		Status:             mapCloudProviderStatus(state.Status),
		LastDiscoveryAt:    lastDiscoveryAt,
		DiscoveredCount:    state.DiscoveredCount,
		Error:              errorStr,
	}
}

// derefStringOr returns the dereferenced string or the fallback if nil.
func derefStringOr(s *string, fallback string) string {
	if s != nil {
		return *s
	}
	return fallback
}

// derefBoolOr returns the dereferenced bool or the fallback if nil.
func derefBoolOr(b *bool, fallback bool) bool {
	if b != nil {
		return *b
	}
	return fallback
}
