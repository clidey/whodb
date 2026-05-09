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

// Package adapters exposes source connectors backed by the existing database
// plugin layer.
package adapters

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/clidey/whodb/core/src"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/importer"
	"github.com/clidey/whodb/core/src/log"
	"github.com/clidey/whodb/core/src/plugins"
	"github.com/clidey/whodb/core/src/query"
	"github.com/clidey/whodb/core/src/querysuggestions"
	"github.com/clidey/whodb/core/src/source"
	"github.com/clidey/whodb/core/src/sourcecatalog"
)

func init() {
	source.RegisterDriver("database", &DatabaseConnector{})
}

// DatabaseConnector opens source sessions backed by the existing database plugin layer.
type DatabaseConnector struct{}

// EngineCredentials converts source-first credentials into the legacy engine
// credential shape required by the current database plugins.
func EngineCredentials(spec source.TypeSpec, credentials *source.Credentials) *engine.Credentials {
	session := &DatabaseSession{
		spec:        spec,
		credentials: credentials,
	}
	return session.engineCredentials(nil)
}

// Open creates a new database-backed source session.
func (c *DatabaseConnector) Open(_ context.Context, spec source.TypeSpec, credentials *source.Credentials) (source.SourceSession, error) {
	plugin := src.MainEngine.Choose(engine.DatabaseType(spec.Connector))
	if plugin == nil {
		return nil, fmt.Errorf("unsupported source connector: %s", spec.Connector)
	}

	return &DatabaseSession{
		spec:        spec,
		plugin:      plugin,
		credentials: credentials,
		validated:   map[databaseObjectCacheKey]struct{}{},
		columns:     map[databaseObjectCacheKey][]source.Column{},
	}, nil
}

// Invalidate clears cached plugin connection state for one database-backed
// source credential set.
func (c *DatabaseConnector) Invalidate(_ context.Context, spec source.TypeSpec, credentials *source.Credentials) error {
	plugins.RemoveConnection(engine.NewPluginConfig(EngineCredentials(spec, credentials)))
	return nil
}

// Shutdown releases all cached plugin connections owned by the database-backed
// source driver.
func (c *DatabaseConnector) Shutdown(ctx context.Context) error {
	plugins.CloseAllConnections(ctx)
	return nil
}

// DatabaseSession adapts one database plugin instance to the source session interfaces.
type DatabaseSession struct {
	spec        source.TypeSpec
	plugin      *engine.Plugin
	credentials *source.Credentials
	mu          sync.RWMutex
	validated   map[databaseObjectCacheKey]struct{}
	columns     map[databaseObjectCacheKey][]source.Column
}

type databaseObjectCacheKey struct {
	namespace string
	name      string
}

type sourceQueryStreamWriterAdapter struct {
	writer source.QueryStreamWriter
}

func (w *sourceQueryStreamWriterAdapter) WriteColumns(columns []engine.Column) error {
	return w.writer.WriteColumns(columns)
}

func (w *sourceQueryStreamWriterAdapter) WriteRow(row []string) error {
	return w.writer.WriteRow(row)
}

// Metadata returns source session metadata derived from the source registry.
func (s *DatabaseSession) Metadata(_ context.Context) (*source.SessionMetadata, error) {
	metadata, _ := sourcecatalog.ResolveSessionMetadata(s.spec.ID, s.spec.Connector)
	if metadata == nil {
		metadata = &source.TypeSessionMetadata{}
	}

	return &source.SessionMetadata{
		SourceType:      s.spec.ID,
		QueryLanguages:  queryLanguagesForSpec(s.spec),
		TypeDefinitions: slices.Clone(metadata.TypeDefinitions),
		Operators:       slices.Clone(metadata.Operators),
		AliasMap:        cloneAliasMap(metadata.AliasMap),
	}, nil
}

// ConnectionFieldOptions returns dynamic options for a connection field.
func (s *DatabaseSession) ConnectionFieldOptions(ctx context.Context, fieldKey string, values map[string]string) ([]string, error) {
	if !strings.EqualFold(fieldKey, "Database") {
		return []string{}, nil
	}

	config := engine.NewPluginConfig(s.engineCredentials(values))
	config.Context = ctx
	return s.plugin.GetDatabases(config)
}

