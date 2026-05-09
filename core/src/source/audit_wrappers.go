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

package source

import (
	"context"
	"io"
	"strings"
	"time"

	coreaudit "github.com/clidey/whodb/core/src/audit"
	"github.com/clidey/whodb/core/src/query"
)

// AuditScope describes the source resource associated with one session.
type AuditScope struct {
	TypeID     string
	ResourceID string
}

// AuditScopeFromCredentials builds a session audit scope from a type spec and credentials.
func AuditScopeFromCredentials(spec TypeSpec, credentials *Credentials) AuditScope {
	scope := AuditScope{TypeID: strings.TrimSpace(spec.ID)}
	if credentials == nil || credentials.ID == nil {
		return scope
	}
	scope.ResourceID = strings.TrimSpace(*credentials.ID)
	return scope
}

// AuditScopeWithResourceID builds a session audit scope with an explicit resource id.
func AuditScopeWithResourceID(spec TypeSpec, resourceID string) AuditScope {
	return AuditScope{
		TypeID:     strings.TrimSpace(spec.ID),
		ResourceID: strings.TrimSpace(resourceID),
	}
}

func (s AuditScope) resource() coreaudit.Resource {
	resourceID := s.ResourceID
	if resourceID == "" {
		resourceID = s.TypeID
	}
	return coreaudit.Resource{
		ID:   resourceID,
		Type: "source",
		Name: s.TypeID,
	}
}

func (s AuditScope) record(ctx context.Context, action string, start time.Time, err error, details map[string]any) {
	recordDetails := cloneAuditDetails(details)
	if err != nil {
		recordDetails["error"] = err.Error()
	}

	event := coreaudit.AuditEvent{
		Timestamp: start,
		Action:    action,
		Resource:  s.resource(),
		Details:   recordDetails,
		Duration:  time.Since(start),
	}
	if err != nil {
		event.Error = err.Error()
		event.Outcome = coreaudit.OutcomeFailure
		event.Severity = coreaudit.SeverityWarn
	}

	coreaudit.RecordWithContext(ctx, event)
}

// AsAvailabilityChecker returns an audited availability checker when supported.
func AsAvailabilityChecker(scope AuditScope, session SourceSession) (AvailabilityChecker, bool) {
	checker, ok := session.(AvailabilityChecker)
	if !ok {
		return nil, false
	}
	return auditedAvailabilityChecker{scope: scope, next: checker}, true
}

// AsSourceBrowser returns an audited source browser when supported.
func AsSourceBrowser(scope AuditScope, session SourceSession) (SourceBrowser, bool) {
	browser, ok := session.(SourceBrowser)
	if !ok {
		return nil, false
	}
	return auditedSourceBrowser{scope: scope, next: browser}, true
}

// AsPaginatedSourceBrowser returns an audited paginated browser when supported.
func AsPaginatedSourceBrowser(scope AuditScope, session SourceSession) (PaginatedSourceBrowser, bool) {
	browser, ok := session.(PaginatedSourceBrowser)
	if !ok {
		return nil, false
	}
	return auditedPaginatedSourceBrowser{scope: scope, next: browser}, true
}

// AsTabularReader returns an audited tabular reader when supported.
func AsTabularReader(scope AuditScope, session SourceSession) (TabularReader, bool) {
	reader, ok := session.(TabularReader)
	if !ok {
		return nil, false
	}
	return auditedTabularReader{scope: scope, next: reader}, true
}

// AsContentReader returns an audited content reader when supported.
func AsContentReader(scope AuditScope, session SourceSession) (ContentReader, bool) {
	reader, ok := session.(ContentReader)
	if !ok {
		return nil, false
	}
	return auditedContentReader{scope: scope, next: reader}, true
}

// AsContentDownloader returns an audited content downloader when supported.
func AsContentDownloader(scope AuditScope, session SourceSession) (ContentDownloader, bool) {
	downloader, ok := session.(ContentDownloader)
	if !ok {
		return nil, false
	}
	return auditedContentDownloader{scope: scope, next: downloader}, true
}

// AsContentUploader returns an audited content uploader when supported.
func AsContentUploader(scope AuditScope, session SourceSession) (ContentUploader, bool) {
	uploader, ok := session.(ContentUploader)
	if !ok {
		return nil, false
	}
	return auditedContentUploader{scope: scope, next: uploader}, true
}

