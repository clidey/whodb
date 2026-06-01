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

	"github.com/99designs/gqlgen/graphql"
	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/internal/testutil"
	"github.com/clidey/whodb/core/src"
	"github.com/clidey/whodb/core/src/common/ssl"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/env"
	"github.com/clidey/whodb/core/src/sourcecatalog"
	"github.com/clidey/whodb/core/src/types"
)

func TestQueryProfilesMapsAliasCustomIDAndSSL(t *testing.T) {
	originalEngine := src.MainEngine
	src.MainEngine = &engine.Engine{}
	t.Cleanup(func() {
		src.MainEngine = originalEngine
	})

	src.MainEngine.AddLoginProfile(types.DatabaseCredentials{
		Type:     "Postgres",
		Alias:    "Reporting",
		Hostname: "db.internal",
		Database: "analytics",
		Source:   "environment",
		Advanced: map[string]string{
			ssl.KeySSLMode: string(ssl.SSLModeRequired),
		},
	})
	src.MainEngine.AddLoginProfile(types.DatabaseCredentials{
		Type:     "MySQL",
		CustomId: "mysql-profile",
		Hostname: "mysql.internal",
		Database: "app",
		Source:   "env",
	})

	profiles, err := (&Resolver{}).Query().SourceProfiles(context.Background())
	if err != nil {
		t.Fatalf("expected profiles query to succeed, got %v", err)
	}

	var reportingProfile *model.SourceProfile
	var mysqlProfile *model.SourceProfile
	for _, profile := range profiles {
		if profile.DisplayName == "Reporting" {
			reportingProfile = profile
		}
		if profile.ID == "mysql-profile" {
			mysqlProfile = profile
		}
	}

	if reportingProfile == nil {
		t.Fatalf("expected reporting profile to be present, got %#v", profiles)
	}
	if !reportingProfile.SSLConfigured {
		t.Fatal("expected SSLConfigured to be true when SSL mode is enabled")
	}
	if mysqlProfile == nil {
		t.Fatalf("expected custom ID profile to be present, got %#v", profiles)
	}
}

func TestQueryDatabaseUsesMinimalConfigWithoutSessionCredentials(t *testing.T) {
	mock := testutil.NewPluginMock(engine.DatabaseType("Postgres"))
	mock.GetDatabasesFunc = func(config *engine.PluginConfig) ([]string, error) {
		if config == nil || config.Credentials == nil {
			t.Fatal("expected credentials to be present")
		}
		if config.Credentials.Type != "Postgres" {
			t.Fatalf("expected type Postgres, got %q", config.Credentials.Type)
		}
		if config.Credentials.Hostname != "" {
			t.Fatalf("expected no session credentials to be used, got hostname %q", config.Credentials.Hostname)
		}
		return []string{"db_a", "db_b"}, nil
	}
	setEngineMock(t, mock)

	databases, err := (&Resolver{}).Query().SourceFieldOptions(context.Background(), "Postgres", "Database", nil)
	if err != nil {
		t.Fatalf("expected database query to succeed, got %v", err)
	}
	if len(databases) != 2 || databases[0] != "db_a" || databases[1] != "db_b" {
		t.Fatalf("unexpected databases result: %#v", databases)
	}
}

