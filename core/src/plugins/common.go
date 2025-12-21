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
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/env"
	"github.com/clidey/whodb/core/src/log"
	"github.com/dgraph-io/ristretto/v2"
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
	connectionCache   *ristretto.Cache[string, *cachedConnection]
	connectionCacheMu sync.Mutex
	// activeConnections tracks all connections for explicit cleanup on shutdown.
	// Ristretto doesn't provide iteration, so we maintain this separately.
	activeConnections   = make(map[string]*cachedConnection)
	activeConnectionsMu sync.Mutex
	// activeOperations tracks in-flight database operations for graceful shutdown.
	activeOperations sync.WaitGroup
	// stopCleanup signals the background cleanup goroutine to stop.
	stopCleanup = make(chan struct{})
)

func init() {
	initConnectionCache()
	startConnectionCleanup()
}

// initConnectionCache sets up the ristretto cache for connections.
func initConnectionCache() {
	var err error
	connectionCache, err = ristretto.NewCache(&ristretto.Config[string, *cachedConnection]{
		NumCounters: maxCachedConnections * 10,
		MaxCost:     maxCachedConnections,
		BufferItems: 64,
	})
	if err != nil {
		log.Logger.WithError(err).Error("Failed to initialize connection cache")
	}
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
	log.Logger.Infof("cleaning up stale connections")
	staleThreshold := time.Now().Unix() - int64(connectionCacheTTL.Seconds())

	activeConnectionsMu.Lock()
	defer activeConnectionsMu.Unlock()

	for key, cached := range activeConnections {
		lastUsed := atomic.LoadInt64(&cached.lastUsed)
		refCount := atomic.LoadInt32(&cached.refCount)

		if lastUsed < staleThreshold && refCount == 0 {
			// Safe to close - stale and not in use
			if connectionCache != nil {
				connectionCache.Del(key)
			}
			delete(activeConnections, key)
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

// getConnectionCacheKey generates a unique hash key for a connection config.
// Uses SHA256 to avoid exposing raw credentials in memory.
func getConnectionCacheKey(config *engine.PluginConfig) string {
	log.Logger.Infof("getting connection cache key")
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
// If the connection is currently in use (refCount > 0), it stays in activeConnections
// and the cleanup goroutine will close it once it becomes idle.
func RemoveConnection(config *engine.PluginConfig) {
	log.Logger.Infof("removing connection")
	key := getConnectionCacheKey(config)

	// Remove from cache so no new operations use this connection
	if connectionCache != nil {
		connectionCache.Del(key)
	}

	// Only remove from map and close if not in use
	activeConnectionsMu.Lock()
	cached, found := activeConnections[key]
	if found && atomic.LoadInt32(&cached.refCount) == 0 {
		delete(activeConnections, key)
		activeConnectionsMu.Unlock()
		closeConnection(cached)
		return
	}
	activeConnectionsMu.Unlock()
	// If refCount > 0, leave it - cleanup goroutine will close it when idle
}

// CloseAllConnections closes all cached connections (call on shutdown).
// It waits for in-flight operations to complete, respecting the context deadline.
func CloseAllConnections(ctx context.Context) {
	// Stop the background cleanup goroutine
	log.Logger.Infof("closing all connections")
	close(stopCleanup)

	// Wait for in-flight operations to complete
	done := make(chan struct{})
	go func() {
		activeOperations.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Logger.Debug("All database operations completed gracefully")
	case <-ctx.Done():
		log.Logger.Warn("Timeout waiting for database operations, force closing connections")
	}

	// Clear the ristretto cache
	if connectionCache != nil {
		connectionCache.Clear()
		connectionCache.Wait()
	}

	// Close all tracked connections
	activeConnectionsMu.Lock()
	defer activeConnectionsMu.Unlock()
	for key, cached := range activeConnections {
		closeConnection(cached)
		delete(activeConnections, key)
	}
}

// getOrCreateConnection retrieves a cached connection or creates a new one.
// Returns the cachedConnection wrapper to allow reference counting.
func getOrCreateConnection(config *engine.PluginConfig, createDB DBCreationFunc) (*cachedConnection, error) {
	log.Logger.Infof("getting or creating connection")
	key := getConnectionCacheKey(config)

	// Try to get from cache first
	if connectionCache != nil {
		if cached, found := connectionCache.Get(key); found && cached != nil {
			if sqlDB, err := cached.db.DB(); err == nil {
				if err := sqlDB.Ping(); err == nil {
					return cached, nil
				}
				// Ping failed - connection is stale, remove it
				connectionCache.Del(key)
				activeConnectionsMu.Lock()
				delete(activeConnections, key)
				activeConnectionsMu.Unlock()
			}
		}
	}

	// Need to create new connection
	connectionCacheMu.Lock()
	defer connectionCacheMu.Unlock()

	// Double-check after acquiring lock in case another goroutine created conn in the meantime
	if connectionCache != nil {
		if cached, found := connectionCache.Get(key); found && cached != nil {
			if sqlDB, err := cached.db.DB(); err == nil {
				if err := sqlDB.Ping(); err == nil {
					return cached, nil
				}
				// Ping failed - connection is stale, remove it
				connectionCache.Del(key)
				activeConnectionsMu.Lock()
				delete(activeConnections, key)
				activeConnectionsMu.Unlock()
			}
		}
	}

	db, err := createDB(config)
	if err != nil {
		return nil, err
	}

	cached := &cachedConnection{
		db:       db,
		lastUsed: time.Now().Unix(),
	}

	// Add to cache and track for cleanup
	if connectionCache != nil {
		connectionCache.Set(key, cached, 1)
	}
	activeConnectionsMu.Lock()
	activeConnections[key] = cached
	activeConnectionsMu.Unlock()

	return cached, nil
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
	cached, err := getOrCreateConnection(config, DB)
	if err != nil {
		var zero T
		return zero, err
	}

	activeOperations.Add(1)
	atomic.StoreInt64(&cached.lastUsed, time.Now().Unix())
	atomic.AddInt32(&cached.refCount, 1)
	defer func() {
		atomic.AddInt32(&cached.refCount, -1)
		activeOperations.Done()
		log.Logger.Infof("operation done")
	}()

	return operation(cached.db)
}
