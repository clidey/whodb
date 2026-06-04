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
	"database/sql"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/clidey/whodb/core/src/engine"
)

type invalidConnPool struct{}

func (invalidConnPool) PrepareContext(context.Context, string) (*sql.Stmt, error) {
	return nil, nil
}

func (invalidConnPool) ExecContext(context.Context, string, ...any) (sql.Result, error) {
	return nil, nil
}

func (invalidConnPool) QueryContext(context.Context, string, ...any) (*sql.Rows, error) {
	return nil, nil
}

func (invalidConnPool) QueryRowContext(context.Context, string, ...any) *sql.Row {
	return &sql.Row{}
}

func resetCacheState(t *testing.T) {
	t.Helper()
	connectionCacheMu.Lock()
	for key, bucket := range connectionCache {
		for secret, cached := range bucket {
			if cached != nil && cached.db != nil {
				if sqlDB, err := cached.db.DB(); err == nil {
					sqlDB.Close()
				}
			}
			delete(bucket, secret)
		}
		delete(connectionCache, key)
	}
	connectionCacheMu.Unlock()
}

func newTestConfig() *engine.PluginConfig {
	return &engine.PluginConfig{
		Credentials: &engine.Credentials{
			Type:     "Sqlite3",
			Hostname: "localhost",
			Username: "user",
			Password: "pw",
			Database: "file::memory:?cache=shared",
		},
	}
}

func TestRemoveConnectionRemovesFromCache(t *testing.T) {
	resetCacheState(t)
	t.Cleanup(func() { resetCacheState(t) })

	cfg := newTestConfig()
	createDB := func(*engine.PluginConfig) (*gorm.DB, error) {
		return gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	}

	_, err := getOrCreateConnection(cfg, createDB)
	if err != nil {
		t.Fatalf("failed to create connection: %v", err)
	}

	// Verify it's in the cache
	key := getConnectionCacheKey(cfg)
	secret := getConnectionCacheSecret(cfg)
	connectionCacheMu.Lock()
	_, exists := getCachedConnectionLocked(key, secret)
	connectionCacheMu.Unlock()
	if !exists {
		t.Fatalf("expected connection to be in cache")
	}

	// Remove it
	RemoveConnection(cfg)

	connectionCacheMu.Lock()
	_, exists = getCachedConnectionLocked(key, secret)
	connectionCacheMu.Unlock()
	if exists {
		t.Fatalf("expected connection to be removed from cache")
	}
}

func TestCleanupStaleConnectionsRemovesOldEntries(t *testing.T) {
	resetCacheState(t)
	t.Cleanup(func() { resetCacheState(t) })

	cfg := newTestConfig()
	createDB := func(*engine.PluginConfig) (*gorm.DB, error) {
		return gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	}

	_, err := getOrCreateConnection(cfg, createDB)
	if err != nil {
		t.Fatalf("failed to create connection: %v", err)
	}

	// Mark as stale by setting lastUsed in the past
	key := getConnectionCacheKey(cfg)
	secret := getConnectionCacheSecret(cfg)
	connectionCacheMu.Lock()
	if cached, ok := getCachedConnectionLocked(key, secret); ok {
		cached.lastUsed = time.Now().Add(-connectionCacheTTL * 2)
	}
	connectionCacheMu.Unlock()

	cleanupStaleConnections()

	connectionCacheMu.Lock()
	_, exists := getCachedConnectionLocked(key, secret)
	connectionCacheMu.Unlock()
	if exists {
		t.Fatalf("expected stale connection to be cleaned up")
	}
}

func TestEvictOldestConnectionPrefersOldest(t *testing.T) {
	resetCacheState(t)
	t.Cleanup(func() { resetCacheState(t) })

	// Inject two connections with different lastUsed timestamps
	connectionCacheMu.Lock()
	connectionCache["new"] = connectionCacheBucket{
		"pw-new": {lastUsed: time.Now()},
	}
	connectionCache["old"] = connectionCacheBucket{
		"pw-old": {lastUsed: time.Now().Add(-10 * time.Minute)},
	}
	connectionCacheMu.Unlock()

	connectionCacheMu.Lock()
	evictOldestConnection("", "")
	connectionCacheMu.Unlock()

	connectionCacheMu.Lock()
	_, oldExists := connectionCache["old"]
	_, newExists := connectionCache["new"]
	connectionCacheMu.Unlock()

	if oldExists {
		t.Fatalf("expected oldest connection to be evicted")
	}
	if !newExists {
		t.Fatalf("expected newer connection to remain")
	}
}

