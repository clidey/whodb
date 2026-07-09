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
	"encoding/json"
	"errors"
	"sort"
	"strconv"
	"strings"

	platformapi "github.com/clidey/whodb/cli/internal/platform"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// PlatformWorkspaceMapInput is the input for the whodb_platform_workspace_map tool.
type PlatformWorkspaceMapInput struct {
	OmitFiles   bool     `json:"omit_files,omitempty" jsonschema:"Omit root folder file and folder summaries. Defaults to false."`
	OmitLineage bool     `json:"omit_lineage,omitempty" jsonschema:"Omit project lineage summary. Defaults to false."`
	Fields      []string `json:"fields,omitempty" jsonschema:"Top-level output fields to include, for example counts, sources, datasets, warnings."`
}

// PlatformResourceGraphInput is the input for the whodb_platform_resource_graph tool.
type PlatformResourceGraphInput struct {
	OmitFiles   bool     `json:"omit_files,omitempty" jsonschema:"Omit root folder file and folder nodes. Defaults to false."`
	OmitLineage bool     `json:"omit_lineage,omitempty" jsonschema:"Omit hosted lineage edges. Defaults to false."`
	Fields      []string `json:"fields,omitempty" jsonschema:"Top-level output fields to include, for example nodes, edges, warnings."`
}

// PlatformNextActionsInput is the input for the whodb_platform_next_actions tool.
type PlatformNextActionsInput struct {
	Goal        string   `json:"goal,omitempty" jsonschema:"Optional user goal used to keep suggested actions relevant."`
	OmitFiles   bool     `json:"omit_files,omitempty" jsonschema:"Omit root folder file and folder summaries. Defaults to false."`
	OmitLineage bool     `json:"omit_lineage,omitempty" jsonschema:"Omit project lineage summary. Defaults to false."`
	Fields      []string `json:"fields,omitempty" jsonschema:"Top-level output fields to include, for example actions, warnings, goal."`
}

// PlatformWorkspaceSummaryInput is the input for the whodb_platform_workspace_summary tool.
type PlatformWorkspaceSummaryInput struct {
	Goal        string   `json:"goal,omitempty" jsonschema:"Optional user goal used to tailor highlights and recommended next tools."`
	OmitFiles   bool     `json:"omit_files,omitempty" jsonschema:"Omit root folder file and folder summaries. Defaults to false."`
	OmitLineage bool     `json:"omit_lineage,omitempty" jsonschema:"Omit project lineage summary. Defaults to false."`
	Fields      []string `json:"fields,omitempty" jsonschema:"Top-level output fields to include, for example scope, counts, highlights, gaps, next_actions."`
}

// PlatformBuildPlanInput is the input for the whodb_platform_build_plan tool.
type PlatformBuildPlanInput struct {
	Goal        string   `json:"goal" jsonschema:"User goal or desired app/data workflow to plan against."`
	OmitFiles   bool     `json:"omit_files,omitempty" jsonschema:"Omit root folder file and folder summaries. Defaults to false."`
	OmitLineage bool     `json:"omit_lineage,omitempty" jsonschema:"Omit project lineage summary. Defaults to false."`
	Fields      []string `json:"fields,omitempty" jsonschema:"Top-level output fields to include, for example phases, prerequisites, warnings."`
}

// PlatformGapAnalysisInput is the input for the whodb_platform_gap_analysis tool.
type PlatformGapAnalysisInput struct {
	Goal        string   `json:"goal,omitempty" jsonschema:"Optional user goal or desired app/data workflow to analyze gaps against."`
	OmitFiles   bool     `json:"omit_files,omitempty" jsonschema:"Omit root folder file and folder summaries. Defaults to false."`
	OmitLineage bool     `json:"omit_lineage,omitempty" jsonschema:"Omit project lineage summary. Defaults to false."`
	Fields      []string `json:"fields,omitempty" jsonschema:"Top-level output fields to include, for example gaps, ready, counts, next_actions."`
}

// PlatformChangeImpactInput is the input for the whodb_platform_change_impact tool.
type PlatformChangeImpactInput struct {
	Resource    string   `json:"resource" jsonschema:"Resource type, for example dataset, ontology, function, transform, file, source, secret, ai_provider"`
	ID          string   `json:"id" jsonschema:"Resource id"`
	Action      string   `json:"action,omitempty" jsonschema:"Optional planned action, for example update, delete, run, deploy, promote_to_dataset"`
	OmitFiles   bool     `json:"omit_files,omitempty" jsonschema:"Omit root folder file and folder nodes. Defaults to false."`
	OmitLineage bool     `json:"omit_lineage,omitempty" jsonschema:"Omit hosted lineage edges. Defaults to false."`
	Fields      []string `json:"fields,omitempty" jsonschema:"Top-level output fields to include, for example target, affected, warnings."`
}

// PlatformWritePlanInput is the input for the whodb_platform_write_plan tool.
type PlatformWritePlanInput struct {
	Resource    string   `json:"resource" jsonschema:"Resource type, for example secret, ai_provider, ontology, dataset, transform, folder, file, function, source_object"`
	ID          string   `json:"id,omitempty" jsonschema:"Resource id for update, delete, or action operations"`
	Action      string   `json:"action,omitempty" jsonschema:"Action name when operation is action, for example deploy, run, rename, move, promote_to_dataset"`
	Operation   string   `json:"operation" jsonschema:"Write operation: create, update, delete, or action"`
	PayloadJSON string   `json:"payload_json,omitempty" jsonschema:"JSON object payload. This is validated and summarized but never executed by this read-only tool."`
	OmitFiles   bool     `json:"omit_files,omitempty" jsonschema:"Omit root folder file and folder nodes for impact lookup. Defaults to false."`
	OmitLineage bool     `json:"omit_lineage,omitempty" jsonschema:"Omit hosted lineage edges for impact lookup. Defaults to false."`
	Fields      []string `json:"fields,omitempty" jsonschema:"Top-level output fields to include, for example preview, affected, warnings."`
}

// PlatformWorkspaceItem is a compact platform resource summary for agents.
type PlatformWorkspaceItem struct {
	ID       string            `json:"id,omitempty"`
	Type     string            `json:"type"`
	Name     string            `json:"name,omitempty"`
	Status   string            `json:"status,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// PlatformLineageSummary is a compact lineage overview.
type PlatformLineageSummary struct {
	NodeCount int `json:"node_count"`
	EdgeCount int `json:"edge_count"`
}

// PlatformWorkspaceMap is a project-level map of hosted platform resources.
type PlatformWorkspaceMap struct {
	Host        string                  `json:"host,omitempty"`
	OrgID       string                  `json:"org_id,omitempty"`
	OrgName     string                  `json:"org_name,omitempty"`
	ProjectID   string                  `json:"project_id,omitempty"`
	ProjectName string                  `json:"project_name,omitempty"`
	Counts      map[string]int          `json:"counts"`
	Sources     []PlatformWorkspaceItem `json:"sources"`
	Secrets     []PlatformWorkspaceItem `json:"secrets"`
	AIProviders []PlatformWorkspaceItem `json:"ai_providers"`
	Datasets    []PlatformWorkspaceItem `json:"datasets"`
	Ontologies  []PlatformWorkspaceItem `json:"ontologies"`
	Transforms  []PlatformWorkspaceItem `json:"transforms"`
	Functions   []PlatformWorkspaceItem `json:"functions"`
	Files       []PlatformWorkspaceItem `json:"files,omitempty"`
	Folders     []PlatformWorkspaceItem `json:"folders,omitempty"`
	StorageUsed int                     `json:"storage_used"`
	Lineage     *PlatformLineageSummary `json:"lineage,omitempty"`
	Warnings    []string                `json:"warnings"`
}

// PlatformResourceGraphNode is one node in the hosted platform resource graph.
type PlatformResourceGraphNode struct {
	ID       string            `json:"id"`
	Type     string            `json:"type"`
	Name     string            `json:"name,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// PlatformResourceGraphEdge is one relationship in the hosted platform resource graph.
type PlatformResourceGraphEdge struct {
	FromID   string `json:"from_id"`
	FromType string `json:"from_type"`
	ToID     string `json:"to_id"`
	ToType   string `json:"to_type"`
	Kind     string `json:"kind"`
}

// PlatformResourceGraph is a normalized relationship graph for hosted platform resources.
type PlatformResourceGraph struct {
	Nodes    []PlatformResourceGraphNode `json:"nodes"`
	Edges    []PlatformResourceGraphEdge `json:"edges"`
	Counts   map[string]int              `json:"counts"`
	Warnings []string                    `json:"warnings"`
}

// PlatformNextAction describes one deterministic suggested next step for an agent.
type PlatformNextAction struct {
	Priority       int      `json:"priority"`
	Area           string   `json:"area"`
	Title          string   `json:"title"`
	Reason         string   `json:"reason"`
	SuggestedTools []string `json:"suggested_tools"`
	ReadOnly       bool     `json:"read_only"`
}

// PlatformNextActions describes suggested next steps based on the selected project.
type PlatformNextActions struct {
	Goal     string               `json:"goal,omitempty"`
	Actions  []PlatformNextAction `json:"actions"`
	Warnings []string             `json:"warnings"`
}

// PlatformWorkspaceSummary is a compact, goal-aware workspace overview for agents.
type PlatformWorkspaceSummary struct {
	Goal             string                  `json:"goal,omitempty"`
	Scope            *PlatformOutputScope    `json:"scope,omitempty"`
	Counts           map[string]int          `json:"counts"`
	Highlights       []string                `json:"highlights"`
	Gaps             []PlatformWorkflowGap   `json:"gaps"`
	NextActions      []PlatformNextAction    `json:"next_actions"`
	RecommendedTools []string                `json:"recommended_tools"`
	Lineage          *PlatformLineageSummary `json:"lineage,omitempty"`
	Warnings         []string                `json:"warnings"`
}

// PlatformBuildPlan is an end-to-end platform workflow plan for a user goal.
type PlatformBuildPlan struct {
	Goal          string                  `json:"goal"`
	Scope         *PlatformOutputScope    `json:"scope,omitempty"`
	Prerequisites []PlatformWorkflowCheck `json:"prerequisites"`
	Phases        []PlatformPlanPhase     `json:"phases"`
	Gaps          []PlatformWorkflowGap   `json:"gaps"`
	Warnings      []string                `json:"warnings"`
}

// PlatformPlanPhase is one phase in a recommended platform workflow.
type PlatformPlanPhase struct {
	Phase       string   `json:"phase"`
	Objective   string   `json:"objective"`
	ReadTools   []string `json:"read_tools,omitempty"`
	WriteTools  []string `json:"write_tools,omitempty"`
	VerifyTools []string `json:"verify_tools,omitempty"`
	Notes       []string `json:"notes,omitempty"`
}

// PlatformGapAnalysis is a goal-aware readiness and missing-capability report.
type PlatformGapAnalysis struct {
	Goal        string                `json:"goal,omitempty"`
	Scope       *PlatformOutputScope  `json:"scope,omitempty"`
	Counts      map[string]int        `json:"counts"`
	Ready       []string              `json:"ready"`
	Gaps        []PlatformWorkflowGap `json:"gaps"`
	NextActions []PlatformNextAction  `json:"next_actions"`
	Warnings    []string              `json:"warnings"`
}

// PlatformWorkflowGap describes one missing or weak platform capability.
type PlatformWorkflowGap struct {
	Area           string   `json:"area"`
	Severity       string   `json:"severity"`
	Missing        string   `json:"missing"`
	Reason         string   `json:"reason"`
	SuggestedTools []string `json:"suggested_tools"`
}

// PlatformWorkflowCheck describes one workspace workflow readiness check.
type PlatformWorkflowCheck struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	Reason   string `json:"reason,omitempty"`
	ToolHint string `json:"tool_hint,omitempty"`
}

