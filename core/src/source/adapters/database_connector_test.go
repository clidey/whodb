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

package adapters

import (
	"context"
	"strings"
	"testing"

	"github.com/clidey/whodb/core/internal/testutil"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/source"
)

func TestDatabaseSessionRunQueryRejectsUnsupportedSurface(t *testing.T) {
	mock := testutil.NewPluginMock(engine.DatabaseType("MongoDB"))
	mock.RawExecuteFunc = func(*engine.PluginConfig, string, ...any) (*engine.GetRowsResult, error) {
		t.Fatalf("expected query execution to be blocked by the source contract")
		return nil, nil
	}

	session := newTestDatabaseSession(testTypeSpec("MongoDB", []source.Surface{source.SurfaceBrowser}), mock)

	_, err := session.RunQuery(context.Background(), "SELECT 1")
	if err == nil || !strings.Contains(err.Error(), "querying") {
		t.Fatalf("expected querying error, got %v", err)
	}
}

func TestDatabaseSessionReadGraphRejectsUnsupportedSurface(t *testing.T) {
	mock := testutil.NewPluginMock(engine.DatabaseType("MySQL"))
	mock.GetGraphFunc = func(*engine.PluginConfig, string) ([]engine.GraphUnit, error) {
		t.Fatalf("expected graph reads to be blocked by the source contract")
		return nil, nil
	}

	session := newTestDatabaseSession(testTypeSpec("MySQL", []source.Surface{source.SurfaceBrowser, source.SurfaceQuery}), mock)

	_, err := session.ReadGraph(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "graph") {
		t.Fatalf("expected graph error, got %v", err)
	}
}

func TestDatabaseSessionListObjectsFiltersInternalObjects(t *testing.T) {
	mock := testutil.NewPluginMock(engine.DatabaseType("Postgres"))
	mock.GetAllSchemasFunc = func(*engine.PluginConfig) ([]string, error) {
		return []string{"public", "pg_catalog", "app"}, nil
	}

	spec := testTypeSpec("Postgres", []source.Surface{source.SurfaceBrowser})
	spec.Traits.Metadata.HiddenObjectNames = map[source.ObjectKind][]string{
		source.ObjectKindSchema: {"pg_catalog"},
	}
	session := newTestDatabaseSession(testTypeWithObjectActions(
		"Postgres",
		[]source.Surface{source.SurfaceBrowser},
		[]source.Action{source.ActionBrowse},
		map[source.ObjectKind][]source.Action{
			source.ObjectKindDatabase: {source.ActionBrowse},
			source.ObjectKindSchema:   {source.ActionBrowse},
			source.ObjectKindTable:    {source.ActionInspect, source.ActionViewRows},
		},
	), mock)
	session.spec.Traits.Metadata = spec.Traits.Metadata

	objects, err := session.ListObjects(context.Background(), &source.ObjectRef{
		Kind: source.ObjectKindDatabase,
		Path: []string{"app"},
	}, nil)
	if err != nil {
		t.Fatalf("expected schema listing to succeed, got %v", err)
	}
	if len(objects) != 2 {
		t.Fatalf("expected two visible schemas, got %#v", objects)
	}
	if objects[0].Name != "public" || objects[1].Name != "app" {
		t.Fatalf("expected internal schema to be filtered, got %#v", objects)
	}
}

func TestDatabaseSessionListObjectsRejectsUnsupportedBrowseAction(t *testing.T) {
	mock := testutil.NewPluginMock(engine.DatabaseType("Postgres"))
	mock.GetStorageUnitsFunc = func(*engine.PluginConfig, string) ([]engine.StorageUnit, error) {
		t.Fatalf("expected browsing to be blocked before storage-unit lookup")
		return nil, nil
	}

	session := newTestDatabaseSession(testTypeWithObjectActions(
		"Postgres",
		[]source.Surface{source.SurfaceBrowser},
		[]source.Action{source.ActionBrowse},
		map[source.ObjectKind][]source.Action{
			source.ObjectKindSchema: {},
			source.ObjectKindTable:  {source.ActionInspect, source.ActionViewRows},
		},
	), mock)

	_, err := session.ListObjects(context.Background(), testSchemaRef(), nil)
	if err == nil || !strings.Contains(err.Error(), "browsing") {
		t.Fatalf("expected browsing error, got %v", err)
	}
}

