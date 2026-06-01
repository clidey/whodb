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
	"errors"
	"testing"

	"github.com/clidey/whodb/core/src/engine"
)

func TestParseFromWhoDB_DefaultAuth(t *testing.T) {
	creds := &engine.Credentials{
		Hostname: "us-central1",
		Advanced: []engine.Record{
			{Key: AdvancedKeyProjectID, Value: "my-project-123"},
		},
	}

	config, err := ParseFromWhoDB(creds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if config.AuthMethod != AuthMethodDefault {
		t.Errorf("expected auth method default, got %s", config.AuthMethod)
	}
	if config.Region != "us-central1" {
		t.Errorf("expected region us-central1, got %s", config.Region)
	}
	if config.ProjectID != "my-project-123" {
		t.Errorf("expected project ID my-project-123, got %s", config.ProjectID)
	}
}

func TestParseFromWhoDB_ServiceAccountKeyAuth(t *testing.T) {
	creds := &engine.Credentials{
		Hostname: "us-east1",
		Advanced: []engine.Record{
			{Key: AdvancedKeyProjectID, Value: "my-project-123"},
			{Key: AdvancedKeyAuthMethod, Value: "service-account-key"},
			{Key: AdvancedKeyServiceAccountKeyPath, Value: "/path/to/key.json"},
		},
	}

	config, err := ParseFromWhoDB(creds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if config.AuthMethod != AuthMethodServiceAccountKey {
		t.Errorf("expected auth method service-account-key, got %s", config.AuthMethod)
	}
	if config.ServiceAccountKeyPath != "/path/to/key.json" {
		t.Errorf("expected key path /path/to/key.json, got %s", config.ServiceAccountKeyPath)
	}
}

func TestParseFromWhoDB_NilCredentials(t *testing.T) {
	_, err := ParseFromWhoDB(nil)
	if err == nil {
		t.Error("expected error for nil credentials")
	}
}

func TestParseFromWhoDB_MissingRegion(t *testing.T) {
	creds := &engine.Credentials{
		Advanced: []engine.Record{
			{Key: AdvancedKeyProjectID, Value: "my-project-123"},
		},
	}

	_, err := ParseFromWhoDB(creds)
	if !errors.Is(err, ErrRegionRequired) {
		t.Errorf("expected ErrRegionRequired, got %v", err)
	}
}

func TestParseFromWhoDB_MissingProjectID(t *testing.T) {
	creds := &engine.Credentials{
		Hostname: "us-central1",
	}

	_, err := ParseFromWhoDB(creds)
	if !errors.Is(err, ErrProjectIDRequired) {
		t.Errorf("expected ErrProjectIDRequired, got %v", err)
	}
}

func TestParseFromWhoDB_ServiceAccountKeyMissingPath(t *testing.T) {
	creds := &engine.Credentials{
		Hostname: "us-central1",
		Advanced: []engine.Record{
			{Key: AdvancedKeyProjectID, Value: "my-project-123"},
			{Key: AdvancedKeyAuthMethod, Value: "service-account-key"},
		},
	}

	_, err := ParseFromWhoDB(creds)
	if !errors.Is(err, ErrServiceAccountKeyPathRequired) {
		t.Errorf("expected ErrServiceAccountKeyPathRequired, got %v", err)
	}
}

func TestParseFromWhoDB_InvalidAuthMethod(t *testing.T) {
	creds := &engine.Credentials{
		Hostname: "us-central1",
		Advanced: []engine.Record{
			{Key: AdvancedKeyProjectID, Value: "my-project-123"},
			{Key: AdvancedKeyAuthMethod, Value: "invalid-method"},
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
		{"DEFAULT", AuthMethodDefault},
		{"Default", AuthMethodDefault},
		{"default", AuthMethodDefault},
		{"SERVICE-ACCOUNT-KEY", AuthMethodServiceAccountKey},
		{"Service-Account-Key", AuthMethodServiceAccountKey},
	}

	for _, tc := range testCases {
		creds := &engine.Credentials{
			Hostname: "us-central1",
			Advanced: []engine.Record{
				{Key: AdvancedKeyProjectID, Value: "my-project-123"},
				{Key: AdvancedKeyAuthMethod, Value: tc.input},
				{Key: AdvancedKeyServiceAccountKeyPath, Value: "/path/to/key.json"},
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

func TestParseFromWhoDB_RegionFromAdvanced(t *testing.T) {
	creds := &engine.Credentials{
		Advanced: []engine.Record{
			{Key: AdvancedKeyProjectID, Value: "my-project-123"},
			{Key: "Region", Value: "europe-west1"},
		},
	}

	config, err := ParseFromWhoDB(creds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if config.Region != "europe-west1" {
		t.Errorf("expected region europe-west1, got %s", config.Region)
	}
}

func TestParseFromWhoDB_HostnameTakesPrecedence(t *testing.T) {
	creds := &engine.Credentials{
		Hostname: "us-central1",
		Advanced: []engine.Record{
			{Key: AdvancedKeyProjectID, Value: "my-project-123"},
			{Key: "Region", Value: "europe-west1"},
		},
	}

	config, err := ParseFromWhoDB(creds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if config.Region != "us-central1" {
		t.Errorf("expected region us-central1 (from Hostname), got %s", config.Region)
	}
}

func TestGCPCredentialConfig_IsServiceAccountKeyAuth(t *testing.T) {
	config := &GCPCredentialConfig{
		ProjectID:  "proj-1",
		Region:     "us-central1",
		AuthMethod: AuthMethodDefault,
	}

	if config.IsServiceAccountKeyAuth() {
		t.Error("expected IsServiceAccountKeyAuth to return false for default auth")
	}

	config.AuthMethod = AuthMethodServiceAccountKey
	if !config.IsServiceAccountKeyAuth() {
		t.Error("expected IsServiceAccountKeyAuth to return true for service-account-key auth")
	}
}

func TestValidate_DirectCall(t *testing.T) {
	config := &GCPCredentialConfig{
		ProjectID:  "proj-1",
		Region:     "us-central1",
		AuthMethod: AuthMethodDefault,
	}

	if err := config.Validate(); err != nil {
		t.Errorf("unexpected error for valid default config: %v", err)
	}
}

func TestValidate_EmptyRegion(t *testing.T) {
	config := &GCPCredentialConfig{
		ProjectID:  "proj-1",
		AuthMethod: AuthMethodDefault,
	}

	if err := config.Validate(); !errors.Is(err, ErrRegionRequired) {
		t.Errorf("expected ErrRegionRequired, got %v", err)
	}
}

func TestValidate_EmptyProjectID(t *testing.T) {
	config := &GCPCredentialConfig{
		Region:     "us-central1",
		AuthMethod: AuthMethodDefault,
	}

	if err := config.Validate(); !errors.Is(err, ErrProjectIDRequired) {
		t.Errorf("expected ErrProjectIDRequired, got %v", err)
	}
}

func TestValidate_ServiceAccountKeyValid(t *testing.T) {
	config := &GCPCredentialConfig{
		ProjectID:             "proj-1",
		Region:                "us-central1",
		AuthMethod:            AuthMethodServiceAccountKey,
		ServiceAccountKeyPath: "/path/to/key.json",
	}

	if err := config.Validate(); err != nil {
		t.Errorf("unexpected error for valid service account key config: %v", err)
	}
}

func TestValidate_ServiceAccountKeyMissingPath(t *testing.T) {
	config := &GCPCredentialConfig{
		ProjectID:  "proj-1",
		Region:     "us-central1",
		AuthMethod: AuthMethodServiceAccountKey,
	}

	if err := config.Validate(); !errors.Is(err, ErrServiceAccountKeyPathRequired) {
		t.Errorf("expected ErrServiceAccountKeyPathRequired, got %v", err)
	}
}

func TestValidate_InvalidAuthMethod(t *testing.T) {
	config := &GCPCredentialConfig{
		ProjectID:  "proj-1",
		Region:     "us-central1",
		AuthMethod: AuthMethod("invalid"),
	}

	if err := config.Validate(); !errors.Is(err, ErrInvalidAuthMethod) {
		t.Errorf("expected ErrInvalidAuthMethod, got %v", err)
	}
}
