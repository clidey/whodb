/*
 * Copyright 2025 Clidey, Inc.
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

func TestParseFromWhoDB_StaticAuth(t *testing.T) {
	creds := &engine.Credentials{
		Hostname: "us-west-2",
		Username: "AKIAIOSFODNN7EXAMPLE",
		Password: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		Advanced: []engine.Record{
			{Key: AdvancedKeyAuthMethod, Value: "static"},
		},
	}

	config, err := ParseFromWhoDB(creds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if config.Region != "us-west-2" {
		t.Errorf("expected region us-west-2, got %s", config.Region)
	}
	if config.AuthMethod != AuthMethodStatic {
		t.Errorf("expected auth method static, got %s", config.AuthMethod)
	}
	if config.AccessKeyID != "AKIAIOSFODNN7EXAMPLE" {
		t.Errorf("expected access key ID AKIAIOSFODNN7EXAMPLE, got %s", config.AccessKeyID)
	}
	if config.SecretAccessKey != "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY" {
		t.Errorf("expected secret access key, got %s", config.SecretAccessKey)
	}
}

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

func TestParseFromWhoDB_IAMAuth(t *testing.T) {
	creds := &engine.Credentials{
		Hostname: "us-east-1",
		Advanced: []engine.Record{
			{Key: AdvancedKeyAuthMethod, Value: "iam"},
		},
	}

	config, err := ParseFromWhoDB(creds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if config.AuthMethod != AuthMethodIAM {
		t.Errorf("expected auth method iam, got %s", config.AuthMethod)
	}
}

func TestParseFromWhoDB_EnvAuth(t *testing.T) {
	creds := &engine.Credentials{
		Hostname: "us-east-1",
		Advanced: []engine.Record{
			{Key: AdvancedKeyAuthMethod, Value: "env"},
		},
	}

	config, err := ParseFromWhoDB(creds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if config.AuthMethod != AuthMethodEnv {
		t.Errorf("expected auth method env, got %s", config.AuthMethod)
	}
}

func TestParseFromWhoDB_CustomEndpoint(t *testing.T) {
	creds := &engine.Credentials{
		Hostname: "us-west-2",
		Username: "test",
		Password: "test",
		Advanced: []engine.Record{
			{Key: AdvancedKeyAuthMethod, Value: "static"},
			{Key: AdvancedKeyEndpoint, Value: "http://localhost:4566"},
		},
	}

	config, err := ParseFromWhoDB(creds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if config.Endpoint != "http://localhost:4566" {
		t.Errorf("expected endpoint http://localhost:4566, got %s", config.Endpoint)
	}
	if !config.HasCustomEndpoint() {
		t.Error("expected HasCustomEndpoint to return true")
	}
}

func TestParseFromWhoDB_SessionToken(t *testing.T) {
	sessionToken := "FwoGZXIvYXdzE..."
	creds := &engine.Credentials{
		Hostname:    "us-west-2",
		Username:    "AKIAIOSFODNN7EXAMPLE",
		Password:    "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		AccessToken: &sessionToken,
		Advanced: []engine.Record{
			{Key: AdvancedKeyAuthMethod, Value: "static"},
		},
	}

	config, err := ParseFromWhoDB(creds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if config.SessionToken != sessionToken {
		t.Errorf("expected session token %s, got %s", sessionToken, config.SessionToken)
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
		Username: "AKIAIOSFODNN7EXAMPLE",
		Password: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		Advanced: []engine.Record{
			{Key: AdvancedKeyAuthMethod, Value: "static"},
		},
	}

	_, err := ParseFromWhoDB(creds)
	if err != ErrRegionRequired {
		t.Errorf("expected ErrRegionRequired, got %v", err)
	}
}

func TestParseFromWhoDB_StaticAuthMissingCredentials(t *testing.T) {
	// Missing access key
	creds := &engine.Credentials{
		Hostname: "us-west-2",
		Password: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		Advanced: []engine.Record{
			{Key: AdvancedKeyAuthMethod, Value: "static"},
		},
	}

	_, err := ParseFromWhoDB(creds)
	if err != ErrStaticCredentialsRequired {
		t.Errorf("expected ErrStaticCredentialsRequired, got %v", err)
	}

	// Missing secret key
	creds = &engine.Credentials{
		Hostname: "us-west-2",
		Username: "AKIAIOSFODNN7EXAMPLE",
		Advanced: []engine.Record{
			{Key: AdvancedKeyAuthMethod, Value: "static"},
		},
	}

	_, err = ParseFromWhoDB(creds)
	if err != ErrStaticCredentialsRequired {
		t.Errorf("expected ErrStaticCredentialsRequired, got %v", err)
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
		{"STATIC", AuthMethodStatic},
		{"Static", AuthMethodStatic},
		{"PROFILE", AuthMethodProfile},
		{"Profile", AuthMethodProfile},
		{"IAM", AuthMethodIAM},
		{"Iam", AuthMethodIAM},
		{"ENV", AuthMethodEnv},
		{"Env", AuthMethodEnv},
		{"DEFAULT", AuthMethodDefault},
		{"Default", AuthMethodDefault},
	}

	for _, tc := range testCases {
		creds := &engine.Credentials{
			Hostname: "us-west-2",
			Username: "AKIAIOSFODNN7EXAMPLE",
			Password: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
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

func TestBuildCredentialsProvider_Static(t *testing.T) {
	config := &AWSCredentialConfig{
		Region:          "us-west-2",
		AuthMethod:      AuthMethodStatic,
		AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
		SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
	}

	provider := config.BuildCredentialsProvider()
	if provider == nil {
		t.Error("expected non-nil credentials provider for static auth")
	}
}

func TestBuildCredentialsProvider_NonStatic(t *testing.T) {
	testCases := []AuthMethod{
		AuthMethodProfile,
		AuthMethodIAM,
		AuthMethodEnv,
		AuthMethodDefault,
	}

	for _, authMethod := range testCases {
		config := &AWSCredentialConfig{
			Region:      "us-west-2",
			AuthMethod:  authMethod,
			ProfileName: "test", // For profile auth
		}

		provider := config.BuildCredentialsProvider()
		if provider != nil {
			t.Errorf("expected nil credentials provider for auth method %s", authMethod)
		}
	}
}

func TestAWSCredentialConfig_Helpers(t *testing.T) {
	config := &AWSCredentialConfig{
		Region:     "us-west-2",
		AuthMethod: AuthMethodStatic,
		Endpoint:   "http://localhost:4566",
	}

	if !config.HasCustomEndpoint() {
		t.Error("expected HasCustomEndpoint to return true")
	}
	if !config.IsStaticAuth() {
		t.Error("expected IsStaticAuth to return true")
	}
	if config.IsProfileAuth() {
		t.Error("expected IsProfileAuth to return false")
	}

	config.AuthMethod = AuthMethodProfile
	if config.IsStaticAuth() {
		t.Error("expected IsStaticAuth to return false")
	}
	if !config.IsProfileAuth() {
		t.Error("expected IsProfileAuth to return true")
	}
}