// PlatformProjectHealth summarizes project-level health for agents.
type PlatformProjectHealth struct {
	Counts   map[string]int          `json:"counts"`
	Checks   []PlatformWorkflowCheck `json:"checks"`
	Warnings []string                `json:"warnings"`
	Scope    *PlatformOutputScope    `json:"scope,omitempty"`
	Next     []PlatformNextAction    `json:"next,omitempty"`
	Graph    *PlatformLineageSummary `json:"graph,omitempty"`
}

// PlatformDataModelSummary summarizes data-model resources and gaps.
type PlatformDataModelSummary struct {
	Sources        []PlatformWorkspaceItem     `json:"sources"`
	Datasets       []PlatformWorkspaceItem     `json:"datasets"`
	Ontologies     []PlatformWorkspaceItem     `json:"ontologies"`
	Relationships  []PlatformResourceGraphEdge `json:"relationships"`
	Gaps           []string                    `json:"gaps"`
	SuggestedTools []string                    `json:"suggested_tools"`
}

// PlatformRuntimeReadiness summarizes executable/runtime readiness.
type PlatformRuntimeReadiness struct {
	AIProviders []PlatformWorkspaceItem `json:"ai_providers"`
	Secrets     []PlatformWorkspaceItem `json:"secrets"`
	Functions   []PlatformWorkspaceItem `json:"functions"`
	Transforms  []PlatformWorkspaceItem `json:"transforms"`
	Checks      []PlatformWorkflowCheck `json:"checks"`
	Warnings    []string                `json:"warnings"`
}

// PlatformChangeImpact summarizes direct graph impact for a planned change.
type PlatformChangeImpact struct {
	Target         PlatformResourceGraphNode   `json:"target"`
	Action         string                      `json:"action,omitempty"`
	Affected       []PlatformResourceGraphNode `json:"affected"`
	Edges          []PlatformResourceGraphEdge `json:"edges"`
	SuggestedReads []string                    `json:"suggested_reads"`
	Warnings       []string                    `json:"warnings"`
}

// PlatformWritePlan validates and summarizes a hosted write without executing it.
type PlatformWritePlan struct {
	Operation            string                      `json:"operation"`
	Resource             string                      `json:"resource"`
	Action               string                      `json:"action,omitempty"`
	Mutation             string                      `json:"mutation"`
	ConfirmationRequired bool                        `json:"confirmation_required"`
	Preview              *PlatformActionPreview      `json:"preview,omitempty"`
	PayloadKeys          []string                    `json:"payload_keys,omitempty"`
	SuggestedReads       []string                    `json:"suggested_reads"`
	Affected             []PlatformResourceGraphNode `json:"affected,omitempty"`
	Warnings             []string                    `json:"warnings"`
}

type platformWorkspaceSnapshot struct {
	session     *platformToolSession
	sources     []platformapi.Source
	secrets     []platformapi.ProjectSecret
	providers   []platformapi.AIProvider
	datasets    []platformapi.Dataset
	ontologies  []platformapi.Ontology
	transforms  []platformapi.Transform
	functions   []platformapi.Function
	folders     []platformapi.ProjectFolder
	files       []platformapi.ProjectFile
	storageUsed int
	lineage     *platformapi.LineageGraph
	warnings    []string
	omitFiles   bool
	omitLineage bool
}

type platformTransformGraph struct {
	Nodes []platformTransformNode `json:"nodes"`
	Edges []platformTransformEdge `json:"edges"`
}

type platformTransformNode struct {
	ID     string         `json:"id"`
	Type   string         `json:"type"`
	Config map[string]any `json:"config"`
}

type platformTransformEdge struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

// HandlePlatformWorkspaceMap returns a compact selected-project workspace map.
func HandlePlatformWorkspaceMap(ctx context.Context, req *mcp.CallToolRequest, input PlatformWorkspaceMapInput) (*mcp.CallToolResult, PlatformReadOutput, error) {
	return platformProjectRead(ctx, "platform_workspace_map", input.Fields, func(ctx context.Context, session *platformToolSession) (any, int, bool, error) {
		snapshot, err := loadPlatformWorkspaceSnapshot(ctx, session, input.OmitFiles, input.OmitLineage)
		if err != nil {
			return nil, 0, false, err
		}
		workspaceMap := buildPlatformWorkspaceMap(snapshot)
		return workspaceMap, sumPlatformCounts(workspaceMap.Counts), false, nil
	})
}

// HandlePlatformResourceGraph returns a normalized selected-project resource graph.
func HandlePlatformResourceGraph(ctx context.Context, req *mcp.CallToolRequest, input PlatformResourceGraphInput) (*mcp.CallToolResult, PlatformReadOutput, error) {
	return platformProjectRead(ctx, "platform_resource_graph", input.Fields, func(ctx context.Context, session *platformToolSession) (any, int, bool, error) {
		snapshot, err := loadPlatformWorkspaceSnapshot(ctx, session, input.OmitFiles, input.OmitLineage)
		if err != nil {
			return nil, 0, false, err
		}
		graph := buildPlatformResourceGraph(snapshot)
		return graph, len(graph.Nodes), false, nil
	})
}

// HandlePlatformNextActions returns deterministic next steps for the selected project.
func HandlePlatformNextActions(ctx context.Context, req *mcp.CallToolRequest, input PlatformNextActionsInput) (*mcp.CallToolResult, PlatformReadOutput, error) {
	return platformProjectRead(ctx, "platform_next_actions", input.Fields, func(ctx context.Context, session *platformToolSession) (any, int, bool, error) {
		snapshot, err := loadPlatformWorkspaceSnapshot(ctx, session, input.OmitFiles, input.OmitLineage)
		if err != nil {
			return nil, 0, false, err
		}
		next := buildPlatformNextActions(snapshot, input.Goal)
		return next, len(next.Actions), false, nil
	})
}

// HandlePlatformWorkspaceSummary returns a compact workspace-wide summary for agents.
func HandlePlatformWorkspaceSummary(ctx context.Context, req *mcp.CallToolRequest, input PlatformWorkspaceSummaryInput) (*mcp.CallToolResult, PlatformReadOutput, error) {
	return platformProjectRead(ctx, "platform_workspace_summary", input.Fields, func(ctx context.Context, session *platformToolSession) (any, int, bool, error) {
		snapshot, err := loadPlatformWorkspaceSnapshot(ctx, session, input.OmitFiles, input.OmitLineage)
		if err != nil {
			return nil, 0, false, err
		}
		summary := buildPlatformWorkspaceSummary(snapshot, input.Goal)
		return summary, len(summary.Highlights) + len(summary.Gaps), false, nil
	})
}

