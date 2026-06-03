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

package history

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/clidey/whodb/cli/internal/config"
)

type Entry struct {
	ID        string    `json:"id"`
	Query     string    `json:"query"`
	Success   bool      `json:"success"`
	Timestamp time.Time `json:"timestamp"`
	Database  string    `json:"database"`
}

type Manager struct {
	entries    []Entry
	maxEntries int
	persist    bool
	appendOnly bool
}

// NewManagerWithConfig creates a history manager using the provided CLI
// configuration. When cfg is nil, it loads configuration from disk.
func NewManagerWithConfig(cfg *config.Config) (*Manager, error) {
	if cfg == nil {
		var err error
		cfg, err = config.LoadConfig()
		if err != nil {
			return nil, fmt.Errorf("error loading config: %w", err)
		}
	}

	m := &Manager{
		entries:    []Entry{},
		maxEntries: cfg.History.MaxEntries,
		persist:    cfg.History.Persist,
		appendOnly: true,
	}

	if m.persist {
		if err := m.load(); err != nil {
			return nil, err
		}
	}

	return m, nil
}

func NewManager() (*Manager, error) {
	return NewManagerWithConfig(nil)
}

func (m *Manager) Add(query string, success bool, database string) error {
	entry := Entry{
		ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
		Query:     query,
		Success:   success,
		Timestamp: time.Now(),
		Database:  database,
	}

	m.entries = append(m.entries, entry)

	trimmed := false
	if len(m.entries) > m.maxEntries {
		m.entries = m.entries[len(m.entries)-m.maxEntries:]
		trimmed = true
	}

	if !m.persist {
		return nil
	}

	if trimmed || !m.appendOnly {
		return m.rewrite()
	}

	return m.appendEntry(entry)
}

func (m *Manager) GetAll() []Entry {
	if len(m.entries) == 0 {
		return nil
	}

	entries := make([]Entry, len(m.entries))
	for i := range m.entries {
		entries[i] = m.entries[len(m.entries)-1-i]
	}
	return entries
}

// SearchByPrefix returns the most recent successful entry whose query starts
// with the given prefix (case-insensitive). Returns nil if no match found.
func (m *Manager) SearchByPrefix(prefix string) *Entry {
	if prefix == "" {
		return nil
	}
	lowerPrefix := strings.ToLower(strings.TrimSpace(prefix))
	for i := len(m.entries) - 1; i >= 0; i-- {
		entry := m.entries[i]
		if entry.Success && strings.HasPrefix(strings.ToLower(strings.TrimSpace(entry.Query)), lowerPrefix) {
			// Don't suggest if it's the exact same text
			if strings.TrimSpace(strings.ToLower(entry.Query)) != lowerPrefix {
				return &entry
			}
		}
	}
	return nil
}

func (m *Manager) Get(id string) (*Entry, error) {
	for _, entry := range m.entries {
		if entry.ID == id {
			return &entry, nil
		}
	}
	return nil, fmt.Errorf("entry not found")
}

func (m *Manager) Clear() error {
	m.entries = []Entry{}
	if m.persist {
		path, err := m.getHistoryPath()
		if err != nil {
			return err
		}
		m.appendOnly = true
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("error removing history file: %w", err)
		}
	}
	return nil
}

func (m *Manager) getHistoryPath() (string, error) {
	configDir, err := config.GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "history.json"), nil
}

func (m *Manager) rewrite() error {
	path, err := m.getHistoryPath()
	if err != nil {
		return err
	}

	if len(m.entries) == 0 {
		m.appendOnly = true
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("error removing history file: %w", err)
		}
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("error creating history directory: %w", err)
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("error opening history file: %w", err)
	}
	defer file.Close() //nolint:errcheck

	encoder := json.NewEncoder(file)
	for _, entry := range m.entries {
		if err := encoder.Encode(entry); err != nil {
			return fmt.Errorf("error writing history entry: %w", err)
		}
	}

	m.appendOnly = true

	return nil
}

func (m *Manager) appendEntry(entry Entry) error {
	path, err := m.getHistoryPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("error creating history directory: %w", err)
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return fmt.Errorf("error opening history file: %w", err)
	}
	defer file.Close() //nolint:errcheck

	if err := json.NewEncoder(file).Encode(entry); err != nil {
		return fmt.Errorf("error appending history entry: %w", err)
	}

	m.appendOnly = true
	return nil
}

func (m *Manager) load() error {
	path, err := m.getHistoryPath()
	if err != nil {
		return err
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}

	if fi, err := os.Stat(path); err == nil {
		if fi.Mode().Perm()&0077 != 0 {
			_ = os.Chmod(path, 0600)
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("error reading history file: %w", err)
	}

	if len(strings.TrimSpace(string(data))) == 0 {
		m.entries = []Entry{}
		m.appendOnly = true
		return nil
	}

	var entries []Entry
	if err := json.Unmarshal(data, &entries); err == nil {
		m.entries = entries
		m.appendOnly = false
		return nil
	}

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var entry Entry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			return fmt.Errorf("error unmarshaling history: %w", err)
		}
		m.entries = append(m.entries, entry)
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading history entries: %w", err)
	}

	m.appendOnly = true
	return nil
}
