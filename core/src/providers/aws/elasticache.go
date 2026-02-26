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

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/elasticache"
	ectypes "github.com/aws/aws-sdk-go-v2/service/elasticache/types"

	awsinfra "github.com/clidey/whodb/core/src/aws"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
	"github.com/clidey/whodb/core/src/providers"
)

func (p *Provider) discoverElastiCache(ctx context.Context) ([]providers.DiscoveredConnection, error) {
	var connections []providers.DiscoveredConnection

	log.Debugf("ElastiCache: starting discovery for provider %s", p.config.ID)

	// Discover Serverless caches
	serverless, err := p.discoverElastiCacheServerless(ctx)
	if err != nil {
		return nil, err
	}
	log.Debugf("ElastiCache: found %d serverless caches", len(serverless))
	connections = append(connections, serverless...)

	// Discover traditional replication groups
	replicationGroups, err := p.discoverElastiCacheReplicationGroups(ctx)
	if err != nil {
		return nil, err
	}
	log.Debugf("ElastiCache: found %d replication groups", len(replicationGroups))
	connections = append(connections, replicationGroups...)

	// Discover standalone clusters (not in replication groups)
	clusters, err := p.discoverElastiCacheClusters(ctx)
	if err != nil {
		return nil, err
	}
	log.Debugf("ElastiCache: found %d standalone clusters", len(clusters))
	connections = append(connections, clusters...)

	return connections, nil
}

func (p *Provider) discoverElastiCacheServerless(ctx context.Context) ([]providers.DiscoveredConnection, error) {
	var connections []providers.DiscoveredConnection
	var nextToken *string

	log.Debugf("ElastiCache: calling DescribeServerlessCaches for provider %s", p.config.ID)

	for {
		input := &elasticache.DescribeServerlessCachesInput{
			NextToken:  nextToken,
			MaxResults: aws.Int32(50),
		}

		output, err := p.elasticacheClient.DescribeServerlessCaches(ctx, input)
		if err != nil {
			log.Errorf("ElastiCache: DescribeServerlessCaches failed: %v", err)
			return nil, awsinfra.HandleAWSError(err)
		}

		log.Debugf("ElastiCache: DescribeServerlessCaches returned %d caches", len(output.ServerlessCaches))

		for _, cache := range output.ServerlessCaches {
			cacheName := aws.ToString(cache.ServerlessCacheName)
			cacheEngine := aws.ToString(cache.Engine)
			status := aws.ToString(cache.Status)
			log.Debugf("ElastiCache: processing serverless cache %s (engine=%s, status=%s)", cacheName, cacheEngine, status)

			if !isRedisCompatibleEngine(cacheEngine) {
				log.Debugf("ElastiCache: skipping serverless cache %s (engine=%s, only Redis/Valkey supported)", cacheName, cacheEngine)
				continue
			}

			conn := p.serverlessCacheToConnection(&cache)
			if conn != nil {
				log.Debugf("ElastiCache: serverless cache %s converted to connection (endpoint=%s)", cacheName, conn.Metadata["endpoint"])
				connections = append(connections, *conn)
			} else {
				log.Warnf("ElastiCache: serverless cache %s returned nil connection", cacheName)
			}
		}

		if output.NextToken == nil {
			break
		}
		nextToken = output.NextToken
	}

	return connections, nil
}

