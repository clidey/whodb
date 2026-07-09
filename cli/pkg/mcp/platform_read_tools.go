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

package mcp

import (
	"context"
	"strings"
	"time"

	platformapi "github.com/clidey/whodb/cli/internal/platform"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const defaultPlatformContentLimit = 64 * 1024

// PlatformSourceConstraintsInput is the input for the whodb_platform_source_constraints tool.
type PlatformSourceConstraintsInput struct {
	Source string   `json:"source" jsonschema:"Hosted source id or name"`
	Ref    string   `json:"ref" jsonschema:"Object ref as kind:path, for example table:public.users"`
	Fields []string `json:"fields,omitempty" jsonschema:"Optional top-level output fields to include"`
}

// PlatformSourceContentInput is the input for the whodb_platform_source_content tool.
type PlatformSourceContentInput struct {
	Source string   `json:"source" jsonschema:"Hosted source id or name"`
	Ref    string   `json:"ref" jsonschema:"Object ref as kind:path, for example file:notes/report.txt"`
	Fields []string `json:"fields,omitempty" jsonschema:"Top-level output fields to include. Prefer only fields needed now; call again with more fields if needed."`
}

// PlatformEntityInput is a selected-project input with one resource id.
type PlatformEntityInput struct {
	ID     string   `json:"id" jsonschema:"Resource id"`
	Fields []string `json:"fields,omitempty" jsonschema:"Top-level output fields to include. Prefer only fields needed now; call again with more fields if needed."`
}

// PlatformProviderModelsInput is the input for the whodb_platform_ai_provider_models tool.
type PlatformProviderModelsInput struct {
	ProviderID string   `json:"provider_id" jsonschema:"Hosted AI provider id"`
	Fields     []string `json:"fields,omitempty" jsonschema:"Optional top-level output fields to include"`
}

// PlatformRowsInput is a selected-project row-preview input.
type PlatformRowsInput struct {
	ID     string   `json:"id" jsonschema:"Resource id"`
	Limit  int      `json:"limit,omitempty" jsonschema:"Maximum rows to return"`
	Offset int      `json:"offset,omitempty" jsonschema:"Row offset"`
	Fields []string `json:"fields,omitempty" jsonschema:"Optional top-level output fields to include"`
}

// PlatformOntologyFollowLinkInput is the input for the whodb_platform_ontology_follow_link tool.
type PlatformOntologyFollowLinkInput struct {
	EntityID    string   `json:"entity_id" jsonschema:"Ontology id"`
	PrimaryKey  string   `json:"primary_key" jsonschema:"Primary key value of the source ontology row"`
	LinkAPIName string   `json:"link_api_name" jsonschema:"Ontology link apiName to follow"`
	Limit       int      `json:"limit,omitempty" jsonschema:"Maximum rows to return"`
	Offset      int      `json:"offset,omitempty" jsonschema:"Row offset"`
	Fields      []string `json:"fields,omitempty" jsonschema:"Optional top-level output fields to include"`
}

// PlatformLineageInput is the input for the whodb_platform_lineage tool.
type PlatformLineageInput struct {
	RootID    string   `json:"root_id" jsonschema:"Root node id"`
	RootType  string   `json:"root_type" jsonschema:"Root node type"`
	Direction string   `json:"direction,omitempty" jsonschema:"Optional lineage direction"`
	MaxDepth  int      `json:"max_depth,omitempty" jsonschema:"Optional maximum graph depth"`
	Fields    []string `json:"fields,omitempty" jsonschema:"Optional top-level output fields to include"`
}

// PlatformLineageNeighborsInput is the input for the whodb_platform_lineage_neighbors tool.
type PlatformLineageNeighborsInput struct {
	NodeID   string   `json:"node_id" jsonschema:"Node id"`
	NodeType string   `json:"node_type" jsonschema:"Node type"`
	Fields   []string `json:"fields,omitempty" jsonschema:"Optional top-level output fields to include"`
}

// PlatformTransformRunsInput is the input for the whodb_platform_transform_runs tool.
type PlatformTransformRunsInput struct {
	TransformID string   `json:"transform_id" jsonschema:"Transform id"`
	Limit       int      `json:"limit,omitempty" jsonschema:"Maximum runs to return"`
	Fields      []string `json:"fields,omitempty" jsonschema:"Optional top-level output fields to include"`
}

// PlatformFilesInput is the input for the whodb_platform_files tool.
type PlatformFilesInput struct {
	FolderID string   `json:"folder_id,omitempty" jsonschema:"Folder id. Omit for project root."`
	Name     string   `json:"name,omitempty" jsonschema:"Optional case-insensitive name substring filter"`
	Kind     string   `json:"kind,omitempty" jsonschema:"Optional entry kind filter: file or folder"`
	MIMEType string   `json:"mime_type,omitempty" jsonschema:"Optional file MIME type substring filter"`
	Fields   []string `json:"fields,omitempty" jsonschema:"Top-level output fields to include. Prefer only fields needed now; call again with more fields if needed."`
}

// PlatformFilePreviewInput is the input for the whodb_platform_file_preview tool.
type PlatformFilePreviewInput struct {
	FileID     string   `json:"file_id" jsonschema:"Project file id"`
	SheetIndex *int     `json:"sheet_index,omitempty" jsonschema:"Optional spreadsheet sheet index"`
	Fields     []string `json:"fields,omitempty" jsonschema:"Top-level output fields to include. Prefer metadata first; call again with body fields only if needed."`
}

// PlatformFileInspectInput is the input for the whodb_platform_file_inspect tool.
type PlatformFileInspectInput struct {
	FileID      string   `json:"file_id" jsonschema:"Project file id"`
	SheetIndex  *int     `json:"sheet_index,omitempty" jsonschema:"Optional spreadsheet sheet index"`
	IncludeRows bool     `json:"include_rows,omitempty" jsonschema:"Include preview rows. Defaults to false to keep context compact."`
	Fields      []string `json:"fields,omitempty" jsonschema:"Top-level output fields to include. Prefer columns and columnMapExample before requesting rows."`
}

// PlatformFileSearchInput is the input for the whodb_platform_file_search tool.
type PlatformFileSearchInput struct {
	Query  string   `json:"query" jsonschema:"File search query"`
	Fields []string `json:"fields,omitempty" jsonschema:"Optional top-level output fields to include"`
}

// PlatformEmptyInput is the input for selected-project list tools.
type PlatformEmptyInput struct {
	Name       string   `json:"name,omitempty" jsonschema:"Optional case-insensitive name substring filter for list tools"`
	Type       string   `json:"type,omitempty" jsonschema:"Optional type filter, for example provider type, transform trigger mode, or function language"`
	Status     string   `json:"status,omitempty" jsonschema:"Optional status filter for resources that expose status"`
	SchemaMode string   `json:"schema_mode,omitempty" jsonschema:"Optional dataset schema mode filter"`
	Deployed   string   `json:"deployed,omitempty" jsonschema:"Optional function deployment filter: true or false"`
	Fields     []string `json:"fields,omitempty" jsonschema:"Top-level output fields to include. Prefer only fields needed now; call again with more fields if needed."`
}

// PlatformReadOutput is the common output for read-only hosted platform tools.
type PlatformReadOutput struct {
	PlatformSetupGuidance
	Data      any                  `json:"data,omitempty"`
	Items     []map[string]any     `json:"items,omitempty"`
	Count     int                  `json:"count"`
	Scope     *PlatformOutputScope `json:"scope,omitempty"`
	Fields    []string             `json:"fields,omitempty"`
	Warnings  []string             `json:"warnings,omitempty"`
	Truncated bool                 `json:"truncated"`
	Error     string               `json:"error,omitempty"`
	RequestID string               `json:"request_id,omitempty"`
}

func registerPlatformReadTool(server *mcp.Server, tool *mcp.Tool, secOpts *SecurityOptions) {
	switch tool.Name {
	case "whodb_platform_workspace_map":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformWorkspaceMapInput) (*mcp.CallToolResult, any, error) {
			return HandlePlatformWorkspaceMap(ctx, req, input)
		})
	case "whodb_platform_resource_graph":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformResourceGraphInput) (*mcp.CallToolResult, any, error) {
			return HandlePlatformResourceGraph(ctx, req, input)
		})
	case "whodb_platform_next_actions":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformNextActionsInput) (*mcp.CallToolResult, any, error) {
			return HandlePlatformNextActions(ctx, req, input)
		})
	case "whodb_platform_workspace_summary":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformWorkspaceSummaryInput) (*mcp.CallToolResult, any, error) {
			return HandlePlatformWorkspaceSummary(ctx, req, input)
		})
	case "whodb_platform_build_plan":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformBuildPlanInput) (*mcp.CallToolResult, any, error) {
			return HandlePlatformBuildPlan(ctx, req, input)
		})
	case "whodb_platform_gap_analysis":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformGapAnalysisInput) (*mcp.CallToolResult, any, error) {
			return HandlePlatformGapAnalysis(ctx, req, input)
		})
	case "whodb_platform_project_health":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformNextActionsInput) (*mcp.CallToolResult, any, error) {
			return HandlePlatformProjectHealth(ctx, req, input)
		})
	case "whodb_platform_data_model_summary":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformResourceGraphInput) (*mcp.CallToolResult, any, error) {
			return HandlePlatformDataModelSummary(ctx, req, input)
		})
	case "whodb_platform_runtime_readiness":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformNextActionsInput) (*mcp.CallToolResult, any, error) {
			return HandlePlatformRuntimeReadiness(ctx, req, input)
		})
	case "whodb_platform_change_impact":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformChangeImpactInput) (*mcp.CallToolResult, any, error) {
			return HandlePlatformChangeImpact(ctx, req, input)
		})
	case "whodb_platform_write_plan":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformWritePlanInput) (*mcp.CallToolResult, any, error) {
			return HandlePlatformWritePlan(ctx, req, input)
		})
	case "whodb_platform_source_constraints":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformSourceConstraintsInput) (*mcp.CallToolResult, any, error) {
			return HandlePlatformSourceConstraints(ctx, req, input)
		})
	case "whodb_platform_source_content":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformSourceContentInput) (*mcp.CallToolResult, any, error) {
			return HandlePlatformSourceContent(ctx, req, input)
		})
	case "whodb_platform_secrets":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformEmptyInput) (*mcp.CallToolResult, any, error) {
			return HandlePlatformSecrets(ctx, req, input)
		})
	case "whodb_platform_ai_providers":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformEmptyInput) (*mcp.CallToolResult, any, error) {
			return HandlePlatformAIProviders(ctx, req, input)
		})
	case "whodb_platform_ai_provider_models":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformProviderModelsInput) (*mcp.CallToolResult, any, error) {
			return HandlePlatformAIProviderModels(ctx, req, input)
		})
	case "whodb_platform_ontologies":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformEmptyInput) (*mcp.CallToolResult, any, error) {
			return HandlePlatformOntologies(ctx, req, input)
		})
	case "whodb_platform_ontology":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformEntityInput) (*mcp.CallToolResult, any, error) {
			return HandlePlatformOntology(ctx, req, input)
		})
	case "whodb_platform_ontology_fast_lookups":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformEntityInput) (*mcp.CallToolResult, any, error) {
			return HandlePlatformOntologyFastLookups(ctx, req, input)
		})
	case "whodb_platform_ontology_fast_lookup_suggestions":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformEntityInput) (*mcp.CallToolResult, any, error) {
			return HandlePlatformOntologyFastLookupSuggestions(ctx, req, input)
		})
	case "whodb_platform_ontology_rows":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformRowsInput) (*mcp.CallToolResult, any, error) {
			return HandlePlatformOntologyRows(ctx, req, input, secOpts)
		})
	case "whodb_platform_ontology_follow_link":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformOntologyFollowLinkInput) (*mcp.CallToolResult, any, error) {
			return HandlePlatformOntologyFollowLink(ctx, req, input, secOpts)
		})
	case "whodb_platform_datasets":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformEmptyInput) (*mcp.CallToolResult, any, error) {
			return HandlePlatformDatasets(ctx, req, input)
		})
	case "whodb_platform_dataset":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformEntityInput) (*mcp.CallToolResult, any, error) {
			return HandlePlatformDataset(ctx, req, input)
		})
	case "whodb_platform_dataset_rows":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformRowsInput) (*mcp.CallToolResult, any, error) {
			return HandlePlatformDatasetRows(ctx, req, input, secOpts)
		})
	case "whodb_platform_lineage":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformLineageInput) (*mcp.CallToolResult, any, error) {
			return HandlePlatformLineage(ctx, req, input)
		})
	case "whodb_platform_lineage_neighbors":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformLineageNeighborsInput) (*mcp.CallToolResult, any, error) {
			return HandlePlatformLineageNeighbors(ctx, req, input)
		})
	case "whodb_platform_project_lineage":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformEmptyInput) (*mcp.CallToolResult, any, error) {
			return HandlePlatformProjectLineage(ctx, req, input)
		})
	case "whodb_platform_transforms":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformEmptyInput) (*mcp.CallToolResult, any, error) {
			return HandlePlatformTransforms(ctx, req, input)
		})
	case "whodb_platform_transform":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformEntityInput) (*mcp.CallToolResult, any, error) {
			return HandlePlatformTransform(ctx, req, input)
		})
	case "whodb_platform_transform_runs":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformTransformRunsInput) (*mcp.CallToolResult, any, error) {
			return HandlePlatformTransformRuns(ctx, req, input)
		})
	case "whodb_platform_functions":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformEmptyInput) (*mcp.CallToolResult, any, error) {
			return HandlePlatformFunctions(ctx, req, input)
		})
	case "whodb_platform_function":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformEntityInput) (*mcp.CallToolResult, any, error) {
			return HandlePlatformFunction(ctx, req, input)
		})
	case "whodb_platform_files":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformFilesInput) (*mcp.CallToolResult, any, error) {
			return HandlePlatformFiles(ctx, req, input)
		})
	case "whodb_platform_file_preview":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformFilePreviewInput) (*mcp.CallToolResult, any, error) {
			return HandlePlatformFilePreview(ctx, req, input)
		})
	case "whodb_platform_file_inspect":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformFileInspectInput) (*mcp.CallToolResult, any, error) {
			return HandlePlatformFileInspect(ctx, req, input)
		})
	case "whodb_platform_file_search":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformFileSearchInput) (*mcp.CallToolResult, any, error) {
			return HandlePlatformFileSearch(ctx, req, input)
		})
	case "whodb_platform_tabular_files":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformEmptyInput) (*mcp.CallToolResult, any, error) {
			return HandlePlatformTabularFiles(ctx, req, input)
		})
	case "whodb_platform_storage_usage":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformEmptyInput) (*mcp.CallToolResult, any, error) {
			return HandlePlatformStorageUsage(ctx, req, input)
		})
	}
}

