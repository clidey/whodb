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
	stdctx "context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/clidey/whodb/core/baml_client"
	"github.com/clidey/whodb/core/baml_client/types"
	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/src/bamlconfig"
	"github.com/clidey/whodb/core/src/envconfig"
	"github.com/clidey/whodb/core/src/log"
	"github.com/clidey/whodb/core/src/source"
	"github.com/clidey/whodb/core/src/sqlguard"
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
	chatContext, cancel := stdctx.WithTimeout(r.Context(), 90*time.Second)
	defer cancel()

	spec, session, err := getSourceSessionForContext(chatContext)
	if err != nil {
		log.Debugf("AI Chat Stream: Failed to create source session: %v", err)
		SendSSEError(w, flusher, "No source session available")
		return
	}
	auditScope := sourceAuditScopeFromContext(r.Context(), spec)
	queryRunner, ok := source.AsReadOnlyQueryRunner(auditScope, session)
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

	SendSSEProgress(w, flusher, "schema", "started")
	// Build object details for the selected chat scope.
	log.Debugf("AI Chat Stream: Building object details for ref=%+v", req.Ref)
	resolvedRef := sourceRefFromInput(req.Ref)
	tableDetails, err := BuildObjectDetails(chatContext, auditScope, session, resolvedRef, spec.Contract.DefaultObjectKind)
	if err != nil {
		log.Debugf("AI Chat Stream: BuildObjectDetails failed: %v", err)
		SendSSEError(w, flusher, "Failed to get table info: "+err.Error())
		return
	}
	log.Debugf("AI Chat Stream: Table details built, length=%d", len(tableDetails))
	SendSSEProgress(w, flusher, "schema", "completed")

	scope := sourceScopeForChat(spec, resolvedRef)

	// Setup BAML context
	dbContext := types.DatabaseContext{
		Database_type:         spec.ID,
		Schema:                scope,
		Tables_and_fields:     tableDetails,
		Previous_conversation: req.Input.PreviousConversation,
	}
	log.Debugf("AI Chat Stream: BAML context created")

	callOpts := bamlconfig.SetupAIClient(modelConfig)
	runChatLoop(chatContext, w, flusher, dbContext, req.Input.Query, callOpts, queryRunner)
}

func runChatLoop(
	ctx stdctx.Context,
	w http.ResponseWriter,
	flusher http.Flusher,
	dbContext types.DatabaseContext,
	userQuery string,
	callOpts []baml_client.CallOptionFunc,
	queryRunner source.ReadOnlyQueryRunner,
) {
	request := userQuery
	var lastFailedSQL, lastQueryError string
	for attempt := 0; attempt < 2; attempt++ {
		SendSSEProgress(w, flusher, "plan", "started")
		stream, err := baml_client.Stream.GenerateSQLQuery(ctx, dbContext, request, callOpts...)
		if err != nil {
			sendModelError(w, flusher, err.Error())
			return
		}
		var responses *[]types.ChatResponse
		for chunk := range stream {
			if chunk.IsError {
				sendModelError(w, flusher, chunk.Error.Error())
				return
			}
			if chunk.IsFinal {
				responses = chunk.Final()
				break
			}
		}
		if err := ctx.Err(); err != nil {
			if err == stdctx.DeadlineExceeded {
				SendSSEError(w, flusher, "The chat request timed out. Please try again.")
			}
			return
		}
		SendSSEProgress(w, flusher, "plan", "completed")
		if responses == nil || len(*responses) == 0 {
			SendSSEError(w, flusher, "The model returned no answer")
			return
		}

		messages := make([]*model.AIChatMessage, 0, len(*responses))
		var retryReason string
		for _, response := range *responses {
			if unsupportedChatTool(response.Text) {
				retryReason = "This chat cannot invoke named tools. Answer using listed database tables or explain the limitation."
				break
			}
			if response.Type == types.ChatMessageTypeSQL {
				step := "query"
				if sqlguard.Classify(response.Text).Mutating {
					step = "draft"
				}
				SendSSEProgress(w, flusher, step, "started")
				message := processFinalResponse(ctx, &response, queryRunner)
				SendSSEProgress(w, flusher, step, "completed")
				if message.Type == "error" {
					retryReason = message.Text
					lastFailedSQL = response.Text
					lastQueryError = message.Text
					break
				}
				messages = append(messages, message)
				continue
			}
			message := processFinalResponse(ctx, &response, queryRunner)
			messages = append(messages, message)
		}
		if retryReason != "" && attempt == 0 {
			SendSSEProgress(w, flusher, "retry", "started")
			request = userQuery + "\nThe previous plan failed: " + retryReason + "\nReturn a corrected answer. Use only SQL on the listed database tables; do not claim that any action has run."
			SendSSEProgress(w, flusher, "retry", "completed")
			continue
		}
		if retryReason != "" {
			if lastFailedSQL != "" {
				SendSSESQLFailure(w, flusher, lastFailedSQL, lastQueryError)
			} else if strings.Contains(retryReason, "cannot invoke named tools") {
				SendSSEMessage(w, flusher, &model.AIChatMessage{Type: "scope:error", Text: "unsupported_tool"})
				SendSSEDone(w, flusher)
			} else {
				SendSSEError(w, flusher, retryReason)
			}
			return
		}
		hasSQL := false
		for _, response := range *responses {
			if response.Type == types.ChatMessageTypeSQL {
				hasSQL = true
				break
			}
		}
		if lastFailedSQL != "" && !hasSQL {
			SendSSESQLFailure(w, flusher, lastFailedSQL, lastQueryError)
			return
		}
		for _, message := range messages {
			if hasSQL && message.Type == "message" {
				continue
			}
			SendSSEMessage(w, flusher, message)
		}
		SendSSEDone(w, flusher)
		return
	}
}

func sendModelError(w http.ResponseWriter, flusher http.Flusher, message string) {
	lower := strings.ToLower(message)
	if strings.Contains(lower, "connection refused") || strings.Contains(lower, "could not connect") || strings.Contains(lower, "connect error") {
		SendSSEMessage(w, flusher, &model.AIChatMessage{Type: "provider:error", Text: sanitizeErrorMessage(message)})
		SendSSEDone(w, flusher)
		return
	}
	SendSSEError(w, flusher, message)
}

func unsupportedChatTool(text string) bool {
	var value map[string]json.RawMessage
	return json.Unmarshal([]byte(strings.TrimSpace(text)), &value) == nil && value["toolName"] != nil
}

func processFinalResponse(ctx stdctx.Context, bamlResp *types.ChatResponse, queryRunner source.ReadOnlyQueryRunner) *model.AIChatMessage {
	message := &model.AIChatMessage{
		Type: bamlconfig.ConvertBAMLTypeToWhoDB(bamlResp.Type),
		Text: bamlResp.Text,
	}

	if bamlResp.Type != types.ChatMessageTypeSQL {
		return message
	}
	if bamlResp.Operation == nil {
		message.Type = "error"
		message.Text = "The model did not specify a query operation"
		return message
	}

	classification := sqlguard.Classify(bamlResp.Text)

	if classification.Mutating {
		message.Type = "sql:" + sqlguard.OperationName(classification)
		message.RequiresConfirmation = true
		return message
	}

	result, err := queryRunner.RunReadOnlyQuery(ctx, bamlResp.Text)
	if err != nil {
		message.Type = "error"
		message.Text = err.Error()
		return message
	}

	message.Type = "sql:get"
	message.Result = ConvertResultToMessage(result)
	return message
}
