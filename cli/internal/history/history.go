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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
}

func NewManager() (*Manager, error) {
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("error loading config: %w", err)
	}

	m := &Manager{
		entries:    []Entry{},
		maxEntries: cfg.History.MaxEntries,
		persist:    cfg.History.Persist,
	}

	if m.persist {
		if err := m.load(); err != nil {
			return nil, err
		}
	}

	return m, nil
}

func (m *Manager) Add(query string, success bool, database string) error {
	entry := Entry{
		ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
		Query:     query,
		Success:   success,
		Timestamp: time.Now(),
		Database:  database,
	}

	m.entries = append([]Entry{entry}, m.entries...)

	if len(m.entries) > m.maxEntries {
		m.entries = m.entries[:m.maxEntries]
	}

	if m.persist {
		return m.save()
	}

	return nil
}

func (m *Manager) GetAll() []Entry {
	return m.entries
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
		return m.save()
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

func (m *Manager) save() error {
	path, err := m.getHistoryPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(m.entries, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling history: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("error writing history file: %w", err)
	}
	// Enforce strict permissions in case file existed with broader perms
	_ = os.Chmod(path, 0600)

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

	if err := json.Unmarshal(data, &m.entries); err != nil {
		return fmt.Errorf("error unmarshaling history: %w", err)
	}

	return nil
}