func platformReadToolDefinitions() []*mcp.Tool {
	return []*mcp.Tool{
		{Name: "whodb_platform_workspace_map", Description: descPlatformWorkspaceMap, Annotations: platformReadOnlyAnnotations("Map Hosted Workspace")},
		{Name: "whodb_platform_resource_graph", Description: descPlatformResourceGraph, Annotations: platformReadOnlyAnnotations("Graph Hosted Resources")},
		{Name: "whodb_platform_next_actions", Description: descPlatformNextActions, Annotations: platformReadOnlyAnnotations("Suggest Hosted Next Actions")},
		{Name: "whodb_platform_workspace_summary", Description: descPlatformWorkspaceSummary, Annotations: platformReadOnlyAnnotations("Summarize Hosted Workspace")},
		{Name: "whodb_platform_build_plan", Description: descPlatformBuildPlan, Annotations: platformReadOnlyAnnotations("Plan Hosted Build")},
		{Name: "whodb_platform_gap_analysis", Description: descPlatformGapAnalysis, Annotations: platformReadOnlyAnnotations("Analyze Hosted Gaps")},
		{Name: "whodb_platform_project_health", Description: descPlatformProjectHealth, Annotations: platformReadOnlyAnnotations("Summarize Hosted Project Health")},
		{Name: "whodb_platform_data_model_summary", Description: descPlatformDataModelSummary, Annotations: platformReadOnlyAnnotations("Summarize Hosted Data Model")},
		{Name: "whodb_platform_runtime_readiness", Description: descPlatformRuntimeReadiness, Annotations: platformReadOnlyAnnotations("Check Hosted Runtime Readiness")},
		{Name: "whodb_platform_change_impact", Description: descPlatformChangeImpact, Annotations: platformReadOnlyAnnotations("Analyze Hosted Change Impact")},
		{Name: "whodb_platform_write_plan", Description: descPlatformWritePlan, Annotations: platformReadOnlyAnnotations("Plan Hosted Write")},
		{Name: "whodb_platform_source_constraints", Description: descPlatformSourceConstraints, Annotations: platformReadOnlyAnnotations("Inspect Hosted Source Constraints")},
		{Name: "whodb_platform_source_content", Description: descPlatformSourceContent, Annotations: platformReadOnlyAnnotations("Read Hosted Source Content")},
		{Name: "whodb_platform_secrets", Description: descPlatformSecrets, Annotations: platformReadOnlyAnnotations("List Hosted Secret Metadata")},
		{Name: "whodb_platform_ai_providers", Description: descPlatformAIProviders, Annotations: platformReadOnlyAnnotations("List Hosted AI Providers")},
		{Name: "whodb_platform_ai_provider_models", Description: descPlatformAIProviderModels, Annotations: platformReadOnlyAnnotations("List Hosted AI Provider Models")},
		{Name: "whodb_platform_ontologies", Description: descPlatformOntologies, Annotations: platformReadOnlyAnnotations("List Hosted Ontologies")},
		{Name: "whodb_platform_ontology", Description: descPlatformOntology, Annotations: platformReadOnlyAnnotations("Inspect Hosted Ontology")},
		{Name: "whodb_platform_ontology_fast_lookups", Description: descPlatformOntologyFastLookups, Annotations: platformReadOnlyAnnotations("List Hosted Ontology Fast Lookups")},
		{Name: "whodb_platform_ontology_fast_lookup_suggestions", Description: descPlatformOntologyFastLookupSuggestions, Annotations: platformReadOnlyAnnotations("Suggest Hosted Ontology Fast Lookups")},
		{Name: "whodb_platform_ontology_rows", Description: descPlatformOntologyRows, Annotations: platformReadOnlyAnnotations("Preview Hosted Ontology Rows")},
		{Name: "whodb_platform_ontology_follow_link", Description: descPlatformOntologyFollowLink, Annotations: platformReadOnlyAnnotations("Follow Hosted Ontology Link")},
		{Name: "whodb_platform_datasets", Description: descPlatformDatasets, Annotations: platformReadOnlyAnnotations("List Hosted Datasets")},
		{Name: "whodb_platform_dataset", Description: descPlatformDataset, Annotations: platformReadOnlyAnnotations("Inspect Hosted Dataset")},
		{Name: "whodb_platform_dataset_rows", Description: descPlatformDatasetRows, Annotations: platformReadOnlyAnnotations("Preview Hosted Dataset Rows")},
		{Name: "whodb_platform_lineage", Description: descPlatformLineage, Annotations: platformReadOnlyAnnotations("Inspect Hosted Lineage")},
		{Name: "whodb_platform_lineage_neighbors", Description: descPlatformLineageNeighbors, Annotations: platformReadOnlyAnnotations("Inspect Hosted Lineage Neighbors")},
		{Name: "whodb_platform_project_lineage", Description: descPlatformProjectLineage, Annotations: platformReadOnlyAnnotations("Inspect Hosted Project Lineage")},
		{Name: "whodb_platform_transforms", Description: descPlatformTransforms, Annotations: platformReadOnlyAnnotations("List Hosted Transforms")},
		{Name: "whodb_platform_transform", Description: descPlatformTransform, Annotations: platformReadOnlyAnnotations("Inspect Hosted Transform")},
		{Name: "whodb_platform_transform_runs", Description: descPlatformTransformRuns, Annotations: platformReadOnlyAnnotations("List Hosted Transform Runs")},
		{Name: "whodb_platform_functions", Description: descPlatformFunctions, Annotations: platformReadOnlyAnnotations("List Hosted Functions")},
		{Name: "whodb_platform_function", Description: descPlatformFunction, Annotations: platformReadOnlyAnnotations("Inspect Hosted Function")},
		{Name: "whodb_platform_files", Description: descPlatformFiles, Annotations: platformReadOnlyAnnotations("List Hosted Files")},
		{Name: "whodb_platform_file_preview", Description: descPlatformFilePreview, Annotations: platformReadOnlyAnnotations("Preview Hosted File")},
		{Name: "whodb_platform_file_inspect", Description: descPlatformFileInspect, Annotations: platformReadOnlyAnnotations("Inspect Hosted File Columns")},
		{Name: "whodb_platform_file_search", Description: descPlatformFileSearch, Annotations: platformReadOnlyAnnotations("Search Hosted Files")},
		{Name: "whodb_platform_tabular_files", Description: descPlatformTabularFiles, Annotations: platformReadOnlyAnnotations("List Hosted Tabular Files")},
		{Name: "whodb_platform_storage_usage", Description: descPlatformStorageUsage, Annotations: platformReadOnlyAnnotations("Inspect Hosted Storage Usage")},
	}
}

