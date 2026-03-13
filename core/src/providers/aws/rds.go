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
	"github.com/aws/aws-sdk-go-v2/service/rds"
	rdstypes "github.com/aws/aws-sdk-go-v2/service/rds/types"

	awsinfra "github.com/clidey/whodb/core/src/aws"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
	"github.com/clidey/whodb/core/src/providers"
)

func (p *Provider) discoverRDS(ctx context.Context) ([]providers.DiscoveredConnection, error) {
	var connections []providers.DiscoveredConnection
	var nextToken *string

	log.Debugf("RDS discoverRDS: starting discovery for provider %s", p.config.ID)

	for page := 0; page < maxPaginationPages; page++ {
		input := &rds.DescribeDBInstancesInput{
			Marker:     nextToken,
			MaxRecords: aws.Int32(50),
		}

		log.Debugf("RDS discoverRDS: calling DescribeDBInstances")
		output, err := p.rdsClient.DescribeDBInstances(ctx, input)
		if err != nil {
			log.Errorf("RDS discoverRDS: DescribeDBInstances failed: %v", err)
			return nil, awsinfra.HandleAWSError(err)
		}
		log.Debugf("RDS discoverRDS: DescribeDBInstances returned %d instances", len(output.DBInstances))

		for _, instance := range output.DBInstances {
			instanceID := aws.ToString(instance.DBInstanceIdentifier)
			engineName := aws.ToString(instance.Engine)
			log.Debugf("RDS discoverRDS: processing instance %s (engine=%s)", instanceID, engineName)

			conn := p.rdsInstanceToConnection(&instance)
			if conn != nil {
				log.Debugf("RDS discoverRDS: instance %s converted to connection (type=%s)", instanceID, conn.DatabaseType)
				connections = append(connections, *conn)
			} else if isEngineHandledByDedicatedDiscovery(engineName) {
				log.Debugf("RDS discoverRDS: skipping instance %s (engine=%s handled by dedicated discovery)", instanceID, engineName)
			} else {
				log.Warnf("RDS discoverRDS: instance %s could not be converted (engine=%s not supported)", instanceID, engineName)
			}
		}

		if output.Marker == nil {
			break
		}
		nextToken = output.Marker
	}

	log.Debugf("RDS discoverRDS: completed, found %d connections", len(connections))
	return connections, nil
}

func (p *Provider) rdsInstanceToConnection(instance *rdstypes.DBInstance) *providers.DiscoveredConnection {
	if instance.Engine == nil || instance.DBInstanceIdentifier == nil {
		return nil
	}

	dbType, ok := mapRDSEngine(*instance.Engine)
	if !ok {
		return nil
	}

	if instance.Endpoint == nil {
		log.Warnf("RDS: instance %s has no endpoint, skipping", *instance.DBInstanceIdentifier)
		return nil
	}

	metadata := make(map[string]string)
	metadata["endpoint"] = aws.ToString(instance.Endpoint.Address)
	metadata["port"] = strconv.Itoa(int(aws.ToInt32(instance.Endpoint.Port)))

	if instance.DBName != nil {
		metadata["databaseName"] = *instance.DBName
	}

	if aws.ToBool(instance.IAMDatabaseAuthenticationEnabled) {
		metadata["iamAuthEnabled"] = "true"
	}

	return &providers.DiscoveredConnection{
		ID:           p.connectionID(*instance.DBInstanceIdentifier),
		ProviderType: providers.ProviderTypeAWS,
		ProviderID:   p.config.ID,
		Name:         *instance.DBInstanceIdentifier,
		DatabaseType: dbType,
		Region:       p.config.Region,
		Status:       mapRDSStatus(instance.DBInstanceStatus),
		Metadata:     metadata,
	}
}

