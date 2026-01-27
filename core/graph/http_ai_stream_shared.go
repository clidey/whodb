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
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
)

// StreamRequest represents the incoming SSE request
type StreamRequest struct {
	ModelType string          `json:"modelType"`
	Token     string          `json:"token"`
	Model     string          `json:"model"`
	Endpoint  string          `json:"endpoint"`
	Schema    string          `json:"schema"`
	Input     model.ChatInput `json:"input"`
}

// StreamContext contains all context needed for streaming
type StreamContext struct {
	Writer   http.ResponseWriter
	Flusher  http.Flusher
	Plugin   *engine.Plugin
	Config   *engine.PluginConfig
	Request  *StreamRequest
}

// ParseStreamRequest parses and validates the SSE request
func ParseStreamRequest(r *http.Request) (*StreamRequest, error) {
	var req StreamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}
	return &req, nil
}

// SetupSSEHeaders configures the response for Server-Sent Events
func SetupSSEHeaders(w http.ResponseWriter) http.Flusher {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, _ := w.(http.Flusher)
	return flusher
}

// BuildTableDetails builds the table schema string for BAML context
func BuildTableDetails(plugin *engine.Plugin, config *engine.PluginConfig, schema string) (string, error) {
	storageUnits, err := plugin.GetStorageUnits(config, schema)
	if err != nil {
		return "", err
	}

	tableDetails := ""
	for _, unit := range storageUnits {
		columns, err := plugin.GetColumnsForTable(config, schema, unit.Name)
		if err != nil {
			log.Logger.WithError(err).Warnf("Failed to get columns for table %s in streaming chat", unit.Name)
			continue
		}

		tableDetails += fmt.Sprintf("table: %s\n", unit.Name)
		for _, col := range columns {
			tableDetails += fmt.Sprintf("- %s (%s)\n", col.Name, col.Type)
		}
	}
	return tableDetails, nil
}

// SendSSEMessage sends a complete message via SSE
func SendSSEMessage(w http.ResponseWriter, flusher http.Flusher, message *model.AIChatMessage) {
	data, err := json.Marshal(message)
	if err != nil {
		log.Logger.WithError(err).Error("Failed to marshal SSE message")
		return
	}
	fmt.Fprintf(w, "event: message\ndata: %s\n\n", data)
	flusher.Flush()
}

// SendSSEChunk sends a streaming chunk via SSE
func SendSSEChunk(w http.ResponseWriter, flusher http.Flusher, chunk map[string]any) {
	data, err := json.Marshal(chunk)
	if err != nil {
		log.Logger.WithError(err).Error("Failed to marshal SSE chunk")
		return
	}
	fmt.Fprintf(w, "event: chunk\ndata: %s\n\n", data)
	flusher.Flush()
}

// SendSSEError sends an error via SSE
func SendSSEError(w http.ResponseWriter, flusher http.Flusher, errorMsg string) {
	errorData := map[string]string{"error": errorMsg}
	data, _ := json.Marshal(errorData)
	fmt.Fprintf(w, "event: error\ndata: %s\n\n", data)
	flusher.Flush()
}

// SendSSEDone sends the done event
func SendSSEDone(w http.ResponseWriter, flusher http.Flusher) {
	fmt.Fprintf(w, "event: done\ndata: {}\n\n")
	flusher.Flush()
}

// ConvertResultToMessage converts engine result to API message
func ConvertResultToMessage(result *engine.GetRowsResult) *model.RowsResult {
	if result == nil {
		return nil
	}

	var columns []*model.Column
	for _, column := range result.Columns {
		columns = append(columns, &model.Column{
			Type: column.Type,
			Name: column.Name,
		})
	}

	return &model.RowsResult{
		Columns: columns,
		Rows:    result.Rows,
	}
}
