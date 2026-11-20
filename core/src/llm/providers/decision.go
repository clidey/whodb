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

package providers

import "context"

// DecisionProvider is implemented by providers that serve the OpenAI Decisions API.
// Callers detect support with a type assertion, so other providers need no stub.
type DecisionProvider interface {
	// DecisionModels returns the models that can answer decisions, preferred first.
	DecisionModels() []string
	// CreateDecision answers every question in the request, returning the answers in question order.
	CreateDecision(ctx context.Context, config *ProviderConfig, request *DecisionRequest) ([]DecisionAnswer, error)
}

// DecisionQuestionType is the kind of answer a decision question asks for.
type DecisionQuestionType string

// Decision question types, using the wire values of the Decisions API.
const (
	DecisionQuestionPredicate DecisionQuestionType = "predicate"
	DecisionQuestionChoice    DecisionQuestionType = "choice"
	DecisionQuestionScore     DecisionQuestionType = "score"
)

// DecisionAnswerType is the kind of answer returned for a decision question.
type DecisionAnswerType string

// Decision answer types, using the wire values of the Decisions API.
// Any question can be refused instead of answered.
const (
	DecisionAnswerPredicate DecisionAnswerType = "predicate"
	DecisionAnswerChoice    DecisionAnswerType = "choice"
	DecisionAnswerScore     DecisionAnswerType = "score"
	DecisionAnswerRefusal   DecisionAnswerType = "refusal"
)

// DecisionRequest is the body of a Decisions API request: text evidence and the
// questions to answer about it.
type DecisionRequest struct {
	Model     string             `json:"model"`
	Input     string             `json:"input"`
	Questions []DecisionQuestion `json:"questions"`
}

// DecisionQuestion is one question about the request input. Choices apply to
// choice questions and Levels, ordered from lowest to highest, to score questions.
type DecisionQuestion struct {
	Type         DecisionQuestionType `json:"type"`
	Name         string               `json:"name"`
	Instructions string               `json:"instructions"`
	Choices      []DecisionChoice     `json:"choices,omitempty"`
	Levels       []DecisionLevel      `json:"levels,omitempty"`
}

// DecisionChoice is one option a choice question can select.
type DecisionChoice struct {
	Value       string `json:"value"`
	Description string `json:"description,omitempty"`
}

// DecisionLevel is one level a score question rates against.
type DecisionLevel struct {
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
}

// DecisionAnswer is the answer to one question. Predicate answers set Probability;
// choice answers set Choice, Confidence, and ChoiceProbabilities; score answers set
// Score, Confidence, and ScoreProbabilities; refusals set only Type and Name.
type DecisionAnswer struct {
	Type        DecisionAnswerType `json:"type"`
	Name        string             `json:"name"`
	Probability float64            `json:"probability"`
	Choice      string             `json:"choice"`
	Score       float64            `json:"score"`
	Confidence  float64            `json:"confidence"`
	// The API returns both lists as "probabilities" with differently typed values,
	// so providers decode them by answer type.
	ChoiceProbabilities []DecisionChoiceProbability `json:"-"`
	ScoreProbabilities  []DecisionScoreProbability  `json:"-"`
}

// DecisionChoiceProbability is the probability given to one choice value.
type DecisionChoiceProbability struct {
	Value       string  `json:"value"`
	Probability float64 `json:"probability"`
}

// DecisionScoreProbability is the probability given to one score level, where
// Value is the level's zero-based index.
type DecisionScoreProbability struct {
	Value       int     `json:"value"`
	Label       string  `json:"label"`
	Probability float64 `json:"probability"`
}
