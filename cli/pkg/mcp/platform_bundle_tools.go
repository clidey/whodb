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
	"strings"
	"time"

	platformapi "github.com/clidey/whodb/cli/internal/platform"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// PlatformDoctorInput is the input for whodb_platform_doctor.
type PlatformDoctorInput struct{}

// PlatformDoctorOutput reports hosted platform readiness for MCP tools.
type PlatformDoctorOutput struct {
	Host                    string   `json:"host,omitempty"`
	Email                   string   `json:"email,omitempty"`
	WorkspaceSelected       bool     `json:"workspace_selected"`
	OrgID                   string   `json:"org_id,omitempty"`
	OrgName                 string   `json:"org_name,omitempty"`
	ProjectID               string   `json:"project_id,omitempty"`
	ProjectName             string   `json:"project_name,omitempty"`
	PlatformVersion         string   `json:"platform_version,omitempty"`
	ManifestProtocolVersion string   `json:"manifest_protocol_version,omitempty"`
	Checks                  []string `json:"checks"`
	Warnings                []string `json:"warnings,omitempty"`
	Error                   string   `json:"error,omitempty"`
	RequestID               string   `json:"request_id,omitempty"`
}

// PlatformBundleExportInput is the input for whodb_platform_bundle_export.
type PlatformBundleExportInput struct{}

// PlatformBundleExportOutput returns a selected-project metadata bundle.
type PlatformBundleExportOutput struct {
	Bundle    *platformapi.ProjectBundle `json:"bundle,omitempty"`
	Counts    map[string]int             `json:"counts,omitempty"`
	Error     string                     `json:"error,omitempty"`
	RequestID string                     `json:"request_id,omitempty"`
}

// PlatformBundlePlanInput is the input for bundle diff and import-plan tools.
type PlatformBundlePlanInput struct {
	BundleJSON string `json:"bundle_json" jsonschema:"Project bundle JSON from whodb_platform_bundle_export or resources export"`
}

// PlatformBundlePlanOutput returns a bundle import plan for the selected project.
type PlatformBundlePlanOutput struct {
	Plan      *platformapi.BundlePlan `json:"plan,omitempty"`
	Counts    map[string]int          `json:"counts,omitempty"`
	Error     string                  `json:"error,omitempty"`
	RequestID string                  `json:"request_id,omitempty"`
}

// PlatformCloneInput is the input for whodb_platform_clone.
type PlatformCloneInput struct {
	Resource string `json:"resource" jsonschema:"Resource to clone: dataset, ontology, transform, or function"`
	Source   string `json:"source" jsonschema:"Source resource id, name, or api name"`
	NewName  string `json:"new_name" jsonschema:"New resource name or ontology api name/display name"`
}

func registerPlatformBundleTool(server *mcp.Server, tool *mcp.Tool, secOpts *SecurityOptions) bool {
	switch tool.Name {
	case "whodb_platform_doctor":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformDoctorInput) (*mcp.CallToolResult, PlatformDoctorOutput, error) {
			return HandlePlatformDoctor(ctx, req, input)
		})
	case "whodb_platform_bundle_export":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformBundleExportInput) (*mcp.CallToolResult, PlatformBundleExportOutput, error) {
			return HandlePlatformBundleExport(ctx, req, input)
		})
	case "whodb_platform_bundle_diff":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformBundlePlanInput) (*mcp.CallToolResult, PlatformBundlePlanOutput, error) {
			return HandlePlatformBundlePlan(ctx, req, input, true, "platform_bundle_diff")
		})
	case "whodb_platform_bundle_import_plan":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformBundlePlanInput) (*mcp.CallToolResult, PlatformBundlePlanOutput, error) {
			return HandlePlatformBundlePlan(ctx, req, input, true, "platform_bundle_import_plan")
		})
	case "whodb_platform_clone":
		if secOpts.ReadOnly {
			return true
		}
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformCloneInput) (*mcp.CallToolResult, PlatformGenericWriteOutput, error) {
			return HandlePlatformClone(ctx, req, input, secOpts.ConfirmWrites)
		})
	default:
		return false
	}
	return true
}

func platformBundleToolDefinitions() []*mcp.Tool {
	return []*mcp.Tool{
		{Name: "whodb_platform_doctor", Description: descPlatformDoctor, Annotations: platformReadOnlyAnnotations("Check Hosted Platform MCP Readiness")},
		{Name: "whodb_platform_bundle_export", Description: descPlatformBundleExport, Annotations: platformReadOnlyAnnotations("Export Hosted Project Bundle")},
		{Name: "whodb_platform_bundle_diff", Description: descPlatformBundleDiff, Annotations: platformReadOnlyAnnotations("Diff Hosted Project Bundle")},
		{Name: "whodb_platform_bundle_import_plan", Description: descPlatformBundleImportPlan, Annotations: platformReadOnlyAnnotations("Plan Hosted Project Bundle Import")},
		{Name: "whodb_platform_clone", Description: descPlatformClone, Annotations: platformDestructiveAnnotations("Clone Hosted Platform Resource")},
	}
}

