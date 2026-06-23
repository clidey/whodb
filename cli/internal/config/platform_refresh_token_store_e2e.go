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

	"github.com/zalando/go-keyring"
)

const platformE2ETokenDirEnv = "WHODB_CLI_E2E_PLATFORM_TOKEN_DIR"

type filePlatformRefreshTokenStore struct{}

func init() {
	platformRefreshTokenStoreOverride = filePlatformRefreshTokenStore{}
}

func (filePlatformRefreshTokenStore) Save(hostURL, accountID, refreshToken string) error {
	dir, ok := platformE2ETokenDir()
	if !ok {
		return keyring.ErrNotFound
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	return os.WriteFile(platformE2ETokenPath(dir, hostURL, accountID), []byte(refreshToken), 0o600)
}

func (filePlatformRefreshTokenStore) Get(hostURL, accountID string) (string, error) {
	dir, ok := platformE2ETokenDir()
	if !ok {
		return "", keyring.ErrNotFound
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
		return nil
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