// HandlePlatformBuildPlan returns an end-to-end platform plan for a user goal.
func HandlePlatformBuildPlan(ctx context.Context, req *mcp.CallToolRequest, input PlatformBuildPlanInput) (*mcp.CallToolResult, PlatformReadOutput, error) {
	if strings.TrimSpace(input.Goal) == "" {
		return nil, PlatformReadOutput{Error: "goal is required", RequestID: generateRequestID("platform_build_plan")}, nil
	}
	return platformProjectRead(ctx, "platform_build_plan", input.Fields, func(ctx context.Context, session *platformToolSession) (any, int, bool, error) {
		snapshot, err := loadPlatformWorkspaceSnapshot(ctx, session, input.OmitFiles, input.OmitLineage)
		if err != nil {
			return nil, 0, false, err
		}
		plan := buildPlatformBuildPlan(snapshot, input.Goal)
		return plan, len(plan.Phases), false, nil
	})
}

// HandlePlatformGapAnalysis returns a goal-aware workspace gap analysis.
func HandlePlatformGapAnalysis(ctx context.Context, req *mcp.CallToolRequest, input PlatformGapAnalysisInput) (*mcp.CallToolResult, PlatformReadOutput, error) {
	return platformProjectRead(ctx, "platform_gap_analysis", input.Fields, func(ctx context.Context, session *platformToolSession) (any, int, bool, error) {
		snapshot, err := loadPlatformWorkspaceSnapshot(ctx, session, input.OmitFiles, input.OmitLineage)
		if err != nil {
			return nil, 0, false, err
		}
		analysis := buildPlatformGapAnalysis(snapshot, input.Goal)
		return analysis, len(analysis.Gaps), false, nil
	})
}

// HandlePlatformProjectHealth returns an agent-focused project health summary.
func HandlePlatformProjectHealth(ctx context.Context, req *mcp.CallToolRequest, input PlatformNextActionsInput) (*mcp.CallToolResult, PlatformReadOutput, error) {
	return platformProjectRead(ctx, "platform_project_health", input.Fields, func(ctx context.Context, session *platformToolSession) (any, int, bool, error) {
		snapshot, err := loadPlatformWorkspaceSnapshot(ctx, session, input.OmitFiles, input.OmitLineage)
		if err != nil {
			return nil, 0, false, err
		}
		health := buildPlatformProjectHealth(snapshot)
		return health, len(health.Checks), false, nil
	})
}

// HandlePlatformDataModelSummary returns an agent-focused data model summary.
func HandlePlatformDataModelSummary(ctx context.Context, req *mcp.CallToolRequest, input PlatformResourceGraphInput) (*mcp.CallToolResult, PlatformReadOutput, error) {
	return platformProjectRead(ctx, "platform_data_model_summary", input.Fields, func(ctx context.Context, session *platformToolSession) (any, int, bool, error) {
		snapshot, err := loadPlatformWorkspaceSnapshot(ctx, session, input.OmitFiles, input.OmitLineage)
		if err != nil {
			return nil, 0, false, err
		}
		summary := buildPlatformDataModelSummary(snapshot)
		return summary, len(summary.Datasets) + len(summary.Ontologies), false, nil
	})
}

// HandlePlatformRuntimeReadiness returns an agent-focused runtime readiness summary.
func HandlePlatformRuntimeReadiness(ctx context.Context, req *mcp.CallToolRequest, input PlatformNextActionsInput) (*mcp.CallToolResult, PlatformReadOutput, error) {
	return platformProjectRead(ctx, "platform_runtime_readiness", input.Fields, func(ctx context.Context, session *platformToolSession) (any, int, bool, error) {
		snapshot, err := loadPlatformWorkspaceSnapshot(ctx, session, input.OmitFiles, input.OmitLineage)
		if err != nil {
			return nil, 0, false, err
		}
		readiness := buildPlatformRuntimeReadiness(snapshot)
		return readiness, len(readiness.Checks), false, nil
	})
}

// HandlePlatformChangeImpact returns direct graph impact for a planned resource change.
func HandlePlatformChangeImpact(ctx context.Context, req *mcp.CallToolRequest, input PlatformChangeImpactInput) (*mcp.CallToolResult, PlatformReadOutput, error) {
	if strings.TrimSpace(input.Resource) == "" || strings.TrimSpace(input.ID) == "" {
		return nil, PlatformReadOutput{Error: "resource and id are required", RequestID: generateRequestID("platform_change_impact")}, nil
	}
	return platformProjectRead(ctx, "platform_change_impact", input.Fields, func(ctx context.Context, session *platformToolSession) (any, int, bool, error) {
		snapshot, err := loadPlatformWorkspaceSnapshot(ctx, session, input.OmitFiles, input.OmitLineage)
		if err != nil {
			return nil, 0, false, err
		}
		impact := buildPlatformChangeImpact(snapshot, input.Resource, input.ID, input.Action)
		return impact, len(impact.Affected), false, nil
	})
}

// HandlePlatformWritePlan validates and summarizes a hosted write without executing it.
func HandlePlatformWritePlan(ctx context.Context, req *mcp.CallToolRequest, input PlatformWritePlanInput) (*mcp.CallToolResult, PlatformReadOutput, error) {
	return platformProjectRead(ctx, "platform_write_plan", input.Fields, func(ctx context.Context, session *platformToolSession) (any, int, bool, error) {
		operation := normalizePlatformWriteToken(input.Operation)
		if operation == "" {
			return nil, 0, false, errors.New("operation is required")
		}
		spec, payload, err := buildPlatformGenericWrite(session, PlatformGenericWriteInput{
			Resource:    input.Resource,
			ID:          input.ID,
			Action:      input.Action,
			PayloadJSON: input.PayloadJSON,
		}, operation)
		if err != nil {
			return nil, 0, false, err
		}
		snapshot, err := loadPlatformWorkspaceSnapshot(ctx, session, input.OmitFiles, input.OmitLineage)
		if err != nil {
			return nil, 0, false, err
		}
		plan := buildPlatformWritePlan(snapshot, session, spec, payload, input.ID)
		return plan, len(plan.Affected), false, nil
	})
}

func loadPlatformWorkspaceSnapshot(ctx context.Context, session *platformToolSession, omitFiles, omitLineage bool) (*platformWorkspaceSnapshot, error) {
	projectID := session.Host.DefaultProjectID
	snapshot := &platformWorkspaceSnapshot{
		session:     session,
		omitFiles:   omitFiles,
		omitLineage: omitLineage,
	}

	var err error
	if snapshot.sources, err = session.Client.ProjectSources(ctx, session.Host.DefaultOrgID, projectID); err != nil {
		return nil, err
	}
	if snapshot.secrets, err = session.Client.ProjectSecrets(ctx, projectID); err != nil {
		return nil, err
	}
	if snapshot.providers, err = session.Client.AIProviders(ctx, projectID); err != nil {
		return nil, err
	}
	if snapshot.datasets, err = session.Client.Datasets(ctx, projectID); err != nil {
		return nil, err
	}
	if snapshot.ontologies, err = session.Client.Ontologies(ctx, projectID); err != nil {
		return nil, err
	}
	if snapshot.transforms, err = session.Client.Transforms(ctx, projectID); err != nil {
		return nil, err
	}
	if snapshot.functions, err = session.Client.Functions(ctx, projectID, []string{"id", "name", "language", "entryPoint", "isDeployed", "providerIds", "ontologyIds", "readOnlyOntologyIds", "providerConfigs", "secretBindings"}); err != nil {
		return nil, err
	}
	if !omitFiles {
		contents, err := session.Client.FolderContents(ctx, projectID, "", []string{"folders", "files", "storageUsed"})
		if err != nil {
			return nil, err
		}
		if contents != nil {
			snapshot.folders = contents.Folders
			snapshot.files = contents.Files
			snapshot.storageUsed = contents.StorageUsed
		}
	}
	usage, err := session.Client.ProjectStorageUsage(ctx, projectID)
	if err != nil {
		return nil, err
	}
	snapshot.storageUsed = usage
	if !omitLineage {
		snapshot.lineage, err = session.Client.ProjectLineage(ctx, projectID)
		if err != nil {
			return nil, err
		}
	}
	snapshot.warnings = platformWorkspaceWarnings(snapshot)
	return snapshot, nil
}