// HandlePlatformSourceConstraints returns field constraints for one hosted source object.
func HandlePlatformSourceConstraints(ctx context.Context, req *mcp.CallToolRequest, input PlatformSourceConstraintsInput) (*mcp.CallToolResult, PlatformReadOutput, error) {
	return platformSourceRefRead(ctx, "platform_source_constraints", input.Source, input.Ref, input.Fields, func(ctx context.Context, session *platformToolSession, source *platformapi.Source, ref platformapi.SourceObjectRefInput) (any, int, bool, error) {
		constraints, err := session.Client.SourceFieldConstraints(ctx, session.Host.DefaultProjectID, source.ID, ref)
		return constraints, len(constraints), false, err
	})
}

// HandlePlatformSourceContent returns content for one hosted source object.
func HandlePlatformSourceContent(ctx context.Context, req *mcp.CallToolRequest, input PlatformSourceContentInput) (*mcp.CallToolResult, PlatformReadOutput, error) {
	return platformSourceRefRead(ctx, "platform_source_content", input.Source, input.Ref, input.Fields, func(ctx context.Context, session *platformToolSession, source *platformapi.Source, ref platformapi.SourceObjectRefInput) (any, int, bool, error) {
		content, err := session.Client.SourceContent(ctx, session.Host.DefaultProjectID, source.ID, ref, input.Fields)
		if err != nil || content == nil {
			return content, 0, false, err
		}
		truncated := truncateSourceContent(content, defaultPlatformContentLimit)
		return content, 1, truncated, nil
	})
}

