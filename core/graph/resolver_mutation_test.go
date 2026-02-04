package graph

import (
	"context"
	"errors"
	"testing"

	"net/http/httptest"

	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/internal/testutil"
	"github.com/clidey/whodb/core/src"
	"github.com/clidey/whodb/core/src/auth"
	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/env"
)

func TestAddRowSuccess(t *testing.T) {
	resolver := &Resolver{}
	mut := resolver.Mutation()

	mock := testutil.NewPluginMock(engine.DatabaseType("Test"))
	mock.StorageUnitExistsFunc = func(*engine.PluginConfig, string, string) (bool, error) { return true, nil }
	addCalled := 0
	mock.AddRowFunc = func(_ *engine.PluginConfig, _, _ string, _ []engine.Record) (bool, error) {
		addCalled++
		return true, nil
	}

	setEngineMock(t, mock)
	ctx := context.WithValue(context.Background(), auth.AuthKey_Credentials, &engine.Credentials{Type: "Test"})

	resp, err := mut.AddRow(ctx, "public", "users", []*model.RecordInput{{Key: "id", Value: "1"}})
	if err != nil {
		t.Fatalf("expected add row to succeed, got %v", err)
	}
	if resp == nil || !resp.Status {
		t.Fatalf("expected status true, got %#v", resp)
	}
	if addCalled != 1 {
		t.Fatalf("expected AddRow to be invoked once, got %d", addCalled)
	}
}

func TestAddRowValidationFailure(t *testing.T) {
	resolver := &Resolver{}
	mut := resolver.Mutation()

	mock := testutil.NewPluginMock(engine.DatabaseType("Test"))
	mock.StorageUnitExistsFunc = func(*engine.PluginConfig, string, string) (bool, error) { return false, nil }
	setEngineMock(t, mock)
	ctx := context.WithValue(context.Background(), auth.AuthKey_Credentials, &engine.Credentials{Type: "Test"})

	if _, err := mut.AddRow(ctx, "public", "missing", nil); err == nil {
		t.Fatalf("expected validation error for missing storage unit")
	}
}

func TestDeleteRowPropagatesPluginError(t *testing.T) {
	resolver := &Resolver{}
	mut := resolver.Mutation()

	mock := testutil.NewPluginMock(engine.DatabaseType("Test"))
	mock.StorageUnitExistsFunc = func(*engine.PluginConfig, string, string) (bool, error) { return true, nil }
	mock.DeleteRowFunc = func(*engine.PluginConfig, string, string, map[string]string) (bool, error) {
		return false, errors.New("delete failed")
	}
	setEngineMock(t, mock)
	ctx := context.WithValue(context.Background(), auth.AuthKey_Credentials, &engine.Credentials{Type: "Test"})

	if _, err := mut.DeleteRow(ctx, "public", "users", nil); err == nil {
		t.Fatalf("expected delete error to propagate")
	}
}

func TestUpdateStorageUnitCallsPlugin(t *testing.T) {
	resolver := &Resolver{}
	mut := resolver.Mutation()

	mock := testutil.NewPluginMock(engine.DatabaseType("Test"))
	mock.StorageUnitExistsFunc = func(*engine.PluginConfig, string, string) (bool, error) { return true, nil }
	updateCalled := 0
	mock.UpdateStorageUnitFunc = func(*engine.PluginConfig, string, string, map[string]string, []string) (bool, error) {
		updateCalled++
		return true, nil
	}
	setEngineMock(t, mock)
	ctx := context.WithValue(context.Background(), auth.AuthKey_Credentials, &engine.Credentials{Type: "Test"})

	resp, err := mut.UpdateStorageUnit(ctx, "public", "users", []*model.RecordInput{{Key: "name", Value: "alice"}}, []string{"name"})
	if err != nil {
		t.Fatalf("expected update to succeed, got %v", err)
	}
	if resp == nil || !resp.Status {
		t.Fatalf("expected true status, got %#v", resp)
	}
	if updateCalled != 1 {
		t.Fatalf("expected UpdateStorageUnit to be called once, got %d", updateCalled)
	}
}

