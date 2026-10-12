/*
 * Copyright 2026 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 */

package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/clidey/whodb/cli/internal/config"
	platformapi "github.com/clidey/whodb/cli/internal/platform"
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// PlatformWorkspaceTarget selects a hosted workspace for one MCP call.
// A host alone selects that host's saved workspace; org clears any inherited project.
type PlatformWorkspaceTarget = config.MCPWorkspace

type platformRequestKey struct{}
type platformRequest struct {
	// Target is the explicit per-call workspace argument, if any.
	Target *PlatformWorkspaceTarget
	// Session is the MCP session default set by whodb_platform_session_use, if any.
	Session *PlatformWorkspaceTarget
	Scope   *PlatformOutputScope
}

// platformSessionScopes holds the workspace default chosen by
// whodb_platform_session_use for each connected MCP session. Entries live in
// memory only: they are never written to the CLI config and vanish when the
// session or process ends. Keyed by session so HTTP transports with several
// clients keep separate defaults.
var platformSessionScopes = struct {
	sync.Mutex
	m map[*mcp.ServerSession]*PlatformWorkspaceTarget
}{m: map[*mcp.ServerSession]*PlatformWorkspaceTarget{}}

func platformSessionScope(session *mcp.ServerSession) *PlatformWorkspaceTarget {
	if session == nil {
		return nil
	}
	platformSessionScopes.Lock()
	defer platformSessionScopes.Unlock()
	target, ok := platformSessionScopes.m[session]
	if !ok {
		return nil
	}
	copied := *target
	return &copied
}

func setPlatformSessionScope(session *mcp.ServerSession, target *PlatformWorkspaceTarget) {
	if session == nil {
		return
	}
	platformSessionScopes.Lock()
	defer platformSessionScopes.Unlock()
	if target == nil {
		delete(platformSessionScopes.m, session)
		return
	}
	copied := *target
	platformSessionScopes.m[session] = &copied
}

// platformRequestContext attaches the per-call target and the session default
// so scope resolution sees both.
func platformRequestContext(ctx context.Context, session *mcp.ServerSession, target *PlatformWorkspaceTarget) (context.Context, *platformRequest) {
	request := &platformRequest{Target: target, Session: platformSessionScope(session)}
	return context.WithValue(ctx, platformRequestKey{}, request), request
}

func platformRequestFromContext(ctx context.Context) *platformRequest {
	request, _ := ctx.Value(platformRequestKey{}).(*platformRequest)
	return request
}

func platformRequestScope(ctx context.Context) (platformapi.SessionScope, bool, error) {
	request := platformRequestFromContext(ctx)
	if runtime := platformRuntimeFromContext(ctx); runtime != nil {
		target := PlatformWorkspaceTarget{Host: runtime.Host}
		if request != nil && request.Target != nil {
			target = *request.Target
		} else if request != nil && request.Session != nil {
			target = *request.Session
		}
		if target.Host == "" {
			target.Host = runtime.Host
		}
		if target.Host != runtime.Host {
			return platformapi.SessionScope{}, true, fmt.Errorf("this server only serves %s", runtime.Host)
		}
		scope := platformapi.SessionScope{Host: target.Host, Org: target.Org, Project: target.Project}
		if scope.Project != "" && scope.Org == "" {
			return scope, true, fmt.Errorf("workspace.project requires workspace.org")
		}
		return scope, true, nil
	}
	if request == nil || request.Target == nil {
		scope := platformapi.SessionScopeFromEnvironment()
		if request != nil && request.Session != nil {
			// The session default was resolved and validated when it was set,
			// so it is applied as-is and never saved.
			return platformapi.SessionScope{Host: request.Session.Host, Org: request.Session.Org, Project: request.Session.Project}, false, nil
		}
		return scope, false, scope.Validate()
	}
	target := request.Target
	scope := platformapi.SessionScope{Host: strings.TrimSpace(target.Host), Org: strings.TrimSpace(target.Org), Project: strings.TrimSpace(target.Project)}
	if scope.Host == "" && scope.Org == "" && scope.Project == "" {
		return scope, true, fmt.Errorf("workspace must specify host or org")
	}
	if scope.Project != "" && scope.Org == "" {
		return scope, true, fmt.Errorf("workspace.project requires workspace.org")
	}
	if scope.Host == "" {
		scope.Host = platformapi.SessionScopeFromEnvironment().Host
	}
	return scope, true, nil
}

