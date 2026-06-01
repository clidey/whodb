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
	"testing"

	sqladmin "google.golang.org/api/sqladmin/v1beta4"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/providers"
)

func TestMapCloudSQLStatus(t *testing.T) {
	testCases := []struct {
		state    string
		expected providers.ConnectionStatus
	}{
		{"RUNNABLE", providers.ConnectionStatusAvailable},
		{"Runnable", providers.ConnectionStatusAvailable},
		{"runnable", providers.ConnectionStatusAvailable},
		{"PENDING_CREATE", providers.ConnectionStatusStarting},
		{"pending_create", providers.ConnectionStatusStarting},
		{"MAINTENANCE", providers.ConnectionStatusStarting},
		{"maintenance", providers.ConnectionStatusStarting},
		{"SUSPENDED", providers.ConnectionStatusStopped},
		{"suspended", providers.ConnectionStatusStopped},
		{"PENDING_DELETE", providers.ConnectionStatusDeleting},
		{"pending_delete", providers.ConnectionStatusDeleting},
		{"FAILED", providers.ConnectionStatusFailed},
		{"failed", providers.ConnectionStatusFailed},
		{"UNKNOWN_STATE", providers.ConnectionStatusUnknown},
		{"", providers.ConnectionStatusUnknown},
	}

	for _, tc := range testCases {
		result := mapCloudSQLStatus(tc.state)
		if result != tc.expected {
			t.Errorf("mapCloudSQLStatus(%s): expected %s, got %s", tc.state, tc.expected, result)
		}
	}
}

func TestMapCloudSQLEngine(t *testing.T) {
	testCases := []struct {
		databaseVersion string
		expectedType    engine.DatabaseType
		expectedOK      bool
	}{
		{"MYSQL_8_0", engine.DatabaseType_MySQL, true},
		{"MYSQL_5_7", engine.DatabaseType_MySQL, true},
		{"POSTGRES_15", engine.DatabaseType_Postgres, true},
		{"POSTGRES_14", engine.DatabaseType_Postgres, true},
		{"SQLSERVER_2019_STANDARD", "", false},
		{"UNKNOWN_ENGINE", "", false},
		{"", "", false},
	}

	for _, tc := range testCases {
		dbType, ok := mapCloudSQLEngine(tc.databaseVersion)
		if ok != tc.expectedOK {
			t.Errorf("mapCloudSQLEngine(%s): expected ok=%t, got ok=%t", tc.databaseVersion, tc.expectedOK, ok)
		}
		if dbType != tc.expectedType {
			t.Errorf("mapCloudSQLEngine(%s): expected type %s, got %s", tc.databaseVersion, tc.expectedType, dbType)
		}
	}
}

func TestDefaultPortForEngine(t *testing.T) {
	testCases := []struct {
		databaseVersion string
		expectedPort    int
	}{
		{"MYSQL_8_0", 3306},
		{"MYSQL_5_7", 3306},
		{"mysql_8_0", 3306},
		{"POSTGRES_15", 5432},
		{"POSTGRES_14", 5432},
		{"postgres_14", 5432},
		{"SQLSERVER_2019_STANDARD", 1433},
		{"sqlserver_2019", 1433},
		{"UNKNOWN_ENGINE", 5432},
	}

	for _, tc := range testCases {
		result := defaultPortForEngine(tc.databaseVersion)
		if result != tc.expectedPort {
			t.Errorf("defaultPortForEngine(%s): expected %d, got %d", tc.databaseVersion, tc.expectedPort, result)
		}
	}
}

func TestHasIAMAuth(t *testing.T) {
	testCases := []struct {
		name     string
		instance *sqladmin.DatabaseInstance
		expected bool
	}{
		{
			name: "IAM auth enabled",
			instance: &sqladmin.DatabaseInstance{
				Settings: &sqladmin.Settings{
					DatabaseFlags: []*sqladmin.DatabaseFlags{
						{Name: "cloudsql.iam_authentication", Value: "on"},
					},
				},
			},
			expected: true,
		},
		{
			name: "IAM auth disabled",
			instance: &sqladmin.DatabaseInstance{
				Settings: &sqladmin.Settings{
					DatabaseFlags: []*sqladmin.DatabaseFlags{
						{Name: "cloudsql.iam_authentication", Value: "off"},
					},
				},
			},
			expected: false,
		},
		{
			name: "no IAM flag",
			instance: &sqladmin.DatabaseInstance{
				Settings: &sqladmin.Settings{
					DatabaseFlags: []*sqladmin.DatabaseFlags{
						{Name: "some_other_flag", Value: "on"},
					},
				},
			},
			expected: false,
		},
		{
			name: "nil settings",
			instance: &sqladmin.DatabaseInstance{
				Settings: nil,
			},
			expected: false,
		},
		{
			name: "no flags",
			instance: &sqladmin.DatabaseInstance{
				Settings: &sqladmin.Settings{},
			},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := hasIAMAuth(tc.instance)
			if result != tc.expected {
				t.Errorf("hasIAMAuth: expected %t, got %t", tc.expected, result)
			}
		})
	}
}

