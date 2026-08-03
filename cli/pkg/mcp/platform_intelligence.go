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

// PlatformResolveResourceInput resolves a human resource reference in the selected project.
type PlatformResolveResourceInput struct {
	Resource string `json:"resource" jsonschema:"Resource type such as source, dataset, ontology, transform, function, secret, or ai_provider"`
	Query    string `json:"query" jsonschema:"Resource id, exact name, or name fragment"`
}

// PlatformResolvedResource is one candidate returned by resource resolution.
type PlatformResolvedResource struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Type  string `json:"type"`
	Match string `json:"match"`
}

// PlatformResolveResourceOutput describes a deterministic or ambiguous resolution.
type PlatformResolveResourceOutput struct {
	PlatformSetupGuidance
	Resource       string                     `json:"resource,omitempty"`
	Query          string                     `json:"query,omitempty"`
	Resolved       *PlatformResolvedResource  `json:"resolved,omitempty"`
	Candidates     []PlatformResolvedResource `json:"candidates,omitempty"`
	Ambiguous      bool                       `json:"ambiguous,omitempty"`
	Error          string                     `json:"error,omitempty"`
	ErrorCode      string                     `json:"error_code,omitempty"`
	Retryable      bool                       `json:"retryable,omitempty"`
	SuggestedTools []string                   `json:"suggested_tools,omitempty"`
	RequestID      string                     `json:"request_id,omitempty"`
}

func registerPlatformResourceResolverTool(server *mcp.Server, tool *mcp.Tool) bool {
	if tool.Name != "whodb_platform_resolve_resource" {
		return false
	}
	mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input PlatformResolveResourceInput) (*mcp.CallToolResult, PlatformResolveResourceOutput, error) {
		return HandlePlatformResolveResource(ctx, req, input)
	})
	return true
}

func platformResolveResourceToolDefinition() *mcp.Tool {
	return &mcp.Tool{Name: "whodb_platform_resolve_resource", Description: "Resolve a hosted resource name or id to an exact project resource before reading, planning, or writing it.", Annotations: platformReadOnlyAnnotations("Resolve Hosted Resource")}
}

// HandlePlatformResolveResource resolves resource names without executing a write.
func HandlePlatformResolveResource(ctx context.Context, req *mcp.CallToolRequest, input PlatformResolveResourceInput) (*mcp.CallToolResult, PlatformResolveResourceOutput, error) {
	requestID := generateRequestID("platform_resolve_resource")
	resource := normalizePlatformWriteToken(input.Resource)
	query := strings.TrimSpace(input.Query)
	if resource == "" || query == "" {
		return nil, PlatformResolveResourceOutput{Error: "resource and query are required", ErrorCode: string(PlatformErrorValidation), RequestID: requestID}, nil
	}
	session, err := loadPlatformWorkspace(ctx)
	if err != nil {
		return nil, platformResolveResourceError(err, requestID), nil
	}
	candidates, err := loadPlatformResourceCandidates(ctx, session, resource)
	if err != nil {
		return nil, platformResolveResourceError(err, requestID), nil
	}
	matched := matchPlatformResources(candidates, query)
	output := PlatformResolveResourceOutput{Resource: resource, Query: query, Candidates: matched, RequestID: requestID}
	if len(matched) == 1 {
		output.Resolved = &matched[0]
		return nil, output, nil
	}
	if len(matched) > 1 {
		output.Ambiguous = true
		output.Error = "resource reference is ambiguous; use the returned id"
		output.ErrorCode = string(PlatformErrorConflict)
		output.SuggestedTools = []string{"whodb_platform_resolve_resource", "whodb_platform_workspace_map"}
		return nil, output, nil
	}
	output.Error = fmt.Sprintf("no %s resource matched %q", resource, query)
	output.ErrorCode = string(PlatformErrorNotFound)
	output.SuggestedTools = []string{"whodb_platform_workspace_map", "whodb_platform_resolve_resource"}
	return nil, output, nil
}

func platformResolveResourceError(err error, requestID string) PlatformResolveResourceOutput {
	code, retryable, tools := platformErrorFields(err)
	return PlatformResolveResourceOutput{Error: err.Error(), ErrorCode: code, Retryable: retryable, SuggestedTools: tools, RequestID: requestID}
}

func loadPlatformResourceCandidates(ctx context.Context, session *platformToolSession, resource string) ([]PlatformResolvedResource, error) {
	projectID := session.Host.DefaultProjectID
	result := make([]PlatformResolvedResource, 0)
	add := func(id, name string) {
		if strings.TrimSpace(id) != "" {
			result = append(result, PlatformResolvedResource{ID: id, Name: name, Type: resource})
		}
	}
	switch resource {
	case "source", "sources":
		items, err := session.Client.ProjectSources(ctx, session.Host.DefaultOrgID, projectID)
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			add(item.ID, item.Name)
		}
	case "secret", "secrets":
		items, err := session.Client.ProjectSecrets(ctx, projectID)
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			add(item.ID, item.Name)
		}
	case "ai_provider", "ai_providers", "provider", "providers":
		items, err := session.Client.AIProviders(ctx, projectID)
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			add(item.ID, item.Name)
		}
	case "ontology", "ontologies":
		items, err := session.Client.Ontologies(ctx, projectID)
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			add(item.ID, item.DisplayName)
		}
	case "dataset", "datasets":
		items, err := session.Client.Datasets(ctx, projectID)
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			add(item.ID, item.Name)
		}
	case "transform", "transforms":
		items, err := session.Client.Transforms(ctx, projectID)
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			add(item.ID, item.Name)
		}
	case "function", "functions":
		items, err := session.Client.Functions(ctx, projectID, []string{"id", "name"})
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			add(item.ID, item.Name)
		}
	default:
		return nil, fmt.Errorf("unsupported resource type %q", resource)
	}
	return result, nil
}

