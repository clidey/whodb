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
	"strings"

	baml "github.com/boundaryml/baml/engine/language_client_go/pkg"
	"github.com/clidey/whodb/core/baml_client"
	"github.com/clidey/whodb/core/baml_client/types"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
)

// RawExecutePlugin defines the interface for executing raw SQL queries
type RawExecutePlugin interface {
	RawExecute(config *engine.PluginConfig, query string) (*engine.GetRowsResult, error)
}

// SQLChatBAML generates SQL queries using BAML for structured prompt engineering
// This replaces the old string-based prompt and JSON parsing approach
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

	// Build BAML context
	dbContext := types.DatabaseContext{
		Database_type:         databaseType,
		Schema:                schema,
		Tables_and_fields:     tableDetails,
		Previous_conversation: previousConversation,
	}

	// Create dynamic BAML client and log request
	callOpts := SetupAIClientWithLogging(config.ExternalModel)

	// Call BAML function to generate SQL
	responses, err := baml_client.GenerateSQLQuery(ctx, dbContext, userQuery, callOpts...)
	if err != nil {
		return nil, err
	}

	// Convert BAML responses to WhoDB ChatMessage format
	var chatMessages []*engine.ChatMessage
	for _, bamlResp := range responses {
		message := &engine.ChatMessage{
			Type:                 string(bamlResp.Type),
			Text:                 bamlResp.Text,
			Result:               &engine.GetRowsResult{},
			RequiresConfirmation: false,
		}

		// Convert BAML type to WhoDB type format
		message.Type = convertBAMLTypeToWhoDB(bamlResp.Type)

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
					// Set operation-specific type
					message.Type = convertOperationType(*bamlResp.Operation)
				}
				message.Result = result
			}
		}

		chatMessages = append(chatMessages, message)
	}

	return chatMessages, nil
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

// SetupAIClientWithLogging creates the BAML client options and logs the AI request configuration.
// This should be used by all AI request paths to ensure consistent logging.
func SetupAIClientWithLogging(externalModel *engine.ExternalModel) []baml_client.CallOptionFunc {
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

	// Log AI model configuration
	fields := log.Fields{
		"provider": externalModel.Type,
		"model":    externalModel.Model,
	}
	if externalModel.Endpoint != "" {
		fields["endpoint"] = externalModel.Endpoint
	}
	log.WithFields(fields).Info("AI chat request")

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

// getBAMLProviderAndOptions maps WhoDB ExternalModel to BAML provider string and options
func getBAMLProviderAndOptions(m *engine.ExternalModel) (string, map[string]any) {
	opts := map[string]any{
		"model": m.Model,
	}

	switch m.Type {
	case "OpenAI":
		if m.Token != "" {
			opts["api_key"] = m.Token
		}
		// Use custom endpoint if provided, otherwise use default
		if m.Endpoint != "" {
			opts["base_url"] = m.Endpoint
		}
		return "openai", opts

	case "Anthropic":
		if m.Token != "" {
			opts["api_key"] = m.Token
		}
		// Use custom endpoint if provided, otherwise use default
		if m.Endpoint != "" {
			opts["base_url"] = m.Endpoint
		}
		return "anthropic", opts

	case "Ollama":
		// Ollama uses openai-generic provider with special options
		endpoint := m.Endpoint
		// Strip trailing slash and /api suffix (Ollama native API path).
		// BAML needs the OpenAI-compatible /v1 path, not the native /api path.
		endpoint = strings.TrimRight(endpoint, "/")
		endpoint = strings.TrimSuffix(endpoint, "/api")
		if !strings.HasSuffix(endpoint, "/v1") {
			endpoint = endpoint + "/v1"
		}
		opts["base_url"] = endpoint
		opts["default_role"] = "user"           // Ollama prefers user role
		opts["request_timeout_ms"] = int(60000) // 60 seconds for local inference
		return "openai-generic", opts

	case "OpenAI-Compatible":
		// Generic OpenAI-compatible endpoint
		if m.Endpoint != "" {
			opts["base_url"] = m.Endpoint
		}
		if m.Token != "" {
			opts["api_key"] = m.Token
		}
		return "openai-generic", opts

	default:
		// Standard generic provider (OpenAI-compatible)
		if m.Endpoint != "" {
			opts["base_url"] = m.Endpoint
		}
		if m.Token != "" {
			opts["api_key"] = m.Token
		}
		return "openai-generic", opts
	}
}