func (p *Provider) discoverElastiCacheReplicationGroups(ctx context.Context) ([]providers.DiscoveredConnection, error) {
	var connections []providers.DiscoveredConnection
	var nextToken *string

	log.Debugf("ElastiCache: calling DescribeReplicationGroups for provider %s", p.config.ID)

	for {
		input := &elasticache.DescribeReplicationGroupsInput{
			Marker:     nextToken,
			MaxRecords: aws.Int32(50),
		}

		output, err := p.elasticacheClient.DescribeReplicationGroups(ctx, input)
		if err != nil {
			log.Errorf("ElastiCache: DescribeReplicationGroups failed: %v", err)
			return nil, awsinfra.HandleAWSError(err)
		}

		log.Debugf("ElastiCache: DescribeReplicationGroups returned %d groups", len(output.ReplicationGroups))

		for _, rg := range output.ReplicationGroups {
			rgID := aws.ToString(rg.ReplicationGroupId)
			status := aws.ToString(rg.Status)
			log.Debugf("ElastiCache: processing replication group %s (status=%s)", rgID, status)

			conn := p.replicationGroupToConnection(&rg)
			if conn != nil {
				log.Debugf("ElastiCache: replication group %s converted to connection (endpoint=%s)", rgID, conn.Metadata["endpoint"])
				connections = append(connections, *conn)
			} else {
				log.Warnf("ElastiCache: replication group %s returned nil connection", rgID)
			}
		}

		if output.Marker == nil {
			break
		}
		nextToken = output.Marker
	}

	return connections, nil
}