func buildPlatformWorkspaceMap(snapshot *platformWorkspaceSnapshot) PlatformWorkspaceMap {
	scope := platformScope(snapshot.session)
	workspaceMap := PlatformWorkspaceMap{
		Host:        scope.Host,
		OrgID:       scope.OrgID,
		OrgName:     scope.OrgName,
		ProjectID:   scope.ProjectID,
		ProjectName: scope.ProjectName,
		Counts:      platformWorkspaceCounts(snapshot),
		Sources:     sourceItems(snapshot.sources),
		Secrets:     secretItems(snapshot.secrets),
		AIProviders: providerItems(snapshot.providers),
		Datasets:    datasetItems(snapshot.datasets),
		Ontologies:  ontologyItems(snapshot.ontologies),
		Transforms:  transformItems(snapshot.transforms),
		Functions:   functionItems(snapshot.functions),
		Files:       fileItems(snapshot.files),
		Folders:     folderItems(snapshot.folders),
		StorageUsed: snapshot.storageUsed,
		Warnings:    append([]string(nil), snapshot.warnings...),
	}
	if snapshot.lineage != nil {
		workspaceMap.Lineage = &PlatformLineageSummary{NodeCount: len(snapshot.lineage.Nodes), EdgeCount: len(snapshot.lineage.Edges)}
	}
	return workspaceMap
}

func buildPlatformResourceGraph(snapshot *platformWorkspaceSnapshot) PlatformResourceGraph {
	nodesByKey := map[string]PlatformResourceGraphNode{}
	var edges []PlatformResourceGraphEdge
	addNode := func(id, nodeType, name string, metadata map[string]string) {
		if id == "" {
			return
		}
		key := nodeType + ":" + id
		if _, ok := nodesByKey[key]; ok {
			return
		}
		nodesByKey[key] = PlatformResourceGraphNode{ID: id, Type: nodeType, Name: name, Metadata: metadata}
	}
	addEdge := func(fromID, fromType, toID, toType, kind string) {
		if fromID == "" || toID == "" {
			return
		}
		edges = append(edges, PlatformResourceGraphEdge{FromID: fromID, FromType: fromType, ToID: toID, ToType: toType, Kind: kind})
	}

	for _, source := range snapshot.sources {
		addNode(source.ID, "source", source.Name, map[string]string{"database_type": source.DatabaseType})
	}
	for _, secret := range snapshot.secrets {
		addNode(secret.ID, "secret", secret.Name, nil)
		for _, usage := range secret.UsedBy {
			addEdge(secret.ID, "secret", usage.ConsumerID, normalizeGraphType(usage.ConsumerType), "secret_binding")
		}
	}
	for _, provider := range snapshot.providers {
		addNode(provider.ID, "ai_provider", provider.Name, map[string]string{"provider_type": provider.ProviderType})
	}
	for _, dataset := range snapshot.datasets {
		addNode(dataset.ID, "dataset", dataset.Name, map[string]string{"schema_mode": dataset.SchemaMode})
		addEdge(dataset.SourceID, "source", dataset.ID, "dataset", "source_dataset")
	}
	ontologyIDByAPIName := map[string]string{}
	for _, ontology := range snapshot.ontologies {
		if strings.TrimSpace(ontology.APIName) != "" {
			ontologyIDByAPIName[ontology.APIName] = ontology.ID
		}
	}
	for _, ontology := range snapshot.ontologies {
		addNode(ontology.ID, "ontology", ontologyDisplayName(ontology), map[string]string{"api_name": ontology.APIName, "status": ontology.Status})
		if ontology.SourceID != nil {
			addEdge(*ontology.SourceID, "source", ontology.ID, "ontology", "source_ontology")
		}
		for _, link := range ontology.Links {
			targetID := ontologyIDByAPIName[link.TargetOntologyAPIName]
			if targetID == "" {
				targetID = link.TargetOntologyAPIName
			}
			addEdge(ontology.ID, "ontology", targetID, "ontology", "ontology_link:"+link.APIName)
		}
	}
	for _, transform := range snapshot.transforms {
		addNode(transform.ID, "transform", transform.Name, map[string]string{"trigger_mode": transform.TriggerMode})
		addTransformGraphNodes(transform, addNode, addEdge)
	}
	for _, fn := range snapshot.functions {
		addNode(fn.ID, "function", fn.Name, map[string]string{"language": fn.Language, "deployed": boolString(fn.IsDeployed)})
		for _, providerID := range fn.ProviderIDs {
			addEdge(fn.ID, "function", providerID, "ai_provider", "uses_provider")
		}
		for _, config := range fn.ProviderConfigs {
			addEdge(fn.ID, "function", config.ProviderID, "ai_provider", "uses_provider_model")
		}
		for _, ontologyID := range fn.OntologyIDs {
			addEdge(fn.ID, "function", ontologyID, "ontology", "writes_ontology")
		}
		for _, ontologyID := range fn.ReadOnlyOntologyIDs {
			addEdge(fn.ID, "function", ontologyID, "ontology", "reads_ontology")
		}
		for _, binding := range fn.SecretBindings {
			addEdge(fn.ID, "function", binding.SecretID, "secret", "uses_secret")
		}
	}
	for _, folder := range snapshot.folders {
		addNode(folder.ID, "folder", folder.Name, map[string]string{"path": folder.Path})
		if folder.ParentID != nil {
			addEdge(*folder.ParentID, "folder", folder.ID, "folder", "contains")
		}
	}
	for _, file := range snapshot.files {
		addNode(file.ID, "file", file.Name, map[string]string{"mime_type": file.MIMEType, "tabular": boolString(file.IsTabular)})
		if file.FolderID != nil {
			addEdge(*file.FolderID, "folder", file.ID, "file", "contains")
		}
		if file.DatasetID != nil {
			addEdge(file.ID, "file", *file.DatasetID, "dataset", "promoted_to_dataset")
		}
	}
	if snapshot.lineage != nil {
		for _, node := range snapshot.lineage.Nodes {
			addNode(node.ID, normalizeGraphType(node.NodeType), node.Name, nil)
		}
		for _, edge := range snapshot.lineage.Edges {
			addEdge(edge.SourceID, normalizeGraphType(edge.SourceType), edge.TargetID, normalizeGraphType(edge.TargetType), "lineage")
		}
	}

	nodes := make([]PlatformResourceGraphNode, 0, len(nodesByKey))
	for _, node := range nodesByKey {
		nodes = append(nodes, node)
	}
	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].Type == nodes[j].Type {
			return nodes[i].ID < nodes[j].ID
		}
		return nodes[i].Type < nodes[j].Type
	})
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].Kind == edges[j].Kind {
			if edges[i].FromType == edges[j].FromType {
				if edges[i].FromID == edges[j].FromID {
					return edges[i].ToID < edges[j].ToID
				}
				return edges[i].FromID < edges[j].FromID
			}
			return edges[i].FromType < edges[j].FromType
		}
		return edges[i].Kind < edges[j].Kind
	})
	return PlatformResourceGraph{Nodes: nodes, Edges: edges, Counts: platformWorkspaceCounts(snapshot), Warnings: append([]string(nil), snapshot.warnings...)}
}

func buildPlatformNextActions(snapshot *platformWorkspaceSnapshot, goal string) PlatformNextActions {
	goal = strings.TrimSpace(goal)
	actions := make([]PlatformNextAction, 0)
	add := func(priority int, area, title, reason string, tools ...string) {
		actions = append(actions, PlatformNextAction{Priority: priority, Area: area, Title: title, Reason: reason, SuggestedTools: tools, ReadOnly: allReadOnlyPlatformTools(tools)})
	}

	if len(snapshot.sources) == 0 {
		add(10, "sources", "Connect a source", "The project has no hosted sources, so agents cannot browse source schemas or sample source data.", "whodb_platform_source_types", "whodb_platform_source_fields", "whodb_platform_source_create")
	} else {
		add(20, "sources", "Inspect source structure", "The project has hosted sources. Browse objects and columns before creating datasets or ontologies from source data.", "whodb_platform_sources", "whodb_platform_source_objects", "whodb_platform_source_columns")
	}
	if len(snapshot.files) > 0 && len(snapshot.datasets) == 0 {
		add(30, "files", "Inspect tabular files for dataset promotion", "The project has files but no datasets. Inspect tabular files before promoting one to a dataset.", "whodb_platform_files", "whodb_platform_tabular_files", "whodb_platform_file_inspect", "whodb_platform_promote_file_to_dataset")
	}
	if len(snapshot.datasets) == 0 {
		add(40, "datasets", "Create or promote a dataset", "The project has no datasets, so dataset rows, ontology backing data, and downstream transforms have nothing durable to operate on.", "whodb_platform_create_dataset", "whodb_platform_promote_file_to_dataset")
	} else {
		add(50, "datasets", "Review dataset shape", "Datasets exist. Inspect schema and preview rows before modeling ontologies or transforms.", "whodb_platform_datasets", "whodb_platform_dataset", "whodb_platform_dataset_rows")
	}
	if len(snapshot.ontologies) == 0 && len(snapshot.datasets) > 0 {
		add(60, "ontology", "Model datasets as ontologies", "Datasets exist but no ontology object types were returned for this project.", "whodb_platform_datasets", "whodb_platform_create")
	} else if len(snapshot.ontologies) > 0 {
		add(70, "ontology", "Inspect ontology relationships", "Ontology object types exist. Inspect links and fast lookup suggestions before adding records or functions.", "whodb_platform_ontologies", "whodb_platform_ontology", "whodb_platform_ontology_fast_lookup_suggestions")
	}
	if len(snapshot.providers) == 0 {
		add(80, "ai_providers", "Configure an AI provider if AI-backed functions are needed", "No AI provider metadata was returned. Functions that call model providers need a configured provider.", "whodb_platform_ai_providers", "whodb_platform_create")
	}
	if len(snapshot.functions) == 0 && len(snapshot.ontologies) > 0 {
		add(90, "functions", "Add ontology functions when behavior is needed", "Ontologies exist but no functions were returned. Functions can add behavior around ontology data.", "whodb_platform_functions", "whodb_platform_create")
	} else if undeployedFunctionCount(snapshot.functions) > 0 {
		add(100, "functions", "Review undeployed functions", "At least one function is not deployed. Inspect it before deploying or redeploying.", "whodb_platform_functions", "whodb_platform_function", "whodb_platform_action")
	}
	if len(snapshot.transforms) == 0 && len(snapshot.datasets) > 0 {
		add(110, "transforms", "Add transforms for repeatable data movement", "Datasets exist but no transforms were returned for this project.", "whodb_platform_transforms", "whodb_platform_create")
	} else if len(snapshot.transforms) > 0 {
		add(120, "transforms", "Check transform runs", "Transforms exist. Inspect recent runs before changing or running a transform.", "whodb_platform_transforms", "whodb_platform_transform_runs")
	}
	if snapshot.lineage != nil && len(snapshot.lineage.Edges) > 0 {
		add(130, "lineage", "Use lineage before changing resources", "Project lineage has relationships. Check affected upstream and downstream resources before writes.", "whodb_platform_project_lineage", "whodb_platform_lineage_neighbors")
	}
	if len(actions) == 0 {
		add(140, "workspace", "Inspect workspace map", "The project has little returned metadata. Start with a compact map and status before deciding on writes.", "whodb_platform_status", "whodb_platform_workspace_map")
	}
	sort.Slice(actions, func(i, j int) bool { return actions[i].Priority < actions[j].Priority })
	return PlatformNextActions{Goal: goal, Actions: actions, Warnings: append([]string(nil), snapshot.warnings...)}
}

