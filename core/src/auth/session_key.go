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
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/zalando/go-keyring"

	"github.com/clidey/whodb/core/src/env"
	"github.com/clidey/whodb/core/src/log"
)

// sessionKeyFileName is the name of the on-disk key file used as the headless
// fallback when no key is provided via env and keyring is unavailable.
const sessionKeyFileName = "session.key"

// keyringSessionKeyName is the keyring entry name for the generated session
// encryption key in desktop mode.
const keyringSessionKeyName = "session-encryption-key"

// ResolveSessionEncryptionKey returns the 64-hex-char (32-byte) key used to
// encrypt stored session credentials, resolved once at startup in priority
// order:
//
//  1. WHODB_ENCRYPTION_KEY env var (errors if set but malformed).
//  2. Desktop mode: a key stored in the OS keyring (generated on first use).
//  3. Otherwise: a 0600 key file in dataDir (generated on first use), with a
//     warning that setting WHODB_ENCRYPTION_KEY is more secure.
func ResolveSessionEncryptionKey(dataDir string) (string, error) {
	if envKey := strings.TrimSpace(env.EncryptionKey); envKey != "" {
		if err := validateKeyHex(envKey); err != nil {
			return "", fmt.Errorf("WHODB_ENCRYPTION_KEY is invalid: %w", err)
		}
		return envKey, nil
	}

	if env.GetIsDesktopMode() {
		return resolveKeyringSessionKey()
	}

	return resolveFileSessionKey(dataDir)
}

// resolveKeyringSessionKey reads the session key from the OS keyring, generating
// and storing one on first use.
func resolveKeyringSessionKey() (string, error) {
	service := GetKeyringServiceName()
	if existing, err := keyring.Get(service, keyringSessionKeyName); err == nil {
		if verr := validateKeyHex(existing); verr == nil {
			return existing, nil
		}
		log.Warn("Session key in keyring is malformed; regenerating")
	}

	key, err := generateKeyHex()
	if err != nil {
		return "", err
	}
	if err := keyring.Set(service, keyringSessionKeyName, key); err != nil {
		return "", fmt.Errorf("failed to store session key in keyring: %w", err)
	}
	return key, nil
}

// resolveFileSessionKey reads the session key from a 0600 file in dataDir,
// generating and writing one on first use.
func resolveFileSessionKey(dataDir string) (string, error) {
	// Path is composed from the server-controlled data directory, not user input.
	path := filepath.Join(dataDir, sessionKeyFileName)
	if data, err := os.ReadFile(path); err == nil { // #nosec G304 -- path is composed from the server-controlled data directory.
		existing := strings.TrimSpace(string(data))
		if verr := validateKeyHex(existing); verr == nil {
			return existing, nil
		}
		log.Warnf("Session key file %s is malformed; regenerating", path)
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("failed to read session key file: %w", err)
	}

	key, err := generateKeyHex()
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, []byte(key), 0600); err != nil {
		return "", fmt.Errorf("failed to write session key file: %w", err)
	}
	log.Warnf("Generated a session encryption key at %s. For stronger security, set WHODB_ENCRYPTION_KEY (a 64-char hex string) so the key is not stored alongside the session database.", path)
	return key, nil
}

// generateKeyHex returns a fresh 32-byte key encoded as 64 hex characters.
func generateKeyHex() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// validateKeyHex verifies that key is a 64-char hex string decoding to 32 bytes.
func validateKeyHex(key string) error {
	decoded, err := hex.DecodeString(key)
	if err != nil || len(decoded) != 32 {
		return errors.New("key must be 64 hex characters (32 bytes)")
	}
	return nil
}