// AsQueryRunner returns an audited query runner when supported.
func AsQueryRunner(scope AuditScope, session SourceSession) (QueryRunner, bool) {
	runner, ok := session.(QueryRunner)
	if !ok {
		return nil, false
	}
	return auditedQueryRunner{scope: scope, next: runner}, true
}

// AsScriptRunner returns an audited script runner when supported.
func AsScriptRunner(scope AuditScope, session SourceSession) (ScriptRunner, bool) {
	runner, ok := session.(ScriptRunner)
	if !ok {
		return nil, false
	}
	return auditedScriptRunner{scope: scope, next: runner}, true
}

// AsGraphReader returns an audited graph reader when supported.
func AsGraphReader(scope AuditScope, session SourceSession) (GraphReader, bool) {
	reader, ok := session.(GraphReader)
	if !ok {
		return nil, false
	}
	return auditedGraphReader{scope: scope, next: reader}, true
}

// AsSourceAssistant returns an audited source assistant when supported.
func AsSourceAssistant(scope AuditScope, session SourceSession) (SourceAssistant, bool) {
	assistant, ok := session.(SourceAssistant)
	if !ok {
		return nil, false
	}
	return auditedSourceAssistant{scope: scope, next: assistant}, true
}

// AsModelAwareSourceAssistant returns an audited model-aware source assistant when supported.
func AsModelAwareSourceAssistant(scope AuditScope, session SourceSession) (ModelAwareSourceAssistant, bool) {
	assistant, ok := session.(ModelAwareSourceAssistant)
	if !ok {
		return nil, false
	}
	return auditedModelAwareSourceAssistant{scope: scope, next: assistant}, true
}

// AsObjectManager returns an audited object manager when supported.
func AsObjectManager(scope AuditScope, session SourceSession) (ObjectManager, bool) {
	manager, ok := session.(ObjectManager)
	if !ok {
		return nil, false
	}
	return auditedObjectManager{scope: scope, next: manager}, true
}

// AsConnectionFieldOptionsReader returns an audited field-options reader when supported.
func AsConnectionFieldOptionsReader(scope AuditScope, session SourceSession) (ConnectionFieldOptionsReader, bool) {
	reader, ok := session.(ConnectionFieldOptionsReader)
	if !ok {
		return nil, false
	}
	return auditedConnectionFieldOptionsReader{scope: scope, next: reader}, true
}

// AsTabularExporter returns an audited tabular exporter when supported.
func AsTabularExporter(scope AuditScope, session SourceSession) (TabularExporter, bool) {
	exporter, ok := session.(TabularExporter)
	if !ok {
		return nil, false
	}
	return auditedTabularExporter{scope: scope, next: exporter}, true
}

// AsNDJSONExporter returns an audited NDJSON exporter when supported.
func AsNDJSONExporter(scope AuditScope, session SourceSession) (NDJSONExporter, bool) {
	exporter, ok := session.(NDJSONExporter)
	if !ok {
		return nil, false
	}
	return auditedNDJSONExporter{scope: scope, next: exporter}, true
}

// AsSecurityReader returns an audited security reader when supported.
func AsSecurityReader(scope AuditScope, session SourceSession) (SecurityReader, bool) {
	reader, ok := session.(SecurityReader)
	if !ok {
		return nil, false
	}
	return auditedSecurityReader{scope: scope, next: reader}, true
}

// AsDataImporter returns an audited data importer when supported.
func AsDataImporter(scope AuditScope, session SourceSession) (DataImporter, bool) {
	importer, ok := session.(DataImporter)
	if !ok {
		return nil, false
	}
	return auditedDataImporter{scope: scope, next: importer}, true
}

// AsMockDataManager returns an audited mock-data manager when supported.
func AsMockDataManager(scope AuditScope, session SourceSession) (MockDataManager, bool) {
	manager, ok := session.(MockDataManager)
	if !ok {
		return nil, false
	}
	return auditedMockDataManager{scope: scope, next: manager}, true
}

// AsQuerySuggester returns an audited query suggester when supported.
func AsQuerySuggester(scope AuditScope, session SourceSession) (QuerySuggester, bool) {
	suggester, ok := session.(QuerySuggester)
	if !ok {
		return nil, false
	}
	return auditedQuerySuggester{scope: scope, next: suggester}, true
}

type auditedAvailabilityChecker struct {
	scope AuditScope
	next  AvailabilityChecker
}

