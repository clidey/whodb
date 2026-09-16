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
	"os"
	"strings"

	"github.com/clidey/whodb/cli/internal/platform"
	"github.com/clidey/whodb/cli/pkg/output"
	"github.com/spf13/cobra"
)

type platformPackageCustomization struct {
	ID               string  `json:"id"`
	InstallationID   string  `json:"installationId"`
	SourceObjectID   string  `json:"sourceObjectId"`
	SourceObjectType string  `json:"sourceObjectType"`
	TargetObjectID   *string `json:"targetObjectId,omitempty"`
	Path             string  `json:"path"`
	Mode             string  `json:"mode"`
	Content          string  `json:"content"`
	UpdatedBy        string  `json:"updatedBy"`
	UpdatedAt        string  `json:"updatedAt"`
}

type platformPackageCustomizationPlan struct {
	InstallationID string `json:"installationId"`
	ObjectID       string `json:"objectId"`
	Path           string `json:"path"`
	Mode           string `json:"mode"`
	ContentBytes   int    `json:"contentBytes"`
	Action         string `json:"action"`
}

var (
	packageCustomizationsCmd     = &cobra.Command{Use: "customizations", Aliases: []string{"overlays"}, Short: "Manage customer-owned package overlays"}
	packageCustomizationObjectID string
	packageCustomizationPath     string
	packageCustomizationMode     string
	packageCustomizationContent  string
	packageCustomizationFile     string
	packageCustomizationApply    bool
)

var packageCustomizationsListCmd = &cobra.Command{
	Use:           "list <installation>",
	Short:         "List customer overlays for a package installation",
	Args:          cobra.ExactArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runPlatformPackageCustomizationsList,
}

var packageCustomizationsSetCmd = &cobra.Command{
	Use:           "set <installation>",
	Short:         "Preview or save a customer app-file overlay",
	Args:          cobra.ExactArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runPlatformPackageCustomizationSet,
}

var packageCustomizationsDeleteCmd = &cobra.Command{
	Use:           "delete <installation>",
	Short:         "Preview or remove a customer overlay and restore the package base",
	Args:          cobra.ExactArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runPlatformPackageCustomizationDelete,
}

func registerPlatformPackageCustomizationCommands() {
	for _, command := range []*cobra.Command{packageCustomizationsSetCmd, packageCustomizationsDeleteCmd} {
		command.Flags().StringVar(&packageCustomizationObjectID, "app", "", "source or installed app id")
		command.Flags().StringVar(&packageCustomizationPath, "path", "", "relative app file path, for example brand.json or src/customer.css")
		command.Flags().BoolVar(&packageCustomizationApply, "apply", false, "apply the displayed customization plan")
		_ = command.MarkFlagRequired("app")
		_ = command.MarkFlagRequired("path")
	}
	packageCustomizationsSetCmd.Flags().StringVar(&packageCustomizationMode, "mode", "replace", "overlay mode: replace, merge-json, or delete")
	packageCustomizationsSetCmd.Flags().StringVar(&packageCustomizationContent, "content", "", "inline overlay content")
	packageCustomizationsSetCmd.Flags().StringVar(&packageCustomizationFile, "file", "", "read overlay content from a local file")
	packageCustomizationsCmd.AddCommand(packageCustomizationsListCmd, packageCustomizationsSetCmd, packageCustomizationsDeleteCmd)
}

func runPlatformPackageCustomizationsList(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	format, session, project, err := platformPackageCommandContext(ctx)
	if err != nil {
		return err
	}
	data, err := session.Client.PlatformQuery(ctx, "PackageInstallationCustomizations", map[string]any{"projectId": project.ID, "installationId": args[0]})
	if err != nil {
		return err
	}
	var customizations []platformPackageCustomization
	if err := remarshalPlatformData(data, &customizations); err != nil {
		return fmt.Errorf("decode package customizations: %w", err)
	}
	if format == output.FormatJSON {
		return writeCommandJSON(cmd, customizations)
	}
	rows := make([][]any, len(customizations))
	for index, customization := range customizations {
		targetID := ""
		if customization.TargetObjectID != nil {
			targetID = *customization.TargetObjectID
		}
		rows[index] = []any{customization.SourceObjectID, targetID, customization.Path, customization.Mode, len(customization.Content), customization.UpdatedAt}
	}
	return newCommandOutput(cmd, format, platformQuiet).WriteQueryResult(tableResult([]string{"source_app", "installed_app", "path", "mode", "bytes", "updated_at"}, rows))
}