func TestDatabaseSessionReadGraphAppliesMetadataContract(t *testing.T) {
	mock := testutil.NewPluginMock(engine.DatabaseType("ClickHouse"))
	mock.GetGraphFunc = func(*engine.PluginConfig, string) ([]engine.GraphUnit, error) {
		return []engine.GraphUnit{
			{
				Unit: engine.StorageUnit{Name: "events"},
				Relations: []engine.GraphUnitRelationship{
					{
						Name:             ".inner_events",
						RelationshipType: engine.GraphUnitRelationshipTypeOneToMany,
					},
					{
						Name:             "sessions",
						RelationshipType: engine.GraphUnitRelationshipTypeOneToMany,
					},
				},
			},
			{
				Unit: engine.StorageUnit{Name: "sessions"},
			},
			{
				Unit: engine.StorageUnit{Name: ".inner_events"},
			},
		}, nil
	}

	spec := testTypeSpec("ClickHouse", []source.Surface{source.SurfaceBrowser, source.SurfaceGraph})
	spec.Traits.Metadata.Graph = source.MetadataFidelityInferred
	spec.Traits.Metadata.HiddenObjectPrefixes = map[source.ObjectKind][]string{
		source.ObjectKindTable: {".inner"},
	}
	session := newTestDatabaseSession(spec, mock)

	graph, err := session.ReadGraph(context.Background(), nil)
	if err != nil {
		t.Fatalf("expected graph read to succeed, got %v", err)
	}
	if len(graph) != 2 {
		t.Fatalf("expected internal graph unit to be filtered, got %#v", graph)
	}
	if len(graph[0].Relations) != 1 {
		t.Fatalf("expected relation to hidden graph unit to be filtered, got %#v", graph[0].Relations)
	}
	if graph[0].Relations[0].MetadataFidelity != source.MetadataFidelityInferred {
		t.Fatalf("expected graph relationship metadata fidelity %q, got %q", source.MetadataFidelityInferred, graph[0].Relations[0].MetadataFidelity)
	}
}

func TestDatabaseSessionReplyRejectsUnsupportedSurface(t *testing.T) {
	mock := testutil.NewPluginMock(engine.DatabaseType("Redis"))
	mock.ChatFunc = func(*engine.PluginConfig, string, string, string) ([]*engine.ChatMessage, error) {
		t.Fatalf("expected chat to be blocked by the source contract")
		return nil, nil
	}

	session := newTestDatabaseSession(testTypeSpec("Redis", []source.Surface{source.SurfaceBrowser}), mock)

	_, err := session.Reply(context.Background(), nil, "", "hello")
	if err == nil || !strings.Contains(err.Error(), "chat") {
		t.Fatalf("expected chat error, got %v", err)
	}
}

func TestDatabaseSessionReadRowsRejectsUnsupportedAction(t *testing.T) {
	mock := testutil.NewPluginMock(engine.DatabaseType("Oracle"))
	mock.StorageUnitExistsFunc = func(*engine.PluginConfig, string, string) (bool, error) {
		t.Fatalf("expected row reads to be blocked before object validation")
		return false, nil
	}

	session := newTestDatabaseSession(testTypeWithObjectActions(
		"Oracle",
		[]source.Surface{source.SurfaceBrowser},
		[]source.Action{source.ActionBrowse},
		map[source.ObjectKind][]source.Action{
			source.ObjectKindTable: {source.ActionInspect},
		},
	), mock)

	_, err := session.ReadRows(context.Background(), testTableRef(), nil, nil, 10, 0)
	if err == nil || !strings.Contains(err.Error(), "viewing rows") {
		t.Fatalf("expected row-view error, got %v", err)
	}
}