func addTransformGraphNodes(transform platformapi.Transform, addNode func(string, string, string, map[string]string), addEdge func(string, string, string, string, string)) {
	if strings.TrimSpace(transform.GraphJSON) == "" {
		return
	}
	var graph platformTransformGraph
	if err := json.Unmarshal([]byte(transform.GraphJSON), &graph); err != nil {
		return
	}
	for _, node := range graph.Nodes {
		nodeID := transformGraphNodeID(transform.ID, node.ID)
		if nodeID == "" {
			continue
		}
		nodeType := strings.TrimSpace(node.Type)
		addNode(nodeID, "transform_node", nodeType, map[string]string{"transform_id": transform.ID, "node_type": nodeType})
		addEdge(transform.ID, "transform", nodeID, "transform_node", "contains_transform_node")
		addTransformGraphResourceEdges(node, nodeID, addEdge)
	}
	for _, edge := range graph.Edges {
		fromID := transformGraphNodeID(transform.ID, edge.Source)
		toID := transformGraphNodeID(transform.ID, edge.Target)
		addEdge(fromID, "transform_node", toID, "transform_node", "transform_pipeline_edge")
	}
}

func addTransformGraphResourceEdges(node platformTransformNode, nodeID string, addEdge func(string, string, string, string, string)) {
	sourceID := transformConfigString(node.Config, "sourceId")
	fileID := transformConfigString(node.Config, "fileId")
	datasetID := transformConfigString(node.Config, "datasetId")
	targetDatasetID := transformConfigString(node.Config, "targetDatasetId")
	ontologyID := transformConfigString(node.Config, "ontologyId")
	functionID := transformConfigString(node.Config, "functionId")

	addEdge(sourceID, "source", nodeID, "transform_node", "transform_reads_source")
	addEdge(fileID, "file", nodeID, "transform_node", "transform_reads_file")
	addEdge(datasetID, "dataset", nodeID, "transform_node", "transform_reads_dataset")
	addEdge(targetDatasetID, "transform_node", targetDatasetID, "dataset", "transform_writes_dataset")
	if normalizeGraphType(node.Type) == "ontology" {
		addEdge(nodeID, "transform_node", ontologyID, "ontology", "transform_writes_ontology")
	} else {
		addEdge(ontologyID, "ontology", nodeID, "transform_node", "transform_reads_ontology")
	}
	addEdge(nodeID, "transform_node", functionID, "function", "transform_calls_function")
}

func transformGraphNodeID(transformID, nodeID string) string {
	transformID = strings.TrimSpace(transformID)
	nodeID = strings.TrimSpace(nodeID)
	if transformID == "" || nodeID == "" {
		return ""
	}
	return transformID + ":" + nodeID
}

func transformConfigString(config map[string]any, key string) string {
	value, ok := config[key]
	if !ok {
		return ""
	}
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(text)
}

func buildPlatformProjectHealth(snapshot *platformWorkspaceSnapshot) PlatformProjectHealth {
	counts := platformWorkspaceCounts(snapshot)
	checks := []PlatformWorkflowCheck{
		workspaceCheck("sources", len(snapshot.sources) > 0, "At least one hosted source is available.", "No hosted sources were returned for this project.", "whodb_platform_sources"),
		workspaceCheck("data_model", len(snapshot.datasets) > 0 || len(snapshot.ontologies) > 0, "Datasets or ontologies are available for modeling.", "No datasets or ontologies were returned.", "whodb_platform_data_model_summary"),
		workspaceCheck("runtime", len(snapshot.transforms) > 0 || len(snapshot.functions) > 0 || len(snapshot.providers) > 0, "Runtime resources are available.", "No transforms, functions, or AI providers were returned.", "whodb_platform_runtime_readiness"),
		workspaceCheck("storage", snapshot.storageUsed >= 0, "Project storage usage is available.", "Project storage usage was not available.", "whodb_platform_storage_usage"),
	}
	if snapshot.lineage != nil {
		checks = append(checks, workspaceCheck("lineage", len(snapshot.lineage.Edges) > 0, "Project lineage has relationship edges.", "Project lineage returned no relationship edges.", "whodb_platform_project_lineage"))
	} else {
		checks = append(checks, PlatformWorkflowCheck{Name: "lineage", Status: "unknown", Reason: "Lineage was omitted or unavailable.", ToolHint: "whodb_platform_project_lineage"})
	}
	next := buildPlatformNextActions(snapshot, "").Actions
	if len(next) > 5 {
		next = next[:5]
	}
	health := PlatformProjectHealth{
		Counts:   counts,
		Checks:   checks,
		Warnings: append([]string(nil), snapshot.warnings...),
		Scope:    platformScope(snapshot.session),
		Next:     next,
	}
	if snapshot.lineage != nil {
		health.Graph = &PlatformLineageSummary{NodeCount: len(snapshot.lineage.Nodes), EdgeCount: len(snapshot.lineage.Edges)}
	}
	return health
}

func buildPlatformWorkspaceSummary(snapshot *platformWorkspaceSnapshot, goal string) PlatformWorkspaceSummary {
	goal = strings.TrimSpace(goal)
	next := buildPlatformNextActions(snapshot, goal).Actions
	if len(next) > 6 {
		next = next[:6]
	}
	gaps := platformWorkflowGaps(snapshot, goal)
	summary := PlatformWorkspaceSummary{
		Goal:             goal,
		Scope:            platformScope(snapshot.session),
		Counts:           platformWorkspaceCounts(snapshot),
		Highlights:       platformWorkspaceHighlights(snapshot),
		Gaps:             gaps,
		NextActions:      next,
		RecommendedTools: recommendedWorkspaceTools(snapshot, goal),
		Warnings:         append([]string(nil), snapshot.warnings...),
	}
	if snapshot.lineage != nil {
		summary.Lineage = &PlatformLineageSummary{NodeCount: len(snapshot.lineage.Nodes), EdgeCount: len(snapshot.lineage.Edges)}
	}
	return summary
}

func buildPlatformBuildPlan(snapshot *platformWorkspaceSnapshot, goal string) PlatformBuildPlan {
	goal = strings.TrimSpace(goal)
	return PlatformBuildPlan{
		Goal:          goal,
		Scope:         platformScope(snapshot.session),
		Prerequisites: platformBuildPrerequisites(snapshot),
		Phases:        platformBuildPlanPhases(goal),
		Gaps:          platformWorkflowGaps(snapshot, goal),
		Warnings:      append([]string{"plan only; no write executed and no confirmation token was created", "hosted platform permissions and write confirmations still apply"}, snapshot.warnings...),
	}
}

func buildPlatformGapAnalysis(snapshot *platformWorkspaceSnapshot, goal string) PlatformGapAnalysis {
	goal = strings.TrimSpace(goal)
	next := buildPlatformNextActions(snapshot, goal).Actions
	if len(next) > 5 {
		next = next[:5]
	}
	return PlatformGapAnalysis{
		Goal:        goal,
		Scope:       platformScope(snapshot.session),
		Counts:      platformWorkspaceCounts(snapshot),
		Ready:       platformReadyAreas(snapshot),
		Gaps:        platformWorkflowGaps(snapshot, goal),
		NextActions: next,
		Warnings:    append([]string(nil), snapshot.warnings...),
	}
}

