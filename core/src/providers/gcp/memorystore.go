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

	"cloud.google.com/go/redis/apiv1/redispb"
	"google.golang.org/api/iterator"

	"github.com/clidey/whodb/core/src/engine"
	gcpinfra "github.com/clidey/whodb/core/src/gcp"
	"github.com/clidey/whodb/core/src/log"
	"github.com/clidey/whodb/core/src/providers"
)

// discoverMemorystore discovers Memorystore for Redis instances in the configured project/region.
func (p *Provider) discoverMemorystore(ctx context.Context) ([]providers.DiscoveredConnection, error) {
	start := time.Now()
	var connections []providers.DiscoveredConnection

	parent := fmt.Sprintf("projects/%s/locations/%s", p.config.ProjectID, p.config.Region)
	log.Debugf("Memorystore: starting discovery for provider %s (parent=%s)", p.config.ID, parent)

	iter := p.memorystoreClient.ListInstances(ctx, &redispb.ListInstancesRequest{
		Parent: parent,
	})

	for {
		if ctx.Err() != nil {
			log.Warnf("Memorystore: context canceled, returning %d results so far", len(connections))
			return connections, ctx.Err()
		}

		instance, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			log.Errorf("Memorystore: ListInstances failed: %v", err)
			return nil, gcpinfra.HandleGCPError(err)
		}

		conn := p.memorystoreInstanceToConnection(instance)
		if conn != nil {
			log.Debugf("Memorystore: instance %s converted to connection", extractResourceName(instance.Name))
			connections = append(connections, *conn)
		}
	}

	log.Debugf("Memorystore: found %d instances in %v", len(connections), time.Since(start))
	return connections, nil
}

func (p *Provider) memorystoreInstanceToConnection(instance *redispb.Instance) *providers.DiscoveredConnection {
	if instance.Host == "" {
		log.Warnf("Memorystore: instance %s has no host, skipping", instance.Name)
		return nil
	}

	instanceName := extractResourceName(instance.Name)

	transitEncryption := "false"
	if instance.TransitEncryptionMode == redispb.Instance_SERVER_AUTHENTICATION {
		transitEncryption = "true"
	}

	metadata := map[string]string{
		"endpoint":          instance.Host,
		"port":              strconv.Itoa(int(instance.Port)),
		"tier":              instance.Tier.String(),
		"transitEncryption": transitEncryption,
		"redisVersion":      instance.RedisVersion,
		"projectId":         p.config.ProjectID,
	}

	if instance.AuthEnabled {
		metadata["authTokenEnabled"] = "true"
	}

	return &providers.DiscoveredConnection{
		ID:           p.connectionID("memorystore-" + instanceName),
		ProviderType: providers.ProviderTypeGCP,
		ProviderID:   p.config.ID,
		Name:         instanceName,
		DatabaseType: engine.DatabaseType_Redis,
		Region:       p.config.Region,
		Status:       mapMemorystoreStatus(instance.State),
		Metadata:     metadata,
	}
}

func mapMemorystoreStatus(state redispb.Instance_State) providers.ConnectionStatus {
	switch state {
	case redispb.Instance_READY:
		return providers.ConnectionStatusAvailable
	case redispb.Instance_CREATING, redispb.Instance_UPDATING, redispb.Instance_IMPORTING:
		return providers.ConnectionStatusStarting
	case redispb.Instance_DELETING:
		return providers.ConnectionStatusDeleting
	case redispb.Instance_FAILING_OVER, redispb.Instance_MAINTENANCE:
		return providers.ConnectionStatusStarting
	default:
		return providers.ConnectionStatusUnknown
	}
}
