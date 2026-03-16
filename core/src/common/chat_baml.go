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

package common

import (
	"context"

	baml "github.com/boundaryml/baml/engine/language_client_go/pkg"
	"github.com/clidey/whodb/core/baml_client"
	"github.com/clidey/whodb/core/baml_client/types"
	"github.com/clidey/whodb/core/src/engine"
)

// RawExecutePlugin defines the interface for executing raw queries
type RawExecutePlugin interface {
	RawExecute(config *engine.PluginConfig, query string, params ...any) (*engine.GetRowsResult, error)
}

// IsNoSQLDatabase returns true if the database type should use the database-agnostic
// BAML prompt (GenerateDBQuery) instead of the SQL-specific prompt (GenerateSQLQuery).
func IsNoSQLDatabase(dbType string) bool {
	switch engine.DatabaseType(dbType) {
	case engine.DatabaseType_MongoDB,
		engine.DatabaseType_Redis,
		engine.DatabaseType_ElasticSearch:
		return true
	}
	return false
}

// ExecuteChatQuery is the single non-streaming chat execution path.
// Picks the right BAML prompt based on database type, calls it, executes
// read queries via plugin.RawExecute(), and gates mutations for confirmation.
// Used by both plugin.Chat() implementations and the HTTP non-streaming fallback.
func ExecuteChatQuery(
	ctx context.Context,
	databaseType string,
	schema string,
	tableDetails string,
	previousConversation string,
	userQuery string,
	config *engine.PluginConfig,
	plugin RawExecutePlugin,
) ([]*engine.ChatMessage, error) {

	dbContext := types.DatabaseContext{
		Database_type:         databaseType,
		Schema:                schema,
		Tables_and_fields:     tableDetails,
		Previous_conversation: previousConversation,
	}

	callOpts := SetupAIClient(config.ExternalModel)

	var responses []types.ChatResponse
	var err error
	if IsNoSQLDatabase(databaseType) {
		responses, err = baml_client.GenerateDBQuery(ctx, dbContext, userQuery, callOpts...)
	} else {
		responses, err = baml_client.GenerateSQLQuery(ctx, dbContext, userQuery, callOpts...)
	}
	if err != nil {
		return nil, err
	}

	var chatMessages []*engine.ChatMessage
	for _, bamlResp := range responses {
		message := &engine.ChatMessage{
			Type:                 string(bamlResp.Type),
			Text:                 bamlResp.Text,
			Result:               &engine.GetRowsResult{},
			RequiresConfirmation: false,
		}

		message.Type = convertBAMLTypeToWhoDB(bamlResp.Type)

		if bamlResp.Type == types.ChatMessageTypeSQL && bamlResp.Operation != nil {
			isMutation := *bamlResp.Operation == types.OperationTypeINSERT ||
				*bamlResp.Operation == types.OperationTypeUPDATE ||
				*bamlResp.Operation == types.OperationTypeDELETE ||
				*bamlResp.Operation == types.OperationTypeCREATE ||
				*bamlResp.Operation == types.OperationTypeALTER ||
				*bamlResp.Operation == types.OperationTypeDROP

			if isMutation {
				message.Type = convertOperationType(*bamlResp.Operation)
				message.RequiresConfirmation = true
				message.Result = nil
			} else {
				result, execErr := plugin.RawExecute(config, bamlResp.Text)
				if execErr != nil {
					message.Type = "error"
					message.Text = execErr.Error()
				} else {
					message.Type = convertOperationType(*bamlResp.Operation)
				}
				message.Result = result
			}
		}

		chatMessages = append(chatMessages, message)
	}

	return chatMessages, nil
}

// SQLChatBAML is a convenience wrapper for SQL databases. Calls ExecuteChatQuery.
// Deprecated: Use ExecuteChatQuery directly.
func SQLChatBAML(
	ctx context.Context,
	databaseType string,
	schema string,
	tableDetails string,
	previousConversation string,
	userQuery string,
	config *engine.PluginConfig,
	plugin RawExecutePlugin,
) ([]*engine.ChatMessage, error) {
	return ExecuteChatQuery(ctx, databaseType, schema, tableDetails, previousConversation, userQuery, config, plugin)
}

// DBChatBAML is a convenience wrapper for NoSQL databases. Calls ExecuteChatQuery.
// Deprecated: Use ExecuteChatQuery directly.
func DBChatBAML(
	ctx context.Context,
	databaseType string,
	schema string,
	tableDetails string,
	previousConversation string,
	userQuery string,
	config *engine.PluginConfig,
	plugin RawExecutePlugin,
) ([]*engine.ChatMessage, error) {
	return ExecuteChatQuery(ctx, databaseType, schema, tableDetails, previousConversation, userQuery, config, plugin)
}

// convertBAMLTypeToWhoDB converts BAML ChatMessageType to WhoDB message type string
func convertBAMLTypeToWhoDB(bamlType types.ChatMessageType) string {
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

// BAMLConfigResolver resolves BAML provider string + options for a given provider type.
// Injected by the llm package at init time to use the provider registry.
type BAMLConfigResolver func(providerType, apiKey, endpoint, model string) (string, map[string]any, error)

var bamlConfigResolver BAMLConfigResolver

// RegisterBAMLConfigResolver sets the function used to resolve BAML config from the provider registry.
func RegisterBAMLConfigResolver(resolver BAMLConfigResolver) {
	bamlConfigResolver = resolver
}

// SetupAIClient creates the BAML client options for the given external model.
func SetupAIClient(externalModel *engine.ExternalModel) []baml_client.CallOptionFunc {
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
func CreateDynamicBAMLClient(externalModel *engine.ExternalModel) *baml.ClientRegistry {
	if externalModel == nil {
		return nil
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
func getBAMLProviderAndOptions(m *engine.ExternalModel) (string, map[string]any) {
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
