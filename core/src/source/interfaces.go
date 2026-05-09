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

	"github.com/clidey/whodb/core/src/query"
)

// SourceSession is the common session interface exposed by a source connector.
type SourceSession interface {
	// Metadata returns session-scoped metadata for editors and query builders.
	Metadata(ctx context.Context) (*SessionMetadata, error)
}

// SourceConnector opens sessions for one source type.
type SourceConnector interface {
	// Open creates a new session for the provided source type and credentials.
	Open(ctx context.Context, spec TypeSpec, credentials *Credentials) (SourceSession, error)
}

// SessionInvalidator clears cached runtime state associated with one source
// type and credential set.
type SessionInvalidator interface {
	// Invalidate clears any cached connections or sessions associated with the
	// provided source type and credentials.
	Invalidate(ctx context.Context, spec TypeSpec, credentials *Credentials) error
}

// DriverShutdowner releases any process-wide runtime state owned by one source
// driver.
type DriverShutdowner interface {
	// Shutdown releases all cached state owned by the source driver.
	Shutdown(ctx context.Context) error
}

// SourceBrowser lists and resolves browseable objects.
type SourceBrowser interface {
	// ListObjects lists objects beneath the provided parent.
	ListObjects(ctx context.Context, parent *ObjectRef, kinds []ObjectKind) ([]Object, error)
	// GetObject loads one object by reference.
	GetObject(ctx context.Context, ref ObjectRef) (*Object, error)
}

// PaginatedSourceBrowser lists browseable objects with an offset and page size.
type PaginatedSourceBrowser interface {
	// ListObjectsPage lists one page of objects beneath the provided parent.
	ListObjectsPage(ctx context.Context, parent *ObjectRef, kinds []ObjectKind, pageSize int, pageOffset int) ([]Object, error)
}

// TabularReader reads row/column data from a source object.
type TabularReader interface {
	// ReadRows returns tabular rows for the provided object reference.
	ReadRows(ctx context.Context, ref ObjectRef, where *query.WhereCondition, sort []*query.SortCondition, pageSize int, pageOffset int) (*RowsResult, error)
	// Columns returns columns for one object.
	Columns(ctx context.Context, ref ObjectRef) ([]Column, error)
	// ColumnsBatch returns columns for multiple objects.
	ColumnsBatch(ctx context.Context, refs []ObjectRef) ([]ObjectColumns, error)
}

// ColumnConstraintReader loads per-column constraint metadata for one source
// object.
type ColumnConstraintReader interface {
	// ColumnConstraints returns database-specific column constraints such as
	// uniqueness, defaults, and check values for one object.
	ColumnConstraints(ctx context.Context, ref ObjectRef) (map[string]map[string]any, error)
}

// ContentReader reads blob/text content from a source object.
type ContentReader interface {
	// ReadContent returns a content payload for the provided object reference.
	ReadContent(ctx context.Context, ref ObjectRef) (*ContentResult, error)
}

// ContentDownloader streams source object content for download.
type ContentDownloader interface {
	// DownloadContent returns a streaming payload for the provided object reference.
	DownloadContent(ctx context.Context, ref ObjectRef) (*ContentDownload, error)
}

// ContentUploader streams source object content into a source.
type ContentUploader interface {
	// UploadContent uploads content beneath a parent source object.
	UploadContent(ctx context.Context, parent *ObjectRef, name string, reader io.Reader, contentType string, sizeBytes int64) (bool, error)
	// ReplaceContent replaces content for the provided object reference, using
	// name as the desired leaf object name when supplied.
	ReplaceContent(ctx context.Context, ref ObjectRef, name string, reader io.Reader, contentType string, sizeBytes int64) (bool, error)
}

// AvailabilityChecker verifies that a source session can reach the underlying
// system with the current credentials.
type AvailabilityChecker interface {
	// IsAvailable reports whether the active source session is usable.
	IsAvailable(ctx context.Context) bool
}

// QueryRunner executes source-native queries.
type QueryRunner interface {
	// RunQuery executes a query against the active source session.
	RunQuery(ctx context.Context, query string, params ...any) (*RowsResult, error)
}

// QueryStreamWriter receives streamed query output row by row.
type QueryStreamWriter interface {
	// WriteColumns writes the result columns once before any rows.
	WriteColumns(columns []Column) error
	// WriteRow writes one streamed row.
	WriteRow(row []string) error
}

