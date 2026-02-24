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

// Package bamlconfig sets BAML environment defaults before the BAML native library loads.
// This package MUST be imported first (before any other imports) in main packages
// to ensure the environment variable is set before baml_client is imported.
//
// Usage in main.go:
//
//	import (
//		_ "github.com/clidey/whodb/core/src/bamlconfig" // Must be first!
//		// ... other imports
//	)
package bamlconfig

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func init() {
	// Set BAML_LIBRARY_PATH for macOS desktop builds
	// This must happen before BAML's init() loads the native library
	configureBamlLibraryPath()

	// Don't override if user explicitly set BAML_LOG
	if os.Getenv("BAML_LOG") != "" {
		return
	}

	// Map WHODB_LOG_LEVEL to BAML_LOG
	level := strings.ToLower(os.Getenv("WHODB_LOG_LEVEL"))

	var bamlLevel string
	switch level {
	case "debug":
		bamlLevel = "debug"
	case "info":
		bamlLevel = "info"
	case "warning", "warn":
		bamlLevel = "warn"
	case "error":
		bamlLevel = "error"
	case "none", "off", "disabled":
		bamlLevel = "off"
	default:
		// Default: only show errors (quieter output)
		bamlLevel = "error"
	}

	os.Setenv("BAML_LOG", bamlLevel)
}

// configureBamlLibraryPath sets BAML_LIBRARY_PATH for macOS desktop builds.
// On macOS, when running as a .app bundle, we bundle a signed copy of the BAML
// native library in Contents/Frameworks/ to satisfy Gatekeeper requirements.
func configureBamlLibraryPath() {
	// Skip if already set
	if existing := os.Getenv("BAML_LIBRARY_PATH"); existing != "" {
		return
	}

	// Only applies to macOS
	if runtime.GOOS != "darwin" {
		return
	}

	// Get executable path to find the bundle
	execPath, err := os.Executable()
	if err != nil {
		return
	}

	// Resolve symlinks to get real path
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return
	}

	// Check if we're running from a .app bundle
	// Expected structure: WhoDB.app/Contents/MacOS/whodb
	macosDir := filepath.Dir(execPath)
	contentsDir := filepath.Dir(macosDir)

	if filepath.Base(macosDir) != "MacOS" || filepath.Base(contentsDir) != "Contents" {
		return // Not running from a bundle
	}

	// Determine architecture-specific dylib name
	var dylibName string
	switch runtime.GOARCH {
	case "arm64":
		dylibName = "libbaml_cffi-aarch64-apple-darwin.dylib"
	case "amd64":
		dylibName = "libbaml_cffi-x86_64-apple-darwin.dylib"
	default:
		return // Unsupported architecture
	}

	// Check if bundled dylib exists
	frameworksDir := filepath.Join(contentsDir, "Frameworks")
	dylibPath := filepath.Join(frameworksDir, dylibName)

	if _, err := os.Stat(dylibPath); err == nil {
		os.Setenv("BAML_LIBRARY_PATH", dylibPath)
	}
}