func recordPlatformScope(ctx context.Context, session *platformToolSession) {
	if request := platformRequestFromContext(ctx); request != nil {
		request.Scope = platformScope(session)
	}
}

// addPlatformTool adds the same per-call workspace contract to every hosted tool.
// The SDK validates the augmented schema before decoding the operation's own input.
func addPlatformTool[In, Out any](server *mcp.Server, tool *mcp.Tool, handler mcp.ToolHandlerFor[In, Out]) {
	inputSchema, err := jsonschema.For[In](nil)
	if err != nil {
		panic(fmt.Sprintf("%s input schema: %v", tool.Name, err))
	}
	targetSchema, err := jsonschema.For[PlatformWorkspaceTarget](nil)
	if err != nil {
		panic(err)
	}
	targetSchema.Description = "Optional target for this call only. Omit to use the session default set by whodb_platform_session_use, then process/saved defaults. Discover targets with whodb_platform_hosts, whodb_platform_orgs and whodb_platform_projects."
	if inputSchema.Properties == nil {
		inputSchema.Properties = map[string]*jsonschema.Schema{}
	}
	inputSchema.Properties["workspace"] = targetSchema
	outputSchema, err := jsonschema.For[Out](nil)
	if err != nil {
		panic(fmt.Sprintf("%s output schema: %v", tool.Name, err))
	}
	scopeSchema, err := jsonschema.For[PlatformOutputScope](nil)
	if err != nil {
		panic(err)
	}
	if outputSchema.Properties == nil {
		outputSchema.Properties = map[string]*jsonschema.Schema{}
	}
	outputSchema.Properties["scope"] = scopeSchema
	outputSchema.Properties["workspace"] = targetSchema
	scopedTool := *tool
	scopedTool.InputSchema = inputSchema
	if reflect.TypeFor[Out]() != reflect.TypeFor[any]() {
		scopedTool.OutputSchema = outputSchema
	}
	scopedTool.Description += " Accepts workspace {host, org, project} for per-call targeting without changing defaults."
	mcp.AddTool(server, &scopedTool, func(ctx context.Context, req *mcp.CallToolRequest, in In) (*mcp.CallToolResult, any, error) {
		var args struct {
			Workspace *PlatformWorkspaceTarget `json:"workspace"`
		}
		arguments := req.Params.Arguments
		if len(arguments) == 0 {
			arguments = json.RawMessage(`{}`)
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, nil, err
		}
		if platformPolicyFromContext(ctx).policy != nil {
			var selectors map[string]json.RawMessage
			if err := json.Unmarshal(arguments, &selectors); err != nil {
				return nil, nil, err
			}
			for _, key := range []string{"org", "project"} {
				if value, ok := selectors[key]; ok && string(value) != `""` && (args.Workspace == nil || (key == "org" && args.Workspace.Org == "") || (key == "project" && args.Workspace.Project == "")) {
					return nil, nil, fmt.Errorf("with a platform policy, include workspace.%s when supplying the top-level selector", key)
				}
			}
		}
		ctx, request := platformRequestContext(ctx, req.Session, args.Workspace)
		if _, _, err := platformRequestScope(ctx); err != nil {
			return nil, nil, err
		}
		result, out, err := handler(ctx, req, in)
		if err != nil {
			return result, nil, err
		}
		raw, err := json.Marshal(out)
		if err != nil {
			return nil, nil, err
		}
		var output map[string]any
		if err := json.Unmarshal(raw, &output); err != nil {
			return nil, nil, err
		}
		if request.Scope != nil {
			output["scope"] = request.Scope
		}
		if args.Workspace != nil {
			output["workspace"] = args.Workspace
		}
		return result, output, nil
	})
}

