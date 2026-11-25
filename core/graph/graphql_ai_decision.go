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
	"fmt"

	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/src/envconfig"
	"github.com/clidey/whodb/core/src/llm"
	"github.com/clidey/whodb/core/src/llm/providers"
	"github.com/clidey/whodb/core/src/source"
)

var aiDecisionQuestionTypes = map[model.AIDecisionQuestionType]providers.DecisionQuestionType{
	model.AIDecisionQuestionTypePredicate: providers.DecisionQuestionPredicate,
	model.AIDecisionQuestionTypeChoice:    providers.DecisionQuestionChoice,
	model.AIDecisionQuestionTypeScore:     providers.DecisionQuestionScore,
}

// runAIDecision resolves the provider credentials for the input and answers its
// questions with the provider's Decisions API.
func runAIDecision(ctx context.Context, input model.AIDecisionInput) (*model.AIDecisionResponse, error) {
	creds := envconfig.ResolveProviderCredentials(stringValue(input.ProviderID), stringValue(input.Token), stringValue(input.Endpoint), input.ModelType)
	externalModel := &source.ExternalModel{
		Type:     creds.ModelType,
		Token:    creds.Token,
		Model:    input.Model,
		Endpoint: creds.Endpoint,
	}

	questions := make([]providers.DecisionQuestion, 0, len(input.Questions))
	for _, q := range input.Questions {
		question := providers.DecisionQuestion{
			Type:         aiDecisionQuestionTypes[q.Type],
			Name:         q.Name,
			Instructions: q.Instructions,
		}
		for _, choice := range q.Choices {
			question.Choices = append(question.Choices, providers.DecisionChoice{Value: choice.Value, Description: stringValue(choice.Description)})
		}
		for _, level := range q.Levels {
			question.Levels = append(question.Levels, providers.DecisionLevel{Label: level.Label, Description: stringValue(level.Description)})
		}
		questions = append(questions, question)
	}

	answers, err := llm.CreateDecision(ctx, externalModel, input.Input, questions)
	if err != nil {
		return nil, err
	}

	response := &model.AIDecisionResponse{Answers: make([]*model.AIDecisionAnswer, 0, len(answers))}
	for _, answer := range answers {
		mapped, err := aiDecisionAnswerToModel(answer)
		if err != nil {
			return nil, err
		}
		response.Answers = append(response.Answers, mapped)
	}
	return response, nil
}

// aiDecisionAnswerToModel converts a provider answer, setting only the fields that apply to its type.
func aiDecisionAnswerToModel(answer providers.DecisionAnswer) (*model.AIDecisionAnswer, error) {
	mapped := &model.AIDecisionAnswer{Name: answer.Name}
	switch answer.Type {
	case providers.DecisionAnswerPredicate:
		mapped.Type = model.AIDecisionAnswerTypePredicate
		mapped.Probability = &answer.Probability
	case providers.DecisionAnswerChoice:
		mapped.Type = model.AIDecisionAnswerTypeChoice
		mapped.Choice = &answer.Choice
		mapped.Confidence = &answer.Confidence
		mapped.ChoiceProbabilities = make([]*model.AIDecisionChoiceProbability, 0, len(answer.ChoiceProbabilities))
		for _, p := range answer.ChoiceProbabilities {
			mapped.ChoiceProbabilities = append(mapped.ChoiceProbabilities, &model.AIDecisionChoiceProbability{Value: p.Value, Probability: p.Probability})
		}
	case providers.DecisionAnswerScore:
		mapped.Type = model.AIDecisionAnswerTypeScore
		mapped.Score = &answer.Score
		mapped.Confidence = &answer.Confidence
		mapped.ScoreProbabilities = make([]*model.AIDecisionScoreProbability, 0, len(answer.ScoreProbabilities))
		for _, p := range answer.ScoreProbabilities {
			mapped.ScoreProbabilities = append(mapped.ScoreProbabilities, &model.AIDecisionScoreProbability{Value: p.Value, Label: p.Label, Probability: p.Probability})
		}
	case providers.DecisionAnswerRefusal:
		mapped.Type = model.AIDecisionAnswerTypeRefusal
	default:
		return nil, fmt.Errorf("unsupported decision answer type %q", answer.Type)
	}
	return mapped, nil
}

// stringValue returns the string s points to, or "" when s is nil.
func stringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