func newTestCloudSQLProvider() *Provider {
	p, _ := New(&Config{
		ID:               "test-cloudsql",
		Name:             "Test Cloud SQL",
		ProjectID:        "my-project-123",
		Region:           "us-central1",
		DiscoverCloudSQL: true,
	})
	return p
}

func TestCloudSQLInstanceToConnection_MySQL(t *testing.T) {
	p := newTestCloudSQLProvider()

	instance := &sqladmin.DatabaseInstance{
		Name:            "my-mysql",
		DatabaseVersion: "MYSQL_8_0",
		Region:          "us-central1",
		State:           "RUNNABLE",
		ConnectionName:  "my-project-123:us-central1:my-mysql",
		IpAddresses: []*sqladmin.IpMapping{
			{IpAddress: "10.0.0.1", Type: "PRIMARY"},
		},
		Settings: &sqladmin.Settings{
			Tier: "db-n1-standard-1",
		},
	}

	conn := p.cloudSQLInstanceToConnection(instance)

	if conn == nil {
		t.Fatal("expected non-nil connection")
	}
	if conn.DatabaseType != engine.DatabaseType_MySQL {
		t.Errorf("expected MySQL, got %s", conn.DatabaseType)
	}
	if conn.Name != "my-mysql" {
		t.Errorf("expected name my-mysql, got %s", conn.Name)
	}
	if conn.Region != "us-central1" {
		t.Errorf("expected region us-central1, got %s", conn.Region)
	}
	if conn.Status != providers.ConnectionStatusAvailable {
		t.Errorf("expected Available status, got %s", conn.Status)
	}
	if conn.Metadata["endpoint"] != "10.0.0.1" {
		t.Errorf("unexpected endpoint: %s", conn.Metadata["endpoint"])
	}
	if conn.Metadata["port"] != "3306" {
		t.Errorf("unexpected port: %s", conn.Metadata["port"])
	}
	if conn.Metadata["connectionName"] != "my-project-123:us-central1:my-mysql" {
		t.Errorf("unexpected connectionName: %s", conn.Metadata["connectionName"])
	}
	if conn.Metadata["databaseVersion"] != "MYSQL_8_0" {
		t.Errorf("unexpected databaseVersion: %s", conn.Metadata["databaseVersion"])
	}
	if conn.Metadata["tier"] != "db-n1-standard-1" {
		t.Errorf("unexpected tier: %s", conn.Metadata["tier"])
	}
	if conn.Metadata["projectId"] != "my-project-123" {
		t.Errorf("unexpected projectId: %s", conn.Metadata["projectId"])
	}
	if conn.ProviderType != providers.ProviderTypeGCP {
		t.Errorf("expected ProviderType GCP, got %s", conn.ProviderType)
	}
	if conn.ProviderID != "test-cloudsql" {
		t.Errorf("expected ProviderID test-cloudsql, got %s", conn.ProviderID)
	}
}

func TestCloudSQLInstanceToConnection_Postgres(t *testing.T) {
	p := newTestCloudSQLProvider()

	instance := &sqladmin.DatabaseInstance{
		Name:            "my-postgres",
		DatabaseVersion: "POSTGRES_15",
		Region:          "us-central1",
		State:           "RUNNABLE",
		ConnectionName:  "my-project-123:us-central1:my-postgres",
		IpAddresses: []*sqladmin.IpMapping{
			{IpAddress: "10.0.0.2", Type: "PRIMARY"},
		},
	}

	conn := p.cloudSQLInstanceToConnection(instance)

	if conn == nil {
		t.Fatal("expected non-nil connection")
	}
	if conn.DatabaseType != engine.DatabaseType_Postgres {
		t.Errorf("expected Postgres, got %s", conn.DatabaseType)
	}
	if conn.Metadata["port"] != "5432" {
		t.Errorf("unexpected port: %s", conn.Metadata["port"])
	}
}

func TestCloudSQLInstanceToConnection_NoIPAddress(t *testing.T) {
	p := newTestCloudSQLProvider()

	instance := &sqladmin.DatabaseInstance{
		Name:            "my-mysql",
		DatabaseVersion: "MYSQL_8_0",
		Region:          "us-central1",
		State:           "RUNNABLE",
		IpAddresses:     []*sqladmin.IpMapping{},
	}

	conn := p.cloudSQLInstanceToConnection(instance)

	if conn != nil {
		t.Error("expected nil connection when no IP address")
	}
}

func TestCloudSQLInstanceToConnection_UnsupportedEngine(t *testing.T) {
	p := newTestCloudSQLProvider()

	instance := &sqladmin.DatabaseInstance{
		Name:            "my-sqlserver",
		DatabaseVersion: "SQLSERVER_2019_STANDARD",
		Region:          "us-central1",
		State:           "RUNNABLE",
		IpAddresses: []*sqladmin.IpMapping{
			{IpAddress: "10.0.0.3", Type: "PRIMARY"},
		},
	}

	conn := p.cloudSQLInstanceToConnection(instance)

	if conn != nil {
		t.Error("expected nil connection for unsupported engine (SQL Server)")
	}
}

