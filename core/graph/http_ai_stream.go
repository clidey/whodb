//go:build !arm

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
	ctx "context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/clidey/whodb/core/baml_client"
	"github.com/clidey/whodb/core/baml_client/stream_types"
	"github.com/clidey/whodb/core/baml_client/types"
	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
)

// aiChatStreamHandler handles real-time streaming of AI chat responses using Server-Sent Events (SSE)
func aiChatStreamHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ModelType string          `json:"modelType"`
		Token     string          `json:"token"`
		Model     string          `json:"model"`
		Endpoint  string          `json:"endpoint"`
		Schema    string          `json:"schema"`
		Input     model.ChatInput `json:"input"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // Disable nginx buffering

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Get plugin and config from context
	plugin, config := GetPluginForContext(r.Context())

	// Set up external model configuration
	if req.ModelType != "" && req.Model != "" {
		config.ExternalModel = &engine.ExternalModel{
			Type:     req.ModelType,
			Token:    req.Token,
			Model:    req.Model,
			Endpoint: req.Endpoint,
		}
	}

	// Create dynamic BAML client and log request
	callOpts := common.SetupAIClientWithLogging(config.ExternalModel)

	// Get table names first
	storageUnits, err := plugin.GetStorageUnits(config, req.Schema)
	if err != nil {
		sendSSEError(w, flusher, fmt.Sprintf("Failed to get table info: %v", err))
		return
	}

	// Build table details string with actual column definitions
	tableDetails := ""
	for _, unit := range storageUnits {
		// Get actual column definitions for each table
		columns, err := plugin.GetColumnsForTable(config, req.Schema, unit.Name)
		if err != nil {
			log.Logger.WithError(err).Warnf("Failed to get columns for table %s in streaming chat", unit.Name)
			continue
		}

		tableDetails += fmt.Sprintf("table: %s\n", unit.Name)
		for _, col := range columns {
			tableDetails += fmt.Sprintf("- %s (%s)\n", col.Name, col.Type)
		}
	}

	// Build BAML context
	dbContext := types.DatabaseContext{
		Database_type:         config.Credentials.Type,
		Schema:                req.Schema,
		Tables_and_fields:     tableDetails,
		Previous_conversation: req.Input.PreviousConversation,
	}

	// Stream BAML responses
	streamCtx := ctx.Background()
	stream, err := baml_client.Stream.GenerateSQLQuery(streamCtx, dbContext, req.Input.Query, callOpts...)
	if err != nil {
		sendSSEError(w, flusher, fmt.Sprintf("Failed to start stream: %v", err))
		return
	}

	// Process stream chunks
	for chunk := range stream {
		if chunk.IsError {
			sendSSEError(w, flusher, chunk.Error.Error())
			return
		}

		if chunk.IsFinal {
			// Send final complete responses
			final := chunk.Final()
			if final != nil {
				log.Logger.Infof("Processing final chunk with %d responses", len(*final))
				for _, bamlResp := range *final {
					log.Logger.Infof("Final response type: %s, operation: %v, text_len: %d", bamlResp.Type, bamlResp.Operation, len(bamlResp.Text))
					// Only execute SQL for SQL-type responses
					if bamlResp.Type == types.ChatMessageTypeSQL {
						message := convertBAMLResponseToMessage(&bamlResp, config, plugin)
						log.Logger.Infof("Sending SQL message: type=%s, has_result=%v, requires_confirmation=%v", message.Type, message.Result != nil, message.RequiresConfirmation)
						sendSSEMessage(w, flusher, message)
					}
					// Message-type responses were already streamed
				}
			} else {
				log.Logger.Warn("Final chunk has no responses")
			}
			// Send done event
			fmt.Fprintf(w, "event: done\ndata: {}\n\n")
			flusher.Flush()
			return
		}

		// Send streaming partial responses
		if chunk.Stream() != nil {
			for _, bamlResp := range *chunk.Stream() {
				// Stream all responses - frontend will filter what to display
				message := convertBAMLStreamResponseToMessage(&bamlResp)
				log.Logger.Infof("Streaming chunk: type=%s, text_len=%d", message["type"], len(message["text"].(string)))
				sendSSEChunk(w, flusher, message)
			}
		}
	}
}

// sendSSEMessage sends a complete message via SSE
func sendSSEMessage(w http.ResponseWriter, flusher http.Flusher, message *model.AIChatMessage) {
	data, err := json.Marshal(message)
	if err != nil {
		log.Logger.WithError(err).Error("Failed to marshal SSE message")
		return
	}
	fmt.Fprintf(w, "event: message\ndata: %s\n\n", data)
	flusher.Flush()
}

// sendSSEChunk sends a streaming chunk via SSE
func sendSSEChunk(w http.ResponseWriter, flusher http.Flusher, chunk map[string]any) {
	data, err := json.Marshal(chunk)
	if err != nil {
		log.Logger.WithError(err).Error("Failed to marshal SSE chunk")
		return
	}
	fmt.Fprintf(w, "event: chunk\ndata: %s\n\n", data)
	flusher.Flush()
}

// sendSSEError sends an error via SSE
func sendSSEError(w http.ResponseWriter, flusher http.Flusher, errorMsg string) {
	errorData := map[string]string{"error": errorMsg}
	data, _ := json.Marshal(errorData)
	fmt.Fprintf(w, "event: error\ndata: %s\n\n", data)
	flusher.Flush()
}

// convertBAMLResponseToMessage converts a BAML response to a chat message (executes SQL if needed)
func convertBAMLResponseToMessage(bamlResp *types.ChatResponse, config *engine.PluginConfig, plugin *engine.Plugin) *model.AIChatMessage {
	message := &model.AIChatMessage{
		Type:                 string(bamlResp.Type),
		Text:                 bamlResp.Text,
		RequiresConfirmation: false,
	}

	// Execute SQL if it's a query
	if bamlResp.Type == types.ChatMessageTypeSQL {
		log.Logger.Infof("Processing SQL response: operation=%v, text_len=%d", bamlResp.Operation, len(bamlResp.Text))

		if bamlResp.Operation == nil {
			// If operation is not specified, assume it's a read query and execute it
			log.Logger.Warn("SQL response missing operation type, executing as read query")
			result, execErr := plugin.RawExecute(config, bamlResp.Text)
			if execErr != nil {
				log.Logger.WithError(execErr).Error("Failed to execute SQL query")
				message.Type = "error"
				message.Text = execErr.Error()
			} else {
				// Convert result
				var columns []*model.Column
				for _, column := range result.Columns {
					columns = append(columns, &model.Column{
						Type: column.Type,
						Name: column.Name,
					})
				}
				message.Result = &model.RowsResult{
					Columns: columns,
					Rows:    result.Rows,
				}
				message.Type = "sql:get"
				log.Logger.Infof("SQL query executed successfully, rows=%d", len(result.Rows))
			}
		} else {
			// Check if operation is a mutation that requires confirmation
			isMutation := *bamlResp.Operation == types.OperationTypeINSERT ||
				*bamlResp.Operation == types.OperationTypeUPDATE ||
				*bamlResp.Operation == types.OperationTypeDELETE ||
				*bamlResp.Operation == types.OperationTypeCREATE ||
				*bamlResp.Operation == types.OperationTypeALTER ||
				*bamlResp.Operation == types.OperationTypeDROP

			if isMutation {
				// Don't execute mutations immediately - require user confirmation
				log.Logger.Infof("SQL mutation detected, requiring confirmation: operation=%v", *bamlResp.Operation)
				message.Type = convertOperationType(*bamlResp.Operation)
				message.RequiresConfirmation = true
				message.Result = nil
			} else {
				// Execute non-mutation queries (SELECT, etc.) immediately
				log.Logger.Infof("Executing non-mutation query: operation=%v", *bamlResp.Operation)
				result, execErr := plugin.RawExecute(config, bamlResp.Text)
				if execErr != nil {
					log.Logger.WithError(execErr).Error("Failed to execute SQL query")
					message.Type = "error"
					message.Text = execErr.Error()
				} else {
					// Convert result
					var columns []*model.Column
					for _, column := range result.Columns {
						columns = append(columns, &model.Column{
							Type: column.Type,
							Name: column.Name,
						})
					}
					message.Result = &model.RowsResult{
						Columns: columns,
						Rows:    result.Rows,
					}
					// Set operation-specific type
					message.Type = convertOperationType(*bamlResp.Operation)
					log.Logger.Infof("SQL query executed successfully, rows=%d", len(result.Rows))
				}
			}
		}
	}

	return message
}

// convertBAMLStreamResponseToMessage converts a streaming BAML response to a simple chunk
func convertBAMLStreamResponseToMessage(bamlResp *stream_types.ChatResponse) map[string]any {
	// Convert enum values to lowercase to match BAML @alias values
	typeStr := ""
	if bamlResp.Type != nil {
		switch *bamlResp.Type {
		case types.ChatMessageTypeSQL:
			typeStr = "sql"
		case types.ChatMessageTypeMESSAGE:
			typeStr = "message"
		case types.ChatMessageTypeERROR:
			typeStr = "error"
		default:
			typeStr = string(*bamlResp.Type)
		}
	}

	opStr := ""
	if bamlResp.Operation != nil {
		switch *bamlResp.Operation {
		case types.OperationTypeGET:
			opStr = "get"
		case types.OperationTypeINSERT:
			opStr = "insert"
		case types.OperationTypeUPDATE:
			opStr = "update"
		case types.OperationTypeDELETE:
			opStr = "delete"
		case types.OperationTypeCREATE:
			opStr = "create"
		case types.OperationTypeALTER:
			opStr = "alter"
		case types.OperationTypeDROP:
			opStr = "drop"
		case types.OperationTypeTEXT:
			opStr = "text"
		default:
			opStr = string(*bamlResp.Operation)
		}
	}

	textStr := ""
	if bamlResp.Text != nil {
		textStr = *bamlResp.Text
	}

	return map[string]any{
		"type":      typeStr,
		"text":      textStr,
		"operation": opStr,
	}
}

// convertOperationType converts BAML OperationType to WhoDB operation string
func convertOperationType(operation types.OperationType) string {
	switch operation {
	case types.OperationTypeGET:
		return "sql:get"
	case types.OperationTypeINSERT:
		return "sql:insert"
	case types.OperationTypeUPDATE:
		return "sql:update"
	case types.OperationTypeDELETE:
		return "sql:delete"
	case types.OperationTypeCREATE:
		return "sql:create"
	case types.OperationTypeALTER:
		return "sql:alter"
	case types.OperationTypeDROP:
		return "sql:drop"
	case types.OperationTypeTEXT:
		return "text"
	default:
		return "sql"
	}
}
