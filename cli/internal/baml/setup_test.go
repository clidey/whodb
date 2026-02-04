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

package baml

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestGetLibraryDir(t *testing.T) {
	dir, err := getLibraryDir()
	if err != nil {
		t.Fatalf("getLibraryDir() error = %v", err)
	}

	if dir == "" {
		t.Error("getLibraryDir() returned empty string")
	}

	// Should be under user's home directory
	homeDir, _ := os.UserHomeDir()
	if !strings.HasPrefix(dir, homeDir) {
		t.Errorf("getLibraryDir() = %s, should be under home directory %s", dir, homeDir)
	}

	// Should end with expected path
	expectedSuffix := filepath.Join(".whodb-cli", "lib")
	if !strings.HasSuffix(dir, expectedSuffix) {
		t.Errorf("getLibraryDir() = %s, should end with %s", dir, expectedSuffix)
	}

	// Directory should exist after calling getLibraryDir
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Errorf("getLibraryDir() should create the directory, but %s doesn't exist", dir)
	}
}

func TestLibraryInfo_CurrentPlatform(t *testing.T) {
	platform := runtime.GOOS + "/" + runtime.GOARCH

	// Skip if on unsupported platform
	if unsupportedPlatforms[platform] {
		t.Skipf("Skipping test on unsupported platform: %s", platform)
	}

	info, ok := libraryInfo[platform]
	if !ok {
		t.Skipf("No library info for platform: %s", platform)
	}

	if info.filename == "" {
		t.Error("Library filename should not be empty")
	}

	if info.url == "" {
		t.Error("Library URL should not be empty")
	}

	// URL should be a valid GitHub release URL
	if !strings.Contains(info.url, "github.com/BoundaryML/baml/releases") {
		t.Errorf("Library URL should be a GitHub release URL, got %s", info.url)
	}
}

func TestUnsupportedPlatforms(t *testing.T) {
	// Verify known unsupported platforms are marked
	knownUnsupported := []string{"linux/arm", "linux/riscv64"}

	for _, platform := range knownUnsupported {
		if !unsupportedPlatforms[platform] {
			t.Errorf("Platform %s should be marked as unsupported", platform)
		}
	}
}

func TestSetup_WithExistingEnvVar(t *testing.T) {
	// Set BAML_LIBRARY_PATH
	originalValue := os.Getenv("BAML_LIBRARY_PATH")
	os.Setenv("BAML_LIBRARY_PATH", "/some/existing/path")
	defer func() {
		if originalValue == "" {
			os.Unsetenv("BAML_LIBRARY_PATH")
		} else {
			os.Setenv("BAML_LIBRARY_PATH", originalValue)
		}
	}()

	// Setup should return early when env var is set
	err := Setup()
	if err != nil {
		t.Errorf("Setup() should succeed when BAML_LIBRARY_PATH is set, got error: %v", err)
	}
}

func TestBAMLVersion(t *testing.T) {
	// Verify BAMLVersion is set
	if BAMLVersion == "" {
		t.Error("BAMLVersion should not be empty")
	}

	// Should be a semantic version format
	parts := strings.Split(BAMLVersion, ".")
	if len(parts) < 2 {
		t.Errorf("BAMLVersion = %s, should be in semantic version format", BAMLVersion)
	}
}

func TestLibraryInfo_AllPlatformsHaveFilenames(t *testing.T) {
	for platform, info := range libraryInfo {
		if info.filename == "" {
			t.Errorf("Platform %s has empty filename", platform)
		}
		if info.url == "" {
			t.Errorf("Platform %s has empty URL", platform)
		}
	}
}

func TestLibraryInfo_FilenameExtensions(t *testing.T) {
	for platform, info := range libraryInfo {
		var expectedExt string
		switch {
		case strings.HasPrefix(platform, "darwin"):
			expectedExt = ".dylib"
		case strings.HasPrefix(platform, "linux"):
			expectedExt = ".so"
		case strings.HasPrefix(platform, "windows"):
			expectedExt = ".dll"
		}

		if expectedExt != "" && !strings.HasSuffix(info.filename, expectedExt) {
			t.Errorf("Platform %s: filename %s should have extension %s", platform, info.filename, expectedExt)
		}
	}
}