// HandlePlatformSecrets lists secret metadata and usage without values.
func HandlePlatformSecrets(ctx context.Context, req *mcp.CallToolRequest, input PlatformEmptyInput) (*mcp.CallToolResult, PlatformReadOutput, error) {
	return platformProjectRead(ctx, "platform_secrets", input.Fields, func(ctx context.Context, session *platformToolSession) (any, int, bool, error) {
		secrets, err := session.Client.ProjectSecrets(ctx, session.Host.DefaultProjectID)
		secrets = filterPlatformReadSlice(secrets, func(secret platformapi.ProjectSecret) bool {
			return platformReadMatchesSubstring(secret.Name, input.Name)
		})
		return secrets, len(secrets), false, err
	})
}

// HandlePlatformAIProviders lists hosted AI provider metadata.
func HandlePlatformAIProviders(ctx context.Context, req *mcp.CallToolRequest, input PlatformEmptyInput) (*mcp.CallToolResult, PlatformReadOutput, error) {
	return platformProjectRead(ctx, "platform_ai_providers", input.Fields, func(ctx context.Context, session *platformToolSession) (any, int, bool, error) {
		providers, err := session.Client.AIProviders(ctx, session.Host.DefaultProjectID)
		providers = filterPlatformReadSlice(providers, func(provider platformapi.AIProvider) bool {
			return platformReadMatchesSubstring(provider.Name, input.Name) && platformReadMatchesEqual(provider.ProviderType, input.Type)
		})
		return providers, len(providers), false, err
	})
}

// HandlePlatformAIProviderModels lists model names for one hosted AI provider.
func HandlePlatformAIProviderModels(ctx context.Context, req *mcp.CallToolRequest, input PlatformProviderModelsInput) (*mcp.CallToolResult, PlatformReadOutput, error) {
	if strings.TrimSpace(input.ProviderID) == "" {
		return nil, PlatformReadOutput{Error: "provider_id is required", RequestID: generateRequestID("platform_ai_provider_models")}, nil
	}
	return platformProjectRead(ctx, "platform_ai_provider_models", input.Fields, func(ctx context.Context, session *platformToolSession) (any, int, bool, error) {
		models, err := session.Client.AIProviderModels(ctx, session.Host.DefaultProjectID, strings.TrimSpace(input.ProviderID))
		return models, len(models), false, err
	})
}

// HandlePlatformOntologies lists hosted ontology object types.
func HandlePlatformOntologies(ctx context.Context, req *mcp.CallToolRequest, input PlatformEmptyInput) (*mcp.CallToolResult, PlatformReadOutput, error) {
	return platformProjectRead(ctx, "platform_ontologies", input.Fields, func(ctx context.Context, session *platformToolSession) (any, int, bool, error) {
		ontologies, err := session.Client.Ontologies(ctx, session.Host.DefaultProjectID)
		ontologies = filterPlatformReadSlice(ontologies, func(ontology platformapi.Ontology) bool {
			nameMatch := platformReadMatchesSubstring(ontology.DisplayName, input.Name) || platformReadMatchesSubstring(ontology.APIName, input.Name)
			return nameMatch && platformReadMatchesEqual(ontology.Status, input.Status)
		})
		return ontologies, len(ontologies), false, err
	})
}

// HandlePlatformOntology returns one hosted ontology object type.
func HandlePlatformOntology(ctx context.Context, req *mcp.CallToolRequest, input PlatformEntityInput) (*mcp.CallToolResult, PlatformReadOutput, error) {
	return platformIDRead(ctx, "platform_ontology", input.ID, input.Fields, func(ctx context.Context, session *platformToolSession, id string) (any, int, bool, error) {
		ontology, err := session.Client.Ontology(ctx, session.Host.DefaultProjectID, id)
		return ontology, 1, false, err
	})
}

// HandlePlatformOntologyFastLookups lists fast lookups for one ontology.
func HandlePlatformOntologyFastLookups(ctx context.Context, req *mcp.CallToolRequest, input PlatformEntityInput) (*mcp.CallToolResult, PlatformReadOutput, error) {
	return platformIDRead(ctx, "platform_ontology_fast_lookups", input.ID, input.Fields, func(ctx context.Context, session *platformToolSession, id string) (any, int, bool, error) {
		lookups, err := session.Client.OntologyFastLookups(ctx, session.Host.DefaultProjectID, id)
		return lookups, len(lookups), false, err
	})
}

// HandlePlatformOntologyFastLookupSuggestions lists suggested fast lookups for one ontology.
func HandlePlatformOntologyFastLookupSuggestions(ctx context.Context, req *mcp.CallToolRequest, input PlatformEntityInput) (*mcp.CallToolResult, PlatformReadOutput, error) {
	return platformIDRead(ctx, "platform_ontology_fast_lookup_suggestions", input.ID, input.Fields, func(ctx context.Context, session *platformToolSession, id string) (any, int, bool, error) {
		suggestions, err := session.Client.OntologyFastLookupSuggestions(ctx, session.Host.DefaultProjectID, id)
		return suggestions, len(suggestions), false, err
	})
}

