/*
 * Copyright 2026 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 */

package mcp

import (
	"context"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// PlatformWorkflowRecipeInput requests a goal-oriented platform workflow recipe.
type PlatformWorkflowRecipeInput struct {
	Goal   string   `json:"goal" jsonschema:"Desired end-to-end platform outcome, for example set up ETL or create an ontology"`
	Fields []string `json:"fields,omitempty" jsonschema:"Optional top-level output fields to include"`
}

// PlatformWorkflowRecipeOutput describes a safe, ordered workflow without executing writes.
type PlatformWorkflowRecipeOutput struct {
	Goal        string               `json:"goal"`
	Recipe      string               `json:"recipe"`
	Description string               `json:"description"`
	Steps       []PlatformRecipeStep `json:"steps"`
	ReadFirst   []string             `json:"read_first"`
	VerifyWith  []string             `json:"verify_with"`
	Warnings    []string             `json:"warnings,omitempty"`
}

// PlatformRecipeStep is one phase in a goal-oriented workflow recipe.
type PlatformRecipeStep struct {
	Order       int      `json:"order"`
	Name        string   `json:"name"`
	Objective   string   `json:"objective"`
	ReadTools   []string `json:"read_tools,omitempty"`
	WriteTools  []string `json:"write_tools,omitempty"`
	VerifyTools []string `json:"verify_tools,omitempty"`
	Notes       []string `json:"notes,omitempty"`
}

func recipeForGoal(goal string) PlatformWorkflowRecipeOutput {
	goal = strings.TrimSpace(goal)
	lower := strings.ToLower(goal)
	recipe := PlatformWorkflowRecipeOutput{Goal: goal, ReadFirst: []string{"whodb_platform_setup_status", "whodb_platform_workspace_summary", "whodb_platform_gap_analysis"}}
	switch {
	case strings.Contains(lower, "etl") || strings.Contains(lower, "pipeline") || strings.Contains(lower, "transform"):
		recipe.Recipe = "etl_pipeline"
		recipe.Description = "Inspect sources, persist or promote data, build a transform, run it, and verify the resulting workspace."
		recipe.Steps = []PlatformRecipeStep{
			{Order: 1, Name: "inspect", Objective: "Understand available sources, files, datasets, and runtime readiness.", ReadTools: []string{"whodb_platform_workspace_map", "whodb_platform_file_inspect", "whodb_platform_runtime_readiness"}},
			{Order: 2, Name: "persist", Objective: "Create or promote the input dataset only after inspecting its schema.", WriteTools: []string{"whodb_platform_create_dataset", "whodb_platform_promote_file_to_dataset"}, VerifyTools: []string{"whodb_platform_dataset", "whodb_platform_dataset_rows"}},
			{Order: 3, Name: "transform", Objective: "Create or update the transform and review its impact before writing.", ReadTools: []string{"whodb_platform_write_plan", "whodb_platform_change_impact"}, WriteTools: []string{"whodb_platform_create", "whodb_platform_update"}},
			{Order: 4, Name: "run", Objective: "Run the transform and wait for the real transform run to reach a terminal state.", WriteTools: []string{"whodb_platform_action"}, VerifyTools: []string{"whodb_platform_transform_wait", "whodb_platform_transform_runs"}},
		}
		recipe.VerifyWith = []string{"whodb_platform_workspace_summary", "whodb_platform_resource_graph", "whodb_platform_dataset_rows"}
	case strings.Contains(lower, "ontology") || strings.Contains(lower, "data model") || strings.Contains(lower, "object type"):
		recipe.Recipe = "ontology_model"
		recipe.Description = "Inspect source constraints, define ontology types and links, then verify the model against real rows."
		recipe.Steps = []PlatformRecipeStep{
			{Order: 1, Name: "inspect", Objective: "Find the source object and inspect columns, keys, and sample rows.", ReadTools: []string{"whodb_platform_workspace_summary", "whodb_platform_source_constraints", "whodb_platform_source_content"}},
			{Order: 2, Name: "model", Objective: "Create or update ontology types and relationships through the confirmation flow.", ReadTools: []string{"whodb_platform_write_plan", "whodb_platform_change_impact"}, WriteTools: []string{"whodb_platform_create", "whodb_platform_update"}},
			{Order: 3, Name: "verify", Objective: "Read the ontology and inspect representative rows and links.", VerifyTools: []string{"whodb_platform_ontology", "whodb_platform_ontology_rows", "whodb_platform_resource_graph"}},
		}
		recipe.VerifyWith = []string{"whodb_platform_ontology", "whodb_platform_ontology_rows", "whodb_platform_resource_graph"}
	case strings.Contains(lower, "app") || strings.Contains(lower, "application"):
		recipe.Recipe = "application"
		recipe.Description = "Understand the data model and runtime dependencies, then create, generate, and verify an application."
		recipe.Steps = []PlatformRecipeStep{
			{Order: 1, Name: "scope", Objective: "Identify the ontologies, functions, providers, and secrets the app will use.", ReadTools: []string{"whodb_platform_workspace_summary", "whodb_platform_data_model_summary", "whodb_platform_runtime_readiness"}},
			{Order: 2, Name: "build", Objective: "Create or update the app and its files with an explicit preview.", ReadTools: []string{"whodb_platform_write_plan", "whodb_platform_change_impact"}, WriteTools: []string{"whodb_platform_create", "whodb_platform_update", "whodb_platform_action"}},
			{Order: 3, Name: "verify", Objective: "Read the app view and files, then inspect affected resources.", VerifyTools: []string{"whodb_platform_app", "whodb_platform_app_view", "whodb_platform_app_files", "whodb_platform_resource_graph"}},
		}
		recipe.VerifyWith = []string{"whodb_platform_app", "whodb_platform_app_view", "whodb_platform_resource_graph"}
	default:
		recipe.Recipe = "platform_discovery"
		recipe.Description = "Start with a compact workspace summary, identify gaps, and choose a platform-specific workflow before making changes."
		recipe.Steps = []PlatformRecipeStep{{Order: 1, Name: "discover", Objective: "Understand the selected workspace and its available capabilities.", ReadTools: []string{"whodb_platform_workspace_summary", "whodb_platform_workspace_map", "whodb_platform_gap_analysis"}}, {Order: 2, Name: "plan", Objective: "Use the discovered resources to create a validated workflow plan.", ReadTools: []string{"whodb_platform_build_plan", "whodb_platform_write_plan"}}, {Order: 3, Name: "execute", Objective: "Apply only after reviewing the exact confirmation preview.", WriteTools: []string{"whodb_platform_workflow_apply"}, VerifyTools: []string{"whodb_platform_workspace_summary"}}}
		recipe.VerifyWith = []string{"whodb_platform_workspace_summary", "whodb_platform_resource_graph"}
	}
	return recipe
}

// HandlePlatformWorkflowRecipe returns a deterministic recipe without contacting the hosted platform.
func HandlePlatformWorkflowRecipe(_ context.Context, _ *mcp.CallToolRequest, input PlatformWorkflowRecipeInput) (*mcp.CallToolResult, PlatformWorkflowRecipeOutput, error) {
	if strings.TrimSpace(input.Goal) == "" {
		return nil, PlatformWorkflowRecipeOutput{Warnings: []string{"goal is required"}}, nil
	}
	return nil, recipeForGoal(input.Goal), nil
}

func platformWorkflowRecipeToolDefinition() *mcp.Tool {
	return &mcp.Tool{Name: "whodb_platform_workflow_recipe", Description: "Return a goal-oriented, read-first recipe for using the hosted WhoDB platform. This never executes writes; use its steps to build and apply a confirmed workflow.", Annotations: platformReadOnlyAnnotations("Plan Hosted Workflow Recipe")}
}
