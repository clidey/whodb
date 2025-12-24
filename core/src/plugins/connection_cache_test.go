package plugins

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/clidey/whodb/core/src/engine"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func resetCacheState(t *testing.T) {
	t.Helper()
	connectionCacheMu.Lock()
	for key, cached := range connectionCache {
		if cached != nil && cached.db != nil {
			if sqlDB, err := cached.db.DB(); err == nil {
				sqlDB.Close()
			}
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

func TestRemoveConnectionClosesWhenIdle(t *testing.T) {
	resetCacheState(t)
	t.Cleanup(func() { resetCacheState(t) })

	cfg := newTestConfig()
	createDB := func(*engine.PluginConfig) (*gorm.DB, error) {
		return gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	}

	cached, err := getOrCreateConnection(cfg, createDB)
	if err != nil {
		t.Fatalf("failed to create connection: %v", err)
	}

	// Mark idle and remove
	atomic.StoreInt32(&cached.refCount, 0)
	RemoveConnection(cfg)

	if _, ok := connectionCache[getConnectionCacheKey(cfg)]; ok {
		t.Fatalf("expected idle connection to be removed from cache")
	}
}

func TestRemoveConnectionDefersWhenInUse(t *testing.T) {
	resetCacheState(t)
	t.Cleanup(func() { resetCacheState(t) })

	cfg := newTestConfig()
	createDB := func(*engine.PluginConfig) (*gorm.DB, error) {
		return gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	}

	cached, err := getOrCreateConnection(cfg, createDB)
	if err != nil {
		t.Fatalf("failed to create connection: %v", err)
	}

	// Simulate in-flight operation
	atomic.StoreInt32(&cached.refCount, 2)
	RemoveConnection(cfg)

	if _, ok := connectionCache[getConnectionCacheKey(cfg)]; !ok {
		t.Fatalf("expected busy connection to remain in cache")
	}
}

func TestCleanupStaleConnectionsRemovesOldIdleEntries(t *testing.T) {
	resetCacheState(t)
	t.Cleanup(func() { resetCacheState(t) })

	cfg := newTestConfig()
	createDB := func(*engine.PluginConfig) (*gorm.DB, error) {
		return gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	}

	cached, err := getOrCreateConnection(cfg, createDB)
	if err != nil {
		t.Fatalf("failed to create connection: %v", err)
	}

	// Mark as stale and idle
	atomic.StoreInt64(&cached.lastUsed, time.Now().Add(-connectionCacheTTL*2).Unix())
	atomic.StoreInt32(&cached.refCount, 0)

	cleanupStaleConnections()

	if _, ok := connectionCache[getConnectionCacheKey(cfg)]; ok {
		t.Fatalf("expected stale idle connection to be cleaned up")
	}
}

func TestEvictOldestIdleConnectionPrefersOldest(t *testing.T) {
	resetCacheState(t)
	t.Cleanup(func() { resetCacheState(t) })

	// Inject two idle connections with different lastUsed timestamps
	connectionCacheMu.Lock()
	connectionCache["new"] = &cachedConnection{lastUsed: time.Now().Unix()}
	connectionCache["old"] = &cachedConnection{lastUsed: time.Now().Add(-10 * time.Minute).Unix()}
	connectionCacheMu.Unlock()

	connectionCacheMu.Lock()
	evictOldestIdleConnection("")
	connectionCacheMu.Unlock()

	if _, ok := connectionCache["old"]; ok {
		t.Fatalf("expected oldest idle connection to be evicted")
	}
	if _, ok := connectionCache["new"]; !ok {
		t.Fatalf("expected newer connection to remain")
	}
}