// ListObjects lists child objects beneath the provided parent.
func (s *DatabaseSession) ListObjects(ctx context.Context, parent *source.ObjectRef, kinds []source.ObjectKind) ([]source.Object, error) {
	nextKind, ok := s.nextKind(parent)
	if !ok {
		return []source.Object{}, nil
	}
	if len(kinds) > 0 && !slices.Contains(kinds, nextKind) {
		return []source.Object{}, nil
	}

	config := s.pluginConfig(ctx, parent)

	switch nextKind {
	case source.ObjectKindDatabase:
		names, err := s.plugin.GetDatabases(config)
		if err != nil {
			return nil, err
		}
		objects := make([]source.Object, 0, len(names))
		for _, name := range names {
			objects = append(objects, s.makeContainerObject(parent, nextKind, name, nil))
		}
		return objects, nil
	case source.ObjectKindSchema:
		names, err := s.plugin.GetAllSchemas(config)
		if err != nil {
			return nil, err
		}
		objects := make([]source.Object, 0, len(names))
		for _, name := range names {
			objects = append(objects, s.makeContainerObject(parent, nextKind, name, nil))
		}
		return objects, nil
	default:
		namespace := s.namespaceForRef(parent)
		units, err := s.plugin.GetStorageUnits(config, namespace)
		if err != nil {
			return nil, err
		}
		objects := make([]source.Object, 0, len(units))
		for _, unit := range units {
			kind := s.kindForUnit(nextKind, unit)
			objectType, _ := s.spec.Contract.ObjectTypeForKind(kind)
			path := appendPath(parent, unit.Name)
			objects = append(objects, source.Object{
				Ref:         source.NewObjectRef(kind, path),
				Kind:        kind,
				Name:        unit.Name,
				Path:        path,
				HasChildren: s.hasChildren(kind),
				Actions:     slices.Clone(objectType.Actions),
				Metadata:    slices.Clone(unit.Attributes),
			})
		}
		return objects, nil
	}
}

// GetObject loads one object by reference.
func (s *DatabaseSession) GetObject(ctx context.Context, ref source.ObjectRef) (*source.Object, error) {
	parent := parentForRef(ref)
	objects, err := s.ListObjects(ctx, parent, []source.ObjectKind{ref.Kind})
	if err != nil {
		return nil, err
	}

	for _, object := range objects {
		if object.Kind == ref.Kind && slices.Equal(object.Path, ref.Path) {
			objectCopy := object
			return &objectCopy, nil
		}
	}

	return nil, fmt.Errorf("source object not found")
}

// ReadRows returns rows for a tabular source object.
func (s *DatabaseSession) ReadRows(ctx context.Context, ref source.ObjectRef, where *query.WhereCondition, sort []*query.SortCondition, pageSize int, pageOffset int) (*source.RowsResult, error) {
	if err := s.ensureObjectAction(ref.Kind, source.ActionViewRows); err != nil {
		return nil, err
	}

	config := s.pluginConfig(ctx, &ref)
	namespace := s.namespaceForRef(&ref)
	name := objectName(ref)
	log.WithFields(map[string]any{
		"sourceType": s.spec.ID,
		"connector":  s.spec.Connector,
		"kind":       ref.Kind,
		"path":       slices.Clone(ref.Path),
		"namespace":  namespace,
		"objectName": name,
		"pageSize":   pageSize,
		"pageOffset": pageOffset,
		"hasWhere":   where != nil,
		"sortCount":  len(sort),
	}).Debug("Source row read requested")
	if err := s.validateObject(config, namespace, name); err != nil {
		log.WithError(err).WithFields(map[string]any{
			"sourceType": s.spec.ID,
			"connector":  s.spec.Connector,
			"kind":       ref.Kind,
			"path":       slices.Clone(ref.Path),
			"namespace":  namespace,
			"objectName": name,
		}).Debug("Source row read failed validation")
		return nil, err
	}

	rows, err := s.plugin.GetRows(config, &engine.GetRowsRequest{
		Schema:      namespace,
		StorageUnit: name,
		Where:       where,
		Sort:        sort,
		PageSize:    pageSize,
		PageOffset:  pageOffset,
	})
	if err != nil {
		log.WithError(err).WithFields(map[string]any{
			"sourceType": s.spec.ID,
			"connector":  s.spec.Connector,
			"kind":       ref.Kind,
			"path":       slices.Clone(ref.Path),
			"namespace":  namespace,
			"objectName": name,
		}).Debug("Source row read failed in plugin")
		return nil, err
	}

	log.WithFields(map[string]any{
		"sourceType":  s.spec.ID,
		"connector":   s.spec.Connector,
		"kind":        ref.Kind,
		"path":        slices.Clone(ref.Path),
		"namespace":   namespace,
		"objectName":  name,
		"rowCount":    len(rows.Rows),
		"columnCount": len(rows.Columns),
		"totalCount":  rows.TotalCount,
	}).Debug("Source row read completed")
	return rows, nil
}

