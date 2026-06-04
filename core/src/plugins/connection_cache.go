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

package plugins

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
)

// cachedConnection holds a cached GORM database instance.
type cachedConnection struct {
	db        *gorm.DB
	lastUsed  time.Time
	sslStatus *engine.SSLStatus
}

type connectionCacheBucket map[string]*cachedConnection

// connectionCacheTTL is how long unused connections stay in cache before cleanup.
const connectionCacheTTL = 5 * time.Minute

// maxCachedConnections limits cache size to prevent memory exhaustion.
const maxCachedConnections = 50

var (
	// connectionCache stores cached GORM instances keyed by non-password config fields,
	// with password-specific entries inside each bucket.
	connectionCache   = make(map[string]connectionCacheBucket)
	connectionCacheMu sync.Mutex
	stopCleanup       = make(chan struct{})
)

func init() {
	startConnectionCleanup()
}

// startConnectionCleanup starts a background goroutine that periodically removes stale connections.
func startConnectionCleanup() {
	go func() {
		ticker := time.NewTicker(connectionCacheTTL / 5)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				cleanupStaleConnections()
			case <-stopCleanup:
				return
			}
		}
	}()
}

// cleanupStaleConnections removes connections that haven't been used within the TTL.
func cleanupStaleConnections() {
	log.Debug("cleaning up stale connections")
	staleThreshold := time.Now().Add(-connectionCacheTTL)

	connectionCacheMu.Lock()
	defer connectionCacheMu.Unlock()

	for key, bucket := range connectionCache {
		for secret, cached := range bucket {
			if cached.lastUsed.Before(staleThreshold) {
				delete(bucket, secret)
				closeGormDB(cached.db)
				log.Debug("Closed stale database connection")
			}
		}
		if len(bucket) == 0 {
			delete(connectionCache, key)
		}
	}
}

// closeGormDB closes the underlying database connection of a GORM instance.
func closeGormDB(db *gorm.DB) {
	if db == nil {
		return
	}
	if sqlDB, err := db.DB(); err == nil {
		if err := sqlDB.Close(); err != nil {
			log.WithError(err).Error("failed to close db connection")
		}
	}
}

// connIdentifier returns a short identifier for logging (type:host:db)
func connIdentifier(config *engine.PluginConfig) string {
	return fmt.Sprintf("%s:%s:%s", config.Credentials.Type, config.Credentials.Hostname, config.Credentials.Database)
}

// shortKey returns a short digest of the cache key for logging.
func shortKey(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:4])
}

// getConnectionCacheKey generates a unique bucket key for non-password connection config fields.
func getConnectionCacheKey(config *engine.PluginConfig) string {
	parts := []string{
		config.Credentials.Type,
		config.Credentials.Hostname,
		config.Credentials.Username,
		config.Credentials.Database,
		strconv.FormatBool(config.Credentials.IsProfile),
	}
	if config.Credentials.Id != nil {
		parts = append(parts, *config.Credentials.Id)
	}
	for _, adv := range config.Credentials.Advanced {
		parts = append(parts, adv.Key, adv.Value)
	}
	return strings.Join(parts, "\x00")
}

func getConnectionCacheSecret(config *engine.PluginConfig) string {
	return config.Credentials.Password
}

func getCachedConnectionLocked(key string, secret string) (*cachedConnection, bool) {
	bucket, found := connectionCache[key]
	if !found {
		return nil, false
	}
	cached, found := bucket[secret]
	return cached, found
}

func setCachedConnectionLocked(key string, secret string, cached *cachedConnection) {
	bucket, found := connectionCache[key]
	if !found {
		bucket = make(connectionCacheBucket)
		connectionCache[key] = bucket
	}
	bucket[secret] = cached
}

func deleteCachedConnectionLocked(key string, secret string) *cachedConnection {
	bucket, found := connectionCache[key]
	if !found {
		return nil
	}
	cached, found := bucket[secret]
	if !found {
		return nil
	}
	delete(bucket, secret)
	if len(bucket) == 0 {
		delete(connectionCache, key)
	}
	return cached
}

func connectionCacheEntryCountLocked() int {
	count := 0
	for _, bucket := range connectionCache {
		count += len(bucket)
	}
	return count
}

