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

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func registerMCPConfigurationResource(server *mcp.Server, opts *ServerOptions) {
	modules := []string{}
	if opts.PlatformEnabled {
		modules = append(modules, "platform")
	}
	if opts.DatabaseEnabled || !opts.PlatformEnabled {
		modules = append(modules, "database")
	}
	server.AddResource(&mcp.Resource{Name: "mcp-configuration", URI: "whodb://mcp/configuration", Description: "Active tool modules, access restrictions and setup instructions; contains no credentials", MIMEType: "application/json"}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		return jsonResource("whodb://mcp/configuration", map[string]any{
			"modules": modules, "read_only": opts.ReadOnly, "confirm_writes": opts.ConfirmWrites,
			"platform_policy": opts.PlatformPolicy, "allowed_connections": opts.AllowedConnections, "default_connection": opts.DefaultConnection,
			"enabled_tools": opts.EnabledTools, "disabled_tools": opts.DisabledTools,
			"setup_command": "whodb setup inspect", "restart_required_after_settings_change": true,
			"targeting": "Platform tools use workspace {host, org, project}; database-only tools use connection. Targets apply per call, without switching shared state.",
		})
	})
}
