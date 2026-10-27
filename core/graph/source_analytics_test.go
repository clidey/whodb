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

package graph

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/internal/testutil"
	"github.com/clidey/whodb/core/src/analytics"
	"github.com/clidey/whodb/core/src/engine"
)

func TestSourceQueryAnalyticsCaptureOutcomesWithoutPrivateContent(t *testing.T) {
	type capturedEvent struct {
		Event      string         `json:"event"`
		Properties map[string]any `json:"properties"`
	}
	var mu sync.Mutex
	var events []capturedEvent
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var batch struct {
			Batch []capturedEvent `json:"batch"`
		}
		if err := json.NewDecoder(r.Body).Decode(&batch); err != nil {
			t.Errorf("decode capture batch: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		mu.Lock()
		events = append(events, batch.Batch...)
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	if err := analytics.Initialize(analytics.Config{
		APIKey: "test-key", Host: server.URL, Source: "backend", Edition: "ce",
		Environment: "test", AppVersion: "test", SuppressClientLogs: true,
	}); err != nil {
		t.Fatal(err)
	}
	defer analytics.Shutdown()
	analytics.SetEnabled(true)
	t.Cleanup(func() { analytics.SetEnabled(false) })

	plugin := testutil.NewPluginMock(engine.DatabaseType("Postgres"))
	plugin.StorageUnitExistsFunc = func(*engine.PluginConfig, string, string) (bool, error) { return true, nil }
	plugin.GetRowsFunc = func(*engine.PluginConfig, *engine.GetRowsRequest) (*engine.GetRowsResult, error) {
		return &engine.GetRowsResult{
			Columns: []engine.Column{{Name: "private-column-name", Type: "TEXT"}},
			Rows:    [][]string{{"private-row-value"}}, TotalCount: 100,
		}, nil
	}
	plugin.GetColumnsForTableFunc = func(*engine.PluginConfig, string, string) ([]engine.Column, error) {
		return []engine.Column{{Name: "private-column-name", Type: "TEXT"}}, nil
	}
	plugin.RawExecuteFunc = func(_ *engine.PluginConfig, query string, _ ...any) (*engine.GetRowsResult, error) {
		switch {
		case strings.Contains(query, "private-error-message"):
			return nil, errors.New("invalid query: private-error-message")
		case strings.Contains(query, "empty"):
			return &engine.GetRowsResult{TotalCount: -1}, nil
		default:
			return &engine.GetRowsResult{Rows: [][]string{{"private-row-value"}}}, nil
		}
	}
	setEngineMock(t, plugin)
	base := testSourceContext("Postgres", map[string]string{"Database": "private-database-name"})
	withID := func(id string) context.Context {
		return analytics.WithMetadata(base, analytics.Metadata{DistinctID: id})
	}
	resolver := (&Resolver{}).Query()
	for _, id := range []string{"", "disabled", "consented-install"} {
		if _, err := resolver.RunSourceQuery(withID(id), "SELECT 'private-query-value'"); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := resolver.RunSourceQuery(withID("consented-install"), "SELECT private-error-message"); err == nil {
		t.Fatal("expected query failure")
	}
	if _, err := resolver.RunSourceQuery(withID("consented-install"), "SELECT empty"); err != nil {
		t.Fatal(err)
	}
	if _, err := resolver.SourceRows(withID("consented-install"), testSourceRef(model.SourceObjectKindTable, "private-database-name", "public", "private-table-name"), nil, nil, 5, 0); err != nil {
		t.Fatal(err)
	}
	analytics.SetEnabled(false)
	if _, err := resolver.RunSourceQuery(withID("consented-install"), "SELECT 'disabled-server'"); err != nil {
		t.Fatal(err)
	}
	analytics.Shutdown()

	mu.Lock()
	defer mu.Unlock()
	if len(events) != 4 {
		t.Fatalf("expected 4 outcome events; missing/disabled identities and disabled capture must emit none, got %d", len(events))
	}
	var rowsSeen, failureSeen, emptySeen bool
	for _, event := range events {
		props := event.Properties
		if props["source"] != "backend" || props["build_edition"] != "ce" || props["source_type"] != "Postgres" || props["duration_bucket"] == nil {
			t.Errorf("missing outcome context: %#v", props)
		}
		if event.Event == "source.rows_queried" {
			rowsSeen = true
			if props["row_count_bucket"] != "one" || props["total_row_count_bucket"] != "21_100" || props["empty_result"] != false {
				t.Errorf("returned rows and total rows must stay separate: %#v", props)
			}
		}
		if props["success"] == false {
			failureSeen = true
			if props["error_code"] != "invalid_input" || props["outcome"] != "failure" {
				t.Errorf("incorrect failure outcome: %#v", props)
			}
		}
		if props["empty_result"] == true {
			emptySeen = true
			if _, exists := props["total_row_count_bucket"]; exists {
				t.Error("an unknown total must not be classified as zero total rows")
			}
		}
	}
	if !rowsSeen || !failureSeen || !emptySeen {
		t.Fatalf("missing row, failure or empty-result outcome: %#v", events)
	}
	encoded, err := json.Marshal(events)
	if err != nil {
		t.Fatal(err)
	}
	for _, private := range []string{"private-query-value", "private-error-message", "private-row-value", "private-column-name", "private-table-name", "private-database-name"} {
		if strings.Contains(string(encoded), private) {
			t.Errorf("capture leaked private content %q", private)
		}
	}
}
