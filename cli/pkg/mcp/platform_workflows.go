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
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/clidey/whodb/cli/internal/config"
	platformapi "github.com/clidey/whodb/cli/internal/platform"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const platformWorkflowVersion = 2
const workflowPayloadLifetime = time.Hour

var (
	platformWorkflowMutex sync.Mutex
	workflowExecutions    sync.Map
	platformWorkflowPath  = defaultPlatformWorkflowPath
)

// PlatformWorkflowStepInput describes one validated generic platform write in a workflow.
type PlatformWorkflowStepInput struct {
	ID             string         `json:"id" jsonschema:"Stable step id used for dependencies and retries."`
	Operation      string         `json:"operation" jsonschema:"create, update, delete, or action"`
	Resource       string         `json:"resource" jsonschema:"Platform resource such as dataset, transform, function, or app"`
	Action         string         `json:"action,omitempty" jsonschema:"Action name when operation is action"`
	TargetID       string         `json:"target_id,omitempty" jsonschema:"Existing resource id for update, delete, or action"`
	TargetFromStep string         `json:"target_from_step,omitempty" jsonschema:"Use the result ID of this earlier create step as target_id."`
	Payload        map[string]any `json:"payload,omitempty" jsonschema:"Structured mutation payload. Use an object with only $step_id set to an earlier create step name wherever its result ID is needed."`
	DependsOn      []string       `json:"depends_on,omitempty" jsonschema:"Step ids that must complete first"`
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
	ID             string                   `json:"id"`
	Operation      string                   `json:"operation"`
	Resource       string                   `json:"resource"`
	Action         string                   `json:"action,omitempty"`
	TargetID       string                   `json:"target_id,omitempty"`
	TargetFromStep string                   `json:"target_from_step,omitempty"`
	PayloadHash    string                   `json:"payload_hash,omitempty"`
	Review         string                   `json:"review,omitempty"`
	Payload        map[string]any           `json:"payload,omitempty"`
	DependsOn      []string                 `json:"depends_on,omitempty"`
	Status         string                   `json:"status"`
	ResultID       string                   `json:"result_id,omitempty"`
	Error          string                   `json:"error,omitempty"`
	Changes        []string                 `json:"changes,omitempty"`
	Preflight      []PlatformWritePreflight `json:"preflight,omitempty"`
}

