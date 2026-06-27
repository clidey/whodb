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

package bamlconfig

import (
	"context"

	baml "github.com/boundaryml/baml/engine/language_client_go/pkg"

	"github.com/clidey/whodb/core/baml_client"
	"github.com/clidey/whodb/core/baml_client/types"
	"github.com/clidey/whodb/core/src/log"
	"github.com/clidey/whodb/core/src/security"
	"github.com/clidey/whodb/core/src/source"
)

// ChatQueryExecutor executes read queries produced by the chat planner.
type ChatQueryExecutor interface {
	RunQuery(ctx context.Context, query string, params ...any) (*source.RowsResult, error)
}

// ChatQueryExecutorFunc adapts a function to ChatQueryExecutor.
type ChatQueryExecutorFunc func(ctx context.Context, query string, params ...any) (*source.RowsResult, error)

// RunQuery executes the wrapped function.
func (fn ChatQueryExecutorFunc) RunQuery(ctx context.Context, query string, params ...any) (*source.RowsResult, error) {
	return fn(ctx, query, params...)
}

// ExecuteChatQuery is the single non-streaming chat execution path.
// It calls the BAML prompt, executes read queries via the supplied executor,
// and gates mutations for user confirmation.
// Used by plugin chat implementations that build SQL context locally.
func ExecuteChatQuery(
	ctx context.Context,
	databaseType string,
	schema string,
	tableDetails string,
	previousConversation string,
	userQuery string,
	model *source.ExternalModel,
	executor ChatQueryExecutor,
) ([]*source.ChatMessage, error) {

	dbContext := types.DatabaseContext{
		Database_type:         databaseType,
		Schema:                schema,
		Tables_and_fields:     tableDetails,
		Previous_conversation: previousConversation,
	}

	callOpts := SetupAIClient(model)
	responses, err := baml_client.GenerateSQLQuery(ctx, dbContext, userQuery, callOpts...)
	if err != nil {
		return nil, err
	}

	var chatMessages []*source.ChatMessage
	for _, bamlResp := range responses {
		chatMessages = append(chatMessages, ProcessChatResponse(ctx, &bamlResp, executor))
	}
	return chatMessages, nil
}

// ProcessChatResponse converts a single BAML ChatResponse into a source chat
// message.
// Read queries are executed immediately; mutations are gated for user confirmation.
func ProcessChatResponse(ctx context.Context, bamlResp *types.ChatResponse, executor ChatQueryExecutor) *source.ChatMessage {
	message := &source.ChatMessage{
		Type: ConvertBAMLTypeToWhoDB(bamlResp.Type),
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
		message.Type = ConvertOperationType(*bamlResp.Operation)
		message.RequiresConfirmation = true
		return message
	}

	result, err := executor.RunQuery(ctx, bamlResp.Text)
	if err != nil {
		message.Type = "error"
		message.Text = err.Error()
	} else {
		message.Type = ConvertOperationType(*bamlResp.Operation)
		message.Result = result
	}
	return message
}

// ConvertBAMLTypeToWhoDB converts BAML ChatMessageType to WhoDB message type string.
func ConvertBAMLTypeToWhoDB(bamlType types.ChatMessageType) string {
	switch bamlType {
	case types.ChatMessageTypeSQL:
		return "sql"
	case types.ChatMessageTypeMESSAGE:
		return "message"
	case types.ChatMessageTypeERROR:
		return "error"
	default:
		return "message"
	}
}

// OperationToString converts a BAML OperationType to its short string form (e.g. "get", "insert").
func OperationToString(op types.OperationType) string {
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

// ConvertOperationType converts a BAML OperationType to the prefixed form used in chat messages (e.g. "sql:get").
func ConvertOperationType(op types.OperationType) string {
	return "sql:" + OperationToString(op)
}

// BAMLConfigResolver resolves BAML provider string + options for a given provider type.
// Injected by the llm package at init time to use the provider registry.
type BAMLConfigResolver func(providerType, apiKey, endpoint, model string) (string, map[string]any, error)

var bamlConfigResolver BAMLConfigResolver

// RegisterBAMLConfigResolver sets the function used to resolve BAML config from the provider registry.
func RegisterBAMLConfigResolver(resolver BAMLConfigResolver) {
	bamlConfigResolver = resolver
}

// SetupAIClient creates the BAML client options for the given external model.
func SetupAIClient(externalModel *source.ExternalModel) []baml_client.CallOptionFunc {
	var callOpts []baml_client.CallOptionFunc
	if externalModel == nil {
		return callOpts
	}
	if externalModel.Model == "" {
		return callOpts
	}

	if registry := CreateDynamicBAMLClient(externalModel); registry != nil {
		callOpts = append(callOpts, baml_client.WithClientRegistry(registry))
	}

	return callOpts
}

// CreateDynamicBAMLClient creates a BAML ClientRegistry with a dynamically configured client
// based on the user's selected provider, model, API key, and endpoint.
// We register as "DefaultClient" to override the BAML function's explicit client reference.
func CreateDynamicBAMLClient(externalModel *source.ExternalModel) *baml.ClientRegistry {
	if externalModel == nil {
		return nil
	}

	// Block SSRF via a user-supplied endpoint. The actual HTTP call happens
	// inside the BAML CFFI layer (no Go dialer to wrap), so validate the URL
	// before handing it off; on failure fall back to the default client.
	if externalModel.Endpoint != "" {
		if err := security.EnforceOutboundURL(externalModel.Endpoint); err != nil {
			log.Errorf("blocked AI provider endpoint: %v", err)
			return nil
		}
	}

	registry := baml.NewClientRegistry()

	provider, opts := getBAMLProviderAndOptions(externalModel)

	// Register as "DefaultClient" to override the static client reference in BAML functions
	registry.AddLlmClient("DefaultClient", provider, opts)
	registry.SetPrimaryClient("DefaultClient")

	return registry
}

// getBAMLProviderAndOptions maps WhoDB ExternalModel to BAML provider string and options.
// Delegates to the provider registry when available, falling back to openai-generic for unknown types.
func getBAMLProviderAndOptions(m *source.ExternalModel) (string, map[string]any) {
	if bamlConfigResolver != nil {
		provider, opts, err := bamlConfigResolver(m.Type, m.Token, m.Endpoint, m.Model)
		if err == nil {
			return provider, opts
		}
	}
	// Fallback for unregistered provider types
	opts := map[string]any{"model": m.Model}
	if m.Endpoint != "" {
		opts["base_url"] = m.Endpoint
	}
	if m.Token != "" {
		opts["api_key"] = m.Token
	}
	return "openai-generic", opts
}
