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
	case strings.Contains(lower, "how many") || strings.Contains(lower, "question") || strings.Contains(lower, "report") || strings.Contains(lower, "analy") || strings.Contains(lower, "count") || strings.Contains(lower, "total") || strings.Contains(lower, "average"):
		recipe.Recipe = "data_question"
		recipe.Description = "Learn the ontology model, then answer the question with bounded queries or exact aggregates and show how the numbers were produced."
		recipe.Steps = []PlatformRecipeStep{
			{Order: 1, Name: "understand", Objective: "Learn entity names, properties and relationships before guessing field names.", ReadTools: []string{"whodb_platform_ontology_describe", "whodb_platform_ontologies"}},
			{Order: 2, Name: "profile", Objective: "Check value ranges and null counts for the properties the question depends on.", ReadTools: []string{"whodb_platform_ontology_stats"}},
			{Order: 3, Name: "answer", Objective: "Use ontology_aggregate for exact totals and group-bys; use ontology_query for filtered rows, search and joins. State the filter used.", ReadTools: []string{"whodb_platform_ontology_aggregate", "whodb_platform_ontology_query"}, Notes: []string{"ontology_query scans at most scan_limit rows (max 5000); prefer ontology_aggregate when totals must be exact.", "Policy-protected ontologies only allow whodb_platform_ontology_rows."}},
			{Order: 4, Name: "drill_down", Objective: "Fetch the specific rows or linked records behind a surprising number.", ReadTools: []string{"whodb_platform_ontology_rows", "whodb_platform_ontology_follow_link", "whodb_platform_ontology_similar"}},
		}
		recipe.VerifyWith = []string{"whodb_platform_ontology_aggregate", "whodb_platform_ontology_rows"}
	case strings.Contains(lower, "import") || strings.Contains(lower, "bulk") || strings.Contains(lower, "load rows") || strings.Contains(lower, "insert"):
		recipe.Recipe = "bulk_import"
		recipe.Description = "Validate the target ontology, insert rows in batches with an idempotency key, and verify counts."
		recipe.Steps = []PlatformRecipeStep{
			{Order: 1, Name: "inspect", Objective: "Confirm the ontology's properties, primary key and writable fields.", ReadTools: []string{"whodb_platform_ontology", "whodb_platform_ontology_record_capabilities"}},
			{Order: 2, Name: "insert", Objective: "Insert rows in batches with whodb_platform_add_ontology_records and a stable idempotency_key per batch so retries are safe.", WriteTools: []string{"whodb_platform_add_ontology_records"}, VerifyTools: []string{"whodb_platform_ontology_aggregate"}},
			{Order: 3, Name: "verify", Objective: "Compare the expected and actual row counts and spot-check rows.", VerifyTools: []string{"whodb_platform_ontology_aggregate", "whodb_platform_ontology_rows"}},
		}
		recipe.VerifyWith = []string{"whodb_platform_ontology_aggregate", "whodb_platform_ontology_rows"}
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
		recipe.Description = "Inspect real resources, build editable app files, exercise workflows and responsive views, then review before promotion. Read whodb://platform/app-workflow for hosted MCP and local CLI paths."
		recipe.Steps = []PlatformRecipeStep{
			{Order: 1, Name: "scope", Objective: "Identify users, jobs, ontologies, functions, and real sample data before planning pages and interactions.", ReadTools: []string{"whodb_platform_workspace_summary", "whodb_platform_ontology_describe", "whodb_platform_ontology_rows", "whodb_platform_functions", "whodb_platform_runtime_readiness"}},
			{Order: 2, Name: "build", Objective: "Plan page, state, and event flows; look up exact component and SDK capabilities, then create or update the app's editable draft files.", ReadTools: []string{"whodb_platform_apps", "whodb_platform_app", "whodb_platform_app_files", "whodb_platform_app_components", "whodb_platform_app_component", "whodb_platform_app_sdks", "whodb_platform_app_sdk", "whodb_platform_write_plan", "whodb_platform_change_impact"}, WriteTools: []string{"whodb_platform_create", "whodb_platform_update", "whodb_platform_action"}, Notes: []string{"With filesystem access, use whodb apps init --from, apps check, and apps publish --draft for the version-controlled path."}},
			{Order: 3, Name: "verify", Objective: "Audit event wiring, exercise each promised interaction, inspect rendered desktop/mobile/tablet views and diagnostics, then read back the draft files.", VerifyTools: []string{"whodb_platform_app_view", "whodb_platform_app_files", "whodb_platform_app_logic_graph", "whodb_platform_app_screenshot", "whodb_platform_resource_graph"}, Notes: []string{"With filesystem access, use whodb apps verify for local-file screenshots and a handler-to-bridge flow trace."}},
			{Order: 4, Name: "promote", Objective: "Review the exact draft and promote only after the workflow passes verification.", ReadTools: []string{"whodb_platform_write_plan"}, WriteTools: []string{"whodb_platform_action"}, VerifyTools: []string{"whodb_platform_object_versions", "whodb_platform_active_version"}},
		}
		recipe.VerifyWith = []string{"whodb_platform_app_view", "whodb_platform_app_screenshot", "whodb_platform_object_versions"}
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
