/*
 * Copyright 2026 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 */

package mcp

import (
	"context"
	"testing"

	platformapi "github.com/clidey/whodb/cli/internal/platform"
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

func TestMatchPlatformResourcesPrefersExactIDsAndNames(t *testing.T) {
	items := []PlatformResolvedResource{{ID: "ds-1", Name: "Customers"}, {ID: "ds-2", Name: "Customer Events"}}
	if got := matchPlatformResources(items, "ds-1"); len(got) != 1 || got[0].Match != "id" {
		t.Fatalf("id resolution = %#v", got)
	}
	if got := matchPlatformResources(items, "Customers"); len(got) != 1 || got[0].ID != "ds-1" || got[0].Match != "exact_name" {
		t.Fatalf("exact name resolution = %#v", got)
	}
	if got := matchPlatformResources(items, "customer"); len(got) != 2 {
		t.Fatalf("fragment resolution = %#v", got)
	}
}

func TestValidatePlatformPayloadUsesKnownShapeAndTypes(t *testing.T) {
	if err := validatePlatformPayload("create:secret", map[string]any{"name": "KEY"}); err == nil {
		t.Fatal("missing required secret value was accepted")
	}
	if err := validatePlatformPayload("create:secret", map[string]any{"name": "KEY", "value": true}); err == nil {
		t.Fatal("wrong secret value type was accepted")
	}
	if err := validatePlatformPayload("create:secret", map[string]any{"name": "KEY", "value": "secret"}); err != nil {
		t.Fatalf("valid secret payload rejected: %v", err)
	}
}

func TestValidatePlatformManifestMutationIsCapabilityAware(t *testing.T) {
	manifest := &platformapi.PlatformManifest{Operations: []platformapi.PlatformManifestOperation{{Kind: "Mutation", Name: "CreateDataset"}}}
	if err := validatePlatformManifestMutation(manifest, "CreateDataset"); err != nil {
		t.Fatalf("published mutation rejected: %v", err)
	}
	if err := validatePlatformManifestMutation(manifest, "DeleteDataset"); err == nil {
		t.Fatal("unpublished mutation accepted")
	}
	if err := validatePlatformManifestMutation(&platformapi.PlatformManifest{}, "DeleteDataset"); err != nil {
		t.Fatalf("empty development manifest should not block writes: %v", err)
	}
}

func TestPlatformRecoveryAdviceIsActionable(t *testing.T) {
	advice := platformRecoveryAdvice(string(PlatformErrorPermission), "forbidden")
	if advice.LikelyCause == "" || len(advice.NextSteps) < 2 {
		t.Fatalf("permission recovery advice is incomplete: %#v", advice)
	}
}

func TestPlatformWritePreflightDoesNotClaimAuthorization(t *testing.T) {
	client := &fakePlatformClient{platformQueryResult: []string{"read"}}
	session := testPlatformSession(client)
	checks := platformWritePreflight(context.Background(), session, platformapi.GenericWriteSpec{Resource: "dataset", Action: "update", Mode: platformapi.GenericWriteModeProjectID, NeedsID: true}, nil, "ds-1")
	for _, check := range checks {
		if check.Name == "permission" && check.Status == "ready" {
			t.Fatal("preflight must not claim authorization from a generic permission response")
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
