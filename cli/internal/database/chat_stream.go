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

package database

import (
	"context"
	"fmt"
	"strings"

	"github.com/clidey/whodb/cli/internal/baml"
	"github.com/clidey/whodb/core/baml_client"
	"github.com/clidey/whodb/core/baml_client/types"
	"github.com/clidey/whodb/core/src/bamlconfig"
	"github.com/clidey/whodb/core/src/source"
)

// SendAIChatStream starts a streaming AI chat and returns a channel of StreamChunks.
// Each chunk contains the accumulated text so far. The final chunk has IsFinal=true
// with the complete ChatMessage responses.
func (m *Manager) SendAIChatStream(ctx context.Context, providerID, modelType, token, schema, model, previousConversation, query string) (<-chan StreamChunk, error) {
	if m.currentConnection == nil {
		return nil, fmt.Errorf("not connected to any database")
	}

	baml.Ensure()

	spec, session, err := m.currentSourceSession(ctx)
	if err != nil {
		return nil, err
	}

	reader, ok := session.(source.TabularReader)
	if !ok {
		return nil, fmt.Errorf("chat is not supported for %s", spec.Label)
	}

	runner, ok := session.(source.QueryRunner)
	if !ok {
		return nil, fmt.Errorf("querying is not supported for %s", spec.Label)
	}

	externalModel := resolveExternalModel(providerID, modelType, token, model)

	tableDetails, err := m.buildSourceChatTableDetails(ctx, spec, session, reader, schema)
	if err != nil {
		return nil, fmt.Errorf("failed to get table info: %w", err)
	}

	// Build BAML context
	dbContext := types.DatabaseContext{
		Database_type:         spec.ID,
		Schema:                schema,
		Tables_and_fields:     tableDetails,
		Previous_conversation: previousConversation,
	}

	// Setup BAML client
	callOpts := bamlconfig.SetupAIClient(externalModel)

	// Start BAML stream
	bamlStream, err := baml_client.Stream.GenerateSQLQuery(ctx, dbContext, query, callOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to start stream: %w", err)
	}

	// Read from BAML stream and convert to StreamChunks
	out := make(chan StreamChunk, 1)
	go func() {
		defer close(out)

		var lastText string

		for chunk := range bamlStream {
			if chunk.IsError {
				out <- StreamChunk{Err: chunk.Error}
				return
			}

			if chunk.IsFinal {
				final := chunk.Final()
				if final == nil {
					out <- StreamChunk{IsFinal: true, Final: []*ChatMessage{}}
					return
				}
				messages := convertSourceFinalResponses(ctx, *final, runner)
				out <- StreamChunk{IsFinal: true, Final: messages}
				return
			}

			// Streaming chunk — accumulate text
			if stream := chunk.Stream(); stream != nil {
				for _, resp := range *stream {
					if resp.Text != nil {
						lastText = *resp.Text
					}
				}
				if lastText != "" {
					out <- StreamChunk{Text: lastText}
				}
			}
		}

		// Stream ended without an explicit IsFinal chunk — synthesize a final message
		// from whatever text we accumulated
		if lastText != "" {
			out <- StreamChunk{
				IsFinal: true,
				Final: []*ChatMessage{{
					Type: "message",
					Text: lastText,
				}},
			}
		} else {
			out <- StreamChunk{IsFinal: true, Final: []*ChatMessage{}}
		}
	}()

	return out, nil
}

func (m *Manager) buildSourceChatTableDetails(ctx context.Context, spec source.TypeSpec, session source.SourceSession, reader source.TabularReader, schema string) (string, error) {
	objects, err := m.listStorageUnitObjects(ctx, spec, session, schema)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	for _, object := range objects {
		fmt.Fprintf(&b, "table: %s\n", object.Name)
		columns, err := reader.Columns(ctx, object.Ref)
		if err != nil {
			continue
		}
		for _, col := range columns {
			fmt.Fprintf(&b, "- %s (%s)\n", col.Name, col.Type)
		}
	}
	return b.String(), nil
}

type sourceChatQueryExecutor struct {
	ctx    context.Context
	runner source.QueryRunner
}

func (e *sourceChatQueryExecutor) RunQuery(_ context.Context, query string, params ...any) (*source.RowsResult, error) {
	return e.runner.RunQuery(e.ctx, query, params...)
}

// convertSourceFinalResponses converts BAML final responses to ChatMessages,
// executing read queries through the active source session and leaving mutations
// gated for user confirmation.
func convertSourceFinalResponses(ctx context.Context, responses []types.ChatResponse, runner source.QueryRunner) []*ChatMessage {
	var messages []*ChatMessage
	executor := &sourceChatQueryExecutor{ctx: ctx, runner: runner}
	for _, resp := range responses {
		resp.Text = strings.TrimRight(strings.TrimSpace(resp.Text), ";")

		chatMsg := bamlconfig.ProcessChatResponse(ctx, &resp, executor)
		messages = append(messages, &ChatMessage{
			Type:                 chatMsg.Type,
			Text:                 chatMsg.Text,
			RequiresConfirmation: chatMsg.RequiresConfirmation,
			Result:               chatMsg.Result,
		})
	}
	return messages
}
