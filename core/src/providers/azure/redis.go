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
	"strconv"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/redis/armredis"

	azureinfra "github.com/clidey/whodb/core/src/azure"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
	"github.com/clidey/whodb/core/src/providers"
)

// discoverRedis discovers Azure Cache for Redis instances.
func (p *Provider) discoverRedis(ctx context.Context) ([]providers.DiscoveredConnection, error) {
	start := time.Now()
	var connections []providers.DiscoveredConnection

	log.Debugf("Azure Redis: starting discovery for provider %s", p.config.ID)

	var pager interface {
		More() bool
		NextPage(ctx context.Context) (armredis.ClientListBySubscriptionResponse, error)
	}

	if p.config.ResourceGroup != "" {
		pager = toRedisListPager(p.redisClient.NewListByResourceGroupPager(p.config.ResourceGroup, nil))
	} else {
		pager = p.redisClient.NewListBySubscriptionPager(nil)
	}

	for pager.More() {
		if ctx.Err() != nil {
			log.Warnf("Azure Redis: context canceled, returning %d results so far", len(connections))
			return connections, ctx.Err()
		}

		page, err := pager.NextPage(ctx)
		if err != nil {
			log.Errorf("Azure Redis: list failed: %v", err)
			return connections, azureinfra.HandleAzureError(err)
		}

		for _, cache := range page.Value {
			if cache == nil || cache.Name == nil {
				continue
			}

			conn := p.redisCacheToConnection(cache)
			connections = append(connections, conn)
		}
	}

	log.Infof("Azure Redis: discovered %d caches in %v", len(connections), time.Since(start))
	return connections, nil
}

func (p *Provider) redisCacheToConnection(cache *armredis.ResourceInfo) providers.DiscoveredConnection {
	name := derefStr(cache.Name)
	location := derefStr(cache.Location)
	resourceGroup := extractResourceGroup(derefStr(cache.ID))

	metadata := map[string]string{
		"location":      location,
		"resourceGroup": resourceGroup,
	}

	var status providers.ConnectionStatus
	if cache.Properties != nil {
		if cache.Properties.HostName != nil {
			metadata["endpoint"] = *cache.Properties.HostName
		}

		// Use SSL port by default; fall back to non-SSL port if SSL is disabled
		if cache.Properties.SSLPort != nil {
			metadata["port"] = strconv.Itoa(int(*cache.Properties.SSLPort))
		}
		if cache.Properties.EnableNonSSLPort != nil {
			metadata["enableNonSslPort"] = strconv.FormatBool(*cache.Properties.EnableNonSSLPort)
			if *cache.Properties.EnableNonSSLPort && cache.Properties.Port != nil {
				metadata["nonSslPort"] = strconv.Itoa(int(*cache.Properties.Port))
			}
		}

		status = mapRedisStatus(cache.Properties.ProvisioningState)
	} else {
		status = providers.ConnectionStatusUnknown
	}

	return providers.DiscoveredConnection{
		ID:           p.connectionID("redis-" + name),
		ProviderType: providers.ProviderTypeAzure,
		ProviderID:   p.config.ID,
		Name:         name,
		DatabaseType: engine.DatabaseType_Redis,
		Region:       location,
		Status:       status,
		Metadata:     metadata,
	}
}

func mapRedisStatus(state *armredis.ProvisioningState) providers.ConnectionStatus {
	if state == nil {
		return providers.ConnectionStatusUnknown
	}
	switch *state {
	case armredis.ProvisioningStateSucceeded:
		return providers.ConnectionStatusAvailable
	case armredis.ProvisioningStateCreating,
		armredis.ProvisioningStateProvisioning,
		armredis.ProvisioningStateLinking,
		armredis.ProvisioningStateScaling,
		armredis.ProvisioningStateUpdating,
		armredis.ProvisioningStateRecoveringScaleFailure:
		return providers.ConnectionStatusStarting
	case armredis.ProvisioningStateDisabled,
		armredis.ProvisioningStateUnlinking,
		armredis.ProvisioningStateUnprovisioning:
		return providers.ConnectionStatusStopped
	case armredis.ProvisioningStateDeleting:
		return providers.ConnectionStatusDeleting
	case armredis.ProvisioningStateFailed:
		return providers.ConnectionStatusFailed
	default:
		return providers.ConnectionStatusUnknown
	}
}

// toRedisListPager adapts the ResourceGroup pager to the subscription-level interface.
type redisRGPagerWrapper struct {
	pager interface {
		More() bool
		NextPage(ctx context.Context) (armredis.ClientListByResourceGroupResponse, error)
	}
}

func (w *redisRGPagerWrapper) More() bool {
	return w.pager.More()
}

func (w *redisRGPagerWrapper) NextPage(ctx context.Context) (armredis.ClientListBySubscriptionResponse, error) {
	resp, err := w.pager.NextPage(ctx)
	if err != nil {
		return armredis.ClientListBySubscriptionResponse{}, err
	}
	return armredis.ClientListBySubscriptionResponse(resp), nil
}

func toRedisListPager(pager interface {
	More() bool
	NextPage(ctx context.Context) (armredis.ClientListByResourceGroupResponse, error)
}) interface {
	More() bool
	NextPage(ctx context.Context) (armredis.ClientListBySubscriptionResponse, error)
} {
	return &redisRGPagerWrapper{pager: pager}
}