func platformWorkspaceHighlights(snapshot *platformWorkspaceSnapshot) []string {
	highlights := make([]string, 0)
	if len(snapshot.sources) > 0 {
		highlights = append(highlights, intString(len(snapshot.sources))+" source(s) connected")
	}
	if len(snapshot.datasets) > 0 {
		highlights = append(highlights, intString(len(snapshot.datasets))+" dataset(s) available")
	}
	if len(snapshot.ontologies) > 0 {
		highlights = append(highlights, intString(len(snapshot.ontologies))+" ontology object type(s) modeled")
	}
	if len(snapshot.transforms) > 0 {
		highlights = append(highlights, intString(len(snapshot.transforms))+" transform(s) available")
	}
	if len(snapshot.functions) > 0 {
		highlights = append(highlights, intString(len(snapshot.functions))+" function(s) available")
	}
	if len(snapshot.providers) > 0 {
		highlights = append(highlights, intString(len(snapshot.providers))+" AI provider(s) configured")
	}
	if len(snapshot.files) > 0 {
		highlights = append(highlights, intString(len(snapshot.files))+" root file(s) available")
	}
	if snapshot.lineage != nil && len(snapshot.lineage.Edges) > 0 {
		highlights = append(highlights, intString(len(snapshot.lineage.Edges))+" lineage edge(s) found")
	}
	if len(highlights) == 0 {
		highlights = append(highlights, "no major platform resources returned for the selected project")
	}
	return highlights
}

func platformBuildPrerequisites(snapshot *platformWorkspaceSnapshot) []PlatformWorkflowCheck {
	return []PlatformWorkflowCheck{
		workspaceCheck("workspace", snapshot.session.Host.DefaultOrgID != "" && snapshot.session.Host.DefaultProjectID != "", "Organization and project are selected.", "No selected organization/project was found.", "whodb_platform_setup_status"),
		workspaceCheck("ingestion", len(snapshot.sources) > 0 || len(snapshot.files) > 0, "A source or project file is available for data ingestion.", "No source or project file was returned.", "whodb_platform_workspace_summary"),
		workspaceCheck("data_model", len(snapshot.datasets) > 0 || len(snapshot.ontologies) > 0, "Dataset or ontology resources are available.", "No dataset or ontology resources were returned.", "whodb_platform_gap_analysis"),
		workspaceCheck("runtime", len(snapshot.transforms) > 0 || len(snapshot.functions) > 0 || len(snapshot.providers) > 0, "Runtime resource metadata is available.", "Runtime resources are not yet represented.", "whodb_platform_runtime_readiness"),
	}
}

func platformBuildPlanPhases(goal string) []PlatformPlanPhase {
	goal = strings.ToLower(goal)
	phases := []PlatformPlanPhase{
		{
			Phase:       "scope",
			Objective:   "Confirm the selected org/project, inventory resources, and identify gaps before touching single resources.",
			ReadTools:   []string{"whodb_platform_setup_status", "whodb_platform_workspace_summary", "whodb_platform_gap_analysis"},
			VerifyTools: []string{"whodb_platform_project_health"},
		},
		{
			Phase:       "ingest",
			Objective:   "Choose an input path: connected source objects or uploaded/tabular project files.",
			ReadTools:   []string{"whodb_platform_sources", "whodb_platform_source_objects", "whodb_platform_files", "whodb_platform_file_inspect"},
			WriteTools:  []string{"whodb_platform_source_create", "whodb_platform_promote_file_to_dataset"},
			VerifyTools: []string{"whodb_platform_source_test", "whodb_platform_datasets"},
		},
		{
			Phase:       "persist",
			Objective:   "Create or verify durable datasets that represent the data the workflow will operate on.",
			ReadTools:   []string{"whodb_platform_data_model_summary", "whodb_platform_dataset", "whodb_platform_dataset_rows"},
			WriteTools:  []string{"whodb_platform_create_dataset", "whodb_platform_create", "whodb_platform_update"},
			VerifyTools: []string{"whodb_platform_dataset", "whodb_platform_dataset_rows"},
		},
		{
			Phase:       "model",
			Objective:   "Model business objects as ontologies and add lookup/link structure only after the data shape is known.",
			ReadTools:   []string{"whodb_platform_ontologies", "whodb_platform_ontology", "whodb_platform_ontology_fast_lookup_suggestions"},
			WriteTools:  []string{"whodb_platform_create", "whodb_platform_update", "whodb_platform_create_ontology_fast_lookup"},
			VerifyTools: []string{"whodb_platform_ontology", "whodb_platform_ontology_rows"},
		},
		{
			Phase:       "automate",
			Objective:   "Add transforms and functions for repeatable computation or behavior around the modeled data.",
			ReadTools:   []string{"whodb_platform_runtime_readiness", "whodb_platform_transforms", "whodb_platform_functions"},
			WriteTools:  []string{"whodb_platform_create", "whodb_platform_update", "whodb_platform_action"},
			VerifyTools: []string{"whodb_platform_transform_runs", "whodb_platform_function"},
		},
		{
			Phase:       "govern",
			Objective:   "Before writes, inspect graph impact and preview the exact mutation; after writes, verify graph and health.",
			ReadTools:   []string{"whodb_platform_resource_graph", "whodb_platform_change_impact", "whodb_platform_write_plan"},
			WriteTools:  []string{"whodb_platform_create", "whodb_platform_update", "whodb_platform_delete", "whodb_platform_action", "whodb_platform_confirm"},
			VerifyTools: []string{"whodb_platform_project_health", "whodb_platform_resource_graph"},
			Notes:       []string{"Never call whodb_platform_confirm without explicit user approval of the exact confirmation preview."},
		},
	}
	if strings.Contains(goal, "app") || strings.Contains(goal, "agent") || strings.Contains(goal, "function") {
		phases = append(phases, PlatformPlanPhase{
			Phase:       "runtime_support",
			Objective:   "Configure secret metadata and AI provider metadata needed by functions or app behavior.",
			ReadTools:   []string{"whodb_platform_runtime_readiness", "whodb_platform_secrets", "whodb_platform_ai_providers", "whodb_platform_ai_provider_models"},
			WriteTools:  []string{"whodb_platform_create", "whodb_platform_update"},
			VerifyTools: []string{"whodb_platform_runtime_readiness"},
			Notes:       []string{"Secret values are never returned by read tools."},
		})
	}
	return phases
}

func platformWorkflowGaps(snapshot *platformWorkspaceSnapshot, goal string) []PlatformWorkflowGap {
	goal = strings.ToLower(strings.TrimSpace(goal))
	gaps := make([]PlatformWorkflowGap, 0)
	add := func(area, severity, missing, reason string, tools ...string) {
		gaps = append(gaps, PlatformWorkflowGap{Area: area, Severity: severity, Missing: missing, Reason: reason, SuggestedTools: tools})
	}
	if len(snapshot.sources) == 0 && len(snapshot.files) == 0 {
		add("ingestion", "high", "source or file", "No connected source or root project file was returned, so there is no obvious input data path.", "whodb_platform_source_types", "whodb_platform_source_create", "whodb_platform_files")
	}
	if len(snapshot.datasets) == 0 {
		severity := "medium"
		if len(snapshot.sources) > 0 || len(snapshot.files) > 0 {
			severity = "high"
		}
		add("datasets", severity, "dataset", "No durable dataset was returned for the selected project.", "whodb_platform_create_dataset", "whodb_platform_promote_file_to_dataset", "whodb_platform_datasets")
	}
	if len(snapshot.ontologies) == 0 {
		add("ontology", "medium", "ontology", "No ontology object type was returned, so business objects are not modeled yet.", "whodb_platform_data_model_summary", "whodb_platform_create")
	}
	if len(snapshot.transforms) == 0 && (strings.Contains(goal, "sync") || strings.Contains(goal, "pipeline") || strings.Contains(goal, "transform") || strings.Contains(goal, "automation")) {
		add("transforms", "medium", "transform", "The goal appears to need repeatable data movement, but no transforms were returned.", "whodb_platform_transforms", "whodb_platform_create")
	}
	if len(snapshot.functions) == 0 && (strings.Contains(goal, "app") || strings.Contains(goal, "agent") || strings.Contains(goal, "function") || strings.Contains(goal, "workflow")) {
		add("functions", "medium", "function", "The goal appears to need runtime behavior, but no functions were returned.", "whodb_platform_functions", "whodb_platform_create")
	}
	if len(snapshot.providers) == 0 && (strings.Contains(goal, "ai") || strings.Contains(goal, "agent") || strings.Contains(goal, "llm")) {
		add("ai_providers", "medium", "AI provider", "The goal appears to need model calls, but no AI providers were returned.", "whodb_platform_ai_providers", "whodb_platform_create")
	}
	if snapshot.lineage == nil || len(snapshot.lineage.Edges) == 0 {
		add("lineage", "low", "lineage relationships", "No lineage edges were returned, so change impact may be incomplete.", "whodb_platform_resource_graph", "whodb_platform_project_lineage")
	}
	return gaps
}

