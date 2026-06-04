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
	"errors"
	"testing"

	"github.com/clidey/whodb/core/src/engine"
)

func TestParseFromWhoDB_ServicePrincipalAuth(t *testing.T) {
	creds := &engine.Credentials{
		Hostname: "sub-12345",
		Advanced: []engine.Record{
			{Key: AdvancedKeyAuthMethod, Value: "service-principal"},
			{Key: AdvancedKeyTenantID, Value: "tenant-abc"},
			{Key: AdvancedKeyClientID, Value: "client-xyz"},
			{Key: AdvancedKeyClientSecret, Value: "secret-123"},
		},
	}

	config, err := ParseFromWhoDB(creds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if config.AuthMethod != AuthMethodServicePrincipal {
		t.Errorf("expected auth method service-principal, got %s", config.AuthMethod)
	}
	if config.TenantID != "tenant-abc" {
		t.Errorf("expected tenant ID tenant-abc, got %s", config.TenantID)
	}
	if config.ClientID != "client-xyz" {
		t.Errorf("expected client ID client-xyz, got %s", config.ClientID)
	}
	if config.ClientSecret != "secret-123" {
		t.Errorf("expected client secret secret-123, got %s", config.ClientSecret)
	}
}

func TestParseFromWhoDB_DefaultAuth(t *testing.T) {
	creds := &engine.Credentials{
		Hostname: "sub-12345",
	}

	config, err := ParseFromWhoDB(creds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if config.AuthMethod != AuthMethodDefault {
		t.Errorf("expected auth method default, got %s", config.AuthMethod)
	}
}

func TestParseFromWhoDB_NilCredentials(t *testing.T) {
	_, err := ParseFromWhoDB(nil)
	if err == nil {
		t.Error("expected error for nil credentials")
	}
}

func TestParseFromWhoDB_MissingSubscriptionID(t *testing.T) {
	creds := &engine.Credentials{
		Advanced: []engine.Record{
			{Key: AdvancedKeyAuthMethod, Value: "default"},
		},
	}

	_, err := ParseFromWhoDB(creds)
	if !errors.Is(err, ErrSubscriptionRequired) {
		t.Errorf("expected ErrSubscriptionRequired, got %v", err)
	}
}

func TestParseFromWhoDB_ServicePrincipalMissingTenantID(t *testing.T) {
	creds := &engine.Credentials{
		Hostname: "sub-12345",
		Advanced: []engine.Record{
			{Key: AdvancedKeyAuthMethod, Value: "service-principal"},
			{Key: AdvancedKeyClientID, Value: "client-xyz"},
			{Key: AdvancedKeyClientSecret, Value: "secret-123"},
		},
	}

	_, err := ParseFromWhoDB(creds)
	if !errors.Is(err, ErrTenantIDRequired) {
		t.Errorf("expected ErrTenantIDRequired, got %v", err)
	}
}

func TestParseFromWhoDB_ServicePrincipalMissingClientID(t *testing.T) {
	creds := &engine.Credentials{
		Hostname: "sub-12345",
		Advanced: []engine.Record{
			{Key: AdvancedKeyAuthMethod, Value: "service-principal"},
			{Key: AdvancedKeyTenantID, Value: "tenant-abc"},
			{Key: AdvancedKeyClientSecret, Value: "secret-123"},
		},
	}

	_, err := ParseFromWhoDB(creds)
	if !errors.Is(err, ErrClientIDRequired) {
		t.Errorf("expected ErrClientIDRequired, got %v", err)
	}
}

func TestParseFromWhoDB_ServicePrincipalMissingClientSecret(t *testing.T) {
	creds := &engine.Credentials{
		Hostname: "sub-12345",
		Advanced: []engine.Record{
			{Key: AdvancedKeyAuthMethod, Value: "service-principal"},
			{Key: AdvancedKeyTenantID, Value: "tenant-abc"},
			{Key: AdvancedKeyClientID, Value: "client-xyz"},
		},
	}

	_, err := ParseFromWhoDB(creds)
	if !errors.Is(err, ErrClientSecretRequired) {
		t.Errorf("expected ErrClientSecretRequired, got %v", err)
	}
}

func TestParseFromWhoDB_InvalidAuthMethod(t *testing.T) {
	creds := &engine.Credentials{
		Hostname: "sub-12345",
		Advanced: []engine.Record{
			{Key: AdvancedKeyAuthMethod, Value: "invalid"},
		},
	}

	_, err := ParseFromWhoDB(creds)
	if !errors.Is(err, ErrInvalidAuthMethod) {
		t.Errorf("expected ErrInvalidAuthMethod, got %v", err)
	}
}

func TestParseFromWhoDB_AuthMethodCaseInsensitive(t *testing.T) {
	testCases := []struct {
		input    string
		expected AuthMethod
	}{
		{"SERVICE-PRINCIPAL", AuthMethodServicePrincipal},
		{"Service-Principal", AuthMethodServicePrincipal},
		{"DEFAULT", AuthMethodDefault},
		{"Default", AuthMethodDefault},
	}

	for _, tc := range testCases {
		creds := &engine.Credentials{
			Hostname: "sub-12345",
			Advanced: []engine.Record{
				{Key: AdvancedKeyAuthMethod, Value: tc.input},
				{Key: AdvancedKeyTenantID, Value: "tenant-abc"},
				{Key: AdvancedKeyClientID, Value: "client-xyz"},
				{Key: AdvancedKeyClientSecret, Value: "secret-123"},
			},
		}

		config, err := ParseFromWhoDB(creds)
		if err != nil {
			t.Errorf("unexpected error for auth method %s: %v", tc.input, err)
			continue
		}
		if config.AuthMethod != tc.expected {
			t.Errorf("expected auth method %s for input %s, got %s", tc.expected, tc.input, config.AuthMethod)
		}
	}
}

func TestParseFromWhoDB_SubscriptionIDFromAdvanced(t *testing.T) {
	creds := &engine.Credentials{
		Advanced: []engine.Record{
			{Key: AdvancedKeySubscriptionID, Value: "sub-from-advanced"},
		},
	}

	config, err := ParseFromWhoDB(creds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if config.SubscriptionID != "sub-from-advanced" {
		t.Errorf("expected subscription ID sub-from-advanced, got %s", config.SubscriptionID)
	}
}

func TestParseFromWhoDB_HostnameTakesPrecedence(t *testing.T) {
	creds := &engine.Credentials{
		Hostname: "sub-from-hostname",
		Advanced: []engine.Record{
			{Key: AdvancedKeySubscriptionID, Value: "sub-from-advanced"},
		},
	}

	config, err := ParseFromWhoDB(creds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if config.SubscriptionID != "sub-from-hostname" {
		t.Errorf("expected subscription ID sub-from-hostname, got %s", config.SubscriptionID)
	}
}

func TestParseFromWhoDB_ResourceGroup(t *testing.T) {
	creds := &engine.Credentials{
		Hostname: "sub-12345",
		Advanced: []engine.Record{
			{Key: AdvancedKeyResourceGroup, Value: "my-rg"},
		},
	}

	config, err := ParseFromWhoDB(creds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if config.ResourceGroup != "my-rg" {
		t.Errorf("expected resource group my-rg, got %s", config.ResourceGroup)
	}
}

func TestAzureCredentialConfig_IsServicePrincipalAuth(t *testing.T) {
	config := &AzureCredentialConfig{
		SubscriptionID: "sub-12345",
		AuthMethod:     AuthMethodDefault,
	}

	if config.IsServicePrincipalAuth() {
		t.Error("expected IsServicePrincipalAuth to return false for default auth")
	}

	config.AuthMethod = AuthMethodServicePrincipal
	if !config.IsServicePrincipalAuth() {
		t.Error("expected IsServicePrincipalAuth to return true for service-principal auth")
	}
}

func TestValidate_DirectCall(t *testing.T) {
	config := &AzureCredentialConfig{
		SubscriptionID: "sub-12345",
		AuthMethod:     AuthMethodDefault,
	}

	if err := config.Validate(); err != nil {
		t.Errorf("unexpected error for valid default config: %v", err)
	}
}

func TestValidate_EmptySubscription(t *testing.T) {
	config := &AzureCredentialConfig{
		AuthMethod: AuthMethodDefault,
	}

	if err := config.Validate(); !errors.Is(err, ErrSubscriptionRequired) {
		t.Errorf("expected ErrSubscriptionRequired, got %v", err)
	}
}

func TestValidate_ServicePrincipalValid(t *testing.T) {
	config := &AzureCredentialConfig{
		SubscriptionID: "sub-12345",
		AuthMethod:     AuthMethodServicePrincipal,
		TenantID:       "tenant-abc",
		ClientID:       "client-xyz",
		ClientSecret:   "secret-123",
	}

	if err := config.Validate(); err != nil {
		t.Errorf("unexpected error for valid service principal config: %v", err)
	}
}
