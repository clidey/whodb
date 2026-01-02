/*
 * Copyright 2025 Clidey, Inc.
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
	"github.com/clidey/whodb/core/src/log"
)

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

	// Determine which BAML client to use based on ExternalModel config
	var callOpts []baml_client.CallOptionFunc
	if config.ExternalModel != nil {
		clientName, model, err := getBAMLClientFromConfig(config.ExternalModel)
		if err != nil {
			log.Logger.WithError(err).Warnf("Failed to get BAML client from config, using default")
		} else if clientName != "" {
			// Create a client registry to override the client at runtime
			registry, err := createClientRegistry(clientName, model, config.ExternalModel.Token)
			if err != nil {
				log.Logger.WithError(err).Warnf("Failed to create client registry, using default")
			} else {
				callOpts = append(callOpts, baml_client.WithClientRegistry(registry))
			}
		}
	}

	// Call BAML function to generate SQL
	responses, err := baml_client.GenerateSQLQuery(ctx, dbContext, userQuery, callOpts...)
	if err != nil {
		return nil, err
	}

	// Convert BAML responses to WhoDB ChatMessage format
	var chatMessages []*engine.ChatMessage
	for _, bamlResp := range responses {
		message := &engine.ChatMessage{
			Type:   string(bamlResp.Type),
			Text:   bamlResp.Text,
			Result: &engine.GetRowsResult{},
		}

		// Convert BAML type to WhoDB type format
		message.Type = convertBAMLTypeToWhoDB(bamlResp.Type)

		// Execute SQL if it's a query
		if bamlResp.Type == types.ChatMessageTypeSQL && bamlResp.Operation != nil {
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
	case types.OperationTypeLINE_CHART:
		return "sql:line-chart"
	case types.OperationTypePIE_CHART:
		return "sql:pie-chart"
	case types.OperationTypeTEXT:
		return "text"
	default:
		return "sql"
	}
}

// getBAMLClientFromConfig maps WhoDB ExternalModel to BAML client name and model
func getBAMLClientFromConfig(externalModel *engine.ExternalModel) (clientName string, model string, err error) {
	if externalModel == nil {
		return "", "", nil
	}

	// Map WhoDB model type to BAML client name
	switch externalModel.Type {
	case "Ollama":
		return "CustomOllama", "", nil
	case "ChatGPT":
		return "CustomGPT5", "", nil
	case "Anthropic":
		return "CustomSonnet4", "", nil
	case "OpenAI-Compatible":
		return "CustomOllama", "", nil // Use Ollama client for OpenAI-compatible APIs
	default:
		// Default to Ollama if type not recognized
		return "CustomOllama", "", nil
	}
}

// createClientRegistry creates a BAML ClientRegistry with the specified client and model
func createClientRegistry(clientName string, model string, apiKey string) (*baml.ClientRegistry, error) {
	registry := baml.NewClientRegistry()

	// Set the primary client for all BAML function calls
	registry.SetPrimaryClient(clientName)

	// If we have a specific model or API key, we could add a dynamic client here
	// For now, we rely on the pre-configured clients in clients.baml

	return registry, nil
}