func (p *Provider) discoverElastiCacheClusters(ctx context.Context) ([]providers.DiscoveredConnection, error) {
	var connections []providers.DiscoveredConnection
	var nextToken *string

	log.Debugf("ElastiCache: calling DescribeCacheClusters for provider %s", p.config.ID)

	for {
		input := &elasticache.DescribeCacheClustersInput{
			Marker:            nextToken,
			MaxRecords:        aws.Int32(50),
			ShowCacheNodeInfo: aws.Bool(true),
		}

		output, err := p.elasticacheClient.DescribeCacheClusters(ctx, input)
		if err != nil {
			log.Errorf("ElastiCache: DescribeCacheClusters failed: %v", err)
			return nil, awsinfra.HandleAWSError(err)
		}

		log.Debugf("ElastiCache: DescribeCacheClusters returned %d clusters", len(output.CacheClusters))

		for _, cluster := range output.CacheClusters {
			clusterID := aws.ToString(cluster.CacheClusterId)
			engineName := aws.ToString(cluster.Engine)
			replicationGroupID := aws.ToString(cluster.ReplicationGroupId)
			log.Debugf("ElastiCache: processing cluster %s (engine=%s, replicationGroupId=%s)", clusterID, engineName, replicationGroupID)

			if cluster.ReplicationGroupId != nil {
				log.Debugf("ElastiCache: skipping cluster %s (belongs to replication group %s)", clusterID, *cluster.ReplicationGroupId)
				continue
			}

			if cluster.Engine != nil && !isRedisCompatibleEngine(*cluster.Engine) {
				log.Debugf("ElastiCache: skipping cluster %s (engine=%s, only Redis/Valkey supported)", clusterID, engineName)
				continue
			}

			conn := p.cacheClusterToConnection(&cluster)
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

func (p *Provider) replicationGroupToConnection(rg *ectypes.ReplicationGroup) *providers.DiscoveredConnection {
	if rg.ReplicationGroupId == nil {
		return nil
	}

	metadata := make(map[string]string)
	metadata["transitEncryption"] = strconv.FormatBool(aws.ToBool(rg.TransitEncryptionEnabled))
	metadata["authTokenEnabled"] = strconv.FormatBool(aws.ToBool(rg.AuthTokenEnabled))

	var endpoint string
	var port int32
	if rg.ConfigurationEndpoint != nil {
		endpoint = aws.ToString(rg.ConfigurationEndpoint.Address)
		port = aws.ToInt32(rg.ConfigurationEndpoint.Port)
	} else if len(rg.NodeGroups) > 0 && rg.NodeGroups[0].PrimaryEndpoint != nil {
		endpoint = aws.ToString(rg.NodeGroups[0].PrimaryEndpoint.Address)
		port = aws.ToInt32(rg.NodeGroups[0].PrimaryEndpoint.Port)
	}

	if endpoint != "" {
		metadata["endpoint"] = endpoint
		metadata["port"] = strconv.Itoa(int(port))
	}

	return &providers.DiscoveredConnection{
		ID:           p.connectionID(*rg.ReplicationGroupId),
		ProviderType: providers.ProviderTypeAWS,
		ProviderID:   p.config.ID,
		Name:         *rg.ReplicationGroupId,
		DatabaseType: engine.DatabaseType_ElastiCache,
		Region:       p.config.Region,
		Status:       mapElastiCacheStatus(aws.ToString(rg.Status)),
		Metadata:     metadata,
	}
}

func (p *Provider) cacheClusterToConnection(cluster *ectypes.CacheCluster) *providers.DiscoveredConnection {
	if cluster.CacheClusterId == nil {
		return nil
	}

	metadata := make(map[string]string)
	metadata["transitEncryption"] = strconv.FormatBool(aws.ToBool(cluster.TransitEncryptionEnabled))
	metadata["authTokenEnabled"] = strconv.FormatBool(aws.ToBool(cluster.AuthTokenEnabled))

	if len(cluster.CacheNodes) > 0 && cluster.CacheNodes[0].Endpoint != nil {
		metadata["endpoint"] = aws.ToString(cluster.CacheNodes[0].Endpoint.Address)
		metadata["port"] = strconv.Itoa(int(aws.ToInt32(cluster.CacheNodes[0].Endpoint.Port)))
	}

	return &providers.DiscoveredConnection{
		ID:           p.connectionID(*cluster.CacheClusterId),
		ProviderType: providers.ProviderTypeAWS,
		ProviderID:   p.config.ID,
		Name:         *cluster.CacheClusterId,
		DatabaseType: engine.DatabaseType_ElastiCache,
		Region:       p.config.Region,
		Status:       mapElastiCacheStatus(aws.ToString(cluster.CacheClusterStatus)),
		Metadata:     metadata,
	}
}

func (p *Provider) serverlessCacheToConnection(cache *ectypes.ServerlessCache) *providers.DiscoveredConnection {
	if cache.ServerlessCacheName == nil {
		return nil
	}

	metadata := make(map[string]string)
	metadata["serverless"] = "true"

	// Serverless caches always have TLS enabled
	metadata["transitEncryption"] = "true"

	if cache.Endpoint != nil {
		metadata["endpoint"] = aws.ToString(cache.Endpoint.Address)
		metadata["port"] = strconv.Itoa(int(aws.ToInt32(cache.Endpoint.Port)))
	}

	return &providers.DiscoveredConnection{
		ID:           p.connectionID(*cache.ServerlessCacheName),
		ProviderType: providers.ProviderTypeAWS,
		ProviderID:   p.config.ID,
		Name:         *cache.ServerlessCacheName,
		DatabaseType: engine.DatabaseType_ElastiCache,
		Region:       p.config.Region,
		Status:       mapServerlessCacheStatus(aws.ToString(cache.Status)),
		Metadata:     metadata,
	}
}

func mapServerlessCacheStatus(status string) providers.ConnectionStatus {
	switch strings.ToLower(status) {
	case "available":
		return providers.ConnectionStatusAvailable
	case "creating", "modifying":
		return providers.ConnectionStatusStarting
	case "deleting":
		return providers.ConnectionStatusDeleting
	case "create-failed":
		return providers.ConnectionStatusFailed
	default:
		return providers.ConnectionStatusUnknown
	}
}

// isRedisCompatibleEngine returns true for Redis-compatible engines (redis, valkey).
func isRedisCompatibleEngine(engine string) bool {
	engine = strings.ToLower(engine)
	return engine == "redis" || engine == "valkey"
}

func mapElastiCacheStatus(status string) providers.ConnectionStatus {
	switch strings.ToLower(status) {
	case "available":
		return providers.ConnectionStatusAvailable
	case "creating", "modifying", "snapshotting", "rebooting cluster nodes":
		return providers.ConnectionStatusStarting
	case "deleted", "deleting":
		return providers.ConnectionStatusDeleting
	case "create-failed", "restore-failed":
		return providers.ConnectionStatusFailed
	default:
		return providers.ConnectionStatusUnknown
	}
}
