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
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/clidey/whodb/cli/internal/config"
	platformapi "github.com/clidey/whodb/cli/internal/platform"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// PlatformSetupStatusInput is the input for whodb_platform_setup_status.
type PlatformSetupStatusInput struct{}

// PlatformSetupStatusOutput reports local hosted platform MCP setup state.
type PlatformSetupStatusOutput struct {
	Host              string   `json:"host"`
	Status            string   `json:"status"`
	Authenticated     bool     `json:"authenticated"`
	WorkspaceSelected bool     `json:"workspace_selected"`
	Email             string   `json:"email,omitempty"`
	AccountID         string   `json:"account_id,omitempty"`
	OrgID             string   `json:"org_id,omitempty"`
	OrgName           string   `json:"org_name,omitempty"`
	ProjectID         string   `json:"project_id,omitempty"`
	ProjectName       string   `json:"project_name,omitempty"`
	Commands          []string `json:"commands"`
	NextSteps         []string `json:"next_steps"`
	Error             string   `json:"error,omitempty"`
	RequestID         string   `json:"request_id,omitempty"`
}

// PlatformSetupGuidance gives agents recovery instructions for setup-related failures.
type PlatformSetupGuidance struct {
	SetupStatus string   `json:"setup_status,omitempty"`
	Commands    []string `json:"commands,omitempty"`
	NextSteps   []string `json:"next_steps,omitempty"`
}

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
	NextSteps               []string `json:"next_steps,omitempty"`
	Commands                []string `json:"commands,omitempty"`
	Error                   string   `json:"error,omitempty"`
	RequestID               string   `json:"request_id,omitempty"`
}

// PlatformBundleExportInput is the input for whodb_platform_bundle_export.
type PlatformBundleExportInput struct {
	IncludeFiles bool `json:"include_files,omitempty" jsonschema:"Include previewable uploaded file content up to max_file_bytes per file"`
	MaxFileBytes int  `json:"max_file_bytes,omitempty" jsonschema:"Maximum bytes to include per uploaded file when include_files is true"`
}

// PlatformBundleExportOutput returns a selected-project metadata bundle.
type PlatformBundleExportOutput struct {
	PlatformSetupGuidance
	Bundle    *platformapi.ProjectBundle `json:"bundle,omitempty"`
	Counts    map[string]int             `json:"counts,omitempty"`
	Error     string                     `json:"error,omitempty"`
	RequestID string                     `json:"request_id,omitempty"`
}

// PlatformBundlePlanInput is the input for bundle diff and import-plan tools.
type PlatformBundlePlanInput struct {
	BundleJSON         string `json:"bundle_json" jsonschema:"Project bundle JSON from whodb_platform_bundle_export or resources export"`
	Prefix             string `json:"prefix,omitempty" jsonschema:"Optional prefix added to imported resource names"`
	RenameConflicts    bool   `json:"rename_conflicts,omitempty" jsonschema:"Create unique names for resources that conflict with existing resources"`
	OverwriteConflicts bool   `json:"overwrite_conflicts,omitempty" jsonschema:"Update resources that conflict with existing resources"`
}

