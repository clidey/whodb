/*
 * Copyright 2025 Clidey, Inc.
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
	Results *engine.GetRowsResult
	Err     error
}

// TablesLoadedMsg is sent when tables are loaded in browser view with timeout support
type TablesLoadedMsg struct {
	Tables  []engine.StorageUnit
	Schemas []string
	Err     error
}

// AutocompleteDebounceMsg is sent after a debounce delay to trigger autocomplete.
// The SeqID is compared to the current sequence to determine if the message is stale.
type AutocompleteDebounceMsg struct {
	SeqID int
	Text  string
	Pos   int
}
