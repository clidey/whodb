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

	awsinfra "github.com/clidey/whodb/core/src/aws"
	"github.com/clidey/whodb/core/src/providers"
)

func TestNew_ValidConfig(t *testing.T) {
	config := &Config{
		ID:     "aws-us-west-2",
		Name:   "Test AWS",
		Region: "us-west-2",
	}

	p, err := New(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if p.Type() != providers.ProviderTypeAWS {
		t.Errorf("expected type %s, got %s", providers.ProviderTypeAWS, p.Type())
	}
	if p.ID() != "aws-us-west-2" {
		t.Errorf("expected ID aws-us-west-2, got %s", p.ID())
	}
	if p.Name() != "Test AWS" {
		t.Errorf("expected Name Test AWS, got %s", p.Name())
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
		Name:   "Test AWS",
		Region: "us-west-2",
	}

	_, err := New(config)
	if err == nil {
		t.Error("expected error for missing ID")
	}
}

func TestNew_MissingRegion(t *testing.T) {
	config := &Config{
		ID:   "aws-us-west-2",
		Name: "Test AWS",
	}

	_, err := New(config)
	if err == nil {
		t.Error("expected error for missing region")
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig("aws-us-west-2", "Test AWS", "us-west-2")

	if config.ID != "aws-us-west-2" {
		t.Errorf("expected ID aws-us-west-2, got %s", config.ID)
	}
	if config.Name != "Test AWS" {
		t.Errorf("expected Name Test AWS, got %s", config.Name)
	}
	if config.Region != "us-west-2" {
		t.Errorf("expected Region us-west-2, got %s", config.Region)
	}
	if config.AuthMethod != awsinfra.AuthMethodDefault {
		t.Errorf("expected AuthMethod default, got %s", config.AuthMethod)
	}
	if !config.DiscoverRDS {
		t.Error("expected DiscoverRDS to be true")
	}
	if !config.DiscoverElastiCache {
		t.Error("expected DiscoverElastiCache to be true")
	}
	if !config.DiscoverDocumentDB {
		t.Error("expected DiscoverDocumentDB to be true")
	}
}

func TestProvider_ConnectionID(t *testing.T) {
	p, _ := New(&Config{
		ID:     "aws-us-west-2",
		Name:   "Test AWS",
		Region: "us-west-2",
	})

	connID := p.connectionID("my-rds-instance")
	expected := "aws-us-west-2/my-rds-instance"
	if connID != expected {
		t.Errorf("expected %s, got %s", expected, connID)
	}
}

func TestProvider_BuildInternalCredentials(t *testing.T) {
	p, _ := New(&Config{
		ID:              "aws-us-west-2",
		Name:            "Test AWS",
		Region:          "us-west-2",
		AuthMethod:      awsinfra.AuthMethodStatic,
		AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
		SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		SessionToken:    "session-token",
	})

	creds := p.buildInternalCredentials()

	if creds.Hostname != "us-west-2" {
		t.Errorf("expected Hostname us-west-2, got %s", creds.Hostname)
	}
	if creds.Username != "AKIAIOSFODNN7EXAMPLE" {
		t.Errorf("expected Username AKIAIOSFODNN7EXAMPLE, got %s", creds.Username)
	}
	if creds.Password != "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY" {
		t.Errorf("unexpected password")
	}
	if creds.AccessToken == nil || *creds.AccessToken != "session-token" {
		t.Errorf("expected AccessToken session-token")
	}

	// Check advanced records
	authMethodFound := false
	for _, r := range creds.Advanced {
		if r.Key == awsinfra.AdvancedKeyAuthMethod && r.Value == "static" {
			authMethodFound = true
		}
	}
	if !authMethodFound {
		t.Error("expected auth method in advanced records")
	}
}

func TestProvider_BuildInternalCredentials_Profile(t *testing.T) {
	p, _ := New(&Config{
		ID:          "aws-us-west-2",
		Name:        "Test AWS",
		Region:      "us-west-2",
		AuthMethod:  awsinfra.AuthMethodProfile,
		ProfileName: "production",
	})

	creds := p.buildInternalCredentials()

	profileFound := false
	for _, r := range creds.Advanced {
		if r.Key == awsinfra.AdvancedKeyProfileName && r.Value == "production" {
			profileFound = true
		}
	}
	if !profileFound {
		t.Error("expected profile name in advanced records")
	}
}

func TestConfig_DiscoveryFlags(t *testing.T) {
	config := &Config{
		ID:                  "test",
		Name:                "Test",
		Region:              "us-east-1",
		DiscoverRDS:         true,
		DiscoverElastiCache: false,
		DiscoverDocumentDB:  true,
	}

	if !config.DiscoverRDS {
		t.Error("expected DiscoverRDS to be true")
	}
	if config.DiscoverElastiCache {
		t.Error("expected DiscoverElastiCache to be false")
	}
	if !config.DiscoverDocumentDB {
		t.Error("expected DiscoverDocumentDB to be true")
	}
}
