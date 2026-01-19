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

package aws

import (
	"context"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/docdb"
	docdbTypes "github.com/aws/aws-sdk-go-v2/service/docdb/types"

	awsinfra "github.com/clidey/whodb/core/src/aws"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/providers"
)

func (p *Provider) discoverDocumentDB(ctx context.Context) ([]providers.DiscoveredConnection, error) {
	var connections []providers.DiscoveredConnection
	var nextToken *string

	for {
		input := &docdb.DescribeDBClustersInput{
			Marker: nextToken,
		}

		output, err := p.docdbClient.DescribeDBClusters(ctx, input)
		if err != nil {
			return nil, awsinfra.HandleAWSError(err)
		}

		for _, cluster := range output.DBClusters {
			if cluster.Engine != nil && !strings.Contains(strings.ToLower(*cluster.Engine), "docdb") {
				continue
			}

			conn := p.docdbClusterToConnection(&cluster)
			if conn != nil {
				connections = append(connections, *conn)
			}
		}

		if output.Marker == nil {
			break
		}
		nextToken = output.Marker
	}

	return connections, nil
}

func (p *Provider) docdbClusterToConnection(cluster *docdbTypes.DBCluster) *providers.DiscoveredConnection {
	if cluster.DBClusterIdentifier == nil {
		return nil
	}

	metadata := make(map[string]string)

	if cluster.Endpoint != nil {
		metadata["endpoint"] = *cluster.Endpoint
	}
	if cluster.Port != nil {
		metadata["port"] = strconv.Itoa(int(*cluster.Port))
	}

	return &providers.DiscoveredConnection{
		ID:           p.connectionID(*cluster.DBClusterIdentifier),
		ProviderType: providers.ProviderTypeAWS,
		ProviderID:   p.config.ID,
		Name:         *cluster.DBClusterIdentifier,
		DatabaseType: engine.DatabaseType_DocumentDB,
		Region:       p.config.Region,
		Status:       mapDocDBStatus(cluster.Status),
		Metadata:     metadata,
	}
}

func mapDocDBStatus(status *string) providers.ConnectionStatus {
	if status == nil {
		return providers.ConnectionStatusUnknown
	}

	switch strings.ToLower(*status) {
	case "available":
		return providers.ConnectionStatusAvailable
	case "creating", "modifying", "upgrading", "migrating", "preparing-data-migration":
		return providers.ConnectionStatusStarting
	case "stopped", "stopping", "starting":
		return providers.ConnectionStatusStopped
	case "deleting":
		return providers.ConnectionStatusDeleting
	case "failed", "inaccessible-encryption-credentials":
		return providers.ConnectionStatusFailed
	default:
		return providers.ConnectionStatusUnknown
	}
}