// Columns returns columns for one source object.
func (s *DatabaseSession) Columns(ctx context.Context, ref source.ObjectRef) ([]source.Column, error) {
	if err := s.ensureObjectAction(ref.Kind, source.ActionInspect); err != nil {
		return nil, err
	}

	config := s.pluginConfig(ctx, &ref)
	namespace := s.namespaceForRef(&ref)
	name := objectName(ref)
	if err := s.validateObject(config, namespace, name); err != nil {
		return nil, err
	}
	if cachedColumns, ok := s.cachedColumns(namespace, name); ok {
		return cachedColumns, nil
	}

	columns, err := s.plugin.GetColumnsForTable(config, namespace, name)
	if err != nil {
		return nil, err
	}
	if err := s.plugin.MarkGeneratedColumns(config, namespace, name, columns); err != nil {
		log.WithError(err).Warn("Failed to mark generated columns for source object")
	}
	s.cacheColumns(namespace, name, columns)
	return cloneSourceColumns(columns), nil
}

// ColumnsBatch returns columns for multiple source objects.
func (s *DatabaseSession) ColumnsBatch(ctx context.Context, refs []source.ObjectRef) ([]source.ObjectColumns, error) {
	results := make([]source.ObjectColumns, 0, len(refs))
	for _, ref := range refs {
		columns, err := s.Columns(ctx, ref)
		if err != nil {
			continue
		}
		results = append(results, source.ObjectColumns{
			Ref:     ref,
			Columns: columns,
		})
	}
	return results, nil
}

// ColumnConstraints returns per-column constraint metadata for one source
// object.
func (s *DatabaseSession) ColumnConstraints(ctx context.Context, ref source.ObjectRef) (map[string]map[string]any, error) {
	if err := s.ensureObjectAction(ref.Kind, source.ActionInspect); err != nil {
		return nil, err
	}

	config := s.pluginConfig(ctx, &ref)
	namespace := s.namespaceForRef(&ref)
	name := objectName(ref)
	if err := s.validateObject(config, namespace, name); err != nil {
		return nil, err
	}

	return s.plugin.GetColumnConstraints(config, namespace, name)
}

// RunQuery executes a query against the source session.
func (s *DatabaseSession) RunQuery(ctx context.Context, query string, params ...any) (*source.RowsResult, error) {
	if err := s.ensureSurface(source.SurfaceQuery); err != nil {
		return nil, err
	}

	config := s.pluginConfig(ctx, nil)
	return s.plugin.RawExecute(config, query, params...)
}

// RunQueryStream executes a query through the active source session's
// streaming path when supported.
func (s *DatabaseSession) RunQueryStream(ctx context.Context, query string, writer source.QueryStreamWriter, params ...any) error {
	if err := s.ensureSurface(source.SurfaceQuery); err != nil {
		return err
	}
	if writer == nil {
		return fmt.Errorf("stream writer is required")
	}

	streamer, ok := s.plugin.PluginFunctions.(engine.QueryStreamer)
	if !ok {
		return fmt.Errorf("streaming queries are not supported for %s", s.spec.Label)
	}

	config := s.pluginConfig(ctx, nil)
	return streamer.StreamRawExecute(config, query, &sourceQueryStreamWriterAdapter{writer: writer}, params...)
}

// RunScript executes a source-native script against the session.
func (s *DatabaseSession) RunScript(ctx context.Context, script string, multiStatement bool, params ...any) (*source.RowsResult, error) {
	if err := s.ensureSurface(source.SurfaceQuery); err != nil {
		return nil, err
	}

	config := s.pluginConfig(ctx, nil)
	config.MultiStatement = multiStatement
	return s.plugin.RawExecute(config, script, params...)
}

// ReadGraph returns graph data for a source scope.
func (s *DatabaseSession) ReadGraph(ctx context.Context, ref *source.ObjectRef) ([]source.GraphUnit, error) {
	if err := s.ensureSurface(source.SurfaceGraph); err != nil {
		return nil, err
	}

	config := s.pluginConfig(ctx, ref)
	scope := ""
	if ref != nil {
		scope = s.graphScopeForRef(*ref)
	}
	return s.plugin.GetGraph(config, scope)
}

// Reply runs AI chat against the source session.
func (s *DatabaseSession) Reply(ctx context.Context, ref *source.ObjectRef, previousConversation string, query string) ([]*source.ChatMessage, error) {
	if err := s.ensureSurface(source.SurfaceChat); err != nil {
		return nil, err
	}

	config := s.pluginConfig(ctx, ref)
	scope := ""
	if ref != nil {
		scope = s.graphScopeForRef(*ref)
	}
	return s.plugin.Chat(config, scope, previousConversation, query)
}