// RemoveConnection removes a specific connection from cache and closes it (call on logout).
func RemoveConnection(config *engine.PluginConfig) {
	connID := connIdentifier(config)
	key := getConnectionCacheKey(config)
	secret := getConnectionCacheSecret(config)
	l := log.WithFields(map[string]any{"conn_id": connID, "cache_key": shortKey(key)})
	l.Debug("RemoveConnection called")

	connectionCacheMu.Lock()
	cached := deleteCachedConnectionLocked(key, secret)
	if cached != nil {
		connectionCacheMu.Unlock()
		closeGormDB(cached.db)
		l.Debug("Connection removed and closed")
		return
	}
	connectionCacheMu.Unlock()
	l.Debug("Connection not found in cache")
}

// CloseAllConnections closes all cached connections (call on shutdown).
func CloseAllConnections(_ context.Context) {
	l := log.WithField("phase", "shutdown")
	l.Info("CloseAllConnections called, stopping cleanup goroutine")

	// Stop the background cleanup goroutine
	close(stopCleanup)

	// Close all cached connections
	connectionCacheMu.Lock()
	connCount := connectionCacheEntryCountLocked()
	l.WithField("conn_count", connCount).Info("Closing cached connections")
	for key, bucket := range connectionCache {
		for secret, cached := range bucket {
			l.WithField("cache_key", shortKey(key)).Debug("Closing connection")
			closeGormDB(cached.db)
			delete(bucket, secret)
		}
		delete(connectionCache, key)
	}
	connectionCacheMu.Unlock()
	l.Info("All connections closed")
}

// getOrCreateConnection retrieves a cached connection or creates a new one.
// Connection creation happens WITHOUT holding the lock to avoid blocking other operations.
func getOrCreateConnection(config *engine.PluginConfig, createDB DBCreationFunc) (*gorm.DB, error) {
	connID := connIdentifier(config)
	key := getConnectionCacheKey(config)
	secret := getConnectionCacheSecret(config)
	l := log.WithFields(map[string]any{"conn_id": connID, "cache_key": shortKey(key)})

	// First, check cache (with lock)
	connectionCacheMu.Lock()
	if cached, found := getCachedConnectionLocked(key, secret); found && cached != nil && cached.db != nil {
		cached.lastUsed = time.Now()
		db := cached.db
		connectionCacheMu.Unlock()
		l.Debug("Cache HIT - reusing cached connection")
		return db, nil
	}
	connectionCacheMu.Unlock()

	// Create new connection WITHOUT holding the lock
	// This is critical - connection creation can take 30+ seconds for slow/failing DBs
	l.Debug("Creating NEW database connection")
	createStart := time.Now()
	db, err := createDB(config)
	if err != nil {
		l.WithFields(map[string]any{"duration_ms": time.Since(createStart).Milliseconds(), "error": err.Error()}).Error("Failed to create connection")
		return nil, err
	}
	l.WithField("duration_ms", time.Since(createStart).Milliseconds()).Info("Connection created successfully")

	// Store in cache (with lock)
	connectionCacheMu.Lock()
	defer connectionCacheMu.Unlock()

	// Check if another goroutine created a connection while we were creating ours
	if cached, found := getCachedConnectionLocked(key, secret); found && cached != nil && cached.db != nil {
		// Another goroutine won the race - use their connection, close ours.
		cached.lastUsed = time.Now()
		l.Debug("using connection created by another goroutine")
		closeGormDB(db)
		return cached.db, nil
	}

	setCachedConnectionLocked(key, secret, &cachedConnection{
		db:       db,
		lastUsed: time.Now(),
	})

	// Evict oldest connection if we exceed the limit
	if connectionCacheEntryCountLocked() > maxCachedConnections {
		evictOldestConnection(key, secret)
	}

	return db, nil
}

// evictOldestConnection removes the oldest connection to stay under maxCachedConnections.
// Must be called while holding connectionCacheMu.
func evictOldestConnection(excludeKey string, excludeSecret string) {
	var oldestKey string
	var oldestSecret string
	var oldestTime = time.Now().Add(time.Hour) // future time as initial value

	for key, bucket := range connectionCache {
		for secret, cached := range bucket {
			if key == excludeKey && secret == excludeSecret {
				continue
			}
			if cached.lastUsed.Before(oldestTime) {
				oldestTime = cached.lastUsed
				oldestKey = key
				oldestSecret = secret
			}
		}
	}

	if oldestKey != "" {
		cached := deleteCachedConnectionLocked(oldestKey, oldestSecret)
		closeGormDB(cached.db)
		log.WithField("cache_key", shortKey(oldestKey)).Debug("Evicted oldest connection to stay under limit")
	}
}
