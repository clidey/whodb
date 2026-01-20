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
	"strconv"

	"github.com/clidey/whodb/core/baml_client"
	"github.com/clidey/whodb/core/baml_client/stream_types"
	"github.com/clidey/whodb/core/baml_client/types"
	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
	"github.com/go-chi/chi/v5"
)

// SetupHTTPServer registers REST API endpoints that wrap GraphQL resolvers for clients
// that cannot use GraphQL directly (e.g., simple HTTP clients, legacy integrations).
func SetupHTTPServer(router chi.Router) {
	router.Get("/api/profiles", getProfilesHandler)
	router.Get("/api/databases", getDatabasesHandler)
	router.Get("/api/schema", getSchemaHandler)
	router.Get("/api/storage-units", getStorageUnitsHandler)
	router.Get("/api/rows", getRowsHandler)
	router.Post("/api/raw-execute", rawExecuteHandler)
	router.Get("/api/graph", getGraphHandler)
	router.Get("/api/ai-models", getAIModelsHandler)
	router.Post("/api/ai-chat", aiChatHandler)
	router.Post("/api/ai-chat/stream", aiChatStreamHandler)

	router.Post("/api/storage-units", addStorageUnitHandler)
	router.Post("/api/rows", addRowHandler)
	router.Delete("/api/rows", deleteRowHandler)

	router.Post("/api/export", HandleExport)
}

var resolver = mutationResolver{}

