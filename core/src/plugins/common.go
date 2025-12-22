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

package plugins

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/env"
	"github.com/clidey/whodb/core/src/log"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// cachedConnection holds a cached database connection with reference counting.
// Reference counting ensures connections aren't closed while operations are in progress.
type cachedConnection struct {
	db       *gorm.DB
	lastUsed int64 // atomic: Unix timestamp of last use
	refCount int32 // atomic: number of active operations using this connection
}

// connectionCacheTTL is how long unused connections stay in cache before cleanup.
const connectionCacheTTL = 5 * time.Minute

// maxCachedConnections limits cache size to prevent memory exhaustion.
const maxCachedConnections = 50

var (
	// connectionCache stores cached database connections keyed by config hash.
	connectionCache   = make(map[string]*cachedConnection)
	connectionCacheMu sync.Mutex
	// activeOperations tracks in-flight database operations for graceful shutdown.
	activeOperations sync.WaitGroup
	// stopCleanup signals the background cleanup goroutine to stop.
	stopCleanup = make(chan struct{})
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

// cleanupStaleConnections removes connections that haven't been used within the TTL
// and have no active operations.
func cleanupStaleConnections() {
	log.Logger.Debug("cleaning up stale connections")
	staleThreshold := time.Now().Unix() - int64(connectionCacheTTL.Seconds())

	connectionCacheMu.Lock()
	defer connectionCacheMu.Unlock()

	for key, cached := range connectionCache {
		lastUsed := atomic.LoadInt64(&cached.lastUsed)
		refCount := atomic.LoadInt32(&cached.refCount)

		if lastUsed < staleThreshold && refCount == 0 {
			// Safe to close - stale and not in use
			delete(connectionCache, key)
			closeConnection(cached)
			log.Logger.Debug("Closed stale database connection")
		}
	}
}

// closeConnection closes the underlying database connection.
func closeConnection(cached *cachedConnection) {
	if cached == nil || cached.db == nil {
		return
	}
	if sqlDB, err := cached.db.DB(); err == nil {
		if err := sqlDB.Close(); err != nil {
			log.Logger.WithError(err).Error("failed to close db connection")
		}
	}
}

// connIdentifier returns a short identifier for logging (type:host:db)
func connIdentifier(config *engine.PluginConfig) string {
	return fmt.Sprintf("%s:%s:%s", config.Credentials.Type, config.Credentials.Hostname, config.Credentials.Database)
}

// shortKey returns first 8 chars of cache key for logging
func shortKey(key string) string {
	if len(key) > 8 {
		return key[:8]
	}
	return key
}

// getConnectionCacheKey generates a unique hash key for a connection config.
// Uses SHA256 to avoid exposing raw credentials in memory.
func getConnectionCacheKey(config *engine.PluginConfig) string {
	parts := []string{
		config.Credentials.Type,
		config.Credentials.Hostname,
		config.Credentials.Username,
		config.Credentials.Password,
		config.Credentials.Database,
		strconv.FormatBool(config.Credentials.IsProfile),
	}
	if config.Credentials.Id != nil {
		parts = append(parts, *config.Credentials.Id)
	}
	for _, adv := range config.Credentials.Advanced {
		parts = append(parts, adv.Key, adv.Value)
	}
	data := strings.Join(parts, "\x00")
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// RemoveConnection removes a specific connection from cache and closes it (call on logout).
// If the connection is currently in use (refCount > 0), it stays in cache
// and the cleanup goroutine will close it once it becomes idle.
func RemoveConnection(config *engine.PluginConfig) {
	connID := connIdentifier(config)
	key := getConnectionCacheKey(config)
	l := log.Logger.WithFields(map[string]any{"conn_id": connID, "cache_key": shortKey(key)})
	l.Debug("RemoveConnection called")

	connectionCacheMu.Lock()
	cached, found := connectionCache[key]
	if found {
		refCount := atomic.LoadInt32(&cached.refCount)
		if refCount == 0 {
			delete(connectionCache, key)
			connectionCacheMu.Unlock()
			l.Debug("Connection closed (refCount=0)")
			closeConnection(cached)
			return
		}
		l.WithField("ref_count", refCount).Debug("Connection in use, deferring close")
	} else {
		l.Debug("Connection not found in cache")
	}
	connectionCacheMu.Unlock()
	// If refCount > 0, leave it - cleanup goroutine will close it when idle
}

// CloseAllConnections closes all cached connections (call on shutdown).
// It waits for in-flight operations to complete, respecting the context deadline.
func CloseAllConnections(ctx context.Context) {
	l := log.Logger.WithField("phase", "shutdown")
	l.Info("CloseAllConnections called, stopping cleanup goroutine")
	// Stop the background cleanup goroutine
	close(stopCleanup)

	// Wait for in-flight operations to complete
	l.Info("Waiting for in-flight operations to complete")
	done := make(chan struct{})
	go func() {
		activeOperations.Wait()
		close(done)
	}()

	select {
	case <-done:
		l.Info("All database operations completed gracefully")
	case <-ctx.Done():
		l.Warn("Timeout waiting for database operations, force closing connections")
	}

	// Close all cached connections
	connectionCacheMu.Lock()
	connCount := len(connectionCache)
	l.WithField("conn_count", connCount).Info("Closing cached connections")
	for key, cached := range connectionCache {
		refCount := atomic.LoadInt32(&cached.refCount)
		l.WithFields(map[string]any{"cache_key": shortKey(key), "ref_count": refCount}).Debug("Closing connection")
		closeConnection(cached)
		delete(connectionCache, key)
	}
	connectionCacheMu.Unlock()
	l.Info("All connections closed")
}

// getOrCreateConnection retrieves a cached connection or creates a new one.
// Returns the cachedConnection wrapper with refCount already incremented to prevent
// race conditions where the connection could be closed before the caller uses it.
// The caller MUST decrement refCount when done (handled by WithConnection's defer).
func getOrCreateConnection(config *engine.PluginConfig, createDB DBCreationFunc) (*cachedConnection, error) {
	connID := connIdentifier(config)
	key := getConnectionCacheKey(config)
	l := log.Logger.WithFields(map[string]any{"conn_id": connID, "cache_key": shortKey(key)})
	l.Debug("getOrCreateConnection called")

	connectionCacheMu.Lock()

	// Check if we have a cached connection
	if cached, found := connectionCache[key]; found && cached != nil {
		// Increment refCount while holding lock to prevent race with cleanup/RemoveConnection
		refCount := atomic.AddInt32(&cached.refCount, 1)
		atomic.StoreInt64(&cached.lastUsed, time.Now().Unix())
		connectionCacheMu.Unlock()

		l.WithField("ref_count", refCount).Debug("Found in cache, pinging to verify")
		if sqlDB, err := cached.db.DB(); err == nil {
			pingStart := time.Now()
			if err := sqlDB.Ping(); err == nil {
				l.WithField("ping_ms", time.Since(pingStart).Milliseconds()).Debug("Cache HIT - connection alive")
				return cached, nil
			}
			l.WithFields(map[string]any{"ping_ms": time.Since(pingStart).Milliseconds(), "error": err.Error()}).Debug("Ping FAILED, removing stale connection")
			// Ping failed - connection is stale, decrement refCount and remove it
			atomic.AddInt32(&cached.refCount, -1)
			connectionCacheMu.Lock()
			delete(connectionCache, key)
			connectionCacheMu.Unlock()
			closeConnection(cached)
		} else {
			// Failed to get underlying DB, decrement refCount
			atomic.AddInt32(&cached.refCount, -1)
		}
	} else {
		connectionCacheMu.Unlock()
		l.Debug("Cache MISS")
	}

	// Need to create new connection - acquire lock for creation
	connectionCacheMu.Lock()
	defer connectionCacheMu.Unlock()

	// Double-check after acquiring lock in case another goroutine created conn
	if cached, found := connectionCache[key]; found && cached != nil {
		l.Debug("Double-check: found in cache after lock, pinging")
		if sqlDB, err := cached.db.DB(); err == nil {
			if err := sqlDB.Ping(); err == nil {
				// Increment refCount while holding lock
				refCount := atomic.AddInt32(&cached.refCount, 1)
				atomic.StoreInt64(&cached.lastUsed, time.Now().Unix())
				l.WithField("ref_count", refCount).Debug("Double-check: connection alive, using cached")
				return cached, nil
			}
			l.Debug("Double-check: ping failed, removing stale")
			delete(connectionCache, key)
			closeConnection(cached)
		}
	}

	l.Debug("Creating NEW database connection")
	createStart := time.Now()
	db, err := createDB(config)
	if err != nil {
		l.WithFields(map[string]any{"duration_ms": time.Since(createStart).Milliseconds(), "error": err.Error()}).Error("Failed to create connection")
		return nil, err
	}
	l.WithField("duration_ms", time.Since(createStart).Milliseconds()).Info("Connection created successfully")

	cached := &cachedConnection{
		db:       db,
		lastUsed: time.Now().Unix(),
		refCount: 1, // Start with refCount=1 since caller will use it
	}

	connectionCache[key] = cached
	l.Debug("Connection cached with refCount=1")

	// Evict oldest idle connections if we exceed the limit
	if len(connectionCache) > maxCachedConnections {
		evictOldestIdleConnection(key)
	}

	return cached, nil
}

// evictOldestIdleConnection removes the oldest idle connection to stay under maxCachedConnections.
// Must be called while holding connectionCacheMu.
func evictOldestIdleConnection(excludeKey string) {
	var oldestKey string
	var oldestTime int64 = time.Now().Unix() + 1 // future time as initial value

	for key, cached := range connectionCache {
		if key == excludeKey {
			continue // don't evict the connection we just added
		}
		refCount := atomic.LoadInt32(&cached.refCount)
		if refCount > 0 {
			continue // don't evict connections in use
		}
		lastUsed := atomic.LoadInt64(&cached.lastUsed)
		if lastUsed < oldestTime {
			oldestTime = lastUsed
			oldestKey = key
		}
	}

	if oldestKey != "" {
		cached := connectionCache[oldestKey]
		delete(connectionCache, oldestKey)
		closeConnection(cached)
		log.Logger.WithField("cache_key", shortKey(oldestKey)).Debug("Evicted oldest idle connection to stay under limit")
	}
}

// SortDirection indicates ascending or descending sort order.
type SortDirection int

const (
	Up   SortDirection = iota // ASC
	Down                      // DESC
)

// Sort represents a column sort condition with direction.
type Sort struct {
	Column    string
	Direction SortDirection
}

// SearchCondition represents a WHERE clause condition that can be atomic, AND, or OR.
type SearchCondition struct {
	And    *AndCondition
	Or     *OrCondition
	Atomic *AtomicCondition
}

// AndCondition represents multiple conditions joined with AND.
type AndCondition struct {
	Conditions []SearchCondition
}

// OrCondition represents multiple conditions joined with OR.
type OrCondition struct {
	Conditions []SearchCondition
}

// AtomicCondition represents a single comparison condition (e.g., column = value).
type AtomicCondition struct {
	Key        string
	Operator   string
	Value      any
	ColumnType string
}

// DBOperation is a function that performs database operations with a GORM connection.
type DBOperation[T any] func(*gorm.DB) (T, error)

// DBCreationFunc is a function that creates a new GORM database connection.
type DBCreationFunc func(pluginConfig *engine.PluginConfig) (*gorm.DB, error)

// ConfigureConnectionPool sets recommended connection pool settings for database connections.
// This should be called after opening a GORM connection to ensure proper pool management.
// Settings are tuned for long-running server applications with connection caching.
func ConfigureConnectionPool(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}

	// MaxOpenConns: Limit concurrent connections to prevent overwhelming the DB server.
	// 10 is reasonable for most use cases - adjust based on DB server capacity.
	sqlDB.SetMaxOpenConns(10)

	// MaxIdleConns: Keep idle connections ready for reuse.
	// Should be <= MaxOpenConns. Higher values reduce connection creation overhead.
	sqlDB.SetMaxIdleConns(5)

	// ConnMaxLifetime: Force connection refresh to handle server-side timeouts.
	// Most DBs have idle timeouts (MySQL: wait_timeout=8h, PostgreSQL: idle_session_timeout).
	// 30 minutes is conservative and works for most databases.
	sqlDB.SetConnMaxLifetime(30 * time.Minute)

	// ConnMaxIdleTime: Close idle connections faster than lifetime.
	// Helps detect and remove half-open connections that the server closed.
	// 5 minutes matches our cache cleanup TTL.
	sqlDB.SetConnMaxIdleTime(5 * time.Minute)

	return nil
}

// GetGormLogConfig returns the GORM logger level based on the environment log level setting.
func GetGormLogConfig() logger.LogLevel {
	switch env.LogLevel {
	case "warning":
		return logger.Warn
	case "error":
		return logger.Error
	default:
		return logger.Silent
	}
}

// WithConnection manages the database connection lifecycle for an operation.
// Connections are cached and reused across operations to prevent connection exhaustion.
// Reference counting ensures connections aren't closed while operations are in progress.
// Stale connections are automatically cleaned up by a background goroutine.
func WithConnection[T any](config *engine.PluginConfig, DB DBCreationFunc, operation DBOperation[T]) (T, error) {
	connID := connIdentifier(config)
	opStart := time.Now()
	l := log.Logger.WithField("conn_id", connID)
	l.Debug("WithConnection START")

	cached, err := getOrCreateConnection(config, DB)
	if err != nil {
		l.WithField("error", err.Error()).Error("WithConnection FAILED to get connection")
		var zero T
		return zero, err
	}

	activeOperations.Add(1)
	// Note: refCount was already incremented in getOrCreateConnection while holding the lock
	// to prevent race conditions with cleanup. We only need to track activeOperations here.
	refCount := atomic.LoadInt32(&cached.refCount)
	l.WithField("ref_count", refCount).Debug("Operation started")

	defer func() {
		newRefCount := atomic.AddInt32(&cached.refCount, -1)
		activeOperations.Done()
		l.WithFields(map[string]any{"duration_ms": time.Since(opStart).Milliseconds(), "ref_count": newRefCount}).Debug("Operation DONE")
	}()

	return operation(cached.db)
}