func matchPlatformResources(items []PlatformResolvedResource, query string) []PlatformResolvedResource {
	needle := strings.ToLower(strings.TrimSpace(query))
	exactID, exactName, partial := make([]PlatformResolvedResource, 0), make([]PlatformResolvedResource, 0), make([]PlatformResolvedResource, 0)
	for _, item := range items {
		id, name := strings.ToLower(item.ID), strings.ToLower(item.Name)
		switch {
		case id == needle:
			item.Match = "id"
			exactID = append(exactID, item)
		case name == needle:
			item.Match = "exact_name"
			exactName = append(exactName, item)
		case strings.Contains(name, needle):
			item.Match = "name_fragment"
			partial = append(partial, item)
		}
	}
	if len(exactID) > 0 {
		return exactID
	}
	if len(exactName) > 0 {
		return exactName
	}
	return partial
}

func validatePlatformManifestMutation(manifest *platformapi.PlatformManifest, mutation string) error {
	if manifest == nil || len(manifest.Operations) == 0 {
		return nil
	}
	return manifest.RequireOperation("Mutation", mutation, "platform write "+mutation)
}

func validatePlatformPayload(key string, payload map[string]any) error {
	shape, ok := platformapi.PayloadShapes[key]
	if !ok {
		return nil
	}
	for _, field := range shape.Fields {
		value, exists := payload[field.Name]
		// Dataset descriptions are historically optional in the typed CLI input even
		// though older hosted schemas published them as non-null. Keep that compatible
		// while enforcing the other published required fields.
		if field.Required && field.Name != "description" && (!exists || value == nil || (field.Type == "string" && strings.TrimSpace(fmt.Sprint(value)) == "")) {
			return fmt.Errorf("payload.%s is required", field.Name)
		}
		if exists && value != nil && !platformPayloadTypeMatches(field.Type, value) {
			return fmt.Errorf("payload.%s must be a %s", field.Name, field.Type)
		}
	}
	return nil
}

func platformPayloadTypeMatches(expected string, value any) bool {
	switch expected {
	case "string":
		_, ok := value.(string)
		return ok
	case "boolean", "bool":
		_, ok := value.(bool)
		return ok
	case "integer", "int":
		switch value.(type) {
		case int, int32, int64, float64:
			return true
		}
		return false
	case "array":
		switch value.(type) {
		case []any, []string:
			return true
		}
		return false
	case "object":
		_, ok := value.(map[string]any)
		return ok
	default:
		return true
	}
}

// PlatformWritePreflight summarizes checks made before a write is confirmed.
type PlatformWritePreflight struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
	Tool   string `json:"tool,omitempty"`
}

func platformWritePreflight(ctx context.Context, session *platformToolSession, spec platformapi.GenericWriteSpec, payload map[string]any, targetID string) []PlatformWritePreflight {
	checks := []PlatformWritePreflight{{Name: "operation_published", Status: "ready", Reason: "The hosted manifest publishes this mutation."}}
	if strings.TrimSpace(targetID) != "" {
		checks = append(checks, PlatformWritePreflight{Name: "target_reference", Status: "ready", Reason: "The write has an explicit target id."})
	}
	if spec.InjectProjectID || spec.Mode == platformapi.GenericWriteModeProjectID {
		status, reason := "ready", "The selected project will scope this operation."
		if strings.TrimSpace(session.Host.DefaultProjectID) == "" {
			status, reason = "blocked_by_workspace", "A project must be selected before this operation."
		}
		checks = append(checks, PlatformWritePreflight{Name: "workspace", Status: status, Reason: reason, Tool: "whodb_platform_use"})
	}
	if targetID != "" {
		permissions, err := session.Client.PlatformQuery(ctx, "MyPermissions", map[string]any{"resourceType": spec.Resource, "resourceId": targetID})
		if err != nil {
			checks = append(checks, PlatformWritePreflight{Name: "permission", Status: "unknown", Reason: "Permissions could not be read; the platform will enforce authorization."})
		} else if permissionListEmpty(permissions) {
			checks = append(checks, PlatformWritePreflight{Name: "permission", Status: "unknown", Reason: "No explicit permission names were returned; the platform remains the authority.", Tool: "whodb_platform_resource_permissions"})
		} else {
			checks = append(checks, PlatformWritePreflight{Name: "permission", Status: "available", Reason: "The signed-in user's permission metadata is available."})
		}
	}
	for _, key := range []string{"sourceId", "datasetId", "ontologyId", "functionId", "providerId", "secretId"} {
		if value, ok := payload[key].(string); ok && strings.TrimSpace(value) == "" {
			checks = append(checks, PlatformWritePreflight{Name: key, Status: "blocked_by_dependency", Reason: key + " was supplied but empty; resolve the dependency first.", Tool: "whodb_platform_resolve_resource"})
		}
	}
	return checks
}

func permissionListEmpty(value any) bool {
	switch typed := value.(type) {
	case nil:
		return true
	case []string:
		return len(typed) == 0
	case []any:
		return len(typed) == 0
	case map[string]any:
		for _, key := range []string{"permissions", "data", "items"} {
			if child, ok := typed[key]; ok {
				return permissionListEmpty(child)
			}
		}
	}
	return false
}
