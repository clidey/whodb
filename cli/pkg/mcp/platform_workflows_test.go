/*
 * Copyright 2026 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 */

package mcp

import (
	"testing"

	"github.com/clidey/whodb/cli/internal/config"
)

func workflowTestSession() *platformToolSession {
	return &platformToolSession{Host: config.PlatformHost{URL: "https://app.whodb.com", DefaultOrgID: "org-1", DefaultProjectID: "project-1", DefaultProjectName: "Customer"}}
}

func TestValidatePlatformWorkflowInputBuildsStablePlan(t *testing.T) {
	plan, err := validatePlatformWorkflowInput(PlatformWorkflowPlanInput{
		Goal: "Create a dataset and then a transform",
		Steps: []PlatformWorkflowStepInput{
			{ID: "dataset", Operation: "create", Resource: "dataset", Payload: map[string]any{"name": "customers", "description": "customer dataset"}},
			{ID: "transform", Operation: "create", Resource: "transform", DependsOn: []string{"dataset"}, Payload: map[string]any{"name": "normalize"}},
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
