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

func TestGetConfigCacheKey_Deterministic(t *testing.T) {
	creds := &engine.Credentials{
		Hostname: "us-west-2",
		Username: "AKIAIOSFODNN7EXAMPLE",
		Password: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		Advanced: []engine.Record{
			{Key: AdvancedKeyAuthMethod, Value: "static"},
		},
	}

	key1 := getConfigCacheKey(creds)
	key2 := getConfigCacheKey(creds)

	if key1 != key2 {
		t.Error("expected cache keys to be deterministic")
	}
}

func TestGetConfigCacheKey_DifferentCredentials(t *testing.T) {
	creds1 := &engine.Credentials{
		Hostname: "us-west-2",
		Username: "AKIAIOSFODNN7EXAMPLE",
		Password: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
	}

	creds2 := &engine.Credentials{
		Hostname: "us-east-1",
		Username: "AKIAIOSFODNN7EXAMPLE",
		Password: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
	}

	key1 := getConfigCacheKey(creds1)
	key2 := getConfigCacheKey(creds2)

	if key1 == key2 {
		t.Error("expected different cache keys for different regions")
	}
}

func TestGetConfigCacheKey_AdvancedRecords(t *testing.T) {
	creds1 := &engine.Credentials{
		Hostname: "us-west-2",
		Username: "AKIAIOSFODNN7EXAMPLE",
		Password: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		Advanced: []engine.Record{
			{Key: AdvancedKeyAuthMethod, Value: "static"},
		},
	}

	creds2 := &engine.Credentials{
		Hostname: "us-west-2",
		Username: "AKIAIOSFODNN7EXAMPLE",
		Password: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		Advanced: []engine.Record{
			{Key: AdvancedKeyAuthMethod, Value: "profile"},
			{Key: AdvancedKeyProfileName, Value: "production"},
		},
	}

	key1 := getConfigCacheKey(creds1)
	key2 := getConfigCacheKey(creds2)

	if key1 == key2 {
		t.Error("expected different cache keys for different advanced records")
	}
}

func TestGetConfigCacheKey_WithId(t *testing.T) {
	id1 := "conn-1"
	id2 := "conn-2"

	creds1 := &engine.Credentials{
		Id:       &id1,
		Hostname: "us-west-2",
		Username: "AKIAIOSFODNN7EXAMPLE",
		Password: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
	}

	creds2 := &engine.Credentials{
		Id:       &id2,
		Hostname: "us-west-2",
		Username: "AKIAIOSFODNN7EXAMPLE",
		Password: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
	}

	key1 := getConfigCacheKey(creds1)
	key2 := getConfigCacheKey(creds2)

	if key1 == key2 {
		t.Error("expected different cache keys for different connection IDs")
	}
}

func TestGetConfigCacheKey_WithAccessToken(t *testing.T) {
	token1 := "token1"
	token2 := "token2"

	creds1 := &engine.Credentials{
		Hostname:    "us-west-2",
		Username:    "AKIAIOSFODNN7EXAMPLE",
		Password:    "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		AccessToken: &token1,
	}

	creds2 := &engine.Credentials{
		Hostname:    "us-west-2",
		Username:    "AKIAIOSFODNN7EXAMPLE",
		Password:    "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		AccessToken: &token2,
	}

	key1 := getConfigCacheKey(creds1)
	key2 := getConfigCacheKey(creds2)

	if key1 == key2 {
		t.Error("expected different cache keys for different access tokens")
	}
}

func TestShortKey(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"abcdefghij", "abcdefgh"},
		{"abcd", "abcd"},
		{"", ""},
		{"12345678901234567890", "12345678"},
	}

	for _, tc := range testCases {
		result := shortKey(tc.input)
		if result != tc.expected {
			t.Errorf("shortKey(%s) = %s, expected %s", tc.input, result, tc.expected)
		}
	}
}

func TestConfigIdentifier(t *testing.T) {
	creds := &engine.Credentials{
		Hostname: "us-west-2",
		Advanced: []engine.Record{
			{Key: AdvancedKeyAuthMethod, Value: "static"},
		},
	}

	id := configIdentifier(creds)
	if id != "us-west-2:static" {
		t.Errorf("expected 'us-west-2:static', got %s", id)
	}

	// Without auth method
	creds = &engine.Credentials{
		Hostname: "eu-west-1",
	}

	id = configIdentifier(creds)
	if id != "eu-west-1:default" {
		t.Errorf("expected 'eu-west-1:default', got %s", id)
	}
}

func TestCacheSize(t *testing.T) {
	// Clear cache first
	configCacheMu.Lock()
	for k := range configCache {
		delete(configCache, k)
	}
	configCacheMu.Unlock()

	// Initial size should be 0
	if CacheSize() != 0 {
		t.Errorf("expected initial cache size 0, got %d", CacheSize())
	}
}

func TestRemoveConfig(t *testing.T) {
	creds := &engine.Credentials{
		Hostname: "test-region-remove",
		Username: "test",
		Password: "test",
	}

	// Add to cache manually
	key := getConfigCacheKey(creds)
	configCacheMu.Lock()
	configCache[key] = &cachedAWSConfig{}
	configCacheMu.Unlock()

	// Verify it's in cache
	configCacheMu.Lock()
	_, found := configCache[key]
	configCacheMu.Unlock()
	if !found {
		t.Fatal("config should be in cache before removal")
	}

	// Remove it
	RemoveConfig(creds)

	// Verify it's removed
	configCacheMu.Lock()
	_, found = configCache[key]
	configCacheMu.Unlock()
	if found {
		t.Error("config should not be in cache after removal")
	}
}

func TestRemoveConfig_NotFound(t *testing.T) {
	creds := &engine.Credentials{
		Hostname: "non-existent-region",
		Username: "test",
		Password: "test",
	}

	// Should not panic when removing non-existent config
	RemoveConfig(creds)
}