// ReplyWithModel runs AI chat against the source session with an explicit model.
func (s *DatabaseSession) ReplyWithModel(ctx context.Context, ref *source.ObjectRef, previousConversation string, query string, model *source.ExternalModel) ([]*source.ChatMessage, error) {
	if err := s.ensureSurface(source.SurfaceChat); err != nil {
		return nil, err
	}

	config := s.pluginConfig(ctx, ref)
	config.ExternalModel = model
	scope := ""
	if ref != nil {
		scope = s.graphScopeForRef(*ref)
	}
	return s.plugin.Chat(config, scope, previousConversation, query)
}

// CreateObject creates a new source object.
func (s *DatabaseSession) CreateObject(ctx context.Context, parent *source.ObjectRef, name string, fields []source.Record) (bool, error) {
	if err := s.ensureCreateChildSupported(parent); err != nil {
		return false, err
	}

	config := s.pluginConfig(ctx, parent)
	namespace := s.namespaceForRef(parent)
	return s.plugin.AddStorageUnit(config, namespace, name, fields)
}

// UpdateObject updates data within an existing source object.
func (s *DatabaseSession) UpdateObject(ctx context.Context, ref source.ObjectRef, values map[string]string, updatedColumns []string) (bool, error) {
	if err := s.ensureObjectAction(ref.Kind, source.ActionUpdateData); err != nil {
		return false, err
	}

	config := s.pluginConfig(ctx, &ref)
	namespace := s.namespaceForRef(&ref)
	name := objectName(ref)
	if err := s.validateObject(config, namespace, name); err != nil {
		return false, err
	}
	return s.plugin.UpdateStorageUnit(config, namespace, name, values, updatedColumns)
}

// AddRow inserts a row into a source object.
func (s *DatabaseSession) AddRow(ctx context.Context, ref source.ObjectRef, values []source.Record) (bool, error) {
	if err := s.ensureObjectAction(ref.Kind, source.ActionInsertData); err != nil {
		return false, err
	}

	config := s.pluginConfig(ctx, &ref)
	namespace := s.namespaceForRef(&ref)
	name := objectName(ref)
	if err := s.validateObject(config, namespace, name); err != nil {
		return false, err
	}
	return s.plugin.AddRow(config, namespace, name, values)
}

// DeleteRow deletes row/document data from a source object.
func (s *DatabaseSession) DeleteRow(ctx context.Context, ref source.ObjectRef, values map[string]string) (bool, error) {
	if err := s.ensureObjectAction(ref.Kind, source.ActionDeleteData); err != nil {
		return false, err
	}

	config := s.pluginConfig(ctx, &ref)
	namespace := s.namespaceForRef(&ref)
	name := objectName(ref)
	if err := s.validateObject(config, namespace, name); err != nil {
		return false, err
	}
	return s.plugin.DeleteRow(config, namespace, name, values)
}

// IsAvailable verifies that the underlying database plugin can reach the
// source with the current credentials.
func (s *DatabaseSession) IsAvailable(ctx context.Context) bool {
	config := s.pluginConfig(ctx, nil)
	return s.plugin.IsAvailable(ctx, config)
}

// ExportRows streams tabular rows for one source object.
func (s *DatabaseSession) ExportRows(ctx context.Context, ref source.ObjectRef, writer func([]string) error, selectedRows []map[string]any) error {
	if err := s.ensureObjectAction(ref.Kind, source.ActionViewRows); err != nil {
		return err
	}

	config := s.pluginConfig(ctx, &ref)
	namespace := s.namespaceForRef(&ref)
	name := objectName(ref)
	return s.plugin.ExportData(config, namespace, name, writer, selectedRows)
}

// ExportRowsNDJSON streams NDJSON rows for one source object.
func (s *DatabaseSession) ExportRowsNDJSON(ctx context.Context, ref source.ObjectRef, writer func(string) error, selectedRows []map[string]any) error {
	if err := s.ensureObjectAction(ref.Kind, source.ActionViewRows); err != nil {
		return err
	}

	config := s.pluginConfig(ctx, &ref)
	namespace := s.namespaceForRef(&ref)
	name := objectName(ref)

	exporter, ok := s.plugin.PluginFunctions.(interface {
		ExportDataNDJSON(config *engine.PluginConfig, namespace string, objectName string, writer func(string) error, selectedRows []map[string]any) error
	})
	if !ok {
		return fmt.Errorf("NDJSON export is not supported")
	}

	return exporter.ExportDataNDJSON(config, namespace, name, writer, selectedRows)
}