// HandlePlatformOntologyRows previews rows for one ontology.
func HandlePlatformOntologyRows(ctx context.Context, req *mcp.CallToolRequest, input PlatformRowsInput, secOpts *SecurityOptions) (*mcp.CallToolResult, PlatformReadOutput, error) {
	return platformRowsRead(ctx, "platform_ontology_rows", input, secOpts, func(ctx context.Context, session *platformToolSession, id string, limit, offset int) (*platformapi.DatasetQueryResult, error) {
		return session.Client.OntologyRows(ctx, session.Host.DefaultProjectID, id, limit, offset)
	})
}

// HandlePlatformOntologyFollowLink follows one ontology link from a row primary key.
func HandlePlatformOntologyFollowLink(ctx context.Context, req *mcp.CallToolRequest, input PlatformOntologyFollowLinkInput, secOpts *SecurityOptions) (*mcp.CallToolResult, PlatformReadOutput, error) {
	if strings.TrimSpace(input.EntityID) == "" || strings.TrimSpace(input.PrimaryKey) == "" || strings.TrimSpace(input.LinkAPIName) == "" {
		return nil, PlatformReadOutput{Error: "entity_id, primary_key, and link_api_name are required", RequestID: generateRequestID("platform_ontology_follow_link")}, nil
	}
	rowsInput := PlatformRowsInput{ID: input.EntityID, Limit: input.Limit, Offset: input.Offset, Fields: input.Fields}
	return platformRowsRead(ctx, "platform_ontology_follow_link", rowsInput, secOpts, func(ctx context.Context, session *platformToolSession, id string, limit, offset int) (*platformapi.DatasetQueryResult, error) {
		return session.Client.OntologyFollowLink(ctx, session.Host.DefaultProjectID, id, input.PrimaryKey, input.LinkAPIName, limit, offset)
	})
}

// HandlePlatformDatasets lists hosted datasets.
func HandlePlatformDatasets(ctx context.Context, req *mcp.CallToolRequest, input PlatformEmptyInput) (*mcp.CallToolResult, PlatformReadOutput, error) {
	return platformProjectRead(ctx, "platform_datasets", input.Fields, func(ctx context.Context, session *platformToolSession) (any, int, bool, error) {
		datasets, err := session.Client.Datasets(ctx, session.Host.DefaultProjectID)
		datasets = filterPlatformReadSlice(datasets, func(dataset platformapi.Dataset) bool {
			return platformReadMatchesSubstring(dataset.Name, input.Name) && platformReadMatchesEqual(dataset.SchemaMode, input.SchemaMode)
		})
		return datasets, len(datasets), false, err
	})
}

// HandlePlatformDataset returns one hosted dataset.
func HandlePlatformDataset(ctx context.Context, req *mcp.CallToolRequest, input PlatformEntityInput) (*mcp.CallToolResult, PlatformReadOutput, error) {
	return platformIDRead(ctx, "platform_dataset", input.ID, input.Fields, func(ctx context.Context, session *platformToolSession, id string) (any, int, bool, error) {
		dataset, err := session.Client.Dataset(ctx, session.Host.DefaultProjectID, id)
		return dataset, 1, false, err
	})
}

// HandlePlatformDatasetRows previews rows for one hosted dataset.
func HandlePlatformDatasetRows(ctx context.Context, req *mcp.CallToolRequest, input PlatformRowsInput, secOpts *SecurityOptions) (*mcp.CallToolResult, PlatformReadOutput, error) {
	return platformRowsRead(ctx, "platform_dataset_rows", input, secOpts, func(ctx context.Context, session *platformToolSession, id string, limit, offset int) (*platformapi.DatasetQueryResult, error) {
		return session.Client.DatasetRows(ctx, session.Host.DefaultProjectID, id, limit, offset)
	})
}

// HandlePlatformLineage returns lineage around one root node.
func HandlePlatformLineage(ctx context.Context, req *mcp.CallToolRequest, input PlatformLineageInput) (*mcp.CallToolResult, PlatformReadOutput, error) {
	if strings.TrimSpace(input.RootID) == "" || strings.TrimSpace(input.RootType) == "" {
		return nil, PlatformReadOutput{Error: "root_id and root_type are required", RequestID: generateRequestID("platform_lineage")}, nil
	}
	return platformProjectRead(ctx, "platform_lineage", input.Fields, func(ctx context.Context, session *platformToolSession) (any, int, bool, error) {
		graph, err := session.Client.Lineage(ctx, session.Host.DefaultProjectID, input.RootID, input.RootType, input.Direction, input.MaxDepth)
		return graph, lineageNodeCount(graph), false, err
	})
}

// HandlePlatformLineageNeighbors returns immediate lineage neighbors for one node.
func HandlePlatformLineageNeighbors(ctx context.Context, req *mcp.CallToolRequest, input PlatformLineageNeighborsInput) (*mcp.CallToolResult, PlatformReadOutput, error) {
	if strings.TrimSpace(input.NodeID) == "" || strings.TrimSpace(input.NodeType) == "" {
		return nil, PlatformReadOutput{Error: "node_id and node_type are required", RequestID: generateRequestID("platform_lineage_neighbors")}, nil
	}
	return platformProjectRead(ctx, "platform_lineage_neighbors", input.Fields, func(ctx context.Context, session *platformToolSession) (any, int, bool, error) {
		graph, err := session.Client.LineageNeighbors(ctx, session.Host.DefaultProjectID, input.NodeID, input.NodeType)
		return graph, lineageNodeCount(graph), false, err
	})
}

// HandlePlatformProjectLineage returns hosted project lineage.
func HandlePlatformProjectLineage(ctx context.Context, req *mcp.CallToolRequest, input PlatformEmptyInput) (*mcp.CallToolResult, PlatformReadOutput, error) {
	return platformProjectRead(ctx, "platform_project_lineage", input.Fields, func(ctx context.Context, session *platformToolSession) (any, int, bool, error) {
		graph, err := session.Client.ProjectLineage(ctx, session.Host.DefaultProjectID)
		return graph, lineageNodeCount(graph), false, err
	})
}

// HandlePlatformTransforms lists hosted transforms.
func HandlePlatformTransforms(ctx context.Context, req *mcp.CallToolRequest, input PlatformEmptyInput) (*mcp.CallToolResult, PlatformReadOutput, error) {
	return platformProjectRead(ctx, "platform_transforms", input.Fields, func(ctx context.Context, session *platformToolSession) (any, int, bool, error) {
		transforms, err := session.Client.Transforms(ctx, session.Host.DefaultProjectID)
		transforms = filterPlatformReadSlice(transforms, func(transform platformapi.Transform) bool {
			return platformReadMatchesSubstring(transform.Name, input.Name) && platformReadMatchesEqual(transform.TriggerMode, input.Type)
		})
		return transforms, len(transforms), false, err
	})
}

// HandlePlatformTransform returns one hosted transform.
func HandlePlatformTransform(ctx context.Context, req *mcp.CallToolRequest, input PlatformEntityInput) (*mcp.CallToolResult, PlatformReadOutput, error) {
	return platformIDRead(ctx, "platform_transform", input.ID, input.Fields, func(ctx context.Context, session *platformToolSession, id string) (any, int, bool, error) {
		transform, err := platformapi.ResolveTransform(ctx, session.Client, session.Host.DefaultProjectID, id)
		return transform, 1, false, err
	})
}

