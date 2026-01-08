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
	"os"
	"path/filepath"
	"testing"
)

func TestParseINIFile_Credentials(t *testing.T) {
	// Create a temporary credentials file
	tempDir := t.TempDir()
	credPath := filepath.Join(tempDir, "credentials")

	content := `[default]
aws_access_key_id = AKIAIOSFODNN7EXAMPLE
aws_secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY

[production]
aws_access_key_id = AKIAI44QH8DHBEXAMPLE
aws_secret_access_key = je7MtGbClwBF/2Zp9Utk/h3yCo8nvbEXAMPLEKEY

[staging]
aws_access_key_id = AKIAISTAGING
aws_secret_access_key = stagingsecret
`
	if err := os.WriteFile(credPath, []byte(content), 0600); err != nil {
		t.Fatalf("Failed to write test credentials file: %v", err)
	}

	profiles, err := parseINIFile(credPath, "credentials", false)
	if err != nil {
		t.Fatalf("parseINIFile failed: %v", err)
	}

	if len(profiles) != 3 {
		t.Errorf("Expected 3 profiles, got %d", len(profiles))
	}

	// Check default profile
	if def, ok := profiles["default"]; !ok {
		t.Error("Expected 'default' profile")
	} else {
		if !def.IsDefault {
			t.Error("Expected default profile to have IsDefault=true")
		}
		if def.Source != "credentials" {
			t.Errorf("Expected source 'credentials', got '%s'", def.Source)
		}
	}

	// Check production profile
	if prod, ok := profiles["production"]; !ok {
		t.Error("Expected 'production' profile")
	} else {
		if prod.IsDefault {
			t.Error("Expected production profile to have IsDefault=false")
		}
	}
}

func TestParseINIFile_Config(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config")

	content := `[default]
region = us-east-1
output = json

[profile production]
region = us-west-2
output = json

[profile staging]
region = eu-west-1
`
	if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	profiles, err := parseINIFile(configPath, "config", true)
	if err != nil {
		t.Fatalf("parseINIFile failed: %v", err)
	}

	if len(profiles) != 3 {
		t.Errorf("Expected 3 profiles, got %d", len(profiles))
	}

	// Check default profile region
	if def, ok := profiles["default"]; !ok {
		t.Error("Expected 'default' profile")
	} else {
		if def.Region != "us-east-1" {
			t.Errorf("Expected region 'us-east-1', got '%s'", def.Region)
		}
	}

	// Check production profile region
	if prod, ok := profiles["production"]; !ok {
		t.Error("Expected 'production' profile")
	} else {
		if prod.Region != "us-west-2" {
			t.Errorf("Expected region 'us-west-2', got '%s'", prod.Region)
		}
	}
}

func TestParseINIFile_WithComments(t *testing.T) {
	tempDir := t.TempDir()
	credPath := filepath.Join(tempDir, "credentials")

	content := `# This is a comment
[default]
aws_access_key_id = AKIAIOSFODNN7EXAMPLE
; This is also a comment
aws_secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
`
	if err := os.WriteFile(credPath, []byte(content), 0600); err != nil {
		t.Fatalf("Failed to write test credentials file: %v", err)
	}

	profiles, err := parseINIFile(credPath, "credentials", false)
	if err != nil {
		t.Fatalf("parseINIFile failed: %v", err)
	}

	if len(profiles) != 1 {
		t.Errorf("Expected 1 profile, got %d", len(profiles))
	}
}

func TestHasEnvCredentials(t *testing.T) {
	// Save current env
	oldAccessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	oldSecretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	defer func() {
		os.Setenv("AWS_ACCESS_KEY_ID", oldAccessKey)
		os.Setenv("AWS_SECRET_ACCESS_KEY", oldSecretKey)
	}()

	// Test with no credentials
	os.Unsetenv("AWS_ACCESS_KEY_ID")
	os.Unsetenv("AWS_SECRET_ACCESS_KEY")
	if hasEnvCredentials() {
		t.Error("Expected hasEnvCredentials to return false when no env vars set")
	}

	// Test with only access key
	os.Setenv("AWS_ACCESS_KEY_ID", "test")
	os.Unsetenv("AWS_SECRET_ACCESS_KEY")
	if hasEnvCredentials() {
		t.Error("Expected hasEnvCredentials to return false with only access key")
	}

	// Test with both credentials
	os.Setenv("AWS_ACCESS_KEY_ID", "test")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "secret")
	if !hasEnvCredentials() {
		t.Error("Expected hasEnvCredentials to return true when both env vars set")
	}
}

func TestDiscoverLocalProfiles_NoFiles(t *testing.T) {
	// Save and clear relevant env vars
	oldConfigFile := os.Getenv("AWS_CONFIG_FILE")
	oldCredsFile := os.Getenv("AWS_SHARED_CREDENTIALS_FILE")
	oldAccessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	oldSecretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	defer func() {
		os.Setenv("AWS_CONFIG_FILE", oldConfigFile)
		os.Setenv("AWS_SHARED_CREDENTIALS_FILE", oldCredsFile)
		os.Setenv("AWS_ACCESS_KEY_ID", oldAccessKey)
		os.Setenv("AWS_SECRET_ACCESS_KEY", oldSecretKey)
	}()

	// Point to non-existent directory
	tempDir := t.TempDir()
	os.Setenv("AWS_CONFIG_FILE", filepath.Join(tempDir, "nonexistent", "config"))
	os.Setenv("AWS_SHARED_CREDENTIALS_FILE", filepath.Join(tempDir, "nonexistent", "credentials"))
	os.Unsetenv("AWS_ACCESS_KEY_ID")
	os.Unsetenv("AWS_SECRET_ACCESS_KEY")

	profiles, err := DiscoverLocalProfiles()
	if err != nil {
		t.Fatalf("DiscoverLocalProfiles failed: %v", err)
	}

	// Should return empty list, not error
	if len(profiles) != 0 {
		t.Errorf("Expected 0 profiles, got %d", len(profiles))
	}
}