// SSLStatus returns the current SSL/TLS status for the active source session.
func (s *DatabaseSession) SSLStatus(ctx context.Context) (*source.SSLStatus, error) {
	config := s.pluginConfig(ctx, nil)
	return s.plugin.GetSSLStatus(config)
}

// ImportData imports parsed tabular data into one source object.
func (s *DatabaseSession) ImportData(ctx context.Context, ref source.ObjectRef, request source.ImportRequest) (*source.ImportResult, error) {
	if err := s.ensureObjectAction(ref.Kind, source.ActionImportData); err != nil {
		return nil, err
	}

	config := s.pluginConfig(ctx, &ref)
	namespace := s.namespaceForRef(&ref)
	name := objectName(ref)
	if err := s.validateObject(config, namespace, name); err != nil {
		return nil, err
	}

	columns, err := s.Columns(ctx, ref)
	if err != nil {
		return nil, err
	}

	result, err := importer.Execute(s.plugin, config, &importer.ExecuteRequest{
		Schema:             namespace,
		StorageUnit:        name,
		Mode:               importer.Mode(request.Mode),
		Parsed:             &importer.ParsedFile{Columns: slices.Clone(request.Parsed.Columns), Rows: cloneImportRows(request.Parsed.Rows), Truncated: request.Parsed.Truncated, Sheet: request.Parsed.Sheet},
		Mapping:            importerColumnMappings(request.Mapping),
		AllowAutoGenerated: request.AllowAutoGenerated,
		BatchSize:          request.BatchSize,
		TargetColumns:      columns,
	})
	if err != nil {
		return nil, err
	}

	return &source.ImportResult{RowsImported: result.RowsImported}, nil
}

// GenerateMockData creates synthetic data for one source object.
func (s *DatabaseSession) GenerateMockData(ctx context.Context, ref source.ObjectRef, rowCount int, fkDensityRatio int, overwriteExisting bool) (*source.MockDataGenerationResult, error) {
	if err := s.ensureObjectAction(ref.Kind, source.ActionGenerateMockData); err != nil {
		return nil, err
	}

	config := s.pluginConfig(ctx, &ref)
	namespace := s.namespaceForRef(&ref)
	name := objectName(ref)

	generator := src.NewMockDataGenerator(fkDensityRatio)
	result, err := generator.Generate(s.plugin, config, namespace, name, rowCount, overwriteExisting)
	if err != nil {
		return nil, err
	}

	details := make([]source.MockDataTableDetail, 0, len(result.Details))
	for _, detail := range result.Details {
		details = append(details, source.MockDataTableDetail{
			Table:            detail.Table,
			RowsGenerated:    detail.RowsGenerated,
			UsedExistingData: detail.UsedExistingData,
		})
	}

	return &source.MockDataGenerationResult{
		TotalGenerated: result.TotalGenerated,
		Details:        details,
		Warnings:       slices.Clone(result.Warnings),
	}, nil
}

// AnalyzeMockDataDependencies computes the dependency plan for mock-data
// generation on one source object.
func (s *DatabaseSession) AnalyzeMockDataDependencies(ctx context.Context, ref source.ObjectRef, rowCount int, fkDensityRatio int) (*source.MockDataDependencyAnalysis, error) {
	if err := s.ensureObjectAction(ref.Kind, source.ActionGenerateMockData); err != nil {
		return nil, err
	}

	config := s.pluginConfig(ctx, &ref)
	namespace := s.namespaceForRef(&ref)
	name := objectName(ref)

	generator := src.NewMockDataGenerator(fkDensityRatio)
	analysis, err := generator.AnalyzeDependencies(s.plugin, config, namespace, name, rowCount)
	if err != nil {
		return nil, err
	}

	tables := make([]source.MockDataTableDependency, 0, len(analysis.Tables))
	for _, table := range analysis.Tables {
		tables = append(tables, source.MockDataTableDependency{
			Table:            table.Table,
			DependsOn:        slices.Clone(table.DependsOn),
			RowCount:         table.RowCount,
			IsBlocked:        table.IsBlocked,
			UsesExistingData: table.UsesExistingData,
		})
	}

	return &source.MockDataDependencyAnalysis{
		GenerationOrder: slices.Clone(analysis.GenerationOrder),
		Tables:          tables,
		TotalRows:       analysis.TotalRows,
		Warnings:        slices.Clone(analysis.Warnings),
		Error:           analysis.Error,
	}, nil
}