func TestDatabaseSessionAddRowRejectsUnsupportedAction(t *testing.T) {
	mock := testutil.NewPluginMock(engine.DatabaseType("Memcached"))
	mock.StorageUnitExistsFunc = func(*engine.PluginConfig, string, string) (bool, error) {
		t.Fatalf("expected inserts to be blocked before object validation")
		return false, nil
	}

	session := newTestDatabaseSession(testTypeWithObjectActions(
		"Memcached",
		[]source.Surface{source.SurfaceBrowser},
		[]source.Action{source.ActionBrowse},
		map[source.ObjectKind][]source.Action{
			source.ObjectKindItem: {source.ActionInspect, source.ActionViewRows},
		},
	), mock)

	_, err := session.AddRow(context.Background(), source.NewObjectRef(source.ObjectKindItem, []string{"item-1"}), nil)
	if err == nil || !strings.Contains(err.Error(), "inserting data") {
		t.Fatalf("expected insert error, got %v", err)
	}
}

func TestDatabaseSessionUpdateObjectRejectsUnsupportedAction(t *testing.T) {
	mock := testutil.NewPluginMock(engine.DatabaseType("Athena"))
	mock.StorageUnitExistsFunc = func(*engine.PluginConfig, string, string) (bool, error) {
		t.Fatalf("expected updates to be blocked before object validation")
		return false, nil
	}

	session := newTestDatabaseSession(testTypeWithObjectActions(
		"Athena",
		[]source.Surface{source.SurfaceBrowser, source.SurfaceQuery},
		[]source.Action{source.ActionBrowse},
		map[source.ObjectKind][]source.Action{
			source.ObjectKindTable: {source.ActionInspect, source.ActionViewRows},
		},
	), mock)

	_, err := session.UpdateObject(context.Background(), testTableRef(), map[string]string{"id": "1"}, []string{"id"})
	if err == nil || !strings.Contains(err.Error(), "updating data") {
		t.Fatalf("expected update error, got %v", err)
	}
}

func TestDatabaseSessionDeleteRowRejectsUnsupportedAction(t *testing.T) {
	mock := testutil.NewPluginMock(engine.DatabaseType("Athena"))
	mock.StorageUnitExistsFunc = func(*engine.PluginConfig, string, string) (bool, error) {
		t.Fatalf("expected deletes to be blocked before object validation")
		return false, nil
	}

	session := newTestDatabaseSession(testTypeWithObjectActions(
		"Athena",
		[]source.Surface{source.SurfaceBrowser, source.SurfaceQuery},
		[]source.Action{source.ActionBrowse},
		map[source.ObjectKind][]source.Action{
			source.ObjectKindTable: {source.ActionInspect, source.ActionViewRows, source.ActionUpdateData},
		},
	), mock)

	_, err := session.DeleteRow(context.Background(), testTableRef(), map[string]string{"id": "1"})
	if err == nil || !strings.Contains(err.Error(), "deleting data") {
		t.Fatalf("expected delete error, got %v", err)
	}
}

func TestDatabaseSessionImportDataRejectsUnsupportedAction(t *testing.T) {
	mock := testutil.NewPluginMock(engine.DatabaseType("ElasticSearch"))
	mock.StorageUnitExistsFunc = func(*engine.PluginConfig, string, string) (bool, error) {
		t.Fatalf("expected imports to be blocked before object validation")
		return false, nil
	}

	session := newTestDatabaseSession(testTypeWithObjectActions(
		"ElasticSearch",
		[]source.Surface{source.SurfaceBrowser},
		[]source.Action{source.ActionBrowse},
		map[source.ObjectKind][]source.Action{
			source.ObjectKindIndex: {source.ActionInspect, source.ActionViewRows, source.ActionInsertData, source.ActionUpdateData},
		},
	), mock)

	_, err := session.ImportData(context.Background(), source.NewObjectRef(source.ObjectKindIndex, []string{"events"}), source.ImportRequest{})
	if err == nil || !strings.Contains(err.Error(), "importing data") {
		t.Fatalf("expected import error, got %v", err)
	}
}