// StreamQueryRunner executes source-native queries through a streaming result
// path when the active source supports it.
type StreamQueryRunner interface {
	// RunQueryStream executes a query and streams the result rows through the
	// supplied writer.
	RunQueryStream(ctx context.Context, query string, writer QueryStreamWriter, params ...any) error
}

// ScriptRunner executes source-native scripts that may require special runtime
// options such as multi-statement support.
type ScriptRunner interface {
	// RunScript executes one script with the requested execution options.
	RunScript(ctx context.Context, script string, multiStatement bool, params ...any) (*RowsResult, error)
}

// GraphReader reads graph data for a source scope.
type GraphReader interface {
	// ReadGraph returns graph units for the provided scope reference, or the
	// default source graph when ref is nil.
	ReadGraph(ctx context.Context, ref *ObjectRef) ([]GraphUnit, error)
}

// SourceAssistant runs AI chat against a source scope.
type SourceAssistant interface {
	// Reply runs the source assistant against the provided scope.
	Reply(ctx context.Context, ref *ObjectRef, previousConversation string, query string) ([]*ChatMessage, error)
}

// ModelAwareSourceAssistant runs AI chat with an explicitly selected external
// model configuration.
type ModelAwareSourceAssistant interface {
	// ReplyWithModel runs the source assistant against the provided scope using
	// the supplied model configuration.
	ReplyWithModel(ctx context.Context, ref *ObjectRef, previousConversation string, query string, model *ExternalModel) ([]*ChatMessage, error)
}

// ObjectManager mutates source objects and row data.
type ObjectManager interface {
	// CreateObject creates a new object beneath the provided parent.
	CreateObject(ctx context.Context, parent *ObjectRef, name string, fields []Record) (bool, error)
	// UpdateObject updates data within an existing object.
	UpdateObject(ctx context.Context, ref ObjectRef, values map[string]string, updatedColumns []string) (bool, error)
	// AddRow inserts a row/document into an object.
	AddRow(ctx context.Context, ref ObjectRef, values []Record) (bool, error)
	// DeleteRow deletes a row/document from an object.
	DeleteRow(ctx context.Context, ref ObjectRef, values map[string]string) (bool, error)
}

// ConnectionFieldOptionsReader loads dynamic options for a connection field.
type ConnectionFieldOptionsReader interface {
	// ConnectionFieldOptions returns selectable values for a connection field.
	ConnectionFieldOptions(ctx context.Context, fieldKey string, values map[string]string) ([]string, error)
}

// TabularExporter streams rows for one object into a caller-provided writer.
type TabularExporter interface {
	// ExportRows writes rows for the provided object reference.
	ExportRows(ctx context.Context, ref ObjectRef, writer func([]string) error, selectedRows []map[string]any) error
}

// NDJSONExporter streams rows for one object as newline-delimited JSON.
type NDJSONExporter interface {
	// ExportRowsNDJSON writes NDJSON rows for the provided object reference.
	ExportRowsNDJSON(ctx context.Context, ref ObjectRef, writer func(string) error, selectedRows []map[string]any) error
}

// SecurityReader exposes connection security metadata for the active source
// session.
type SecurityReader interface {
	// SSLStatus returns the current SSL/TLS status, or nil when it does not
	// apply to the active source.
	SSLStatus(ctx context.Context) (*SSLStatus, error)
}

// DataImporter applies parsed tabular data to a destination source object.
type DataImporter interface {
	// ImportData imports parsed rows into the provided object reference.
	ImportData(ctx context.Context, ref ObjectRef, request ImportRequest) (*ImportResult, error)
}

// MockDataManager handles mock-data planning and generation for supported
// source objects.
type MockDataManager interface {
	// GenerateMockData creates synthetic rows/documents for the provided object.
	GenerateMockData(ctx context.Context, ref ObjectRef, rowCount int, fkDensityRatio int, overwriteExisting bool) (*MockDataGenerationResult, error)
	// AnalyzeMockDataDependencies returns the dependency order required to
	// generate mock data for the provided object.
	AnalyzeMockDataDependencies(ctx context.Context, ref ObjectRef, rowCount int, fkDensityRatio int) (*MockDataDependencyAnalysis, error)
}

// QuerySuggester returns source-scoped suggestions for the query UI.
type QuerySuggester interface {
	// QuerySuggestions returns suggested prompts for the current source scope.
	QuerySuggestions(ctx context.Context, ref *ObjectRef) ([]QuerySuggestion, error)
}
