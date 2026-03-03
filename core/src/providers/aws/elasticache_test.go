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
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	ectypes "github.com/aws/aws-sdk-go-v2/service/elasticache/types"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/providers"
)

func TestMapElastiCacheStatus(t *testing.T) {
	testCases := []struct {
		status   string
		expected providers.ConnectionStatus
	}{
		{"available", providers.ConnectionStatusAvailable},
		{"Available", providers.ConnectionStatusAvailable},
		{"AVAILABLE", providers.ConnectionStatusAvailable},
		{"creating", providers.ConnectionStatusStarting},
		{"modifying", providers.ConnectionStatusStarting},
		{"snapshotting", providers.ConnectionStatusStarting},
		{"rebooting cluster nodes", providers.ConnectionStatusStarting},
		{"deleted", providers.ConnectionStatusDeleting},
		{"deleting", providers.ConnectionStatusDeleting},
		{"create-failed", providers.ConnectionStatusFailed},
		{"restore-failed", providers.ConnectionStatusFailed},
		{"unknown-status", providers.ConnectionStatusUnknown},
		{"", providers.ConnectionStatusUnknown},
	}

	for _, tc := range testCases {
		result := mapElastiCacheStatus(tc.status)
		if result != tc.expected {
			t.Errorf("mapElastiCacheStatus(%s): expected %s, got %s", tc.status, tc.expected, result)
		}
	}
}

func TestMapElastiCacheStatus_CaseInsensitive(t *testing.T) {
	statuses := []string{"available", "Available", "AVAILABLE", "AvAiLaBlE"}
	for _, s := range statuses {
		result := mapElastiCacheStatus(s)
		if result != providers.ConnectionStatusAvailable {
			t.Errorf("mapElastiCacheStatus(%s): expected Available, got %s", s, result)
		}
	}
}

func TestMapServerlessCacheStatus(t *testing.T) {
	testCases := []struct {
		status   string
		expected providers.ConnectionStatus
	}{
		{"available", providers.ConnectionStatusAvailable},
		{"creating", providers.ConnectionStatusStarting},
		{"modifying", providers.ConnectionStatusStarting},
		{"deleting", providers.ConnectionStatusDeleting},
		{"create-failed", providers.ConnectionStatusFailed},
		{"update-failed", providers.ConnectionStatusFailed},
		{"delete-failed", providers.ConnectionStatusFailed},
		{"unknown", providers.ConnectionStatusUnknown},
	}

	for _, tc := range testCases {
		result := mapServerlessCacheStatus(tc.status)
		if result != tc.expected {
			t.Errorf("mapServerlessCacheStatus(%s): expected %s, got %s", tc.status, tc.expected, result)
		}
	}
}

func newTestElastiCacheProvider() *Provider {
	p, _ := New(&Config{
		ID:                  "test-ec",
		Name:                "Test ElastiCache",
		Region:              "us-west-2",
		DiscoverElastiCache: true,
	})
	return p
}

func TestReplicationGroupToConnection_HappyPath(t *testing.T) {
	p := newTestElastiCacheProvider()
	rg := &ectypes.ReplicationGroup{
		ReplicationGroupId:       aws.String("my-redis-rg"),
		Status:                   aws.String("available"),
		TransitEncryptionEnabled: aws.Bool(true),
		AuthTokenEnabled:         aws.Bool(false),
		ConfigurationEndpoint: &ectypes.Endpoint{
			Address: aws.String("my-redis-rg.abc.cache.amazonaws.com"),
			Port:    aws.Int32(6379),
		},
	}

	conn := p.replicationGroupToConnection(rg)
	if conn == nil {
		t.Fatal("expected non-nil connection")
	}
	if conn.DatabaseType != engine.DatabaseType_ElastiCache {
		t.Errorf("expected ElastiCache, got %s", conn.DatabaseType)
	}
	if conn.Metadata["endpoint"] != "my-redis-rg.abc.cache.amazonaws.com" {
		t.Errorf("unexpected endpoint: %s", conn.Metadata["endpoint"])
	}
	if conn.Metadata["port"] != "6379" {
		t.Errorf("unexpected port: %s", conn.Metadata["port"])
	}
}

func TestReplicationGroupToConnection_NilID(t *testing.T) {
	p := newTestElastiCacheProvider()
	rg := &ectypes.ReplicationGroup{
		ReplicationGroupId: nil,
	}

	conn := p.replicationGroupToConnection(rg)
	if conn != nil {
		t.Error("expected nil for nil ID")
	}
}

func TestReplicationGroupToConnection_NoEndpoint(t *testing.T) {
	p := newTestElastiCacheProvider()
	rg := &ectypes.ReplicationGroup{
		ReplicationGroupId:       aws.String("my-redis-rg"),
		Status:                   aws.String("creating"),
		TransitEncryptionEnabled: aws.Bool(false),
		AuthTokenEnabled:         aws.Bool(false),
		// No ConfigurationEndpoint, no NodeGroups
	}

	conn := p.replicationGroupToConnection(rg)
	if conn != nil {
		t.Error("expected nil for no endpoint")
	}
}