func TestDatabaseSessionGenerateMockDataRejectsUnsupportedAction(t *testing.T) {
	mock := testutil.NewPluginMock(engine.DatabaseType("Redis"))
	session := newTestDatabaseSession(testTypeWithObjectActions(
		"Redis",
		[]source.Surface{source.SurfaceBrowser},
		[]source.Action{source.ActionBrowse},
		map[source.ObjectKind][]source.Action{
			source.ObjectKindKey: {source.ActionInspect, source.ActionViewRows, source.ActionInsertData, source.ActionUpdateData},
		},
	), mock)

	_, err := session.GenerateMockData(context.Background(), source.NewObjectRef(source.ObjectKindKey, []string{"0", "users"}), 10, 0, false)
	if err == nil || !strings.Contains(err.Error(), "generating mock data") {
		t.Fatalf("expected mock-data error, got %v", err)
	}
}

func TestDatabaseSessionCreateObjectRejectsUnsupportedParentAction(t *testing.T) {
	mock := testutil.NewPluginMock(engine.DatabaseType("Oracle"))
	mock.AddStorageUnitFunc = func(*engine.PluginConfig, string, string, []engine.Record) (bool, error) {
		t.Fatalf("expected object creation to be blocked by the source contract")
		return false, nil
	}

	session := newTestDatabaseSession(testTypeWithObjectActions(
		"Oracle",
		[]source.Surface{source.SurfaceBrowser},
		[]source.Action{source.ActionBrowse},
		map[source.ObjectKind][]source.Action{
			source.ObjectKindSchema: {source.ActionBrowse},
			source.ObjectKindTable:  {source.ActionInspect, source.ActionViewRows},
		},
	), mock)

	_, err := session.CreateObject(context.Background(), testSchemaRef(), "users", nil)
	if err == nil || !strings.Contains(err.Error(), "creating child objects") {
		t.Fatalf("expected create-child error, got %v", err)
	}
}

func TestDatabaseSessionAddRowAllowsSupportedAction(t *testing.T) {
	mock := testutil.NewPluginMock(engine.DatabaseType("Postgres"))
	mock.StorageUnitExistsFunc = func(*engine.PluginConfig, string, string) (bool, error) {
		return true, nil
	}
	addCalled := false
	mock.AddRowFunc = func(*engine.PluginConfig, string, string, []engine.Record) (bool, error) {
		addCalled = true
		return true, nil
	}

	session := newTestDatabaseSession(testTypeWithObjectActions(
		"Postgres",
		[]source.Surface{source.SurfaceBrowser, source.SurfaceQuery, source.SurfaceChat, source.SurfaceGraph},
		[]source.Action{source.ActionBrowse},
		map[source.ObjectKind][]source.Action{
			source.ObjectKindSchema: {source.ActionBrowse, source.ActionCreateChild},
			source.ObjectKindTable: {
				source.ActionInspect,
				source.ActionViewRows,
				source.ActionInsertData,
				source.ActionUpdateData,
				source.ActionImportData,
				source.ActionGenerateMockData,
			},
		},
	), mock)

	status, err := session.AddRow(context.Background(), testTableRef(), []source.Record{{Key: "name", Value: "alice"}})
	if err != nil {
		t.Fatalf("expected insert to succeed, got %v", err)
	}
	if !status || !addCalled {
		t.Fatalf("expected plugin insert to run, got status=%t addCalled=%t", status, addCalled)
	}
}

