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
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const mcpUsageInstructions = `WhoDB MCP works without installed skills or shell access.
If a relevant WhoDB skill is available, use it as supplementary guidance. No skill
installation is required. Live tool schemas and active settings are authoritative.

Start with whodb_mcp_setup(action="inspect") when available: it returns setup
choices, a settings schema, active configuration, and this usage guide. The same
active settings are available at whodb://mcp/configuration. tools/list tells you
which tools are actually enabled; modules and filters may hide tools mentioned in
examples. Do not treat a hidden tool as missing data or install a skill to enable it.

SETUP: Ask which modules (platform, database, or both), targets, write mode, and
restrictions the user wants. Call whodb_mcp_setup(action="preview", patch={...}).
Omitted settings are preserved. Show the exact proposed settings, including any
expanded access, and obtain user approval. Call action="apply" with the returned
confirmation_token and approved=true; do not send a new patch. A newer preview
replaces the previous one. Tokens expire after five minutes. Saved settings take
effect after restarting the MCP connection; keep the single whodb mcp serve entry.
Process flags still override saved settings. action="verify" checks the current
process's configured targets; run it again after reconnecting to check new settings.
When the setup tool is filtered out, ask the user to configure WhoDB locally.

TARGETING: Discover saved platform hosts with whodb_platform_hosts. List orgs with
workspace:{host}, then projects with workspace:{host,org}. Resolve the intended
target using whodb_platform_workspace_resolve. Pass workspace:{host,org,project}
on subsequent platform calls and check returned scope. This selects only that
call, including parallel UAT/production calls; never switch shared defaults to
route a task. Resolve ambiguous names before acting. Database-only tools instead
use connection: discover names with whodb_connections, then schemas, tables, and
columns before querying. Database-only connections can be remote. The two modules
can run together; platform workspace arguments do not select database connections.

WRITES: Read the active write mode and target restrictions. Explain the exact
operation and target before requesting approval. Confirm only the approved preview
with its matching token; platform confirmations remain bound to host, account,
org, and project. Never silently change targets or relax restrictions after errors.

RECOVERY: For platform login/workspace errors use whodb_platform_setup_status.
If authentication is needed, give the user the returned whodb login --host <url>
command for browser sign-in; an MCP-only agent need not execute it. For a missing
database connection, ask the user to add it with whodb setup. Never request
passwords, tokens, or connection strings in chat or settings patches. Read errors
and verification results and report what needs user action instead of claiming
setup succeeded. Shell access is optional; browser sign-in and credential entry
remain user actions.

WORKFLOWS: For platform projects, use workspace_summary, build_plan, or gap_analysis
first; whodb://platform/concepts and whodb://platform/schema explain the product
model and operation payloads. For databases, explore schema before queries, use
bounded results and specific columns, parameterize user values, and send one query
at a time. Use explain, diff, erd, audit, and suggestions when relevant. Never
invent a tool, table, field, relationship, or successful verification.`

type mcpSetupInput struct {
	Action            string         `json:"action" jsonschema:"inspect, preview, apply, or verify; begin with inspect"`
	Patch             map[string]any `json:"patch,omitempty" jsonschema:"Credential-free settings patch for preview only; discover schema with inspect"`
	ConfirmationToken string         `json:"confirmation_token,omitempty" jsonschema:"Token from the exact preview approved by the user; apply only"`
	Approved          bool           `json:"approved,omitempty" jsonschema:"Set true only after the user approves the exact configuration preview"`
}

func registerMCPSetupTool(server *mcp.Server, opts *ServerOptions) {
	filters := &ToolEnablement{EnabledTools: opts.EnabledTools, DisabledTools: opts.DisabledTools}
	if opts.SetupHandler == nil || !filters.isToolEnabled("whodb_mcp_setup") {
		return
	}
	var mu sync.Mutex
	var token string
	var patch, preview json.RawMessage
	var expires time.Time
	mcp.AddTool(server, &mcp.Tool{
		Name:        "whodb_mcp_setup",
		Description: "Start here for self-contained WhoDB MCP guidance and setup, without skills or shell access. inspect returns settings/schema/choices; preview validates a patch without saving; apply saves only a token-bound, user-approved preview; verify checks active targets. Restart after changes. Never send credentials.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: false, DestructiveHint: new(true), OpenWorldHint: new(true)},
	}, func(ctx context.Context, req *mcp.CallToolRequest, input mcpSetupInput) (*mcp.CallToolResult, map[string]any, error) {
		mu.Lock()
		defer mu.Unlock()
		out := map[string]any{"guidance": mcpUsageInstructions, "active": mcpConfiguration(opts)}
		fail := func(err error) (*mcp.CallToolResult, map[string]any, error) {
			out["error"] = err.Error()
			return &mcp.CallToolResult{IsError: true}, out, nil
		}
		if input.Action != "preview" && input.Patch != nil {
			return fail(fmt.Errorf("patch is accepted only for preview"))
		}
		if input.Action != "apply" && (input.Approved || input.ConfirmationToken != "") {
			return fail(fmt.Errorf("approval and confirmation_token are accepted only for apply"))
		}
		var result json.RawMessage
		var err error
		switch input.Action {
		case "inspect", "verify":
			result, err = opts.SetupHandler(ctx, input.Action, nil)
		case "preview":
			token, patch, preview = "", nil, nil
			if input.Patch == nil {
				return fail(fmt.Errorf("preview requires a settings patch"))
			}
			patch, err = json.Marshal(input.Patch)
			if err == nil {
				result, err = opts.SetupHandler(ctx, "validate", patch)
			}
			if err == nil {
				preview = bytes.Clone(result)
				token, expires = rand.Text(), time.Now().Add(5*time.Minute)
				out["confirmation_token"], out["expires_at"] = token, expires
			}
		case "apply":
			if !input.Approved || token == "" || input.ConfirmationToken != token || !time.Now().Before(expires) {
				return fail(fmt.Errorf("apply requires user approval and a valid token from the latest preview"))
			}
			token = ""
			result, err = opts.SetupHandler(ctx, "validate", patch)
			if err == nil && !bytes.Equal(result, preview) {
				return fail(fmt.Errorf("saved settings changed since preview; preview again and obtain approval"))
			}
			if err == nil {
				result, err = opts.SetupHandler(ctx, "apply", preview)
			}
		default:
			return fail(fmt.Errorf("action must be inspect, preview, apply, or verify"))
		}
		if len(result) > 0 {
			out["result"] = result
		}
		if err != nil {
			return fail(err)
		}
		return nil, out, nil
	})
}
