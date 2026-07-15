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
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/clidey/whodb/core/src/crypto"
	"github.com/clidey/whodb/core/src/log"
	"github.com/clidey/whodb/core/src/source"
)

// sessionDBFileName is the SQLite file holding the session store, created inside
// the data directory. It is distinct from any user-configured sqlite3 data source.
const sessionDBFileName = "whodb.db"

// errSessionNotFound indicates no live session matched the token.
var errSessionNotFound = errors.New("session not found")

// errSessionInvalid indicates a matching session row exists but its credentials
// could not be decrypted (for example after the encryption key changed). The
// caller should treat this as an expired session and clear the cookie.
var errSessionInvalid = errors.New("session invalid")

// sessionRow is the persisted session record. The opaque session token and CSRF
// token are stored only as SHA-256 hashes; the database credentials are stored
// as AES-256-GCM ciphertext.
type sessionRow struct {
	SessionHash          string `gorm:"primaryKey"`
	EncryptedCredentials []byte
	CSRFTokenHash        string
	ExpiresAt            time.Time `gorm:"index"`
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// TableName sets the table name for sessionRow.
func (sessionRow) TableName() string { return "sessions" }

var (
	sessionDB     *gorm.DB
	sessionKeyHex string
	sessionMu     sync.RWMutex
)

// InitSessionStore opens (and migrates) the SQLite-backed session store in
// dataDir and records the encryption key used to protect stored credentials.
// It is safe to call once at startup in server mode.
func InitSessionStore(dataDir, encryptionKey string) error {
	if err := validateKeyHex(encryptionKey); err != nil {
		return fmt.Errorf("session store: %w", err)
	}
	dsn := "file:" + filepath.Join(dataDir, sessionDBFileName) + "?_busy_timeout=5000&_journal_mode=WAL"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		return fmt.Errorf("session store: open: %w", err)
	}
	if err := db.AutoMigrate(&sessionRow{}); err != nil {
		return fmt.Errorf("session store: migrate: %w", err)
	}

	sessionMu.Lock()
	sessionDB = db
	sessionKeyHex = encryptionKey
	sessionMu.Unlock()
	return nil
}

// SessionStoreEnabled reports whether the session store has been initialized.
func SessionStoreEnabled() bool {
	sessionMu.RLock()
	defer sessionMu.RUnlock()
	return sessionDB != nil
}

// CreateSession encrypts and stores the given credentials, returning the opaque
// session token and CSRF token to hand to the client, plus the expiry time. The
// tokens themselves are never persisted — only their hashes.
func CreateSession(credentials *source.Credentials, ttl time.Duration) (token, csrfToken string, expiresAt time.Time, err error) {
	sessionMu.RLock()
	db, key := sessionDB, sessionKeyHex
	sessionMu.RUnlock()
	if db == nil {
		return "", "", time.Time{}, errors.New("session store not initialized")
	}

	// Marshaling the credentials (including any AccessToken) is intentional — the
	// result is immediately AES-256-GCM encrypted before it is ever stored.
	plaintext, err := json.Marshal(credentials) //nolint:gosec // encrypted before storage
	if err != nil {
		return "", "", time.Time{}, err
	}
	encrypted, err := crypto.Encrypt(key, string(plaintext))
	if err != nil {
		return "", "", time.Time{}, err
	}

	token, err = randomToken(48)
	if err != nil {
		return "", "", time.Time{}, err
	}
	csrfToken, err = randomToken(32)
	if err != nil {
		return "", "", time.Time{}, err
	}

	expiresAt = time.Now().Add(ttl)
	row := sessionRow{
		SessionHash:          hashToken(token),
		EncryptedCredentials: encrypted,
		CSRFTokenHash:        hashToken(csrfToken),
		ExpiresAt:            expiresAt,
	}
	if err := db.Create(&row).Error; err != nil {
		return "", "", time.Time{}, err
	}
	return token, csrfToken, expiresAt, nil
}

// LookupSession resolves a session token to its stored credentials. It returns
// errSessionNotFound when no live (unexpired) session matches, and
// errSessionInvalid (after deleting the offending row) when the row exists but
// its credentials cannot be decrypted. needsRefresh is true when less than half
// the TTL window remains, signaling the caller to slide the expiry forward.
func LookupSession(token string, ttl time.Duration) (creds *source.Credentials, csrfHash string, needsRefresh bool, err error) {
	sessionMu.RLock()
	db, key := sessionDB, sessionKeyHex
	sessionMu.RUnlock()
	if db == nil {
		return nil, "", false, errors.New("session store not initialized")
	}

	var row sessionRow
	err = db.Where("session_hash = ? AND expires_at > ?", hashToken(token), time.Now()).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, "", false, errSessionNotFound
	}
	if err != nil {
		return nil, "", false, err
	}

	plaintext, err := crypto.Decrypt(key, row.EncryptedCredentials)
	if err != nil {
		// Undecryptable (e.g. key rotated/lost). Drop the row and treat as expired.
		_ = db.Where("session_hash = ?", row.SessionHash).Delete(&sessionRow{}).Error
		return nil, "", false, errSessionInvalid
	}
	credentials := &source.Credentials{}
	if err := json.Unmarshal([]byte(plaintext), credentials); err != nil {
		_ = db.Where("session_hash = ?", row.SessionHash).Delete(&sessionRow{}).Error
		return nil, "", false, errSessionInvalid
	}

	needsRefresh = time.Until(row.ExpiresAt) < ttl/2
	return credentials, row.CSRFTokenHash, needsRefresh, nil
}

// RefreshSession slides the session expiry forward by ttl from now.
func RefreshSession(token string, ttl time.Duration) (time.Time, error) {
	sessionMu.RLock()
	db := sessionDB
	sessionMu.RUnlock()
	if db == nil {
		return time.Time{}, errors.New("session store not initialized")
	}
	expiresAt := time.Now().Add(ttl)
	err := db.Model(&sessionRow{}).
		Where("session_hash = ?", hashToken(token)).
		Update("expires_at", expiresAt).Error
	return expiresAt, err
}

// DeleteSession removes the session identified by token (used at logout).
func DeleteSession(token string) error {
	sessionMu.RLock()
	db := sessionDB
	sessionMu.RUnlock()
	if db == nil {
		return nil
	}
	return db.Where("session_hash = ?", hashToken(token)).Delete(&sessionRow{}).Error
}

// CleanupExpiredSessions deletes all sessions whose expiry has passed.
func CleanupExpiredSessions() error {
	sessionMu.RLock()
	db := sessionDB
	sessionMu.RUnlock()
	if db == nil {
		return nil
	}
	return db.Where("expires_at <= ?", time.Now()).Delete(&sessionRow{}).Error
}

// StartSessionCleanup runs CleanupExpiredSessions immediately and then on the
// given interval until the returned cancel function is called or ctx is done.
func StartSessionCleanup(ctx context.Context, interval time.Duration) func() {
	ctx, cancel := context.WithCancel(ctx)
	go func() {
		if err := CleanupExpiredSessions(); err != nil {
			log.Debugf("session cleanup: %v", err)
		}
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := CleanupExpiredSessions(); err != nil {
					log.Debugf("session cleanup: %v", err)
				}
			}
		}
	}()
	return cancel
}

// randomToken returns a URL-safe base64 string of the given random byte length.
func randomToken(byteLen int) (string, error) {
	buf := make([]byte, byteLen)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
