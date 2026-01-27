//go:build !ee && !arm && !riscv64

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
	"net/http"

	"github.com/clidey/whodb/core/baml_client"
	"github.com/clidey/whodb/core/baml_client/stream_types"
	"github.com/clidey/whodb/core/baml_client/types"
	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
)

func init() {
	RegisterAIChatStreamHandler(ceAIChatStreamHandler)
}

func ceAIChatStreamHandler(w http.ResponseWriter, r *http.Request) {
	// Parse request
	req, err := ParseStreamRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Setup SSE
	flusher := SetupSSEHeaders(w)
	if flusher == nil {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Get plugin and config
	plugin, config := GetPluginForContext(r.Context())
	config.ExternalModel = &engine.ExternalModel{
		Type:     req.ModelType,
		Token:    req.Token,
		Model:    req.Model,
		Endpoint: req.Endpoint,
	}

	// Build table details
	tableDetails, err := BuildTableDetails(plugin, config, req.Schema)
	if err != nil {
		SendSSEError(w, flusher, "Failed to get table info: "+err.Error())
		return
	}

	// Setup BAML context
	dbContext := types.DatabaseContext{
		Database_type:         config.Credentials.Type,
		Schema:                req.Schema,
		Tables_and_fields:     tableDetails,
		Previous_conversation: req.Input.PreviousConversation,
	}

	// Create BAML stream
	callOpts := common.SetupAIClientWithLogging(config.ExternalModel)
	stream, err := baml_client.Stream.GenerateSQLQuery(ctx.Background(), dbContext, req.Input.Query, callOpts...)
	if err != nil {
		SendSSEError(w, flusher, "Failed to start stream: "+err.Error())
		return
	}

	// Process stream
	processStream(w, flusher, stream, plugin, config)
}

func processStream(
	w http.ResponseWriter,
	flusher http.Flusher,
	stream <-chan baml_client.StreamValue[[]stream_types.ChatResponse, []types.ChatResponse],
	plugin *engine.Plugin,
	config *engine.PluginConfig,
) {
	for chunk := range stream {
		if chunk.IsError {
			SendSSEError(w, flusher, chunk.Error.Error())
			return
		}

		if chunk.IsFinal {
			processFinalChunk(w, flusher, chunk.Final(), plugin, config)
			SendSSEDone(w, flusher)
			return
		}

		if chunk.Stream() != nil {
			for _, bamlResp := range *chunk.Stream() {
				SendSSEChunk(w, flusher, convertStreamResponse(&bamlResp))
			}
		}
	}
}

func processFinalChunk(w http.ResponseWriter, flusher http.Flusher, responses *[]types.ChatResponse, plugin *engine.Plugin, config *engine.PluginConfig) {
	if responses == nil {
		return
	}

	for _, bamlResp := range *responses {
		if bamlResp.Type == types.ChatMessageTypeSQL {
			message := executeSQLResponse(&bamlResp, plugin, config)
			SendSSEMessage(w, flusher, message)
		}
	}
}

func executeSQLResponse(bamlResp *types.ChatResponse, plugin *engine.Plugin, config *engine.PluginConfig) *model.AIChatMessage {
	message := &model.AIChatMessage{
		Type:                 string(bamlResp.Type),
		Text:                 bamlResp.Text,
		RequiresConfirmation: false,
	}

	if bamlResp.Operation == nil {
		result, err := plugin.RawExecute(config, bamlResp.Text)
		if err != nil {
			message.Type = "error"
			message.Text = err.Error()
		} else {
			message.Result = ConvertResultToMessage(result)
			message.Type = "sql:get"
		}
		return message
	}

	// Check if mutation
	isMutation := *bamlResp.Operation == types.OperationTypeINSERT ||
		*bamlResp.Operation == types.OperationTypeUPDATE ||
		*bamlResp.Operation == types.OperationTypeDELETE ||
		*bamlResp.Operation == types.OperationTypeCREATE ||
		*bamlResp.Operation == types.OperationTypeALTER ||
		*bamlResp.Operation == types.OperationTypeDROP

	if isMutation {
		message.Type = convertOperationType(*bamlResp.Operation)
		message.RequiresConfirmation = true
	} else {
		result, err := plugin.RawExecute(config, bamlResp.Text)
		if err != nil {
			message.Type = "error"
			message.Text = err.Error()
		} else {
			message.Result = ConvertResultToMessage(result)
			message.Type = convertOperationType(*bamlResp.Operation)
		}
	}

	return message
}

func convertStreamResponse(bamlResp *stream_types.ChatResponse) map[string]any {
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
		opStr = operationToString(*bamlResp.Operation)
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

func operationToString(op types.OperationType) string {
	switch op {
	case types.OperationTypeGET:
		return "get"
	case types.OperationTypeINSERT:
		return "insert"
	case types.OperationTypeUPDATE:
		return "update"
	case types.OperationTypeDELETE:
		return "delete"
	case types.OperationTypeCREATE:
		return "create"
	case types.OperationTypeALTER:
		return "alter"
	case types.OperationTypeDROP:
		return "drop"
	case types.OperationTypeTEXT:
		return "text"
	default:
		return string(op)
	}
}

func convertOperationType(operation types.OperationType) string {
	return "sql:" + operationToString(operation)
}
