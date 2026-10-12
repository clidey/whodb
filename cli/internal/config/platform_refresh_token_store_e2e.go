//go:build e2e_platform

/*
 * Copyright 2026 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 */

package config

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"

	"github.com/clidey/whodb/cli/pkg/identity"
	"github.com/zalando/go-keyring"
)

const platformE2ETokenDirEnv = "WHODB_CLI_E2E_PLATFORM_TOKEN_DIR"

// filePlatformRefreshTokenStore keeps live e2e refresh tokens in a directory
// named by WHODB_CLI_E2E_PLATFORM_TOKEN_DIR. When that variable is unset the
// store falls back to the (mockable) OS keyring so unit tests in the same
// binary keep working under the e2e_platform build tag.
type filePlatformRefreshTokenStore struct{}

func init() {
	platformRefreshTokenStoreOverride = filePlatformRefreshTokenStore{}
}

func (filePlatformRefreshTokenStore) Save(hostURL, accountID, refreshToken string) error {
	dir, ok := platformE2ETokenDir()
	if !ok {
		return keyring.Set(identity.Current().KeyringService, platformRefreshTokenKey(hostURL, accountID), refreshToken)
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	return os.WriteFile(platformE2ETokenPath(dir, hostURL, accountID), []byte(refreshToken), 0o600)
}

func (filePlatformRefreshTokenStore) Get(hostURL, accountID string) (string, error) {
	dir, ok := platformE2ETokenDir()
	if !ok {
		return keyring.Get(identity.Current().KeyringService, platformRefreshTokenKey(hostURL, accountID))
	}
	raw, err := os.ReadFile(platformE2ETokenPath(dir, hostURL, accountID))
	if os.IsNotExist(err) {
		return "", keyring.ErrNotFound
	}
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func (filePlatformRefreshTokenStore) Delete(hostURL, accountID string) error {
	dir, ok := platformE2ETokenDir()
	if !ok {
		err := keyring.Delete(identity.Current().KeyringService, platformRefreshTokenKey(hostURL, accountID))
		if err == keyring.ErrNotFound {
			return nil
		}
		return err
	}
	err := os.Remove(platformE2ETokenPath(dir, hostURL, accountID))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func platformE2ETokenDir() (string, bool) {
	dir := os.Getenv(platformE2ETokenDirEnv)
	return dir, dir != ""
}

func platformE2ETokenPath(dir, hostURL, accountID string) string {
	sum := sha256.Sum256([]byte(platformRefreshTokenKey(hostURL, accountID)))
	return filepath.Join(dir, hex.EncodeToString(sum[:]))
}
