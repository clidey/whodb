// Copyright 2025 Clidey, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package graph

import (
	"context"
	"errors"
	"testing"

	"github.com/clidey/whodb/core/internal/testutil"
	"github.com/clidey/whodb/core/src"
	"github.com/clidey/whodb/core/src/auth"
	"github.com/clidey/whodb/core/src/engine"
)

func TestMapColumnsToModelMergesMetadata(t *testing.T) {
	columns := []engine.Column{
		{
			Name: "id",
			Type: "INT",
		},
		{
			Name: "user_id",
			Type: "INT",
		},
	}

	constraints := map[string]map[string]any{
		"id": {
			"primary": true,
		},
	}

	foreignKeys := map[string]*engine.ForeignKeyRelationship{
		"user_id": {
			ColumnName:       "user_id",
			ReferencedTable:  "users",
			ReferencedColumn: "id",
		},
	}

	result := MapColumnsToModel(columns, constraints, foreignKeys)

	if len(result) != 2 {
		t.Fatalf("expected 2 columns, got %d", len(result))
	}

	if !result[0].IsPrimary {
		t.Fatalf("expected constraint to mark id as primary key")
	}

	if !result[1].IsForeignKey || result[1].ReferencedTable == nil || *result[1].ReferencedTable != "users" {
		t.Fatalf("expected foreign key metadata to be merged")
	}
}

func TestFetchColumnsForStorageUnitValidatesAndEnriches(t *testing.T) {
	mock := testutil.NewPluginMock(engine.DatabaseType("Test"))
	mock.StorageUnitExistsFunc = func(_ *engine.PluginConfig, schema string, storageUnit string) (bool, error) {
		if schema == "public" && storageUnit == "orders" {
			return true, nil
		}
		return false, nil
	}
	mock.GetColumnsForTableFunc = func(_ *engine.PluginConfig, _ string, _ string) ([]engine.Column, error) {
		return []engine.Column{
			{Name: "id", Type: "INT"},
			{Name: "customer_id", Type: "INT"},
		}, nil
	}
	mock.GetColumnConstraintsFunc = func(_ *engine.PluginConfig, _ string, _ string) (map[string]map[string]any, error) {
		return map[string]map[string]any{
			"id": {"primary": true},
		}, nil
	}
	mock.GetForeignKeysFunc = func(_ *engine.PluginConfig, _ string, _ string) (map[string]*engine.ForeignKeyRelationship, error) {
		return map[string]*engine.ForeignKeyRelationship{
			"customer_id": {
				ColumnName:       "customer_id",
				ReferencedTable:  "customers",
				ReferencedColumn: "id",
			},
		}, nil
	}

	config := engine.NewPluginConfig(&engine.Credentials{Type: "Test"})
	result, err := FetchColumnsForStorageUnit(mock, config, "public", "orders")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 columns, got %d", len(result))
	}

	if !result[0].IsPrimary {
		t.Fatalf("expected primary key to be derived from constraints")
	}

	if !result[1].IsForeignKey || result[1].ReferencedTable == nil || *result[1].ReferencedTable != "customers" {
		t.Fatalf("expected foreign key metadata to be added to model")
	}
}

func TestFetchColumnsForStorageUnitFailsWhenMissing(t *testing.T) {
	mock := testutil.NewPluginMock(engine.DatabaseType("Test"))
	mock.StorageUnitExistsFunc = func(_ *engine.PluginConfig, _ string, _ string) (bool, error) {
		return false, nil
	}

	config := engine.NewPluginConfig(&engine.Credentials{Type: "Test"})
	if _, err := FetchColumnsForStorageUnit(mock, config, "public", "missing"); err == nil {
		t.Fatalf("expected validation error when storage unit does not exist")
	}
}

func TestValidateStorageUnit(t *testing.T) {
	mock := testutil.NewPluginMock(engine.DatabaseType("Test"))

	mock.StorageUnitExistsFunc = func(_ *engine.PluginConfig, _ string, _ string) (bool, error) {
		return true, nil
	}
	if err := ValidateStorageUnit(mock, engine.NewPluginConfig(&engine.Credentials{Type: "Test"}), "public", "orders"); err != nil {
		t.Fatalf("expected validation to pass when storage unit exists, got %v", err)
	}

	mock.StorageUnitExistsFunc = func(_ *engine.PluginConfig, _ string, _ string) (bool, error) {
		return false, nil
	}
	if err := ValidateStorageUnit(mock, engine.NewPluginConfig(&engine.Credentials{Type: "Test"}), "public", "missing"); err == nil {
		t.Fatalf("expected error when storage unit does not exist")
	}

	mock.StorageUnitExistsFunc = func(_ *engine.PluginConfig, _ string, _ string) (bool, error) {
		return false, errors.New("validation failed")
	}
	if err := ValidateStorageUnit(mock, engine.NewPluginConfig(&engine.Credentials{Type: "Test"}), "public", "missing"); err == nil {
		t.Fatalf("expected error to bubble up when validation fails")
	}
}

func TestGetPluginForContextUsesEngine(t *testing.T) {
	original := src.MainEngine
	t.Cleanup(func() {
		src.MainEngine = original
	})

	engineInstance := &engine.Engine{}
	plugin := &engine.Plugin{Type: "Test"}
	engineInstance.RegistryPlugin(plugin)
	src.MainEngine = engineInstance

	creds := &engine.Credentials{Type: "Test"}
	ctx := context.WithValue(context.Background(), auth.AuthKey_Credentials, creds)

	chosen, config := GetPluginForContext(ctx)

	if chosen != plugin {
		t.Fatalf("expected engine to return registered plugin, got %v", chosen)
	}
	if config.Credentials != creds {
		t.Fatalf("expected config to carry credentials from context")
	}
}