// PlatformBundlePlanOutput returns a bundle import plan for the selected project.
type PlatformBundlePlanOutput struct {
	PlatformSetupGuidance
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
	case "whodb_platform_setup_status":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformSetupStatusInput) (*mcp.CallToolResult, PlatformSetupStatusOutput, error) {
			return HandlePlatformSetupStatus(ctx, req, input)
		})
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
	case "whodb_platform_bundle_import":
		if secOpts.ReadOnly {
			return true
		}
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformBundlePlanInput) (*mcp.CallToolResult, PlatformGenericWriteOutput, error) {
			return HandlePlatformBundleImport(ctx, req, input, secOpts.ConfirmWrites)
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
		{Name: "whodb_platform_setup_status", Description: descPlatformSetupStatus, Annotations: platformReadOnlyAnnotations("Check Hosted Platform MCP Setup")},
		{Name: "whodb_platform_doctor", Description: descPlatformDoctor, Annotations: platformReadOnlyAnnotations("Check Hosted Platform MCP Readiness")},
		{Name: "whodb_platform_bundle_export", Description: descPlatformBundleExport, Annotations: platformReadOnlyAnnotations("Export Hosted Project Bundle")},
		{Name: "whodb_platform_bundle_diff", Description: descPlatformBundleDiff, Annotations: platformReadOnlyAnnotations("Diff Hosted Project Bundle")},
		{Name: "whodb_platform_bundle_import_plan", Description: descPlatformBundleImportPlan, Annotations: platformReadOnlyAnnotations("Plan Hosted Project Bundle Import")},
		{Name: "whodb_platform_bundle_import", Description: descPlatformBundleImport, Annotations: platformDestructiveAnnotations("Import Hosted Project Bundle")},
		{Name: "whodb_platform_clone", Description: descPlatformClone, Annotations: platformDestructiveAnnotations("Clone Hosted Platform Resource")},
	}
}

// HandlePlatformSetupStatus reports local hosted platform setup without requiring a valid session.
func HandlePlatformSetupStatus(ctx context.Context, req *mcp.CallToolRequest, input PlatformSetupStatusInput) (*mcp.CallToolResult, PlatformSetupStatusOutput, error) {
	requestID := generateRequestID("platform_setup_status")
	startTime := time.Now()
	output := buildPlatformSetupStatus(requestID)
	success := output.Status == "ready"
	TrackToolCall(ctx, "platform_setup_status", requestID, success, time.Since(startTime).Milliseconds(), map[string]any{"status": output.Status})
	return nil, output, nil
}

func buildPlatformSetupStatus(requestID string) PlatformSetupStatusOutput {
	host := platformapi.DefaultHost
	cfg, err := config.LoadConfigWithoutSecrets()
	if err != nil {
		output := platformSetupStatusFor(host, "config_error")
		output.Error = fmt.Sprintf("cannot load hosted WhoDB config: %v", err)
		output.RequestID = requestID
		return output
	}
	if strings.TrimSpace(cfg.Platform.DefaultHost) != "" {
		host = cfg.Platform.DefaultHost
	}
	normalizedHost, err := platformapi.NormalizeHost(host)
	if err != nil {
		output := platformSetupStatusFor(platformapi.DefaultHost, "config_error")
		output.Error = err.Error()
		output.RequestID = requestID
		return output
	}
	host = normalizedHost
	entry, ok := cfg.GetPlatformHost(host)
	if !ok || strings.TrimSpace(entry.AccountID) == "" {
		output := platformSetupStatusFor(host, "needs_login")
		output.RequestID = requestID
		return output
	}

	output := platformSetupStatusFor(host, "")
	output.Authenticated = true
	output.Email = entry.Email
	output.AccountID = entry.AccountID
	output.OrgID = entry.DefaultOrgID
	output.OrgName = entry.DefaultOrgName
	output.ProjectID = entry.DefaultProjectID
	output.ProjectName = entry.DefaultProjectName
	output.WorkspaceSelected = strings.TrimSpace(entry.DefaultOrgID) != "" && strings.TrimSpace(entry.DefaultProjectID) != ""

	if _, err := cfg.GetPlatformRefreshToken(host, entry.AccountID); err != nil {
		output.Authenticated = false
		output.Status = "needs_login"
		output.Error = fmt.Sprintf("cannot load hosted WhoDB refresh token: %v", err)
	} else if !output.WorkspaceSelected {
		output.Status = "needs_workspace"
	} else {
		output.Status = "ready"
	}
	applyPlatformSetupGuidance(&output)
	output.RequestID = requestID
	return output
}

