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

package history

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewManager(t *testing.T) {
	tempDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", origHome)

	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	if mgr == nil {
		t.Fatal("NewManager returned nil")
	}
}

func TestAdd(t *testing.T) {
	tempDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", origHome)

	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	err = mgr.Add("SELECT * FROM users", true, "testdb")
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	entries := mgr.GetAll()
	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(entries))
	}

	if entries[0].Query != "SELECT * FROM users" {
		t.Errorf("Expected query 'SELECT * FROM users', got '%s'", entries[0].Query)
	}

	if entries[0].Database != "testdb" {
		t.Errorf("Expected database 'testdb', got '%s'", entries[0].Database)
	}

	if !entries[0].Success {
		t.Error("Expected Success to be true")
	}
}

func TestGetAll(t *testing.T) {
	tempDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", origHome)

	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	entries := mgr.GetAll()
	if len(entries) != 0 {
		t.Errorf("Expected 0 entries initially, got %d", len(entries))
	}

	mgr.Add("SELECT 1", true, "testdb")
	mgr.Add("SELECT 2", true, "testdb")

	entries = mgr.GetAll()
	if len(entries) != 2 {
		t.Fatalf("Expected 2 entries, got %d", len(entries))
	}
}

func TestClear(t *testing.T) {
	tempDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", origHome)

	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	mgr.Add("SELECT 1", true, "testdb")
	mgr.Add("SELECT 2", true, "testdb")

	entries := mgr.GetAll()
	if len(entries) != 2 {
		t.Fatalf("Expected 2 entries before clear, got %d", len(entries))
	}

	err = mgr.Clear()
	if err != nil {
		t.Fatalf("Clear failed: %v", err)
	}

	entries = mgr.GetAll()
	if len(entries) != 0 {
		t.Errorf("Expected 0 entries after clear, got %d", len(entries))
	}
}

func TestPersistence(t *testing.T) {
	tempDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", origHome)

	mgr1, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	err = mgr1.Add("SELECT * FROM test", true, "testdb")
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	historyPath := filepath.Join(tempDir, ".whodb-cli", "history.json")
	if _, err := os.Stat(historyPath); os.IsNotExist(err) {
		t.Fatalf("History file was not created at %s", historyPath)
	}

	mgr2, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed on reload: %v", err)
	}

	entries := mgr2.GetAll()
	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry after reload, got %d", len(entries))
	}

	if entries[0].Query != "SELECT * FROM test" {
		t.Errorf("Expected query 'SELECT * FROM test', got '%s'", entries[0].Query)
	}
}

func TestAdd_MultipleDatabases(t *testing.T) {
	tempDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", origHome)

	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	mgr.Add("SELECT * FROM db1_users", true, "db1")
	mgr.Add("SELECT * FROM db2_users", true, "db2")

	entries := mgr.GetAll()
	if len(entries) != 2 {
		t.Fatalf("Expected 2 entries, got %d", len(entries))
	}

	db1Count := 0
	db2Count := 0
	for _, e := range entries {
		if e.Database == "db1" {
			db1Count++
		}
		if e.Database == "db2" {
			db2Count++
		}
	}

	if db1Count != 1 {
		t.Errorf("Expected 1 entry for db1, got %d", db1Count)
	}
	if db2Count != 1 {
		t.Errorf("Expected 1 entry for db2, got %d", db2Count)
	}
}

func TestAdd_FailedQueries(t *testing.T) {
	tempDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", origHome)

	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	mgr.Add("SELECT * FROM nonexistent", false, "testdb")

	entries := mgr.GetAll()
	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(entries))
	}

	if entries[0].Success {
		t.Error("Expected Success to be false")
	}
}

func TestGet(t *testing.T) {
	tempDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", origHome)

	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	mgr.Add("SELECT * FROM users", true, "testdb")

	entries := mgr.GetAll()
	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(entries))
	}

	id := entries[0].ID

	entry, err := mgr.Get(id)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if entry.Query != "SELECT * FROM users" {
		t.Errorf("Expected query 'SELECT * FROM users', got '%s'", entry.Query)
	}
}

func TestGet_NotFound(t *testing.T) {
	tempDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", origHome)

	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	_, err = mgr.Get("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent entry")
	}
}