func platformReadyAreas(snapshot *platformWorkspaceSnapshot) []string {
	ready := make([]string, 0)
	if len(snapshot.sources) > 0 {
		ready = append(ready, "sources")
	}
	if len(snapshot.files) > 0 {
		ready = append(ready, "files")
	}
	if len(snapshot.datasets) > 0 {
		ready = append(ready, "datasets")
	}
	if len(snapshot.ontologies) > 0 {
		ready = append(ready, "ontologies")
	}
	if len(snapshot.transforms) > 0 {
		ready = append(ready, "transforms")
	}
	if len(snapshot.functions) > 0 {
		ready = append(ready, "functions")
	}
	if len(snapshot.providers) > 0 {
		ready = append(ready, "ai_providers")
	}
	if len(snapshot.secrets) > 0 {
		ready = append(ready, "secrets")
	}
	if snapshot.lineage != nil && len(snapshot.lineage.Edges) > 0 {
		ready = append(ready, "lineage")
	}
	return ready
}

func recommendedWorkspaceTools(snapshot *platformWorkspaceSnapshot, goal string) []string {
	tools := []string{"whodb_platform_project_health", "whodb_platform_resource_graph", "whodb_platform_gap_analysis"}
	goal = strings.ToLower(strings.TrimSpace(goal))
	if len(snapshot.datasets) > 0 || len(snapshot.ontologies) > 0 || strings.Contains(goal, "data") || strings.Contains(goal, "model") {
		tools = append(tools, "whodb_platform_data_model_summary")
	}
	if len(snapshot.transforms) > 0 || len(snapshot.functions) > 0 || len(snapshot.providers) > 0 || strings.Contains(goal, "app") || strings.Contains(goal, "runtime") || strings.Contains(goal, "agent") {
		tools = append(tools, "whodb_platform_runtime_readiness")
	}
	if len(snapshot.sources) > 0 {
		tools = append(tools, "whodb_platform_sources", "whodb_platform_source_objects")
	}
	if len(snapshot.files) > 0 {
		tools = append(tools, "whodb_platform_files", "whodb_platform_file_inspect")
	}
	return uniqueStrings(tools)
}

func buildPlatformDataModelSummary(snapshot *platformWorkspaceSnapshot) PlatformDataModelSummary {
	graph := buildPlatformResourceGraph(snapshot)
	relationships := make([]PlatformResourceGraphEdge, 0)
	for _, edge := range graph.Edges {
		if dataModelRelationship(edge) {
			relationships = append(relationships, edge)
		}
	}
	gaps := make([]string, 0)
	if len(snapshot.datasets) == 0 {
		gaps = append(gaps, "no datasets returned")
	}
	if len(snapshot.ontologies) == 0 {
		gaps = append(gaps, "no ontologies returned")
	}
	if len(relationships) == 0 {
		gaps = append(gaps, "no data-model relationships found")
	}
	return PlatformDataModelSummary{
		Sources:        sourceItems(snapshot.sources),
		Datasets:       datasetItems(snapshot.datasets),
		Ontologies:     ontologyItems(snapshot.ontologies),
		Relationships:  relationships,
		Gaps:           gaps,
		SuggestedTools: []string{"whodb_platform_datasets", "whodb_platform_ontologies", "whodb_platform_resource_graph", "whodb_platform_project_lineage"},
	}
}

func dataModelRelationship(edge PlatformResourceGraphEdge) bool {
	if strings.HasPrefix(edge.Kind, "ontology_link") {
		return true
	}
	switch edge.Kind {
	case "source_dataset", "source_ontology", "promoted_to_dataset", "lineage", "transform_reads_dataset", "transform_writes_dataset", "transform_reads_ontology", "transform_writes_ontology":
		return true
	default:
		return false
	}
}

func buildPlatformRuntimeReadiness(snapshot *platformWorkspaceSnapshot) PlatformRuntimeReadiness {
	checks := []PlatformWorkflowCheck{
		workspaceCheck("ai_providers", len(snapshot.providers) > 0, "AI provider metadata is available.", "No AI providers were returned.", "whodb_platform_ai_providers"),
		workspaceCheck("secrets", len(snapshot.secrets) > 0, "Secret metadata is available.", "No secrets were returned. This is fine unless runtime resources need secret bindings.", "whodb_platform_secrets"),
		workspaceCheck("functions", len(snapshot.functions) > 0, "Functions are available.", "No functions were returned.", "whodb_platform_functions"),
		workspaceCheck("transforms", len(snapshot.transforms) > 0, "Transforms are available.", "No transforms were returned.", "whodb_platform_transforms"),
	}
	undeployed := undeployedFunctionCount(snapshot.functions)
	warnings := append([]string(nil), snapshot.warnings...)
	if undeployed > 0 {
		warnings = append(warnings, intString(undeployed)+" function(s) are not deployed")
	}
	return PlatformRuntimeReadiness{
		AIProviders: providerItems(snapshot.providers),
		Secrets:     secretItems(snapshot.secrets),
		Functions:   functionItems(snapshot.functions),
		Transforms:  transformItems(snapshot.transforms),
		Checks:      checks,
		Warnings:    warnings,
	}
}

func buildPlatformChangeImpact(snapshot *platformWorkspaceSnapshot, resource, id, action string) PlatformChangeImpact {
	resourceType := normalizeGraphType(resource)
	id = strings.TrimSpace(id)
	action = strings.TrimSpace(action)
	graph := buildPlatformResourceGraph(snapshot)
	target := PlatformResourceGraphNode{ID: id, Type: resourceType}
	for _, node := range graph.Nodes {
		if node.ID == id && node.Type == resourceType {
			target = node
			break
		}
	}
	edges := make([]PlatformResourceGraphEdge, 0)
	affectedByKey := map[string]PlatformResourceGraphNode{}
	for _, edge := range graph.Edges {
		fromMatches := edge.FromID == id && edge.FromType == resourceType
		toMatches := edge.ToID == id && edge.ToType == resourceType
		if !fromMatches && !toMatches {
			continue
		}
		edges = append(edges, edge)
		if fromMatches {
			addAffectedGraphNode(affectedByKey, graph.Nodes, edge.ToID, edge.ToType)
		}
		if toMatches {
			addAffectedGraphNode(affectedByKey, graph.Nodes, edge.FromID, edge.FromType)
		}
	}
	affected := make([]PlatformResourceGraphNode, 0, len(affectedByKey))
	for _, node := range affectedByKey {
		affected = append(affected, node)
	}
	sortGraphNodes(affected)
	warnings := make([]string, 0)
	if target.Name == "" && len(edges) == 0 {
		warnings = append(warnings, "resource was not found in the selected project graph")
	}
	if isHighImpactPlatformAction(action) {
		warnings = append(warnings, "planned action is destructive or high impact; read affected resources before executing")
	}
	return PlatformChangeImpact{
		Target:         target,
		Action:         action,
		Affected:       affected,
		Edges:          edges,
		SuggestedReads: suggestedReadsForResource(resourceType, action),
		Warnings:       warnings,
	}
}

func buildPlatformWritePlan(snapshot *platformWorkspaceSnapshot, session *platformToolSession, spec platformapi.GenericWriteSpec, payload map[string]any, targetID string) PlatformWritePlan {
	action := &PendingPlatformAction{
		Operation:   spec.Mutation,
		Resource:    spec.Resource,
		Action:      spec.Action,
		Host:        session.Host.URL,
		OrgID:       session.Host.DefaultOrgID,
		ProjectID:   session.Host.DefaultProjectID,
		ProjectName: session.Host.DefaultProjectName,
		Summary:     platformGenericWriteSummary(spec, payload),
		Changes:     genericWriteChanges(payload),
		Mutation:    spec.Mutation,
		Variables:   payload,
	}
	preview := action.Preview()
	affected := []PlatformResourceGraphNode(nil)
	if strings.TrimSpace(targetID) != "" {
		affected = buildPlatformChangeImpact(snapshot, spec.Resource, targetID, spec.Action).Affected
	}
	warnings := []string{"plan only; no write executed and no confirmation token was created", "hosted platform permissions still apply when the write runs"}
	if isHighImpactPlatformAction(spec.Action) {
		warnings = append(warnings, "planned action is destructive or high impact")
	}
	return PlatformWritePlan{
		Operation:            spec.Action,
		Resource:             spec.Resource,
		Action:               spec.Action,
		Mutation:             spec.Mutation,
		ConfirmationRequired: true,
		Preview:              preview,
		PayloadKeys:          genericWriteChanges(payload),
		SuggestedReads:       suggestedReadsForResource(spec.Resource, spec.Action),
		Affected:             affected,
		Warnings:             warnings,
	}
}

func workspaceCheck(name string, ok bool, okReason, missingReason, toolHint string) PlatformWorkflowCheck {
	status := "ok"
	reason := okReason
	if !ok {
		status = "warning"
		reason = missingReason
	}
	return PlatformWorkflowCheck{Name: name, Status: status, Reason: reason, ToolHint: toolHint}
}