func platformSetupStatusFor(host, status string) PlatformSetupStatusOutput {
	output := PlatformSetupStatusOutput{
		Host:   host,
		Status: status,
	}
	applyPlatformSetupGuidance(&output)
	return output
}

func platformSetupGuidanceFromStatus(output PlatformSetupStatusOutput) PlatformSetupGuidance {
	return PlatformSetupGuidance{
		SetupStatus: output.Status,
		Commands:    output.Commands,
		NextSteps:   output.NextSteps,
	}
}

func platformSetupGuidanceForCurrentConfig(requestID string) PlatformSetupGuidance {
	return platformSetupGuidanceFromStatus(buildPlatformSetupStatus(requestID))
}

func platformSetupGuidanceForError(err error, requestID string) PlatformSetupGuidance {
	if err == nil || !isPlatformSetupError(err) {
		return PlatformSetupGuidance{}
	}
	return platformSetupGuidanceForCurrentConfig(requestID)
}

func isPlatformSetupError(err error) bool {
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "not logged in") ||
		strings.Contains(message, "refresh token") ||
		strings.Contains(message, "refresh hosted whodb login") ||
		strings.Contains(message, "no hosted whodb workspace selected") ||
		strings.Contains(message, "cannot load hosted whodb config")
}

func applyPlatformSetupGuidance(output *PlatformSetupStatusOutput) {
	host := strings.TrimSpace(output.Host)
	if host == "" {
		host = platformapi.DefaultHost
		output.Host = host
	}
	loginCommand := fmt.Sprintf("whodb-cli login --host %s", host)
	useCommand := platformUseCommand(host)
	switch output.Status {
	case "ready":
		output.Commands = []string{}
		output.NextSteps = []string{
			"Use whodb_platform_project_health for broad project context.",
			"Read whodb://platform/concepts for the product model and workflow recipes.",
		}
	case "needs_workspace":
		output.Commands = []string{useCommand}
		output.NextSteps = []string{
			"Run whodb_platform_orgs to list available organizations.",
			"Run whodb_platform_projects with the selected organization.",
			"Ask the user to run: " + useCommand,
		}
	default:
		output.Commands = []string{loginCommand, useCommand}
		output.NextSteps = []string{
			"Ask the user to run: " + loginCommand,
			"Then ask the user to select a workspace with: " + useCommand,
		}
		if output.Status == "" {
			output.Status = "needs_login"
		}
	}
}

