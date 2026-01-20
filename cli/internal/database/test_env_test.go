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

package database

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/clidey/whodb/cli/internal/config"
)

var (
	testHomeOnce sync.Once
	testHome     string
)

func setupTestEnv(t *testing.T) {
	t.Helper()

	testHomeOnce.Do(func() {
		dir, err := os.MkdirTemp("", "whodb-cli-test-home-")
		if err != nil {
			t.Fatalf("Failed to create test home: %v", err)
		}
		testHome = dir
	})

	if err := os.Setenv("HOME", testHome); err != nil {
		t.Fatalf("Failed to set HOME: %v", err)
	}
	if err := os.Setenv("USERPROFILE", testHome); err != nil {
		t.Fatalf("Failed to set USERPROFILE: %v", err)
	}
	if err := os.Setenv("XDG_DATA_HOME", testHome); err != nil {
		t.Fatalf("Failed to set XDG_DATA_HOME: %v", err)
	}
	if err := os.Setenv("APPDATA", testHome); err != nil {
		t.Fatalf("Failed to set APPDATA: %v", err)
	}

	cleanupConfigFiles(t)
}

func cleanupConfigFiles(t *testing.T) {
	t.Helper()

	configPath, err := config.GetConfigPath()
	if err != nil {
		t.Fatalf("GetConfigPath failed: %v", err)
	}
	if err := os.Remove(configPath); err != nil && !os.IsNotExist(err) {
		t.Fatalf("Remove config file failed: %v", err)
	}

	configDir, err := config.GetConfigDir()
	if err != nil {
		t.Fatalf("GetConfigDir failed: %v", err)
	}
	historyPath := filepath.Join(configDir, "history.json")
	if err := os.Remove(historyPath); err != nil && !os.IsNotExist(err) {
		t.Fatalf("Remove history file failed: %v", err)
	}
}
