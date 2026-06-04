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

package gcp

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	memcachepb "cloud.google.com/go/memcache/apiv1/memcachepb"
	"google.golang.org/api/iterator"

	"github.com/clidey/whodb/core/src/engine"
	gcpinfra "github.com/clidey/whodb/core/src/gcp"
	"github.com/clidey/whodb/core/src/log"
	"github.com/clidey/whodb/core/src/providers"
)

// discoverMemcached discovers Memorystore for Memcached instances in the configured project/region.
func (p *Provider) discoverMemcached(ctx context.Context) ([]providers.DiscoveredConnection, error) {
	start := time.Now()
	var connections []providers.DiscoveredConnection

	parent := fmt.Sprintf("projects/%s/locations/%s", p.config.ProjectID, p.config.Region)
	log.Debugf("Memcached: starting discovery for provider %s (parent=%s)", p.config.ID, parent)

	iter := p.memcachedClient.ListInstances(ctx, &memcachepb.ListInstancesRequest{
		Parent: parent,
	})

	for {
		if ctx.Err() != nil {
			log.Warnf("Memcached: context canceled, returning %d results so far", len(connections))
			return connections, ctx.Err()
		}

		instance, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			log.Errorf("Memcached: ListInstances failed: %v", err)
			return nil, gcpinfra.HandleGCPError(err)
		}

		conn := p.memcachedInstanceToConnection(instance)
		if conn != nil {
			connections = append(connections, *conn)
		}
	}

	log.Infof("Memcached: found %d instances in %v", len(connections), time.Since(start))
	return connections, nil
}

func (p *Provider) memcachedInstanceToConnection(instance *memcachepb.Instance) *providers.DiscoveredConnection {
	if len(instance.MemcacheNodes) == 0 {
		log.Warnf("Memcached: instance %s has no nodes, skipping", instance.Name)
		return nil
	}

	firstNode := instance.MemcacheNodes[0]
	if firstNode.Host == "" {
		log.Warnf("Memcached: instance %s node has no host, skipping", instance.Name)
		return nil
	}

	instanceName := extractResourceName(instance.Name)

	metadata := map[string]string{
		"endpoint":  firstNode.Host,
		"port":      strconv.Itoa(int(firstNode.Port)),
		"nodeCount": strconv.Itoa(int(instance.NodeCount)),
		"projectId": p.config.ProjectID,
	}

	if instance.DisplayName != "" {
		metadata["displayName"] = instance.DisplayName
	}

	return &providers.DiscoveredConnection{
		ID:           p.connectionID("memcached-" + instanceName),
		ProviderType: providers.ProviderTypeGCP,
		ProviderID:   p.config.ID,
		Name:         instanceName,
		DatabaseType: engine.DatabaseType_Memcached,
		Region:       p.config.Region,
		Status:       mapMemcachedStatus(instance.State),
		Metadata:     metadata,
	}
}

func mapMemcachedStatus(state memcachepb.Instance_State) providers.ConnectionStatus {
	switch state {
	case memcachepb.Instance_READY:
		return providers.ConnectionStatusAvailable
	case memcachepb.Instance_CREATING, memcachepb.Instance_UPDATING:
		return providers.ConnectionStatusStarting
	case memcachepb.Instance_DELETING:
		return providers.ConnectionStatusDeleting
	case memcachepb.Instance_PERFORMING_MAINTENANCE:
		return providers.ConnectionStatusStarting
	default:
		return providers.ConnectionStatusUnknown
	}
}
