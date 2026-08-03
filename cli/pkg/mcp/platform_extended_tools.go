/*
 * Copyright 2026 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 */

package mcp

import (
	"context"
	"fmt"
	"strings"

	platformapi "github.com/clidey/whodb/cli/internal/platform"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// PlatformAppInput selects one hosted app or app-related view.
type PlatformAppInput struct {
	ID      string   `json:"id,omitempty" jsonschema:"App id for detail, files, or version view"`
	Version int      `json:"version,omitempty" jsonschema:"App version for a version view"`
	Env     string   `json:"env,omitempty" jsonschema:"Optional app environment for the current view"`
	Fields  []string `json:"fields,omitempty" jsonschema:"Optional top-level output fields"`
}

// PlatformPackageInput selects one hosted package or installation.
type PlatformPackageInput struct {
	ID     string   `json:"id,omitempty" jsonschema:"Package or installation id"`
	Search string   `json:"search,omitempty" jsonschema:"Optional package library search"`
	Fields []string `json:"fields,omitempty" jsonschema:"Optional top-level output fields"`
}

// PlatformPackagePreviewInput describes the input object for a package dry run.
type PlatformPackagePreviewInput struct {
	InputJSON string   `json:"input_json" jsonschema:"JSON object matching the selected package preview input type"`
	Fields    []string `json:"fields,omitempty" jsonschema:"Optional top-level output fields"`
}

// PlatformVersionInput selects an object version history.
type PlatformVersionInput struct {
	ID         string   `json:"id" jsonschema:"Object id"`
	ObjectType string   `json:"object_type" jsonschema:"Versionable object type: app, function, transform, dataset, or ontology"`
	Fields     []string `json:"fields,omitempty" jsonschema:"Optional top-level output fields"`
}

// PlatformAccessInput selects resource access information.
type PlatformAccessInput struct {
	ResourceType string   `json:"resource_type,omitempty" jsonschema:"Resource type, for example dataset, app, or function"`
	ResourceID   string   `json:"resource_id,omitempty" jsonschema:"Resource id"`
	TeamID       string   `json:"team_id,omitempty" jsonschema:"Team id for team members"`
	Fields       []string `json:"fields,omitempty" jsonschema:"Optional top-level output fields"`
}

func registerPlatformExtendedReadTool(server *mcp.Server, tool *mcp.Tool) bool {
	switch tool.Name {
	case "whodb_platform_apps":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformAppInput) (*mcp.CallToolResult, any, error) {
			return handlePlatformExtendedQuery(ctx, "platform_apps", "ProjectApps", map[string]any{}, input.Fields)
		})
	case "whodb_platform_app":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformAppInput) (*mcp.CallToolResult, any, error) {
			return handlePlatformExtendedQuery(ctx, "platform_app", "AppDetail", map[string]any{"id": strings.TrimSpace(input.ID)}, input.Fields)
		})
	case "whodb_platform_app_files":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformAppInput) (*mcp.CallToolResult, any, error) {
			return handlePlatformExtendedQuery(ctx, "platform_app_files", "AppFiles", map[string]any{"appId": strings.TrimSpace(input.ID)}, input.Fields)
		})
	case "whodb_platform_app_view":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformAppInput) (*mcp.CallToolResult, any, error) {
			return handlePlatformExtendedQuery(ctx, "platform_app_view", "AppView", map[string]any{"id": strings.TrimSpace(input.ID), "env": optionalPlatformString(input.Env)}, input.Fields)
		})
	case "whodb_platform_app_version_view":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformAppInput) (*mcp.CallToolResult, any, error) {
			return handlePlatformExtendedQuery(ctx, "platform_app_version_view", "AppVersionView", map[string]any{"appId": strings.TrimSpace(input.ID), "version": input.Version}, input.Fields)
		})
	case "whodb_platform_packages":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformPackageInput) (*mcp.CallToolResult, any, error) {
			return handlePlatformExtendedQuery(ctx, "platform_packages", "Packages", map[string]any{}, input.Fields)
		})
	case "whodb_platform_package":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformPackageInput) (*mcp.CallToolResult, any, error) {
			return handlePlatformExtendedQuery(ctx, "platform_package", "PackageDetail", map[string]any{"packageId": strings.TrimSpace(input.ID)}, input.Fields)
		})
	case "whodb_platform_package_installations":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformPackageInput) (*mcp.CallToolResult, any, error) {
			return handlePlatformExtendedQuery(ctx, "platform_package_installations", "PackageInstallations", map[string]any{}, input.Fields)
		})
	case "whodb_platform_package_installation_update":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformPackageInput) (*mcp.CallToolResult, any, error) {
			return handlePlatformExtendedQuery(ctx, "platform_package_installation_update", "PackageInstallationUpdate", map[string]any{"installationId": strings.TrimSpace(input.ID)}, input.Fields)
		})
	case "whodb_platform_package_library":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformPackageInput) (*mcp.CallToolResult, any, error) {
			return handlePlatformExtendedQuery(ctx, "platform_package_library", "PackageLibrary", map[string]any{"search": optionalPlatformString(input.Search)}, input.Fields)
		})
	case "whodb_platform_shared_package":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformPackageInput) (*mcp.CallToolResult, any, error) {
			return handlePlatformExtendedQuery(ctx, "platform_shared_package", "SharedPackage", map[string]any{"shareToken": strings.TrimSpace(input.ID)}, input.Fields)
		})
	case "whodb_platform_preview_create_package", "whodb_platform_preview_install_package", "whodb_platform_preview_import_package", "whodb_platform_preview_shared_install", "whodb_platform_preview_shared_import":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformPackagePreviewInput) (*mcp.CallToolResult, any, error) {
			operation := map[string]string{
				"whodb_platform_preview_create_package":  "PreviewCreatePackage",
				"whodb_platform_preview_install_package": "PreviewInstallPackage",
				"whodb_platform_preview_import_package":  "PreviewImportPackage",
				"whodb_platform_preview_shared_install":  "PreviewSharedPackageInstall",
				"whodb_platform_preview_shared_import":   "PreviewSharedPackageImport",
			}[tool.Name]
			payload, err := parsePayloadJSON(input.InputJSON)
			if err != nil {
				return nil, PlatformReadOutput{Error: err.Error(), RequestID: generateRequestID(tool.Name)}, nil
			}
			if operation == "PreviewCreatePackage" {
				payload["projectId"] = "selected"
			} else {
				payload["targetProjectId"] = "selected"
			}
			return handlePlatformExtendedQuery(ctx, tool.Name, operation, map[string]any{"input": payload}, input.Fields)
		})
	case "whodb_platform_object_versions":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformVersionInput) (*mcp.CallToolResult, any, error) {
			return handlePlatformExtendedQuery(ctx, "platform_object_versions", "ObjectVersions", map[string]any{"objectId": strings.TrimSpace(input.ID), "objectType": strings.TrimSpace(input.ObjectType)}, input.Fields)
		})
	case "whodb_platform_active_version":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformVersionInput) (*mcp.CallToolResult, any, error) {
			return handlePlatformExtendedQuery(ctx, "platform_active_version", "ActiveProdVersion", map[string]any{"objectId": strings.TrimSpace(input.ID), "objectType": strings.TrimSpace(input.ObjectType)}, input.Fields)
		})
	case "whodb_platform_project_active_versions":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformVersionInput) (*mcp.CallToolResult, any, error) {
			return handlePlatformExtendedQuery(ctx, "platform_project_active_versions", "ProjectActiveProdVersions", map[string]any{}, input.Fields)
		})
	case "whodb_platform_resource_access":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformAccessInput) (*mcp.CallToolResult, any, error) {
			return handlePlatformExtendedQuery(ctx, "platform_resource_access", "WhoHasAccess", map[string]any{"resourceType": strings.TrimSpace(input.ResourceType), "resourceId": strings.TrimSpace(input.ResourceID)}, input.Fields)
		})
	case "whodb_platform_resource_permissions":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformAccessInput) (*mcp.CallToolResult, any, error) {
			return handlePlatformExtendedQuery(ctx, "platform_resource_permissions", "MyPermissions", map[string]any{"resourceType": strings.TrimSpace(input.ResourceType), "resourceId": strings.TrimSpace(input.ResourceID)}, input.Fields)
		})
	case "whodb_platform_resource_types_access":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformAccessInput) (*mcp.CallToolResult, any, error) {
			return handlePlatformExtendedQuery(ctx, "platform_resource_types_access", "WhatCanIAccess", map[string]any{"resourceType": strings.TrimSpace(input.ResourceType)}, input.Fields)
		})
	case "whodb_platform_deletion_impact":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformAccessInput) (*mcp.CallToolResult, any, error) {
			return handlePlatformExtendedQuery(ctx, "platform_deletion_impact", "DeletionImpact", map[string]any{"resourceType": strings.TrimSpace(input.ResourceType), "resourceId": strings.TrimSpace(input.ResourceID)}, input.Fields)
		})
	case "whodb_platform_deleted_resources":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformAccessInput) (*mcp.CallToolResult, any, error) {
			return handlePlatformExtendedQuery(ctx, "platform_deleted_resources", "DeletedResources", map[string]any{}, input.Fields)
		})
	case "whodb_platform_project_access":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformAccessInput) (*mcp.CallToolResult, any, error) {
			return handlePlatformExtendedQuery(ctx, "platform_project_access", "ProjectAccessMatrix", map[string]any{}, input.Fields)
		})
	case "whodb_platform_project_resources":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformAccessInput) (*mcp.CallToolResult, any, error) {
			return handlePlatformExtendedQuery(ctx, "platform_project_resources", "ProjectResourceSummary", map[string]any{}, input.Fields)
		})
	case "whodb_platform_org_resources":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformAccessInput) (*mcp.CallToolResult, any, error) {
			return handlePlatformExtendedQuery(ctx, "platform_org_resources", "OrgResources", map[string]any{}, input.Fields)
		})
	case "whodb_platform_org_members":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformAccessInput) (*mcp.CallToolResult, any, error) {
			return handlePlatformExtendedQuery(ctx, "platform_org_members", "OrgMembers", map[string]any{}, input.Fields)
		})
	case "whodb_platform_org_domains":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformAccessInput) (*mcp.CallToolResult, any, error) {
			return handlePlatformExtendedQuery(ctx, "platform_org_domains", "OrganizationDomains", map[string]any{}, input.Fields)
		})
	case "whodb_platform_org_sso":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformAccessInput) (*mcp.CallToolResult, any, error) {
			return handlePlatformExtendedQuery(ctx, "platform_org_sso", "OrganizationSSOProviders", map[string]any{}, input.Fields)
		})
	case "whodb_platform_shared_with_me":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformAccessInput) (*mcp.CallToolResult, any, error) {
			return handlePlatformExtendedQuery(ctx, "platform_shared_with_me", "SharedWithMe", map[string]any{}, input.Fields)
		})
	case "whodb_platform_shared_by_me":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformAccessInput) (*mcp.CallToolResult, any, error) {
			return handlePlatformExtendedQuery(ctx, "platform_shared_by_me", "SharedByMe", map[string]any{}, input.Fields)
		})
	case "whodb_platform_my_grants":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformAccessInput) (*mcp.CallToolResult, any, error) {
			return handlePlatformExtendedQuery(ctx, "platform_my_grants", "MyGrants", map[string]any{}, input.Fields)
		})
	case "whodb_platform_teams":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformAccessInput) (*mcp.CallToolResult, any, error) {
			return handlePlatformExtendedQuery(ctx, "platform_teams", "Teams", map[string]any{}, input.Fields)
		})
	case "whodb_platform_team_members":
		mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformAccessInput) (*mcp.CallToolResult, any, error) {
			return handlePlatformExtendedQuery(ctx, "platform_team_members", "TeamMembers", map[string]any{"teamId": strings.TrimSpace(input.TeamID)}, input.Fields)
		})
	default:
		return false
	}
	return true
}