func TestQueryRowValidatesPaginationAndEnrichesColumns(t *testing.T) {
	t.Run("rejects invalid page size before hitting plugins", func(t *testing.T) {
		originalMaxPageSize := env.MaxPageSize
		env.MaxPageSize = 10
		t.Cleanup(func() {
			env.MaxPageSize = originalMaxPageSize
		})

		_, err := (&Resolver{}).Query().SourceRows(context.Background(), testSourceRef(model.SourceObjectKindTable, "app", "public", "orders"), nil, nil, 11, 0)
		if err == nil || !strings.Contains(err.Error(), "pageSize must not exceed 10") {
			t.Fatalf("expected max page size validation error, got %v", err)
		}
	})

	t.Run("maps row and column metadata", func(t *testing.T) {
		mock := testutil.NewPluginMock(engine.DatabaseType("Postgres"))
		mock.StorageUnitExistsFunc = func(*engine.PluginConfig, string, string) (bool, error) { return true, nil }
		mock.GetRowsFunc = func(*engine.PluginConfig, *engine.GetRowsRequest) (*engine.GetRowsResult, error) {
			return &engine.GetRowsResult{
				Columns: []engine.Column{
					{Name: "id", Type: "INTEGER"},
					{Name: "user_id", Type: "INTEGER"},
				},
				Rows:       [][]string{{"1", "42"}},
				TotalCount: 17,
			}, nil
		}
		refTable := "users"
		refColumn := "id"
		mock.GetColumnsForTableFunc = func(*engine.PluginConfig, string, string) ([]engine.Column, error) {
			return []engine.Column{
				{Name: "id", Type: "INTEGER", IsPrimary: true},
				{Name: "user_id", Type: "INTEGER", IsForeignKey: true, ReferencedTable: &refTable, ReferencedColumn: &refColumn},
			}, nil
		}
		setEngineMock(t, mock)

		ctx := testSourceContext("Postgres", map[string]string{"Database": "app"})
		result, err := (&Resolver{}).Query().SourceRows(ctx, testSourceRef(model.SourceObjectKindTable, "app", "public", "orders"), nil, nil, 5, 0)
		if err != nil {
			t.Fatalf("expected row query to succeed, got %v", err)
		}
		if result.TotalCount != 17 || len(result.Rows) != 1 {
			t.Fatalf("unexpected row result: %#v", result)
		}
		if len(result.Columns) != 2 || !result.Columns[0].IsPrimary {
			t.Fatalf("expected primary key metadata to be attached, got %#v", result.Columns)
		}
		if !result.Columns[1].IsForeignKey || result.Columns[1].ReferencedTable == nil || *result.Columns[1].ReferencedTable != "users" {
			t.Fatalf("expected foreign key metadata to be attached, got %#v", result.Columns[1])
		}
	})
}

func TestSourceQueriesRejectUnsupportedSourceObjectActions(t *testing.T) {
	mock := testutil.NewPluginMock(engine.DatabaseType("Postgres"))
	setEngineMock(t, mock)
	ctx := testSourceContext("Postgres", map[string]string{"Database": "app"})
	schemaRef := testSourceRef(model.SourceObjectKindSchema, "app", "public")
	tableRef := testSourceRef(model.SourceObjectKindTable, "app", "public", "orders")

	if _, err := (&Resolver{}).Query().SourceRows(ctx, schemaRef, nil, nil, 5, 0); err == nil || !strings.Contains(err.Error(), "viewing rows is not supported") {
		t.Fatalf("expected row-view action rejection, got %v", err)
	}
	if _, err := (&Resolver{}).Query().SourceContent(ctx, tableRef); err == nil || !strings.Contains(err.Error(), "viewing content is not supported") {
		t.Fatalf("expected content-view action rejection, got %v", err)
	}
	if _, err := (&Resolver{}).Query().SourceColumns(ctx, schemaRef); err == nil || !strings.Contains(err.Error(), "inspecting objects is not supported") {
		t.Fatalf("expected inspect action rejection, got %v", err)
	}
}

func TestQueryColumnsBatchSkipsFailedTables(t *testing.T) {
	mock := testutil.NewPluginMock(engine.DatabaseType("Postgres"))
	mock.StorageUnitExistsFunc = func(*engine.PluginConfig, string, string) (bool, error) { return true, nil }
	mock.GetColumnsForTableFunc = func(_ *engine.PluginConfig, _ string, storageUnit string) ([]engine.Column, error) {
		if storageUnit == "broken" {
			return nil, errors.New("boom")
		}
		return []engine.Column{{Name: "id", Type: "INTEGER"}}, nil
	}
	setEngineMock(t, mock)

	ctx := testSourceContext("Postgres", map[string]string{"Database": "app"})
	result, err := (&Resolver{}).Query().SourceColumnsBatch(ctx, []*model.SourceObjectRefInput{
		testSourceRefPtr(model.SourceObjectKindTable, "app", "public", "users"),
		testSourceRefPtr(model.SourceObjectKindTable, "app", "public", "broken"),
	})
	if err != nil {
		t.Fatalf("expected columns batch to succeed, got %v", err)
	}
	if len(result) != 1 || result[0].Ref == nil || result[0].Ref.Path[len(result[0].Ref.Path)-1] != "users" {
		t.Fatalf("expected only successful tables to be returned, got %#v", result)
	}
}

