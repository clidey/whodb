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

package azure

import (
	"context"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/cosmos/armcosmos/v3"

	azureinfra "github.com/clidey/whodb/core/src/azure"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
	"github.com/clidey/whodb/core/src/providers"
)

// discoverCosmosDB discovers Azure Cosmos DB accounts with MongoDB API.
// Only accounts with Kind == "MongoDB" or the EnableMongo capability are included.
func (p *Provider) discoverCosmosDB(ctx context.Context) ([]providers.DiscoveredConnection, error) {
	start := time.Now()
	var connections []providers.DiscoveredConnection

	log.Debugf("Azure CosmosDB: starting discovery for provider %s", p.config.ID)

	var pager interface {
		More() bool
		NextPage(ctx context.Context) (armcosmos.DatabaseAccountsClientListResponse, error)
	}

	if p.config.ResourceGroup != "" {
		pager = toCosmosListPager(p.cosmosClient.NewListByResourceGroupPager(p.config.ResourceGroup, nil))
	} else {
		pager = p.cosmosClient.NewListPager(nil)
	}

	for pager.More() {
		if ctx.Err() != nil {
			log.Warnf("Azure CosmosDB: context canceled, returning %d results so far", len(connections))
			return connections, ctx.Err()
		}

		page, err := pager.NextPage(ctx)
		if err != nil {
			log.Errorf("Azure CosmosDB: list failed: %v", err)
			return connections, azureinfra.HandleAzureError(err)
		}

		for _, account := range page.Value {
			if account == nil || account.Name == nil {
				continue
			}

			if !isMongoDBAccount(account) {
				continue
			}

			conn := p.cosmosAccountToConnection(account)
			connections = append(connections, conn)
		}
	}

	log.Infof("Azure CosmosDB: discovered %d MongoDB accounts in %v", len(connections), time.Since(start))
	return connections, nil
}

// isMongoDBAccount checks if a Cosmos DB account uses the MongoDB API.
func isMongoDBAccount(account *armcosmos.DatabaseAccountGetResults) bool {
	// Check Kind field first — most reliable
	if account.Kind != nil && *account.Kind == armcosmos.DatabaseAccountKindMongoDB {
		return true
	}

	// Fall back to checking capabilities for EnableMongo
	if account.Properties != nil {
		for _, cap := range account.Properties.Capabilities {
			if cap != nil && cap.Name != nil && strings.EqualFold(*cap.Name, "EnableMongo") {
				return true
			}
		}
	}

	return false
}

func (p *Provider) cosmosAccountToConnection(account *armcosmos.DatabaseAccountGetResults) providers.DiscoveredConnection {
	name := derefStr(account.Name)
	location := derefStr(account.Location)
	resourceGroup := extractResourceGroup(derefStr(account.ID))

	metadata := map[string]string{
		"port":          "10255",
		"location":      location,
		"resourceGroup": resourceGroup,
	}

	if account.Kind != nil {
		metadata["kind"] = string(*account.Kind)
	}

	var status providers.ConnectionStatus
	if account.Properties != nil {
		if account.Properties.DocumentEndpoint != nil {
			// Extract hostname from the document endpoint URL
			endpoint := *account.Properties.DocumentEndpoint
			endpoint = strings.TrimPrefix(endpoint, "https://")
			endpoint = strings.TrimPrefix(endpoint, "http://")
			endpoint = strings.TrimSuffix(endpoint, "/")
			if idx := strings.Index(endpoint, ":"); idx != -1 {
				endpoint = endpoint[:idx]
			}
			metadata["endpoint"] = endpoint
		}
		status = mapCosmosStatus(account.Properties.ProvisioningState)
	} else {
		status = providers.ConnectionStatusUnknown
	}

	return providers.DiscoveredConnection{
		ID:           p.connectionID("cosmos-" + name),
		ProviderType: providers.ProviderTypeAzure,
		ProviderID:   p.config.ID,
		Name:         name,
		DatabaseType: engine.DatabaseType_MongoDB,
		Region:       location,
		Status:       status,
		Metadata:     metadata,
	}
}

func mapCosmosStatus(state *string) providers.ConnectionStatus {
	if state == nil {
		return providers.ConnectionStatusUnknown
	}
	switch strings.ToLower(*state) {
	case "succeeded":
		return providers.ConnectionStatusAvailable
	case "creating", "updating":
		return providers.ConnectionStatusStarting
	case "deleting":
		return providers.ConnectionStatusDeleting
	case "failed":
		return providers.ConnectionStatusFailed
	default:
		return providers.ConnectionStatusUnknown
	}
}

// toCosmosListPager adapts the ResourceGroup pager to the subscription-level interface.
type cosmosRGPagerWrapper struct {
	pager interface {
		More() bool
		NextPage(ctx context.Context) (armcosmos.DatabaseAccountsClientListByResourceGroupResponse, error)
	}
}

func (w *cosmosRGPagerWrapper) More() bool {
	return w.pager.More()
}

func (w *cosmosRGPagerWrapper) NextPage(ctx context.Context) (armcosmos.DatabaseAccountsClientListResponse, error) {
	resp, err := w.pager.NextPage(ctx)
	if err != nil {
		return armcosmos.DatabaseAccountsClientListResponse{}, err
	}
	return armcosmos.DatabaseAccountsClientListResponse(resp), nil
}

func toCosmosListPager(pager interface {
	More() bool
	NextPage(ctx context.Context) (armcosmos.DatabaseAccountsClientListByResourceGroupResponse, error)
}) interface {
	More() bool
	NextPage(ctx context.Context) (armcosmos.DatabaseAccountsClientListResponse, error)
} {
	return &cosmosRGPagerWrapper{pager: pager}
}
