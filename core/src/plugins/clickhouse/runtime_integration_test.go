//go:build integration

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

package clickhouse

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/clidey/whodb/core/src/engine"
)

func clickHouseIntegrationPlugin(t *testing.T) *ClickHousePlugin {
	t.Helper()

	plugin, ok := NewClickHousePlugin().PluginFunctions.(*ClickHousePlugin)
	if !ok {
		t.Fatalf("unexpected clickhouse plugin type %T", NewClickHousePlugin().PluginFunctions)
	}
	return plugin
}

func clickHouseIntegrationConfig() *engine.PluginConfig {
	return engine.NewPluginConfig(&engine.Credentials{
		Type:     string(engine.DatabaseType_ClickHouse),
		Hostname: "localhost",
		Username: "user",
		Password: "password",
		Database: "test_db",
		Advanced: []engine.Record{{Key: "Port", Value: "9000"}},
	})
}

func waitForClickHouseOrders(t *testing.T, plugin *ClickHousePlugin, config *engine.PluginConfig) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Minute)
	for time.Now().Before(deadline) {
		if !plugin.IsAvailable(context.Background(), config) {
			time.Sleep(2 * time.Second)
			continue
		}

		exists, err := plugin.StorageUnitExists(config, "test_db", "orders")
		if err == nil && exists {
			count, countErr := plugin.GetRowCount(config, "test_db", "orders", nil)
			if countErr == nil && count > 0 {
				return
			}
		}

		time.Sleep(2 * time.Second)
	}

	t.Fatal("timed out waiting for seeded clickhouse data")
}

func TestClickHouseSeededRuntimePaths(t *testing.T) {
	plugin := clickHouseIntegrationPlugin(t)
	config := clickHouseIntegrationConfig()
	waitForClickHouseOrders(t, plugin, config)

	databases, err := plugin.GetDatabases(config)
	if err != nil {
		t.Fatalf("GetDatabases failed: %v", err)
	}
	if !slices.Contains(databases, "test_db") {
		t.Fatalf("expected databases %#v to contain test_db", databases)
	}

	if _, err := plugin.GetAllSchemas(config); !errors.Is(err, errors.ErrUnsupported) {
		t.Fatalf("expected clickhouse GetAllSchemas to be unsupported, got %v", err)
	}

	columns, err := plugin.GetColumnsForTable(config, "test_db", "orders")
	if err != nil {
		t.Fatalf("GetColumnsForTable failed: %v", err)
	}
	foundPrimaryID := false
	for _, column := range columns {
		if column.Name == "id" && column.IsPrimary {
			foundPrimaryID = true
			break
		}
	}
	if !foundPrimaryID {
		t.Fatalf("expected clickhouse id column to be marked primary, got %#v", columns)
	}

	rawRows, err := plugin.RawExecute(config, "SELECT status FROM test_db.orders ORDER BY id LIMIT 1")
	if err != nil {
		t.Fatalf("RawExecute failed: %v", err)
	}
	if len(rawRows.Rows) != 1 {
		t.Fatalf("expected one clickhouse row, got %#v", rawRows.Rows)
	}

	sslStatus, err := plugin.GetSSLStatus(config)
	if err != nil {
		t.Fatalf("GetSSLStatus failed: %v", err)
	}
	if sslStatus.IsEnabled || sslStatus.Mode != "disabled" {
		t.Fatalf("expected clickhouse SSL to be disabled, got %#v", sslStatus)
	}

	multiStatementConfig := *config
	multiStatementConfig.MultiStatement = true
	if _, err := plugin.RawExecute(&multiStatementConfig, "SELECT 1; SELECT 2;"); !errors.Is(err, engine.ErrMultiStatementUnsupported) {
		t.Fatalf("expected clickhouse multi-statement error, got %v", err)
	}
}

func TestClickHouseMutationRuntimePaths(t *testing.T) {
	plugin := clickHouseIntegrationPlugin(t)
	config := clickHouseIntegrationConfig()
	waitForClickHouseOrders(t, plugin, config)

	table := fmt.Sprintf("intg_ch_%d", time.Now().UnixNano())
	_, _ = plugin.RawExecute(config, fmt.Sprintf("DROP TABLE IF EXISTS test_db.%s SYNC", table))
	defer plugin.RawExecute(config, fmt.Sprintf("DROP TABLE IF EXISTS test_db.%s SYNC", table))

	created, err := plugin.AddStorageUnit(config, "test_db", table, []engine.Record{
		{Key: "id", Value: "UInt32", Extra: map[string]string{"Primary": "true", "Nullable": "false"}},
		{Key: "tags", Value: "Array(String)", Extra: map[string]string{"Primary": "false", "Nullable": "false"}},
		{Key: "status", Value: "String", Extra: map[string]string{"Primary": "false", "Nullable": "false"}},
	})
	if err != nil || !created {
		t.Fatalf("AddStorageUnit failed: created=%t err=%v", created, err)
	}
	for range 10 {
		exists, existsErr := plugin.StorageUnitExists(config, "test_db", table)
		if existsErr == nil && exists {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	if _, err := plugin.RawExecute(config, fmt.Sprintf("INSERT INTO test_db.%s (id, tags, status) VALUES (1, ['alpha'], 'pending')", table)); err != nil {
		t.Fatalf("failed to insert clickhouse runtime row: %v", err)
	}

	updated, err := plugin.UpdateStorageUnit(config, "test_db", table, map[string]string{
		"id":     "1",
		"tags":   "[beta,gamma]",
		"status": "complete",
	}, []string{"tags", "status"})
	if err != nil || !updated {
		t.Fatalf("UpdateStorageUnit failed: updated=%t err=%v", updated, err)
	}

	var updatedRows *engine.GetRowsResult
	for range 10 {
		time.Sleep(200 * time.Millisecond)
		updatedRows, err = plugin.RawExecute(config, fmt.Sprintf("SELECT status, toString(tags) FROM test_db.%s WHERE id = 1", table))
		if err != nil {
			t.Fatalf("failed to read clickhouse updated row: %v", err)
		}
		if len(updatedRows.Rows) > 0 {
			allUpdated := true
			for _, row := range updatedRows.Rows {
				if !strings.Contains(row[0], "complete") || !strings.Contains(row[1], "beta") {
					allUpdated = false
					break
				}
			}
			if allUpdated {
				break
			}
		}
	}
	if updatedRows == nil || len(updatedRows.Rows) == 0 {
		t.Fatalf("expected clickhouse updated rows, got %#v", updatedRows)
	}
	for _, row := range updatedRows.Rows {
		if !strings.Contains(row[0], "complete") || !strings.Contains(row[1], "beta") {
			t.Fatalf("unexpected clickhouse updated row %#v", row)
		}
	}

	cleared, err := plugin.ClearTableData(config, "test_db", table)
	if err != nil || !cleared {
		t.Fatalf("ClearTableData failed: cleared=%t err=%v", cleared, err)
	}

	for range 10 {
		time.Sleep(200 * time.Millisecond)
		count, countErr := plugin.GetRowCount(config, "test_db", table, nil)
		if countErr != nil {
			t.Fatalf("GetRowCount failed after clickhouse clear: %v", countErr)
		}
		if count == 0 {
			return
		}
	}

	t.Fatalf("expected clickhouse table %q to be empty after ClearTableData", table)
}