// HandlePlatformTransformRuns lists runs for one hosted transform.
func HandlePlatformTransformRuns(ctx context.Context, req *mcp.CallToolRequest, input PlatformTransformRunsInput) (*mcp.CallToolResult, PlatformReadOutput, error) {
	if strings.TrimSpace(input.TransformID) == "" {
		return nil, PlatformReadOutput{Error: "transform_id is required", RequestID: generateRequestID("platform_transform_runs")}, nil
	}
	return platformProjectRead(ctx, "platform_transform_runs", input.Fields, func(ctx context.Context, session *platformToolSession) (any, int, bool, error) {
		runs, err := session.Client.TransformRuns(ctx, session.Host.DefaultProjectID, input.TransformID, input.Limit)
		return runs, len(runs), false, err
	})
}

// HandlePlatformFunctions lists hosted ontology functions.
func HandlePlatformFunctions(ctx context.Context, req *mcp.CallToolRequest, input PlatformEmptyInput) (*mcp.CallToolResult, PlatformReadOutput, error) {
	return platformProjectRead(ctx, "platform_functions", input.Fields, func(ctx context.Context, session *platformToolSession) (any, int, bool, error) {
		functions, err := session.Client.Functions(ctx, session.Host.DefaultProjectID, input.Fields)
		functions = filterPlatformReadSlice(functions, func(fn platformapi.Function) bool {
			return platformReadMatchesSubstring(fn.Name, input.Name) && platformReadMatchesEqual(fn.Language, input.Type) && platformReadMatchesBool(fn.IsDeployed, input.Deployed)
		})
		functions, truncated := truncateFunctions(functions, defaultPlatformContentLimit)
		return functions, len(functions), truncated, err
	})
}

// HandlePlatformFunction returns one hosted ontology function.
func HandlePlatformFunction(ctx context.Context, req *mcp.CallToolRequest, input PlatformEntityInput) (*mcp.CallToolResult, PlatformReadOutput, error) {
	return platformIDRead(ctx, "platform_function", input.ID, input.Fields, func(ctx context.Context, session *platformToolSession, id string) (any, int, bool, error) {
		function, err := session.Client.Function(ctx, session.Host.DefaultProjectID, id, input.Fields)
		if err != nil || function == nil {
			return function, 0, false, err
		}
		truncated := truncateFunction(function, defaultPlatformContentLimit)
		return function, 1, truncated, nil
	})
}

// HandlePlatformFiles lists hosted files in one project folder.
func HandlePlatformFiles(ctx context.Context, req *mcp.CallToolRequest, input PlatformFilesInput) (*mcp.CallToolResult, PlatformReadOutput, error) {
	return platformProjectRead(ctx, "platform_files", input.Fields, func(ctx context.Context, session *platformToolSession) (any, int, bool, error) {
		contents, err := session.Client.FolderContents(ctx, session.Host.DefaultProjectID, input.FolderID, input.Fields)
		if contents == nil {
			return contents, 0, false, err
		}
		contents = filterPlatformFileContents(contents, input)
		return contents, len(contents.Folders) + len(contents.Files), false, err
	})
}

// HandlePlatformFilePreview previews one hosted project file.
func HandlePlatformFilePreview(ctx context.Context, req *mcp.CallToolRequest, input PlatformFilePreviewInput) (*mcp.CallToolResult, PlatformReadOutput, error) {
	if strings.TrimSpace(input.FileID) == "" {
		return nil, PlatformReadOutput{Error: "file_id is required", RequestID: generateRequestID("platform_file_preview")}, nil
	}
	return platformProjectRead(ctx, "platform_file_preview", input.Fields, func(ctx context.Context, session *platformToolSession) (any, int, bool, error) {
		preview, err := session.Client.FilePreview(ctx, session.Host.DefaultProjectID, input.FileID, input.SheetIndex, input.Fields)
		if err != nil || preview == nil {
			return preview, 0, false, err
		}
		truncated := truncateFilePreview(preview, defaultPlatformContentLimit)
		return preview, 1, truncated, nil
	})
}

// HandlePlatformFileInspect inspects one hosted tabular file for dataset promotion.
func HandlePlatformFileInspect(ctx context.Context, req *mcp.CallToolRequest, input PlatformFileInspectInput) (*mcp.CallToolResult, PlatformReadOutput, error) {
	if strings.TrimSpace(input.FileID) == "" {
		return nil, PlatformReadOutput{Error: "file_id is required", RequestID: generateRequestID("platform_file_inspect")}, nil
	}
	return platformProjectRead(ctx, "platform_file_inspect", input.Fields, func(ctx context.Context, session *platformToolSession) (any, int, bool, error) {
		preview, err := session.Client.FilePreview(ctx, session.Host.DefaultProjectID, input.FileID, input.SheetIndex, nil)
		if err != nil {
			return nil, 0, false, err
		}
		inspection := platformapi.InspectFilePreview(input.FileID, preview)
		if !input.IncludeRows {
			inspection.Rows = nil
		}
		return inspection, len(inspection.Columns), false, nil
	})
}

// HandlePlatformFileSearch searches hosted project files.
func HandlePlatformFileSearch(ctx context.Context, req *mcp.CallToolRequest, input PlatformFileSearchInput) (*mcp.CallToolResult, PlatformReadOutput, error) {
	if strings.TrimSpace(input.Query) == "" {
		return nil, PlatformReadOutput{Error: "query is required", RequestID: generateRequestID("platform_file_search")}, nil
	}
	return platformProjectRead(ctx, "platform_file_search", input.Fields, func(ctx context.Context, session *platformToolSession) (any, int, bool, error) {
		files, err := session.Client.SearchProjectFiles(ctx, session.Host.DefaultProjectID, input.Query)
		return files, len(files), false, err
	})
}

// HandlePlatformTabularFiles lists hosted tabular project files.
func HandlePlatformTabularFiles(ctx context.Context, req *mcp.CallToolRequest, input PlatformEmptyInput) (*mcp.CallToolResult, PlatformReadOutput, error) {
	return platformProjectRead(ctx, "platform_tabular_files", input.Fields, func(ctx context.Context, session *platformToolSession) (any, int, bool, error) {
		files, err := session.Client.ProjectTabularFiles(ctx, session.Host.DefaultProjectID)
		return files, len(files), false, err
	})
}

// HandlePlatformStorageUsage returns hosted project storage usage in bytes.
func HandlePlatformStorageUsage(ctx context.Context, req *mcp.CallToolRequest, input PlatformEmptyInput) (*mcp.CallToolResult, PlatformReadOutput, error) {
	return platformProjectRead(ctx, "platform_storage_usage", input.Fields, func(ctx context.Context, session *platformToolSession) (any, int, bool, error) {
		usage, err := session.Client.ProjectStorageUsage(ctx, session.Host.DefaultProjectID)
		return map[string]int{"storage_used": usage}, 1, false, err
	})
}

