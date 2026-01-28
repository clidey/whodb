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

// Package baml handles automatic download and setup of the BAML native library.
// This package must be imported before any code that uses BAML.
package baml

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// BAMLVersion is the version of BAML to download.
// This is set at build time via ldflags from core/go.mod.
var BAMLVersion = "0.217.0" // default fallback, overridden by ldflags

// Library filenames by platform
var libraryInfo = map[string]struct {
	filename string
	url      string
}{
	"darwin/amd64": {
		filename: "libbaml_cffi.dylib",
		url:      "https://github.com/BoundaryML/baml/releases/download/" + BAMLVersion + "/libbaml_cffi-x86_64-apple-darwin.dylib",
	},
	"darwin/arm64": {
		filename: "libbaml_cffi.dylib",
		url:      "https://github.com/BoundaryML/baml/releases/download/" + BAMLVersion + "/libbaml_cffi-aarch64-apple-darwin.dylib",
	},
	"linux/amd64": {
		filename: "libbaml_cffi.so",
		url:      "https://github.com/BoundaryML/baml/releases/download/" + BAMLVersion + "/libbaml_cffi-x86_64-unknown-linux-gnu.so",
	},
	"linux/arm64": {
		filename: "libbaml_cffi.so",
		url:      "https://github.com/BoundaryML/baml/releases/download/" + BAMLVersion + "/libbaml_cffi-aarch64-unknown-linux-gnu.so",
	},
	"windows/amd64": {
		filename: "baml_cffi.dll",
		url:      "https://github.com/BoundaryML/baml/releases/download/" + BAMLVersion + "/baml_cffi-x86_64-pc-windows-msvc.dll",
	},
	"windows/arm64": {
		filename: "baml_cffi.dll",
		url:      "https://github.com/BoundaryML/baml/releases/download/" + BAMLVersion + "/baml_cffi-aarch64-pc-windows-msvc.dll",
	},
}

// Unsupported platforms (BAML doesn't provide binaries)
var unsupportedPlatforms = map[string]bool{
	"linux/arm":     true, // 32-bit ARM
	"linux/riscv64": true,
}

func init() {
	if err := Setup(); err != nil {
		// Don't fail hard - AI features just won't work
		fmt.Fprintf(os.Stderr, "Warning: BAML setup failed: %v\n", err)
		fmt.Fprintf(os.Stderr, "AI features will be unavailable.\n")
	}
}

// Setup ensures the BAML native library is available.
// It downloads the library if not present and sets BAML_LIBRARY_PATH.
func Setup() error {
	// Check if BAML_LIBRARY_PATH is already set (e.g., in Docker)
	if os.Getenv("BAML_LIBRARY_PATH") != "" {
		return nil
	}

	platform := runtime.GOOS + "/" + runtime.GOARCH

	// Check if platform is unsupported
	if unsupportedPlatforms[platform] {
		return fmt.Errorf("BAML is not supported on %s", platform)
	}

	info, ok := libraryInfo[platform]
	if !ok {
		return fmt.Errorf("unsupported platform: %s", platform)
	}

	// Get library directory
	libDir, err := getLibraryDir()
	if err != nil {
		return fmt.Errorf("failed to get library directory: %w", err)
	}

	libPath := filepath.Join(libDir, info.filename)

	// Check if library already exists
	if _, err := os.Stat(libPath); err == nil {
		// Library exists, set the path
		os.Setenv("BAML_LIBRARY_PATH", libPath)
		return nil
	}

	// Download the library
	fmt.Fprintf(os.Stderr, "Downloading AI components (one-time setup)... ")

	if err := downloadLibrary(info.url, libPath); err != nil {
		fmt.Fprintf(os.Stderr, "failed\n")
		return fmt.Errorf("failed to download BAML library: %w", err)
	}

	fmt.Fprintf(os.Stderr, "done\n")

	// Set the environment variable
	os.Setenv("BAML_LIBRARY_PATH", libPath)

	return nil
}

// getLibraryDir returns the directory where BAML libraries are stored.
func getLibraryDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	libDir := filepath.Join(homeDir, ".whodb-cli", "lib")

	// Create directory if it doesn't exist
	if err := os.MkdirAll(libDir, 0755); err != nil {
		return "", err
	}

	return libDir, nil
}

// downloadLibrary downloads a file from url to destPath.
func downloadLibrary(url, destPath string) error {
	// Create a temporary file in the same directory
	destDir := filepath.Dir(destPath)
	tmpFile, err := os.CreateTemp(destDir, "baml-download-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	// Clean up temp file on error
	defer func() {
		if tmpPath != "" {
			os.Remove(tmpPath)
		}
	}()

	// Download the file
	resp, err := http.Get(url)
	if err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		tmpFile.Close()
		return fmt.Errorf("download failed with status: %s", resp.Status)
	}

	// Copy to temp file
	_, err = io.Copy(tmpFile, resp.Body)
	if err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to write file: %w", err)
	}

	// Close before rename
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Make executable on Unix
	if !strings.HasSuffix(destPath, ".dll") {
		if err := os.Chmod(tmpPath, 0755); err != nil {
			return fmt.Errorf("failed to set permissions: %w", err)
		}
	}

	// Atomic rename
	if err := os.Rename(tmpPath, destPath); err != nil {
		return fmt.Errorf("failed to move file: %w", err)
	}

	// Clear tmpPath so defer doesn't try to remove it
	tmpPath = ""

	return nil
}
