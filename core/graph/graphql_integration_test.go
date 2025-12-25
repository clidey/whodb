package graph

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/internal/testutil"
	"github.com/clidey/whodb/core/src"
	"github.com/clidey/whodb/core/src/auth"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/env"
	"github.com/clidey/whodb/core/src/types"
)

func TestGraphQLAddRowMutation(t *testing.T) {
	mock := testutil.NewPluginMock(engine.DatabaseType("Test"))
	mock.StorageUnitExistsFunc = func(*engine.PluginConfig, string, string) (bool, error) { return true, nil }
	mock.AddRowFunc = func(*engine.PluginConfig, string, string, []engine.Record) (bool, error) { return true, nil }
	setEngineMock(t, mock)

	srv := handler.NewDefaultServer(NewExecutableSchema(Config{Resolvers: &Resolver{}}))

	query := `
	mutation AddRow($schema: String!, $table: String!) {
		AddRow(schema: $schema, storageUnit: $table, values: [{Key:"id", Value:"1"}]) {
			Status
		}
	}`
	body := map[string]any{
		"query":     query,
		"variables": map[string]any{"schema": "public", "table": "users"},
	}
	payload, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/query", bytes.NewReader(payload))
	ctx := context.WithValue(req.Context(), auth.AuthKey_Credentials, &engine.Credentials{Type: "Test"})
	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 response, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data struct {
			AddRow model.StatusResponse `json:"AddRow"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if !resp.Data.AddRow.Status {
		t.Fatalf("expected AddRow status true, got %+v", resp.Data.AddRow)
	}
}

func TestGraphQLDatabaseMetadataQueryReturnsNilWhenNotProvided(t *testing.T) {
	mock := testutil.NewPluginMock(engine.DatabaseType("Test"))
	mock.GetDatabaseMetadataFunc = func() *engine.DatabaseMetadata { return nil }
	setEngineMock(t, mock)

	srv := handler.NewDefaultServer(NewExecutableSchema(Config{Resolvers: &Resolver{}}))

	query := `query { DatabaseMetadata { databaseType } }`
	body := map[string]any{"query": query}
	payload, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/query", bytes.NewReader(payload))
	ctx := context.WithValue(req.Context(), auth.AuthKey_Credentials, &engine.Credentials{Type: "Test"})
	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 response, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data struct {
			DatabaseMetadata *model.DatabaseMetadata `json:"DatabaseMetadata"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.Data.DatabaseMetadata != nil {
		t.Fatalf("expected nil metadata when plugin returns nil, got %+v", resp.Data.DatabaseMetadata)
	}
}

func TestGraphQLRowQueryWithSortAndWhere(t *testing.T) {
	mock := testutil.NewPluginMock(engine.DatabaseType("Test"))
	mock.StorageUnitExistsFunc = func(*engine.PluginConfig, string, string) (bool, error) { return true, nil }
	mock.GetRowsFunc = func(_ *engine.PluginConfig, _, _ string, where *model.WhereCondition, sort []*model.SortCondition, _, _ int) (*engine.GetRowsResult, error) {
		if where == nil || where.Atomic == nil || where.Atomic.Key != "id" || where.Atomic.Operator != "=" || where.Atomic.Value != "1" {
			t.Fatalf("unexpected where clause passed to plugin: %#v", where)
		}
		if len(sort) != 1 || sort[0].Column != "id" || sort[0].Direction != model.SortDirectionAsc {
			t.Fatalf("unexpected sort: %#v", sort)
		}
		return &engine.GetRowsResult{
			Columns: []engine.Column{{Name: "id", Type: "int"}},
			Rows:    [][]string{{"1"}},
		}, nil
	}
	setEngineMock(t, mock)

	srv := handler.NewDefaultServer(NewExecutableSchema(Config{Resolvers: &Resolver{}}))

	query := `
	query Row($schema:String!, $table:String!){
		Row(schema:$schema, storageUnit:$table, where:{Type:Atomic, Atomic:{Key:"id", Operator:"=", Value:"1", ColumnType:"int"}}, sort:[{Column:"id", Direction:ASC}], pageSize:10, pageOffset:0){
			Rows
		}
	}`
	body := map[string]any{
		"query":     query,
		"variables": map[string]any{"schema": "public", "table": "users"},
	}
	payload, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/query", bytes.NewReader(payload))
	ctx := context.WithValue(req.Context(), auth.AuthKey_Credentials, &engine.Credentials{Type: "Test"})
	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 response, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data struct {
			Row *model.RowsResult `json:"Row"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.Data.Row == nil || len(resp.Data.Row.Rows) != 1 {
		t.Fatalf("expected one row, got %#v body=%s", resp.Data.Row, w.Body.String())
	}
}

func TestGraphQLProfilesQueryUsesEngineProfiles(t *testing.T) {
	origEngine := src.MainEngine
	src.MainEngine = &engine.Engine{}
	src.MainEngine.AddLoginProfile(types.DatabaseCredentials{
		Alias:    "alias",
		Hostname: "db.local",
		Username: "alice",
		Database: "app",
		Type:     "Test",
	})
	t.Cleanup(func() { src.MainEngine = origEngine })

	srv := handler.NewDefaultServer(NewExecutableSchema(Config{Resolvers: &Resolver{}}))
	query := `query { Profiles { Id Alias Type Database IsEnvironmentDefined } }`
	body, _ := json.Marshal(map[string]any{"query": query})

	req := httptest.NewRequest(http.MethodPost, "/api/query", bytes.NewReader(body))
	req = req.WithContext(context.WithValue(req.Context(), auth.AuthKey_Credentials, &engine.Credentials{Type: "Test"}))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 response, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data struct {
			Profiles []struct {
				ID                   string  `json:"Id"`
				Alias                *string `json:"Alias"`
				Type                 string  `json:"Type"`
				Database             *string `json:"Database"`
				IsEnvironmentDefined bool    `json:"IsEnvironmentDefined"`
			} `json:"Profiles"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if len(resp.Data.Profiles) != 1 || resp.Data.Profiles[0].ID == "" || resp.Data.Profiles[0].Database == nil {
		t.Fatalf("expected profile to be returned, got %#v", resp.Data.Profiles)
	}
}

func TestGraphQLVersionUsesEnvFallback(t *testing.T) {
	origVersion := env.ApplicationVersion
	env.ApplicationVersion = ""
	t.Cleanup(func() { env.ApplicationVersion = origVersion })

	srv := handler.NewDefaultServer(NewExecutableSchema(Config{Resolvers: &Resolver{}}))
	body, _ := json.Marshal(map[string]any{"query": `query { Version }`})
	req := httptest.NewRequest(http.MethodPost, "/api/query", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 response, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Data struct {
			Version string `json:"Version"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.Data.Version != "development" {
		t.Fatalf("expected development fallback version, got %s", resp.Data.Version)
	}
}