func (a auditedAvailabilityChecker) IsAvailable(ctx context.Context) bool {
	start := time.Now()
	available := a.next.IsAvailable(ctx)
	event := coreaudit.AuditEvent{
		Timestamp: start,
		Action:    "source.check_availability",
		Resource:  a.scope.resource(),
		Details: map[string]any{
			"available": available,
		},
		Duration: time.Since(start),
	}
	if !available {
		event.Outcome = coreaudit.OutcomeFailure
		event.Severity = coreaudit.SeverityWarn
	}
	coreaudit.RecordWithContext(ctx, event)
	return available
}

type auditedSourceBrowser struct {
	scope AuditScope
	next  SourceBrowser
}

func (a auditedSourceBrowser) ListObjects(ctx context.Context, parent *ObjectRef, kinds []ObjectKind) ([]Object, error) {
	start := time.Now()
	objects, err := a.next.ListObjects(ctx, parent, kinds)
	details := objectRefDetails(parent)
	details["kind_filters"] = kindStrings(kinds)
	details["object_count"] = len(objects)
	a.scope.record(ctx, "source.list_objects", start, err, details)
	return objects, err
}

func (a auditedSourceBrowser) GetObject(ctx context.Context, ref ObjectRef) (*Object, error) {
	start := time.Now()
	object, err := a.next.GetObject(ctx, ref)
	details := objectRefDetails(&ref)
	if object != nil {
		details["object_name"] = object.Name
	}
	a.scope.record(ctx, "source.get_object", start, err, details)
	return object, err
}

type auditedPaginatedSourceBrowser struct {
	scope AuditScope
	next  PaginatedSourceBrowser
}

func (a auditedPaginatedSourceBrowser) ListObjectsPage(ctx context.Context, parent *ObjectRef, kinds []ObjectKind, pageSize int, pageOffset int) ([]Object, error) {
	start := time.Now()
	objects, err := a.next.ListObjectsPage(ctx, parent, kinds, pageSize, pageOffset)
	details := objectRefDetails(parent)
	details["kind_filters"] = kindStrings(kinds)
	details["page_size"] = pageSize
	details["page_offset"] = pageOffset
	details["object_count"] = len(objects)
	a.scope.record(ctx, "source.list_objects_page", start, err, details)
	return objects, err
}

type auditedTabularReader struct {
	scope AuditScope
	next  TabularReader
}

func (a auditedTabularReader) ReadRows(ctx context.Context, ref ObjectRef, where *query.WhereCondition, sort []*query.SortCondition, pageSize int, pageOffset int) (*RowsResult, error) {
	start := time.Now()
	result, err := a.next.ReadRows(ctx, ref, where, sort, pageSize, pageOffset)
	details := objectRefDetails(&ref)
	details["page_size"] = pageSize
	details["page_offset"] = pageOffset
	details["has_where"] = where != nil
	details["sort_count"] = len(sort)
	if result != nil {
		details["row_count"] = len(result.Rows)
		details["column_count"] = len(result.Columns)
		details["total_count"] = result.TotalCount
	}
	a.scope.record(ctx, "source.read_rows", start, err, details)
	return result, err
}

func (a auditedTabularReader) Columns(ctx context.Context, ref ObjectRef) ([]Column, error) {
	start := time.Now()
	columns, err := a.next.Columns(ctx, ref)
	details := objectRefDetails(&ref)
	details["column_count"] = len(columns)
	a.scope.record(ctx, "source.columns", start, err, details)
	return columns, err
}

func (a auditedTabularReader) ColumnsBatch(ctx context.Context, refs []ObjectRef) ([]ObjectColumns, error) {
	start := time.Now()
	columns, err := a.next.ColumnsBatch(ctx, refs)
	details := map[string]any{
		"ref_count":    len(refs),
		"result_count": len(columns),
	}
	a.scope.record(ctx, "source.columns_batch", start, err, details)
	return columns, err
}

type auditedContentReader struct {
	scope AuditScope
	next  ContentReader
}

func (a auditedContentReader) ReadContent(ctx context.Context, ref ObjectRef) (*ContentResult, error) {
	start := time.Now()
	result, err := a.next.ReadContent(ctx, ref)
	details := objectRefDetails(&ref)
	if result != nil {
		details["mime_type"] = result.MIMEType
		details["size_bytes"] = result.SizeBytes
		details["is_binary"] = result.IsBinary
		details["truncated"] = result.Truncated
	}
	a.scope.record(ctx, "source.read_content", start, err, details)
	return result, err
}

