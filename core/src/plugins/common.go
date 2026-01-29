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
	"time"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/env"
	"github.com/clidey/whodb/core/src/log"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// cachedConnection holds a cached GORM database instance.
type cachedConnection struct {
	db       *gorm.DB
	lastUsed time.Time
}

// connectionCacheTTL is how long unused connections stay in cache before cleanup.
const connectionCacheTTL = 5 * time.Minute

// maxCachedConnections limits cache size to prevent memory exhaustion.
const maxCachedConnections = 50

var (
	// connectionCache stores cached GORM instances keyed by config hash.
	connectionCache   = make(map[string]*cachedConnection)
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
	log.Logger.Debug("cleaning up stale connections")
	staleThreshold := time.Now().Add(-connectionCacheTTL)

	connectionCacheMu.Lock()
	defer connectionCacheMu.Unlock()

	for key, cached := range connectionCache {
		if cached.lastUsed.Before(staleThreshold) {
			delete(connectionCache, key)
			closeGormDB(cached.db)
			log.Logger.Debug("Closed stale database connection")
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
// codeql[go/weak-crypto-algorithm]: SHA256 is intentional for cache key generation, not used for password storage
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
func RemoveConnection(config *engine.PluginConfig) {
	connID := connIdentifier(config)
	key := getConnectionCacheKey(config)
	l := log.Logger.WithFields(map[string]any{"conn_id": connID, "cache_key": shortKey(key)})
	l.Debug("RemoveConnection called")

	connectionCacheMu.Lock()
	cached, found := connectionCache[key]
	if found {
		delete(connectionCache, key)
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
	l := log.Logger.WithField("phase", "shutdown")
	l.Info("CloseAllConnections called, stopping cleanup goroutine")

	// Stop the background cleanup goroutine
	close(stopCleanup)

	// Close all cached connections
	connectionCacheMu.Lock()
	connCount := len(connectionCache)
	l.WithField("conn_count", connCount).Info("Closing cached connections")
	for key, cached := range connectionCache {
		l.WithField("cache_key", shortKey(key)).Debug("Closing connection")
		closeGormDB(cached.db)
		delete(connectionCache, key)
	}
	connectionCacheMu.Unlock()
	l.Info("All connections closed")
}

// getOrCreateConnection retrieves a cached connection or creates a new one.
// Validates cached connections with a ping before returning.
// Connection creation happens WITHOUT holding the lock to avoid blocking other operations.
func getOrCreateConnection(config *engine.PluginConfig, createDB DBCreationFunc) (*gorm.DB, error) {
	connID := connIdentifier(config)
	key := getConnectionCacheKey(config)
	l := log.Logger.WithFields(map[string]any{"conn_id": connID, "cache_key": shortKey(key)})

	// First, check cache (with lock)
	connectionCacheMu.Lock()
	if cached, found := connectionCache[key]; found && cached != nil && cached.db != nil {
		cached.lastUsed = time.Now()
		db := cached.db
		connectionCacheMu.Unlock()

		// Validate connection with ping (outside lock to avoid blocking)
		if sqlDB, err := db.DB(); err == nil && sqlDB != nil {
			if err := sqlDB.Ping(); err == nil {
				l.Debug("Cache HIT - connection alive")
				return db, nil
			}
			l.WithError(err).Debug("Ping failed, will create new connection")
		}

		// Connection is stale - remove from cache
		connectionCacheMu.Lock()
		if existingCached, stillExists := connectionCache[key]; stillExists && existingCached.db == db {
			delete(connectionCache, key)
		}
		connectionCacheMu.Unlock()
	} else {
		connectionCacheMu.Unlock()
	}

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
	if cached, found := connectionCache[key]; found && cached != nil && cached.db != nil {
		// Another goroutine won the race - use their connection, close ours
		if sqlDB, err := cached.db.DB(); err == nil && sqlDB != nil {
			if err := sqlDB.Ping(); err == nil {
				cached.lastUsed = time.Now()
				l.Debug("Race: using connection created by another goroutine")
				// Close the connection we just created since we won't use it
				closeGormDB(db)
				return cached.db, nil
			}
		}
		// Their connection is stale - remove it, use ours
		delete(connectionCache, key)
	}

	connectionCache[key] = &cachedConnection{
		db:       db,
		lastUsed: time.Now(),
	}

	// Evict oldest connection if we exceed the limit
	if len(connectionCache) > maxCachedConnections {
		evictOldestConnection(key)
	}

	return db, nil
}

// evictOldestConnection removes the oldest connection to stay under maxCachedConnections.
// Must be called while holding connectionCacheMu.
func evictOldestConnection(excludeKey string) {
	var oldestKey string
	var oldestTime = time.Now().Add(time.Hour) // future time as initial value

	for key, cached := range connectionCache {
		if key == excludeKey {
			continue
		}
		if cached.lastUsed.Before(oldestTime) {
			oldestTime = cached.lastUsed
			oldestKey = key
		}
	}

	if oldestKey != "" {
		cached := connectionCache[oldestKey]
		delete(connectionCache, oldestKey)
		closeGormDB(cached.db)
		log.Logger.WithField("cache_key", shortKey(oldestKey)).Debug("Evicted oldest connection to stay under limit")
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
// The underlying sql.DB handles connection pooling internally.
// If config.Transaction is set (as a *gorm.DB), it will be used instead of creating a new connection
func WithConnection[T any](config *engine.PluginConfig, DB DBCreationFunc, operation DBOperation[T]) (T, error) {
	// Check if we're operating within a transaction
	if config != nil && config.Transaction != nil {
		if tx, ok := config.Transaction.(*gorm.DB); ok {
			return operation(tx)
		}
	}

	db, err := getOrCreateConnection(config, DB)
	if err != nil {
		log.Logger.WithFields(map[string]any{
			"conn_id": connIdentifier(config),
			"error":   err.Error(),
		}).Error("WithConnection FAILED to get connection")
		var zero T
		return zero, err
	}

	if db == nil {
		var zero T
		return zero, fmt.Errorf("internal error: nil database connection")
	}

	return operation(db)
}