func TestQuerySourceTypesMapsCatalogEntries(t *testing.T) {
	result, err := (&Resolver{}).Query().SourceTypes(context.Background())
	if err != nil {
		t.Fatalf("expected source types query to succeed, got %v", err)
	}
	if len(result) == 0 {
		t.Fatal("expected source types to be returned")
	}

	var postgres *model.SourceType
	for _, entry := range result {
		if entry.ID == "Postgres" {
			postgres = entry
			break
		}
	}
	if postgres == nil {
		t.Fatalf("expected Postgres source type to be present")
	}

	fieldByKey := map[string]*model.SourceConnectionField{}
	for _, field := range postgres.ConnectionFields {
		fieldByKey[field.Key] = field
	}
	if fieldByKey["Hostname"] == nil || !fieldByKey["Hostname"].Required {
		t.Fatalf("expected Hostname connection field to be mapped, got %#v", postgres.ConnectionFields)
	}
	if fieldByKey["Database"] == nil || !fieldByKey["Database"].Required {
		t.Fatalf("expected Database connection field to be required, got %#v", postgres.ConnectionFields)
	}
	if len(postgres.SSLModes) == 0 {
		t.Fatal("expected postgres SSL modes to be exposed")
	}
	if postgres.Contract == nil || postgres.Contract.DefaultObjectKind != model.SourceObjectKindTable {
		t.Fatalf("expected Postgres source contract to be mapped, got %#v", postgres.Contract)
	}

	portField := fieldByKey["Port"]
	if portField == nil || portField.DefaultValue == nil || *portField.DefaultValue != "5432" {
		t.Fatalf("expected default port field to be included, got %#v", postgres.ConnectionFields)
	}
}

func TestQuerySourceQuerySuggestionsCapsAtThree(t *testing.T) {
	mock := testutil.NewPluginMock(engine.DatabaseType("Postgres"))
	mock.GetStorageUnitsFunc = func(*engine.PluginConfig, string) ([]engine.StorageUnit, error) {
		return []engine.StorageUnit{
			{Name: "users"},
			{Name: "orders"},
			{Name: "payments"},
			{Name: "ignored"},
		}, nil
	}
	setEngineMock(t, mock)

	ctx := testSourceContext("Postgres", map[string]string{"Database": "app"})
	suggestions, err := (&Resolver{}).Query().SourceQuerySuggestions(ctx, testSourceRefPtr(model.SourceObjectKindSchema, "app", "public"))
	if err != nil {
		t.Fatalf("expected suggestions query to succeed, got %v", err)
	}
	if len(suggestions) != 3 {
		t.Fatalf("expected suggestions to be capped at 3, got %#v", suggestions)
	}
	if !strings.Contains(suggestions[0].Description, "users") || suggestions[1].Category != "AGGREGATE" {
		t.Fatalf("expected deterministic suggestion text/categories, got %#v", suggestions)
	}
	for _, suggestion := range suggestions {
		if strings.Contains(suggestion.Description, "ignored") {
			t.Fatalf("did not expect truncated table to appear in suggestions, got %#v", suggestions)
		}
	}
}

