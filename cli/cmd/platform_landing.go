/*
 * Copyright 2026 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 */

package cmd

import (
	"fmt"

	"github.com/clidey/whodb/cli/internal/config"
	platformapi "github.com/clidey/whodb/cli/internal/platform"
	"github.com/spf13/cobra"
)

func showPlatformLanding(cmd *cobra.Command) error {
	cfg, err := config.LoadConfigWithoutSecrets()
	if err != nil {
		return err
	}
	host, err := resolvePlatformHost(cfg, "")
	if err != nil {
		return err
	}
	scope := platformapi.SessionScopeFromEnvironment()
	if err := scope.Validate(); err != nil {
		return err
	}
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "WhoDB platform\nHost: %s\n", host)
	entry, ok := cfg.GetPlatformHost(host)
	if !ok || entry.AccountID == "" {
		fmt.Fprintf(out, "No saved login. Sign in with: whodb login --host %s\n", host)
	} else {
		fmt.Fprintf(out, "Saved account: %s\n", entry.Email)
		org, project := entry.DefaultOrgName, entry.DefaultProjectName
		if org == "" {
			org = entry.DefaultOrgID
		}
		if project == "" {
			project = entry.DefaultProjectID
		}
		if scope.HasWorkspace() {
			org, project = scope.Org, scope.Project
		}
		if org != "" && project != "" {
			fmt.Fprintf(out, "Workspace: %s / %s\n", org, project)
		} else {
			fmt.Fprintln(out, "Select a default: whodb use --org <org> --project <project>")
		}
		fmt.Fprintln(out, "Check live login and permissions: whodb status")
	}
	fmt.Fprintln(out, "\nDiscover workspaces: whodb orgs list; whodb projects list --org <org>\nConnect an agent:    whodb mcp serve\nDatabase terminal:  whodb --tui\nDatabase MCP:       whodb mcp serve --database")
	return nil
}
