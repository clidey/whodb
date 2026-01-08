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

// Package config provides a unified configuration file for all WhoDB components.
// Each component (CLI, server, desktop) owns its own section of the config.
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/clidey/whodb/core/src/common/datadir"
)

const configFileName = "config.json"

// Section names for each component
const (
	SectionCLI     = "cli"
	SectionAWS     = "aws"
	SectionDesktop = "desktop"
)

var (
	configPath     string
	configPathOnce sync.Once
	configMu       sync.RWMutex
)

// RawConfig represents the entire config file as a map of sections.
// Each section is owned by a specific component.
type RawConfig map[string]json.RawMessage

// getConfigPath returns the path to the unified config file.
func getConfigPath(opts datadir.Options) (string, error) {
	var err error
	configPathOnce.Do(func() {
		var dir string
		dir, err = datadir.Get(opts)
		if err != nil {
			return
		}
		configPath = filepath.Join(dir, configFileName)
	})
	return configPath, err
}

// GetPath returns the config file path for the given options.
// This is useful for displaying to users or debugging.
func GetPath(opts datadir.Options) (string, error) {
	return getConfigPath(opts)
}

// ReadSection reads a specific section from the config file into the provided value.
// If the section doesn't exist, the value is left unchanged.
// Returns error only for file read/parse errors, not for missing sections.
func ReadSection(section string, value any, opts datadir.Options) error {
	configMu.RLock()
	defer configMu.RUnlock()

	path, err := getConfigPath(opts)
	if err != nil {
		return err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No config file yet, use defaults
		}
		return err
	}

	var raw RawConfig
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	sectionData, exists := raw[section]
	if !exists {
		return nil // Section doesn't exist, use defaults
	}

	return json.Unmarshal(sectionData, value)
}

// WriteSection writes a specific section to the config file, preserving other sections.
// Uses atomic write (temp file + rename) to prevent corruption.
func WriteSection(section string, value any, opts datadir.Options) error {
	configMu.Lock()
	defer configMu.Unlock()

	path, err := getConfigPath(opts)
	if err != nil {
		return err
	}

	// Read existing config (or start fresh)
	var raw RawConfig
	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		raw = make(RawConfig)
	} else {
		if err := json.Unmarshal(data, &raw); err != nil {
			// If existing file is corrupted, start fresh
			raw = make(RawConfig)
		}
	}

	// Marshal the section value
	sectionData, err := json.Marshal(value)
	if err != nil {
		return err
	}

	// Update the section
	raw[section] = sectionData

	// Marshal the entire config with pretty printing
	newData, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}

	// Atomic write: write to temp file, then rename
	dir := filepath.Dir(path)
	tempFile, err := os.CreateTemp(dir, ".config-*.tmp")
	if err != nil {
		return err
	}
	tempPath := tempFile.Name()

	// Write data and close
	if _, err := tempFile.Write(newData); err != nil {
		tempFile.Close()
		os.Remove(tempPath)
		return err
	}
	if err := tempFile.Close(); err != nil {
		os.Remove(tempPath)
		return err
	}

	// Atomic rename
	if err := os.Rename(tempPath, path); err != nil {
		os.Remove(tempPath)
		return err
	}

	return nil
}

// ResetConfigPath resets the cached config path.
// This is only for testing - do not use in production code.
func ResetConfigPath() {
	configPathOnce = sync.Once{}
	configPath = ""
}
