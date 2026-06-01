// Copyright 2025 Clidey, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package plugins

import (
	"context"
	"fmt"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/clidey/whodb/core/src/engine"
)

func TestWithConnectionCachesConnections(t *testing.T) {
	resetConnectionCache(t)
	t.Cleanup(func() {
		resetConnectionCache(t)
	})

	creations := 0
	cfg := &engine.PluginConfig{
		Credentials: &engine.Credentials{
			Type:     "Sqlite3",
			Hostname: "localhost",
			Username: "user",
			Password: "pass",
			Database: "file::memory:?cache=shared",
			Advanced: []engine.Record{
				{Key: "mode", Value: "memory"},
			},
		},
	}

	createDB := func(*engine.PluginConfig) (*gorm.DB, error) {
		creations++
		return gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	}

	var pointers []string
	operation := func(db *gorm.DB) (string, error) {
		sqlDB, err := db.DB()
		if err != nil {
			return "", err
		}
		pointers = append(pointers, fmt.Sprintf("%p", sqlDB))
		return "ok", nil
	}

	for range 2 {
		if _, err := WithConnection(cfg, createDB, operation); err != nil {
			t.Fatalf("WithConnection returned error: %v", err)
		}
	}

	if creations != 1 {
		t.Fatalf("expected connection to be created once, got %d", creations)
	}

	if len(pointers) != 2 || pointers[0] != pointers[1] {
		t.Fatalf("expected cached connection to be reused between calls")
	}
}

func TestGetConnectionCacheKeyIgnoresPasswordButTracksAdvancedConfig(t *testing.T) {
	cfg := &engine.PluginConfig{
		Credentials: &engine.Credentials{
			Type:     "Postgres",
			Hostname: "localhost",
			Username: "alice",
			Password: "secret1",
			Database: "db1",
		},
	}

	key1 := getConnectionCacheKey(cfg)

	cfg.Credentials.Password = "secret2"
	key2 := getConnectionCacheKey(cfg)
	if key1 != key2 {
		t.Fatalf("changing password should not alter the outer cache bucket key")
	}

	cfg.Credentials.Advanced = []engine.Record{{Key: "sslmode", Value: "require"}}
	key3 := getConnectionCacheKey(cfg)
	if key2 == key3 {
		t.Fatalf("changing advanced config should alter cache key")
	}
}

func TestWithConnectionAppliesOperationContext(t *testing.T) {
	resetConnectionCache(t)
	t.Cleanup(func() {
		resetConnectionCache(t)
	})

	type ctxKey string

	cfg := &engine.PluginConfig{
		Credentials: &engine.Credentials{
			Type:     "Sqlite3",
			Hostname: "localhost",
			Username: "user",
			Password: "pass",
			Database: "file::memory:?cache=shared",
		},
		Context: context.WithValue(context.Background(), ctxKey("request_id"), "ctx-123"),
	}

	createDB := func(*engine.PluginConfig) (*gorm.DB, error) {
		return gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	}

	if _, err := WithConnection(cfg, createDB, func(db *gorm.DB) (string, error) {
		if db.Statement.Context == nil {
			t.Fatal("expected gorm statement context to be set")
		}
		if got := db.Statement.Context.Value(ctxKey("request_id")); got != "ctx-123" {
			t.Fatalf("expected propagated context value %q, got %v", "ctx-123", got)
		}
		return "ok", nil
	}); err != nil {
		t.Fatalf("WithConnection returned error: %v", err)
	}
}

func resetConnectionCache(t *testing.T) {
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