func runPlatformPackageCustomizationSet(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	format, session, project, err := platformPackageCommandContext(ctx)
	if err != nil {
		return err
	}
	content, err := packageCustomizationInputContent()
	if err != nil {
		return err
	}
	mode := normalizePlatformPackageCustomizationMode(packageCustomizationMode)
	plan := platformPackageCustomizationPlan{InstallationID: args[0], ObjectID: packageCustomizationObjectID, Path: packageCustomizationPath, Mode: mode, ContentBytes: len(content), Action: "upsert"}
	if !packageCustomizationApply {
		return writePlatformPackageCustomizationPlan(cmd, format, plan)
	}
	result, err := session.Client.PlatformMutation(ctx, "UpsertPackageInstallationCustomization", map[string]any{"input": map[string]any{
		"targetProjectId": project.ID,
		"installationId":  args[0],
		"objectId":        packageCustomizationObjectID,
		"path":            packageCustomizationPath,
		"mode":            mode,
		"content":         content,
	}})
	if err != nil {
		return err
	}
	var customization platformPackageCustomization
	if err := json.Unmarshal(result.Data, &customization); err != nil {
		return fmt.Errorf("decode package customization: %w", err)
	}
	if format == output.FormatJSON {
		return writeAutomationEnvelope(cmd, "packages.customizations.set", customization)
	}
	newCommandOutput(cmd, format, platformQuiet).Success("Saved %s overlay for %s", customization.Mode, customization.Path)
	return nil
}

func runPlatformPackageCustomizationDelete(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	format, session, project, err := platformPackageCommandContext(ctx)
	if err != nil {
		return err
	}
	plan := platformPackageCustomizationPlan{InstallationID: args[0], ObjectID: packageCustomizationObjectID, Path: packageCustomizationPath, Action: "delete"}
	if !packageCustomizationApply {
		return writePlatformPackageCustomizationPlan(cmd, format, plan)
	}
	_, err = session.Client.PlatformMutation(ctx, "DeletePackageInstallationCustomization", map[string]any{"input": map[string]any{
		"targetProjectId": project.ID,
		"installationId":  args[0],
		"objectId":        packageCustomizationObjectID,
		"path":            packageCustomizationPath,
	}})
	if err != nil {
		return err
	}
	if format == output.FormatJSON {
		return writeAutomationEnvelope(cmd, "packages.customizations.delete", plan)
	}
	newCommandOutput(cmd, format, platformQuiet).Success("Removed overlay for %s and restored the package base", packageCustomizationPath)
	return nil
}

func platformPackageCommandContext(ctx context.Context) (output.Format, *platformSession, *platform.Project, error) {
	format, err := output.ParseFormat(platformFormat)
	if err != nil {
		return format, nil, nil, err
	}
	session, err := loadPlatformSession(ctx, platformHost)
	if err != nil {
		return format, nil, nil, err
	}
	org, project, err := resolvePlatformProject(ctx, session, platformResourceOrg, platformResourceProject)
	if err != nil {
		return format, nil, nil, err
	}
	session.Client.SetWorkspaceContext(org.ID, project.ID)
	return format, session, project, nil
}

func packageCustomizationInputContent() (string, error) {
	if strings.TrimSpace(packageCustomizationFile) != "" && packageCustomizationContent != "" {
		return "", fmt.Errorf("use only one of --file or --content")
	}
	if strings.TrimSpace(packageCustomizationFile) == "" {
		return packageCustomizationContent, nil
	}
	raw, err := os.ReadFile(packageCustomizationFile)
	if err != nil {
		return "", fmt.Errorf("read customization file: %w", err)
	}
	return string(raw), nil
}

func normalizePlatformPackageCustomizationMode(value string) string {
	return strings.ReplaceAll(strings.ToLower(strings.TrimSpace(value)), "-", "_")
}

func writePlatformPackageCustomizationPlan(cmd *cobra.Command, format output.Format, plan platformPackageCustomizationPlan) error {
	if format == output.FormatJSON {
		return writeAutomationEnvelope(cmd, "packages.customizations.plan", plan)
	}
	return newCommandOutput(cmd, format, platformQuiet).WriteQueryResult(tableResult(
		[]string{"action", "installation", "app", "path", "mode", "bytes"},
		[][]any{{plan.Action, plan.InstallationID, plan.ObjectID, plan.Path, plan.Mode, plan.ContentBytes}},
	))
}
