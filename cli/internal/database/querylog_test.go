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

package database

import (
	"testing"
	"time"

	"github.com/clidey/whodb/core/src/engine"
)

func TestQueryLogEntry(t *testing.T) {
	entry := QueryLogEntry{
		Query:     "SELECT 1",
		Timestamp: time.Now(),
		Duration:  50 * time.Millisecond,
		Success:   true,
		RowCount:  1,
	}

	if entry.Query != "SELECT 1" {
		t.Errorf("expected query 'SELECT 1', got %q", entry.Query)
	}
	if !entry.Success {
		t.Error("expected Success to be true")
	}
	if entry.RowCount != 1 {
		t.Errorf("expected RowCount 1, got %d", entry.RowCount)
	}
	if entry.Error != "" {
		t.Errorf("expected empty Error, got %q", entry.Error)
	}
}

func TestQueryLogEntryError(t *testing.T) {
	entry := QueryLogEntry{
		Query:     "SELECT * FROM nonexistent",
		Timestamp: time.Now(),
		Duration:  10 * time.Millisecond,
		Success:   false,
		Error:     "table not found",
	}

	if entry.Success {
		t.Error("expected Success to be false")
	}
	if entry.Error != "table not found" {
		t.Errorf("expected error 'table not found', got %q", entry.Error)
	}
}

func TestLogQueryRingBuffer(t *testing.T) {
	m := &Manager{
		queryLog: nil,
	}

	// Add MaxQueryLogEntries + 20 entries
	for i := 0; i < MaxQueryLogEntries+20; i++ {
		m.logQuery("SELECT 1", time.Now(), &engine.GetRowsResult{}, nil)
	}

	if len(m.queryLog) != MaxQueryLogEntries {
		t.Errorf("expected %d entries, got %d", MaxQueryLogEntries, len(m.queryLog))
	}
}

func TestLogQuerySuccess(t *testing.T) {
	m := &Manager{
		queryLog: nil,
	}

	result := &engine.GetRowsResult{
		Rows: [][]string{{"a"}, {"b"}, {"c"}},
	}
	start := time.Now()
	m.logQuery("SELECT * FROM users", start, result, nil)

	entries := m.GetQueryLog()
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	entry := entries[0]
	if entry.Query != "SELECT * FROM users" {
		t.Errorf("expected query 'SELECT * FROM users', got %q", entry.Query)
	}
	if !entry.Success {
		t.Error("expected Success to be true")
	}
	if entry.RowCount != 3 {
		t.Errorf("expected RowCount 3, got %d", entry.RowCount)
	}
	if entry.Duration < 0 {
		t.Error("expected non-negative duration")
	}
}

func TestLogQueryError(t *testing.T) {
	m := &Manager{
		queryLog: nil,
	}

	start := time.Now()
	testErr := ErrReadOnly
	m.logQuery("INSERT INTO users VALUES (1)", start, nil, testErr)

	entries := m.GetQueryLog()
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	entry := entries[0]
	if entry.Success {
		t.Error("expected Success to be false")
	}
	if entry.Error == "" {
		t.Error("expected non-empty Error")
	}
	if entry.RowCount != 0 {
		t.Errorf("expected RowCount 0, got %d", entry.RowCount)
	}
}

func TestGetQueryLogReturnsCopy(t *testing.T) {
	m := &Manager{
		queryLog: nil,
	}

	m.logQuery("SELECT 1", time.Now(), nil, nil)

	log1 := m.GetQueryLog()
	log1[0].Query = "MODIFIED"

	log2 := m.GetQueryLog()
	if log2[0].Query == "MODIFIED" {
		t.Error("GetQueryLog should return a copy, not a reference")
	}
}

func TestLogQueryNilResult(t *testing.T) {
	m := &Manager{
		queryLog: nil,
	}

	m.logQuery("DELETE FROM users", time.Now(), nil, nil)

	entries := m.GetQueryLog()
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].RowCount != 0 {
		t.Errorf("expected RowCount 0 for nil result, got %d", entries[0].RowCount)
	}
}