// PlatformHostInfo describes a saved login without exposing tokens or cached manifests.
type PlatformHostInfo struct {
	Host      string `json:"host"`
	AccountID string `json:"account_id,omitempty"`
	Email     string `json:"email,omitempty"`
	Default   bool   `json:"default"`
	OrgID     string `json:"org_id,omitempty"`
	ProjectID string `json:"project_id,omitempty"`
}

// PlatformHostsOutput lists locally saved hosts; login validity is checked on use.
type PlatformHostsOutput struct {
	Hosts []PlatformHostInfo `json:"hosts"`
}

func platformScopeToolDefinitions() []*mcp.Tool {
	return []*mcp.Tool{
		{Name: "whodb_platform_hosts", Description: "List saved platform hosts and account metadata without contacting servers. Saved credentials may have expired; status checks the selected login.", Annotations: platformReadOnlyAnnotations("List Saved Platform Hosts")},
		{Name: "whodb_platform_workspace_resolve", Description: "Resolve workspace {host, org, project} to canonical IDs and verify the login. Does not switch or save defaults.", Annotations: platformReadOnlyAnnotations("Resolve Platform Workspace")},
		{Name: "whodb_platform_session_use", Description: descPlatformSessionUse, Annotations: platformReadOnlyAnnotations("Set Session Default Workspace")},
		{Name: "whodb_platform_session_clear", Description: descPlatformSessionClear, Annotations: platformReadOnlyAnnotations("Clear Session Default Workspace")},
	}
}

const descPlatformSessionUse = `Set the default host, organization, and project for this MCP session only.

Pass workspace {host, org, project}. The login is verified and the target is resolved to canonical IDs; later calls that omit workspace use it. Nothing is written to the CLI config, so other chats, terminals, and MCP sessions are unaffected. Use this once at the start of a chat that should stay on one host (for example UAT or production), then call whodb_platform_status to confirm. An explicit workspace on a later call still overrides it for that call. Call whodb_platform_session_clear to return to process/saved defaults.`

const descPlatformSessionClear = `Clear the session default set by whodb_platform_session_use.

Later calls that omit workspace fall back to process overrides, then saved CLI defaults. Nothing is written to the CLI config.`

// PlatformSessionScopeInput is the input for whodb_platform_session_use and
// whodb_platform_session_clear; the target comes from the shared workspace argument.
type PlatformSessionScopeInput struct{}

// PlatformSessionScopeOutput reports the session default after a change.
type PlatformSessionScopeOutput struct {
	Status       string               `json:"status"`
	SessionScope *PlatformOutputScope `json:"session_scope,omitempty"`
	Error        string               `json:"error,omitempty"`
	RequestID    string               `json:"request_id,omitempty"`
}

func handlePlatformSessionUse(ctx context.Context, req *mcp.CallToolRequest, input PlatformSessionScopeInput) (*mcp.CallToolResult, PlatformSessionScopeOutput, error) {
	requestID := generateRequestID("platform_session_use")
	request := platformRequestFromContext(ctx)
	if request == nil || request.Target == nil {
		return nil, PlatformSessionScopeOutput{Status: "error", Error: "pass workspace {host, org, project} to select the session default", RequestID: requestID}, nil
	}
	session, err := loadPlatformToolSession(ctx)
	if err != nil {
		return nil, PlatformSessionScopeOutput{Status: "error", Error: err.Error(), RequestID: requestID}, nil
	}
	target := &PlatformWorkspaceTarget{Host: session.Host.URL, Org: session.Host.DefaultOrgID, Project: session.Host.DefaultProjectID}
	setPlatformSessionScope(req.Session, target)
	TrackToolCall(ctx, "platform_session_use", requestID, true, 0, map[string]any{"workspace_selected": hasPlatformWorkspace(session)})
	return nil, PlatformSessionScopeOutput{Status: "ok", SessionScope: platformScope(session), RequestID: requestID}, nil
}

