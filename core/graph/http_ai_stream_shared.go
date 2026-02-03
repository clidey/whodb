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
	"strings"

	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
)

// StreamRequest represents the incoming SSE request
type StreamRequest struct {
	ModelType  string          `json:"modelType"`
	Token      string          `json:"token"`
	Model      string          `json:"model"`
	Endpoint   string          `json:"endpoint"`
	Schema     string          `json:"schema"`
	ProviderId string          `json:"providerId"`
	Input      model.ChatInput `json:"input"`
}

// StreamContext contains all context needed for streaming
type StreamContext struct {
	Writer  http.ResponseWriter
	Flusher http.Flusher
	Plugin  *engine.Plugin
	Config  *engine.PluginConfig
	Request *StreamRequest
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

// SendSSEError sends an error via SSE and completes the stream
func SendSSEError(w http.ResponseWriter, flusher http.Flusher, errorMsg string) {
	// Sanitize error message to avoid leaking technical details
	sanitized := sanitizeErrorMessage(errorMsg)

	// For better UX, send errors as chat messages instead of error events
	// This makes them appear inline in the chat
	SendSSEMessage(w, flusher, &model.AIChatMessage{
		Type: "error",
		Text: sanitized,
	})

	// Send done event to stop loading spinner
	SendSSEDone(w, flusher)
}

// sanitizeErrorMessage removes technical details and JSON from error messages
func sanitizeErrorMessage(msg string) string {
	// Check for specific error patterns and return user-friendly messages

	// LLM Service errors
	if strings.Contains(msg, "LLM Failure:") || strings.Contains(msg, "ServiceError") {
		// Extract any meaningful message after the error type
		if strings.Contains(msg, "throttling") || strings.Contains(msg, "rate limit") {
			return "AI service is busy. Please try again in a moment."
		}
		if strings.Contains(msg, "access denied") || strings.Contains(msg, "AccessDenied") {
			return "Access denied. Please check your credentials."
		}
		if strings.Contains(msg, "validation") || strings.Contains(msg, "ValidationException") {
			return "Invalid request. Please check your model selection."
		}
		return "AI service error. Please try again."
	}

	// BAML/Network errors
	if strings.Contains(msg, "reqwest::Error") || strings.Contains(msg, "Failed to build request") {
		return "Unable to connect to AI service. Please check your configuration."
	}

	if strings.Contains(msg, "RelativeUrlWithoutBase") {
		return "AI service configuration error. Please contact support."
	}

	// Stream/connection errors
	if strings.Contains(msg, "Failed to start stream") || strings.Contains(msg, "Failed to stream") {
		return "Unable to query. Please try again."
	}

	// API key/auth errors
	if strings.Contains(msg, "API key") || strings.Contains(msg, "api_key") ||
	   strings.Contains(msg, "OPENAI_API_KEY") || strings.Contains(msg, "unauthorized") {
		return "AI service not configured. Please set up your API key."
	}

	// Model errors
	if strings.Contains(msg, "model") && (strings.Contains(msg, "not found") || strings.Contains(msg, "invalid")) {
		return "Selected model is not available. Please choose a different model."
	}

	// If error contains technical markers (JSON, stack traces, etc.), simplify
	if len(msg) > 150 || strings.Contains(msg, "{") || strings.Contains(msg, "---") ||
	   strings.Contains(msg, "Error {") || strings.Contains(msg, "Prompt:") {
		// Try to extract first meaningful sentence
		for _, prefix := range []string{"Failed to", "Unable to", "Error:", "error:"} {
			if idx := strings.Index(msg, prefix); idx >= 0 {
				remaining := msg[idx:]
				// Find end of sentence (period, colon, newline)
				for _, end := range []string{".", ":", "\n"} {
					if endIdx := strings.Index(remaining, end); endIdx > 10 && endIdx < 100 {
						return remaining[:endIdx] + "."
					}
				}
			}
		}
		// Couldn't extract meaningful message, return generic
		return "Unable to query. Please try again."
	}

	// Return original if it's short and doesn't contain technical details
	return msg
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
