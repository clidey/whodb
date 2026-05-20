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
	"errors"
	"strings"
	"testing"

	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/internal/testutil"
	"github.com/clidey/whodb/core/src"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/env"
	"github.com/clidey/whodb/core/src/mockdata"
	"github.com/clidey/whodb/core/src/source"
	"github.com/clidey/whodb/core/src/sourcecatalog"
)

func TestAddRowSuccess(t *testing.T) {
	resolver := &Resolver{}
	mut := resolver.Mutation()

	mock := testutil.NewPluginMock(engine.DatabaseType("Postgres"))
	mock.StorageUnitExistsFunc = func(*engine.PluginConfig, string, string) (bool, error) { return true, nil }
	addCalled := 0
	mock.AddRowFunc = func(_ *engine.PluginConfig, _, _ string, _ []engine.Record) (bool, error) {
		addCalled++
		return true, nil
	}

	setEngineMock(t, mock)
	ctx := testSourceContext("Postgres", map[string]string{"Database": "app"})

	resp, err := mut.AddSourceRow(ctx, testSourceRef(model.SourceObjectKindTable, "app", "public", "users"), []*model.RecordInput{{Key: "id", Value: "1"}})
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

	mock := testutil.NewPluginMock(engine.DatabaseType("Postgres"))
	mock.StorageUnitExistsFunc = func(*engine.PluginConfig, string, string) (bool, error) { return false, nil }
	setEngineMock(t, mock)
	ctx := testSourceContext("Postgres", map[string]string{"Database": "app"})

	if _, err := mut.AddSourceRow(ctx, testSourceRef(model.SourceObjectKindTable, "app", "public", "missing"), nil); err == nil {
		t.Fatalf("expected validation error for missing storage unit")
	}
}

func TestDeleteRowPropagatesPluginError(t *testing.T) {
	resolver := &Resolver{}
	mut := resolver.Mutation()

	mock := testutil.NewPluginMock(engine.DatabaseType("Postgres"))
	mock.StorageUnitExistsFunc = func(*engine.PluginConfig, string, string) (bool, error) { return true, nil }
	mock.DeleteRowFunc = func(*engine.PluginConfig, string, string, map[string]string) (bool, error) {
		return false, errors.New("delete failed")
	}
	setEngineMock(t, mock)
	ctx := testSourceContext("Postgres", map[string]string{"Database": "app"})

	if _, err := mut.DeleteSourceRow(ctx, testSourceRef(model.SourceObjectKindTable, "app", "public", "users"), nil); err == nil {
		t.Fatalf("expected delete error to propagate")
	}
}

