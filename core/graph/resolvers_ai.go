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
	"context"
	"strings"

	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/src/auth"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/env"
	"github.com/clidey/whodb/core/src/envconfig"
	"github.com/clidey/whodb/core/src/llm"
	"github.com/clidey/whodb/core/src/log"
)

// AIProviders is the resolver for the AIProviders field.
func (r *queryResolver) AIProviders(ctx context.Context) ([]*model.AIProvider, error) {
	chatProviders := envconfig.GetConfiguredChatProviders()
	var aiProviders []*model.AIProvider
	for _, provider := range chatProviders {
		aiProviders = append(aiProviders, &model.AIProvider{
			Type:                 provider.Type,
			Name:                 provider.Name,
			ProviderID:           provider.ProviderId,
			IsEnvironmentDefined: true,
			IsGeneric:            provider.IsGeneric,
		})
	}
	return aiProviders, nil
}

// AIModel is the resolver for the AIModel field.
func (r *queryResolver) AIModel(ctx context.Context, providerID *string, modelType string, token *string) ([]string, error) {
	config := engine.NewPluginConfig(auth.GetCredentials(ctx))

	// Initialize ExternalModel to prevent nil pointer dereference
	config.ExternalModel = &engine.ExternalModel{
		Type: modelType,
	}

	if providerID != nil {
		// Try to find provider in environment-defined providers first
		chatProviders := envconfig.GetConfiguredChatProviders()
		found := false
		for _, provider := range chatProviders {
			if provider.ProviderId == *providerID {
				config.ExternalModel.Token = provider.APIKey
				found = true

				// For generic providers, return models from config instead of querying registry
				if provider.IsGeneric {
					for _, genericProvider := range env.GenericProviders {
						if genericProvider.ProviderId == *providerID {
							return genericProvider.Models, nil
						}
					}
				}
				break
			}
		}
		// If provider not found in environment but token is provided, use the token
		// This handles user-added providers that aren't in environment
		if !found && token != nil {
			config.ExternalModel.Token = *token
		}
	} else if token != nil {
		config.ExternalModel.Token = *token
	}
	models, err := llm.Instance(config).GetSupportedModels()
	if err != nil {
		log.WithFields(log.Fields{
			"operation":   "GetSupportedModels",
			"model_type":  modelType,
			"provider_id": providerID,
			"error":       err.Error(),
		}).Error("AI operation failed")
		return nil, err
	}
	return models, nil
}

// AIChat is the resolver for the AIChat field.
func (r *queryResolver) AIChat(ctx context.Context, providerID *string, modelType string, token *string, schema string, input model.ChatInput) ([]*model.AIChatMessage, error) {
	plugin, config := GetPluginForContext(ctx)
	typeArg := config.Credentials.Type
	providerId := ""
	if providerID != nil {
		providerId = *providerID
	}
	requestToken := ""
	if token != nil {
		requestToken = *token
	}
	creds := envconfig.ResolveProviderCredentials(providerId, requestToken, "", modelType)
	config.ExternalModel = &engine.ExternalModel{
		Type:     creds.ModelType,
		Token:    creds.Token,
		Model:    input.Model,
		Endpoint: creds.Endpoint,
	}
	messages, err := plugin.Chat(config, schema, input.PreviousConversation, input.Query)

	if err != nil {
		log.WithFields(log.Fields{
			"operation":     "Chat",
			"schema":        schema,
			"database_type": typeArg,
			"model":         input.Model,
			"model_type":    modelType,
			"provider_id":   providerID,
			"query":         input.Query,
			"error":         err.Error(),
		}).Error("AI chat operation failed")
		return nil, err
	}

	var chatResponse []*model.AIChatMessage

	for _, message := range messages {
		var result *model.RowsResult
		if strings.HasPrefix(message.Type, "sql") {
			var columns []*model.Column
			for _, column := range message.Result.Columns {
				columns = append(columns, &model.Column{
					Type: column.Type,
					Name: column.Name,
				})
			}
			result = &model.RowsResult{
				Columns: columns,
				Rows:    message.Result.Rows,
			}
		}
		chatResponse = append(chatResponse, &model.AIChatMessage{
			Type:                 message.Type,
			Result:               result,
			Text:                 message.Text,
			RequiresConfirmation: message.RequiresConfirmation,
		})
	}

	return chatResponse, nil
}

// ExecuteConfirmedSQL is the resolver for the ExecuteConfirmedSQL field.
func (r *mutationResolver) ExecuteConfirmedSQL(ctx context.Context, query string, operationType string) (*model.AIChatMessage, error) {
	// Get plugin and config from context
	plugin, config := GetPluginForContext(ctx)

	// Execute the SQL query
	result, execErr := plugin.RawExecute(config, query)

	message := &model.AIChatMessage{
		Type:                 operationType,
		Text:                 query,
		RequiresConfirmation: false,
	}

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
	}

	return message, nil
}

// GenerateChatTitle is the resolver for the GenerateChatTitle field.
func (r *mutationResolver) GenerateChatTitle(ctx context.Context, input model.GenerateChatTitleInput) (*model.GenerateChatTitleResponse, error) {
	return generateChatTitleImpl(ctx, input)
}
