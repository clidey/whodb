/*
 * Copyright 2026 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 */

package mcp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/clidey/whodb/cli/internal/config"
	platformapi "github.com/clidey/whodb/cli/internal/platform"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const platformWorkflowVersion = 1

var (
	platformWorkflowMutex sync.Mutex
	platformWorkflowPath  = defaultPlatformWorkflowPath
)

// PlatformWorkflowStepInput describes one validated generic platform write in a workflow.
type PlatformWorkflowStepInput struct {
	ID        string         `json:"id" jsonschema:"Stable step id used for dependencies and retries."`
	Operation string         `json:"operation" jsonschema:"create, update, delete, or action"`
	Resource  string         `json:"resource" jsonschema:"Platform resource such as dataset, transform, function, or app"`
	Action    string         `json:"action,omitempty" jsonschema:"Action name when operation is action"`
	TargetID  string         `json:"target_id,omitempty" jsonschema:"Existing resource id for update, delete, or action"`
	Payload   map[string]any `json:"payload,omitempty" jsonschema:"Structured non-secret mutation payload"`
	DependsOn []string       `json:"depends_on,omitempty" jsonschema:"Step ids that must complete first"`
}

// PlatformWorkflowPlanInput creates a persisted plan without executing it.
type PlatformWorkflowPlanInput struct {
	Goal  string                      `json:"goal" jsonschema:"Desired end state for this hosted platform workflow"`
	Steps []PlatformWorkflowStepInput `json:"steps" jsonschema:"Ordered hosted platform writes to validate and execute"`
}

// PlatformWorkflowApplyInput applies or resumes a persisted workflow plan.
type PlatformWorkflowApplyInput struct {
	PlanID string `json:"plan_id" jsonschema:"Workflow plan id returned by whodb_platform_workflow_plan"`
}

// PlatformWorkflowGetInput identifies one persisted workflow plan.
type PlatformWorkflowGetInput struct {
	PlanID string `json:"plan_id" jsonschema:"Workflow plan id"`
}

// PlatformWorkflowListInput lists persisted workflow plans for the current host and workspace.
type PlatformWorkflowListInput struct{}

type platformWorkflowStep struct {
	ID        string         `json:"id"`
	Operation string         `json:"operation"`
	Resource  string         `json:"resource"`
	Action    string         `json:"action,omitempty"`
	TargetID  string         `json:"target_id,omitempty"`
	Payload   map[string]any `json:"payload,omitempty"`
	DependsOn []string       `json:"depends_on,omitempty"`
	Status    string         `json:"status"`
	ResultID  string         `json:"result_id,omitempty"`
	Error     string         `json:"error,omitempty"`
	Changes   []string       `json:"changes,omitempty"`
}