func TestDatabaseSessionUpdateObjectAllowsSupportedAction(t *testing.T) {
	mock := testutil.NewPluginMock(engine.DatabaseType("Postgres"))
	mock.StorageUnitExistsFunc = func(*engine.PluginConfig, string, string) (bool, error) {
		return true, nil
	}
	updateCalled := false
	mock.UpdateStorageUnitFunc = func(*engine.PluginConfig, string, string, map[string]string, []string) (bool, error) {
		updateCalled = true
		return true, nil
	}

	session := newTestDatabaseSession(testTypeWithObjectActions(
		"Postgres",
		[]source.Surface{source.SurfaceBrowser, source.SurfaceQuery, source.SurfaceChat, source.SurfaceGraph},
		[]source.Action{source.ActionBrowse},
		map[source.ObjectKind][]source.Action{
			source.ObjectKindSchema: {source.ActionBrowse, source.ActionCreateChild},
			source.ObjectKindTable: {
				source.ActionInspect,
				source.ActionViewRows,
				source.ActionInsertData,
				source.ActionUpdateData,
				source.ActionDeleteData,
				source.ActionImportData,
				source.ActionGenerateMockData,
			},
		},
	), mock)

	status, err := session.UpdateObject(context.Background(), testTableRef(), map[string]string{"id": "1", "name": "alice"}, []string{"name"})
	if err != nil {
		t.Fatalf("expected update to succeed, got %v", err)
	}
	if !status || !updateCalled {
		t.Fatalf("expected plugin update to run, got status=%t updateCalled=%t", status, updateCalled)
	}
}

func TestDatabaseSessionDeleteRowAllowsSupportedAction(t *testing.T) {
	mock := testutil.NewPluginMock(engine.DatabaseType("Postgres"))
	mock.StorageUnitExistsFunc = func(*engine.PluginConfig, string, string) (bool, error) {
		return true, nil
	}
	deleteCalled := false
	mock.DeleteRowFunc = func(*engine.PluginConfig, string, string, map[string]string) (bool, error) {
		deleteCalled = true
		return true, nil
	}

	session := newTestDatabaseSession(testTypeWithObjectActions(
		"Postgres",
		[]source.Surface{source.SurfaceBrowser, source.SurfaceQuery, source.SurfaceChat, source.SurfaceGraph},
		[]source.Action{source.ActionBrowse},
		map[source.ObjectKind][]source.Action{
			source.ObjectKindSchema: {source.ActionBrowse, source.ActionCreateChild},
			source.ObjectKindTable: {
				source.ActionInspect,
				source.ActionViewRows,
				source.ActionInsertData,
				source.ActionUpdateData,
				source.ActionDeleteData,
				source.ActionImportData,
				source.ActionGenerateMockData,
			},
		},
	), mock)

	status, err := session.DeleteRow(context.Background(), testTableRef(), map[string]string{"id": "1"})
	if err != nil {
		t.Fatalf("expected delete to succeed, got %v", err)
	}
	if !status || !deleteCalled {
		t.Fatalf("expected plugin delete to run, got status=%t deleteCalled=%t", status, deleteCalled)
	}
}