func TestQueryHealthReportsDatabaseStatus(t *testing.T) {
	t.Run("healthy plugin reports healthy database", func(t *testing.T) {
		mock := testutil.NewPluginMock(engine.DatabaseType("Postgres"))
		mock.IsAvailableFunc = func(context.Context, *engine.PluginConfig) bool { return true }
		setEngineMock(t, mock)

		ctx := testSourceContext("Postgres", nil)
		status, err := (&Resolver{}).Query().Health(ctx)
		if err != nil {
			t.Fatalf("expected health query to succeed, got %v", err)
		}
		if status.Server != "healthy" || status.Database != "healthy" {
			t.Fatalf("expected healthy server/database, got %#v", status)
		}
	})

	t.Run("failed availability reports database error", func(t *testing.T) {
		mock := testutil.NewPluginMock(engine.DatabaseType("Postgres"))
		mock.IsAvailableFunc = func(context.Context, *engine.PluginConfig) bool { return false }
		setEngineMock(t, mock)

		ctx := testSourceContext("Postgres", nil)
		status, err := (&Resolver{}).Query().Health(ctx)
		if err != nil {
			t.Fatalf("expected health query to succeed, got %v", err)
		}
		if status.Database != "error" {
			t.Fatalf("expected database error status, got %#v", status)
		}
	})
}

func TestMutationExecuteConfirmedSQLMapsResultsAndErrors(t *testing.T) {
	t.Run("successful execution returns query result", func(t *testing.T) {
		mock := testutil.NewPluginMock(engine.DatabaseType("Postgres"))
		mock.RawExecuteFunc = func(*engine.PluginConfig, string, ...any) (*engine.GetRowsResult, error) {
			return &engine.GetRowsResult{
				Columns: []engine.Column{{Name: "id", Type: "INTEGER"}},
				Rows:    [][]string{{"1"}},
			}, nil
		}
		setEngineMock(t, mock)

		ctx := testSourceContext("Postgres", nil)
		message, err := (&Resolver{}).Mutation().ExecuteConfirmedSQL(ctx, "SELECT 1", "sql:get")
		if err != nil {
			t.Fatalf("expected confirmed SQL execution to succeed, got %v", err)
		}
		if message.Type != "sql:get" || message.Result == nil || len(message.Result.Rows) != 1 {
			t.Fatalf("unexpected confirmed SQL message: %#v", message)
		}
	})

	t.Run("execution errors become error messages", func(t *testing.T) {
		mock := testutil.NewPluginMock(engine.DatabaseType("Postgres"))
		mock.RawExecuteFunc = func(*engine.PluginConfig, string, ...any) (*engine.GetRowsResult, error) {
			return nil, errors.New("query failed")
		}
		setEngineMock(t, mock)

		ctx := testSourceContext("Postgres", nil)
		message, err := (&Resolver{}).Mutation().ExecuteConfirmedSQL(ctx, "DELETE FROM orders", "sql:delete")
		if err != nil {
			t.Fatalf("expected confirmed SQL execution to return a mapped message, got %v", err)
		}
		if message.Type != "error" || message.Text != "query failed" {
			t.Fatalf("expected error message, got %#v", message)
		}
	})
}

func TestMutationExecuteConfirmedSQLRejectsUnsupportedSourceScripts(t *testing.T) {
	mock := testutil.NewPluginMock(engine.DatabaseType("Redis"))
	setEngineMock(t, mock)

	ctx := testSourceContext("Redis", nil)
	if _, err := (&Resolver{}).Mutation().ExecuteConfirmedSQL(ctx, "DEL key", "sql:delete"); err == nil || !strings.Contains(err.Error(), "querying is not supported") {
		t.Fatalf("expected confirmed SQL to reject unsupported source scripts, got %v", err)
	}
}