type auditedContentDownloader struct {
	scope AuditScope
	next  ContentDownloader
}

func (a auditedContentDownloader) DownloadContent(ctx context.Context, ref ObjectRef) (*ContentDownload, error) {
	start := time.Now()
	result, err := a.next.DownloadContent(ctx, ref)
	details := objectRefDetails(&ref)
	if result != nil {
		details["mime_type"] = result.MIMEType
		details["size_bytes"] = result.SizeBytes
		details["file_name"] = result.FileName
	}
	a.scope.record(ctx, "source.download_content", start, err, details)
	return result, err
}

type auditedContentUploader struct {
	scope AuditScope
	next  ContentUploader
}

func (a auditedContentUploader) UploadContent(ctx context.Context, parent *ObjectRef, name string, reader io.Reader, contentType string, sizeBytes int64) (bool, error) {
	start := time.Now()
	status, err := a.next.UploadContent(ctx, parent, name, reader, contentType, sizeBytes)
	details := objectRefDetails(parent)
	details["name"] = strings.TrimSpace(name)
	details["content_type"] = strings.TrimSpace(contentType)
	details["size_bytes"] = sizeBytes
	details["status"] = status
	a.scope.record(ctx, "source.upload_content", start, err, details)
	return status, err
}

func (a auditedContentUploader) ReplaceContent(ctx context.Context, ref ObjectRef, name string, reader io.Reader, contentType string, sizeBytes int64) (bool, error) {
	start := time.Now()
	status, err := a.next.ReplaceContent(ctx, ref, name, reader, contentType, sizeBytes)
	details := objectRefDetails(&ref)
	details["name"] = strings.TrimSpace(name)
	details["content_type"] = strings.TrimSpace(contentType)
	details["size_bytes"] = sizeBytes
	details["status"] = status
	a.scope.record(ctx, "source.replace_content", start, err, details)
	return status, err
}

type auditedQueryRunner struct {
	scope AuditScope
	next  QueryRunner
}

func (a auditedQueryRunner) RunQuery(ctx context.Context, queryText string, params ...any) (*RowsResult, error) {
	start := time.Now()
	result, err := a.next.RunQuery(ctx, queryText, params...)
	details := map[string]any{
		"query_operation": coreaudit.QueryOperation(queryText),
		"query_length":    len(strings.TrimSpace(queryText)),
		"param_count":     len(params),
	}
	if result != nil {
		details["row_count"] = len(result.Rows)
		details["column_count"] = len(result.Columns)
		details["total_count"] = result.TotalCount
	}
	a.scope.record(ctx, "source.run_query", start, err, details)
	return result, err
}

type auditedScriptRunner struct {
	scope AuditScope
	next  ScriptRunner
}

func (a auditedScriptRunner) RunScript(ctx context.Context, script string, multiStatement bool, params ...any) (*RowsResult, error) {
	start := time.Now()
	result, err := a.next.RunScript(ctx, script, multiStatement, params...)
	details := map[string]any{
		"query_operation": coreaudit.QueryOperation(script),
		"query_length":    len(strings.TrimSpace(script)),
		"param_count":     len(params),
		"multi_statement": multiStatement,
	}
	if result != nil {
		details["row_count"] = len(result.Rows)
		details["column_count"] = len(result.Columns)
		details["total_count"] = result.TotalCount
	}
	a.scope.record(ctx, "source.run_script", start, err, details)
	return result, err
}

type auditedGraphReader struct {
	scope AuditScope
	next  GraphReader
}

func (a auditedGraphReader) ReadGraph(ctx context.Context, ref *ObjectRef) ([]GraphUnit, error) {
	start := time.Now()
	units, err := a.next.ReadGraph(ctx, ref)
	details := objectRefDetails(ref)
	details["unit_count"] = len(units)
	a.scope.record(ctx, "source.read_graph", start, err, details)
	return units, err
}

type auditedSourceAssistant struct {
	scope AuditScope
	next  SourceAssistant
}

func (a auditedSourceAssistant) Reply(ctx context.Context, ref *ObjectRef, previousConversation string, query string) ([]*ChatMessage, error) {
	start := time.Now()
	messages, err := a.next.Reply(ctx, ref, previousConversation, query)
	details := objectRefDetails(ref)
	details["query_length"] = len(strings.TrimSpace(query))
	details["has_previous_conversation"] = strings.TrimSpace(previousConversation) != ""
	details["message_count"] = len(messages)
	a.scope.record(ctx, "source.reply", start, err, details)
	return messages, err
}