func TestReplicationGroupToConnection_NodeGroupEndpoint(t *testing.T) {
	p := newTestElastiCacheProvider()
	rg := &ectypes.ReplicationGroup{
		ReplicationGroupId:       aws.String("my-redis-rg"),
		Status:                   aws.String("available"),
		TransitEncryptionEnabled: aws.Bool(false),
		AuthTokenEnabled:         aws.Bool(false),
		NodeGroups: []ectypes.NodeGroup{
			{
				PrimaryEndpoint: &ectypes.Endpoint{
					Address: aws.String("primary.abc.cache.amazonaws.com"),
					Port:    aws.Int32(6379),
				},
			},
		},
	}

	conn := p.replicationGroupToConnection(rg)
	if conn == nil {
		t.Fatal("expected non-nil connection from NodeGroup endpoint")
	}
	if conn.Metadata["endpoint"] != "primary.abc.cache.amazonaws.com" {
		t.Errorf("unexpected endpoint: %s", conn.Metadata["endpoint"])
	}
}

func TestServerlessCacheToConnection_HappyPath(t *testing.T) {
	p := newTestElastiCacheProvider()
	cache := &ectypes.ServerlessCache{
		ServerlessCacheName: aws.String("my-serverless"),
		Status:              aws.String("available"),
		Endpoint: &ectypes.Endpoint{
			Address: aws.String("my-serverless.abc.cache.amazonaws.com"),
			Port:    aws.Int32(6379),
		},
	}

	conn := p.serverlessCacheToConnection(cache)
	if conn == nil {
		t.Fatal("expected non-nil connection")
	}
	if conn.Metadata["serverless"] != "true" {
		t.Error("expected serverless=true in metadata")
	}
	if conn.Metadata["transitEncryption"] != "true" {
		t.Error("expected transitEncryption=true for serverless")
	}
}

func TestServerlessCacheToConnection_NilName(t *testing.T) {
	p := newTestElastiCacheProvider()
	cache := &ectypes.ServerlessCache{
		ServerlessCacheName: nil,
	}

	conn := p.serverlessCacheToConnection(cache)
	if conn != nil {
		t.Error("expected nil for nil name")
	}
}

func TestServerlessCacheToConnection_NoEndpoint(t *testing.T) {
	p := newTestElastiCacheProvider()
	cache := &ectypes.ServerlessCache{
		ServerlessCacheName: aws.String("my-serverless"),
		Status:              aws.String("creating"),
		Endpoint:            nil,
	}

	conn := p.serverlessCacheToConnection(cache)
	if conn != nil {
		t.Error("expected nil for no endpoint")
	}
}

func TestCacheClusterToConnection_HappyPath(t *testing.T) {
	p := newTestElastiCacheProvider()
	cluster := &ectypes.CacheCluster{
		CacheClusterId:           aws.String("my-cluster"),
		CacheClusterStatus:      aws.String("available"),
		TransitEncryptionEnabled: aws.Bool(false),
		AuthTokenEnabled:         aws.Bool(false),
		CacheNodes: []ectypes.CacheNode{
			{
				Endpoint: &ectypes.Endpoint{
					Address: aws.String("my-cluster.abc.cache.amazonaws.com"),
					Port:    aws.Int32(6379),
				},
			},
		},
	}

	conn := p.cacheClusterToConnection(cluster)
	if conn == nil {
		t.Fatal("expected non-nil connection")
	}
	if conn.Metadata["endpoint"] != "my-cluster.abc.cache.amazonaws.com" {
		t.Errorf("unexpected endpoint: %s", conn.Metadata["endpoint"])
	}
}

func TestCacheClusterToConnection_NilID(t *testing.T) {
	p := newTestElastiCacheProvider()
	cluster := &ectypes.CacheCluster{
		CacheClusterId: nil,
	}

	conn := p.cacheClusterToConnection(cluster)
	if conn != nil {
		t.Error("expected nil for nil ID")
	}
}

func TestCacheClusterToConnection_NoNodes(t *testing.T) {
	p := newTestElastiCacheProvider()
	cluster := &ectypes.CacheCluster{
		CacheClusterId:           aws.String("my-cluster"),
		CacheClusterStatus:      aws.String("creating"),
		TransitEncryptionEnabled: aws.Bool(false),
		AuthTokenEnabled:         aws.Bool(false),
		CacheNodes:               []ectypes.CacheNode{},
	}

	conn := p.cacheClusterToConnection(cluster)
	if conn != nil {
		t.Error("expected nil for no cache nodes")
	}
}