func TestMutationImportSQLValidatesSourcesAndExecutesScripts(t *testing.T) {
	mutation := (&Resolver{}).Mutation()
	ctx := testSourceContext("Postgres", nil)

	t.Run("rejects missing or conflicting SQL sources", func(t *testing.T) {
		setEngineMock(t, testutil.NewPluginMock(engine.DatabaseType("Postgres")))

		result, err := mutation.ImportSQL(ctx, model.ImportSQLInput{})
		if err != nil {
			t.Fatalf("expected validation error to be returned as result, got %v", err)
		}
		if result.Status || result.Detail == nil || *result.Detail != importErrorSQLSourceMissing {
			t.Fatalf("expected missing source validation result, got %#v", result)
		}

		script := "SELECT 1"
		upload := graphql.Upload{Filename: "query.sql"}
		result, err = mutation.ImportSQL(ctx, model.ImportSQLInput{
			Script: &script,
			File:   &upload,
		})
		if err != nil {
			t.Fatalf("expected conflicting source validation result, got %v", err)
		}
		if result.Status || result.Detail == nil || *result.Detail != importErrorSQLSourceBoth {
			t.Fatalf("expected conflicting source validation result, got %#v", result)
		}
	})

	t.Run("executes script with multistatement enabled", func(t *testing.T) {
		mock := testutil.NewPluginMock(engine.DatabaseType("Postgres"))
		mock.RawExecuteFunc = func(config *engine.PluginConfig, query string, _ ...any) (*engine.GetRowsResult, error) {
			if !config.MultiStatement {
				t.Fatal("expected SQL import to enable multistatement mode")
			}
			if query != "CREATE TABLE demo(id INT);" {
				t.Fatalf("unexpected SQL script: %q", query)
			}
			return &engine.GetRowsResult{}, nil
		}
		setEngineMock(t, mock)

		script := "CREATE TABLE demo(id INT);"
		result, err := mutation.ImportSQL(ctx, model.ImportSQLInput{Script: &script})
		if err != nil {
			t.Fatalf("expected SQL import to succeed, got %v", err)
		}
		if !result.Status || result.Detail != nil {
			t.Fatalf("expected successful import result, got %#v", result)
		}
	})

	t.Run("maps unsupported multistatement errors to validation keys", func(t *testing.T) {
		mock := testutil.NewPluginMock(engine.DatabaseType("Postgres"))
		mock.RawExecuteFunc = func(*engine.PluginConfig, string, ...any) (*engine.GetRowsResult, error) {
			return nil, engine.ErrMultiStatementUnsupported
		}
		setEngineMock(t, mock)

		script := "DROP TABLE demo;"
		result, err := mutation.ImportSQL(ctx, model.ImportSQLInput{Script: &script})
		if err != nil {
			t.Fatalf("expected unsupported error to be returned as result, got %v", err)
		}
		if result.Status || result.Detail == nil || *result.Detail != importErrorSQLMultiStatementUnsupported {
			t.Fatalf("expected unsupported multistatement result, got %#v", result)
		}
	})

	t.Run("rejects sources without script execution", func(t *testing.T) {
		mock := testutil.NewPluginMock(engine.DatabaseType("Redis"))
		setEngineMock(t, mock)

		script := "DEL key"
		result, err := mutation.ImportSQL(testSourceContext("Redis", nil), model.ImportSQLInput{Script: &script})
		if err == nil || result != nil || !strings.Contains(err.Error(), "querying is not supported") {
			t.Fatalf("expected SQL import to reject unsupported source scripts, result=%#v err=%v", result, err)
		}
	})
}

func TestCatalogHasStableDefaultPortForPostgres(t *testing.T) {
	spec, ok := sourcecatalog.Find(string(engine.DatabaseType_Postgres))
	if !ok {
		t.Fatal("expected Postgres source type to be registered")
	}
	portField, ok := spec.ConnectionFieldByKey("Port")
	if !ok || portField.DefaultValue != "5432" {
		t.Fatalf("expected postgres default port 5432, got %#v (ok=%t)", portField, ok)
	}
}

// TestSourceTypesContract validates that every source type returned by the
// GraphQL resolver is self-consistent and free of model-level defects that
// would cause frontend/backend disagreement.
func TestSourceTypesContractIsSelfConsistent(t *testing.T) {
	t.Parallel()

	types, err := (&Resolver{}).Query().SourceTypes(context.Background())
	if err != nil {
		t.Fatalf("expected source types query to succeed, got %v", err)
	}
	if len(types) == 0 {
		t.Fatal("expected at least one source type")
	}

	seenIDs := map[string]struct{}{}
	for _, st := range types {
		t.Run(st.ID, func(t *testing.T) {
			t.Parallel()
			validateSourceTypeContract(t, st)
		})

		// Check for duplicate IDs across the catalog
		if _, dup := seenIDs[st.ID]; dup {
			t.Errorf("duplicate source type ID %q", st.ID)
		}
		seenIDs[st.ID] = struct{}{}
	}
}

