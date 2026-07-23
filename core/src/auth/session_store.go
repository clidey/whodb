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

	sqlite3 "github.com/mattn/go-sqlite3"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/clidey/whodb/core/src/common/datadir"
	"github.com/clidey/whodb/core/src/crypto"
	"github.com/clidey/whodb/core/src/env"
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
	sessionDBPath string
	sessionMu     sync.RWMutex

	ensureOnce  sync.Once
	stopCleanup func()

	corruptionAlertOnce sync.Once

	// sessionStoreDisabled is set via DisableSessionStore before the server
	// starts (e.g. by an edition with its own session mechanism). It is only
	// ever written once, at startup, before EnsureSessionStore's sync.Once can
	// run, so no locking is needed.
	sessionStoreDisabled bool
)

// DisableSessionStore opts out of initializing the CE encrypted session store
// entirely. Editions that implement their own browser-session mechanism (and
// never call CreateSession) should call this before the server starts so no
// unused session database file or cleanup ticker is created. It must be
// called before EnsureSessionStore runs (i.e. before app.Run/InitializeRouter).
func DisableSessionStore() {
	sessionStoreDisabled = true
}

// isSessionDBCorrupted reports whether err indicates the SQLite session
// database file itself is unreadable (disk image malformed, or not a
// database file at all) rather than a transient condition like a lock
// timeout. These codes mean the file needs manual operator intervention —
// no in-process retry or query will recover it.
func isSessionDBCorrupted(err error) bool {
	var sqliteErr sqlite3.Error
	if !errors.As(err, &sqliteErr) {
		return false
	}
	return errors.Is(sqliteErr.Code, sqlite3.ErrCorrupt) || errors.Is(sqliteErr.Code, sqlite3.ErrNotADB)
}

// alertSessionDBCorrupted logs a one-time, high-visibility warning that the
// session store file appears corrupted. It does not attempt any automatic
// repair: the file is operator data (session.key lives alongside it) and
// deleting or rewriting it without consent could destroy an encryption key
// or mask an underlying disk problem. The operator must intervene manually,
// typically by stopping the server and removing the session database file
// (sessions are ephemeral; this only forces re-login).
func alertSessionDBCorrupted() {
	corruptionAlertOnce.Do(func() {
		log.Errorf(
			"Session store database appears corrupted (%s). All sessions are unusable until this is resolved. "+
				"Stop the server and delete the file to force a fresh store (all users will need to log in again).",
			sessionDBPath,
		)
	})
}

// EnsureSessionStore initializes the session store exactly once for the current
// process and returns a stop function for its cleanup ticker. It is called from
// router initialization so every server entry point (app.Run, the test server,
// etc.) gets the store — desktop and CLI modes are skipped (they use the OS
// keyring / Authorization-header flow), as is any edition that called
// DisableSessionStore. Failures are non-fatal except a malformed,
// explicitly-set WHODB_ENCRYPTION_KEY.
func EnsureSessionStore() func() {
	ensureOnce.Do(func() {
		if env.GetIsDesktopMode() || env.GetIsCLIMode() || sessionStoreDisabled {
			return
		}

		dataDir := env.DataDir
		if dataDir == "" {
			resolved, err := datadir.Get(datadir.Options{
				AppName:           "whodb",
				EnterpriseEdition: env.IsEnterpriseEdition,
				Development:       env.IsDevelopment,
			})
			if err != nil {
				log.Warnf("Session store disabled: could not resolve data directory: %v", err)
				return
			}
			dataDir = resolved
		}

		key, err := ResolveSessionEncryptionKey(dataDir)
		if err != nil {
			if env.EncryptionKey != "" {
				// Operator explicitly set a bad key — fail loudly rather than ignore it.
				log.Fatalf("Invalid WHODB_ENCRYPTION_KEY: %v", err)
			}
			log.Warnf("Session store disabled: could not resolve encryption key: %v", err)
			return
		}

		if err := InitSessionStore(dataDir, key); err != nil {
			log.Warnf("Session store disabled: %v", err)
			return
		}

		stopCleanup = StartSessionCleanup(context.Background(), time.Hour)
	})
	return stopCleanup
}

// InitSessionStore opens (and migrates) the SQLite-backed session store in
// dataDir and records the encryption key used to protect stored credentials.
// It is safe to call once at startup in server mode.
func InitSessionStore(dataDir, encryptionKey string) error {
	if err := validateKeyHex(encryptionKey); err != nil {
		return fmt.Errorf("session store: %w", err)
	}
	dbPath := filepath.Join(dataDir, sessionDBFileName)
	dsn := "file:" + dbPath + "?_busy_timeout=5000&_journal_mode=WAL"
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
	sessionDBPath = dbPath
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
	plaintext, err := json.Marshal(credentials) // #nosec G117 -- plaintext is immediately AES-256-GCM encrypted before storage.
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
	if isSessionDBCorrupted(err) {
		alertSessionDBCorrupted()
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