func addAffectedGraphNode(target map[string]PlatformResourceGraphNode, nodes []PlatformResourceGraphNode, id, nodeType string) {
	if id == "" {
		return
	}
	key := nodeType + ":" + id
	if _, ok := target[key]; ok {
		return
	}
	for _, node := range nodes {
		if node.ID == id && node.Type == nodeType {
			target[key] = node
			return
		}
	}
	target[key] = PlatformResourceGraphNode{ID: id, Type: nodeType}
}

func sortGraphNodes(nodes []PlatformResourceGraphNode) {
	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].Type == nodes[j].Type {
			return nodes[i].ID < nodes[j].ID
		}
		return nodes[i].Type < nodes[j].Type
	})
}

func isHighImpactPlatformAction(action string) bool {
	switch normalizeGraphType(action) {
	case "delete", "move", "deploy", "redeploy", "run", "upload", "promote_to_dataset":
		return true
	default:
		return false
	}
}

func suggestedReadsForResource(resource, action string) []string {
	tools := []string{"whodb_platform_resource_graph"}
	switch normalizeGraphType(resource) {
	case "dataset":
		tools = append(tools, "whodb_platform_dataset", "whodb_platform_dataset_rows", "whodb_platform_project_lineage")
	case "ontology":
		tools = append(tools, "whodb_platform_ontology", "whodb_platform_ontology_fast_lookups", "whodb_platform_project_lineage")
	case "function":
		tools = append(tools, "whodb_platform_function", "whodb_platform_runtime_readiness")
	case "transform":
		tools = append(tools, "whodb_platform_transform", "whodb_platform_transform_runs", "whodb_platform_project_lineage")
	case "file":
		tools = append(tools, "whodb_platform_files", "whodb_platform_file_inspect", "whodb_platform_file_preview")
	case "folder":
		tools = append(tools, "whodb_platform_files")
	case "source":
		tools = append(tools, "whodb_platform_sources", "whodb_platform_source_config", "whodb_platform_source_test")
	case "secret":
		tools = append(tools, "whodb_platform_secrets", "whodb_platform_runtime_readiness")
	case "ai_provider":
		tools = append(tools, "whodb_platform_ai_providers", "whodb_platform_ai_provider_models", "whodb_platform_runtime_readiness")
	case "source_object":
		tools = append(tools, "whodb_platform_sources", "whodb_platform_source_objects")
	}
	if isHighImpactPlatformAction(action) {
		tools = append(tools, "whodb_platform_change_impact")
	}
	return uniqueStrings(tools)
}

func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		if seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}

func platformWorkspaceCounts(snapshot *platformWorkspaceSnapshot) map[string]int {
	counts := map[string]int{
		"sources":      len(snapshot.sources),
		"secrets":      len(snapshot.secrets),
		"ai_providers": len(snapshot.providers),
		"datasets":     len(snapshot.datasets),
		"ontologies":   len(snapshot.ontologies),
		"transforms":   len(snapshot.transforms),
		"functions":    len(snapshot.functions),
		"folders":      len(snapshot.folders),
		"files":        len(snapshot.files),
	}
	if snapshot.lineage != nil {
		counts["lineage_nodes"] = len(snapshot.lineage.Nodes)
		counts["lineage_edges"] = len(snapshot.lineage.Edges)
	}
	return counts
}

func sumPlatformCounts(counts map[string]int) int {
	total := 0
	for key, count := range counts {
		if strings.HasPrefix(key, "lineage_") {
			continue
		}
		total += count
	}
	return total
}

func platformWorkspaceWarnings(snapshot *platformWorkspaceSnapshot) []string {
	warnings := make([]string, 0)
	if snapshot.omitFiles {
		warnings = append(warnings, "file summaries omitted by request")
	}
	if snapshot.omitLineage {
		warnings = append(warnings, "lineage summary omitted by request")
	}
	if len(snapshot.sources) == 0 {
		warnings = append(warnings, "no hosted sources returned for the selected project")
	}
	if len(snapshot.datasets) == 0 && len(snapshot.files) == 0 {
		warnings = append(warnings, "no datasets or root files returned for the selected project")
	}
	return warnings
}

func sourceItems(sources []platformapi.Source) []PlatformWorkspaceItem {
	items := make([]PlatformWorkspaceItem, 0, len(sources))
	for _, source := range sources {
		items = append(items, PlatformWorkspaceItem{ID: source.ID, Type: "source", Name: source.Name, Metadata: map[string]string{"database_type": source.DatabaseType}})
	}
	return items
}

func secretItems(secrets []platformapi.ProjectSecret) []PlatformWorkspaceItem {
	items := make([]PlatformWorkspaceItem, 0, len(secrets))
	for _, secret := range secrets {
		items = append(items, PlatformWorkspaceItem{ID: secret.ID, Type: "secret", Name: secret.Name, Metadata: map[string]string{"used_by_count": intString(len(secret.UsedBy))}})
	}
	return items
}

func providerItems(providers []platformapi.AIProvider) []PlatformWorkspaceItem {
	items := make([]PlatformWorkspaceItem, 0, len(providers))
	for _, provider := range providers {
		items = append(items, PlatformWorkspaceItem{ID: provider.ID, Type: "ai_provider", Name: provider.Name, Metadata: map[string]string{"provider_type": provider.ProviderType}})
	}
	return items
}

func datasetItems(datasets []platformapi.Dataset) []PlatformWorkspaceItem {
	items := make([]PlatformWorkspaceItem, 0, len(datasets))
	for _, dataset := range datasets {
		items = append(items, PlatformWorkspaceItem{ID: dataset.ID, Type: "dataset", Name: dataset.Name, Metadata: map[string]string{"schema_mode": dataset.SchemaMode, "rows": intString(dataset.RowCount)}})
	}
	return items
}

func ontologyItems(ontologies []platformapi.Ontology) []PlatformWorkspaceItem {
	items := make([]PlatformWorkspaceItem, 0, len(ontologies))
	for _, ontology := range ontologies {
		items = append(items, PlatformWorkspaceItem{ID: ontology.ID, Type: "ontology", Name: ontologyDisplayName(ontology), Status: ontology.Status, Metadata: map[string]string{"api_name": ontology.APIName, "properties": intString(len(ontology.Properties)), "links": intString(len(ontology.Links))}})
	}
	return items
}

func transformItems(transforms []platformapi.Transform) []PlatformWorkspaceItem {
	items := make([]PlatformWorkspaceItem, 0, len(transforms))
	for _, transform := range transforms {
		items = append(items, PlatformWorkspaceItem{ID: transform.ID, Type: "transform", Name: transform.Name, Metadata: map[string]string{"trigger_mode": transform.TriggerMode}})
	}
	return items
}

func functionItems(functions []platformapi.Function) []PlatformWorkspaceItem {
	items := make([]PlatformWorkspaceItem, 0, len(functions))
	for _, fn := range functions {
		status := "draft"
		if fn.IsDeployed {
			status = "deployed"
		}
		items = append(items, PlatformWorkspaceItem{ID: fn.ID, Type: "function", Name: fn.Name, Status: status, Metadata: map[string]string{"language": fn.Language, "entry_point": fn.EntryPoint}})
	}
	return items
}

func fileItems(files []platformapi.ProjectFile) []PlatformWorkspaceItem {
	items := make([]PlatformWorkspaceItem, 0, len(files))
	for _, file := range files {
		items = append(items, PlatformWorkspaceItem{ID: file.ID, Type: "file", Name: file.Name, Metadata: map[string]string{"mime_type": file.MIMEType, "tabular": boolString(file.IsTabular), "size_bytes": intString(file.SizeBytes)}})
	}
	return items
}

func folderItems(folders []platformapi.ProjectFolder) []PlatformWorkspaceItem {
	items := make([]PlatformWorkspaceItem, 0, len(folders))
	for _, folder := range folders {
		items = append(items, PlatformWorkspaceItem{ID: folder.ID, Type: "folder", Name: folder.Name, Metadata: map[string]string{"path": folder.Path}})
	}
	return items
}

func normalizeGraphType(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.ReplaceAll(value, "-", "_")
	value = strings.ReplaceAll(value, " ", "_")
	return value
}

func ontologyDisplayName(ontology platformapi.Ontology) string {
	if strings.TrimSpace(ontology.DisplayName) != "" {
		return ontology.DisplayName
	}
	return ontology.APIName
}

func boolString(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func intString(value int) string {
	return strconv.Itoa(value)
}

func undeployedFunctionCount(functions []platformapi.Function) int {
	count := 0
	for _, fn := range functions {
		if !fn.IsDeployed {
			count++
		}
	}
	return count
}

func allReadOnlyPlatformTools(tools []string) bool {
	for _, tool := range platformToolDefinitions() {
		for _, name := range tools {
			if tool.Name == name && tool.Annotations != nil && !tool.Annotations.ReadOnlyHint {
				return false
			}
		}
	}
	return true
}
