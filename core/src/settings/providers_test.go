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

package settings

import (
	"errors"
	"testing"

	"github.com/clidey/whodb/core/src/providers"
)

// resetProviders clears all providers for test isolation.
func resetProviders() {
	awsProvidersMu.Lock()
	defer awsProvidersMu.Unlock()

	// Clear the providers map
	for id := range awsProviders {
		delete(awsProviders, id)
	}

	// Clear the registry
	registry := providers.GetDefaultRegistry()
	registry.Close(nil)
}

func TestAddAWSProvider_Success(t *testing.T) {
	resetProviders()
	skipPersist = true
	defer func() { skipPersist = false }()

	cfg := &AWSProviderConfig{
		ID:          "test-provider-1",
		Name:        "Test Provider",
		Region:      "us-west-2",
		AuthMethod:  "default",
		DiscoverRDS: true,
	}

	state, err := AddAWSProvider(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if state.Config.ID != cfg.ID {
		t.Errorf("expected ID %s, got %s", cfg.ID, state.Config.ID)
	}
	if state.Status != "Connected" {
		t.Errorf("expected status Connected, got %s", state.Status)
	}
}

func TestAddAWSProvider_DuplicateID(t *testing.T) {
	resetProviders()
	skipPersist = true
	defer func() { skipPersist = false }()

	cfg := &AWSProviderConfig{
		ID:         "test-duplicate",
		Name:       "Test Provider",
		Region:     "us-west-2",
		AuthMethod: "default",
	}

	_, err := AddAWSProvider(cfg)
	if err != nil {
		t.Fatalf("first add should succeed: %v", err)
	}

	_, err = AddAWSProvider(cfg)
	if !errors.Is(err, ErrProviderAlreadyExists) {
		t.Errorf("expected ErrProviderAlreadyExists, got %v", err)
	}
}

func TestGetAWSProvider_Success(t *testing.T) {
	resetProviders()
	skipPersist = true
	defer func() { skipPersist = false }()

	cfg := &AWSProviderConfig{
		ID:         "test-get",
		Name:       "Test Get",
		Region:     "us-east-1",
		AuthMethod: "default",
	}

	_, err := AddAWSProvider(cfg)
	if err != nil {
		t.Fatalf("add failed: %v", err)
	}

	state, err := GetAWSProvider("test-get")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}

	if state.Config.Name != "Test Get" {
		t.Errorf("expected name 'Test Get', got %s", state.Config.Name)
	}
}

func TestGetAWSProvider_NotFound(t *testing.T) {
	resetProviders()

	_, err := GetAWSProvider("nonexistent")
	if !errors.Is(err, ErrProviderNotFound) {
		t.Errorf("expected ErrProviderNotFound, got %v", err)
	}
}

func TestGetAWSProviders(t *testing.T) {
	resetProviders()
	skipPersist = true
	defer func() { skipPersist = false }()

	// Add two providers
	cfg1 := &AWSProviderConfig{
		ID: "test-list-1", Name: "Provider 1", Region: "us-west-2", AuthMethod: "default",
	}
	cfg2 := &AWSProviderConfig{
		ID: "test-list-2", Name: "Provider 2", Region: "eu-west-1", AuthMethod: "default",
	}

	AddAWSProvider(cfg1)
	AddAWSProvider(cfg2)

	providers := GetAWSProviders()
	if len(providers) != 2 {
		t.Errorf("expected 2 providers, got %d", len(providers))
	}
}

func TestUpdateAWSProvider_Success(t *testing.T) {
	resetProviders()
	skipPersist = true
	defer func() { skipPersist = false }()

	cfg := &AWSProviderConfig{
		ID:          "test-update",
		Name:        "Original Name",
		Region:      "us-west-2",
		AuthMethod:  "default",
		DiscoverRDS: true,
	}

	_, err := AddAWSProvider(cfg)
	if err != nil {
		t.Fatalf("add failed: %v", err)
	}

	updatedCfg := &AWSProviderConfig{
		ID:                  "test-update",
		Name:                "Updated Name",
		Region:              "us-west-2",
		AuthMethod:          "default",
		DiscoverRDS:         true,
		DiscoverElastiCache: true,
	}

	state, err := UpdateAWSProvider("test-update", updatedCfg)
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}

	if state.Config.Name != "Updated Name" {
		t.Errorf("expected name 'Updated Name', got %s", state.Config.Name)
	}
	if !state.Config.DiscoverElastiCache {
		t.Error("expected DiscoverElastiCache to be true")
	}
}

func TestUpdateAWSProvider_NotFound(t *testing.T) {
	resetProviders()
	skipPersist = true
	defer func() { skipPersist = false }()

	cfg := &AWSProviderConfig{
		ID: "nonexistent", Name: "Test", Region: "us-west-2", AuthMethod: "default",
	}

	_, err := UpdateAWSProvider("nonexistent", cfg)
	if !errors.Is(err, ErrProviderNotFound) {
		t.Errorf("expected ErrProviderNotFound, got %v", err)
	}
}

func TestRemoveAWSProvider_Success(t *testing.T) {
	resetProviders()
	skipPersist = true
	defer func() { skipPersist = false }()

	cfg := &AWSProviderConfig{
		ID: "test-remove", Name: "To Remove", Region: "us-west-2", AuthMethod: "default",
	}

	_, err := AddAWSProvider(cfg)
	if err != nil {
		t.Fatalf("add failed: %v", err)
	}

	err = RemoveAWSProvider("test-remove")
	if err != nil {
		t.Fatalf("remove failed: %v", err)
	}

	// Verify removed
	_, err = GetAWSProvider("test-remove")
	if !errors.Is(err, ErrProviderNotFound) {
		t.Error("provider should be removed")
	}
}

func TestRemoveAWSProvider_NotFound(t *testing.T) {
	resetProviders()

	err := RemoveAWSProvider("nonexistent")
	if !errors.Is(err, ErrProviderNotFound) {
		t.Errorf("expected ErrProviderNotFound, got %v", err)
	}
}

func TestGenerateProviderID(t *testing.T) {
	tests := []struct {
		name     string
		region   string
		expected string
	}{
		{"MyProvider", "us-west-2", "aws-MyProvider-us-west-2"},
		{"Test Provider", "eu-west-1", "aws-TestProvider-eu-west-1"},
		{"test-with-dash", "ap-south-1", "aws-test-with-dash-ap-south-1"},
		{"Special@Chars!", "us-east-1", "aws-SpecialChars-us-east-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateProviderID(tt.name, tt.region)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

