package graph

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/internal/testutil"
	"github.com/clidey/whodb/core/src"
	"github.com/clidey/whodb/core/src/auth"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/env"
)

func TestGenerateMockDataRejectsWhenNotAllowed(t *testing.T) {
	originalFlag := env.DisableMockDataGeneration
	t.Cleanup(func() { env.DisableMockDataGeneration = originalFlag })
	env.DisableMockDataGeneration = "*"

	r := &mutationResolver{}
	_, err := r.GenerateMockData(context.Background(), model.MockDataGenerationInput{
		Schema:            "public",
		StorageUnit:       "users",
		RowCount:          10,
		Method:            "default",
		OverwriteExisting: false,
	})
	if err == nil {
		t.Fatalf("expected mock data generation to be rejected when disabled")
	}
}

func TestGenerateMockDataHandlesSchemaAndConstraintErrors(t *testing.T) {
	r := &mutationResolver{}
	mock := testutil.NewPluginMock(engine.DatabaseType("Test"))
	columnsCalled := false
	mock.GetColumnsForTableFunc = func(_ *engine.PluginConfig, _, _ string) ([]engine.Column, error) {
		columnsCalled = true
		return nil, errors.New("failed to fetch columns")
	}

	ctx := context.WithValue(context.Background(), auth.AuthKey_Credentials, &engine.Credentials{Type: "Test"})
	src.MainEngine = &engine.Engine{}
	src.MainEngine.RegistryPlugin(mock.AsPlugin())

	_, err := r.GenerateMockData(ctx, model.MockDataGenerationInput{
		Schema:            "public",
		StorageUnit:       "orders",
		RowCount:          5,
		Method:            "default",
		OverwriteExisting: false,
	})
	if err == nil || !columnsCalled {
		t.Fatalf("expected error when GetColumnsForTable fails and function to be called")
	}
}

func TestGenerateMockDataSucceedsForNoSQLPlugin(t *testing.T) {
	r := &mutationResolver{}
	mock := testutil.NewPluginMock(engine.DatabaseType("Test"))
	mock.GetColumnsForTableFunc = func(_ *engine.PluginConfig, _, _ string) ([]engine.Column, error) {
		return []engine.Column{{Name: "name", Type: "text"}}, nil
	}
	mock.GetColumnConstraintsFunc = func(*engine.PluginConfig, string, string) (map[string]map[string]any, error) {
		return map[string]map[string]any{}, nil
	}

	bulkCalled := false
	mock.BulkAddRowsFunc = func(_ *engine.PluginConfig, _, _ string, rows [][]engine.Record) (bool, error) {
		bulkCalled = true
		if len(rows) != 2 {
			t.Fatalf("expected 2 rows, got %d", len(rows))
		}
		for i, row := range rows {
			if len(row) == 0 || row[0].Key != "name" {
				t.Fatalf("row %d: expected generated value for name column, got %#v", i, row)
			}
		}
		return true, nil
	}

	ctx := context.WithValue(context.Background(), auth.AuthKey_Credentials, &engine.Credentials{Type: "Test"})
	origEngine := src.MainEngine
	src.MainEngine = &engine.Engine{}
	src.MainEngine.RegistryPlugin(mock.AsPlugin())
	t.Cleanup(func() { src.MainEngine = origEngine })

	status, err := r.GenerateMockData(ctx, model.MockDataGenerationInput{
		Schema:            "public",
		StorageUnit:       "orders",
		RowCount:          2,
		Method:            "default",
		OverwriteExisting: false,
	})
	if err != nil {
		t.Fatalf("expected mock data generation to succeed, got %v", err)
	}
	if status == nil || status.AmountGenerated != 2 {
		t.Fatalf("expected two rows generated, got %#v", status)
	}
	if !bulkCalled {
		t.Fatalf("expected BulkAddRows to be called")
	}
}

func TestGenerateMockDataStopsWhenExceedingMax(t *testing.T) {
	r := &mutationResolver{}
	mock := testutil.NewPluginMock(engine.DatabaseType("Test"))
	mock.GetColumnsForTableFunc = func(_ *engine.PluginConfig, _, _ string) ([]engine.Column, error) {
		return []engine.Column{{Name: "name", Type: "text"}}, nil
	}
	mock.GetColumnConstraintsFunc = func(*engine.PluginConfig, string, string) (map[string]map[string]any, error) {
		return map[string]map[string]any{}, nil
	}
	mock.BulkAddRowsFunc = func(_ *engine.PluginConfig, _, _ string, _ [][]engine.Record) (bool, error) { return true, nil }

	ctx := context.WithValue(context.Background(), auth.AuthKey_Credentials, &engine.Credentials{Type: "Test"})
	origEngine := src.MainEngine
	src.MainEngine = &engine.Engine{}
	src.MainEngine.RegistryPlugin(mock.AsPlugin())
	t.Cleanup(func() { src.MainEngine = origEngine })

	_, err := r.GenerateMockData(ctx, model.MockDataGenerationInput{
		Schema:            "public",
		StorageUnit:       "orders",
		RowCount:          env.GetMockDataGenerationMaxRowCount() + 1,
		Method:            "default",
		OverwriteExisting: false,
	})
	if err == nil {
		t.Fatalf("expected error when requested rows exceed max limit")
	}
}

func TestGenerateMockDataFailsWhenClearTableFails(t *testing.T) {
	r := &mutationResolver{}
	mock := testutil.NewPluginMock(engine.DatabaseType("Test"))
	mock.GetColumnsForTableFunc = func(_ *engine.PluginConfig, _, _ string) ([]engine.Column, error) {
		return []engine.Column{{Name: "name", Type: "text"}}, nil
	}
	mock.GetColumnConstraintsFunc = func(*engine.PluginConfig, string, string) (map[string]map[string]any, error) {
		return map[string]map[string]any{}, nil
	}
	clearCalled := false
	mock.ClearTableDataFunc = func(*engine.PluginConfig, string, string) (bool, error) {
		clearCalled = true
		return false, errors.New("clear failed")
	}
	mock.BulkAddRowsFunc = func(_ *engine.PluginConfig, _, _ string, rows [][]engine.Record) (bool, error) {
		return true, nil
	}

	ctx := context.WithValue(context.Background(), auth.AuthKey_Credentials, &engine.Credentials{Type: "Test"})
	origEngine := src.MainEngine
	src.MainEngine = &engine.Engine{}
	src.MainEngine.RegistryPlugin(mock.AsPlugin())
	t.Cleanup(func() { src.MainEngine = origEngine })

	// Overwrite mode requires clearing the table first
	// If clear fails, the entire operation should fail to prevent duplicate data
	_, err := r.GenerateMockData(ctx, model.MockDataGenerationInput{
		Schema:            "public",
		StorageUnit:       "orders",
		RowCount:          1,
		Method:            "default",
		OverwriteExisting: true,
	})
	if err == nil {
		t.Fatalf("expected error when clear fails in overwrite mode")
	}
	if !clearCalled {
		t.Fatalf("expected ClearTableData to be called")
	}
	if !strings.Contains(err.Error(), "clear") {
		t.Fatalf("expected error message to mention clear failure, got %v", err)
	}
}