type platformWorkflowPlan struct {
	AccountID   string                 `json:"account_id,omitempty"`
	Version     int                    `json:"version"`
	ID          string                 `json:"id"`
	Hash        string                 `json:"hash"`
	Goal        string                 `json:"goal"`
	Host        string                 `json:"host"`
	OrgID       string                 `json:"org_id"`
	ProjectID   string                 `json:"project_id"`
	Project     string                 `json:"project_name,omitempty"`
	Status      string                 `json:"status"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	StagedUntil *time.Time             `json:"staged_until,omitempty"`
	Steps       []platformWorkflowStep `json:"steps"`
}

// PlatformWorkflowOutput is the compact, non-secret workflow response.
type PlatformWorkflowOutput struct {
	PlatformSetupGuidance
	ConfirmationRequired bool                    `json:"confirmation_required,omitempty"`
	ConfirmationToken    string                  `json:"confirmation_token,omitempty"`
	ConfirmationExpiry   string                  `json:"confirmation_expiry,omitempty"`
	Plan                 map[string]any          `json:"plan,omitempty"`
	Plans                []map[string]any        `json:"plans,omitempty"`
	Status               string                  `json:"status,omitempty"`
	Message              string                  `json:"message,omitempty"`
	Error                string                  `json:"error,omitempty"`
	ErrorCode            string                  `json:"error_code,omitempty"`
	Retryable            bool                    `json:"retryable,omitempty"`
	SuggestedTools       []string                `json:"suggested_tools,omitempty"`
	Recovery             *PlatformRecoveryAdvice `json:"recovery,omitempty"`
	RequestID            string                  `json:"request_id,omitempty"`
}

func platformWorkflowError(err error, requestID string) PlatformWorkflowOutput {
	errorCode, retryable, suggestedTools := platformErrorFields(err)
	advice := platformRecoveryAdvice(errorCode, err.Error())
	return PlatformWorkflowOutput{Error: err.Error(), ErrorCode: errorCode, Retryable: retryable, SuggestedTools: suggestedTools, Recovery: &advice, RequestID: requestID}
}

func defaultPlatformWorkflowPath() (string, error) {
	dir, err := config.GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "platform-workflows.json"), nil
}

func loadPlatformWorkflowPlans(contexts ...context.Context) ([]platformWorkflowPlan, error) {
	if hasPlatformRuntime(contexts...) {
		return platformStateFor(contexts...).Workflows, nil
	}
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
	if pruneWorkflowPayloads(plans, time.Now()) {
		if err := savePlatformWorkflowPlans(plans); err != nil {
			return nil, err
		}
	}
	return plans, nil
}

func savePlatformWorkflowPlans(plans []platformWorkflowPlan, contexts ...context.Context) error {
	if hasPlatformRuntime(contexts...) {
		state := platformStateFor(contexts...)
		state.Workflows = plans
		if state.Checkpoint != nil {
			return state.Checkpoint()
		}
		return nil
	}
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
		if closeErr := tmp.Close(); closeErr != nil {
			return fmt.Errorf("set temporary workflow file permissions: %w (close: %v)", err, closeErr)
		}
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		if closeErr := tmp.Close(); closeErr != nil {
			return fmt.Errorf("write temporary workflow file: %w (close: %v)", err, closeErr)
		}
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

// Step references are explicit JSON objects so ordinary strings always stay literal.
func workflowValue(field string, value any, lookup func(string) (string, error)) (any, error) {
	switch v := value.(type) {
	case map[string]any:
		if ref, ok := v["$step_id"]; ok {
			if field != "id" && !strings.HasSuffix(field, "Id") && !strings.HasSuffix(field, "Ids") {
				return nil, fmt.Errorf("field %q does not take a step result ID", field)
			}
			id, ok := ref.(string)
			if !ok || id == "" || len(v) != 1 {
				return nil, errors.New("expected a single $step_id string")
			}
			return lookup(id)
		}
		out := make(map[string]any, len(v))
		for key, child := range v {
			switch strings.ToLower(strings.ReplaceAll(key, "_", "")) {
			case "password", "apikey", "clientsecret", "privatekey", "token", "accesstoken", "refreshtoken", "sharetoken":
				return nil, fmt.Errorf("workflow payloads cannot contain secrets (%s)", key)
			}
			resolved, err := workflowValue(key, child, lookup)
			if err != nil {
				return nil, err
			}
			out[key] = resolved
		}
		return out, nil
	case []any:
		out := make([]any, len(v))
		for i, child := range v {
			resolved, err := workflowValue(field, child, lookup)
			if err != nil {
				return nil, err
			}
			out[i] = resolved
		}
		return out, nil
	}
	return value, nil
}

func workflowWriteInput(step platformWorkflowStep, lookup func(string) (string, error)) (PlatformGenericWriteInput, error) {
	input := PlatformGenericWriteInput{Resource: step.Resource, Action: step.Action, ID: step.TargetID, Payload: make(map[string]any, len(step.Payload))}
	if step.TargetFromStep != "" {
		id, err := lookup(step.TargetFromStep)
		if err != nil {
			return input, err
		}
		input.ID = id
	}
	for k, v := range step.Payload {
		resolved, err := workflowValue(k, v, lookup)
		if err != nil {
			return input, err
		}
		input.Payload[k] = resolved
	}
	return input, nil
}

func workflowDigest(value any) string {
	data, _ := json.Marshal(value)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func pruneWorkflowPayloads(plans []platformWorkflowPlan, now time.Time) bool {
	changed := false
	for i := range plans {
		p := &plans[i]
		if p.StagedUntil == nil || p.Status == "completed" || now.Before(*p.StagedUntil) || p.Status == "expired" {
			continue
		}
		for j := range p.Steps {
			p.Steps[j].Payload = nil
		}
		p.Status = "expired"
		changed = true
	}
	return changed
}

func workflowReview(step platformWorkflowStep, input PlatformGenericWriteInput) string {
	operation := step.Operation
	if step.Operation == "action" {
		operation = step.Action
	}
	summary := operation + " " + step.Resource
	if step.Resource == "ontology" && operation == "add_record" {
		summary = "add row to ontology"
	}
	if operation == "promote" {
		summary = "publish " + fmt.Sprint(input.Payload["objectType"])
	}
	if input.ID != "" {
		summary += fmt.Sprintf(" %q", input.ID)
	}
	for _, field := range []string{"objectId", "appId", "targetProjectId", "targetOrgId", "apiName", "name", "path"} {
		if value, ok := input.Payload[field].(string); ok {
			summary += fmt.Sprintf(" %s=%q", field, value)
		}
	}
	if content, ok := input.Payload["content"].(string); ok {
		summary += fmt.Sprintf(" (%d bytes, SHA-256 %x)", len(content), sha256.Sum256([]byte(content)))
	}
	return summary
}

func validatePlatformWorkflowInput(input PlatformWorkflowPlanInput, session *platformToolSession) (platformWorkflowPlan, error) {
	if strings.TrimSpace(input.Goal) == "" {
		return platformWorkflowPlan{}, errors.New("goal is required")
	}
	if len(input.Steps) == 0 {
		return platformWorkflowPlan{}, errors.New("at least one workflow step is required")
	}
	plan := platformWorkflowPlan{
		AccountID: session.Host.AccountID,
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
		if operation != "action" && action != "" {
			return platformWorkflowPlan{}, fmt.Errorf("step %q has an action on a %s operation", id, operation)
		}
		key := operation + ":" + resource

		if operation == "action" {
			if action == "" {
				return platformWorkflowPlan{}, fmt.Errorf("step %q requires action", id)
			}
			key = "action:" + action + ":" + resource
		}
		spec, ok := platformapi.GenericWriteSpecs[key]
		if !ok {
			return platformWorkflowPlan{}, fmt.Errorf("step %q uses unsupported platform write %q", id, key)
		}
		if spec.Mode == platformapi.GenericWriteModeFileUpload {
			return platformWorkflowPlan{}, fmt.Errorf("step %q: local file uploads require direct confirmation", id)
		}
		// Normalize programmatic callers to the JSON types used by the MCP transport.
		raw, err := json.Marshal(stepInput.Payload)
		if err != nil {
			return platformWorkflowPlan{}, fmt.Errorf("step %q: %w", id, err)
		}
		var payload map[string]any
		if err := json.Unmarshal(raw, &payload); err != nil {
			return platformWorkflowPlan{}, fmt.Errorf("step %q: %w", id, err)
		}
		fields := platformapi.PayloadShapes[key].Fields
		for name := range payload {
			index := slices.IndexFunc(fields, func(f platformapi.PayloadField) bool { return f.Name == name })
			if index < 0 {
				return platformWorkflowPlan{}, fmt.Errorf("step %q: field %q is not declared in the write schema", id, name)
			}
			if fields[index].Secret {
				return platformWorkflowPlan{}, fmt.Errorf("step %q: workflow payloads cannot contain secrets (%s)", id, name)
			}
		}
		step := platformWorkflowStep{
			ID: id, Operation: operation, Resource: resource, Action: action,
			TargetID: strings.TrimSpace(stepInput.TargetID), TargetFromStep: stepInput.TargetFromStep,
			DependsOn: slices.Clone(stepInput.DependsOn), Status: "pending", Payload: payload,
		}
		appFile := resource == "app" && (action == "upsert_file" || action == "delete_file")
		if step.TargetFromStep != "" && !spec.NeedsID && !appFile {
			return platformWorkflowPlan{}, fmt.Errorf("step %q: use a payload $step_id reference for this operation", id)
		}
		if step.TargetFromStep != "" && step.TargetID != "" {
			return platformWorkflowPlan{}, fmt.Errorf("step %q: choose target_id or target_from_step", id)
		}
		if appFile && (step.TargetFromStep != "" || step.TargetID != "") && step.Payload["appId"] != nil && (step.TargetFromStep != "" || step.Payload["appId"] != step.TargetID) {
			return platformWorkflowPlan{}, fmt.Errorf("step %q: appId conflicts with the step target", id)
		}
		input, err := workflowWriteInput(step, func(ref string) (string, error) {
			prior := findWorkflowStep(plan.Steps, ref)
			if prior == nil || prior.Operation != "create" {
				return "", fmt.Errorf("step %q references %q; expected an earlier create step", id, ref)
			}
			if !slices.Contains(step.DependsOn, ref) {
				step.DependsOn = append(step.DependsOn, ref)
			}
			return "result of " + ref, nil
		})
		if err != nil {
			return platformWorkflowPlan{}, err
		}
		slices.Sort(step.DependsOn)
		if spec.NeedsID && input.ID == "" {
			return platformWorkflowPlan{}, fmt.Errorf("step %q requires target_id or target_from_step", id)
		}
		if _, _, err := buildPlatformGenericWrite(session, input, operation); err != nil {
			return platformWorkflowPlan{}, fmt.Errorf("step %q: %w", id, err)
		}
		if err := validatePlatformWriteInput(input, operation, spec); err != nil {
			return platformWorkflowPlan{}, fmt.Errorf("step %q: %w", id, err)
		}
		step.Changes = genericWriteChanges(step.Payload)
		step.Review = workflowReview(step, input)
		if (resource == "app" && step.Payload["content"] != nil) || step.Payload["files"] != nil {
			expires := plan.CreatedAt.Add(workflowPayloadLifetime)
			plan.StagedUntil = &expires
		}
		if len(step.Payload) > 0 {
			step.PayloadHash = workflowDigest(step.Payload)
		}
		plan.Steps = append(plan.Steps, step)
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
	copyPlan.StagedUntil = nil
	copyPlan.Steps = slices.Clone(plan.Steps)
	for i := range copyPlan.Steps {
		s := &copyPlan.Steps[i]
		s.Status, s.ResultID, s.Error = "", "", ""
		s.Preflight = nil
		if plan.Version >= 2 {
			s.Payload = nil
		}
	}
	return workflowDigest(copyPlan)
}

func workflowOutputPlan(plan platformWorkflowPlan) map[string]any {
	steps := make([]map[string]any, 0, len(plan.Steps))
	for _, step := range plan.Steps {
		entry := map[string]any{"id": step.ID, "operation": step.Operation, "resource": step.Resource, "status": step.Status, "changes": step.Changes}
		if step.Review != "" {
			entry["review"] = step.Review
		}
		if step.TargetFromStep != "" {
			entry["target_from_step"] = step.TargetFromStep
		}
		if step.Action != "" {
			entry["action"] = step.Action
		}
		if step.TargetID != "" {
			entry["target_id"] = step.TargetID
		}
		if len(step.DependsOn) > 0 {
			entry["depends_on"] = step.DependsOn
		}
		if len(step.Preflight) > 0 {
			entry["preflight"] = step.Preflight
		}
		if step.ResultID != "" {
			entry["result_id"] = step.ResultID
		}
		if step.Error != "" {
			entry["error"] = step.Error
		}
		steps = append(steps, entry)
	}
	output := map[string]any{"account_id": plan.AccountID, "id": plan.ID, "hash": plan.Hash, "goal": plan.Goal, "host": plan.Host, "org_id": plan.OrgID, "project_id": plan.ProjectID, "project_name": plan.Project, "status": plan.Status, "created_at": plan.CreatedAt, "updated_at": plan.UpdatedAt, "steps": steps}
	if plan.StagedUntil != nil {
		output["staged_until"] = *plan.StagedUntil
	}
	return output
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
		addPlatformTool(server, tool, createPlatformWorkflowHandler())
	case "whodb_platform_workflow_get":
		addPlatformTool(server, tool, createPlatformWorkflowGetHandler())
	case "whodb_platform_workflow_list":
		addPlatformTool(server, tool, createPlatformWorkflowListHandler())
	case "whodb_platform_workflow_apply":
		if !secOpts.ReadOnly {
			addPlatformTool(server, tool, createPlatformWorkflowApplyHandlerWithConfirmation(secOpts.ConfirmWrites))
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
		return nil, PlatformWorkflowOutput{PlatformSetupGuidance: platformSetupGuidanceForCurrentConfig(ctx, requestID), Error: err.Error(), RequestID: requestID}, nil
	}
	plan, err := validatePlatformWorkflowInput(input, session)
	if err != nil {
		return nil, platformWorkflowError(err, requestID), nil
	}
	for index := range plan.Steps {
		step := &plan.Steps[index]
		input, err := workflowWriteInput(*step, func(ref string) (string, error) { return "result of " + ref, nil })
		if err != nil {
			return nil, platformWorkflowError(err, requestID), nil
		}
		spec, payload, buildErr := buildPlatformGenericWrite(session, input, step.Operation)
		if buildErr != nil {
			return nil, platformWorkflowError(fmt.Errorf("step %s: %w", step.ID, buildErr), requestID), nil
		}
		if capabilityErr := validatePlatformWriteCapability(ctx, session, spec); capabilityErr != nil {
			return nil, platformWorkflowError(fmt.Errorf("step %s: %w", step.ID, capabilityErr), requestID), nil
		}
		step.Preflight = platformWritePreflight(ctx, session, spec, payload, step.TargetID)
	}
	platformWorkflowMutex.Lock()
	defer platformWorkflowMutex.Unlock()
	plans, err := loadPlatformWorkflowPlans(ctx)
	if err != nil {
		return nil, platformWorkflowError(err, requestID), nil
	}
	plans = append(plans, plan)
	if err := savePlatformWorkflowPlans(plans, ctx); err != nil {
		return nil, platformWorkflowError(err, requestID), nil
	}
	return nil, PlatformWorkflowOutput{Plan: workflowOutputPlan(plan), Status: "planned", Message: "Workflow plan created. Review it, then apply it when the user approves.", RequestID: requestID}, nil
}

func HandlePlatformWorkflowGet(ctx context.Context, req *mcp.CallToolRequest, input PlatformWorkflowGetInput) (*mcp.CallToolResult, PlatformWorkflowOutput, error) {
	requestID := generateRequestID("platform_workflow_get")
	if strings.TrimSpace(input.PlanID) == "" {
		return nil, PlatformWorkflowOutput{Error: "plan_id is required", RequestID: requestID}, nil
	}
	var selected *platformToolSession
	if request := platformRequestFromContext(ctx); request != nil && request.Target != nil {
		var err error
		selected, err = loadPlatformWorkspace(ctx)
		if err != nil {
			return nil, PlatformWorkflowOutput{Error: err.Error(), RequestID: requestID}, nil
		}
	}
	platformWorkflowMutex.Lock()
	defer platformWorkflowMutex.Unlock()
	plans, err := loadPlatformWorkflowPlans(ctx)
	if err != nil {
		return nil, platformWorkflowError(err, requestID), nil
	}
	plan, _, err := findPlatformWorkflow(plans, input.PlanID)
	if err != nil {
		return nil, PlatformWorkflowOutput{Error: err.Error(), RequestID: requestID}, nil
	}
	if err := checkPlatformPolicy(ctx, plan.Host, plan.OrgID, plan.ProjectID); err != nil {
		return nil, PlatformWorkflowOutput{Error: err.Error(), RequestID: requestID}, nil
	}
	if selected != nil && (plan.AccountID != selected.Host.AccountID || plan.Host != selected.Host.URL || plan.OrgID != selected.Host.DefaultOrgID || plan.ProjectID != selected.Host.DefaultProjectID) {
		return nil, PlatformWorkflowOutput{Error: "workflow belongs to a different hosted workspace or account", RequestID: requestID}, nil
	}
	recordPlatformScope(ctx, &platformToolSession{Host: config.PlatformHost{URL: plan.Host, AccountID: plan.AccountID, DefaultOrgID: plan.OrgID, DefaultProjectID: plan.ProjectID, DefaultProjectName: plan.Project}})
	return nil, PlatformWorkflowOutput{Plan: workflowOutputPlan(*plan), Status: plan.Status, RequestID: requestID}, nil
}

func HandlePlatformWorkflowList(ctx context.Context, req *mcp.CallToolRequest, input PlatformWorkflowListInput) (*mcp.CallToolResult, PlatformWorkflowOutput, error) {
	requestID := generateRequestID("platform_workflow_list")
	session, err := loadPlatformWorkspace(ctx)
	if err != nil {
		return nil, PlatformWorkflowOutput{PlatformSetupGuidance: platformSetupGuidanceForCurrentConfig(ctx, requestID), Error: err.Error(), RequestID: requestID}, nil
	}
	platformWorkflowMutex.Lock()
	defer platformWorkflowMutex.Unlock()
	plans, err := loadPlatformWorkflowPlans(ctx)
	if err != nil {
		return nil, PlatformWorkflowOutput{Error: err.Error(), RequestID: requestID}, nil
	}
	items := make([]map[string]any, 0)
	for _, plan := range plans {
		if plan.AccountID == session.Host.AccountID && plan.Host == session.Host.URL && plan.OrgID == session.Host.DefaultOrgID && plan.ProjectID == session.Host.DefaultProjectID {
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
		return nil, PlatformWorkflowOutput{PlatformSetupGuidance: platformSetupGuidanceForCurrentConfig(ctx, requestID), Error: err.Error(), RequestID: requestID}, nil
	}
	platformWorkflowMutex.Lock()
	plans, err := loadPlatformWorkflowPlans(ctx)
	if err != nil {
		platformWorkflowMutex.Unlock()
		return nil, PlatformWorkflowOutput{Error: err.Error(), RequestID: requestID}, nil
	}
	plan, _, err := findPlatformWorkflow(plans, input.PlanID)
	if err != nil {
		platformWorkflowMutex.Unlock()
		return nil, PlatformWorkflowOutput{Error: err.Error(), RequestID: requestID}, nil
	}
	if plan.AccountID != session.Host.AccountID || plan.Host != session.Host.URL || plan.OrgID != session.Host.DefaultOrgID || plan.ProjectID != session.Host.DefaultProjectID {
		platformWorkflowMutex.Unlock()
		return nil, PlatformWorkflowOutput{Error: "workflow belongs to a different hosted workspace or account; create a new plan", RequestID: requestID}, nil
	}
	if plan.Status == "completed" {
		output := workflowOutputPlan(*plan)
		platformWorkflowMutex.Unlock()
		return nil, PlatformWorkflowOutput{Plan: output, Status: "completed", Message: "Workflow already completed; no mutations were repeated.", RequestID: requestID}, nil
	}
	if plan.Status == "expired" || (plan.StagedUntil != nil && !time.Now().Before(*plan.StagedUntil)) {
		platformWorkflowMutex.Unlock()
		return nil, platformWorkflowError(errors.New("workflow inputs expired; create a new plan"), requestID), nil
	}
	changes := make([]string, 0, len(plan.Steps))
	for i := 0; i < len(plan.Steps); i++ {
		step := plan.Steps[i]
		if step.Status == "completed" {
			continue
		}
		review := step.Review
		if review == "" {
			review = workflowReview(step, PlatformGenericWriteInput{ID: step.TargetID, Payload: step.Payload})
		}
		if step.Resource == "ontology" && step.Action == "add_record" {
			count := 1
			for i+1 < len(plan.Steps) {
				next := plan.Steps[i+1]
				if next.Resource != step.Resource || next.Action != step.Action || next.TargetID != step.TargetID || next.TargetFromStep != step.TargetFromStep || next.Status == "completed" {
					break
				}
				i++
				count++
			}
			if count > 1 {
				review = fmt.Sprintf("%s (%d rows)", review, count)
			}
		}
		changes = append(changes, review)
	}
	action := &PendingPlatformAction{
		PayloadHash: hashWorkflowPlan(*plan), AccountID: session.Host.AccountID,
		Operation: "workflow_apply", Resource: "workflow", Action: "apply", Summary: plan.Goal,
		Host: plan.Host, OrgID: plan.OrgID, ProjectID: plan.ProjectID, ProjectName: plan.Project,
		WorkflowPlanID: plan.ID, WorkflowSteps: len(plan.Steps), Changes: changes,
	}
	platformWorkflowMutex.Unlock()
	if !confirmWrites {
		result, err := executePlatformWorkflow(ctx, session, plan.ID, action.PayloadHash, requestID)
		if err != nil {
			return nil, PlatformWorkflowOutput{Status: "failed", Error: err.Error(), RequestID: requestID}, nil
		}
		_, output, err := HandlePlatformWorkflowGet(ctx, req, PlatformWorkflowGetInput{PlanID: plan.ID})
		output.Message = result.Message
		output.RequestID = requestID
		return nil, output, err
	}
	token, expiresAt := storePendingPlatformAction(action, ctx)
	return nil, PlatformWorkflowOutput{ConfirmationRequired: true, ConfirmationToken: token, ConfirmationExpiry: expiresAt.UTC().Format(time.RFC3339), Status: "confirmation_required", Plan: workflowOutputPlan(*plan), Message: "Workflow requires user approval. Call whodb_platform_confirm with confirmation_token after the user approves.", RequestID: requestID}, nil
}

func executePlatformWorkflow(ctx context.Context, session *platformToolSession, planID, approvedHash, requestID string) (ConfirmOutput, error) {
	if _, busy := workflowExecutions.LoadOrStore(planID, true); busy {
		return ConfirmOutput{}, errors.New("workflow is already executing")
	}
	defer workflowExecutions.Delete(planID)
	platformWorkflowMutex.Lock()
	plans, err := loadPlatformWorkflowPlans(ctx)
	if err != nil {
		platformWorkflowMutex.Unlock()
		return ConfirmOutput{}, err
	}
	plan, index, err := findPlatformWorkflow(plans, planID)
	if err != nil {
		platformWorkflowMutex.Unlock()
		return ConfirmOutput{}, err
	}
	if plan.AccountID != session.Host.AccountID || plan.Host != session.Host.URL || plan.OrgID != session.Host.DefaultOrgID || plan.ProjectID != session.Host.DefaultProjectID {
		platformWorkflowMutex.Unlock()
		return ConfirmOutput{}, errors.New("workflow workspace or account changed before confirmation")
	}
	if approvedHash == "" || approvedHash != hashWorkflowPlan(*plan) {
		platformWorkflowMutex.Unlock()
		return ConfirmOutput{}, errors.New("workflow changed; request a new approval")
	}
	if plan.Version >= 2 && plan.Status != "completed" {
		if plan.StagedUntil != nil && !time.Now().Before(*plan.StagedUntil) {
			platformWorkflowMutex.Unlock()
			return ConfirmOutput{}, errors.New("workflow inputs expired; create a new plan")
		}
		for _, step := range plan.Steps {
			if step.Status == "completed" {
				continue
			}
			digest := ""
			if len(step.Payload) > 0 {
				digest = workflowDigest(step.Payload)
			}
			if digest != step.PayloadHash {
				platformWorkflowMutex.Unlock()
				return ConfirmOutput{}, errors.New("workflow payload changed; create a new plan")
			}
		}
	}
	if plan.Status == "completed" {
		output := workflowOutputPlan(*plan)
		platformWorkflowMutex.Unlock()
		return ConfirmOutput{Message: "Workflow already completed; no mutations were repeated.", Columns: []string{"workflow_id", "status", "plan"}, Rows: [][]any{{plan.ID, "completed", output}}, RequestID: requestID}, nil
	}
	plan.Status = "running"
	plan.UpdatedAt = time.Now().UTC()
	saveErr := savePlatformWorkflowPlans(plans, ctx)
	platformWorkflowMutex.Unlock()
	if saveErr != nil {
		return ConfirmOutput{}, saveErr
	}
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
				_ = persistWorkflowPlan(ctx, index, *plan)
				return ConfirmOutput{}, errors.New(step.Error)
			}
		}
		input := PlatformGenericWriteInput{Resource: step.Resource, Action: step.Action, ID: step.TargetID, Payload: step.Payload}
		if plan.Version >= 2 {
			input, err = workflowWriteInput(*step, func(ref string) (string, error) {
				prior := findWorkflowStep(plan.Steps, ref)
				if prior == nil || prior.Status != "completed" || prior.ResultID == "" {
					return "", fmt.Errorf("step %q has no completed result ID", ref)
				}
				return prior.ResultID, nil
			})
		}
		var spec platformapi.GenericWriteSpec
		var payload map[string]any
		if err == nil {
			spec, payload, err = buildPlatformGenericWrite(session, input, step.Operation)
		}
		if err != nil {
			step.Status = "failed"
			step.Error = err.Error()
			plan.Status = "failed"
			plan.UpdatedAt = time.Now().UTC()
			_ = persistWorkflowPlan(ctx, index, *plan)
			return ConfirmOutput{}, fmt.Errorf("workflow step %s: %w", step.ID, err)
		}
		result, err := executePlatformMutation(ctx, session.Client, spec.Mutation, session.Host.DefaultProjectID, payload)
		if err != nil {
			step.Status = "failed"
			// Remote diagnostics can quote mutation inputs, including file contents.
			step.Error = fmt.Sprintf("%s failed", spec.Mutation)
			plan.Status = "failed"
			plan.UpdatedAt = time.Now().UTC()
			_ = persistWorkflowPlan(ctx, index, *plan)
			return ConfirmOutput{}, fmt.Errorf("workflow step %s: %w", step.ID, err)
		}
		if plan.Version >= 2 {
			step.Payload = nil
		}
		step.Status = "completed"
		step.Error = ""
		step.ResultID = platformMutationResultID(result)
		completed++
		plan.UpdatedAt = time.Now().UTC()
		if err := persistWorkflowPlan(ctx, index, *plan); err != nil {
			return ConfirmOutput{}, err
		}
	}
	plan.Status = "completed"
	plan.UpdatedAt = time.Now().UTC()
	if err := persistWorkflowPlan(ctx, index, *plan); err != nil {
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

func persistWorkflowPlan(ctx context.Context, index int, plan platformWorkflowPlan) error {
	platformWorkflowMutex.Lock()
	defer platformWorkflowMutex.Unlock()
	plans, err := loadPlatformWorkflowPlans(ctx)
	if err != nil {
		return err
	}
	if index < 0 || index >= len(plans) || plans[index].ID != plan.ID {
		return errors.New("workflow plan changed while executing")
	}
	plans[index] = plan
	return savePlatformWorkflowPlans(plans, ctx)
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
		{Name: "whodb_platform_workflow_plan", Description: "Validate and persist a multi-step hosted WhoDB workflow without executing writes. Batch related writes for one review. Use target_from_step or payload objects like {\"$step_id\":\"create-app\"} for prior create results. Plans with staged file contents expire after one hour.", Annotations: platformReadOnlyAnnotations("Plan Hosted Platform Workflow")},
		{Name: "whodb_platform_workflow_get", Description: "Read one hosted WhoDB workflow plan and its step status without returning payload values.", Annotations: platformReadOnlyAnnotations("Get Hosted Workflow Plan")},
		{Name: "whodb_platform_workflow_list", Description: "List persisted hosted WhoDB workflow plans for the selected workspace.", Annotations: platformReadOnlyAnnotations("List Hosted Workflow Plans")},
		{Name: "whodb_platform_workflow_apply", Description: "Request confirmation to apply or resume a hosted WhoDB workflow plan. Completed steps are skipped on retry.", Annotations: platformDestructiveAnnotations("Apply Hosted Platform Workflow")},
	}
}
