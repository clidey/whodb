/*
 * Copyright 2026 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 */

package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/clidey/whodb/cli/internal/platform"
	"github.com/clidey/whodb/cli/pkg/output"
	"github.com/spf13/cobra"
)

type platformPackageUpdateChange struct {
	Action           string  `json:"action"`
	PruneAction      string  `json:"pruneAction"`
	SourceObjectID   string  `json:"sourceObjectId"`
	SourceObjectType string  `json:"sourceObjectType"`
	SourceVersion    int     `json:"sourceVersion"`
	TargetObjectID   *string `json:"targetObjectId,omitempty"`
	TargetObjectType *string `json:"targetObjectType,omitempty"`
}

type platformPackageUpdateInfo struct {
	Available        bool                          `json:"available"`
	SourceAvailable  bool                          `json:"sourceAvailable"`
	CurrentPackageID string                        `json:"currentPackageId"`
	CurrentVersion   string                        `json:"currentVersion"`
	LatestPackageID  *string                       `json:"latestPackageId,omitempty"`
	LatestVersion    *string                       `json:"latestVersion,omitempty"`
	Channel          string                        `json:"channel"`
	Changes          []platformPackageUpdateChange `json:"changes"`
}

type platformPackageInstallation struct {
	ID             string                    `json:"id"`
	PackageName    string                    `json:"packageName"`
	PackageVersion string                    `json:"packageVersion"`
	PackageChannel string                    `json:"packageChannel"`
	Status         string                    `json:"status"`
	UpdatedAt      string                    `json:"updatedAt"`
	UpdateInfo     platformPackageUpdateInfo `json:"updateInfo"`
}

var (
	packagesCmd           = &cobra.Command{Use: "packages", Short: "Install and safely update hosted WhoDB packages"}
	packageUpdateBindings []string
	packageUpdatePrune    bool
	packageUpdateApply    bool
)

var packagesListCmd = platformProjectListCommand("list", "List package installations in a project", func(ctx context.Context, session *platformSession, project *platform.Project) (any, *output.QueryResult, error) {
	data, err := session.Client.PlatformQuery(ctx, "PackageInstallations", map[string]any{"projectId": project.ID})
	if err != nil {
		return nil, nil, err
	}
	var installations []platformPackageInstallation
	if err := remarshalPlatformData(data, &installations); err != nil {
		return nil, nil, fmt.Errorf("decode package installations: %w", err)
	}
	rows := make([][]any, len(installations))
	for index, installation := range installations {
		latest := ""
		if installation.UpdateInfo.LatestVersion != nil {
			latest = *installation.UpdateInfo.LatestVersion
		}
		rows[index] = []any{installation.ID, installation.PackageName, installation.PackageVersion, latest, installation.PackageChannel, installation.Status, installation.UpdateInfo.Available, installation.UpdatedAt}
	}
	return installations, tableResult([]string{"id", "name", "version", "latest", "channel", "status", "update_available", "updated_at"}, rows), nil
})

var packagesUpdateCmd = &cobra.Command{
	Use:           "update <installation>",
	Short:         "Preview or apply a package installation update",
	Args:          cobra.ExactArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runPlatformPackageUpdate,
}

func registerPlatformPackageCommands() {
	packagesCmd.PersistentFlags().StringVar(&platformResourceOrg, "org", "", "organization id, slug, or name (defaults to selected organization)")
	packagesCmd.PersistentFlags().StringVar(&platformResourceProject, "project", "", "project id, slug, or name (defaults to selected project)")
	packagesUpdateCmd.Flags().StringArrayVar(&packageUpdateBindings, "binding", nil, "customer binding as requirement-key=target-id; repeatable")
	packagesUpdateCmd.Flags().BoolVar(&packageUpdatePrune, "prune", false, "delete removed package-owned apps, functions, and transforms; customer data is detached and preserved")
	packagesUpdateCmd.Flags().BoolVar(&packageUpdateApply, "apply", false, "apply the displayed update plan")
	registerPlatformPackageLifecycleCommands()
	registerPlatformPackageCustomizationCommands()
	packagesCmd.AddCommand(packagesListCmd, packagesUpdateCmd, packageCustomizationsCmd)
}

func runPlatformPackageUpdate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	format, err := output.ParseFormat(platformFormat)
	if err != nil {
		return err
	}
	session, err := loadPlatformSession(ctx, platformHost)
	if err != nil {
		return err
	}
	org, project, err := resolvePlatformProject(ctx, session, platformResourceOrg, platformResourceProject)
	if err != nil {
		return err
	}
	session.Client.SetWorkspaceContext(org.ID, project.ID)

	planData, err := session.Client.PlatformQuery(ctx, "PackageInstallationUpdate", map[string]any{"projectId": project.ID, "installationId": args[0]})
	if err != nil {
		return err
	}
	var plan platformPackageUpdateInfo
	if err := remarshalPlatformData(planData, &plan); err != nil {
		return fmt.Errorf("decode package update plan: %w", err)
	}
	if !packageUpdateApply {
		return writePlatformPackageUpdatePlan(cmd, format, plan)
	}
	if !plan.SourceAvailable {
		return fmt.Errorf("the source package is no longer available")
	}
	bindings, err := parsePackageBindings(packageUpdateBindings)
	if err != nil {
		return err
	}
	input := map[string]any{
		"targetProjectId": project.ID,
		"installationId":  args[0],
		"pruneRemoved":    packageUpdatePrune,
	}
	if strings.TrimSpace(packageRollbackID) != "" {
		input["packageId"] = strings.TrimSpace(packageRollbackID)
	}
	if len(bindings) > 0 {
		input["bindings"] = bindings
	}
	result, err := session.Client.PlatformMutation(ctx, "UpdatePackageInstallation", map[string]any{"input": input})
	if err != nil {
		return err
	}
	var applied any
	if err := json.Unmarshal(result.Data, &applied); err != nil {
		return fmt.Errorf("decode package update result: %w", err)
	}
	if format == output.FormatJSON {
		return writeAutomationEnvelope(cmd, "packages.update", applied)
	}
	newCommandOutput(cmd, format, platformQuiet).Success("Updated package installation %s", args[0])
	return nil
}

func parsePackageBindings(values []string) ([]map[string]any, error) {
	bindings := make([]map[string]any, 0, len(values))
	for _, value := range values {
		key, targetID, found := strings.Cut(value, "=")
		key = strings.TrimSpace(key)
		targetID = strings.TrimSpace(targetID)
		if !found || key == "" || targetID == "" {
			return nil, fmt.Errorf("invalid binding %q: expected requirement-key=target-id", value)
		}
		bindings = append(bindings, map[string]any{"key": key, "targetId": targetID})
	}
	return bindings, nil
}

func writePlatformPackageUpdatePlan(cmd *cobra.Command, format output.Format, plan platformPackageUpdateInfo) error {
	if format == output.FormatJSON {
		return writeAutomationEnvelope(cmd, "packages.update.plan", plan)
	}
	rows := make([][]any, len(plan.Changes))
	for index, change := range plan.Changes {
		targetID := ""
		if change.TargetObjectID != nil {
			targetID = *change.TargetObjectID
		}
		rows[index] = []any{change.Action, change.SourceObjectType, change.SourceObjectID, change.SourceVersion, targetID, change.PruneAction}
	}
	return newCommandOutput(cmd, format, platformQuiet).WriteQueryResult(tableResult([]string{"action", "type", "source_id", "version", "target_id", "on_prune"}, rows))
}
