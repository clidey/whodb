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

package tui

import (
	"time"

	"github.com/clidey/whodb/cli/internal/database"
	"github.com/clidey/whodb/cli/internal/schemadiff"
	"github.com/clidey/whodb/core/src/engine"
)

// OperationState tracks the state of async operations
type OperationState int

const (
	OperationIdle OperationState = iota
	OperationRunning
	OperationCancelling
)

// QueryExecutedMsg is sent when a query execution completes (success or error)
type QueryExecutedMsg struct {
	Result *engine.GetRowsResult
	Query  string
	Err    error
}

// QueryCancelledMsg is sent when a query is cancelled by the user
type QueryCancelledMsg struct {
	Query string
}

// QueryTimeoutMsg is sent when a query times out
type QueryTimeoutMsg struct {
	Query   string
	Timeout time.Duration
}

// HistoryQueryMsg is sent when a query from history is re-executed
type HistoryQueryMsg struct {
	Result *engine.GetRowsResult
	Query  string
	Err    error
}

// PageLoadedMsg is sent when a page of data is loaded in results view
type PageLoadedMsg struct {
	Results   *engine.GetRowsResult
	Err       error
	Schema    string
	TableName string
}

// AutocompleteDebounceMsg is sent after a debounce delay to trigger autocomplete.
// The SeqID is compared to the current sequence to determine if the message is stale.
type AutocompleteDebounceMsg struct {
	SeqID int
	Text  string
	Pos   int
}

// tablesLoadedMsg is sent when tables are loaded in browser view
type tablesLoadedMsg struct {
	tables  []engine.StorageUnit
	schemas []string
	schema  string
	err     error
}

// chatResponseMsg is sent when an AI chat response is received
type chatResponseMsg struct {
	messages []*database.ChatMessage
	query    string
	err      error
}

// chatStreamChunkMsg is sent for each streaming chunk during AI chat.
type chatStreamChunkMsg struct {
	text string // accumulated text so far
}

// chatStreamDoneMsg is sent when the AI chat stream completes.
type chatStreamDoneMsg struct {
	messages []*database.ChatMessage
	err      error
}

// modelsLoadedMsg is sent when AI models are loaded
type modelsLoadedMsg struct {
	models []string
	err    error
}

// connectionResultMsg is sent when a connection attempt completes
type connectionResultMsg struct {
	err           error
	statusMessage string
}

// dockerConnectionsLoadedMsg is sent when background Docker detection
// completes for the connection list.
type dockerConnectionsLoadedMsg struct {
	items []connectionItem
}

// cloudConnectionsLoadedMsg is sent when background cloud discovery completes
// for the connection list.
type cloudConnectionsLoadedMsg struct {
	items []connectionItem
}

// escTimeoutTickMsg is sent to tick the ESC quit confirmation timer
type escTimeoutTickMsg struct{}

// exportResultMsg is sent when an export operation completes
type exportResultMsg struct {
	success       bool
	err           error
	savedFilePath string
}

// schemaLoadedMsg is sent when the database schema is loaded
type schemaLoadedMsg struct {
	tables []tableWithColumns
	err    error
	schema string
}

// schemaTableColumnsLoadedMsg is sent when a schema-view table's columns have
// been loaded on demand.
type schemaTableColumnsLoadedMsg struct {
	tableName string
	columns   []engine.Column
	err       error
}

// statusMessageTimeoutMsg is sent to auto-dismiss transient status messages
type statusMessageTimeoutMsg struct{}

// escConfirmTimeoutMsg resets the Esc-to-disconnect confirmation after a timeout.
type escConfirmTimeoutMsg struct{}

// explainResultMsg is sent when an EXPLAIN query completes.
type explainResultMsg struct {
	query string
	plan  string
	err   error
}

// schemaDiffResultMsg is sent when a schema diff comparison completes.
type schemaDiffResultMsg struct {
	result *schemadiff.Result
	err    error
}

// tableWithColumns pairs a storage unit with its column metadata
type tableWithColumns struct {
	StorageUnit    engine.StorageUnit
	Columns        []engine.Column
	ColumnsLoaded  bool
	ColumnsLoading bool
}