func platformProjectRead(ctx context.Context, toolName string, fields []string, read func(context.Context, *platformToolSession) (any, int, bool, error)) (*mcp.CallToolResult, PlatformReadOutput, error) {
	requestID := generateRequestID(toolName)
	startTime := time.Now()
	session, err := loadPlatformWorkspace(ctx)
	if err != nil {
		TrackToolCall(ctx, toolName, requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "platform_session"})
		return nil, PlatformReadOutput{PlatformSetupGuidance: platformSetupGuidanceForCurrentConfig(requestID), Error: err.Error(), RequestID: requestID}, nil
	}
	data, count, truncated, err := read(ctx, session)
	if err != nil {
		TrackToolCall(ctx, toolName, requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "platform_query"})
		return nil, PlatformReadOutput{Error: err.Error(), RequestID: requestID}, nil
	}
	TrackToolCall(ctx, toolName, requestID, true, time.Since(startTime).Milliseconds(), map[string]any{"count": count, "truncated": truncated})
	return nil, platformReadOutput(session, toolName, data, count, truncated, requestID, fields), nil
}

func platformIDRead(ctx context.Context, toolName, id string, fields []string, read func(context.Context, *platformToolSession, string) (any, int, bool, error)) (*mcp.CallToolResult, PlatformReadOutput, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, PlatformReadOutput{Error: "id is required", RequestID: generateRequestID(toolName)}, nil
	}
	return platformProjectRead(ctx, toolName, fields, func(ctx context.Context, session *platformToolSession) (any, int, bool, error) {
		return read(ctx, session, id)
	})
}

func platformRowsRead(ctx context.Context, toolName string, input PlatformRowsInput, secOpts *SecurityOptions, read func(context.Context, *platformToolSession, string, int, int) (*platformapi.DatasetQueryResult, error)) (*mcp.CallToolResult, PlatformReadOutput, error) {
	id := strings.TrimSpace(input.ID)
	if id == "" {
		return nil, PlatformReadOutput{Error: "id is required", RequestID: generateRequestID(toolName)}, nil
	}
	if input.Offset < 0 {
		return nil, PlatformReadOutput{Error: "offset must be non-negative", RequestID: generateRequestID(toolName)}, nil
	}
	return platformProjectRead(ctx, toolName, input.Fields, func(ctx context.Context, session *platformToolSession) (any, int, bool, error) {
		limit := platformRowsLimit(input.Limit, secOpts)
		rows, err := read(ctx, session, id, limit, input.Offset)
		if rows == nil {
			return rows, 0, false, err
		}
		return rows, len(rows.Rows), rows.Total > len(rows.Rows), err
	})
}

func platformSourceRefRead(ctx context.Context, toolName, sourceValue, refValue string, fields []string, read func(context.Context, *platformToolSession, *platformapi.Source, platformapi.SourceObjectRefInput) (any, int, bool, error)) (*mcp.CallToolResult, PlatformReadOutput, error) {
	requestID := generateRequestID(toolName)
	startTime := time.Now()
	session, source, err := loadPlatformSource(ctx, sourceValue)
	if err != nil {
		TrackToolCall(ctx, toolName, requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "platform_source"})
		return nil, PlatformReadOutput{Error: err.Error(), RequestID: requestID}, nil
	}
	ref, err := parsePlatformRequiredRef(refValue)
	if err != nil {
		TrackToolCall(ctx, toolName, requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "validation"})
		return nil, PlatformReadOutput{Error: err.Error(), RequestID: requestID}, nil
	}
	data, count, truncated, err := read(ctx, session, source, ref)
	if err != nil {
		TrackToolCall(ctx, toolName, requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "platform_query"})
		return nil, PlatformReadOutput{Error: err.Error(), RequestID: requestID}, nil
	}
	TrackToolCall(ctx, toolName, requestID, true, time.Since(startTime).Milliseconds(), map[string]any{"count": count, "truncated": truncated})
	return nil, platformReadOutput(session, toolName, data, count, truncated, requestID, fields), nil
}

func truncateFunctions(functions []platformapi.Function, limit int) ([]platformapi.Function, bool) {
	truncated := false
	for i := range functions {
		if truncateFunction(&functions[i], limit) {
			truncated = true
		}
	}
	return functions, truncated
}

func truncateFunction(function *platformapi.Function, limit int) bool {
	truncated := false
	if function == nil {
		return false
	}
	for i := range function.Files {
		content, wasTruncated := truncateString(function.Files[i].Content, limit)
		function.Files[i].Content = content
		if wasTruncated {
			truncated = true
		}
	}
	return truncated
}

func truncateSourceContent(content *platformapi.SourceContent, limit int) bool {
	if content == nil || content.Text == nil {
		return false
	}
	value, truncated := truncateString(*content.Text, limit)
	content.Text = &value
	if truncated {
		content.Truncated = true
	}
	return truncated
}

func truncateFilePreview(preview *platformapi.FilePreviewResult, limit int) bool {
	if preview == nil || preview.TextContent == nil {
		return false
	}
	value, truncated := truncateString(*preview.TextContent, limit)
	preview.TextContent = &value
	return truncated
}

func truncateString(value string, limit int) (string, bool) {
	if limit <= 0 || len(value) <= limit {
		return value, false
	}
	return value[:limit], true
}

func lineageNodeCount(graph *platformapi.LineageGraph) int {
	if graph == nil {
		return 0
	}
	return len(graph.Nodes)
}

func filterPlatformReadSlice[T any](values []T, keep func(T) bool) []T {
	filtered := make([]T, 0, len(values))
	for _, value := range values {
		if keep(value) {
			filtered = append(filtered, value)
		}
	}
	return filtered
}

func filterPlatformFileContents(contents *platformapi.FolderContents, input PlatformFilesInput) *platformapi.FolderContents {
	if contents == nil {
		return nil
	}
	filtered := *contents
	filtered.Folders = nil
	filtered.Files = nil
	if platformReadMatchesEqual("folder", input.Kind) {
		filtered.Folders = filterPlatformReadSlice(contents.Folders, func(folder platformapi.ProjectFolder) bool {
			return platformReadMatchesSubstring(folder.Name, input.Name)
		})
	}
	if platformReadMatchesEqual("file", input.Kind) {
		filtered.Files = filterPlatformReadSlice(contents.Files, func(file platformapi.ProjectFile) bool {
			return platformReadMatchesSubstring(file.Name, input.Name) && platformReadMatchesSubstring(file.MIMEType, input.MIMEType)
		})
	}
	return &filtered
}

func platformReadMatchesSubstring(value, filter string) bool {
	needle := strings.TrimSpace(filter)
	if needle == "" {
		return true
	}
	return strings.Contains(strings.ToLower(value), strings.ToLower(needle))
}

func platformReadMatchesEqual(value, filter string) bool {
	expected := strings.TrimSpace(filter)
	if expected == "" {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(value), expected)
}

func platformReadMatchesBool(value bool, filter string) bool {
	expected := strings.TrimSpace(filter)
	if expected == "" {
		return true
	}
	switch strings.ToLower(expected) {
	case "true", "yes", "1":
		return value
	case "false", "no", "0":
		return !value
	default:
		return false
	}
}

