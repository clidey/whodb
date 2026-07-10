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
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
)

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
	switch log.GetLevel() {
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
// If config.Transaction is set (as a *gorm.DB), it will be used instead of creating a new connection.
// Multi-statement connections bypass the cache and are closed immediately after use.
func WithConnection[T any](config *engine.PluginConfig, dbFn DBCreationFunc, operation DBOperation[T]) (T, error) {
	if config == nil {
		var zero T
		return zero, errors.New("plugin configuration is required")
	}

	// Check if we're operating within a transaction
	if config.Transaction != nil {
		if tx, ok := config.Transaction.(*gorm.DB); ok {
			return operation(tx.WithContext(config.OperationContext()))
		}
	}

	// Multi-statement connections are one-off (e.g., SQL imports). Create a fresh
	// connection, run the operation, and close it — no caching.
	if config.MultiStatement {
		db, err := dbFn(config)
		if err != nil {
			var zero T
			return zero, err
		}
		defer closeGormDB(db)
		return operation(db.WithContext(config.OperationContext()))
	}

	db, err := getOrCreateConnection(config, dbFn)
	if err != nil {
		log.WithFields(map[string]any{
			"conn_id": connIdentifier(config),
			"error":   err.Error(),
		}).Warn("WithConnection FAILED to get connection")
		var zero T
		return zero, err
	}

	if db == nil {
		var zero T
		return zero, errors.New("internal error: nil database connection")
	}

	return operation(db.WithContext(config.OperationContext()))
}

// GetCachedSSLStatus retrieves the SSL status from the connection cache.
// Returns nil if not cached or connection doesn't exist.
func GetCachedSSLStatus(config *engine.PluginConfig) *engine.SSLStatus {
	key := getConnectionCacheKey(config)
	secret := getConnectionCacheSecret(config)

	connectionCacheMu.Lock()
	defer connectionCacheMu.Unlock()

	if cached, found := getCachedConnectionLocked(key, secret); found {
		return cached.sslStatus
	}
	return nil
}

// SetCachedSSLStatus stores SSL status in the connection cache.
func SetCachedSSLStatus(config *engine.PluginConfig, status *engine.SSLStatus) {
	key := getConnectionCacheKey(config)
	secret := getConnectionCacheSecret(config)

	connectionCacheMu.Lock()
	defer connectionCacheMu.Unlock()

	if cached, found := getCachedConnectionLocked(key, secret); found {
		cached.sslStatus = status
	}
}
