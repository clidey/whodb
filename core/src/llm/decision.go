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

package llm

import (
	"context"
	"errors"
	"fmt"

	"github.com/clidey/whodb/core/src/env"
	"github.com/clidey/whodb/core/src/llm/providers"
	"github.com/clidey/whodb/core/src/source"
)

// DecisionModels returns the models the provider type can answer decisions with,
// preferred first, or nil when the provider does not support decisions or the
// server has turned them off.
func DecisionModels(modelType string) []string {
	if env.DisableAIDecisions {
		return nil
	}
	provider, err := providers.GetProvider(providers.LLMType(modelType))
	if err != nil {
		return nil
	}
	decisionProvider, ok := provider.(providers.DecisionProvider)
	if !ok {
		return nil
	}
	return decisionProvider.DecisionModels()
}

// CreateDecision answers questions about input with the Decisions API of the model's
// provider and returns the answers in question order. It fails for providers that do
// not implement providers.DecisionProvider, and when the server has turned decisions off.
func CreateDecision(ctx context.Context, model *source.ExternalModel, input string, questions []providers.DecisionQuestion) ([]providers.DecisionAnswer, error) {
	if env.DisableAIDecisions {
		return nil, errors.New("decisions are turned off on this server")
	}
	provider, err := providers.GetProvider(providers.LLMType(model.Type))
	if err != nil {
		return nil, err
	}
	decisionProvider, ok := provider.(providers.DecisionProvider)
	if !ok {
		return nil, fmt.Errorf("the %s provider does not support decisions", model.Type)
	}

	config := &providers.ProviderConfig{
		Type:     providers.LLMType(model.Type),
		APIKey:   model.Token,
		Endpoint: model.Endpoint,
	}
	return decisionProvider.CreateDecision(ctx, config, &providers.DecisionRequest{
		Model:     model.Model,
		Input:     input,
		Questions: questions,
	})
}
