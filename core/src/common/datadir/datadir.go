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

package datadir

import (
	"os"
	"path/filepath"
	"runtime"
)

// Options configures the data directory path.
type Options struct {
	// AppName is the base application name (e.g., "whodb").
	AppName string
	// EnterpriseEdition appends "-ee" to distinguish EE data.
	EnterpriseEdition bool
	// Development appends "-dev" to distinguish dev data.
	Development bool
}

// Get returns the data directory path for the given options.
// The directory is created if it doesn't exist.
//
// Paths by OS:
//   - Linux: $XDG_DATA_HOME/<app>/ or ~/.local/share/<app>/
//   - macOS: ~/Library/Application Support/<app>/
//   - Windows: %APPDATA%\<app>\
func Get(opts Options) (string, error) {
	if opts.AppName == "" {
		opts.AppName = "whodb"
	}

	base, err := getBaseDir()
	if err != nil {
		return "", err
	}

	// Build app name with suffixes
	appName := opts.AppName
	if opts.EnterpriseEdition {
		appName += "-ee"
	}
	if opts.Development {
		appName += "-dev"
	}

	dataDir := filepath.Join(base, appName)

	// Create directory if it doesn't exist
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return "", err
	}

	return dataDir, nil
}

// getBaseDir returns the base data directory for the current OS.
func getBaseDir() (string, error) {
	switch runtime.GOOS {
	case "windows":
		// %APPDATA% (e.g., C:\Users\<user>\AppData\Roaming)
		if appData := os.Getenv("APPDATA"); appData != "" {
			return appData, nil
		}
		// Fallback
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, "AppData", "Roaming"), nil

	case "darwin":
		// ~/Library/Application Support
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, "Library", "Application Support"), nil

	default:
		// Linux and others: XDG Base Directory spec
		if xdgData := os.Getenv("XDG_DATA_HOME"); xdgData != "" {
			return xdgData, nil
		}
		// Default: ~/.local/share
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".local", "share"), nil
	}
}