type platformWorkflowPlan struct {
	Version   int                    `json:"version"`
	ID        string                 `json:"id"`
	Hash      string                 `json:"hash"`
	Goal      string                 `json:"goal"`
	Host      string                 `json:"host"`
	OrgID     string                 `json:"org_id"`
	ProjectID string                 `json:"project_id"`
	Project   string                 `json:"project_name,omitempty"`
	Status    string                 `json:"status"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
	Steps     []platformWorkflowStep `json:"steps"`
}

// PlatformWorkflowOutput is the compact, non-secret workflow response.
type PlatformWorkflowOutput struct {
	PlatformSetupGuidance
	ConfirmationRequired bool             `json:"confirmation_required,omitempty"`
	ConfirmationToken    string           `json:"confirmation_token,omitempty"`
	ConfirmationExpiry   string           `json:"confirmation_expiry,omitempty"`
	Plan                 map[string]any   `json:"plan,omitempty"`
	Plans                []map[string]any `json:"plans,omitempty"`
	Status               string           `json:"status,omitempty"`
	Message              string           `json:"message,omitempty"`
	Error                string           `json:"error,omitempty"`
	ErrorCode            string           `json:"error_code,omitempty"`
	Retryable            bool             `json:"retryable,omitempty"`
	SuggestedTools       []string         `json:"suggested_tools,omitempty"`
	RequestID            string           `json:"request_id,omitempty"`
}

func platformWorkflowError(err error, requestID string) PlatformWorkflowOutput {
	errorCode, retryable, suggestedTools := platformErrorFields(err)
	return PlatformWorkflowOutput{Error: err.Error(), ErrorCode: errorCode, Retryable: retryable, SuggestedTools: suggestedTools, RequestID: requestID}
}

func defaultPlatformWorkflowPath() (string, error) {
	dir, err := config.GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "platform-workflows.json"), nil
}

func loadPlatformWorkflowPlans() ([]platformWorkflowPlan, error) {
	path, err := platformWorkflowPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return []platformWorkflowPlan{}, nil
	}
	if err != nil {
		return nil, err
	}
	var plans []platformWorkflowPlan
	if err := json.Unmarshal(data, &plans); err != nil {
		return nil, fmt.Errorf("read platform workflow state: %w", err)
	}
	return plans, nil
}

func savePlatformWorkflowPlans(plans []platformWorkflowPlan) error {
	path, err := platformWorkflowPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(plans, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".platform-workflows-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

func redactWorkflowPayload(payload map[string]any) (map[string]any, error) {
	copyPayload := cloneWorkflowMap(payload)
	if workflowPayloadContainsSecret(copyPayload) {
		return nil, errors.New("workflow payloads cannot contain secrets; execute secret writes directly so credentials are not persisted")
	}
	return copyPayload, nil
}

func cloneWorkflowMap(input map[string]any) map[string]any {
	if input == nil {
		return map[string]any{}
	}
	data, _ := json.Marshal(input)
	var output map[string]any
	if json.Unmarshal(data, &output) != nil || output == nil {
		return map[string]any{}
	}
	return output
}

func workflowPayloadContainsSecret(value any) bool {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			if sensitivePlatformWriteKey(key) {
				return true
			}
			if workflowPayloadContainsSecret(child) {
				return true
			}
		}
	case []any:
		for _, child := range typed {
			if workflowPayloadContainsSecret(child) {
				return true
			}
		}
	}
	return false
}

func validatePlatformWorkflowInput(input PlatformWorkflowPlanInput, session *platformToolSession) (platformWorkflowPlan, error) {
	if strings.TrimSpace(input.Goal) == "" {
		return platformWorkflowPlan{}, errors.New("goal is required")
	}
	if len(input.Steps) == 0 {
		return platformWorkflowPlan{}, errors.New("at least one workflow step is required")
	}
	plan := platformWorkflowPlan{
		Version:   platformWorkflowVersion,
		ID:        generateRequestID("workflow"),
		Goal:      strings.TrimSpace(input.Goal),
		Host:      session.Host.URL,
		OrgID:     session.Host.DefaultOrgID,
		ProjectID: session.Host.DefaultProjectID,
		Project:   session.Host.DefaultProjectName,
		Status:    "planned",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		Steps:     make([]platformWorkflowStep, 0, len(input.Steps)),
	}
	ids := map[string]struct{}{}
	for index, stepInput := range input.Steps {
		id := strings.TrimSpace(stepInput.ID)
		if id == "" {
			id = fmt.Sprintf("step-%d", index+1)
		}
		if _, exists := ids[id]; exists {
			return platformWorkflowPlan{}, fmt.Errorf("duplicate workflow step id %q", id)
		}
		ids[id] = struct{}{}
		operation := normalizePlatformWriteToken(stepInput.Operation)
		if operation != "create" && operation != "update" && operation != "delete" && operation != "action" {
			return platformWorkflowPlan{}, fmt.Errorf("step %q has unsupported operation %q", id, stepInput.Operation)
		}
		resource := normalizePlatformWriteToken(stepInput.Resource)
		action := normalizePlatformWriteToken(stepInput.Action)
		key := operation + ":" + resource

		if operation == "action" {
			if action == "" {
				return platformWorkflowPlan{}, fmt.Errorf("step %q requires action", id)
			}
			key = "action:" + action + ":" + resource
		}
		if _, ok := platformapi.GenericWriteSpecs[key]; !ok {
			return platformWorkflowPlan{}, fmt.Errorf("step %q uses unsupported platform write %q", id, key)
		}
		payload, err := redactWorkflowPayload(stepInput.Payload)
		if err != nil {
			return platformWorkflowPlan{}, fmt.Errorf("step %q: %w", id, err)
		}
		if operation != "create" && strings.TrimSpace(stepInput.TargetID) == "" {
			return platformWorkflowPlan{}, fmt.Errorf("step %q requires target_id", id)
		}
		if _, _, err := buildPlatformGenericWrite(session, PlatformGenericWriteInput{Resource: resource, Action: action, ID: strings.TrimSpace(stepInput.TargetID), Payload: payload}, operation); err != nil {
			return platformWorkflowPlan{}, fmt.Errorf("step %q: %w", id, err)
		}
		if err := validateWorkflowPayloadShape(key, payload); err != nil {
			return platformWorkflowPlan{}, fmt.Errorf("step %q: %w", id, err)
		}
		changes := genericWriteChanges(payload)
		plan.Steps = append(plan.Steps, platformWorkflowStep{ID: id, Operation: operation, Resource: resource, Action: action, TargetID: strings.TrimSpace(stepInput.TargetID), Payload: payload, DependsOn: append([]string(nil), stepInput.DependsOn...), Status: "pending", Changes: changes})
	}
	for _, step := range plan.Steps {
		for _, dependency := range step.DependsOn {
			if _, ok := ids[strings.TrimSpace(dependency)]; !ok {
				return platformWorkflowPlan{}, fmt.Errorf("step %q depends on unknown step %q", step.ID, dependency)
			}
			if dependency == step.ID {
				return platformWorkflowPlan{}, fmt.Errorf("step %q cannot depend on itself", step.ID)
			}
		}
	}
	if err := validateWorkflowDAG(plan.Steps); err != nil {
		return platformWorkflowPlan{}, err
	}
	plan.Hash = hashWorkflowPlan(plan)
	return plan, nil
}

func validateWorkflowPayloadShape(key string, payload map[string]any) error {
	shape, ok := platformapi.PayloadShapes[key]
	if !ok {
		return nil
	}
	for _, field := range shape.Fields {
		if !field.Required {
			continue
		}
		value, exists := payload[field.Name]
		if !exists || value == nil || (field.Type == "string" && strings.TrimSpace(fmt.Sprint(value)) == "") {
			return fmt.Errorf("payload.%s is required", field.Name)
		}
	}
	return nil
}

func validateWorkflowDAG(steps []platformWorkflowStep) error {
	state := map[string]int{}
	byID := make(map[string]platformWorkflowStep, len(steps))
	for _, step := range steps {
		byID[step.ID] = step
	}
	var visit func(string) error
	visit = func(id string) error {
		if state[id] == 1 {
			return fmt.Errorf("workflow dependency cycle includes step %q", id)
		}
		if state[id] == 2 {
			return nil
		}
		state[id] = 1
		for _, dep := range byID[id].DependsOn {
			if err := visit(strings.TrimSpace(dep)); err != nil {
				return err
			}
		}
		state[id] = 2
		return nil
	}
	for _, step := range steps {
		if err := visit(step.ID); err != nil {
			return err
		}
	}
	return nil
}

func hashWorkflowPlan(plan platformWorkflowPlan) string {
	copyPlan := plan
	copyPlan.ID, copyPlan.Hash, copyPlan.Status = "", "", ""
	copyPlan.CreatedAt, copyPlan.UpdatedAt = time.Time{}, time.Time{}
	data, _ := json.Marshal(copyPlan)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func workflowOutputPlan(plan platformWorkflowPlan) map[string]any {
	steps := make([]map[string]any, 0, len(plan.Steps))
	for _, step := range plan.Steps {
		entry := map[string]any{"id": step.ID, "operation": step.Operation, "resource": step.Resource, "status": step.Status, "changes": step.Changes}
		if step.Action != "" {
			entry["action"] = step.Action
		}
		if step.TargetID != "" {
			entry["target_id"] = step.TargetID
		}
		if len(step.DependsOn) > 0 {
			entry["depends_on"] = step.DependsOn
		}
		if step.ResultID != "" {
			entry["result_id"] = step.ResultID
		}
		if step.Error != "" {
			entry["error"] = step.Error
		}
		steps = append(steps, entry)
	}
	return map[string]any{"id": plan.ID, "hash": plan.Hash, "goal": plan.Goal, "host": plan.Host, "org_id": plan.OrgID, "project_id": plan.ProjectID, "project_name": plan.Project, "status": plan.Status, "created_at": plan.CreatedAt, "updated_at": plan.UpdatedAt, "steps": steps}
}

func findPlatformWorkflow(plans []platformWorkflowPlan, id string) (*platformWorkflowPlan, int, error) {
	for index := range plans {
		if plans[index].ID == id {
			return &plans[index], index, nil
		}
	}
	return nil, -1, fmt.Errorf("workflow plan %q not found", id)
}

func createPlatformWorkflowHandler() func(context.Context, *mcp.CallToolRequest, PlatformWorkflowPlanInput) (*mcp.CallToolResult, PlatformWorkflowOutput, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input PlatformWorkflowPlanInput) (*mcp.CallToolResult, PlatformWorkflowOutput, error) {
		return HandlePlatformWorkflowPlan(ctx, req, input)
	}
}

func createPlatformWorkflowGetHandler() func(context.Context, *mcp.CallToolRequest, PlatformWorkflowGetInput) (*mcp.CallToolResult, PlatformWorkflowOutput, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input PlatformWorkflowGetInput) (*mcp.CallToolResult, PlatformWorkflowOutput, error) {
		return HandlePlatformWorkflowGet(ctx, req, input)
	}
}

func createPlatformWorkflowListHandler() func(context.Context, *mcp.CallToolRequest, PlatformWorkflowListInput) (*mcp.CallToolResult, PlatformWorkflowOutput, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input PlatformWorkflowListInput) (*mcp.CallToolResult, PlatformWorkflowOutput, error) {
		return HandlePlatformWorkflowList(ctx, req, input)
	}
}

func registerPlatformWorkflowTool(server *mcp.Server, tool *mcp.Tool, secOpts *SecurityOptions) bool {
	switch tool.Name {
	case "whodb_platform_workflow_plan":
		mcp.AddTool(server, tool, createPlatformWorkflowHandler())
	case "whodb_platform_workflow_get":
		mcp.AddTool(server, tool, createPlatformWorkflowGetHandler())
	case "whodb_platform_workflow_list":
		mcp.AddTool(server, tool, createPlatformWorkflowListHandler())
	case "whodb_platform_workflow_apply":
		if !secOpts.ReadOnly {
			mcp.AddTool(server, tool, createPlatformWorkflowApplyHandlerWithConfirmation(secOpts.ConfirmWrites))
		}
	default:
		return false
	}
	return true
}

func createPlatformWorkflowApplyHandlerWithConfirmation(confirmWrites bool) func(context.Context, *mcp.CallToolRequest, PlatformWorkflowApplyInput) (*mcp.CallToolResult, PlatformWorkflowOutput, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input PlatformWorkflowApplyInput) (*mcp.CallToolResult, PlatformWorkflowOutput, error) {
		return handlePlatformWorkflowApply(ctx, req, input, confirmWrites)
	}
}

func HandlePlatformWorkflowPlan(ctx context.Context, req *mcp.CallToolRequest, input PlatformWorkflowPlanInput) (*mcp.CallToolResult, PlatformWorkflowOutput, error) {
	requestID := generateRequestID("platform_workflow_plan")
	session, err := loadPlatformWorkspace(ctx)
	if err != nil {
		return nil, PlatformWorkflowOutput{PlatformSetupGuidance: platformSetupGuidanceForCurrentConfig(requestID), Error: err.Error(), RequestID: requestID}, nil
	}
	plan, err := validatePlatformWorkflowInput(input, session)
	if err != nil {
		return nil, platformWorkflowError(err, requestID), nil
	}
	platformWorkflowMutex.Lock()
	defer platformWorkflowMutex.Unlock()
	plans, err := loadPlatformWorkflowPlans()
	if err != nil {
		return nil, platformWorkflowError(err, requestID), nil
	}
	plans = append(plans, plan)
	if err := savePlatformWorkflowPlans(plans); err != nil {
		return nil, platformWorkflowError(err, requestID), nil
	}
	return nil, PlatformWorkflowOutput{Plan: workflowOutputPlan(plan), Status: "planned", Message: "Workflow plan created. Review it, then apply it when the user approves.", RequestID: requestID}, nil
}

func HandlePlatformWorkflowGet(ctx context.Context, req *mcp.CallToolRequest, input PlatformWorkflowGetInput) (*mcp.CallToolResult, PlatformWorkflowOutput, error) {
	requestID := generateRequestID("platform_workflow_get")
	if strings.TrimSpace(input.PlanID) == "" {
		return nil, PlatformWorkflowOutput{Error: "plan_id is required", RequestID: requestID}, nil
	}
	platformWorkflowMutex.Lock()
	defer platformWorkflowMutex.Unlock()
	plans, err := loadPlatformWorkflowPlans()
	if err != nil {
		return nil, platformWorkflowError(err, requestID), nil
	}
	plan, _, err := findPlatformWorkflow(plans, input.PlanID)
	if err != nil {
		return nil, PlatformWorkflowOutput{Error: err.Error(), RequestID: requestID}, nil
	}
	return nil, PlatformWorkflowOutput{Plan: workflowOutputPlan(*plan), Status: plan.Status, RequestID: requestID}, nil
}

func HandlePlatformWorkflowList(ctx context.Context, req *mcp.CallToolRequest, input PlatformWorkflowListInput) (*mcp.CallToolResult, PlatformWorkflowOutput, error) {
	requestID := generateRequestID("platform_workflow_list")
	session, err := loadPlatformWorkspace(ctx)
	if err != nil {
		return nil, PlatformWorkflowOutput{PlatformSetupGuidance: platformSetupGuidanceForCurrentConfig(requestID), Error: err.Error(), RequestID: requestID}, nil
	}
	platformWorkflowMutex.Lock()
	defer platformWorkflowMutex.Unlock()
	plans, err := loadPlatformWorkflowPlans()
	if err != nil {
		return nil, PlatformWorkflowOutput{Error: err.Error(), RequestID: requestID}, nil
	}
	items := make([]map[string]any, 0)
	for _, plan := range plans {
		if plan.Host == session.Host.URL && plan.OrgID == session.Host.DefaultOrgID && plan.ProjectID == session.Host.DefaultProjectID {
			items = append(items, workflowOutputPlan(plan))
		}
	}
	return nil, PlatformWorkflowOutput{Plans: items, Status: "ok", RequestID: requestID}, nil
}

func HandlePlatformWorkflowApply(ctx context.Context, req *mcp.CallToolRequest, input PlatformWorkflowApplyInput) (*mcp.CallToolResult, PlatformWorkflowOutput, error) {
	return handlePlatformWorkflowApply(ctx, req, input, true)
}

func handlePlatformWorkflowApply(ctx context.Context, req *mcp.CallToolRequest, input PlatformWorkflowApplyInput, confirmWrites bool) (*mcp.CallToolResult, PlatformWorkflowOutput, error) {
	requestID := generateRequestID("platform_workflow_apply")
	if strings.TrimSpace(input.PlanID) == "" {
		return nil, PlatformWorkflowOutput{Error: "plan_id is required", RequestID: requestID}, nil
	}
	session, err := loadPlatformWorkspace(ctx)
	if err != nil {
		return nil, PlatformWorkflowOutput{PlatformSetupGuidance: platformSetupGuidanceForCurrentConfig(requestID), Error: err.Error(), RequestID: requestID}, nil
	}
	platformWorkflowMutex.Lock()
	plans, err := loadPlatformWorkflowPlans()
	if err != nil {
		platformWorkflowMutex.Unlock()
		return nil, PlatformWorkflowOutput{Error: err.Error(), RequestID: requestID}, nil
	}
	plan, _, err := findPlatformWorkflow(plans, input.PlanID)
	if err != nil {
		platformWorkflowMutex.Unlock()
		return nil, PlatformWorkflowOutput{Error: err.Error(), RequestID: requestID}, nil
	}
	if plan.Host != session.Host.URL || plan.OrgID != session.Host.DefaultOrgID || plan.ProjectID != session.Host.DefaultProjectID {
		platformWorkflowMutex.Unlock()
		return nil, PlatformWorkflowOutput{Error: "workflow belongs to a different hosted workspace", RequestID: requestID}, nil
	}
	if plan.Status == "completed" {
		output := workflowOutputPlan(*plan)
		platformWorkflowMutex.Unlock()
		return nil, PlatformWorkflowOutput{Plan: output, Status: "completed", Message: "Workflow already completed; no mutations were repeated.", RequestID: requestID}, nil
	}
	action := &PendingPlatformAction{Operation: "workflow_apply", Resource: "workflow", Action: "apply", Summary: plan.Goal, Host: plan.Host, OrgID: plan.OrgID, ProjectID: plan.ProjectID, ProjectName: plan.Project, WorkflowPlanID: plan.ID, WorkflowSteps: len(plan.Steps), Changes: []string{fmt.Sprintf("%d workflow steps", len(plan.Steps))}}
	platformWorkflowMutex.Unlock()
	if !confirmWrites {
		result, err := executePlatformWorkflow(ctx, session, plan.ID, requestID)
		if err != nil {
			return nil, PlatformWorkflowOutput{Status: "failed", Error: err.Error(), RequestID: requestID}, nil
		}
		return nil, PlatformWorkflowOutput{Status: "completed", Message: result.Message, Plan: workflowOutputPlan(*plan), RequestID: requestID}, nil
	}
	token, expiresAt := storePendingPlatformAction(action)
	return nil, PlatformWorkflowOutput{ConfirmationRequired: true, ConfirmationToken: token, ConfirmationExpiry: expiresAt.UTC().Format(time.RFC3339), Status: "confirmation_required", Plan: workflowOutputPlan(*plan), Message: "Workflow requires user approval. Call whodb_platform_confirm with confirmation_token after the user approves.", RequestID: requestID}, nil
}

func executePlatformWorkflow(ctx context.Context, session *platformToolSession, planID, requestID string) (ConfirmOutput, error) {
	platformWorkflowMutex.Lock()
	plans, err := loadPlatformWorkflowPlans()
	if err != nil {
		platformWorkflowMutex.Unlock()
		return ConfirmOutput{}, err
	}
	plan, index, err := findPlatformWorkflow(plans, planID)
	if err != nil {
		platformWorkflowMutex.Unlock()
		return ConfirmOutput{}, err
	}
	if plan.Host != session.Host.URL || plan.OrgID != session.Host.DefaultOrgID || plan.ProjectID != session.Host.DefaultProjectID {
		platformWorkflowMutex.Unlock()
		return ConfirmOutput{}, errors.New("workflow workspace changed before confirmation")
	}
	if plan.Status == "completed" {
		output := workflowOutputPlan(*plan)
		platformWorkflowMutex.Unlock()
		return ConfirmOutput{Message: "Workflow already completed; no mutations were repeated.", Columns: []string{"workflow_id", "status", "plan"}, Rows: [][]any{{plan.ID, "completed", output}}, RequestID: requestID}, nil
	}
	plan.Status = "running"
	plan.UpdatedAt = time.Now().UTC()
	_ = savePlatformWorkflowPlans(plans)
	platformWorkflowMutex.Unlock()
	completed := 0
	for stepIndex := range plan.Steps {
		step := &plan.Steps[stepIndex]
		if step.Status == "completed" {
			completed++
			continue
		}
		for _, dependency := range step.DependsOn {
			dep := findWorkflowStep(plan.Steps, dependency)
			if dep == nil || dep.Status != "completed" {
				step.Status = "blocked"
				step.Error = fmt.Sprintf("dependency %q is not completed", dependency)
				plan.Status = "failed"
				plan.UpdatedAt = time.Now().UTC()
				_ = persistWorkflowPlan(index, *plan)
				return ConfirmOutput{}, errors.New(step.Error)
			}
		}
		spec, payload, err := buildPlatformGenericWrite(session, PlatformGenericWriteInput{Resource: step.Resource, Action: step.Action, ID: step.TargetID, Payload: step.Payload}, step.Operation)
		if err != nil {
			step.Status = "failed"
			step.Error = err.Error()
			plan.Status = "failed"
			plan.UpdatedAt = time.Now().UTC()
			_ = persistWorkflowPlan(index, *plan)
			return ConfirmOutput{}, fmt.Errorf("workflow step %s: %w", step.ID, err)
		}
		result, err := executePlatformMutation(ctx, session.Client, spec.Mutation, session.Host.DefaultProjectID, payload)
		if err != nil {
			step.Status = "failed"
			step.Error = err.Error()
			plan.Status = "failed"
			plan.UpdatedAt = time.Now().UTC()
			_ = persistWorkflowPlan(index, *plan)
			return ConfirmOutput{}, fmt.Errorf("workflow step %s: %w", step.ID, err)
		}
		step.Status = "completed"
		step.Error = ""
		step.ResultID = platformMutationResultID(result)
		completed++
		plan.UpdatedAt = time.Now().UTC()
		if err := persistWorkflowPlan(index, *plan); err != nil {
			return ConfirmOutput{}, err
		}
	}
	plan.Status = "completed"
	plan.UpdatedAt = time.Now().UTC()
	if err := persistWorkflowPlan(index, *plan); err != nil {
		return ConfirmOutput{}, err
	}
	resultIDs := make([]string, 0, len(plan.Steps))
	for _, step := range plan.Steps {
		if step.ResultID != "" {
			resultIDs = append(resultIDs, step.ResultID)
		}
	}
	return ConfirmOutput{Columns: []string{"workflow_id", "status", "completed_steps", "step_count", "result_ids"}, Rows: [][]any{{plan.ID, plan.Status, completed, len(plan.Steps), strings.Join(resultIDs, ",")}}, Message: fmt.Sprintf("Hosted platform workflow completed: %d steps", completed), RequestID: requestID}, nil
}

func persistWorkflowPlan(index int, plan platformWorkflowPlan) error {
	platformWorkflowMutex.Lock()
	defer platformWorkflowMutex.Unlock()
	plans, err := loadPlatformWorkflowPlans()
	if err != nil {
		return err
	}
	if index < 0 || index >= len(plans) || plans[index].ID != plan.ID {
		return errors.New("workflow plan changed while executing")
	}
	plans[index] = plan
	return savePlatformWorkflowPlans(plans)
}

func findWorkflowStep(steps []platformWorkflowStep, id string) *platformWorkflowStep {
	for index := range steps {
		if steps[index].ID == id {
			return &steps[index]
		}
	}
	return nil
}

func platformWorkflowToolDefinitions() []*mcp.Tool {
	return []*mcp.Tool{
		{Name: "whodb_platform_workflow_plan", Description: "Validate and persist a multi-step hosted WhoDB workflow without executing writes. Use this for end-to-end goals such as source to dataset to transform to app.", Annotations: platformReadOnlyAnnotations("Plan Hosted Platform Workflow")},
		{Name: "whodb_platform_workflow_get", Description: "Read one hosted WhoDB workflow plan and its step status without returning payload values.", Annotations: platformReadOnlyAnnotations("Get Hosted Workflow Plan")},
		{Name: "whodb_platform_workflow_list", Description: "List persisted hosted WhoDB workflow plans for the selected workspace.", Annotations: platformReadOnlyAnnotations("List Hosted Workflow Plans")},
		{Name: "whodb_platform_workflow_apply", Description: "Request confirmation to apply or resume a hosted WhoDB workflow plan. Completed steps are skipped on retry.", Annotations: platformDestructiveAnnotations("Apply Hosted Platform Workflow")},
	}
}
