//go:build !arm && !riscv64

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
	"github.com/clidey/whodb/core/src/bamlconfig"
	"github.com/clidey/whodb/core/src/envconfig"
	"github.com/clidey/whodb/core/src/log"
	"github.com/clidey/whodb/core/src/source"
)

func init() {
	RegisterAIChatStreamHandler(ceAIChatStreamHandler)
}

func ceAIChatStreamHandler(w http.ResponseWriter, r *http.Request) {
	log.Debugf("AI Chat Stream: Handler started")

	// Parse request
	req, err := ParseStreamRequest(r)
	if err != nil {
		log.Debugf("AI Chat Stream: ParseStreamRequest failed: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	log.Debugf("AI Chat Stream: Request parsed - model=%s, ref=%+v, query=%s", req.ModelType, req.Ref, req.Input.Query)

	// Setup SSE
	flusher := SetupSSEHeaders(w)
	if flusher == nil {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	spec, session, err := getSourceSessionForContext(r.Context())
	if err != nil {
		log.Debugf("AI Chat Stream: Failed to create source session: %v", err)
		SendSSEError(w, flusher, "No source session available")
		return
	}
	auditScope := sourceAuditScopeFromContext(r.Context(), spec)
	queryRunner, ok := source.AsQueryRunner(auditScope, session)
	if !ok {
		SendSSEError(w, flusher, "Source queries are not supported")
		return
	}

	// Build ExternalModel, resolving credentials from environment if providerId is set
	creds := envconfig.ResolveProviderCredentials(req.ProviderId, req.Token, req.Endpoint, req.ModelType)
	modelConfig := &source.ExternalModel{
		Type:     creds.ModelType,
		Token:    creds.Token,
		Model:    req.Model,
		Endpoint: creds.Endpoint,
	}

	// Build object details for the selected chat scope.
	log.Debugf("AI Chat Stream: Building object details for ref=%+v", req.Ref)
	resolvedRef := sourceRefFromInput(req.Ref)
	tableDetails, err := BuildObjectDetails(r.Context(), auditScope, session, resolvedRef, spec.Contract.DefaultObjectKind)
	if err != nil {
		log.Debugf("AI Chat Stream: BuildObjectDetails failed: %v", err)
		SendSSEError(w, flusher, "Failed to get table info: "+err.Error())
		return
	}
	log.Debugf("AI Chat Stream: Table details built, length=%d", len(tableDetails))

	scope := sourceScopeForChat(spec, resolvedRef)

	// Setup BAML context
	dbContext := types.DatabaseContext{
		Database_type:         spec.ID,
		Schema:                scope,
		Tables_and_fields:     tableDetails,
		Previous_conversation: req.Input.PreviousConversation,
	}
	log.Debugf("AI Chat Stream: BAML context created")

	// Create BAML stream
	log.Debugf("AI Chat Stream: Setting up AI client...")
	callOpts := bamlconfig.SetupAIClient(modelConfig)
	log.Debugf("AI Chat Stream: Starting BAML GenerateSQLQuery stream...")
	stream, err := baml_client.Stream.GenerateSQLQuery(ctx.Background(), dbContext, req.Input.Query, callOpts...)
	if err != nil {
		log.Debugf("AI Chat Stream: GenerateSQLQuery failed: %v", err)
		SendSSEError(w, flusher, "Failed to start stream: "+err.Error())
		return
	}
	log.Debugf("AI Chat Stream: BAML stream created successfully")

	// Process stream
	log.Debugf("AI Chat Stream: Starting to process stream...")
	processStream(r.Context(), w, flusher, stream, queryRunner)
	log.Debugf("AI Chat Stream: Stream processing completed")
}

func processStream(
	ctx ctx.Context,
	w http.ResponseWriter,
	flusher http.Flusher,
	stream <-chan baml_client.StreamValue[[]stream_types.ChatResponse, []types.ChatResponse],
	queryRunner source.QueryRunner,
) {
	for chunk := range stream {
		if chunk.IsError {
			SendSSEError(w, flusher, chunk.Error.Error())
			return
		}

		if chunk.IsFinal {
			processFinalChunk(ctx, w, flusher, chunk.Final(), queryRunner)
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

func processFinalChunk(ctx ctx.Context, w http.ResponseWriter, flusher http.Flusher, responses *[]types.ChatResponse, queryRunner source.QueryRunner) {
	if responses == nil {
		return
	}

	for _, bamlResp := range *responses {
		SendSSEMessage(w, flusher, processFinalResponse(ctx, &bamlResp, queryRunner))
	}
}

func processFinalResponse(ctx ctx.Context, bamlResp *types.ChatResponse, queryRunner source.QueryRunner) *model.AIChatMessage {
	message := &model.AIChatMessage{
		Type: bamlconfig.ConvertBAMLTypeToWhoDB(bamlResp.Type),
		Text: bamlResp.Text,
	}

	if bamlResp.Type != types.ChatMessageTypeSQL || bamlResp.Operation == nil {
		return message
	}

	isMutation := *bamlResp.Operation == types.OperationTypeINSERT ||
		*bamlResp.Operation == types.OperationTypeUPDATE ||
		*bamlResp.Operation == types.OperationTypeDELETE ||
		*bamlResp.Operation == types.OperationTypeCREATE ||
		*bamlResp.Operation == types.OperationTypeALTER ||
		*bamlResp.Operation == types.OperationTypeDROP

	if isMutation {
		message.Type = bamlconfig.ConvertOperationType(*bamlResp.Operation)
		message.RequiresConfirmation = true
		return message
	}

	result, err := queryRunner.RunQuery(ctx, bamlResp.Text)
	if err != nil {
		message.Type = "error"
		message.Text = err.Error()
		return message
	}

	message.Type = bamlconfig.ConvertOperationType(*bamlResp.Operation)
	message.Result = ConvertResultToMessage(result)
	return message
}

func convertStreamResponse(bamlResp *stream_types.ChatResponse) map[string]any {
	typeStr := ""
	if bamlResp.Type != nil {
		typeStr = bamlconfig.ConvertBAMLTypeToWhoDB(*bamlResp.Type)
	}

	opStr := ""
	if bamlResp.Operation != nil {
		opStr = bamlconfig.OperationToString(*bamlResp.Operation)
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
