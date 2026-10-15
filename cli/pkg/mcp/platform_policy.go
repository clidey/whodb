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
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/clidey/whodb/cli/internal/config"
	platformapi "github.com/clidey/whodb/cli/internal/platform"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// PlatformPolicy restricts platform access for one MCP server. Workspace rules use canonical IDs.
type PlatformPolicy config.MCPPlatformPolicy

// LoadPlatformPolicy reads and validates an explicit platform access policy.
func LoadPlatformPolicy(path string) (*PlatformPolicy, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return nil, fmt.Errorf("platform policy must be a JSON object")
	}
	var p PlatformPolicy
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&p); err != nil {
		return nil, err
	}
	if err := dec.Decode(new(any)); err != io.EOF {
		return nil, fmt.Errorf("platform policy must contain one JSON object")
	}
	if err := ValidatePlatformPolicy(&p); err != nil {
		return nil, err
	}
	return &p, nil
}

// ValidatePlatformPolicy normalizes hosts and validates required workspace identifiers.
func ValidatePlatformPolicy(p *PlatformPolicy) error {
	if p == nil {
		return nil
	}
	for _, hosts := range [][]string{p.AllowedHosts, p.ReadOnlyHosts} {
		for i, host := range hosts {
			normalized, err := platformapi.NormalizeHost(host)
			if err != nil || host == "" {
				return fmt.Errorf("invalid policy host %q", host)
			}
			hosts[i] = normalized
		}
	}
	for _, targets := range [][]PlatformWorkspaceTarget{p.AllowedWorkspaces, p.ReadOnlyWorkspaces} {
		for i, target := range targets {
			if target.Host == "" || target.Org == "" || target.Project == "" {
				return fmt.Errorf("policy workspaces require host, org and project IDs")
			}
			host, err := platformapi.NormalizeHost(target.Host)
			if err != nil {
				return err
			}
			targets[i].Host = host
		}
	}
	return nil
}

type platformPolicyKey struct{}
type platformPolicyRequest struct {
	policy *PlatformPolicy
	write  bool
	tool   string
}

func platformPolicyFromContext(ctx context.Context) platformPolicyRequest {
	p, _ := ctx.Value(platformPolicyKey{}).(platformPolicyRequest)
	return p
}

func (p *PlatformPolicy) allowsHost(host string) bool {
	if p == nil {
		return true
	}
	if len(p.AllowedHosts) > 0 && !containsPolicyHost(p.AllowedHosts, host) {
		return false
	}
	if len(p.AllowedWorkspaces) > 0 {
		for _, w := range p.AllowedWorkspaces {
			if w.Host == host {
				return true
			}
		}
		return false
	}
	return true
}
func containsPolicyHost(hosts []string, host string) bool {
	for _, h := range hosts {
		if h == host {
			return true
		}
	}
	return false
}
func matchesPolicyWorkspace(targets []PlatformWorkspaceTarget, host, org, project string) bool {
	for _, w := range targets {
		if w.Host == host && w.Org == org && w.Project == project {
			return true
		}
	}
	return false
}
func checkPlatformPolicy(ctx context.Context, host, org, project string) error {
	r := platformPolicyFromContext(ctx)
	p := r.policy
	if p == nil {
		return nil
	}
	if !p.allowsHost(host) {
		return fmt.Errorf("platform policy denies host %s", host)
	}
	discovery := r.tool == "whodb_platform_orgs" || r.tool == "whodb_platform_projects"
	discoveryOrg := org
	if r.tool == "whodb_platform_orgs" {
		discoveryOrg = ""
	}
	if len(p.AllowedWorkspaces) > 0 && !matchesPolicyWorkspace(p.AllowedWorkspaces, host, org, project) && (!discovery || !p.allowsDiscovery(host, discoveryOrg)) {
		return fmt.Errorf("platform policy denies workspace; pass an allowed workspace with host, org and project")
	}
	if r.write && (containsPolicyHost(p.ReadOnlyHosts, host) || matchesPolicyWorkspace(p.ReadOnlyWorkspaces, host, org, project)) {
		return fmt.Errorf("platform policy makes this workspace read-only")
	}
	return nil
}

func platformPolicyMiddleware(p *PlatformPolicy) mcp.Middleware {
	writes := map[string]bool{}
	for _, tool := range platformToolDefinitions() {
		writes[tool.Name] = tool.Annotations == nil || !tool.Annotations.ReadOnlyHint
	}
	return func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			write := false
			name := ""
			if call, ok := req.(*mcp.CallToolRequest); ok {
				name = call.Params.Name
				write = writes[name]
			}
			return next(context.WithValue(ctx, platformPolicyKey{}, platformPolicyRequest{policy: p, write: write, tool: name}), method, req)
		}
	}
}

// ValidatePlatformTools rejects unknown platform tool selectors before starting the server.
func ValidatePlatformTools(enabled, disabled []string) error {
	known := map[string]bool{}
	for _, t := range platformToolDefinitions() {
		known[t.Name] = true
	}
	for _, list := range [][]string{enabled, disabled} {
		for _, name := range list {
			if !known[name] {
				return fmt.Errorf("unknown platform tool %q; use full whodb_platform_* names", name)
			}
		}
	}
	return nil
}

func (p *PlatformPolicy) allowsDiscovery(host, org string) bool {
	if p == nil || len(p.AllowedWorkspaces) == 0 {
		return true
	}
	for _, w := range p.AllowedWorkspaces {
		if w.Host == host && (org == "" || w.Org == org) {
			return true
		}
	}
	return false
}