func getProfilesHandler(w http.ResponseWriter, r *http.Request) {
	profiles, err := resolver.Query().Profiles(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = json.NewEncoder(w).Encode(profiles)
	if err != nil {
		return
	}
}

func getDatabasesHandler(w http.ResponseWriter, r *http.Request) {
	typeArg := r.URL.Query().Get("type")
	databases, err := resolver.Query().Database(r.Context(), typeArg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = json.NewEncoder(w).Encode(databases)
	if err != nil {
		return
	}
}

func getSchemaHandler(w http.ResponseWriter, r *http.Request) {
	schemas, err := resolver.Query().Schema(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = json.NewEncoder(w).Encode(schemas)
	if err != nil {
		return
	}
}

func getStorageUnitsHandler(w http.ResponseWriter, r *http.Request) {
	schema := r.URL.Query().Get("schema")
	storageUnits, err := resolver.Query().StorageUnit(r.Context(), schema)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = json.NewEncoder(w).Encode(storageUnits)
	if err != nil {
		return
	}
}

func getRowsHandler(w http.ResponseWriter, r *http.Request) {
	schema := r.URL.Query().Get("schema")
	storageUnit := r.URL.Query().Get("storageUnit")
	pageSize := parseQueryParamToInt(r.URL.Query().Get("pageSize"))
	pageOffset := parseQueryParamToInt(r.URL.Query().Get("pageOffset"))

	// TODO: Add where condition parsing from query params if needed
	rowsResult, err := resolver.Query().Row(r.Context(), schema, storageUnit, &model.WhereCondition{}, []*model.SortCondition{}, pageSize, pageOffset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = json.NewEncoder(w).Encode(rowsResult)
	if err != nil {
		return
	}
}

func rawExecuteHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Query string `json:"query"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	rowsResult, err := resolver.Query().RawExecute(r.Context(), req.Query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = json.NewEncoder(w).Encode(rowsResult)
	if err != nil {
		return
	}
}

func getGraphHandler(w http.ResponseWriter, r *http.Request) {
	schema := r.URL.Query().Get("schema")
	graphUnits, err := resolver.Query().Graph(r.Context(), schema)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = json.NewEncoder(w).Encode(graphUnits)
	if err != nil {
		return
	}
}

func getAIModelsHandler(w http.ResponseWriter, r *http.Request) {
	modelType := r.URL.Query().Get("modelType")
	token := r.URL.Query().Get("token")
	models, err := resolver.Query().AIModel(r.Context(), nil, modelType, &token)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = json.NewEncoder(w).Encode(models)
	if err != nil {
		return
	}
}

func aiChatHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ModelType string          `json:"modelType"`
		Token     string          `json:"token"`
		Schema    string          `json:"schema"`
		Input     model.ChatInput `json:"input"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	messages, err := resolver.Query().AIChat(r.Context(), nil, req.ModelType, &req.Token, req.Schema, req.Input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = json.NewEncoder(w).Encode(messages)
	if err != nil {
		return
	}
}

func addStorageUnitHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Schema      string               `json:"schema"`
		StorageUnit string               `json:"storageUnit"`
		Fields      []*model.RecordInput `json:"fields"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	status, err := resolver.Mutation().AddStorageUnit(r.Context(), req.Schema, req.StorageUnit, req.Fields)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = json.NewEncoder(w).Encode(status)
	if err != nil {
		return
	}
}

func addRowHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Schema      string               `json:"schema"`
		StorageUnit string               `json:"storageUnit"`
		Values      []*model.RecordInput `json:"values"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	status, err := resolver.Mutation().AddRow(r.Context(), req.Schema, req.StorageUnit, req.Values)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = json.NewEncoder(w).Encode(status)
	if err != nil {
		return
	}
}

func deleteRowHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Schema      string               `json:"schema"`
		StorageUnit string               `json:"storageUnit"`
		Values      []*model.RecordInput `json:"values"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	status, err := resolver.Mutation().DeleteRow(r.Context(), req.Schema, req.StorageUnit, req.Values)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = json.NewEncoder(w).Encode(status)
	if err != nil {
		return
	}
}

func parseQueryParamToInt(queryParam string) int {
	if queryParam == "" {
		return 0
	}
	value, err := strconv.Atoi(queryParam)
	if err != nil {
		return 0
	}
	return value
}

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

	// Build table context (same as non-streaming version)
	rows, err := plugin.GetStorageUnits(config, req.Schema)
	if err != nil {
		sendSSEError(w, flusher, fmt.Sprintf("Failed to get table info: %v", err))
		return
	}

	// Build table details string
	tableDetails := ""
	for _, unit := range rows {
		tableDetails += fmt.Sprintf("table: %s\n", unit.Name)
		for _, attr := range unit.Attributes {
			tableDetails += fmt.Sprintf("- %s (%s)\n", attr.Key, attr.Value)
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
				for _, bamlResp := range *final {
					// Only execute SQL for SQL-type responses
					if bamlResp.Type == types.ChatMessageTypeSQL {
						message := convertBAMLResponseToMessage(&bamlResp, config, plugin)
						sendSSEMessage(w, flusher, message)
					}
					// Message-type responses were already streamed
				}
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
	if bamlResp.Type == types.ChatMessageTypeSQL && bamlResp.Operation != nil {
		// Check if operation is a mutation that requires confirmation
		isMutation := *bamlResp.Operation == types.OperationTypeINSERT ||
			*bamlResp.Operation == types.OperationTypeUPDATE ||
			*bamlResp.Operation == types.OperationTypeDELETE ||
			*bamlResp.Operation == types.OperationTypeCREATE ||
			*bamlResp.Operation == types.OperationTypeALTER ||
			*bamlResp.Operation == types.OperationTypeDROP

		if isMutation {
			// Don't execute mutations immediately - require user confirmation
			message.Type = convertOperationType(*bamlResp.Operation)
			message.RequiresConfirmation = true
			message.Result = nil
		} else {
			// Execute non-mutation queries (SELECT, etc.) immediately
			result, execErr := plugin.RawExecute(config, bamlResp.Text)
			if execErr != nil {
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

// getOperationString safely converts operation pointer to string
func getOperationString(operation *types.OperationType) string {
	if operation == nil {
		return ""
	}
	return convertOperationType(*operation)
}