func platformExtendedReadToolDefinitions() []*mcp.Tool {
	read := func(name, description string) *mcp.Tool {
		return &mcp.Tool{Name: name, Description: description, Annotations: platformReadOnlyAnnotations(description)}
	}
	return []*mcp.Tool{
		read("whodb_platform_apps", "List hosted ontology-powered apps in the selected project."),
		read("whodb_platform_app", "Inspect one hosted app, including its generated definition."),
		read("whodb_platform_app_files", "List the files belonging to one hosted app."),
		read("whodb_platform_app_view", "Read the current hosted app view and generated files."),
		read("whodb_platform_app_version_view", "Read a promoted hosted app version."),
		read("whodb_platform_packages", "List hosted package versions available in the selected project."),
		read("whodb_platform_package", "Inspect one hosted package and its requirements."),
		read("whodb_platform_package_installations", "List hosted package installations in the selected project."),
		read("whodb_platform_package_installation_update", "Check whether a package installation has an update."),
		read("whodb_platform_package_library", "Search packages shared with the organization or available through the package library."),
		read("whodb_platform_shared_package", "Inspect a package shared through a hosted package share token."),
		read("whodb_platform_preview_create_package", "Preview package contents and validation issues before creating a package."),
		read("whodb_platform_preview_install_package", "Preview package installation contents, requirements, and conflicts."),
		read("whodb_platform_preview_import_package", "Preview package import contents, requirements, and conflicts."),
		read("whodb_platform_preview_shared_install", "Preview installation of a shared package."),
		read("whodb_platform_preview_shared_import", "Preview import of a shared package."),
		read("whodb_platform_object_versions", "List promoted versions for an app, function, transform, dataset, or ontology."),
		read("whodb_platform_active_version", "Read the active production version for one versioned object."),
		read("whodb_platform_project_active_versions", "List active production versions in the selected project."),
		read("whodb_platform_resource_access", "List subjects with access to a hosted resource."),
		read("whodb_platform_resource_permissions", "List the signed-in user's permissions for a hosted resource."),
		read("whodb_platform_resource_types_access", "List hosted resource ids the signed-in user can access for a resource type."),
		read("whodb_platform_deletion_impact", "Inspect hosted dependents, references, and shared resources before deletion."),
		read("whodb_platform_deleted_resources", "List restorable hosted resources visible to the signed-in user."),
		read("whodb_platform_project_access", "Read the selected project's resource access matrix."),
		read("whodb_platform_project_resources", "Summarize resources in the selected project for governance and deletion planning."),
		read("whodb_platform_org_resources", "List resources visible in the selected organization."),
		read("whodb_platform_org_members", "List members of the selected organization."),
		read("whodb_platform_org_domains", "List domains configured for the selected organization."),
		read("whodb_platform_org_sso", "List SSO providers configured for the selected organization without client secrets."),
		read("whodb_platform_shared_with_me", "List resources shared with the signed-in user."),
		read("whodb_platform_shared_by_me", "List resources shared by the signed-in user."),
		read("whodb_platform_my_grants", "List the signed-in user's hosted resource grants."),
		read("whodb_platform_teams", "List organization teams."),
		read("whodb_platform_team_members", "List members of one organization team."),
	}
}