// HandlePlatformDoctor reports whether hosted platform MCP tools are ready to use.
func HandlePlatformDoctor(ctx context.Context, req *mcp.CallToolRequest, input PlatformDoctorInput) (*mcp.CallToolResult, PlatformDoctorOutput, error) {
	requestID := generateRequestID("platform_doctor")
	startTime := time.Now()
	session, err := loadPlatformToolSession(ctx)
	if err != nil {
		setup := buildPlatformSetupStatus(requestID)
		TrackToolCall(ctx, "platform_doctor", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "platform_session"})
		return nil, PlatformDoctorOutput{
			Host:              setup.Host,
			Email:             setup.Email,
			WorkspaceSelected: setup.WorkspaceSelected,
			OrgID:             setup.OrgID,
			OrgName:           setup.OrgName,
			ProjectID:         setup.ProjectID,
			ProjectName:       setup.ProjectName,
			Checks:            []string{"setup_status:" + setup.Status},
			Warnings:          setup.NextSteps,
			NextSteps:         setup.NextSteps,
			Commands:          setup.Commands,
			Error:             err.Error(),
			RequestID:         requestID,
		}, nil
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
		setup := platformSetupStatusFor(session.Host.URL, "needs_workspace")
		output.Warnings = append(output.Warnings, setup.NextSteps...)
		output.NextSteps = setup.NextSteps
		output.Commands = setup.Commands
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
		return nil, PlatformBundleExportOutput{PlatformSetupGuidance: platformSetupGuidanceForCurrentConfig(requestID), Error: err.Error(), RequestID: requestID}, nil
	}
	project := selectedPlatformProject(session)
	bundle, err := platformapi.BuildProjectBundleWithOptions(ctx, session.Client, session.Host.URL, session.Host.DefaultOrgID, session.Host.DefaultOrgName, project, platformapi.BundleExportOptions{
		IncludeFiles: input.IncludeFiles,
		MaxFileBytes: input.MaxFileBytes,
	})
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
		return nil, PlatformBundlePlanOutput{PlatformSetupGuidance: platformSetupGuidanceForCurrentConfig(requestID), Error: err.Error(), RequestID: requestID}, nil
	}
	plan, err := platformapi.PlanBundleImportWithOptions(ctx, session.Client, session.Host.URL, selectedPlatformProject(session), &bundle, platformapi.BundleImportOptions{
		DryRun:             dryRun,
		Prefix:             input.Prefix,
		RenameConflicts:    input.RenameConflicts,
		OverwriteConflicts: input.OverwriteConflicts,
	})
	if err != nil {
		TrackToolCall(ctx, toolName, requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "platform_query"})
		return nil, PlatformBundlePlanOutput{Error: err.Error(), RequestID: requestID}, nil
	}
	counts := platformBundlePlanCounts(plan)
	TrackToolCall(ctx, toolName, requestID, true, time.Since(startTime).Milliseconds(), platformBundlePlanTelemetryCounts(plan))
	return nil, PlatformBundlePlanOutput{Plan: plan, Counts: counts, RequestID: requestID}, nil
}

// HandlePlatformBundleImport prepares or executes a bundle import into the selected project.
func HandlePlatformBundleImport(ctx context.Context, req *mcp.CallToolRequest, input PlatformBundlePlanInput, confirmWrites bool) (*mcp.CallToolResult, PlatformGenericWriteOutput, error) {
	requestID := generateRequestID("platform_bundle_import")
	startTime := time.Now()
	var bundle platformapi.ProjectBundle
	if strings.TrimSpace(input.BundleJSON) == "" {
		return nil, PlatformGenericWriteOutput{Error: "bundle_json is required", RequestID: requestID}, nil
	}
	if err := json.Unmarshal([]byte(input.BundleJSON), &bundle); err != nil {
		return nil, PlatformGenericWriteOutput{Error: "decode bundle_json: " + err.Error(), RequestID: requestID}, nil
	}
	if bundle.BundleVersion != 1 {
		return nil, PlatformGenericWriteOutput{Error: "unsupported bundle version", RequestID: requestID}, nil
	}
	session, err := loadPlatformWorkspace(ctx)
	if err != nil {
		TrackToolCall(ctx, "platform_bundle_import", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "platform_session"})
		return nil, platformGenericWriteSetupError(err, requestID), nil
	}
	plan, err := platformapi.PlanBundleImportWithOptions(ctx, session.Client, session.Host.URL, selectedPlatformProject(session), &bundle, platformapi.BundleImportOptions{
		DryRun:             false,
		Prefix:             input.Prefix,
		RenameConflicts:    input.RenameConflicts,
		OverwriteConflicts: input.OverwriteConflicts,
	})
	if err != nil {
		TrackToolCall(ctx, "platform_bundle_import", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "platform_query"})
		return nil, PlatformGenericWriteOutput{Error: err.Error(), RequestID: requestID}, nil
	}
	action := &PendingPlatformAction{
		Operation:   "bundle_import",
		Resource:    "bundle",
		Action:      "import",
		Host:        session.Host.URL,
		OrgID:       session.Host.DefaultOrgID,
		ProjectID:   session.Host.DefaultProjectID,
		ProjectName: session.Host.DefaultProjectName,
		Summary:     "Import hosted project bundle",
		Changes:     platformBundlePlanChanges(plan),
		BundlePlan:  plan,
	}
	if !confirmWrites {
		output, err := executePendingPlatformAction(ctx, action, requestID)
		if err != nil {
			TrackToolCall(ctx, "platform_bundle_import", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "platform_action"})
			return nil, PlatformGenericWriteOutput{PlatformSetupGuidance: platformSetupGuidanceForError(err, requestID), Error: err.Error(), RequestID: requestID}, nil
		}
		raw, _ := json.Marshal(output)
		TrackToolCall(ctx, "platform_bundle_import", requestID, true, time.Since(startTime).Milliseconds(), map[string]any{"confirmation_required": false})
		return nil, PlatformGenericWriteOutput{Status: "ok", ResultJSON: string(raw), RequestID: requestID}, nil
	}
	token, expiresAt := storePendingPlatformAction(action)
	TrackToolCall(ctx, "platform_bundle_import", requestID, true, time.Since(startTime).Milliseconds(), map[string]any{"confirmation_required": true})
	return nil, platformGenericConfirmationOutput(requestID, token, expiresAt, "bundle_import", action.Preview()), nil
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
		return nil, platformGenericWriteSetupError(err, requestID), nil
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
		"secrets":      len(bundle.Secrets),
		"ai_providers": len(bundle.AIProviders),
		"datasets":     len(bundle.Datasets),
		"ontologies":   len(bundle.Ontologies),
		"transforms":   len(bundle.Transforms),
		"functions":    len(bundle.Functions),
		"folders":      len(bundle.Folders),
		"files":        len(bundle.Files),
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

