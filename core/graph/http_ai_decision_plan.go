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
	stdctx "context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/clidey/whodb/core/baml_client"
	"github.com/clidey/whodb/core/baml_client/types"
	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/src/source"
	"github.com/clidey/whodb/core/src/sqlguard"
)

// chatModeDecisions selects Decisions planning instead of SQL chat for a stream request.
const chatModeDecisions = "decisions"

// maxDecisionRows is the most rows one Decisions run sends, one request per row.
const maxDecisionRows = 100

var decisionPlanQuestionTypes = map[types.DecisionQuestionKind]model.AIDecisionQuestionType{
	types.DecisionQuestionKindPREDICATE: model.AIDecisionQuestionTypePredicate,
	types.DecisionQuestionKindCHOICE:    model.AIDecisionQuestionTypeChoice,
	types.DecisionQuestionKindSCORE:     model.AIDecisionQuestionTypeScore,
}

// runDecisionPlan asks the chat model for a read-only query and typed questions, runs
// the query, and streams the plan with its rows for the user to review. The model sees
// only the schema and conversation, and nothing is sent to the Decisions API: that
// happens per row from the browser after the user approves.
func runDecisionPlan(
	ctx stdctx.Context,
	w http.ResponseWriter,
	flusher http.Flusher,
	dbContext types.DatabaseContext,
	userRequest string,
	callOpts []baml_client.CallOptionFunc,
	queryRunner source.ReadOnlyQueryRunner,
	decisionModel string,
) {
	request := userRequest
	for attempt := range 2 {
		SendSSEProgress(w, flusher, "plan", "started")
		plan, err := baml_client.PlanDecisions(ctx, dbContext, request, maxDecisionRows, callOpts...)
		if err != nil {
			if errors.Is(ctx.Err(), stdctx.DeadlineExceeded) {
				SendSSEError(w, flusher, "The chat request timed out. Please try again.")
				return
			}
			sendModelError(w, flusher, err.Error())
			return
		}
		SendSSEProgress(w, flusher, "plan", "completed")

		query := ""
		if plan.Sql != nil {
			query = strings.TrimSpace(*plan.Sql)
		}
		if query == "" && len(plan.Questions) == 0 {
			SendSSEMessage(w, flusher, &model.AIChatMessage{Type: "message", Text: plan.Message})
			SendSSEDone(w, flusher)
			return
		}

		var result *source.RowsResult
		failure := validateDecisionPlan(query, plan.Questions)
		queryFailed := false
		if failure == nil {
			SendSSEProgress(w, flusher, "query", "started")
			result, failure = queryRunner.RunReadOnlyQuery(ctx, query)
			SendSSEProgress(w, flusher, "query", "completed")
			queryFailed = failure != nil
		}
		if failure != nil {
			if attempt == 0 {
				SendSSEProgress(w, flusher, "retry", "started")
				request = userRequest + "\nThe previous plan failed: " + failure.Error() + "\nReturn a corrected plan."
				SendSSEProgress(w, flusher, "retry", "completed")
				continue
			}
			if queryFailed {
				SendSSESQLFailure(w, flusher, query, failure.Error())
			} else {
				SendSSEError(w, flusher, failure.Error())
			}
			return
		}

		SendSSEMessage(w, flusher, decisionPlanMessage(plan, query, result, decisionModel))
		SendSSEDone(w, flusher)
		return
	}
}

// validateDecisionPlan checks a plan against the Decisions API limits before its query
// runs, so no request can fail on a malformed question after the user approves.
func validateDecisionPlan(query string, questions []types.DecisionQuestionPlan) error {
	if query == "" {
		return errors.New("the plan has no query")
	}
	if classification := sqlguard.Classify(query); classification.Mutating {
		return fmt.Errorf("decisions only read data: %s", classification.Reason)
	}
	if len(questions) == 0 || len(questions) > 200 {
		return errors.New("a plan needs between 1 and 200 questions")
	}
	names := make(map[string]bool, len(questions))
	for i := range questions {
		question := &questions[i]
		name := strings.TrimSpace(question.Name)
		if name == "" {
			return errors.New("every question needs a name")
		}
		if names[name] {
			return fmt.Errorf("question names must be unique, but %q is repeated", name)
		}
		names[name] = true
		switch question.Type {
		case types.DecisionQuestionKindPREDICATE:
		case types.DecisionQuestionKindCHOICE:
			if len(question.Choices) < 2 || len(question.Choices) > 255 {
				return fmt.Errorf("choice question %q needs between 2 and 255 choices", name)
			}
			values := make(map[string]bool, len(question.Choices))
			for _, choice := range question.Choices {
				if choice.Value == "" || values[choice.Value] {
					return fmt.Errorf("choice question %q needs unique, non-empty values", name)
				}
				values[choice.Value] = true
			}
		case types.DecisionQuestionKindSCORE:
			if len(question.Levels) < 2 {
				return fmt.Errorf("score question %q needs at least 2 levels", name)
			}
			for _, level := range question.Levels {
				if level.Label == "" {
					return fmt.Errorf("score question %q needs a label for every level", name)
				}
			}
		default:
			return fmt.Errorf("question %q has an unsupported type", name)
		}
	}
	return nil
}

// decisionPlanMessage builds the chat message the user reviews. Its Text becomes
// conversation context for later turns, so it describes the plan without row values.
func decisionPlanMessage(plan types.DecisionPlan, query string, result *source.RowsResult, decisionModel string) *model.AIChatMessage {
	rows := result.Rows
	truncated := len(rows) > maxDecisionRows
	if truncated {
		rows = rows[:maxDecisionRows]
	}

	var summary strings.Builder
	fmt.Fprintf(&summary, "Decisions plan: %s\nQuery: %s\nQuestions:", plan.Message, query)
	questions := make([]*model.AIDecisionQuestion, 0, len(plan.Questions))
	for i := range plan.Questions {
		planned := &plan.Questions[i]
		question := &model.AIDecisionQuestion{
			Type:         decisionPlanQuestionTypes[planned.Type],
			Name:         strings.TrimSpace(planned.Name),
			Instructions: planned.Instructions,
		}
		var options []string
		switch planned.Type {
		case types.DecisionQuestionKindCHOICE:
			for _, choice := range planned.Choices {
				question.Choices = append(question.Choices, &model.AIDecisionChoice{Value: choice.Value, Description: stringPtr(choice.Description)})
				options = append(options, choice.Value)
			}
		case types.DecisionQuestionKindSCORE:
			for _, level := range planned.Levels {
				question.Levels = append(question.Levels, &model.AIDecisionLevel{Label: level.Label, Description: stringPtr(level.Description)})
				options = append(options, level.Label)
			}
		}
		fmt.Fprintf(&summary, "\n- %s (%s", question.Name, strings.ToLower(string(question.Type)))
		if len(options) > 0 {
			fmt.Fprintf(&summary, ": %s", strings.Join(options, ", "))
		}
		summary.WriteString(")")
		questions = append(questions, question)
	}

	return &model.AIChatMessage{
		Type:                 "decision",
		Text:                 summary.String(),
		Result:               ConvertResultToMessage(&source.RowsResult{Columns: result.Columns, Rows: rows}),
		RequiresConfirmation: len(rows) > 0,
		Decision: &model.AIChatDecision{
			Model:     decisionModel,
			Query:     query,
			Questions: questions,
			Truncated: truncated,
		},
	}
}
