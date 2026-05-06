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

package graph

import (
	"context"
	"testing"
	"time"

	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/src/auth"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/providers"
	"github.com/clidey/whodb/core/src/settings"
	"github.com/clidey/whodb/core/src/source"
)

func testSourceContext(sourceType string, values map[string]string) context.Context {
	return context.WithValue(context.Background(), auth.AuthKey_Source, &source.Credentials{
		SourceType: sourceType,
		Values:     cloneStringMap(values),
	})
}

func testSourceRef(kind model.SourceObjectKind, path ...string) model.SourceObjectRefInput {
	return model.SourceObjectRefInput{
		Kind: kind,
		Path: path,
	}
}

func testSourceRefPtr(kind model.SourceObjectKind, path ...string) *model.SourceObjectRefInput {
	ref := testSourceRef(kind, path...)
	return &ref
}

func cloneStringMap(values map[string]string) map[string]string {
	if values == nil {
		return map[string]string{}
	}

	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func TestMapColumnsToModelPreservesMetadata(t *testing.T) {
	refTable := "users"
	refColumn := "id"
	length := 255
	precision := 10
	scale := 2
	columns := []engine.Column{
		{
			Name:             "id",
			Type:             "INTEGER",
			IsPrimary:        true,
			Length:           &length,
			Precision:        &precision,
			Scale:            &scale,
			IsForeignKey:     true,
			ReferencedTable:  &refTable,
			ReferencedColumn: &refColumn,
		},
	}

	mapped := MapColumnsToModel(columns)
	if len(mapped) != 1 {
		t.Fatalf("expected one mapped column, got %#v", mapped)
	}
	if mapped[0].Name != "id" || !mapped[0].IsPrimary || !mapped[0].IsForeignKey {
		t.Fatalf("expected mapped column metadata to be preserved, got %#v", mapped[0])
	}
	if mapped[0].ReferencedTable == nil || *mapped[0].ReferencedTable != "users" {
		t.Fatalf("expected referenced table metadata to be preserved, got %#v", mapped[0])
	}
}

func TestStateToProviderModelsMapOptionalFields(t *testing.T) {
	discoveryTime := time.Date(2026, time.April, 16, 10, 30, 0, 0, time.UTC)

	awsModel := stateToAWSProvider(&settings.AWSProviderState{
		Config: &settings.AWSProviderConfig{
			ID:                  "aws-1",
			Name:                "AWS Prod",
			Region:              "us-east-1",
			ProfileName:         "default",
			DiscoverRDS:         true,
			DiscoverElastiCache: true,
			DiscoverDocumentDB:  true,
			DiscoverS3:          true,
		},
		Status:          "Connected",
		LastDiscoveryAt: &discoveryTime,
		DiscoveredCount: 3,
		Error:           "warning",
	})
	if awsModel.ProfileName == nil || *awsModel.ProfileName != "default" {
		t.Fatalf("expected AWS profile name to be mapped, got %#v", awsModel)
	}
	if awsModel.LastDiscoveryAt == nil || *awsModel.LastDiscoveryAt != "2026-04-16T10:30:00Z" {
		t.Fatalf("expected AWS last discovery timestamp, got %#v", awsModel.LastDiscoveryAt)
	}
	if awsModel.Status != "Connected" || awsModel.Error == nil || *awsModel.Error != "warning" {
		t.Fatalf("expected AWS status and error to be mapped, got %#v", awsModel)
	}
	if !awsModel.DiscoverS3 {
		t.Fatalf("expected AWS S3 discovery flag to be mapped, got %#v", awsModel)
	}

	azureModel := stateToAzureProvider(&settings.AzureProviderState{
		Config: &settings.AzureProviderConfig{
			ID:                 "azure-1",
			Name:               "Azure Prod",
			SubscriptionID:     "sub-123",
			TenantID:           "tenant-123",
			ResourceGroup:      "rg-prod",
			DiscoverPostgreSQL: true,
			DiscoverMySQL:      true,
			DiscoverRedis:      true,
			DiscoverCosmosDB:   true,
		},
		Status:          "Discovering",
		LastDiscoveryAt: &discoveryTime,
		DiscoveredCount: 4,
	})
	if azureModel.TenantID == nil || *azureModel.TenantID != "tenant-123" {
		t.Fatalf("expected Azure tenant ID to be mapped, got %#v", azureModel)
	}
	if azureModel.ResourceGroup == nil || *azureModel.ResourceGroup != "rg-prod" {
		t.Fatalf("expected Azure resource group to be mapped, got %#v", azureModel)
	}
	if azureModel.SubscriptionID != "sub-123" || azureModel.Region != "sub-123" {
		t.Fatalf("expected Azure subscription data to be mapped, got %#v", azureModel)
	}
	if azureModel.Status != "Discovering" {
		t.Fatalf("expected Azure status to be mapped, got %#v", azureModel.Status)
	}

	gcpModel := stateToGCPProvider(&settings.GCPProviderState{
		Config: &settings.GCPProviderConfig{
			ID:                    "gcp-1",
			Name:                  "GCP Prod",
			ProjectID:             "project-123",
			Region:                "europe-west1",
			ServiceAccountKeyPath: "/tmp/key.json",
			DiscoverCloudSQL:      true,
			DiscoverAlloyDB:       true,
			DiscoverMemorystore:   true,
		},
		Status:          "Error",
		LastDiscoveryAt: &discoveryTime,
		DiscoveredCount: 2,
		Error:           "bad credentials",
	})
	if gcpModel.ServiceAccountKeyPath == nil || *gcpModel.ServiceAccountKeyPath != "/tmp/key.json" {
		t.Fatalf("expected GCP service account path to be mapped, got %#v", gcpModel)
	}
	if gcpModel.ProjectID != "project-123" || gcpModel.Region != "europe-west1" {
		t.Fatalf("expected GCP project and region to be mapped, got %#v", gcpModel)
	}
	if gcpModel.Status != "Error" || gcpModel.Error == nil || *gcpModel.Error != "bad credentials" {
		t.Fatalf("expected GCP status and error to be mapped, got %#v", gcpModel)
	}
}

func TestDiscoveredConnectionToModelFiltersMetadata(t *testing.T) {
	modelConn := discoveredConnectionToModel(&providers.DiscoveredConnection{
		ID:           "aws-1/prod-db",
		ProviderType: providers.ProviderTypeAWS,
		ProviderID:   "aws-1",
		Name:         "prod-db",
		DatabaseType: engine.DatabaseType_Postgres,
		Region:       "us-east-1",
		Status:       providers.ConnectionStatusAvailable,
		Metadata: map[string]string{
			"endpoint":     "db.internal",
			"port":         "5432",
			"databaseName": "app",
			"service":      "rds",
			"bucket":       "reports",
			"authMethod":   "profile",
			"profileName":  "dev",
			"secret":       "should-not-leak",
		},
	})

	if modelConn.ProviderType != "AWS" || modelConn.Status != "Available" {
		t.Fatalf("expected provider and status enums to be mapped, got %#v", modelConn)
	}
	if modelConn.Region == nil || *modelConn.Region != "us-east-1" {
		t.Fatalf("expected region to be mapped, got %#v", modelConn.Region)
	}

	metadata := map[string]string{}
	for _, record := range modelConn.Metadata {
		metadata[record.Key] = record.Value
	}
	if metadata["endpoint"] != "db.internal" ||
		metadata["port"] != "5432" ||
		metadata["databaseName"] != "app" ||
		metadata["service"] != "rds" ||
		metadata["bucket"] != "reports" ||
		metadata["authMethod"] != "profile" ||
		metadata["profileName"] != "dev" {
		t.Fatalf("expected allowed metadata to be preserved, got %#v", metadata)
	}
	if _, found := metadata["secret"]; found {
		t.Fatalf("expected secret metadata to be filtered out, got %#v", metadata)
	}
}

func TestResolverHelperFallbacks(t *testing.T) {
	if got := mapCloudProviderStatus("Connected"); got != "Connected" {
		t.Fatalf("expected connected provider status, got %q", got)
	}
	if got := mapCloudProviderStatus("mystery"); got != "Disconnected" {
		t.Fatalf("expected unknown provider status to fall back to Disconnected, got %q", got)
	}
	if got := mapProviderTypeToModel(providers.ProviderTypeAzure); got != "Azure" {
		t.Fatalf("expected Azure provider type, got %q", got)
	}
	if got := mapProviderTypeToModel(providers.ProviderType("custom")); got != "AWS" {
		t.Fatalf("expected unknown provider type to fall back to AWS, got %q", got)
	}
	if got := mapConnectionStatusToModel(providers.ConnectionStatusStopped); got != "Stopped" {
		t.Fatalf("expected Stopped connection status, got %q", got)
	}
	if got := mapConnectionStatusToModel(providers.ConnectionStatus("custom")); got != "Unknown" {
		t.Fatalf("expected unknown connection status to fall back to Unknown, got %q", got)
	}

	if got := derefStringOr(nil, "fallback"); got != "fallback" {
		t.Fatalf("expected string fallback, got %q", got)
	}
	if got := derefStringOr(stringPtr("value"), "fallback"); got != "value" {
		t.Fatalf("expected string pointer value, got %q", got)
	}
	if got := derefBoolOr(nil, true); !got {
		t.Fatalf("expected bool fallback true, got %t", got)
	}
	if got := derefBoolOr(boolPtr(false), true); got {
		t.Fatalf("expected bool pointer value false, got %t", got)
	}
}

func boolPtr(value bool) *bool { return &value }
