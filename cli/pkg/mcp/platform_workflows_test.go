/*
 * Copyright 2026 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 */

package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/clidey/whodb/cli/internal/config"
	platformapi "github.com/clidey/whodb/cli/internal/platform"
)

func workflowTestSession() *platformToolSession {
	return &platformToolSession{Host: config.PlatformHost{URL: "https://app.whodb.com", DefaultOrgID: "org-1", DefaultProjectID: "project-1", DefaultProjectName: "Customer"}}
}

func TestValidatePlatformWorkflowInputBuildsStablePlan(t *testing.T) {
	plan, err := validatePlatformWorkflowInput(PlatformWorkflowPlanInput{
		Goal: "Create a dataset and then a transform",
		Steps: []PlatformWorkflowStepInput{
			{ID: "dataset", Operation: "create", Resource: "dataset", Payload: map[string]any{"name": "customers", "description": "customer dataset"}},
			{ID: "transform", Operation: "create", Resource: "transform", DependsOn: []string{"dataset"}, Payload: map[string]any{"name": "normalize", "graphJson": `{"nodes":[],"edges":[]}`}},
		},
	}, workflowTestSession())
	if err != nil {
		t.Fatalf("validatePlatformWorkflowInput() error = %v", err)
	}
	if plan.ID == "" || plan.Hash == "" || plan.Status != "planned" {
		t.Fatalf("plan metadata = %#v", plan)
	}
	if len(plan.Steps) != 2 || plan.Steps[1].DependsOn[0] != "dataset" {
		t.Fatalf("plan steps = %#v", plan.Steps)
	}
	if output := workflowOutputPlan(plan); output["steps"].([]map[string]any)[0]["payload"] != nil {
		t.Fatal("workflow output exposed a payload")
	}
}

func TestValidatePlatformWorkflowInputRejectsUnsafeAndInvalidPlans(t *testing.T) {
	tests := []struct {
		name  string
		input PlatformWorkflowPlanInput
		want  string
	}{
		{name: "secret", input: PlatformWorkflowPlanInput{Goal: "save secret", Steps: []PlatformWorkflowStepInput{{Operation: "create", Resource: "secret", Payload: map[string]any{"name": "token", "value": "secret"}}}}, want: "cannot contain secrets"},
		{name: "provider credential", input: PlatformWorkflowPlanInput{Goal: "change provider", Steps: []PlatformWorkflowStepInput{{Operation: "update", Resource: "ai_provider", TargetID: "provider", Payload: map[string]any{"apiKey": "fixture"}}}}, want: "cannot contain secrets"},
		{name: "nested credential", input: PlatformWorkflowPlanInput{Goal: "create dataset", Steps: []PlatformWorkflowStepInput{{Operation: "create", Resource: "dataset", Payload: map[string]any{"name": "table", "columns": []map[string]any{{"password": "fixture"}}}}}}, want: "cannot contain secrets"},
		{name: "local upload", input: PlatformWorkflowPlanInput{Goal: "upload", Steps: []PlatformWorkflowStepInput{{Operation: "action", Resource: "file", Action: "upload", Payload: map[string]any{"file_path": "fixture.txt"}}}}, want: "local file uploads"},
		{name: "missing create", input: PlatformWorkflowPlanInput{Goal: "add file", Steps: []PlatformWorkflowStepInput{{Operation: "action", Resource: "app", Action: "upsert_file", TargetFromStep: "missing"}}}, want: "earlier create step"},
		{name: "cycle", input: PlatformWorkflowPlanInput{Goal: "cycle", Steps: []PlatformWorkflowStepInput{{ID: "a", Operation: "create", Resource: "dataset", DependsOn: []string{"b"}, Payload: map[string]any{"name": "a", "description": "a"}}, {ID: "b", Operation: "create", Resource: "dataset", DependsOn: []string{"a"}, Payload: map[string]any{"name": "b", "description": "b"}}}}, want: "dependency cycle"},
		{name: "unknown dependency", input: PlatformWorkflowPlanInput{Goal: "bad dependency", Steps: []PlatformWorkflowStepInput{{ID: "a", Operation: "create", Resource: "dataset", DependsOn: []string{"missing"}, Payload: map[string]any{"name": "a", "description": "a"}}}}, want: "unknown step"},
		{name: "target", input: PlatformWorkflowPlanInput{Goal: "update", Steps: []PlatformWorkflowStepInput{{Operation: "update", Resource: "dataset"}}}, want: "requires target_id"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := validatePlatformWorkflowInput(test.input, workflowTestSession())
			if err == nil || !containsWorkflowError(err.Error(), test.want) {
				t.Fatalf("error = %v, want substring %q", err, test.want)
			}
		})
	}
}

