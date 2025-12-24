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