func platformBundlePlanChanges(plan *platformapi.BundlePlan) []string {
	if plan == nil {
		return nil
	}
	changes := make([]string, 0, len(plan.Actions))
	for _, action := range plan.Actions {
		if action.Action == "create" || action.Action == "update" {
			changes = append(changes, action.Action+" "+action.Resource+" "+action.Name)
			for _, impact := range action.Impacts {
				if strings.TrimSpace(impact) != "" {
					changes = append(changes, action.Resource+" "+action.Name+": "+impact)
				}
			}
		}
	}
	return changes
}

func executePlatformBundlePlan(ctx context.Context, session *platformToolSession, plan *platformapi.BundlePlan, requestID string) (ConfirmOutput, error) {
	dependencies := platformapi.BundleDependencyMap{}
	for _, action := range plan.Actions {
		platformapi.AddBundleDependencyMapping(dependencies, action.SourceID, action.TargetID)
	}
	rows := make([][]any, 0, len(plan.Actions))
	for i := range plan.Actions {
		action := &plan.Actions[i]
		if action.Action != "create" && action.Action != "update" {
			rows = append(rows, []any{action.Resource, action.Name, action.Action, action.Reason, action.TargetID})
			continue
		}
		platformapi.ApplyBundleDependencyMap(action, dependencies)
		if action.Resource == "file" {
			file, err := uploadPlatformBundleFileAction(ctx, session, action)
			if err != nil {
				action.Action = "failed"
				action.Reason = err.Error()
				rows = append(rows, []any{action.Resource, action.Name, action.Action, action.Reason, action.TargetID})
				continue
			}
			action.TargetID = file.ID
			action.Action = "created"
			rows = append(rows, []any{action.Resource, action.Name, action.Action, action.Reason, action.TargetID})
			continue
		}
		raw, err := json.Marshal(platformapi.BundleMutationPayload(action))
		if err != nil {
			action.Action = "failed"
			action.Reason = err.Error()
			rows = append(rows, []any{action.Resource, action.Name, action.Action, action.Reason, action.TargetID})
			continue
		}
		spec, variables, err := buildPlatformGenericWrite(session, PlatformGenericWriteInput{
			Resource:    action.Resource,
			ID:          action.TargetID,
			PayloadJSON: string(raw),
		}, action.Action)
		if err != nil {
			action.Action = "failed"
			action.Reason = err.Error()
			rows = append(rows, []any{action.Resource, action.Name, action.Action, action.Reason, action.TargetID})
			continue
		}
		result, err := executePlatformMutation(ctx, session.Client, spec.Mutation, session.Host.DefaultProjectID, variables)
		if err != nil {
			action.Action = "failed"
			action.Reason = err.Error()
			rows = append(rows, []any{action.Resource, action.Name, action.Action, action.Reason, action.TargetID})
			continue
		}
		if id := platformMutationResultID(result); id != "" {
			action.TargetID = id
			platformapi.AddBundleDependencyMapping(dependencies, action.SourceID, action.TargetID)
		}
		if action.Action == "create" {
			action.Action = "created"
		} else {
			action.Action = "updated"
		}
		rows = append(rows, []any{action.Resource, action.Name, action.Action, action.Reason, action.TargetID})
	}
	return ConfirmOutput{
		Columns:   []string{"resource", "name", "action", "reason", "target_id"},
		Rows:      rows,
		Message:   "Hosted platform bundle import completed",
		RequestID: requestID,
	}, nil
}

