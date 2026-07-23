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

package auth

import (
	"errors"
	"sync"
	"testing"
	"time"

	sqlite3 "github.com/mattn/go-sqlite3"

	"github.com/clidey/whodb/core/src/source"
)

const storeTestKey = "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f"

func newTestStore(t *testing.T) {
	t.Helper()
	if err := InitSessionStore(t.TempDir(), storeTestKey); err != nil {
		t.Fatalf("InitSessionStore: %v", err)
	}
	t.Cleanup(func() {
		sessionMu.Lock()
		sessionDB = nil
		sessionKeyHex = ""
		sessionMu.Unlock()
	})
}

func testCredentials() *source.Credentials {
	id := "profile-1"
	return &source.Credentials{
		ID:         &id,
		SourceType: "Postgres",
		Values:     map[string]string{"Hostname": "db.internal", "Password": "s3cr3t"},
	}
}

func TestCreateAndLookupSession(t *testing.T) {
	newTestStore(t)
	ttl := time.Hour

	token, csrf, _, err := CreateSession(testCredentials(), ttl)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if token == "" || csrf == "" {
		t.Fatal("expected non-empty token and csrf")
	}

	creds, csrfHash, needsRefresh, err := LookupSession(token, ttl)
	if err != nil {
		t.Fatalf("LookupSession: %v", err)
	}
	if creds.SourceType != "Postgres" || creds.Values["Password"] != "s3cr3t" {
		t.Fatalf("credentials not round-tripped: %+v", creds)
	}
	if csrfHash != hashToken(csrf) {
		t.Fatal("csrf hash mismatch")
	}
	if needsRefresh {
		t.Fatal("fresh session should not need refresh")
	}
}

func TestLookupExpiredSession(t *testing.T) {
	newTestStore(t)
	// Negative TTL creates an already-expired row.
	token, _, _, err := CreateSession(testCredentials(), -time.Minute)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if _, _, _, err := LookupSession(token, time.Hour); !errors.Is(err, errSessionNotFound) {
		t.Fatalf("got %v want errSessionNotFound", err)
	}
}

func TestLookupNeedsRefreshBelowHalfWindow(t *testing.T) {
	newTestStore(t)
	ttl := time.Hour
	// Create with a short remaining window (< ttl/2).
	token, _, _, err := CreateSession(testCredentials(), 20*time.Minute)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	_, _, needsRefresh, err := LookupSession(token, ttl)
	if err != nil {
		t.Fatalf("LookupSession: %v", err)
	}
	if !needsRefresh {
		t.Fatal("expected needsRefresh when remaining < ttl/2")
	}
}

func TestRefreshSessionExtendsExpiry(t *testing.T) {
	newTestStore(t)
	ttl := time.Hour
	token, _, _, err := CreateSession(testCredentials(), 20*time.Minute)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if _, err := RefreshSession(token, ttl); err != nil {
		t.Fatalf("RefreshSession: %v", err)
	}
	_, _, needsRefresh, err := LookupSession(token, ttl)
	if err != nil {
		t.Fatalf("LookupSession: %v", err)
	}
	if needsRefresh {
		t.Fatal("session should not need refresh after being extended")
	}
}

func TestLookupDecryptFailInvalidatesSession(t *testing.T) {
	newTestStore(t)
	token, _, _, err := CreateSession(testCredentials(), time.Hour)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	// Swap the key so the stored ciphertext no longer decrypts.
	sessionMu.Lock()
	sessionKeyHex = "1f1e1d1c1b1a191817161514131211100f0e0d0c0b0a09080706050403020100"
	sessionMu.Unlock()

	if _, _, _, err := LookupSession(token, time.Hour); !errors.Is(err, errSessionInvalid) {
		t.Fatalf("got %v want errSessionInvalid", err)
	}
	// The offending row should have been deleted; a second lookup is not-found.
	sessionMu.Lock()
	sessionKeyHex = storeTestKey
	sessionMu.Unlock()
	if _, _, _, err := LookupSession(token, time.Hour); !errors.Is(err, errSessionNotFound) {
		t.Fatalf("got %v want errSessionNotFound after invalidation", err)
	}
}

func TestDeleteSession(t *testing.T) {
	newTestStore(t)
	token, _, _, err := CreateSession(testCredentials(), time.Hour)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if err := DeleteSession(token); err != nil {
		t.Fatalf("DeleteSession: %v", err)
	}
	if _, _, _, err := LookupSession(token, time.Hour); !errors.Is(err, errSessionNotFound) {
		t.Fatalf("got %v want errSessionNotFound", err)
	}
}

func TestCleanupExpiredSessions(t *testing.T) {
	newTestStore(t)
	expired, _, _, _ := CreateSession(testCredentials(), -time.Minute)
	live, _, _, _ := CreateSession(testCredentials(), time.Hour)

	if err := CleanupExpiredSessions(); err != nil {
		t.Fatalf("CleanupExpiredSessions: %v", err)
	}

	var count int64
	sessionDB.Model(&sessionRow{}).Where("session_hash = ?", hashToken(expired)).Count(&count)
	if count != 0 {
		t.Fatal("expired session should have been cleaned up")
	}
	if _, _, _, err := LookupSession(live, time.Hour); err != nil {
		t.Fatalf("live session should survive cleanup: %v", err)
	}
}

// TestDisableSessionStoreSkipsInitialization guards the EE opt-out: editions
// with their own browser-session mechanism (which never call CreateSession)
// call DisableSessionStore so EnsureSessionStore does not create an unused
// session database file and cleanup ticker.
func TestDisableSessionStoreSkipsInitialization(t *testing.T) {
	t.Cleanup(func() {
		sessionStoreDisabled = false
		ensureOnce = sync.Once{}
		sessionMu.Lock()
		sessionDB = nil
		sessionKeyHex = ""
		sessionMu.Unlock()
	})

	DisableSessionStore()
	if stop := EnsureSessionStore(); stop != nil {
		t.Fatal("expected no cleanup stop function when session store is disabled")
	}

	sessionMu.RLock()
	db := sessionDB
	sessionMu.RUnlock()
	if db != nil {
		t.Fatal("expected session store to remain uninitialized when disabled")
	}
}

func TestIsSessionDBCorrupted(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "corrupt disk image", err: sqlite3.Error{Code: sqlite3.ErrCorrupt}, want: true},
		{name: "not a database file", err: sqlite3.Error{Code: sqlite3.ErrNotADB}, want: true},
		{name: "busy is transient, not corruption", err: sqlite3.Error{Code: sqlite3.ErrBusy}, want: false},
		{name: "locked is transient, not corruption", err: sqlite3.Error{Code: sqlite3.ErrLocked}, want: false},
		{name: "non-sqlite error", err: errors.New("boom"), want: false},
		{name: "nil error", err: nil, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isSessionDBCorrupted(tt.err); got != tt.want {
				t.Fatalf("isSessionDBCorrupted(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}