func containsWorkflowError(message, want string) bool {
	for index := 0; index+len(want) <= len(message); index++ {
		if message[index:index+len(want)] == want {
			return true
		}
	}
	return false
}

func TestPlatformWorkflowStateIsPrivateAndRoundTrips(t *testing.T) {
	oldPath := platformWorkflowPath
	path := t.TempDir() + "/workflows.json"
	platformWorkflowPath = func() (string, error) { return path, nil }
	t.Cleanup(func() { platformWorkflowPath = oldPath })

	plan, err := validatePlatformWorkflowInput(PlatformWorkflowPlanInput{Goal: "round trip", Steps: []PlatformWorkflowStepInput{{Operation: "create", Resource: "dataset", Payload: map[string]any{"name": "safe", "description": "safe dataset"}}}}, workflowTestSession())
	if err != nil {
		t.Fatal(err)
	}
	if err := savePlatformWorkflowPlans([]platformWorkflowPlan{plan}); err != nil {
		t.Fatal(err)
	}
	plans, err := loadPlatformWorkflowPlans()
	if err != nil || len(plans) != 1 {
		t.Fatalf("loadPlatformWorkflowPlans() = %v, %#v", err, plans)
	}
	if plans[0].Steps[0].Payload["name"] != "safe" {
		t.Fatalf("round-tripped payload = %#v", plans[0].Steps[0].Payload)
	}
}

type recordingWorkflowClient struct {
	fakePlatformClient
	calls []platformWorkflowCall
}

type platformWorkflowCall struct {
	name      string
	variables map[string]any
}

func (c *recordingWorkflowClient) PlatformMutation(ctx context.Context, name string, variables map[string]any) (*platformapi.PlatformMutationResult, error) {
	c.calls = append(c.calls, platformWorkflowCall{name, variables})
	return &platformapi.PlatformMutationResult{Data: json.RawMessage(fmt.Sprintf(`{"id":"%s-result"}`, name))}, nil
}