// discoverRDSClusters discovers Aurora cluster-level endpoints (writer and reader).
func (p *Provider) discoverRDSClusters(ctx context.Context) ([]providers.DiscoveredConnection, error) {
	var connections []providers.DiscoveredConnection
	var nextToken *string

	log.Debugf("RDS discoverRDSClusters: starting for provider %s", p.config.ID)

	for page := 0; page < maxPaginationPages; page++ {
		input := &rds.DescribeDBClustersInput{
			Marker:     nextToken,
			MaxRecords: aws.Int32(50),
		}

		output, err := p.rdsClient.DescribeDBClusters(ctx, input)
		if err != nil {
			log.Errorf("RDS discoverRDSClusters: DescribeDBClusters failed: %v", err)
			return nil, awsinfra.HandleAWSError(err)
		}

		log.Debugf("RDS discoverRDSClusters: DescribeDBClusters returned %d clusters", len(output.DBClusters))

		for _, cluster := range output.DBClusters {
			if cluster.Engine == nil || cluster.DBClusterIdentifier == nil {
				continue
			}

			dbType, ok := mapRDSEngine(*cluster.Engine)
			if !ok {
				log.Debugf("RDS discoverRDSClusters: skipping cluster %s (engine=%s not supported)", aws.ToString(cluster.DBClusterIdentifier), aws.ToString(cluster.Engine))
				continue
			}

			clusterID := aws.ToString(cluster.DBClusterIdentifier)
			log.Debugf("RDS discoverRDSClusters: processing cluster %s (engine=%s)", clusterID, aws.ToString(cluster.Engine))

			// Writer endpoint
			if cluster.Endpoint != nil {
				port := int32(0)
				if cluster.Port != nil {
					port = *cluster.Port
				}
				metadata := map[string]string{
					"endpoint":          *cluster.Endpoint,
					"port":              strconv.Itoa(int(port)),
					"endpointType":      "writer",
					"clusterIdentifier": clusterID,
				}
				if aws.ToBool(cluster.IAMDatabaseAuthenticationEnabled) {
					metadata["iamAuthEnabled"] = "true"
				}
				connections = append(connections, providers.DiscoveredConnection{
					ID:           p.connectionID(clusterID + "-writer"),
					ProviderType: providers.ProviderTypeAWS,
					ProviderID:   p.config.ID,
					Name:         clusterID + " (writer)",
					DatabaseType: dbType,
					Region:       p.config.Region,
					Status:       mapRDSStatus(cluster.Status),
					Metadata:     metadata,
				})
			}

			// Reader endpoint
			if cluster.ReaderEndpoint != nil {
				port := int32(0)
				if cluster.Port != nil {
					port = *cluster.Port
				}
				metadata := map[string]string{
					"endpoint":          *cluster.ReaderEndpoint,
					"port":              strconv.Itoa(int(port)),
					"endpointType":      "reader",
					"clusterIdentifier": clusterID,
				}
				if aws.ToBool(cluster.IAMDatabaseAuthenticationEnabled) {
					metadata["iamAuthEnabled"] = "true"
				}
				connections = append(connections, providers.DiscoveredConnection{
					ID:           p.connectionID(clusterID + "-reader"),
					ProviderType: providers.ProviderTypeAWS,
					ProviderID:   p.config.ID,
					Name:         clusterID + " (reader)",
					DatabaseType: dbType,
					Region:       p.config.Region,
					Status:       mapRDSStatus(cluster.Status),
					Metadata:     metadata,
				})
			}
		}

		if output.Marker == nil {
			break
		}
		nextToken = output.Marker
	}

	log.Debugf("RDS discoverRDSClusters: found %d cluster endpoints", len(connections))
	return connections, nil
}

// discoverRDSProxies discovers RDS Proxy endpoints.
func (p *Provider) discoverRDSProxies(ctx context.Context) ([]providers.DiscoveredConnection, error) {
	var connections []providers.DiscoveredConnection
	var nextToken *string

	log.Debugf("RDS discoverRDSProxies: starting for provider %s", p.config.ID)

	for page := 0; page < maxPaginationPages; page++ {
		input := &rds.DescribeDBProxiesInput{
			Marker:     nextToken,
			MaxRecords: aws.Int32(50),
		}

		output, err := p.rdsClient.DescribeDBProxies(ctx, input)
		if err != nil {
			log.Errorf("RDS discoverRDSProxies: DescribeDBProxies failed: %v", err)
			return nil, awsinfra.HandleAWSError(err)
		}

		log.Debugf("RDS discoverRDSProxies: DescribeDBProxies returned %d proxies", len(output.DBProxies))

		for _, proxy := range output.DBProxies {
			if proxy.DBProxyName == nil || proxy.Endpoint == nil {
				continue
			}

			dbType := mapProxyEngineFamily(aws.ToString(proxy.EngineFamily))
			if dbType == "" {
				log.Debugf("RDS discoverRDSProxies: skipping proxy %s (engine=%s not supported)", aws.ToString(proxy.DBProxyName), aws.ToString(proxy.EngineFamily))
				continue
			}
			log.Debugf("RDS discoverRDSProxies: processing proxy %s (engine=%s, status=%s)", aws.ToString(proxy.DBProxyName), aws.ToString(proxy.EngineFamily), proxy.Status)

			defaultPort := "3306"
			if dbType == engine.DatabaseType_Postgres {
				defaultPort = "5432"
			}

			metadata := map[string]string{
				"endpoint":     *proxy.Endpoint,
				"port":         defaultPort,
				"endpointType": "proxy",
				"proxyName":    *proxy.DBProxyName,
			}
			if aws.ToBool(proxy.RequireTLS) {
				metadata["requireTLS"] = "true"
			}

			connections = append(connections, providers.DiscoveredConnection{
				ID:           p.connectionID("proxy-" + *proxy.DBProxyName),
				ProviderType: providers.ProviderTypeAWS,
				ProviderID:   p.config.ID,
				Name:         *proxy.DBProxyName + " (proxy)",
				DatabaseType: dbType,
				Region:       p.config.Region,
				Status:       mapProxyStatus(string(proxy.Status)),
				Metadata:     metadata,
			})
		}

		if output.Marker == nil {
			break
		}
		nextToken = output.Marker
	}

	log.Debugf("RDS discoverRDSProxies: found %d proxies", len(connections))
	return connections, nil
}