func uploadPlatformBundleFileAction(ctx context.Context, session *platformToolSession, action *platformapi.BundleAction) (*platformapi.ProjectFile, error) {
	content, _ := action.Payload["content"].(string)
	if strings.TrimSpace(action.Name) == "" {
		return nil, fmt.Errorf("file name is required")
	}
	tmpDir, err := os.MkdirTemp("", "whodb-mcp-bundle-file-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)
	path := filepath.Join(tmpDir, filepath.Base(action.Name))
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		return nil, err
	}
	folderID, _ := action.Payload["folderId"].(string)
	if strings.TrimSpace(folderID) == "" {
		return session.Client.UploadProjectFile(ctx, session.Host.DefaultProjectID, nil, path)
	}
	return session.Client.UploadProjectFile(ctx, session.Host.DefaultProjectID, &folderID, path)
}

func platformMutationResultID(result *platformapi.PlatformMutationResult) string {
	if result == nil || len(result.Data) == 0 {
		return ""
	}
	var payload map[string]any
	if err := json.Unmarshal(result.Data, &payload); err != nil {
		return ""
	}
	id, _ := payload["id"].(string)
	return id
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

const descPlatformSetupStatus = `Check hosted WhoDB MCP setup before auth-dependent tools.

Use this as the first tool in hosted platform mode, or whenever another platform tool reports login/workspace errors. It does not require a valid hosted session. It reports whether the local CLI is logged in, whether an organization/project is selected, and exact commands the user can run to fix setup.`

const descPlatformDoctor = `Check hosted WhoDB MCP readiness.

Use this when platform tools fail or before a workflow. It reports login, selected workspace, manifest availability, and actionable warnings.`

const descPlatformBundleExport = `Export selected hosted project metadata as a portable bundle.

The bundle includes metadata for secrets, datasets, ontologies, transforms, functions, folders, and files. Secret values are never exported. File content is only exported when include_files is true and is capped by max_file_bytes.`

const descPlatformBundleDiff = `Compare a project bundle against the selected hosted project.

Pass bundle_json from whodb_platform_bundle_export or the CLI resources export command. This is read-only and returns create/skip actions.`

const descPlatformBundleImportPlan = `Plan how a project bundle would import into the selected hosted project.

This MCP tool is intentionally plan-only. Use the CLI resources import --yes command for execution after review.`

const descPlatformBundleImport = `Import a project bundle into the selected hosted project.

Use whodb_platform_bundle_import_plan first when possible. Default mode returns a confirmation token; do not call whodb_platform_confirm until the user approves the import preview. Supports prefix, rename_conflicts, and overwrite_conflicts.`

const descPlatformClone = `Clone a hosted dataset, ontology, transform, or function in the selected project.

Default mode returns a confirmation token; do not call whodb_platform_confirm until the user approves the preview.`
