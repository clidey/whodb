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

package azure

import (
	"testing"

	azureinfra "github.com/clidey/whodb/core/src/azure"
	"github.com/clidey/whodb/core/src/providers"
)

func TestNew_ValidConfig(t *testing.T) {
	config := &Config{
		ID:             "azure-sub-1",
		Name:           "Test Azure",
		SubscriptionID: "00000000-0000-0000-0000-000000000000",
	}

	p, err := New(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if p.Type() != providers.ProviderTypeAzure {
		t.Errorf("expected type %s, got %s", providers.ProviderTypeAzure, p.Type())
	}
	if p.ID() != "azure-sub-1" {
		t.Errorf("expected ID azure-sub-1, got %s", p.ID())
	}
	if p.Name() != "Test Azure" {
		t.Errorf("expected Name Test Azure, got %s", p.Name())
	}
}

func TestNew_NilConfig(t *testing.T) {
	_, err := New(nil)
	if err == nil {
		t.Error("expected error for nil config")
	}
}

func TestNew_MissingID(t *testing.T) {
	config := &Config{
		Name:           "Test Azure",
		SubscriptionID: "00000000-0000-0000-0000-000000000000",
	}

	_, err := New(config)
	if err == nil {
		t.Error("expected error for missing ID")
	}
}

func TestNew_MissingSubscriptionID(t *testing.T) {
	config := &Config{
		ID:   "azure-sub-1",
		Name: "Test Azure",
	}

	_, err := New(config)
	if err == nil {
		t.Error("expected error for missing subscription ID")
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig("azure-sub-1", "Test Azure", "00000000-0000-0000-0000-000000000000")

	if config.ID != "azure-sub-1" {
		t.Errorf("expected ID azure-sub-1, got %s", config.ID)
	}
	if config.Name != "Test Azure" {
		t.Errorf("expected Name Test Azure, got %s", config.Name)
	}
	if config.SubscriptionID != "00000000-0000-0000-0000-000000000000" {
		t.Errorf("expected SubscriptionID 00000000-0000-0000-0000-000000000000, got %s", config.SubscriptionID)
	}
	if config.AuthMethod != azureinfra.AuthMethodDefault {
		t.Errorf("expected AuthMethod default, got %s", config.AuthMethod)
	}
	if !config.DiscoverPostgreSQL {
		t.Error("expected DiscoverPostgreSQL to be true")
	}
	if !config.DiscoverMySQL {
		t.Error("expected DiscoverMySQL to be true")
	}
	if !config.DiscoverRedis {
		t.Error("expected DiscoverRedis to be true")
	}
	if !config.DiscoverCosmosDB {
		t.Error("expected DiscoverCosmosDB to be true")
	}
}

func TestProvider_ConnectionID(t *testing.T) {
	p, _ := New(&Config{
		ID:             "azure-sub-1",
		Name:           "Test Azure",
		SubscriptionID: "00000000-0000-0000-0000-000000000000",
	})

	connID := p.connectionID("pg-my-server")
	expected := "azure-sub-1/pg-my-server"
	if connID != expected {
		t.Errorf("expected %s, got %s", expected, connID)
	}
}

func TestProvider_BuildInternalCredentials(t *testing.T) {
	p, _ := New(&Config{
		ID:             "azure-sub-1",
		Name:           "Test Azure",
		SubscriptionID: "00000000-0000-0000-0000-000000000000",
		AuthMethod:     azureinfra.AuthMethodDefault,
	})

	creds := p.buildInternalCredentials()

	if creds.Hostname != "00000000-0000-0000-0000-000000000000" {
		t.Errorf("expected Hostname 00000000-0000-0000-0000-000000000000, got %s", creds.Hostname)
	}

	authMethodFound := false
	for _, r := range creds.Advanced {
		if r.Key == azureinfra.AdvancedKeyAuthMethod && r.Value == "default" {
			authMethodFound = true
		}
	}
	if !authMethodFound {
		t.Error("expected auth method in advanced records")
	}
}

func TestProvider_BuildInternalCredentials_ServicePrincipal(t *testing.T) {
	p, _ := New(&Config{
		ID:             "azure-sub-1",
		Name:           "Test Azure",
		SubscriptionID: "00000000-0000-0000-0000-000000000000",
		AuthMethod:     azureinfra.AuthMethodServicePrincipal,
		TenantID:       "tenant-123",
		ClientID:       "client-456",
		ClientSecret:   "secret-789",
		ResourceGroup:  "my-rg",
	})

	creds := p.buildInternalCredentials()

	expected := map[string]string{
		azureinfra.AdvancedKeyAuthMethod:    "service-principal",
		azureinfra.AdvancedKeyTenantID:      "tenant-123",
		azureinfra.AdvancedKeyClientID:      "client-456",
		azureinfra.AdvancedKeyClientSecret:  "secret-789",
		azureinfra.AdvancedKeyResourceGroup: "my-rg",
	}

	found := make(map[string]string)
	for _, r := range creds.Advanced {
		found[r.Key] = r.Value
	}

	for k, v := range expected {
		if found[k] != v {
			t.Errorf("expected advanced record %s=%s, got %s", k, v, found[k])
		}
	}
}

func TestConfig_DiscoveryFlags(t *testing.T) {
	config := &Config{
		ID:                 "test",
		Name:               "Test",
		SubscriptionID:     "sub-1",
		DiscoverPostgreSQL: true,
		DiscoverMySQL:      false,
		DiscoverRedis:      true,
		DiscoverCosmosDB:   false,
	}

	if !config.DiscoverPostgreSQL {
		t.Error("expected DiscoverPostgreSQL to be true")
	}
	if config.DiscoverMySQL {
		t.Error("expected DiscoverMySQL to be false")
	}
	if !config.DiscoverRedis {
		t.Error("expected DiscoverRedis to be true")
	}
	if config.DiscoverCosmosDB {
		t.Error("expected DiscoverCosmosDB to be false")
	}
}

func TestExtractResourceGroup(t *testing.T) {
	testCases := []struct {
		resourceID string
		expected   string
	}{
		{
			"/subscriptions/00000000/resourceGroups/my-rg/providers/Microsoft.DBforPostgreSQL/flexibleServers/my-server",
			"my-rg",
		},
		{
			"/subscriptions/00000000/resourcegroups/lower-rg/providers/Microsoft.Cache/Redis/my-redis",
			"lower-rg",
		},
		{
			"/subscriptions/00000000/RESOURCEGROUPS/UPPER-RG/providers/Microsoft.DocumentDB/databaseAccounts/my-cosmos",
			"UPPER-RG",
		},
		{
			"",
			"",
		},
		{
			"/subscriptions/00000000/providers/Microsoft.DBforPostgreSQL/flexibleServers/my-server",
			"",
		},
		{
			"/subscriptions/00000000/resourceGroups/trailing-rg",
			"trailing-rg",
		},
	}

	for _, tc := range testCases {
		result := extractResourceGroup(tc.resourceID)
		if result != tc.expected {
			t.Errorf("extractResourceGroup(%s): expected %q, got %q", tc.resourceID, tc.expected, result)
		}
	}
}

func TestConfig_String_ExcludesSecrets(t *testing.T) {
	config := &Config{
		ID:             "azure-sub-1",
		Name:           "Test Azure",
		SubscriptionID: "sub-123",
		ClientSecret:   "super-secret-value",
		AuthMethod:     azureinfra.AuthMethodServicePrincipal,
	}

	s := config.String()
	if s == "" {
		t.Error("expected non-empty string")
	}
	if contains(s, "super-secret-value") {
		t.Error("String() should not include client secret")
	}
}

func contains(s, substr string) bool {
	return indexOf(s, substr) != -1
}