// QuerySuggestions returns source-scoped suggestions for the query UI.
func (s *DatabaseSession) QuerySuggestions(ctx context.Context, ref *source.ObjectRef) ([]source.QuerySuggestion, error) {
	config := s.pluginConfig(ctx, ref)
	scope := s.namespaceForRef(ref)
	units, err := s.plugin.GetStorageUnits(config, scope)
	if err != nil {
		return nil, err
	}
	suggestions := querysuggestions.FromStorageUnits(units)

	mapped := make([]source.QuerySuggestion, 0, len(suggestions))
	for _, suggestion := range suggestions {
		mapped = append(mapped, source.QuerySuggestion{
			Description: suggestion.Description,
			Category:    suggestion.Category,
		})
	}
	return mapped, nil
}

func (s *DatabaseSession) engineCredentials(values map[string]string) *engine.Credentials {
	mergedValues := s.credentials.CloneValues()
	for _, field := range s.spec.ConnectionFields {
		if field.DefaultValue == "" {
			continue
		}
		if _, ok := mergedValues[field.Key]; !ok {
			mergedValues[field.Key] = field.DefaultValue
		}
	}
	for key, value := range values {
		mergedValues[key] = value
	}

	engineCredentials := &engine.Credentials{
		Id:          s.credentials.ID,
		Type:        s.spec.ID,
		AccessToken: s.credentials.AccessToken,
		IsProfile:   s.credentials.IsProfile,
	}

	knownFields := map[string]bool{}
	for _, field := range s.spec.ConnectionFields {
		value := mergedValues[field.Key]
		if value == "" {
			continue
		}
		knownFields[field.Key] = true

		switch field.CredentialField {
		case source.CredentialFieldHostname:
			engineCredentials.Hostname = value
		case source.CredentialFieldUsername:
			engineCredentials.Username = value
		case source.CredentialFieldPassword:
			engineCredentials.Password = value
		case source.CredentialFieldDatabase:
			engineCredentials.Database = value
		case source.CredentialFieldAdvanced:
			advancedKey := field.AdvancedKey
			if advancedKey == "" {
				advancedKey = field.Key
			}
			engineCredentials.Advanced = append(engineCredentials.Advanced, engine.Record{
				Key:   advancedKey,
				Value: value,
			})
		}
	}

	for key, value := range mergedValues {
		if value == "" || knownFields[key] {
			continue
		}
		engineCredentials.Advanced = append(engineCredentials.Advanced, engine.Record{
			Key:   key,
			Value: value,
		})
	}

	return engineCredentials
}

func (s *DatabaseSession) credentialsForRef(ref *source.ObjectRef) *engine.Credentials {
	engineCredentials := s.engineCredentials(nil)
	if ref == nil {
		return engineCredentials
	}

	if databaseName := s.valueForKind(ref.Path, source.ObjectKindDatabase); databaseName != "" {
		engineCredentials.Database = databaseName
	}

	return engineCredentials
}

func (s *DatabaseSession) pluginConfig(ctx context.Context, ref *source.ObjectRef) *engine.PluginConfig {
	config := engine.NewPluginConfig(s.credentialsForRef(ref))
	config.Context = ctx
	return config
}

func (s *DatabaseSession) nextKind(parent *source.ObjectRef) (source.ObjectKind, bool) {
	depth := 0
	if parent != nil {
		depth = len(parent.Path)
	}
	if depth >= len(s.spec.Contract.BrowsePath) {
		return "", false
	}
	return s.spec.Contract.BrowsePath[depth], true
}

func (s *DatabaseSession) namespaceForRef(ref *source.ObjectRef) string {
	if ref == nil {
		return ""
	}

	defaultIndex := slices.Index(s.spec.Contract.BrowsePath, s.spec.Contract.DefaultObjectKind)
	if defaultIndex <= 0 || defaultIndex-1 >= len(ref.Path) {
		return ""
	}
	return ref.Path[defaultIndex-1]
}

func (s *DatabaseSession) graphScopeForRef(ref source.ObjectRef) string {
	if s.spec.Contract.GraphScopeKind == nil {
		return ""
	}
	return s.valueForKind(ref.Path, *s.spec.Contract.GraphScopeKind)
}

func (s *DatabaseSession) valueForKind(path []string, kind source.ObjectKind) string {
	index := slices.Index(s.spec.Contract.BrowsePath, kind)
	if index < 0 || index >= len(path) {
		return ""
	}
	return path[index]
}

func (s *DatabaseSession) makeContainerObject(parent *source.ObjectRef, kind source.ObjectKind, name string, metadata []engine.Record) source.Object {
	objectType, _ := s.spec.Contract.ObjectTypeForKind(kind)
	path := appendPath(parent, name)
	return source.Object{
		Ref:         source.NewObjectRef(kind, path),
		Kind:        kind,
		Name:        name,
		Path:        path,
		HasChildren: s.hasChildren(kind),
		Actions:     slices.Clone(objectType.Actions),
		Metadata:    slices.Clone(metadata),
	}
}

