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
	"fmt"
	"strings"

	"github.com/clidey/whodb/core/baml_client"
	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/env"
	"github.com/clidey/whodb/core/src/log"
)

// generateChatTitleImpl generates a chat title using BAML
func generateChatTitleImpl(c ctx.Context, input model.GenerateChatTitleInput) (*model.GenerateChatTitleResponse, error) {
	log.DebugFileAlways("Generate Chat Title: Started with query=%s, model=%s", input.Query, input.Model)

	// Handle very short or unclear inputs - return empty to keep default name
	query := strings.TrimSpace(input.Query)
	queryLower := strings.ToLower(query)

	// For very short inputs, greetings, or unclear queries, skip title generation
	// This signals the frontend to keep the default "Chat X" name
	if len(query) <= 3 ||
	   queryLower == "hi" ||
	   queryLower == "hello" ||
	   queryLower == "hey" ||
	   queryLower == "hi!" ||
	   queryLower == "hello!" ||
	   queryLower == "hey!" ||
	   queryLower == "blah" ||
	   queryLower == "test" {
		return &model.GenerateChatTitleResponse{
			Title: "", // Empty title means "keep default"
		}, nil
	}

	// Build token lookup if providerId is set
	token := ""
	if input.Token != nil {
		token = *input.Token
	}
	endpoint := ""
	if input.Endpoint != nil {
		endpoint = *input.Endpoint
	}

	if input.ProviderID != nil && *input.ProviderID != "" && token == "" {
		for _, provider := range env.GetConfiguredChatProviders() {
			if provider.ProviderId == *input.ProviderID {
				token = provider.APIKey
				if endpoint == "" {
					endpoint = provider.Endpoint
				}
				break
			}
		}
	}

	// Build ExternalModel for BAML
	externalModel := &engine.ExternalModel{
		Type:     input.ModelType,
		Token:    token,
		Model:    input.Model,
		Endpoint: endpoint,
	}

	// Create the prompt for title generation
	titlePrompt := fmt.Sprintf(
		"Generate a very short title (maximum 5 words) for a chat conversation that starts with this question: \"%s\"\n\n"+
			"Return ONLY the title text, nothing else. No quotes, no explanations. The title should be concise and descriptive.",
		input.Query,
	)

	log.DebugFileAlways("Generate Chat Title: Calling BAML GenerateChatTitle")

	// Setup BAML context and call
	callOpts := common.SetupAIClientWithLogging(externalModel)
	stream, err := baml_client.Stream.GenerateChatTitle(c, titlePrompt, callOpts...)
	if err != nil {
		log.DebugFileAlways("Generate Chat Title: BAML call failed: %v", err)
		return nil, fmt.Errorf("failed to generate title: %w", err)
	}

	// Wait for the response
	var title string
	for chunk := range stream {
		if chunk.IsError {
			log.DebugFileAlways("Generate Chat Title: Stream error: %v", chunk.Error)
			return nil, fmt.Errorf("failed to generate title: %w", chunk.Error)
		}
		if chunk.IsFinal {
			finalTitle := chunk.Final()
			if finalTitle != nil {
				title = *finalTitle
			}
			break
		}
	}

	// Clean up the title (remove quotes, trim, limit length)
	title = strings.Trim(title, `"'`)
	title = strings.TrimSpace(title)
	if len(title) > 50 {
		title = title[:50]
	}

	log.DebugFileAlways("Generate Chat Title: Generated title=%s", title)

	return &model.GenerateChatTitleResponse{
		Title: title,
	}, nil
}