// HandlePlatformDoctor reports whether hosted platform MCP tools are ready to use.
func HandlePlatformDoctor(ctx context.Context, req *mcp.CallToolRequest, input PlatformDoctorInput) (*mcp.CallToolResult, PlatformDoctorOutput, error) {
	requestID := generateRequestID("platform_doctor")
	startTime := time.Now()
	session, err := loadPlatformToolSession(ctx)
	if err != nil {
		TrackToolCall(ctx, "platform_doctor", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "platform_session"})
		return nil, PlatformDoctorOutput{Error: err.Error(), RequestID: requestID}, nil
	}
	output := PlatformDoctorOutput{
		Host:              session.Host.URL,
		WorkspaceSelected: hasPlatformWorkspace(session),
		OrgID:             session.Host.DefaultOrgID,
		OrgName:           session.Host.DefaultOrgName,
		ProjectID:         session.Host.DefaultProjectID,
		ProjectName:       session.Host.DefaultProjectName,
		RequestID:         requestID,
	}
	output.Checks = append(output.Checks, "authenticated")
	user, err := session.Client.Me(ctx)
	if err != nil {
		output.Error = err.Error()
		TrackToolCall(ctx, "platform_doctor", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "platform_user"})
		return nil, output, nil
	}
	output.Email = user.Email
	manifest, err := session.Client.PlatformManifest(ctx)
	if err != nil {
		output.Error = err.Error()
		TrackToolCall(ctx, "platform_doctor", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "platform_manifest"})
		return nil, output, nil
	}
	output.PlatformVersion = manifest.PlatformVersion
	output.ManifestProtocolVersion = manifest.ManifestProtocolVersion
	output.Checks = append(output.Checks, "manifest_available")
	if output.WorkspaceSelected {
		output.Checks = append(output.Checks, "workspace_selected")
	} else {
		output.Warnings = append(output.Warnings, "No hosted org/project is selected. Run whodb-cli use --org <org> --project <project> before project-scoped tools.")
	}
	TrackToolCall(ctx, "platform_doctor", requestID, true, time.Since(startTime).Milliseconds(), map[string]any{"workspace_selected": output.WorkspaceSelected})
	return nil, output, nil
}

// HandlePlatformBundleExport exports selected-project metadata as a bundle.
func HandlePlatformBundleExport(ctx context.Context, req *mcp.CallToolRequest, input PlatformBundleExportInput) (*mcp.CallToolResult, PlatformBundleExportOutput, error) {
	requestID := generateRequestID("platform_bundle_export")
	startTime := time.Now()
	session, err := loadPlatformWorkspace(ctx)
	if err != nil {
		TrackToolCall(ctx, "platform_bundle_export", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "platform_session"})
		return nil, PlatformBundleExportOutput{Error: err.Error(), RequestID: requestID}, nil
	}
	project := selectedPlatformProject(session)
	bundle, err := platformapi.BuildProjectBundle(ctx, session.Client, session.Host.URL, session.Host.DefaultOrgID, session.Host.DefaultOrgName, project)
	if err != nil {
		TrackToolCall(ctx, "platform_bundle_export", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "platform_query"})
		return nil, PlatformBundleExportOutput{Error: err.Error(), RequestID: requestID}, nil
	}
	TrackToolCall(ctx, "platform_bundle_export", requestID, true, time.Since(startTime).Milliseconds(), platformBundleTelemetryCounts(bundle))
	return nil, PlatformBundleExportOutput{Bundle: bundle, Counts: platformBundleCounts(bundle), RequestID: requestID}, nil
}

// HandlePlatformBundlePlan plans a bundle import into the selected project.
func HandlePlatformBundlePlan(ctx context.Context, req *mcp.CallToolRequest, input PlatformBundlePlanInput, dryRun bool, toolName string) (*mcp.CallToolResult, PlatformBundlePlanOutput, error) {
	requestID := generateRequestID(toolName)
	startTime := time.Now()
	var bundle platformapi.ProjectBundle
	if strings.TrimSpace(input.BundleJSON) == "" {
		return nil, PlatformBundlePlanOutput{Error: "bundle_json is required", RequestID: requestID}, nil
	}
	if err := json.Unmarshal([]byte(input.BundleJSON), &bundle); err != nil {
		return nil, PlatformBundlePlanOutput{Error: "decode bundle_json: " + err.Error(), RequestID: requestID}, nil
	}
	if bundle.BundleVersion != 1 {
		return nil, PlatformBundlePlanOutput{Error: "unsupported bundle version", RequestID: requestID}, nil
	}
	session, err := loadPlatformWorkspace(ctx)
	if err != nil {
		TrackToolCall(ctx, toolName, requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "platform_session"})
		return nil, PlatformBundlePlanOutput{Error: err.Error(), RequestID: requestID}, nil
	}
	plan, err := platformapi.PlanBundleImport(ctx, session.Client, session.Host.URL, selectedPlatformProject(session), &bundle, dryRun, nil)
	if err != nil {
		TrackToolCall(ctx, toolName, requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "platform_query"})
		return nil, PlatformBundlePlanOutput{Error: err.Error(), RequestID: requestID}, nil
	}
	counts := platformBundlePlanCounts(plan)
	TrackToolCall(ctx, toolName, requestID, true, time.Since(startTime).Milliseconds(), platformBundlePlanTelemetryCounts(plan))
	return nil, PlatformBundlePlanOutput{Plan: plan, Counts: counts, RequestID: requestID}, nil
}