func TestGetOrCreateConnectionReusesCache(t *testing.T) {
	resetCacheState(t)
	t.Cleanup(func() { resetCacheState(t) })

	cfg := newTestConfig()
	callCount := 0
	createDB := func(*engine.PluginConfig) (*gorm.DB, error) {
		callCount++
		return gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	}

	// First call - should create
	db1, err := getOrCreateConnection(cfg, createDB)
	if err != nil {
		t.Fatalf("first call failed: %v", err)
	}
	if callCount != 1 {
		t.Fatalf("expected createDB to be called once, got %d", callCount)
	}

	// Second call - should reuse cache
	db2, err := getOrCreateConnection(cfg, createDB)
	if err != nil {
		t.Fatalf("second call failed: %v", err)
	}
	if callCount != 1 {
		t.Fatalf("expected createDB to still be 1 (cache hit), got %d", callCount)
	}

	if db1 != db2 {
		t.Fatalf("expected same db instance from cache")
	}
}

func TestGetOrCreateConnectionReusesCachedHandleWithoutSQLDB(t *testing.T) {
	resetCacheState(t)
	t.Cleanup(func() { resetCacheState(t) })

	cfg := newTestConfig()
	callCount := 0
	createDB := func(*engine.PluginConfig) (*gorm.DB, error) {
		callCount++
		db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
		if err != nil {
			return nil, err
		}
		db.ConnPool = invalidConnPool{}
		if db.Statement != nil {
			db.Statement.ConnPool = invalidConnPool{}
		}
		return db, nil
	}

	db1, err := getOrCreateConnection(cfg, createDB)
	if err != nil {
		t.Fatalf("first call failed: %v", err)
	}
	db2, err := getOrCreateConnection(cfg, createDB)
	if err != nil {
		t.Fatalf("second call failed: %v", err)
	}

	if callCount != 1 {
		t.Fatalf("expected createDB to be called once for cached handle reuse, got %d", callCount)
	}
	if db1 != db2 {
		t.Fatalf("expected cached handle to be reused when sql.DB is unavailable")
	}
}

func TestGetOrCreateConnectionSeparatesDifferentPasswords(t *testing.T) {
	resetCacheState(t)
	t.Cleanup(func() { resetCacheState(t) })

	cfg1 := newTestConfig()
	cfg2 := newTestConfig()
	cfg2.Credentials.Password = "different"

	callCount := 0
	createDB := func(*engine.PluginConfig) (*gorm.DB, error) {
		callCount++
		return gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	}

	db1, err := getOrCreateConnection(cfg1, createDB)
	if err != nil {
		t.Fatalf("first call failed: %v", err)
	}

	db2, err := getOrCreateConnection(cfg2, createDB)
	if err != nil {
		t.Fatalf("second call failed: %v", err)
	}

	if callCount != 2 {
		t.Fatalf("expected separate connections for different passwords, got %d creations", callCount)
	}

	if db1 == db2 {
		t.Fatalf("expected different db instances for different passwords")
	}
}

func TestCachedSSLStatusSeparatesDifferentPasswords(t *testing.T) {
	resetCacheState(t)
	t.Cleanup(func() { resetCacheState(t) })

	cfg1 := newTestConfig()
	cfg2 := newTestConfig()
	cfg2.Credentials.Password = "different"

	createDB := func(*engine.PluginConfig) (*gorm.DB, error) {
		return gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	}

	if _, err := getOrCreateConnection(cfg1, createDB); err != nil {
		t.Fatalf("first connection failed: %v", err)
	}
	if _, err := getOrCreateConnection(cfg2, createDB); err != nil {
		t.Fatalf("second connection failed: %v", err)
	}

	expected := &engine.SSLStatus{IsEnabled: true, Mode: "require"}
	SetCachedSSLStatus(cfg1, expected)

	if got := GetCachedSSLStatus(cfg1); got != expected {
		t.Fatalf("expected SSL status to be cached for first password")
	}
	if got := GetCachedSSLStatus(cfg2); got != nil {
		t.Fatalf("expected no SSL status for different password, got %+v", got)
	}
}