const descPlatformWorkspaceMap = `Return a compact selected-project map across hosted sources, secrets, AI providers, datasets, ontologies, transforms, functions, root files, storage, and lineage.

Use this when the agent needs a full resource inventory after first checking whodb_platform_workspace_summary. It is read-only and never returns secret values.`

const descPlatformResourceGraph = `Return a normalized graph of hosted platform resources and relationships in the selected project.

Use this when an agent needs to understand how sources, datasets, files, ontologies, functions, secrets, AI providers, and lineage relate before proposing changes.`

const descPlatformNextActions = `Return deterministic suggested next actions for the selected hosted project.

Use this as a deeper action-ranking view after whodb_platform_workspace_summary or whodb_platform_gap_analysis. Suggestions are advisory only; backend permissions and write confirmations still apply.`

const descPlatformWorkspaceSummary = `Return a compact, goal-aware summary of the selected hosted project.

Use this when the user asks what the platform project contains or what can be built. It combines scope, counts, highlights, gaps, next actions, recommended tools, warnings, and lineage summary without drilling into every item.`

const descPlatformBuildPlan = `Return an end-to-end hosted platform build plan for a user goal.

Use this before coordinating a cross-resource workflow. It plans scope, ingest, persist, model, automate, runtime support, and governance phases using existing WhoDB platform terminology. It is read-only and never creates confirmation tokens.`

const descPlatformGapAnalysis = `Return a goal-aware gap analysis for the selected hosted project.

Use this when the agent needs to explain what is missing before building an app, data product, ontology workflow, transform pipeline, or AI-backed function. It returns ready areas, gaps, suggested tools, and next actions.`

const descPlatformProjectHealth = `Return a compact health summary for the selected hosted project.

Use this for readiness diagnostics after whodb_platform_workspace_summary or whodb_platform_gap_analysis. It combines project counts, readiness checks, warnings, selected scope, and top suggested next actions.`

const descPlatformDataModelSummary = `Summarize selected-project data-model resources.

Use this before ontology, dataset, or lineage work. It returns sources, datasets, ontologies, graph relationships, modeling gaps, and the read tools an agent should call next.`

const descPlatformRuntimeReadiness = `Summarize selected-project runtime readiness.

Use this before function, transform, AI provider, or secret work. It returns runtime resources, deployment/readiness checks, and warnings without exposing secret values.`

const descPlatformChangeImpact = `Analyze direct graph impact for a planned change.

Pass resource, id, and optional action before update, delete, deploy, run, move, or promotion work. It returns directly connected resources and recommended reads; it does not execute any change.`

const descPlatformWritePlan = `Validate and preview a hosted platform write without executing it.

Use this before calling mutating tools when the agent needs a safe dry-run summary. It reuses the same generic write specs as real writes, returns payload keys, suggested reads, and direct impact, but does not create a confirmation token.`

const descPlatformSourceConstraints = `Describe editable field constraints for one hosted source object.

Use a source name or id and an object ref such as table:public.users. This tool is read-only.`

const descPlatformSourceContent = `Read content for one hosted source object when the source supports content reads.

Use a source name or id and an object ref such as file:notes/report.txt. Prefer fields to request only the content metadata or body fields needed now, then call again with more fields only if needed. Text content is capped in MCP output.`

const descPlatformSecrets = `List hosted project secret metadata and usage. Use name to filter by secret name when known.

This tool never returns secret values, redacted placeholders, or credential material. Authorization is enforced by the hosted platform.`

const descPlatformAIProviders = `List hosted AI provider metadata in the selected project. Use name and type filters when known.

This tool never returns provider API keys.`

const descPlatformAIProviderModels = `List model names available from one hosted AI provider.`

const descPlatformOntologies = `List ontology object types in the selected hosted project. Use name and status filters when known.`

const descPlatformOntology = `Inspect one ontology object type by id.`

const descPlatformOntologyFastLookups = `List saved fast lookups for one ontology object type.`

const descPlatformOntologyFastLookupSuggestions = `List suggested fast lookups for one ontology object type.`

const descPlatformOntologyRows = `Preview rows for one ontology object type.

Results are capped by the requested limit and the MCP --max-rows setting when provided.`

const descPlatformOntologyFollowLink = `Follow one ontology link from a row primary key and preview linked rows.

Results are capped by the requested limit and the MCP --max-rows setting when provided.`

const descPlatformDatasets = `List datasets in the selected hosted project. Use name and schema_mode filters when known.`

const descPlatformDataset = `Inspect one hosted dataset by id.`

const descPlatformDatasetRows = `Preview rows for one hosted dataset.

Results are capped by the requested limit and the MCP --max-rows setting when provided.`

const descPlatformLineage = `Return lineage around one root node in the selected hosted project.`

const descPlatformLineageNeighbors = `Return immediate lineage neighbors for one node in the selected hosted project.`

const descPlatformProjectLineage = `Return project-level lineage for the selected hosted project.`

const descPlatformTransforms = `List transforms in the selected hosted project. Use name and type filters when known; type maps to trigger mode for this tool.`

const descPlatformTransform = `Inspect one transform in the selected hosted project. Use fields to request only the top-level fields needed for the current task.`

const descPlatformTransformRuns = `List recent runs for one hosted transform.`

const descPlatformFunctions = `List ontology functions in the selected hosted project.

Use name, type, and deployed filters when known; type maps to language for this tool. Prefer fields such as ["id", "name", "description", "isDeployed"] for discovery. Request heavier fields such as "files" only when source code is needed. Function source file content is capped in MCP output. Secret bindings include secret ids only, never secret values.`

const descPlatformFunction = `Inspect one ontology function by id.

Prefer fields to request only the details needed now, for example ["id", "name", "description"]. Request heavier fields such as "files" only when source code is needed. Function source file content is capped in MCP output. Secret bindings include secret ids only, never secret values.`

const descPlatformFiles = `List folders and files in a hosted project folder.

Omit folder_id to list the project root. Use name, kind, and mime_type filters when known. Prefer fields such as ["files", "folders"] or ["storageUsed"] to request only the folder data needed now.`

const descPlatformFilePreview = `Preview one hosted project file.

Prefer fields such as ["mimeType", "sizeBytes", "isTabular"] before requesting body fields. Request "textContent" or "tabular" only when the file body or rows are needed. Text content is capped in MCP output. Tabular previews are returned as provided by the hosted platform.`

const descPlatformFileInspect = `Inspect one hosted tabular file and infer promote-to-dataset column mappings.

Use this before whodb_platform_action with resource "file" and action "promote_to_dataset". Prefer fields such as ["columns", "columnMapExample", "columnMapFlags"] first. Set include_rows only when sample row values are needed. The suggested mappings are convenience output; the hosted platform still validates promotion writes.`

const descPlatformFileSearch = `Search files in the selected hosted project.`

const descPlatformTabularFiles = `List hosted project files that can be used as tabular data.`

const descPlatformStorageUsage = `Return selected hosted project storage usage in bytes.`