func TestCloudSQLInstanceToConnection_EmptyName(t *testing.T) {
	p := newTestCloudSQLProvider()

	instance := &sqladmin.DatabaseInstance{
		Name:            "",
		DatabaseVersion: "MYSQL_8_0",
		IpAddresses: []*sqladmin.IpMapping{
			{IpAddress: "10.0.0.1", Type: "PRIMARY"},
		},
	}

	conn := p.cloudSQLInstanceToConnection(instance)

	if conn != nil {
		t.Error("expected nil connection for empty name")
	}
}

func TestCloudSQLInstanceToConnection_EmptyDatabaseVersion(t *testing.T) {
	p := newTestCloudSQLProvider()

	instance := &sqladmin.DatabaseInstance{
		Name:            "my-instance",
		DatabaseVersion: "",
		IpAddresses: []*sqladmin.IpMapping{
			{IpAddress: "10.0.0.1", Type: "PRIMARY"},
		},
	}

	conn := p.cloudSQLInstanceToConnection(instance)

	if conn != nil {
		t.Error("expected nil connection for empty database version")
	}
}

func TestCloudSQLInstanceToConnection_NilSettings(t *testing.T) {
	p := newTestCloudSQLProvider()

	instance := &sqladmin.DatabaseInstance{
		Name:            "my-mysql",
		DatabaseVersion: "MYSQL_8_0",
		Region:          "us-central1",
		State:           "RUNNABLE",
		Settings:        nil,
		IpAddresses: []*sqladmin.IpMapping{
			{IpAddress: "10.0.0.1", Type: "PRIMARY"},
		},
	}

	conn := p.cloudSQLInstanceToConnection(instance)

	if conn == nil {
		t.Fatal("expected non-nil connection")
	}
	if _, ok := conn.Metadata["tier"]; ok {
		t.Error("expected no tier metadata when Settings is nil")
	}
}

func TestCloudSQLInstanceToConnection_IAMAuth(t *testing.T) {
	p := newTestCloudSQLProvider()

	instance := &sqladmin.DatabaseInstance{
		Name:            "my-postgres",
		DatabaseVersion: "POSTGRES_15",
		Region:          "us-central1",
		State:           "RUNNABLE",
		IpAddresses: []*sqladmin.IpMapping{
			{IpAddress: "10.0.0.2", Type: "PRIMARY"},
		},
		Settings: &sqladmin.Settings{
			DatabaseFlags: []*sqladmin.DatabaseFlags{
				{Name: "cloudsql.iam_authentication", Value: "on"},
			},
		},
	}

	conn := p.cloudSQLInstanceToConnection(instance)

	if conn == nil {
		t.Fatal("expected non-nil connection")
	}
	if conn.Metadata["iamAuthEnabled"] != "true" {
		t.Errorf("expected iamAuthEnabled=true, got %s", conn.Metadata["iamAuthEnabled"])
	}
}

func TestCloudSQLInstanceToConnection_FallbackIPAddress(t *testing.T) {
	p := newTestCloudSQLProvider()

	instance := &sqladmin.DatabaseInstance{
		Name:            "my-mysql",
		DatabaseVersion: "MYSQL_8_0",
		Region:          "us-central1",
		State:           "RUNNABLE",
		IpAddresses: []*sqladmin.IpMapping{
			{IpAddress: "10.0.0.5", Type: "OUTGOING"},
		},
	}

	conn := p.cloudSQLInstanceToConnection(instance)

	if conn == nil {
		t.Fatal("expected non-nil connection")
	}
	if conn.Metadata["endpoint"] != "10.0.0.5" {
		t.Errorf("expected fallback IP 10.0.0.5, got %s", conn.Metadata["endpoint"])
	}
}

func TestCloudSQLInstanceToConnection_PrimaryIPPreferred(t *testing.T) {
	p := newTestCloudSQLProvider()

	instance := &sqladmin.DatabaseInstance{
		Name:            "my-mysql",
		DatabaseVersion: "MYSQL_8_0",
		Region:          "us-central1",
		State:           "RUNNABLE",
		IpAddresses: []*sqladmin.IpMapping{
			{IpAddress: "10.0.0.5", Type: "OUTGOING"},
			{IpAddress: "10.0.0.1", Type: "PRIMARY"},
		},
	}

	conn := p.cloudSQLInstanceToConnection(instance)

	if conn == nil {
		t.Fatal("expected non-nil connection")
	}
	if conn.Metadata["endpoint"] != "10.0.0.1" {
		t.Errorf("expected PRIMARY IP 10.0.0.1, got %s", conn.Metadata["endpoint"])
	}
}
