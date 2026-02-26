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

	"github.com/clidey/whodb/core/src/engine"
)

func TestParseFromWhoDB_ProfileAuth(t *testing.T) {
	creds := &engine.Credentials{
		Hostname: "eu-west-1",
		Advanced: []engine.Record{
			{Key: AdvancedKeyAuthMethod, Value: "profile"},
			{Key: AdvancedKeyProfileName, Value: "production"},
		},
	}

	config, err := ParseFromWhoDB(creds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if config.AuthMethod != AuthMethodProfile {
		t.Errorf("expected auth method profile, got %s", config.AuthMethod)
	}
	if config.ProfileName != "production" {
		t.Errorf("expected profile name production, got %s", config.ProfileName)
	}
}

func TestParseFromWhoDB_DefaultAuth(t *testing.T) {
	creds := &engine.Credentials{
		Hostname: "ap-southeast-1",
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

func TestParseFromWhoDB_MissingRegion(t *testing.T) {
	creds := &engine.Credentials{
		Advanced: []engine.Record{
			{Key: AdvancedKeyAuthMethod, Value: "default"},
		},
	}

	_, err := ParseFromWhoDB(creds)
	if err != ErrRegionRequired {
		t.Errorf("expected ErrRegionRequired, got %v", err)
	}
}

func TestParseFromWhoDB_ProfileAuthMissingName(t *testing.T) {
	creds := &engine.Credentials{
		Hostname: "us-west-2",
		Advanced: []engine.Record{
			{Key: AdvancedKeyAuthMethod, Value: "profile"},
		},
	}

	_, err := ParseFromWhoDB(creds)
	if err != ErrProfileNameRequired {
		t.Errorf("expected ErrProfileNameRequired, got %v", err)
	}
}

func TestParseFromWhoDB_InvalidAuthMethod(t *testing.T) {
	creds := &engine.Credentials{
		Hostname: "us-west-2",
		Advanced: []engine.Record{
			{Key: AdvancedKeyAuthMethod, Value: "invalid"},
		},
	}

	_, err := ParseFromWhoDB(creds)
	if err != ErrInvalidAuthMethod {
		t.Errorf("expected ErrInvalidAuthMethod, got %v", err)
	}
}

func TestParseFromWhoDB_AuthMethodCaseInsensitive(t *testing.T) {
	testCases := []struct {
		input    string
		expected AuthMethod
	}{
		{"PROFILE", AuthMethodProfile},
		{"Profile", AuthMethodProfile},
		{"DEFAULT", AuthMethodDefault},
		{"Default", AuthMethodDefault},
	}

	for _, tc := range testCases {
		creds := &engine.Credentials{
			Hostname: "us-west-2",
			Advanced: []engine.Record{
				{Key: AdvancedKeyAuthMethod, Value: tc.input},
				{Key: AdvancedKeyProfileName, Value: "test"}, // For profile auth
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

func TestAWSCredentialConfig_IsProfileAuth(t *testing.T) {
	config := &AWSCredentialConfig{
		Region:     "us-west-2",
		AuthMethod: AuthMethodDefault,
	}

	if config.IsProfileAuth() {
		t.Error("expected IsProfileAuth to return false for default auth")
	}

	config.AuthMethod = AuthMethodProfile
	if !config.IsProfileAuth() {
		t.Error("expected IsProfileAuth to return true for profile auth")
	}
}