func validateSourceTypeContract(t *testing.T, st *model.SourceType) {
	t.Helper()

	// Required identity fields
	if st.ID == "" {
		t.Error("Id must not be empty")
	}
	if st.Label == "" {
		t.Error("Label must not be empty")
	}
	if st.Connector == "" {
		t.Error("Connector must not be empty")
	}

	// Traits must exist with all sub-groups
	if st.Traits == nil {
		t.Fatal("Traits must not be nil")
	}
	if st.Traits.Connection == nil {
		t.Error("Traits.Connection must not be nil")
	}
	if st.Traits.Presentation == nil {
		t.Error("Traits.Presentation must not be nil")
	}
	if st.Traits.Query == nil {
		t.Error("Traits.Query must not be nil")
	}
	if st.Traits.MockData == nil {
		t.Error("Traits.MockData must not be nil")
	}
	if st.Traits.Metadata == nil {
		t.Error("Traits.Metadata must not be nil")
	}

	// Contract must exist
	if st.Contract == nil {
		t.Fatal("Contract must not be nil")
	}
	contract := st.Contract

	// Model must be a valid enum (non-empty string)
	if contract.Model == "" {
		t.Error("Contract.Model must not be empty")
	}

	// DefaultObjectKind must be declared in ObjectTypes
	defaultKindOK := false
	for _, ot := range contract.ObjectTypes {
		if ot != nil && ot.Kind == contract.DefaultObjectKind {
			defaultKindOK = true
			break
		}
	}
	if !defaultKindOK {
		t.Errorf("Contract.DefaultObjectKind %q is not declared in ObjectTypes", contract.DefaultObjectKind)
	}

	// BrowsePath kinds must all be declared in ObjectTypes
	for _, browseKind := range contract.BrowsePath {
		found := false
		for _, ot := range contract.ObjectTypes {
			if ot != nil && ot.Kind == browseKind {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Contract.BrowsePath kind %q is not declared in ObjectTypes", browseKind)
		}
	}

	// GraphScopeKind must be in ObjectTypes (when set)
	if contract.GraphScopeKind != nil {
		found := false
		for _, ot := range contract.ObjectTypes {
			if ot != nil && ot.Kind == *contract.GraphScopeKind {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Contract.GraphScopeKind %q is not declared in ObjectTypes", *contract.GraphScopeKind)
		}
	}

	// Every object type must have a non-empty Kind, SingularLabel, PluralLabel
	for i, ot := range contract.ObjectTypes {
		if ot == nil {
			t.Errorf("Contract.ObjectTypes[%d] is nil", i)
			continue
		}
		if ot.Kind == "" {
			t.Errorf("Contract.ObjectTypes[%d].Kind must not be empty", i)
		}
		if ot.SingularLabel == "" {
			t.Errorf("Contract.ObjectTypes[%d].SingularLabel must not be empty", i)
		}
		if ot.PluralLabel == "" {
			t.Errorf("Contract.ObjectTypes[%d].PluralLabel must not be empty", i)
		}
	}

	// ConnectionFields: every field must have a non-empty Key
	for i, field := range st.ConnectionFields {
		if field == nil {
			t.Errorf("ConnectionFields[%d] is nil", i)
			continue
		}
		if field.Key == "" {
			t.Errorf("ConnectionFields[%d].Key must not be empty", i)
		}
	}

	// SSLModes: each entry must have a non-empty value
	for i, mode := range st.SSLModes {
		if mode == nil {
			t.Errorf("SSLModes[%d] is nil", i)
			continue
		}
		if mode.Value == "" {
			t.Errorf("SSLModes[%d].Value must not be empty", i)
		}
	}
}