type auditedModelAwareSourceAssistant struct {
	scope AuditScope
	next  ModelAwareSourceAssistant
}

func (a auditedModelAwareSourceAssistant) ReplyWithModel(ctx context.Context, ref *ObjectRef, previousConversation string, query string, model *ExternalModel) ([]*ChatMessage, error) {
	start := time.Now()
	messages, err := a.next.ReplyWithModel(ctx, ref, previousConversation, query, model)
	details := objectRefDetails(ref)
	details["query_length"] = len(strings.TrimSpace(query))
	details["has_previous_conversation"] = strings.TrimSpace(previousConversation) != ""
	details["message_count"] = len(messages)
	if model != nil {
		details["model_type"] = strings.TrimSpace(model.Type)
		details["model_name"] = strings.TrimSpace(model.Model)
		details["has_endpoint"] = strings.TrimSpace(model.Endpoint) != ""
	}
	a.scope.record(ctx, "source.reply_with_model", start, err, details)
	return messages, err
}

type auditedObjectManager struct {
	scope AuditScope
	next  ObjectManager
}

func (a auditedObjectManager) CreateObject(ctx context.Context, parent *ObjectRef, name string, fields []Record) (bool, error) {
	start := time.Now()
	status, err := a.next.CreateObject(ctx, parent, name, fields)
	details := objectRefDetails(parent)
	details["name"] = strings.TrimSpace(name)
	details["field_count"] = len(fields)
	details["status"] = status
	a.scope.record(ctx, "source.create_object", start, err, details)
	return status, err
}

func (a auditedObjectManager) UpdateObject(ctx context.Context, ref ObjectRef, values map[string]string, updatedColumns []string) (bool, error) {
	start := time.Now()
	status, err := a.next.UpdateObject(ctx, ref, values, updatedColumns)
	details := objectRefDetails(&ref)
	details["value_count"] = len(values)
	details["updated_column_count"] = len(updatedColumns)
	details["status"] = status
	a.scope.record(ctx, "source.update_object", start, err, details)
	return status, err
}

func (a auditedObjectManager) AddRow(ctx context.Context, ref ObjectRef, values []Record) (bool, error) {
	start := time.Now()
	status, err := a.next.AddRow(ctx, ref, values)
	details := objectRefDetails(&ref)
	details["value_count"] = len(values)
	details["status"] = status
	a.scope.record(ctx, "source.add_row", start, err, details)
	return status, err
}

func (a auditedObjectManager) DeleteRow(ctx context.Context, ref ObjectRef, values map[string]string) (bool, error) {
	start := time.Now()
	status, err := a.next.DeleteRow(ctx, ref, values)
	details := objectRefDetails(&ref)
	details["value_count"] = len(values)
	details["status"] = status
	a.scope.record(ctx, "source.delete_row", start, err, details)
	return status, err
}

type auditedConnectionFieldOptionsReader struct {
	scope AuditScope
	next  ConnectionFieldOptionsReader
}

func (a auditedConnectionFieldOptionsReader) ConnectionFieldOptions(ctx context.Context, fieldKey string, values map[string]string) ([]string, error) {
	start := time.Now()
	options, err := a.next.ConnectionFieldOptions(ctx, fieldKey, values)
	details := map[string]any{
		"field_key":    strings.TrimSpace(fieldKey),
		"value_count":  len(values),
		"option_count": len(options),
	}
	a.scope.record(ctx, "source.connection_field_options", start, err, details)
	return options, err
}

type auditedTabularExporter struct {
	scope AuditScope
	next  TabularExporter
}

func (a auditedTabularExporter) ExportRows(ctx context.Context, ref ObjectRef, writer func([]string) error, selectedRows []map[string]any) error {
	start := time.Now()
	err := a.next.ExportRows(ctx, ref, writer, selectedRows)
	details := objectRefDetails(&ref)
	details["selected_row_count"] = len(selectedRows)
	a.scope.record(ctx, "source.export_rows", start, err, details)
	return err
}

type auditedNDJSONExporter struct {
	scope AuditScope
	next  NDJSONExporter
}