func TestDatabaseSessionColumnsCachesMetadataWithinSession(t *testing.T) {
	mock := testutil.NewPluginMock(engine.DatabaseType("Postgres"))

	validationCalls := 0
	columnCalls := 0
	markCalls := 0
	length := 32

	mock.StorageUnitExistsFunc = func(*engine.PluginConfig, string, string) (bool, error) {
		validationCalls++
		return true, nil
	}
	mock.GetColumnsForTableFunc = func(*engine.PluginConfig, string, string) ([]engine.Column, error) {
		columnCalls++
		return []engine.Column{
			{Name: "id", Type: "INTEGER", IsPrimary: true, Length: &length},
		}, nil
	}
	mock.MarkGeneratedColumnsFunc = func(_ *engine.PluginConfig, _ string, _ string, columns []engine.Column) error {
		markCalls++
		columns[0].IsAutoIncrement = true
		return nil
	}

	spec := testTypeWithObjectActions(
		"Postgres",
		[]source.Surface{source.SurfaceBrowser},
		[]source.Action{source.ActionBrowse},
		map[source.ObjectKind][]source.Action{
			source.ObjectKindTable: {source.ActionInspect, source.ActionViewRows},
		},
	)
	spec.Traits.Metadata.Columns = source.MetadataFidelityExact
	session := newTestDatabaseSession(spec, mock)

	columns, err := session.Columns(context.Background(), testTableRef())
	if err != nil {
		t.Fatalf("expected first columns call to succeed, got %v", err)
	}
	if len(columns) != 1 || columns[0].Length == nil {
		t.Fatalf("expected one cached column with length, got %#v", columns)
	}
	if columns[0].MetadataFidelity != source.MetadataFidelityExact {
		t.Fatalf("expected column metadata fidelity %q, got %q", source.MetadataFidelityExact, columns[0].MetadataFidelity)
	}

	columns[0].Name = "mutated"
	columns[0].IsAutoIncrement = false
	*columns[0].Length = 99

	cachedColumns, err := session.Columns(context.Background(), testTableRef())
	if err != nil {
		t.Fatalf("expected cached columns call to succeed, got %v", err)
	}
	if len(cachedColumns) != 1 {
		t.Fatalf("expected one cached column, got %#v", cachedColumns)
	}
	if cachedColumns[0].Name != "id" {
		t.Fatalf("expected cached column name to remain unchanged, got %#v", cachedColumns[0])
	}
	if !cachedColumns[0].IsAutoIncrement {
		t.Fatalf("expected cached generated-column metadata to be preserved, got %#v", cachedColumns[0])
	}
	if cachedColumns[0].Length == nil || *cachedColumns[0].Length != 32 {
		t.Fatalf("expected cached column length to remain 32, got %#v", cachedColumns[0].Length)
	}
	if validationCalls != 1 {
		t.Fatalf("expected one validation call, got %d", validationCalls)
	}
	if columnCalls != 1 {
		t.Fatalf("expected one column lookup call, got %d", columnCalls)
	}
	if markCalls != 1 {
		t.Fatalf("expected one generated-column call, got %d", markCalls)
	}
}

func TestDatabaseSessionFieldConstraintsAppliesMetadataContract(t *testing.T) {
	mock := testutil.NewPluginMock(engine.DatabaseType("Postgres"))
	mock.StorageUnitExistsFunc = func(*engine.PluginConfig, string, string) (bool, error) {
		return true, nil
	}
	mock.GetColumnsForTableFunc = func(*engine.PluginConfig, string, string) ([]engine.Column, error) {
		return []engine.Column{{Name: "id", Type: "INTEGER", IsPrimary: true}}, nil
	}
	mock.GetColumnConstraintsFunc = func(*engine.PluginConfig, string, string) (map[string]map[string]any, error) {
		return map[string]map[string]any{
			"id": {"nullable": false},
		}, nil
	}

	spec := testTypeWithObjectActions(
		"Postgres",
		[]source.Surface{source.SurfaceBrowser},
		[]source.Action{source.ActionBrowse},
		map[source.ObjectKind][]source.Action{
			source.ObjectKindTable: {source.ActionInspect, source.ActionViewRows},
		},
	)
	spec.Traits.Metadata.Columns = source.MetadataFidelityExact
	spec.Traits.Metadata.Constraints = source.MetadataFidelityExact
	session := newTestDatabaseSession(spec, mock)

	fields, err := session.FieldConstraints(context.Background(), testTableRef())
	if err != nil {
		t.Fatalf("expected field constraints to succeed, got %v", err)
	}
	if len(fields) != 1 {
		t.Fatalf("expected one field constraint, got %#v", fields)
	}
	if fields[0].MetadataFidelity != source.MetadataFidelityExact {
		t.Fatalf("expected constraint metadata fidelity %q, got %q", source.MetadataFidelityExact, fields[0].MetadataFidelity)
	}
}