func TestWorkflowSingleApprovalForAppBuild(t *testing.T) {
	client := &recordingWorkflowClient{}
	session := testPlatformSession(client)
	withPlatformSessionLoader(t, func(context.Context) (*platformToolSession, error) { return session, nil })
	state := NewPlatformState()
	ctx := WithPlatformRuntime(t.Context(), &PlatformRuntime{State: state, RequireBrowserApproval: true})
	var entity map[string]any
	if err := json.Unmarshal([]byte(platformapi.PayloadShapes["create:ontology"].Examples[0]), &entity); err != nil {
		t.Fatal(err)
	}
	ref := func(id string) map[string]any { return map[string]any{"$step_id": id} }
	const source = "page source fixture"
	_, plan, err := HandlePlatformWorkflowPlan(ctx, nil, PlatformWorkflowPlanInput{Goal: "Publish customer portal", Steps: []PlatformWorkflowStepInput{
		{ID: "entity", Operation: "create", Resource: "ontology", Payload: entity},
		{ID: "row", Operation: "action", Resource: "ontology", Action: "add_record", TargetFromStep: "entity", Payload: map[string]any{"values": []any{map[string]any{"Key": "name", "Value": "Ada"}}}},
		{ID: "app", Operation: "create", Resource: "app", Payload: map[string]any{"name": "Portal", "ontologyIds": []any{ref("entity")}}},
		{ID: "page", Operation: "action", Resource: "app", Action: "upsert_file", TargetFromStep: "app", Payload: map[string]any{"path": "page.js", "content": source}},
		{ID: "publish", Operation: "action", Resource: "object", Action: "promote", Payload: map[string]any{"objectType": "app", "objectId": ref("app")}},
	}})
	if err != nil || plan.Error != "" {
		t.Fatalf("plan: %v, %s", err, plan.Error)
	}
	_, apply, err := HandlePlatformWorkflowApply(ctx, nil, PlatformWorkflowApplyInput{PlanID: plan.Plan["id"].(string)})
	if err != nil || apply.Error != "" {
		t.Fatalf("apply: %v, %s", err, apply.Error)
	}
	approvals := state.PendingApprovals(session.Host.AccountID)
	if len(approvals) != 1 {
		t.Fatalf("approvals: %v", approvals)
	}
	preview := strings.Join(approvals[0].Preview.Changes, " ")
	for _, want := range []string{"customer", "add row", "Portal", "page.js", "publish app"} {
		if !strings.Contains(preview, want) {
			t.Errorf("review missing %s: %s", want, preview)
		}
	}
	if strings.Contains(preview, source) {
		t.Fatal("source leaked into review")
	}
	confirm := func() ConfirmOutput {
		_, out, err := HandlePlatformConfirm(ctx, nil, ConfirmInput{Token: apply.ConfirmationToken})
		if err != nil {
			t.Fatal(err)
		}
		return out
	}
	if confirm().Error == "" || !state.DecidePendingApproval(session.Host.AccountID, apply.ConfirmationToken, true) {
		t.Fatal("expected browser approval")
	}
	state.Workflows[0].Steps[3].Payload["content"] = "changed"
	if confirm().Error == "" || len(client.calls) != 0 {
		t.Fatal("changed bytes should invalidate approval before any write")
	}
	state.Workflows[0].Steps[3].Payload["content"] = source
	if out := confirm(); out.Error != "" {
		t.Fatal(out.Error)
	}
	if len(client.calls) != 5 || client.calls[1].variables["entityId"] != "CreateOntology-result" || client.calls[3].variables["appId"] != "CreateApp-result" || client.calls[3].variables["content"] != source || client.calls[4].variables["input"].(map[string]any)["objectId"] != "CreateApp-result" {
		t.Fatalf("executed writes: %#v", client.calls)
	}
	data, err := json.Marshal(state)
	if err != nil || strings.Contains(string(data), source) {
		t.Fatal("completed plan retained source")
	}
}

func TestWorkflowStagedSourceExpiry(t *testing.T) {
	plan, err := validatePlatformWorkflowInput(PlatformWorkflowPlanInput{Goal: "edit file", Steps: []PlatformWorkflowStepInput{{Operation: "action", Action: "upsert_file", Resource: "app", TargetID: "app", Payload: map[string]any{"path": "main.js", "content": "fixture"}}}}, workflowTestSession())
	if err != nil {
		t.Fatal(err)
	}
	state := NewPlatformState()
	state.Workflows = []platformWorkflowPlan{plan}
	deadline := state.NextCleanupAt()
	if deadline != plan.CreatedAt.Add(workflowPayloadLifetime) {
		t.Fatal("missing staged payload cleanup deadline")
	}
	state.PruneExpired(deadline)
	step := state.Workflows[0].Steps[0]
	if state.Workflows[0].Status != "expired" || step.Payload != nil || !strings.Contains(step.Review, "main.js") || step.PayloadHash == "" {
		t.Fatal("expected only the review and hash after expiry")
	}
}