func (a auditedNDJSONExporter) ExportRowsNDJSON(ctx context.Context, ref ObjectRef, writer func(string) error, selectedRows []map[string]any) error {
	start := time.Now()
	err := a.next.ExportRowsNDJSON(ctx, ref, writer, selectedRows)
	details := objectRefDetails(&ref)
	details["selected_row_count"] = len(selectedRows)
	a.scope.record(ctx, "source.export_rows_ndjson", start, err, details)
	return err
}

type auditedSecurityReader struct {
	scope AuditScope
	next  SecurityReader
}

func (a auditedSecurityReader) SSLStatus(ctx context.Context) (*SSLStatus, error) {
	start := time.Now()
	status, err := a.next.SSLStatus(ctx)
	details := map[string]any{}
	if status != nil {
		details["enabled"] = status.IsEnabled
		details["mode"] = status.Mode
	}
	a.scope.record(ctx, "source.ssl_status", start, err, details)
	return status, err
}

type auditedDataImporter struct {
	scope AuditScope
	next  DataImporter
}

func (a auditedDataImporter) ImportData(ctx context.Context, ref ObjectRef, request ImportRequest) (*ImportResult, error) {
	start := time.Now()
	result, err := a.next.ImportData(ctx, ref, request)
	details := objectRefDetails(&ref)
	details["mode"] = string(request.Mode)
	details["parsed_rows"] = len(request.Parsed.Rows)
	details["parsed_columns"] = len(request.Parsed.Columns)
	details["mapping_count"] = len(request.Mapping)
	if result != nil {
		details["rows_imported"] = result.RowsImported
	}
	a.scope.record(ctx, "source.import_data", start, err, details)
	return result, err
}

type auditedMockDataManager struct {
	scope AuditScope
	next  MockDataManager
}

func (a auditedMockDataManager) GenerateMockData(ctx context.Context, ref ObjectRef, rowCount int, fkDensityRatio int, overwriteExisting bool) (*MockDataGenerationResult, error) {
	start := time.Now()
	result, err := a.next.GenerateMockData(ctx, ref, rowCount, fkDensityRatio, overwriteExisting)
	details := objectRefDetails(&ref)
	details["row_count"] = rowCount
	details["fk_density_ratio"] = fkDensityRatio
	details["overwrite_existing"] = overwriteExisting
	if result != nil {
		details["generated_total"] = result.TotalGenerated
		details["warning_count"] = len(result.Warnings)
	}
	a.scope.record(ctx, "source.generate_mock_data", start, err, details)
	return result, err
}

func (a auditedMockDataManager) AnalyzeMockDataDependencies(ctx context.Context, ref ObjectRef, rowCount int, fkDensityRatio int) (*MockDataDependencyAnalysis, error) {
	start := time.Now()
	result, err := a.next.AnalyzeMockDataDependencies(ctx, ref, rowCount, fkDensityRatio)
	details := objectRefDetails(&ref)
	details["row_count"] = rowCount
	details["fk_density_ratio"] = fkDensityRatio
	if result != nil {
		details["table_count"] = len(result.Tables)
		details["warning_count"] = len(result.Warnings)
		details["has_error"] = strings.TrimSpace(result.Error) != ""
	}
	a.scope.record(ctx, "source.analyze_mock_data_dependencies", start, err, details)
	return result, err
}

type auditedQuerySuggester struct {
	scope AuditScope
	next  QuerySuggester
}

func (a auditedQuerySuggester) QuerySuggestions(ctx context.Context, ref *ObjectRef) ([]QuerySuggestion, error) {
	start := time.Now()
	suggestions, err := a.next.QuerySuggestions(ctx, ref)
	details := objectRefDetails(ref)
	details["suggestion_count"] = len(suggestions)
	a.scope.record(ctx, "source.query_suggestions", start, err, details)
	return suggestions, err
}

func cloneAuditDetails(details map[string]any) map[string]any {
	if len(details) == 0 {
		return map[string]any{}
	}

	cloned := make(map[string]any, len(details))
	for key, value := range details {
		cloned[key] = value
	}
	return cloned
}

func objectRefDetails(ref *ObjectRef) map[string]any {
	if ref == nil {
		return map[string]any{"has_ref": false}
	}

	path := make([]string, len(ref.Path))
	copy(path, ref.Path)
	return map[string]any{
		"has_ref": true,
		"kind":    string(ref.Kind),
		"locator": ref.Locator,
		"path":    path,
	}
}

func kindStrings(kinds []ObjectKind) []string {
	values := make([]string, 0, len(kinds))
	for _, kind := range kinds {
		values = append(values, string(kind))
	}
	return values
}
