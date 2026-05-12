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
	"sync"
	"time"

	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/src"
	"github.com/clidey/whodb/core/src/auth"
	"github.com/clidey/whodb/core/src/dashboard"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
	"github.com/clidey/whodb/core/src/providers"
	"github.com/clidey/whodb/core/src/settings"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

// Resolver owns GraphQL resolver dependencies.
type Resolver struct {
	DashboardService        dashboard.ServiceAPI
	dashboardServiceFactory func() (dashboard.ServiceAPI, error)
	dashboardServiceMu      sync.Mutex
}

// NewResolver creates the GraphQL dependency resolver.
func NewResolver() *Resolver {
	return newResolverWithDashboardFactory(func() (dashboard.ServiceAPI, error) {
		return dashboard.NewServiceFromEnv()
	})
}

func newResolverWithDashboardFactory(factory func() (dashboard.ServiceAPI, error)) *Resolver {
	return &Resolver{dashboardServiceFactory: factory}
}

// GetPluginForContext returns the appropriate database plugin and config for the current session.
func GetPluginForContext(ctx context.Context) (*engine.Plugin, *engine.PluginConfig) {
	creds := auth.GetCredentials(ctx)
	log.Debugf("[GetPluginForContext] credentials: type=%s, hostname=%s, advanced count=%d", creds.Type, creds.Hostname, len(creds.Advanced))
	for _, rec := range creds.Advanced {
		log.Debugf("[GetPluginForContext] advanced: key=%q value=%q", rec.Key, rec.Value)
	}
	config := engine.NewPluginConfig(creds)
	plugin := src.MainEngine.Choose(engine.DatabaseType(config.Credentials.Type))
	return plugin, config
}

// ValidateStorageUnit checks that a storage unit exists in the given schema.
// This prevents SQL injection by ensuring only existing table names are used.
func ValidateStorageUnit(plugin engine.PluginFunctions, config *engine.PluginConfig, schema string, storageUnit string) error {
	exists, err := plugin.StorageUnitExists(config, schema, storageUnit)
	if err != nil {
		return fmt.Errorf("failed to validate storage unit: %w", err)
	}
	if !exists {
		return fmt.Errorf("storage unit %q not found in schema %q", storageUnit, schema)
	}
	return nil
}

// MapColumnsToModel converts engine columns to GraphQL model columns
func MapColumnsToModel(columnsResult []engine.Column) []*model.Column {
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

// FetchColumnsForStorageUnit retrieves column information for a single storage unit.
func FetchColumnsForStorageUnit(
	plugin engine.PluginFunctions,
	config *engine.PluginConfig,
	schema string,
	storageUnit string,
) ([]*model.Column, error) {
	if err := ValidateStorageUnit(plugin, config, schema, storageUnit); err != nil {
		return nil, err
	}

	columnsResult, err := plugin.GetColumnsForTable(config, schema, storageUnit)
	if err != nil {
		return nil, fmt.Errorf("failed to get columns for %s.%s: %w", schema, storageUnit, err)
	}

	return MapColumnsToModel(columnsResult), nil
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
		Status:              mapCloudProviderStatus(state.Status),
		LastDiscoveryAt:     lastDiscoveryAt,
		DiscoveredCount:     state.DiscoveredCount,
		Error:               errorStr,
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
	default:
		// Default to AWS since only cloud providers appear in DiscoveredConnection
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
		DatabaseType: string(conn.DatabaseType),
		Region:       region,
		Status:       mapConnectionStatusToModel(conn.Status),
		Metadata:     metadata,
	}
}

func requireDashboardService(resolver *Resolver) (dashboard.ServiceAPI, error) {
	if resolver == nil {
		return nil, dashboard.ErrMetadataStoreNotConfigured
	}
	if resolver.DashboardService != nil {
		return resolver.DashboardService, nil
	}
	if resolver.dashboardServiceFactory == nil {
		return nil, dashboard.ErrMetadataStoreNotConfigured
	}

	resolver.dashboardServiceMu.Lock()
	defer resolver.dashboardServiceMu.Unlock()

	if resolver.DashboardService != nil {
		return resolver.DashboardService, nil
	}

	service, err := resolver.dashboardServiceFactory()
	if err != nil {
		log.Warnf("dashboard metadata unavailable: %v", err)
		return nil, err
	}

	resolver.DashboardService = service
	return service, nil
}

func mapDashboardModel(item dashboard.Dashboard) *model.Dashboard {
	widgets := make([]*model.DashboardWidget, 0, len(item.Widgets))
	for _, widget := range item.Widgets {
		widgets = append(widgets, mapWidgetModel(widget))
	}

	return &model.Dashboard{
		ID:          item.ID,
		Name:        item.Name,
		Description: item.Description,
		RefreshRule: item.RefreshRule,
		Widgets:     widgets,
		CreatedAt:   item.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:   item.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func mapWidgetModel(item dashboard.Widget) *model.DashboardWidget {
	return &model.DashboardWidget{
		ID:            item.ID,
		Type:          item.Type,
		Title:         item.Title,
		Description:   item.Description,
		Layout:        string(item.Layout),
		Query:         item.Query,
		QueryContext:  rawJSONToStringPtr(item.QueryContext),
		Visualization: rawJSONToStringPtr(item.Visualization),
		Snapshot:      rawJSONToStringPtr(item.Snapshot),
		SortOrder:     item.SortOrder,
	}
}

func rawJSONToStringPtr(value []byte) *string {
	if len(value) == 0 {
		return nil
	}

	text := string(value)
	return &text
}