func TestUpdateStorageUnitCallsPlugin(t *testing.T) {
	resolver := &Resolver{}
	mut := resolver.Mutation()

	mock := testutil.NewPluginMock(engine.DatabaseType("Postgres"))
	mock.StorageUnitExistsFunc = func(*engine.PluginConfig, string, string) (bool, error) { return true, nil }
	updateCalled := 0
	mock.UpdateStorageUnitFunc = func(*engine.PluginConfig, string, string, map[string]string, []string) (bool, error) {
		updateCalled++
		return true, nil
	}
	setEngineMock(t, mock)
	ctx := testSourceContext("Postgres", map[string]string{"Database": "app"})

	resp, err := mut.UpdateSourceObject(ctx, testSourceRef(model.SourceObjectKindTable, "app", "public", "users"), []*model.RecordInput{{Key: "name", Value: "alice"}}, []string{"name"})
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

func TestMutationsRejectUnsupportedSourceObjectActions(t *testing.T) {
	resolver := &Resolver{}
	mut := resolver.Mutation()

	mock := testutil.NewPluginMock(engine.DatabaseType("QuestDB"))
	setEngineMock(t, mock)
	ctx := testSourceContext("QuestDB", map[string]string{"Database": "app"})
	ref := testSourceRef(model.SourceObjectKindTable, "app", "users")

	if _, err := mut.UpdateSourceObject(ctx, ref, []*model.RecordInput{{Key: "name", Value: "alice"}}, []string{"name"}); err == nil || !strings.Contains(err.Error(), "updating data is not supported") {
		t.Fatalf("expected update action rejection, got %v", err)
	}
	if _, err := mut.DeleteSourceRow(ctx, ref, []*model.RecordInput{{Key: "id", Value: "1"}}); err == nil || !strings.Contains(err.Error(), "deleting data is not supported") {
		t.Fatalf("expected delete action rejection, got %v", err)
	}
	if _, err := mut.ImportSourceObjectFile(ctx, model.ImportFileInput{Ref: &ref}); err == nil || !strings.Contains(err.Error(), "importing data is not supported") {
		t.Fatalf("expected import action rejection, got %v", err)
	}
	if _, err := mut.GenerateMockData(ctx, model.MockDataGenerationInput{
		Ref:               &ref,
		RowCount:          1,
		Method:            "default",
		OverwriteExisting: false,
	}); err == nil || !strings.Contains(err.Error(), "generating mock data is not supported") {
		t.Fatalf("expected mock-data action rejection, got %v", err)
	}
	if _, err := mut.CreateSourceObject(ctx, &ref, "child", nil); err == nil || !strings.Contains(err.Error(), "creating child objects is not supported") {
		t.Fatalf("expected create-child action rejection, got %v", err)
	}
}

func TestQueryMockDataMaxRowCount(t *testing.T) {
	resolver := &Resolver{}
	query := resolver.Query()

	result, err := query.MockDataMaxRowCount(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result != mockdata.GetMockDataGenerationMaxRowCount() {
		t.Fatalf("expected mock data max row count %d, got %d", mockdata.GetMockDataGenerationMaxRowCount(), result)
	}
}

func TestQuerySourceSessionMetadataMapsFields(t *testing.T) {
	resolver := &Resolver{}
	query := resolver.Query()

	mock := testutil.NewPluginMock(engine.DatabaseType("Postgres"))
	setEngineMock(t, mock)
	setSourceSessionMetadata(t, "Postgres", source.TypeSessionMetadata{
		TypeDefinitions: []source.TypeDefinition{{
			ID:               "text",
			Label:            "Text",
			HasLength:        true,
			HasPrecision:     false,
			DefaultLength:    new(255),
			DefaultPrecision: nil,
			Category:         source.TypeCategoryText,
		}},
		Operators: []string{"=", "LIKE"},
		AliasMap:  map[string]string{"varchar": "text"},
	})

	ctx := testSourceContext("Postgres", map[string]string{"Database": "app"})
	result, err := query.SourceSessionMetadata(ctx)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result == nil || result.SourceType != "Postgres" {
		t.Fatalf("expected source session metadata to be returned, got %#v", result)
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
	if len(result.QueryLanguages) == 0 || result.QueryLanguages[0] != "sql" {
		t.Fatalf("expected sql query language metadata to be mapped, got %#v", result.QueryLanguages)
	}
}

func TestLoginFailsWhenPluginUnavailable(t *testing.T) {
	resolver := &Resolver{}
	mut := resolver.Mutation()

	mock := testutil.NewPluginMock(engine.DatabaseType("Postgres"))
	mock.IsAvailableFunc = func(context.Context, *engine.PluginConfig) bool { return false }
	setEngineMock(t, mock)

	_, err := mut.LoginSource(context.Background(), model.SourceLoginInput{
		SourceType: "Postgres",
		Values: []*model.RecordInput{
			{Key: "Hostname", Value: "h"},
			{Key: "Username", Value: "u"},
			{Key: "Password", Value: "p"},
			{Key: "Database", Value: "d"},
		},
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

	_, err := mut.LoginSource(context.Background(), model.SourceLoginInput{
		SourceType: "Postgres",
		Values: []*model.RecordInput{
			{Key: "Hostname", Value: "h"},
			{Key: "Username", Value: "u"},
			{Key: "Password", Value: "p"},
			{Key: "Database", Value: "d"},
		},
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

func setSourceSessionMetadata(t *testing.T, id string, metadata source.TypeSessionMetadata) {
	t.Helper()

	original, ok := sourcecatalog.ResolveSessionMetadata(id)
	sourcecatalog.RegisterSessionMetadata(id, metadata)
	t.Cleanup(func() {
		if ok && original != nil {
			sourcecatalog.RegisterSessionMetadata(id, *original)
		}
	})
}

//go:fix inline
func intPtr(i int) *int { return new(i) }
