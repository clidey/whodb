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
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
)

// cachedAWSConfig holds a cached AWS configuration with usage tracking.
type cachedAWSConfig struct {
	config   aws.Config
	lastUsed time.Time
}

// configCacheTTL is how long unused configs stay in cache before cleanup.
// Longer than connection cache (5 min) since AWS configs are lightweight.
const configCacheTTL = 10 * time.Minute

// maxCachedConfigs limits cache size to prevent memory exhaustion.
const maxCachedConfigs = 100

var (
	// configCache stores cached AWS configs keyed by credential hash.
	configCache   = make(map[string]*cachedAWSConfig)
	configCacheMu sync.Mutex
	stopCleanup   chan struct{}
	cleanupOnce   sync.Once
)

func init() {
	startConfigCleanup()
}

// startConfigCleanup starts a background goroutine that periodically removes stale configs.
func startConfigCleanup() {
	cleanupOnce.Do(func() {
		stopCleanup = make(chan struct{})
		go func() {
			ticker := time.NewTicker(configCacheTTL / 5)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					cleanupStaleConfigs()
				case <-stopCleanup:
					return
				}
			}
		}()
	})
}

// cleanupStaleConfigs removes configs that haven't been used within the TTL.
func cleanupStaleConfigs() {
	log.Logger.Debug("Cleaning up stale AWS configs")
	staleThreshold := time.Now().Add(-configCacheTTL)

	configCacheMu.Lock()
	defer configCacheMu.Unlock()

	for key, cached := range configCache {
		if cached.lastUsed.Before(staleThreshold) {
			delete(configCache, key)
			log.Logger.Debug("Removed stale AWS config from cache")
		}
	}
}

// getConfigCacheKey generates a unique hash key for an AWS config.
// Uses SHA256 to avoid exposing raw credentials in memory.
// codeql[go/weak-crypto-algorithm]: SHA256 is intentional for cache key generation, not used for password storage
func getConfigCacheKey(creds *engine.Credentials) string {
	parts := []string{
		"aws",
		creds.Hostname, // Region
		creds.Username, // Access Key ID
		creds.Password, // Secret Access Key
		strconv.FormatBool(creds.IsProfile),
	}
	if creds.AccessToken != nil {
		parts = append(parts, *creds.AccessToken)
	}
	if creds.Id != nil {
		parts = append(parts, *creds.Id)
	}
	for _, adv := range creds.Advanced {
		parts = append(parts, adv.Key, adv.Value)
	}
	data := strings.Join(parts, "\x00")
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// shortKey returns first 8 chars of cache key for logging.
func shortKey(key string) string {
	if len(key) > 8 {
		return key[:8]
	}
	return key
}

// configIdentifier returns a short identifier for logging (region:authMethod).
func configIdentifier(creds *engine.Credentials) string {
	authMethod := "default"
	for _, adv := range creds.Advanced {
		if adv.Key == AdvancedKeyAuthMethod && adv.Value != "" {
			authMethod = adv.Value
			break
		}
	}
	return creds.Hostname + ":" + authMethod
}

// GetOrCreateConfig retrieves a cached AWS config or creates a new one.
// This is the primary entry point for cached config access.
func GetOrCreateConfig(ctx context.Context, creds *engine.Credentials) (aws.Config, error) {
	configID := configIdentifier(creds)
	key := getConfigCacheKey(creds)
	l := log.Logger.WithFields(map[string]any{"config_id": configID, "cache_key": shortKey(key)})

	// Check cache first (with lock)
	configCacheMu.Lock()
	if cached, found := configCache[key]; found {
		cached.lastUsed = time.Now()
		cfg := cached.config
		configCacheMu.Unlock()
		l.Debug("AWS config cache HIT")
		return cfg, nil
	}
	configCacheMu.Unlock()

	// Create new config (without lock to avoid blocking)
	l.Debug("Creating NEW AWS config")
	createStart := time.Now()
	cfg, err := LoadAWSConfig(ctx, creds)
	if err != nil {
		l.WithFields(map[string]any{
			"duration_ms": time.Since(createStart).Milliseconds(),
			"error":       err.Error(),
		}).Error("Failed to create AWS config")
		return aws.Config{}, err
	}
	l.WithField("duration_ms", time.Since(createStart).Milliseconds()).Debug("AWS config created successfully")

	// Store in cache (with lock)
	configCacheMu.Lock()
	defer configCacheMu.Unlock()

	// Check if another goroutine created a config while we were creating ours
	if cached, found := configCache[key]; found {
		cached.lastUsed = time.Now()
		l.Debug("Using AWS config created by another goroutine")
		return cached.config, nil
	}

	configCache[key] = &cachedAWSConfig{
		config:   cfg,
		lastUsed: time.Now(),
	}

	// Evict oldest config if we exceed the limit
	if len(configCache) > maxCachedConfigs {
		evictOldestConfig(key)
	}

	return cfg, nil
}

// evictOldestConfig removes the oldest config to stay under maxCachedConfigs.
// Must be called while holding configCacheMu.
func evictOldestConfig(excludeKey string) {
	var oldestKey string
	var oldestTime = time.Now().Add(time.Hour) // future time as initial value

	for key, cached := range configCache {
		if key == excludeKey {
			continue
		}
		if cached.lastUsed.Before(oldestTime) {
			oldestTime = cached.lastUsed
			oldestKey = key
		}
	}

	if oldestKey != "" {
		delete(configCache, oldestKey)
		log.Logger.WithField("cache_key", shortKey(oldestKey)).Debug("Evicted oldest AWS config to stay under limit")
	}
}

// RemoveConfig removes a specific config from cache.
// Call this when a connection fails or credentials are invalidated.
func RemoveConfig(creds *engine.Credentials) {
	configID := configIdentifier(creds)
	key := getConfigCacheKey(creds)
	l := log.Logger.WithFields(map[string]any{"config_id": configID, "cache_key": shortKey(key)})

	configCacheMu.Lock()
	if _, found := configCache[key]; found {
		delete(configCache, key)
		configCacheMu.Unlock()
		l.Debug("AWS config removed from cache")
		return
	}
	configCacheMu.Unlock()
	l.Debug("AWS config not found in cache")
}

// CloseAllConfigs clears the entire config cache.
// Call this on application shutdown.
func CloseAllConfigs(_ context.Context) {
	l := log.Logger.WithField("phase", "shutdown")
	l.Info("CloseAllConfigs called, stopping cleanup goroutine")

	// Stop the background cleanup goroutine
	if stopCleanup != nil {
		close(stopCleanup)
	}

	// Clear all cached configs
	configCacheMu.Lock()
	configCount := len(configCache)
	l.WithField("config_count", configCount).Info("Clearing AWS config cache")
	for key := range configCache {
		delete(configCache, key)
	}
	configCacheMu.Unlock()
	l.Info("AWS config cache cleared")
}

// CacheSize returns the current number of cached configs.
// Useful for monitoring and testing.
func CacheSize() int {
	configCacheMu.Lock()
	defer configCacheMu.Unlock()
	return len(configCache)
}
