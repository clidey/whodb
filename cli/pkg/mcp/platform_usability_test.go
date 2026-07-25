/*
 * Copyright 2026 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 */

package mcp

import (
	"testing"
)

func TestPlatformWorkflowRecipeChoosesGoalSpecificPlan(t *testing.T) {
	tests := []struct {
		goal   string
		recipe string
		tool   string
	}{
		{goal: "set up an ETL pipeline", recipe: "etl_pipeline", tool: "whodb_platform_transform_wait"},
		{goal: "model customers as an ontology", recipe: "ontology_model", tool: "whodb_platform_ontology"},
		{goal: "build an app", recipe: "application", tool: "whodb_platform_app"},
		{goal: "help me understand this project", recipe: "platform_discovery", tool: "whodb_platform_workspace_summary"},
	}
	for _, test := range tests {
		output := recipeForGoal(test.goal)
		if output.Recipe != test.recipe {
			t.Errorf("recipeForGoal(%q) = %q, want %q", test.goal, output.Recipe, test.recipe)
		}
		found := false
		for _, step := range output.Steps {
			for _, tool := range append(append(append([]string{}, step.ReadTools...), step.WriteTools...), step.VerifyTools...) {
				if tool == test.tool {
					found = true
				}
			}
		}
		if !found {
			t.Errorf("recipe %q does not mention %q", output.Recipe, test.tool)
		}
	}
}

func TestPaginatePlatformListFiltersAndLimits(t *testing.T) {
	data, count, truncated := paginatePlatformList([]map[string]any{
		{"name": "Customers", "status": "ready", "type": "table"},
		{"name": "Orders", "status": "draft", "type": "table"},
		{"name": "Customer transform", "status": "ready", "type": "scheduled"},
	}, "customer", "", "ready", "", "", 0, 1)
	items, ok := data.([]map[string]any)
	if !ok || len(items) != 1 || count != 1 || !truncated {
		t.Fatalf("paginatePlatformList() = %#v, count=%d, truncated=%v", data, count, truncated)
	}
	if items[0]["name"] != "Customers" {
		t.Fatalf("paginatePlatformList() returned %#v", items[0])
	}
}

func TestTransformRunTerminalStatuses(t *testing.T) {
	for _, status := range []string{"success", "completed", "failed", "cancelled"} {
		if !isTerminalTransformRunStatus(status) {
			t.Errorf("status %q should be terminal", status)
		}
	}
	for _, status := range []string{"queued", "running", "pending"} {
		if isTerminalTransformRunStatus(status) {
			t.Errorf("status %q should not be terminal", status)
		}
	}
}

func TestPlatformIdempotencyKeyIsWorkspaceScoped(t *testing.T) {
	session := testPlatformSession(nil)
	key := scopedPlatformIdempotencyKey(session, PlatformGenericWriteInput{Resource: "dataset", IdempotencyKey: "request-1"}, "create")
	if key == "" || key == "request-1" || key == "request-1|create" {
		t.Fatalf("idempotency key was not scoped: %q", key)
	}
	if scopedPlatformIdempotencyKey(session, PlatformGenericWriteInput{Resource: "dataset"}, "create") != "" {
		t.Fatal("empty idempotency key should remain disabled")
	}
}
