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

package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/clidey/whodb/cli/internal/config"
	"github.com/clidey/whodb/cli/internal/schemadiff"
)

func createDiffTestSQLitePath(t *testing.T, name string) string {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), name)
	file, err := os.Create(dbPath)
	if err != nil {
		t.Fatalf("create sqlite db: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close sqlite db: %v", err)
	}
	return dbPath
}

func saveDiffTestConnection(t *testing.T, conn config.Connection) {
	t.Helper()

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	cfg.AddConnection(conn)
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}
}

func TestMainModel_Update_CtrlVOpensSchemaDiff(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	setupTestEnv(t)

	currentDB := createDiffTestSQLitePath(t, "current.db")
	otherDB := createDiffTestSQLitePath(t, "other.db")

	currentConn := &config.Connection{
		Name:     "current",
		Type:     "Sqlite3",
		Host:     currentDB,
		Database: currentDB,
	}
	saveDiffTestConnection(t, config.Connection{
		Name:     "other",
		Type:     "Sqlite3",
		Host:     otherDB,
		Database: otherDB,
	})

	m := NewMainModelWithConnection(currentConn)
	if m.err != nil {
		t.Skipf("Skipping test - database plugin not available: %v", m.err)
	}

	m.mode = ViewBrowser
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	_, _ = m.Update(tea.KeyPressMsg{Code: 'v', Mod: tea.ModCtrl})

	if m.mode != ViewDiff {
		t.Fatalf("expected mode ViewDiff after ctrl+v, got %v", m.mode)
	}
	if len(m.diffView.connections) < 2 {
		t.Fatalf("expected at least two selectable connections, got %d", len(m.diffView.connections))
	}
	if !strings.Contains(m.diffView.currentConnectionLabel(m.diffView.fromIndex), "current") {
		t.Fatalf("expected current connection to be selected first, got %q", m.diffView.currentConnectionLabel(m.diffView.fromIndex))
	}
	if !m.diffView.editing {
		t.Fatal("expected diff view to start in editing mode")
	}
}

func TestSchemaDiffView_ResultModeRendersSharedSummary(t *testing.T) {
	setupTestEnv(t)

	m := NewMainModel()
	if m.err != nil {
		t.Fatalf("NewMainModel failed: %v", m.err)
	}

	v := NewSchemaDiffView(m)
	v.width = 120
	v.height = 40
	v.connections = []config.Connection{
		{Name: "from", Type: "Postgres"},
		{Name: "to", Type: "MySQL"},
	}

	result := &schemadiff.Result{
		From: schemadiff.SchemaReference{Connection: "from", Type: "Postgres", Schema: "public"},
		To:   schemadiff.SchemaReference{Connection: "to", Type: "MySQL", Schema: "app"},
		Summary: schemadiff.Summary{
			HasDifferences:       true,
			AddedStorageUnits:    1,
			RemovedStorageUnits:  0,
			ChangedStorageUnits:  2,
			AddedColumns:         3,
			RemovedColumns:       1,
			ChangedColumns:       1,
			AddedRelationships:   1,
			RemovedRelationships: 0,
			ChangedRelationships: 1,
		},
	}

	updated, _ := v.Update(schemaDiffResultMsg{result: result})
	view := updated.View()

	if !strings.Contains(view, "Schema Diff") {
		t.Fatalf("expected schema diff title, got: %s", view)
	}
	if !strings.Contains(view, "Relationships +1 -0 ~1") {
		t.Fatalf("expected relationship summary, got: %s", view)
	}
	if updated.helpSafe() != true {
		t.Fatal("expected result mode to be help-safe")
	}
}

func TestSchemaDiffView_HelpSafe_FalseWhileEditing(t *testing.T) {
	setupTestEnv(t)

	m := NewMainModel()
	if m.err != nil {
		t.Fatalf("NewMainModel failed: %v", m.err)
	}

	v := NewSchemaDiffView(m)
	v.editing = true
	if v.helpSafe() {
		t.Fatal("expected editing mode to disable the help overlay")
	}
}