func handlePlatformExtendedQuery(ctx context.Context, toolName, operation string, variables map[string]any, fields []string) (*mcp.CallToolResult, PlatformReadOutput, error) {
	if err := validatePlatformExtendedQuery(operation, variables); err != nil {
		return nil, PlatformReadOutput{Error: err.Error(), RequestID: generateRequestID(toolName)}, nil
	}
	if variables == nil {
		variables = make(map[string]any)
	}
	return platformProjectRead(ctx, toolName, fields, func(ctx context.Context, session *platformToolSession) (any, int, bool, error) {
		variables["projectId"] = session.Host.DefaultProjectID
		variables["orgId"] = session.Host.DefaultOrgID
		if input, ok := variables["input"].(map[string]any); ok {
			if operation == "PreviewCreatePackage" {
				input["projectId"] = session.Host.DefaultProjectID
			} else {
				input["targetProjectId"] = session.Host.DefaultProjectID
			}
		}
		data, err := session.Client.PlatformQuery(ctx, operation, variables)
		return data, platformQueryCount(data), false, err
	})
}

func validatePlatformExtendedQuery(operation string, variables map[string]any) error {
	for _, key := range []string{"id", "appId", "packageId", "installationId", "objectId", "teamId", "resourceId"} {
		if value, ok := variables[key].(string); ok && value == "" {
			return fmt.Errorf("%s is required", snakeCasePlatformKey(key))
		}
	}
	if operation == "ObjectVersions" || operation == "ActiveProdVersion" {
		if strings.TrimSpace(stringValue(variables["objectType"])) == "" {
			return fmt.Errorf("object_type is required")
		}
	}
	if operation == "WhatCanIAccess" && strings.TrimSpace(stringValue(variables["resourceType"])) == "" {
		return fmt.Errorf("resource_type is required")
	}
	return nil
}

func stringValue(value any) string {
	text, _ := value.(string)
	return text
}

func snakeCasePlatformKey(value string) string {
	var out []rune
	for i, char := range value {
		if i > 0 && char >= 'A' && char <= 'Z' {
			out = append(out, '_')
		}
		out = append(out, []rune(strings.ToLower(string(char)))...)
	}
	return string(out)
}

func platformQueryCount(data any) int {
	switch value := data.(type) {
	case []any:
		return len(value)
	case map[string]any:
		return 1
	default:
		return 0
	}
}

func optionalPlatformString(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return strings.TrimSpace(value)
}

var _ platformClient = (*platformapi.Client)(nil)