func TestDatabaseSessionReadRowsThenColumnsReusesValidation(t *testing.T) {
	mock := testutil.NewPluginMock(engine.DatabaseType("Postgres"))

	validationCalls := 0
	rowCalls := 0
	columnCalls := 0
	markCalls := 0

	mock.StorageUnitExistsFunc = func(*engine.PluginConfig, string, string) (bool, error) {
		validationCalls++
		return true, nil
	}
	mock.GetRowsFunc = func(*engine.PluginConfig, *engine.GetRowsRequest) (*engine.GetRowsResult, error) {
		rowCalls++
		return &engine.GetRowsResult{
			Columns: []engine.Column{{Name: "id", Type: "INTEGER"}},
			Rows:    [][]string{{"1"}},
		}, nil
	}
	mock.GetColumnsForTableFunc = func(*engine.PluginConfig, string, string) ([]engine.Column, error) {
		columnCalls++
		return []engine.Column{{Name: "id", Type: "INTEGER"}}, nil
	}
	mock.MarkGeneratedColumnsFunc = func(_ *engine.PluginConfig, _ string, _ string, _ []engine.Column) error {
		markCalls++
		return nil
	}

	session := newTestDatabaseSession(testTypeWithObjectActions(
		"Postgres",
		[]source.Surface{source.SurfaceBrowser},
		[]source.Action{source.ActionBrowse},
		map[source.ObjectKind][]source.Action{
			source.ObjectKindTable: {source.ActionInspect, source.ActionViewRows},
		},
	), mock)

	if _, err := session.ReadRows(context.Background(), testTableRef(), nil, nil, 10, 0); err != nil {
		t.Fatalf("expected row read to succeed, got %v", err)
	}
	if _, err := session.Columns(context.Background(), testTableRef()); err != nil {
		t.Fatalf("expected columns lookup to succeed, got %v", err)
	}

	if validationCalls != 1 {
		t.Fatalf("expected validation to run once across row and column reads, got %d", validationCalls)
	}
	if rowCalls != 1 {
		t.Fatalf("expected one row lookup, got %d", rowCalls)
	}
	if columnCalls != 1 {
		t.Fatalf("expected one column lookup, got %d", columnCalls)
	}
	if markCalls != 1 {
		t.Fatalf("expected one generated-column call, got %d", markCalls)
	}
}

func newTestDatabaseSession(spec source.TypeSpec, mock *testutil.PluginMock) *DatabaseSession {
	spec.Contract = source.NormalizeContract(spec.Contract)
	return &DatabaseSession{
		spec:      spec,
		plugin:    mock.AsPlugin(),
		validated: map[databaseObjectCacheKey]struct{}{},
		columns:   map[databaseObjectCacheKey][]source.Column{},
		credentials: &source.Credentials{
			SourceType: spec.ID,
			Values: map[string]string{
				"Database": "app",
			},
		},
	}
}

func testTypeSpec(label string, surfaces []source.Surface) source.TypeSpec {
	return testTypeWithObjectActions(label, surfaces, []source.Action{source.ActionBrowse}, map[source.ObjectKind][]source.Action{})
}

func testTypeWithObjectActions(label string, surfaces []source.Surface, rootActions []source.Action, objectActions map[source.ObjectKind][]source.Action) source.TypeSpec {
	objectTypes := make([]source.ObjectType, 0, len(objectActions))
	for kind, actions := range objectActions {
		objectTypes = append(objectTypes, source.ObjectType{
			Kind:      kind,
			DataShape: source.DataShapeTabular,
			Actions:   actions,
			Views:     []source.View{source.ViewGrid, source.ViewMetadata},
		})
	}

	return source.TypeSpec{
		ID:        label,
		Label:     label,
		Connector: label,
		Contract: source.Contract{
			Surfaces:          surfaces,
			RootActions:       rootActions,
			BrowsePath:        []source.ObjectKind{source.ObjectKindDatabase, source.ObjectKindSchema, source.ObjectKindTable},
			DefaultObjectKind: source.ObjectKindTable,
			ObjectTypes:       objectTypes,
		},
	}
}

func testTableRef() source.ObjectRef {
	return source.NewObjectRef(source.ObjectKindTable, []string{"app", "public", "users"})
}

func testSchemaRef() *source.ObjectRef {
	ref := source.NewObjectRef(source.ObjectKindSchema, []string{"app", "public"})
	return &ref
}
