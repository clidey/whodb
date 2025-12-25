package graph

import (
	"context"
	"errors"
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
	rowsCalled := false
	mock.GetRowsFunc = func(_ *engine.PluginConfig, _, _ string, _ *model.WhereCondition, _ []*model.SortCondition, _ int, _ int) (*engine.GetRowsResult, error) {
		rowsCalled = true
		return nil, errors.New("failed to fetch schema")
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
	if err == nil || !rowsCalled {
		t.Fatalf("expected error when GetRows fails and function to be called")
	}
}

func TestGenerateMockDataSucceedsForNoSQLPlugin(t *testing.T) {
	r := &mutationResolver{}
	mock := testutil.NewPluginMock(engine.DatabaseType("Test"))
	mock.GetRowsFunc = func(_ *engine.PluginConfig, _, _ string, _ *model.WhereCondition, _ []*model.SortCondition, _ int, _ int) (*engine.GetRowsResult, error) {
		return &engine.GetRowsResult{
			Columns: []engine.Column{{Name: "name", Type: "text"}},
			Rows:    [][]string{},
		}, nil
	}
	mock.GetColumnConstraintsFunc = func(*engine.PluginConfig, string, string) (map[string]map[string]any, error) {
		return map[string]map[string]any{}, nil
	}

	callCount := 0
	mock.AddRowFunc = func(_ *engine.PluginConfig, _, _ string, values []engine.Record) (bool, error) {
		callCount++
		if len(values) == 0 || values[0].Key != "name" {
			t.Fatalf("expected generated value for name column, got %#v", values)
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
	if callCount < 2 {
		t.Fatalf("expected AddRow to be called for each requested row, got %d", callCount)
	}
}

func TestGenerateMockDataStopsWhenExceedingMax(t *testing.T) {
	r := &mutationResolver{}
	mock := testutil.NewPluginMock(engine.DatabaseType("Test"))
	mock.GetRowsFunc = func(_ *engine.PluginConfig, _, _ string, _ *model.WhereCondition, _ []*model.SortCondition, _ int, _ int) (*engine.GetRowsResult, error) {
		return &engine.GetRowsResult{
			Columns: []engine.Column{{Name: "name", Type: "text"}},
			Rows:    [][]string{},
		}, nil
	}
	mock.GetColumnConstraintsFunc = func(*engine.PluginConfig, string, string) (map[string]map[string]any, error) {
		return map[string]map[string]any{}, nil
	}
	mock.AddRowFunc = func(_ *engine.PluginConfig, _, _ string, _ []engine.Record) (bool, error) { return true, nil }

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

func TestGenerateMockDataErrorsWhenClearTableFails(t *testing.T) {
	r := &mutationResolver{}
	mock := testutil.NewPluginMock(engine.DatabaseType("Test"))
	mock.GetRowsFunc = func(_ *engine.PluginConfig, _, _ string, _ *model.WhereCondition, _ []*model.SortCondition, _ int, _ int) (*engine.GetRowsResult, error) {
		return &engine.GetRowsResult{
			Columns: []engine.Column{{Name: "name", Type: "text"}},
			Rows:    [][]string{},
		}, nil
	}
	mock.GetColumnConstraintsFunc = func(*engine.PluginConfig, string, string) (map[string]map[string]any, error) {
		return map[string]map[string]any{}, nil
	}
	mock.ClearTableDataFunc = func(*engine.PluginConfig, string, string) (bool, error) {
		return false, errors.New("clear failed")
	}

	ctx := context.WithValue(context.Background(), auth.AuthKey_Credentials, &engine.Credentials{Type: "Test"})
	origEngine := src.MainEngine
	src.MainEngine = &engine.Engine{}
	src.MainEngine.RegistryPlugin(mock.AsPlugin())
	t.Cleanup(func() { src.MainEngine = origEngine })

	_, err := r.GenerateMockData(ctx, model.MockDataGenerationInput{
		Schema:            "public",
		StorageUnit:       "orders",
		RowCount:          1,
		Method:            "default",
		OverwriteExisting: true,
	})
	if err == nil {
		t.Fatalf("expected error when clearing table fails")
	}
}