func handlePlatformSessionClear(ctx context.Context, req *mcp.CallToolRequest, input PlatformSessionScopeInput) (*mcp.CallToolResult, PlatformSessionScopeOutput, error) {
	requestID := generateRequestID("platform_session_clear")
	setPlatformSessionScope(req.Session, nil)
	TrackToolCall(ctx, "platform_session_clear", requestID, true, 0, nil)
	return nil, PlatformSessionScopeOutput{Status: "cleared", RequestID: requestID}, nil
}

func handlePlatformHosts(ctx context.Context, req *mcp.CallToolRequest, input PlatformStatusInput) (*mcp.CallToolResult, PlatformHostsOutput, error) {
	if runtime := platformRuntimeFromContext(ctx); runtime != nil {
		return nil, PlatformHostsOutput{Hosts: []PlatformHostInfo{{Host: runtime.Host, AccountID: runtime.UserID, Email: runtime.Email, Default: true}}}, nil
	}
	cfg, err := config.LoadConfigWithoutSecrets()
	if err != nil {
		return nil, PlatformHostsOutput{}, err
	}
	hosts := make([]PlatformHostInfo, 0, len(cfg.Platform.Hosts))
	for _, host := range cfg.Platform.Hosts {
		if !platformPolicyFromContext(ctx).policy.allowsHost(host.URL) {
			continue
		}
		if policy := platformPolicyFromContext(ctx).policy; policy != nil && len(policy.AllowedWorkspaces) > 0 && !matchesPolicyWorkspace(policy.AllowedWorkspaces, host.URL, host.DefaultOrgID, host.DefaultProjectID) {
			host.DefaultOrgID, host.DefaultProjectID = "", ""
		}
		hosts = append(hosts, PlatformHostInfo{Host: host.URL, AccountID: host.AccountID, Email: host.Email, Default: host.URL == cfg.Platform.DefaultHost, OrgID: host.DefaultOrgID, ProjectID: host.DefaultProjectID})
	}
	return nil, PlatformHostsOutput{Hosts: hosts}, nil
}

func selectPlatformToolOrg(ctx context.Context, session *platformToolSession, org *platformapi.Organization) (*platformapi.Organization, error) {
	request := platformRequestFromContext(ctx)
	if request != nil && request.Target != nil && request.Target.Org != "" && session.Host.DefaultOrgID != "" && session.Host.DefaultOrgID != org.ID {
		return nil, fmt.Errorf("org conflicts with explicit workspace.org")
	}
	if session.Host.DefaultOrgID != org.ID {
		session.Host.DefaultProjectID, session.Host.DefaultProjectName = "", ""
	}
	session.Host.DefaultOrgID, session.Host.DefaultOrgName = org.ID, org.Name
	session.Client.SetWorkspaceContext(org.ID, session.Host.DefaultProjectID)
	recordPlatformScope(ctx, session)
	return org, nil
}

func selectPlatformToolProject(ctx context.Context, session *platformToolSession, org *platformapi.Organization, project *platformapi.Project) (*platformapi.Organization, *platformapi.Project, error) {
	request := platformRequestFromContext(ctx)
	if request != nil && request.Target != nil && request.Target.Project != "" && session.Host.DefaultProjectID != "" && session.Host.DefaultProjectID != project.ID {
		return nil, nil, fmt.Errorf("project conflicts with explicit workspace.project")
	}
	session.Host.DefaultProjectID, session.Host.DefaultProjectName = project.ID, project.Name
	session.Client.SetWorkspaceContext(org.ID, project.ID)
	recordPlatformScope(ctx, session)
	return org, project, nil
}