func TestQueryMockDataMaxRowCount(t *testing.T) {
	resolver := &Resolver{}
	query := resolver.Query()

	ctx := context.WithValue(context.Background(), auth.AuthKey_Credentials, &engine.Credentials{Type: "Test"})
	result, err := query.MockDataMaxRowCount(ctx)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result != env.GetMockDataGenerationMaxRowCount() {
		t.Fatalf("expected mock data max row count %d, got %d", env.GetMockDataGenerationMaxRowCount(), result)
	}
}

func TestQueryDatabaseMetadataMapsFields(t *testing.T) {
	resolver := &Resolver{}
	query := resolver.Query()

	mock := testutil.NewPluginMock(engine.DatabaseType("Test"))
	mock.GetDatabaseMetadataFunc = func() *engine.DatabaseMetadata {
		return &engine.DatabaseMetadata{
			DatabaseType: "Test",
			TypeDefinitions: []engine.TypeDefinition{{
				ID:               "text",
				Label:            "Text",
				HasLength:        true,
				HasPrecision:     false,
				DefaultLength:    intPtr(255),
				DefaultPrecision: nil,
				Category:         engine.TypeCategoryText,
			}},
			Operators: []string{"=", "LIKE"},
			AliasMap:  map[string]string{"varchar": "text"},
		}
	}
	setEngineMock(t, mock)

	ctx := context.WithValue(context.Background(), auth.AuthKey_Credentials, &engine.Credentials{Type: "Test"})
	result, err := query.DatabaseMetadata(ctx)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result == nil || result.DatabaseType != "Test" {
		t.Fatalf("expected database metadata to be returned, got %#v", result)
	}
	if len(result.TypeDefinitions) != 1 || result.TypeDefinitions[0].ID != "text" {
		t.Fatalf("expected type definitions to be mapped, got %#v", result.TypeDefinitions)
	}
	if len(result.AliasMap) != 1 {
		t.Fatalf("expected alias map to be converted, got %#v", result.AliasMap)
	}
	found := false
	for _, rec := range result.AliasMap {
		if rec.Key == "varchar" && rec.Value == "text" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected varchar alias to be present, got %#v", result.AliasMap)
	}
}

func TestLoginFailsWhenPluginUnavailable(t *testing.T) {
	resolver := &Resolver{}
	mut := resolver.Mutation()

	mock := testutil.NewPluginMock(engine.DatabaseType("Test"))
	mock.IsAvailableFunc = func(*engine.PluginConfig) bool { return false }
	setEngineMock(t, mock)

	ctx := context.WithValue(context.Background(), auth.AuthKey_Credentials, &engine.Credentials{Type: "Test"})
	reqCtx := context.WithValue(ctx, common.RouterKey_ResponseWriter, httptest.NewRecorder())

	_, err := mut.Login(reqCtx, model.LoginCredentials{
		Type:     "Test",
		Hostname: "h",
		Username: "u",
		Password: "p",
		Database: "d",
	})
	if err == nil {
		t.Fatalf("expected login to fail when plugin unavailable")
	}
}

func TestLoginFailsWhenCredentialFormDisabled(t *testing.T) {
	resolver := &Resolver{}
	mut := resolver.Mutation()

	orig := env.DisableCredentialForm
	env.DisableCredentialForm = true
	t.Cleanup(func() { env.DisableCredentialForm = orig })

	ctx := context.WithValue(context.Background(), auth.AuthKey_Credentials, &engine.Credentials{Type: "Test"})
	reqCtx := context.WithValue(ctx, common.RouterKey_ResponseWriter, httptest.NewRecorder())

	_, err := mut.Login(reqCtx, model.LoginCredentials{
		Type:     "Test",
		Hostname: "h",
		Username: "u",
		Password: "p",
		Database: "d",
	})
	if err == nil {
		t.Fatalf("expected login to fail when credential form disabled")
	}
}

func setEngineMock(t *testing.T, mock *testutil.PluginMock) {
	t.Helper()
	orig := src.MainEngine
	src.MainEngine = &engine.Engine{}
	src.MainEngine.RegistryPlugin(mock.AsPlugin())
	t.Cleanup(func() { src.MainEngine = orig })
}

func intPtr(i int) *int { return &i }
