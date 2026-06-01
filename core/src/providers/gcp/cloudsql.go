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
	"strconv"
	"strings"
	"time"

	sqladmin "google.golang.org/api/sqladmin/v1beta4"

	"github.com/clidey/whodb/core/src/engine"
	gcpinfra "github.com/clidey/whodb/core/src/gcp"
	"github.com/clidey/whodb/core/src/log"
	"github.com/clidey/whodb/core/src/providers"
)

// discoverCloudSQL discovers Cloud SQL instances in the configured project.
func (p *Provider) discoverCloudSQL(ctx context.Context) ([]providers.DiscoveredConnection, error) {
	start := time.Now()
	var connections []providers.DiscoveredConnection
	var nextToken string

	log.Debugf("Cloud SQL: starting discovery for provider %s (project=%s)", p.config.ID, p.config.ProjectID)

	for page := range maxPaginationPages {
		if ctx.Err() != nil {
			log.Warnf("Cloud SQL: context canceled, returning %d results so far", len(connections))
			return connections, ctx.Err()
		}

		call := p.sqladminService.Instances.List(p.config.ProjectID).Context(ctx)
		if nextToken != "" {
			call = call.PageToken(nextToken)
		}

		output, err := call.Do()
		if err != nil {
			log.Errorf("Cloud SQL: Instances.List failed: %v", err)
			return nil, gcpinfra.HandleGCPError(err)
		}

		log.Debugf("Cloud SQL: Instances.List returned %d instances", len(output.Items))

		for _, instance := range output.Items {
			// Filter by region if the instance is in a different region
			if instance.Region != "" && instance.Region != p.config.Region {
				log.Debugf("Cloud SQL: skipping instance %s (region=%s, want=%s)", instance.Name, instance.Region, p.config.Region)
				continue
			}

			conn := p.cloudSQLInstanceToConnection(instance)
			if conn != nil {
				log.Debugf("Cloud SQL: instance %s converted to connection (type=%s)", instance.Name, conn.DatabaseType)
				connections = append(connections, *conn)
			} else {
				log.Debugf("Cloud SQL: instance %s could not be converted (version=%s)", instance.Name, instance.DatabaseVersion)
			}
		}

		if output.NextPageToken == "" {
			break
		}
		nextToken = output.NextPageToken

		if page > 0 && page%10 == 0 {
			log.Debugf("Cloud SQL: processed %d pages, %d instances so far", page, len(connections))
		}
		if page == maxPaginationPages-1 {
			log.Warnf("Cloud SQL: hit pagination limit of %d pages, results may be incomplete", maxPaginationPages)
		}
	}

	log.Infof("Cloud SQL: found %d connections in %v", len(connections), time.Since(start))
	return connections, nil
}

func (p *Provider) cloudSQLInstanceToConnection(instance *sqladmin.DatabaseInstance) *providers.DiscoveredConnection {
	if instance.Name == "" || instance.DatabaseVersion == "" {
		return nil
	}

	dbType, ok := mapCloudSQLEngine(instance.DatabaseVersion)
	if !ok {
		return nil
	}

	// Find the primary IP address
	var endpoint string
	for _, addr := range instance.IpAddresses {
		if addr.Type == "PRIMARY" || (endpoint == "" && addr.IpAddress != "") {
			endpoint = addr.IpAddress
		}
	}

	if endpoint == "" {
		log.Warnf("Cloud SQL: instance %s has no IP address, skipping", instance.Name)
		return nil
	}

	port := defaultPortForEngine(instance.DatabaseVersion)

	metadata := map[string]string{
		"endpoint":        endpoint,
		"port":            strconv.Itoa(port),
		"connectionName":  instance.ConnectionName,
		"databaseVersion": instance.DatabaseVersion,
		"projectId":       p.config.ProjectID,
	}

	if instance.Settings != nil && instance.Settings.Tier != "" {
		metadata["tier"] = instance.Settings.Tier
	}

	if hasIAMAuth(instance) {
		metadata["iamAuthEnabled"] = "true"
	}

	return &providers.DiscoveredConnection{
		ID:           p.connectionID("cloudsql-" + instance.Name),
		ProviderType: providers.ProviderTypeGCP,
		ProviderID:   p.config.ID,
		Name:         instance.Name,
		DatabaseType: dbType,
		Region:       p.config.Region,
		Status:       mapCloudSQLStatus(instance.State),
		Metadata:     metadata,
	}
}

// hasIAMAuth checks if Cloud SQL IAM database authentication is enabled.
func hasIAMAuth(instance *sqladmin.DatabaseInstance) bool {
	if instance.Settings == nil {
		return false
	}
	for _, flag := range instance.Settings.DatabaseFlags {
		if flag.Name == "cloudsql.iam_authentication" && flag.Value == "on" {
			return true
		}
	}
	return false
}

// defaultPortForEngine returns the default port based on the Cloud SQL database version.
func defaultPortForEngine(databaseVersion string) int {
	v := strings.ToUpper(databaseVersion)
	switch {
	case strings.HasPrefix(v, "MYSQL"):
		return 3306
	case strings.HasPrefix(v, "POSTGRES"):
		return 5432
	case strings.HasPrefix(v, "SQLSERVER"):
		return 1433
	default:
		return 5432
	}
}

var mapCloudSQLEngineExtension func(string) (engine.DatabaseType, bool)

// SetCloudSQLEngineMapper registers an extension function for mapping Cloud SQL
// database versions to database types. This allows EE to add support for SQL Server.
func SetCloudSQLEngineMapper(fn func(string) (engine.DatabaseType, bool)) {
	mapCloudSQLEngineExtension = fn
}

func mapCloudSQLEngine(databaseVersion string) (engine.DatabaseType, bool) {
	v := strings.ToUpper(databaseVersion)

	switch {
	case strings.HasPrefix(v, "MYSQL"):
		return engine.DatabaseType_MySQL, true
	case strings.HasPrefix(v, "POSTGRES"):
		return engine.DatabaseType_Postgres, true
	}

	// Check extension mapper (EE can add SQLSERVER support)
	if mapCloudSQLEngineExtension != nil {
		return mapCloudSQLEngineExtension(databaseVersion)
	}

	return "", false
}

func mapCloudSQLStatus(state string) providers.ConnectionStatus {
	switch strings.ToUpper(state) {
	case "RUNNABLE":
		return providers.ConnectionStatusAvailable
	case "PENDING_CREATE", "MAINTENANCE":
		return providers.ConnectionStatusStarting
	case "SUSPENDED":
		return providers.ConnectionStatusStopped
	case "PENDING_DELETE":
		return providers.ConnectionStatusDeleting
	case "FAILED":
		return providers.ConnectionStatusFailed
	default:
		return providers.ConnectionStatusUnknown
	}
}
