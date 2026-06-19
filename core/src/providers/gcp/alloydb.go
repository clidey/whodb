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
	"strings"
	"time"

	alloydbpb "cloud.google.com/go/alloydb/apiv1/alloydbpb"
	"google.golang.org/api/iterator"

	"github.com/clidey/whodb/core/src/engine"
	gcpinfra "github.com/clidey/whodb/core/src/gcp"
	"github.com/clidey/whodb/core/src/log"
	"github.com/clidey/whodb/core/src/providers"
)

// discoverAlloyDB discovers AlloyDB clusters and their instances in the configured project/region.
// AlloyDB is PostgreSQL-compatible, so all instances map to the Postgres database type.
func (p *Provider) discoverAlloyDB(ctx context.Context) ([]providers.DiscoveredConnection, error) {
	start := time.Now()
	var connections []providers.DiscoveredConnection

	parent := fmt.Sprintf("projects/%s/locations/%s", p.config.ProjectID, p.config.Region)
	log.Debugf("AlloyDB: starting discovery for provider %s (parent=%s)", p.config.ID, parent)

	clusterIter := p.alloydbClient.ListClusters(ctx, &alloydbpb.ListClustersRequest{
		Parent: parent,
	})

	clusterCount := 0
	for {
		if ctx.Err() != nil {
			log.Warnf("AlloyDB: context canceled, returning %d results so far", len(connections))
			return connections, ctx.Err()
		}
		if clusterCount >= maxPaginationPages {
			log.Warnf("AlloyDB: hit cluster pagination limit of %d", maxPaginationPages)
			break
		}
		cluster, err := clusterIter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			log.Errorf("AlloyDB: ListClusters failed: %v", err)
			return nil, gcpinfra.HandleGCPError(err)
		}

		clusterCount++
		clusterName := extractResourceName(cluster.Name)
		log.Debugf("AlloyDB: processing cluster %s", clusterName)

		// Discover instances within this cluster
		instanceIter := p.alloydbClient.ListInstances(ctx, &alloydbpb.ListInstancesRequest{
			Parent: cluster.Name,
		})

		for instanceCount := 0; ; instanceCount++ {
			if ctx.Err() != nil {
				log.Warnf("AlloyDB: context canceled during instance iteration, returning %d results so far", len(connections))
				return connections, ctx.Err()
			}
			if instanceCount >= maxPaginationPages {
				log.Warnf("AlloyDB: hit instance pagination limit of %d for cluster %s", maxPaginationPages, clusterName)
				break
			}
			instance, err := instanceIter.Next()
			if errors.Is(err, iterator.Done) {
				break
			}
			if err != nil {
				log.Errorf("AlloyDB: ListInstances failed for cluster %s: %v", clusterName, err)
				return connections, fmt.Errorf("AlloyDB ListInstances for cluster %s: %w", clusterName, err)
			}

			conn := p.alloyDBInstanceToConnection(instance, clusterName)
			if conn != nil {
				log.Debugf("AlloyDB: instance %s converted to connection", extractResourceName(instance.Name))
				connections = append(connections, *conn)
			}
		}
	}

	log.Debugf("AlloyDB: found %d instances across %d clusters in %v", len(connections), clusterCount, time.Since(start))
	return connections, nil
}

func (p *Provider) alloyDBInstanceToConnection(instance *alloydbpb.Instance, clusterName string) *providers.DiscoveredConnection {
	// Prefer public IP if available, fall back to private IP
	endpoint := instance.PublicIpAddress
	if endpoint == "" {
		endpoint = instance.IpAddress
	}
	if endpoint == "" {
		log.Warnf("AlloyDB: instance %s has no IP address, skipping", instance.Name)
		return nil
	}

	instanceName := extractResourceName(instance.Name)

	// Map instance type to endpointType metadata (mirrors AWS Aurora writer/reader pattern)
	endpointType := "writer"
	suffix := " (writer)"
	switch instance.InstanceType {
	case alloydbpb.Instance_READ_POOL:
		endpointType = "reader"
		suffix = " (reader)"
	case alloydbpb.Instance_SECONDARY:
		endpointType = "reader"
		suffix = " (cross-region reader)"
	}

	metadata := map[string]string{
		"endpoint":     endpoint,
		"port":         "5432",
		"endpointType": endpointType,
		"clusterName":  clusterName,
		"instanceType": instance.InstanceType.String(),
		"projectId":    p.config.ProjectID,
	}

	log.Debugf("AlloyDB: instance %s type=%s endpoint=%s endpointType=%s", instanceName, instance.InstanceType, endpoint, endpointType)

	return &providers.DiscoveredConnection{
		ID:           p.connectionID("alloydb-" + instanceName),
		ProviderType: providers.ProviderTypeGCP,
		ProviderID:   p.config.ID,
		Name:         clusterName + "/" + instanceName + suffix,
		DatabaseType: engine.DatabaseType_Postgres,
		Region:       p.config.Region,
		Status:       mapAlloyDBStatus(instance.State),
		Metadata:     metadata,
	}
}

func mapAlloyDBStatus(state alloydbpb.Instance_State) providers.ConnectionStatus {
	switch state {
	case alloydbpb.Instance_READY:
		return providers.ConnectionStatusAvailable
	case alloydbpb.Instance_CREATING, alloydbpb.Instance_PROMOTING, alloydbpb.Instance_BOOTSTRAPPING:
		return providers.ConnectionStatusStarting
	case alloydbpb.Instance_STOPPED:
		return providers.ConnectionStatusStopped
	case alloydbpb.Instance_DELETING:
		return providers.ConnectionStatusDeleting
	case alloydbpb.Instance_FAILED, alloydbpb.Instance_MAINTENANCE:
		return providers.ConnectionStatusFailed
	default:
		return providers.ConnectionStatusUnknown
	}
}

// extractResourceName returns the last component of a fully-qualified GCP resource name.
// e.g., "projects/myproject/locations/us-central1/clusters/mycluster" -> "mycluster"
func extractResourceName(fullName string) string {
	parts := strings.Split(fullName, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return fullName
}