func (s *DatabaseSession) hasChildren(kind source.ObjectKind) bool {
	index := slices.Index(s.spec.Contract.BrowsePath, kind)
	return index >= 0 && index < len(s.spec.Contract.BrowsePath)-1
}

func (s *DatabaseSession) kindForUnit(defaultKind source.ObjectKind, unit engine.StorageUnit) source.ObjectKind {
	for _, attribute := range unit.Attributes {
		if !strings.EqualFold(attribute.Key, "Type") {
			continue
		}

		switch strings.ToUpper(strings.TrimSpace(attribute.Value)) {
		case "TABLE":
			return source.ObjectKindTable
		case "VIEW":
			return source.ObjectKindView
		case "COLLECTION":
			return source.ObjectKindCollection
		case "INDEX":
			return source.ObjectKindIndex
		case "KEY":
			return source.ObjectKindKey
		case "ITEM":
			return source.ObjectKindItem
		}
	}
	return defaultKind
}

func (s *DatabaseSession) validateObject(config *engine.PluginConfig, namespace string, name string) error {
	if s.isValidatedObject(namespace, name) {
		return nil
	}

	log.WithFields(map[string]any{
		"sourceType": s.spec.ID,
		"connector":  s.spec.Connector,
		"namespace":  namespace,
		"objectName": name,
	}).Debug("Validating source object")
	exists, err := s.plugin.StorageUnitExists(config, namespace, name)
	if err != nil {
		log.WithError(err).WithFields(map[string]any{
			"sourceType": s.spec.ID,
			"connector":  s.spec.Connector,
			"namespace":  namespace,
			"objectName": name,
		}).Debug("Source object validation errored")
		return fmt.Errorf("failed to validate source object: %w", err)
	}
	if !exists {
		log.WithFields(map[string]any{
			"sourceType": s.spec.ID,
			"connector":  s.spec.Connector,
			"namespace":  namespace,
			"objectName": name,
		}).Debug("Source object validation reported missing object")
		return fmt.Errorf("source object %q not found", name)
	}
	log.WithFields(map[string]any{
		"sourceType": s.spec.ID,
		"connector":  s.spec.Connector,
		"namespace":  namespace,
		"objectName": name,
	}).Debug("Source object validation succeeded")
	s.rememberValidatedObject(namespace, name)
	return nil
}

func (s *DatabaseSession) cacheKey(namespace string, name string) databaseObjectCacheKey {
	return databaseObjectCacheKey{namespace: namespace, name: name}
}

func (s *DatabaseSession) isValidatedObject(namespace string, name string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.validated) == 0 {
		return false
	}

	_, ok := s.validated[s.cacheKey(namespace, name)]
	return ok
}

func (s *DatabaseSession) rememberValidatedObject(namespace string, name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.validated == nil {
		s.validated = map[databaseObjectCacheKey]struct{}{}
	}

	s.validated[s.cacheKey(namespace, name)] = struct{}{}
}

func (s *DatabaseSession) cachedColumns(namespace string, name string) ([]source.Column, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.columns) == 0 {
		return nil, false
	}

	columns, ok := s.columns[s.cacheKey(namespace, name)]
	if !ok {
		return nil, false
	}

	return cloneSourceColumns(columns), true
}

func (s *DatabaseSession) cacheColumns(namespace string, name string, columns []source.Column) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.columns == nil {
		s.columns = map[databaseObjectCacheKey][]source.Column{}
	}

	s.columns[s.cacheKey(namespace, name)] = cloneSourceColumns(columns)
	if s.validated == nil {
		s.validated = map[databaseObjectCacheKey]struct{}{}
	}
	s.validated[s.cacheKey(namespace, name)] = struct{}{}
}

func (s *DatabaseSession) ensureSurface(surface source.Surface) error {
	if s.spec.Contract.SupportsSurface(surface) {
		return nil
	}

	return fmt.Errorf("%s is not supported for %s", sourceSurfaceDescription(surface), s.spec.Label)
}

func (s *DatabaseSession) ensureObjectAction(kind source.ObjectKind, action source.Action) error {
	objectType, ok := s.spec.Contract.ObjectTypeForKind(kind)
	if !ok {
		return fmt.Errorf("%s objects are not supported for %s", kind, s.spec.Label)
	}
	if objectType.SupportsAction(action) {
		return nil
	}

	return fmt.Errorf("%s is not supported for %s objects in %s", sourceActionDescription(action), kind, s.spec.Label)
}

