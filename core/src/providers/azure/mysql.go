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
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/mysql/armmysqlflexibleservers"

	azureinfra "github.com/clidey/whodb/core/src/azure"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
	"github.com/clidey/whodb/core/src/providers"
)

// discoverMySQL discovers Azure Database for MySQL Flexible Servers.
func (p *Provider) discoverMySQL(ctx context.Context) ([]providers.DiscoveredConnection, error) {
	start := time.Now()
	var connections []providers.DiscoveredConnection

	log.Debugf("Azure MySQL: starting discovery for provider %s", p.config.ID)

	var pager interface {
		More() bool
		NextPage(ctx context.Context) (armmysqlflexibleservers.ServersClientListResponse, error)
	}

	if p.config.ResourceGroup != "" {
		pager = toMySQLListPager(p.mysqlClient.NewListByResourceGroupPager(p.config.ResourceGroup, nil))
	} else {
		pager = p.mysqlClient.NewListPager(nil)
	}

	for pager.More() {
		if ctx.Err() != nil {
			log.Warnf("Azure MySQL: context canceled, returning %d results so far", len(connections))
			return connections, ctx.Err()
		}

		page, err := pager.NextPage(ctx)
		if err != nil {
			log.Errorf("Azure MySQL: list failed: %v", err)
			return connections, azureinfra.HandleAzureError(err)
		}

		for _, server := range page.Value {
			if server == nil || server.Name == nil {
				continue
			}

			conn := p.mysqlServerToConnection(server)
			connections = append(connections, conn)
		}
	}

	log.Infof("Azure MySQL: discovered %d servers in %v", len(connections), time.Since(start))
	return connections, nil
}

func (p *Provider) mysqlServerToConnection(server *armmysqlflexibleservers.Server) providers.DiscoveredConnection {
	name := derefStr(server.Name)
	location := derefStr(server.Location)
	resourceGroup := extractResourceGroup(derefStr(server.ID))

	metadata := map[string]string{
		"port":          "3306",
		"location":      location,
		"resourceGroup": resourceGroup,
	}

	var status providers.ConnectionStatus
	if server.Properties != nil {
		if server.Properties.FullyQualifiedDomainName != nil {
			metadata["endpoint"] = *server.Properties.FullyQualifiedDomainName
		}
		if server.Properties.Version != nil {
			metadata["version"] = string(*server.Properties.Version)
		}
		status = mapMySQLStatus(server.Properties.State)
	} else {
		status = providers.ConnectionStatusUnknown
	}

	if server.SKU != nil && server.SKU.Name != nil {
		metadata["sku"] = *server.SKU.Name
	}

	return providers.DiscoveredConnection{
		ID:           p.connectionID("mysql-" + name),
		ProviderType: providers.ProviderTypeAzure,
		ProviderID:   p.config.ID,
		Name:         name,
		DatabaseType: engine.DatabaseType_MySQL,
		Region:       location,
		Status:       status,
		Metadata:     metadata,
	}
}

func mapMySQLStatus(state *armmysqlflexibleservers.ServerState) providers.ConnectionStatus {
	if state == nil {
		return providers.ConnectionStatusUnknown
	}
	switch *state {
	case armmysqlflexibleservers.ServerStateReady:
		return providers.ConnectionStatusAvailable
	case armmysqlflexibleservers.ServerStateStarting,
		armmysqlflexibleservers.ServerStateUpdating:
		return providers.ConnectionStatusStarting
	case armmysqlflexibleservers.ServerStateStopped,
		armmysqlflexibleservers.ServerStateStopping,
		armmysqlflexibleservers.ServerStateDisabled:
		return providers.ConnectionStatusStopped
	case armmysqlflexibleservers.ServerStateDropping:
		return providers.ConnectionStatusDeleting
	default:
		return providers.ConnectionStatusUnknown
	}
}

// toMySQLListPager adapts the ResourceGroup pager to the subscription-level interface.
type mysqlRGPagerWrapper struct {
	pager interface {
		More() bool
		NextPage(ctx context.Context) (armmysqlflexibleservers.ServersClientListByResourceGroupResponse, error)
	}
}

func (w *mysqlRGPagerWrapper) More() bool {
	return w.pager.More()
}

func (w *mysqlRGPagerWrapper) NextPage(ctx context.Context) (armmysqlflexibleservers.ServersClientListResponse, error) {
	resp, err := w.pager.NextPage(ctx)
	if err != nil {
		return armmysqlflexibleservers.ServersClientListResponse{}, err
	}
	return armmysqlflexibleservers.ServersClientListResponse(resp), nil
}

func toMySQLListPager(pager interface {
	More() bool
	NextPage(ctx context.Context) (armmysqlflexibleservers.ServersClientListByResourceGroupResponse, error)
}) interface {
	More() bool
	NextPage(ctx context.Context) (armmysqlflexibleservers.ServersClientListResponse, error)
} {
	return &mysqlRGPagerWrapper{pager: pager}
}
