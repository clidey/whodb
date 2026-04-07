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
	"fmt"
	"strings"

	"github.com/clidey/whodb/core/baml_client"
	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/src/bamlconfig"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/envconfig"
	"github.com/clidey/whodb/core/src/log"
)

// GenerateChatTitleFunc is the function signature for chat title generation.
type GenerateChatTitleFunc func(c ctx.Context, input model.GenerateChatTitleInput) (*model.GenerateChatTitleResponse, error)

// registeredGenerateChatTitle allows overriding the default implementation.
var registeredGenerateChatTitle GenerateChatTitleFunc

// RegisterGenerateChatTitle allows registering a custom chat title implementation.
func RegisterGenerateChatTitle(fn GenerateChatTitleFunc) {
	registeredGenerateChatTitle = fn
}

func init() {
	// Register the default implementation
	registeredGenerateChatTitle = ceGenerateChatTitle
}

// generateChatTitleImpl delegates to the registered implementation.
func generateChatTitleImpl(c ctx.Context, input model.GenerateChatTitleInput) (*model.GenerateChatTitleResponse, error) {
	if registeredGenerateChatTitle != nil {
		return registeredGenerateChatTitle(c, input)
	}
	return nil, fmt.Errorf("chat title generation not available")
}

// ceGenerateChatTitle generates a chat title using BAML
func ceGenerateChatTitle(c ctx.Context, input model.GenerateChatTitleInput) (*model.GenerateChatTitleResponse, error) {
	log.Debugf("Generate Chat Title: Started with query=%s, model=%s", input.Query, input.Model)

	// Handle very short or unclear inputs - return empty to keep default name
	query := strings.TrimSpace(input.Query)
	queryLower := strings.ToLower(query)

	// For very short inputs, greetings, or unclear queries, skip title generation
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
			Title: "",
		}, nil
	}

	// Resolve credentials from environment if providerId is set
	providerId := ""
	if input.ProviderID != nil {
		providerId = *input.ProviderID
	}
	requestToken := ""
	if input.Token != nil {
		requestToken = *input.Token
	}
	requestEndpoint := ""
	if input.Endpoint != nil {
		requestEndpoint = *input.Endpoint
	}
	creds := envconfig.ResolveProviderCredentials(providerId, requestToken, requestEndpoint, input.ModelType)

	externalModel := &engine.ExternalModel{
		Type:     creds.ModelType,
		Token:    creds.Token,
		Model:    input.Model,
		Endpoint: creds.Endpoint,
	}

	titlePrompt := fmt.Sprintf(
		"Generate a very short title (maximum 5 words) for a chat conversation that starts with this question: \"%s\"\n\n"+
			"Return ONLY the title text, nothing else. No quotes, no explanations. The title should be concise and descriptive.",
		input.Query,
	)

	log.Debugf("Generate Chat Title: Calling BAML GenerateChatTitle")

	callOpts := bamlconfig.SetupAIClient(externalModel)
	stream, err := baml_client.Stream.GenerateChatTitle(c, titlePrompt, callOpts...)
	if err != nil {
		log.Debugf("Generate Chat Title: BAML call failed: %v", err)
		return nil, fmt.Errorf("failed to generate title: %w", err)
	}

	var title string
	for chunk := range stream {
		if chunk.IsError {
			log.Debugf("Generate Chat Title: Stream error: %v", chunk.Error)
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

	title = strings.Trim(title, `"'`)
	title = strings.TrimSpace(title)
	if len(title) > 50 {
		title = title[:50]
	}

	log.Debugf("Generate Chat Title: Generated title=%s", title)

	return &model.GenerateChatTitleResponse{
		Title: title,
	}, nil
}
