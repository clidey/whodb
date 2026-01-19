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

package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/clidey/whodb/core/src/common/datadir"
)

// testOptions returns datadir options pointing to a temp directory.
func setupTestDir(t *testing.T) (datadir.Options, func()) {
	t.Helper()

	tempDir := t.TempDir()
	oldXDG := os.Getenv("XDG_DATA_HOME")
	os.Setenv("XDG_DATA_HOME", tempDir)
	ResetConfigPath()

	cleanup := func() {
		os.Setenv("XDG_DATA_HOME", oldXDG)
		ResetConfigPath()
	}

	return datadir.Options{AppName: "whodb-test"}, cleanup
}

func TestWriteSection_CreatesConfigFile(t *testing.T) {
	opts, cleanup := setupTestDir(t)
	defer cleanup()

	type TestSection struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	section := TestSection{Name: "test", Value: 42}
	err := WriteSection("test", section, opts)
	if err != nil {
		t.Fatalf("WriteSection failed: %v", err)
	}

	// Verify the file was created
	path, err := GetPath(opts)
	if err != nil {
		t.Fatalf("GetPath failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read config file: %v", err)
	}

	var raw RawConfig
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal config: %v", err)
	}

	if _, exists := raw["test"]; !exists {
		t.Error("test section should exist in config")
	}
}

func TestWriteSection_PreservesOtherSections(t *testing.T) {
	opts, cleanup := setupTestDir(t)
	defer cleanup()

	type Section struct {
		Data string `json:"data"`
	}

	// Write first section
	section1 := Section{Data: "first"}
	if err := WriteSection("section1", section1, opts); err != nil {
		t.Fatalf("WriteSection (1) failed: %v", err)
	}

	// Write second section
	section2 := Section{Data: "second"}
	if err := WriteSection("section2", section2, opts); err != nil {
		t.Fatalf("WriteSection (2) failed: %v", err)
	}

	// Verify both sections exist
	var read1, read2 Section
	if err := ReadSection("section1", &read1, opts); err != nil {
		t.Fatalf("ReadSection (1) failed: %v", err)
	}
	if err := ReadSection("section2", &read2, opts); err != nil {
		t.Fatalf("ReadSection (2) failed: %v", err)
	}

	if read1.Data != "first" {
		t.Errorf("expected section1 data 'first', got '%s'", read1.Data)
	}
	if read2.Data != "second" {
		t.Errorf("expected section2 data 'second', got '%s'", read2.Data)
	}
}

func TestReadSection_MissingFile(t *testing.T) {
	opts, cleanup := setupTestDir(t)
	defer cleanup()

	type Section struct {
		Data string `json:"data"`
	}

	var section Section
	section.Data = "default" // Set a default

	// Reading from non-existent file should not error and leave default
	err := ReadSection("test", &section, opts)
	if err != nil {
		t.Fatalf("ReadSection should not error for missing file: %v", err)
	}

	if section.Data != "default" {
		t.Errorf("expected default value to be preserved, got '%s'", section.Data)
	}
}

func TestReadSection_MissingSection(t *testing.T) {
	opts, cleanup := setupTestDir(t)
	defer cleanup()

	type Section struct {
		Data string `json:"data"`
	}

	// Create a config with one section
	section1 := Section{Data: "exists"}
	if err := WriteSection("section1", section1, opts); err != nil {
		t.Fatalf("WriteSection failed: %v", err)
	}

	// Try to read a different section
	var section2 Section
	section2.Data = "default"

	err := ReadSection("nonexistent", &section2, opts)
	if err != nil {
		t.Fatalf("ReadSection should not error for missing section: %v", err)
	}

	if section2.Data != "default" {
		t.Errorf("expected default value to be preserved, got '%s'", section2.Data)
	}
}

func TestReadSection_CorruptedJSON(t *testing.T) {
	opts, cleanup := setupTestDir(t)
	defer cleanup()

	// Get the config path and create a corrupted file
	path, err := GetPath(opts)
	if err != nil {
		t.Fatalf("GetPath failed: %v", err)
	}

	// Create the directory
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}

	// Write corrupted JSON
	if err := os.WriteFile(path, []byte("{invalid json"), 0644); err != nil {
		t.Fatalf("failed to write corrupted config: %v", err)
	}

	type Section struct {
		Data string `json:"data"`
	}

	var section Section
	err = ReadSection("test", &section, opts)
	if err == nil {
		t.Error("ReadSection should error for corrupted JSON")
	}
}

func TestWriteSection_AtomicWrite(t *testing.T) {
	opts, cleanup := setupTestDir(t)
	defer cleanup()

	type Section struct {
		Data string `json:"data"`
	}

	// Write initial data
	section := Section{Data: "initial"}
	if err := WriteSection("test", section, opts); err != nil {
		t.Fatalf("WriteSection failed: %v", err)
	}

	// Update the data
	section.Data = "updated"
	if err := WriteSection("test", section, opts); err != nil {
		t.Fatalf("WriteSection (update) failed: %v", err)
	}

	// Read back
	var read Section
	if err := ReadSection("test", &read, opts); err != nil {
		t.Fatalf("ReadSection failed: %v", err)
	}

	if read.Data != "updated" {
		t.Errorf("expected 'updated', got '%s'", read.Data)
	}

	// Verify no temp files left behind
	path, _ := GetPath(opts)
	dir := filepath.Dir(path)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("failed to read dir: %v", err)
	}

	for _, entry := range entries {
		if filepath.Ext(entry.Name()) == ".tmp" {
			t.Errorf("temp file should not exist: %s", entry.Name())
		}
	}
}

func TestRawConfig(t *testing.T) {
	// Test that RawConfig is a map of json.RawMessage
	raw := make(RawConfig)
	raw["test"] = json.RawMessage(`{"key":"value"}`)

	data, err := json.Marshal(raw)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var parsed RawConfig
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	// Verify the key exists and can be parsed
	var inner struct {
		Key string `json:"key"`
	}
	if err := json.Unmarshal(parsed["test"], &inner); err != nil {
		t.Fatalf("failed to unmarshal inner: %v", err)
	}
	if inner.Key != "value" {
		t.Errorf("expected key='value', got key='%s'", inner.Key)
	}
}

func TestSectionConstants(t *testing.T) {
	// Verify section constants are defined
	if SectionCLI != "cli" {
		t.Errorf("expected SectionCLI to be 'cli', got '%s'", SectionCLI)
	}
	if SectionAWS != "aws" {
		t.Errorf("expected SectionAWS to be 'aws', got '%s'", SectionAWS)
	}
	if SectionDesktop != "desktop" {
		t.Errorf("expected SectionDesktop to be 'desktop', got '%s'", SectionDesktop)
	}
}