func (s *DatabaseSession) ensureCreateChildSupported(parent *source.ObjectRef) error {
	if parent == nil {
		if slices.Contains(s.spec.Contract.RootActions, source.ActionCreateChild) {
			return nil
		}
		return fmt.Errorf("%s is not supported at the source root for %s", sourceActionDescription(source.ActionCreateChild), s.spec.Label)
	}

	return s.ensureObjectAction(parent.Kind, source.ActionCreateChild)
}

func queryLanguagesForSpec(spec source.TypeSpec) []string {
	if spec.Contract.SupportsSurface(source.SurfaceQuery) {
		return []string{"sql"}
	}
	return []string{}
}

func cloneAliasMap(aliasMap map[string]string) map[string]string {
	if len(aliasMap) == 0 {
		return map[string]string{}
	}

	cloned := make(map[string]string, len(aliasMap))
	for key, value := range aliasMap {
		cloned[key] = value
	}
	return cloned
}

func sourceSurfaceDescription(surface source.Surface) string {
	switch surface {
	case source.SurfaceQuery:
		return "querying"
	case source.SurfaceGraph:
		return "graph views"
	case source.SurfaceChat:
		return "chat"
	case source.SurfaceBrowser:
		return "browsing"
	default:
		return strings.ToLower(string(surface))
	}
}

func sourceActionDescription(action source.Action) string {
	switch action {
	case source.ActionBrowse:
		return "browsing"
	case source.ActionInspect:
		return "inspecting objects"
	case source.ActionViewRows:
		return "viewing rows"
	case source.ActionViewContent:
		return "viewing content"
	case source.ActionViewDefinition:
		return "viewing definitions"
	case source.ActionCreateChild:
		return "creating child objects"
	case source.ActionDelete:
		return "deleting objects"
	case source.ActionInsertData:
		return "inserting data"
	case source.ActionUpdateData:
		return "updating data"
	case source.ActionDeleteData:
		return "deleting data"
	case source.ActionImportData:
		return "importing data"
	case source.ActionGenerateMockData:
		return "generating mock data"
	case source.ActionExecute:
		return "executing actions"
	case source.ActionViewGraph:
		return "viewing graphs"
	default:
		return strings.ToLower(string(action))
	}
}

func appendPath(parent *source.ObjectRef, name string) []string {
	if parent == nil {
		return []string{name}
	}
	path := slices.Clone(parent.Path)
	path = append(path, name)
	return path
}

func parentForRef(ref source.ObjectRef) *source.ObjectRef {
	if len(ref.Path) == 0 {
		return nil
	}

	if len(ref.Path) == 1 {
		return nil
	}

	parent := source.NewObjectRef(ref.Kind, ref.Path[:len(ref.Path)-1])
	return &parent
}

func objectName(ref source.ObjectRef) string {
	if len(ref.Path) == 0 {
		return ""
	}
	return ref.Path[len(ref.Path)-1]
}

func cloneImportRows(rows [][]string) [][]string {
	cloned := make([][]string, 0, len(rows))
	for _, row := range rows {
		cloned = append(cloned, slices.Clone(row))
	}
	return cloned
}

func cloneSourceColumns(columns []source.Column) []source.Column {
	if columns == nil {
		return nil
	}

	cloned := make([]source.Column, 0, len(columns))
	for _, column := range columns {
		cloned = append(cloned, source.Column{
			Type:             column.Type,
			Name:             column.Name,
			IsNullable:       column.IsNullable,
			IsPrimary:        column.IsPrimary,
			IsAutoIncrement:  column.IsAutoIncrement,
			IsComputed:       column.IsComputed,
			IsForeignKey:     column.IsForeignKey,
			ReferencedTable:  cloneStringPointer(column.ReferencedTable),
			ReferencedColumn: cloneStringPointer(column.ReferencedColumn),
			Length:           cloneIntPointer(column.Length),
			Precision:        cloneIntPointer(column.Precision),
			Scale:            cloneIntPointer(column.Scale),
		})
	}

	return cloned
}

func cloneStringPointer(value *string) *string {
	if value == nil {
		return nil
	}

	cloned := *value
	return &cloned
}

func cloneIntPointer(value *int) *int {
	if value == nil {
		return nil
	}

	cloned := *value
	return &cloned
}

func importerColumnMappings(mappings []source.ImportColumnMapping) []importer.ColumnMapping {
	if len(mappings) == 0 {
		return nil
	}

	converted := make([]importer.ColumnMapping, 0, len(mappings))
	for _, mapping := range mappings {
		converted = append(converted, importer.ColumnMapping{
			SourceColumn: mapping.SourceColumn,
			TargetColumn: mapping.TargetColumn,
			Skip:         mapping.Skip,
		})
	}
	return converted
}