var mapRDSEngineExtension func(string) (engine.DatabaseType, bool)

// SetRDSEngineMapper registers an extension function for mapping RDS engine names
// to database types. This allows EE to add support for additional engines without modifying CE code
func SetRDSEngineMapper(fn func(string) (engine.DatabaseType, bool)) {
	mapRDSEngineExtension = fn
}

// isEngineHandledByDedicatedDiscovery returns true for engines that have their own
// dedicated AWS discovery mechanism and should be silently skipped in RDS discovery.
func isEngineHandledByDedicatedDiscovery(engineName string) bool {
	engineName = strings.ToLower(engineName)
	switch {
	case strings.HasPrefix(engineName, "docdb"):
		// DocumentDB clusters are discovered via discoverDocumentDB()
		return true
	case strings.HasPrefix(engineName, "neptune"):
		// Neptune is a graph database with its own API (not currently supported)
		return true
	default:
		return false
	}
}

func mapRDSEngine(engineName string) (engine.DatabaseType, bool) {
	engineName = strings.ToLower(engineName)

	switch {
	case engineName == "mysql" || strings.HasPrefix(engineName, "mysql-"):
		return engine.DatabaseType_MySQL, true
	case engineName == "mariadb" || strings.HasPrefix(engineName, "mariadb-"):
		return engine.DatabaseType_MariaDB, true
	case engineName == "postgres" || engineName == "postgresql" ||
		strings.HasPrefix(engineName, "postgres-") || strings.HasPrefix(engineName, "postgresql-"):
		return engine.DatabaseType_Postgres, true
	case strings.HasPrefix(engineName, "aurora-mysql"):
		return engine.DatabaseType_MySQL, true
	case strings.HasPrefix(engineName, "aurora-postgresql"):
		return engine.DatabaseType_Postgres, true
	}

	if mapRDSEngineExtension != nil {
		return mapRDSEngineExtension(engineName)
	}

	return "", false
}

// mapProxyEngineFamily maps RDS Proxy EngineFamily to DatabaseType.
func mapProxyEngineFamily(family string) engine.DatabaseType {
	switch strings.ToUpper(family) {
	case "MYSQL":
		return engine.DatabaseType_MySQL
	case "POSTGRESQL":
		return engine.DatabaseType_Postgres
	default:
		if mapRDSEngineExtension != nil {
			if dbType, ok := mapRDSEngineExtension(strings.ToLower(family)); ok {
				return dbType
			}
		}
		return ""
	}
}

func mapProxyStatus(status string) providers.ConnectionStatus {
	switch strings.ToLower(status) {
	case "available":
		return providers.ConnectionStatusAvailable
	case "creating", "modifying":
		return providers.ConnectionStatusStarting
	case "deleting":
		return providers.ConnectionStatusDeleting
	case "incompatible-network", "insufficient-resource-limits":
		return providers.ConnectionStatusFailed
	default:
		return providers.ConnectionStatusUnknown
	}
}

func mapRDSStatus(status *string) providers.ConnectionStatus {
	if status == nil {
		return providers.ConnectionStatusUnknown
	}

	switch strings.ToLower(*status) {
	case "available":
		return providers.ConnectionStatusAvailable
	case "starting", "creating", "configuring-enhanced-monitoring", "modifying", "upgrading":
		return providers.ConnectionStatusStarting
	case "stopped", "stopping", "storage-optimization":
		return providers.ConnectionStatusStopped
	case "deleting":
		return providers.ConnectionStatusDeleting
	case "failed", "restore-error", "incompatible-credentials", "incompatible-parameters", "incompatible-options":
		return providers.ConnectionStatusFailed
	default:
		return providers.ConnectionStatusUnknown
	}
}
