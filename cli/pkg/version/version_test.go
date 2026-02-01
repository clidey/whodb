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

package version

import (
	"runtime"
	"strings"
	"testing"
)

func TestGet(t *testing.T) {
	info := Get()

	if info.Version == "" {
		t.Error("Version should not be empty")
	}

	if info.GoVersion == "" {
		t.Error("GoVersion should not be empty")
	}

	if info.GoVersion != runtime.Version() {
		t.Errorf("GoVersion = %s, want %s", info.GoVersion, runtime.Version())
	}

	if info.Platform == "" {
		t.Error("Platform should not be empty")
	}

	expectedPlatform := runtime.GOOS + "/" + runtime.GOARCH
	if info.Platform != expectedPlatform {
		t.Errorf("Platform = %s, want %s", info.Platform, expectedPlatform)
	}
}

func TestInfo_String(t *testing.T) {
	info := Info{
		Version:   "1.0.0",
		Commit:    "abc123",
		BuildDate: "2026-01-01",
		GoVersion: "go1.21.0",
		Platform:  "linux/amd64",
	}

	str := info.String()

	// Check that all fields are present in the string output
	if !strings.Contains(str, "1.0.0") {
		t.Error("String() should contain version")
	}
	if !strings.Contains(str, "abc123") {
		t.Error("String() should contain commit")
	}
	if !strings.Contains(str, "2026-01-01") {
		t.Error("String() should contain build date")
	}
	if !strings.Contains(str, "go1.21.0") {
		t.Error("String() should contain Go version")
	}
	if !strings.Contains(str, "linux/amd64") {
		t.Error("String() should contain platform")
	}
}

func TestShort(t *testing.T) {
	// Save original values
	origVersion := Version
	origCommit := Commit
	defer func() {
		Version = origVersion
		Commit = origCommit
	}()

	Version = "1.2.3"
	Commit = "def456"

	short := Short()

	if !strings.Contains(short, "1.2.3") {
		t.Errorf("Short() = %s, should contain version 1.2.3", short)
	}
	if !strings.Contains(short, "def456") {
		t.Errorf("Short() = %s, should contain commit def456", short)
	}
	if !strings.Contains(short, "whodb-cli") {
		t.Errorf("Short() = %s, should contain 'whodb-cli'", short)
	}
}

func TestDefaultValues(t *testing.T) {
	// Test that default values are set
	if Version == "" {
		t.Error("Version should have a default value")
	}
	if Commit == "" {
		t.Error("Commit should have a default value")
	}
	if BuildDate == "" {
		t.Error("BuildDate should have a default value")
	}
}