// HandlePlatformClone clones a dataset, ontology, transform, or function.
func HandlePlatformClone(ctx context.Context, req *mcp.CallToolRequest, input PlatformCloneInput, confirmWrites bool) (*mcp.CallToolResult, PlatformGenericWriteOutput, error) {
	requestID := generateRequestID("platform_clone")
	resource := normalizePlatformWriteToken(input.Resource)
	if resource == "" || strings.TrimSpace(input.Source) == "" || strings.TrimSpace(input.NewName) == "" {
		return nil, PlatformGenericWriteOutput{Error: "resource, source, and new_name are required", RequestID: requestID}, nil
	}
	session, err := loadPlatformWorkspace(ctx)
	if err != nil {
		return nil, PlatformGenericWriteOutput{Error: err.Error(), RequestID: requestID}, nil
	}
	payload, err := platformapi.BuildClonePayload(ctx, session.Client, session.Host.DefaultProjectID, resource, input.Source, input.NewName)
	if err != nil {
		return nil, PlatformGenericWriteOutput{Error: err.Error(), RequestID: requestID}, nil
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, PlatformGenericWriteOutput{Error: err.Error(), RequestID: requestID}, nil
	}
	return handlePlatformTypedGenericWrite(ctx, "whodb_platform_clone", PlatformGenericWriteInput{Resource: resource, PayloadJSON: string(raw)}, "create", confirmWrites)
}

func selectedPlatformProject(session *platformToolSession) *platformapi.Project {
	return &platformapi.Project{
		ID:    session.Host.DefaultProjectID,
		OrgID: session.Host.DefaultOrgID,
		Name:  session.Host.DefaultProjectName,
	}
}

func platformBundleCounts(bundle *platformapi.ProjectBundle) map[string]int {
	return map[string]int{
		"secrets":    len(bundle.Secrets),
		"datasets":   len(bundle.Datasets),
		"ontologies": len(bundle.Ontologies),
		"transforms": len(bundle.Transforms),
		"functions":  len(bundle.Functions),
		"folders":    len(bundle.Folders),
		"files":      len(bundle.Files),
	}
}

func platformBundleTelemetryCounts(bundle *platformapi.ProjectBundle) map[string]any {
	counts := platformBundleCounts(bundle)
	telemetry := make(map[string]any, len(counts))
	for key, value := range counts {
		telemetry[key] = value
	}
	return telemetry
}

func platformBundlePlanCounts(plan *platformapi.BundlePlan) map[string]int {
	counts := map[string]int{}
	for _, action := range plan.Actions {
		counts[action.Action]++
	}
	return counts
}

func platformBundlePlanTelemetryCounts(plan *platformapi.BundlePlan) map[string]any {
	counts := platformBundlePlanCounts(plan)
	telemetry := make(map[string]any, len(counts))
	for key, value := range counts {
		telemetry[key] = value
	}
	return telemetry
}

const descPlatformDoctor = `Check hosted WhoDB MCP readiness.

Use this when platform tools fail or before a workflow. It reports login, selected workspace, manifest availability, and actionable warnings.`

const descPlatformBundleExport = `Export selected hosted project metadata as a portable bundle.

The bundle includes metadata for secrets, datasets, ontologies, transforms, functions, folders, and files. Secret values and file bytes are not exported.`

const descPlatformBundleDiff = `Compare a project bundle against the selected hosted project.

Pass bundle_json from whodb_platform_bundle_export or the CLI resources export command. This is read-only and returns create/skip actions.`

const descPlatformBundleImportPlan = `Plan how a project bundle would import into the selected hosted project.

This MCP tool is intentionally plan-only. Use the CLI resources import --yes command for execution after review.`

const descPlatformClone = `Clone a hosted dataset, ontology, transform, or function in the selected project.

Default mode returns a confirmation token; do not call whodb_platform_confirm until the user approves the preview.`
