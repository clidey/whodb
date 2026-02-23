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

	for {
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

	metadata := make(map[string]string)

	if instance.Endpoint != nil {
		metadata["endpoint"] = aws.ToString(instance.Endpoint.Address)
		metadata["port"] = strconv.Itoa(int(aws.ToInt32(instance.Endpoint.Port)))
	}

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
	case "failed", "restore-error", "incompatible-credentials", "incompatible-parameters":
		return providers.ConnectionStatusFailed
	default:
		return providers.ConnectionStatusUnknown
	}
}
