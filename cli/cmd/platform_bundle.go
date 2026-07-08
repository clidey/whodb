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

package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/clidey/whodb/cli/internal/platform"
	"github.com/clidey/whodb/cli/pkg/output"
	"github.com/spf13/cobra"
)

var (
	platformBundlePath         string
	platformImportDryRun       bool
	platformBundlePrefix       string
	platformRenameConflicts    bool
	platformOverwriteConflicts bool
	platformBundleIncludeFiles bool
	platformBundleMaxFileBytes int
	platformBundleToOrg        string
	platformBundleToProject    string
)

var resourcesExportCmd = &cobra.Command{
	Use:           "export",
	Short:         "Export hosted WhoDB project metadata as a portable bundle",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPlatformBundleExport(cmd)
	},
}

var resourcesDiffCmd = &cobra.Command{
	Use:           "diff",
	Short:         "Show what a hosted WhoDB project bundle import would change",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPlatformBundlePlan(cmd, true)
	},
}

var resourcesImportCmd = &cobra.Command{
	Use:           "import",
	Short:         "Import hosted WhoDB project metadata from a portable bundle",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPlatformBundlePlan(cmd, platformImportDryRun)
	},
}

var backupProjectCmd = &cobra.Command{
	Use:           "backup-project",
	Short:         "Back up the selected hosted WhoDB project as a portable bundle",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPlatformBundleExport(cmd)
	},
}

var restoreProjectCmd = &cobra.Command{
	Use:           "restore-project",
	Short:         "Restore a hosted WhoDB project from a portable bundle",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPlatformBundlePlan(cmd, platformImportDryRun)
	},
}

var cloneProjectCmd = &cobra.Command{
	Use:           "clone-project",
	Short:         "Clone the selected hosted WhoDB project bundle into an existing target project",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPlatformProjectClone(cmd)
	},
}

func runPlatformBundleExport(cmd *cobra.Command) error {
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
	session.Host.DefaultOrgID = org.ID
	session.Host.DefaultOrgName = org.Name
	session.Host.DefaultProjectID = project.ID
	session.Host.DefaultProjectName = project.Name
	session.Client.SetWorkspaceContext(org.ID, project.ID)
	bundle, err := buildPlatformProjectBundle(ctx, session, project)
	if err != nil {
		return err
	}
	raw, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	if strings.TrimSpace(platformExportOutPath) == "" {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	if err := os.WriteFile(filepath.Clean(platformExportOutPath), raw, 0600); err != nil {
		return err
	}
	if format == output.FormatJSON {
		return writeCommandJSON(cmd, map[string]any{"out": filepath.Clean(platformExportOutPath), "bundle": bundle})
	}
	return newCommandOutput(cmd, format, platformQuiet).WriteQueryResult(platformBundleSummaryTable(bundle))
}

func buildPlatformProjectBundle(ctx context.Context, session *platformSession, project *platform.Project) (*platform.ProjectBundle, error) {
	return platform.BuildProjectBundleWithOptions(ctx, session.Client, session.Host.URL, session.Host.DefaultOrgID, session.Host.DefaultOrgName, project, platform.BundleExportOptions{
		IncludeFiles: platformBundleIncludeFiles,
		MaxFileBytes: platformBundleMaxFileBytes,
	})
}

func runPlatformBundlePlan(cmd *cobra.Command, dryRun bool) error {
	if strings.TrimSpace(platformBundlePath) == "" {
		return fmt.Errorf("--file is required")
	}
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
	session.Host.DefaultOrgID = org.ID
	session.Host.DefaultOrgName = org.Name
	session.Host.DefaultProjectID = project.ID
	session.Host.DefaultProjectName = project.Name
	session.Client.SetWorkspaceContext(org.ID, project.ID)
	bundle, err := readPlatformProjectBundle(platformBundlePath)
	if err != nil {
		return err
	}
	plan, err := planPlatformBundleImport(ctx, session, project, bundle, platform.BundleImportOptions{
		DryRun:             dryRun,
		Prefix:             platformBundlePrefix,
		RenameConflicts:    platformRenameConflicts,
		OverwriteConflicts: platformOverwriteConflicts,
		Getenv:             os.Getenv,
	})
	if err != nil {
		return err
	}
	if !dryRun && !platformWriteYes {
		return fmt.Errorf("import requires --yes; run resources diff --file %s first to review the plan", platformBundlePath)
	}
	if !dryRun {
		executePlatformBundlePlanCLI(ctx, session, project.ID, plan)
	}
	if format == output.FormatJSON {
		return writeCommandJSON(cmd, plan)
	}
	return newCommandOutput(cmd, format, platformQuiet).WriteQueryResult(platformBundlePlanTable(plan))
}

func readPlatformProjectBundle(path string) (*platform.ProjectBundle, error) {
	raw, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, fmt.Errorf("read bundle: %w", err)
	}
	var bundle platform.ProjectBundle
	if err := json.Unmarshal(raw, &bundle); err != nil {
		return nil, fmt.Errorf("decode bundle: %w", err)
	}
	if bundle.BundleVersion != 1 {
		return nil, fmt.Errorf("unsupported bundle version %d", bundle.BundleVersion)
	}
	return &bundle, nil
}

func planPlatformBundleImport(ctx context.Context, session *platformSession, project *platform.Project, bundle *platform.ProjectBundle, options platform.BundleImportOptions) (*platform.BundlePlan, error) {
	return platform.PlanBundleImportWithOptions(ctx, session.Client, session.Host.URL, project, bundle, options)
}

func runPlatformProjectClone(cmd *cobra.Command) error {
	if strings.TrimSpace(platformBundleToProject) == "" {
		return fmt.Errorf("--to-project is required")
	}
	ctx := context.Background()
	format, err := output.ParseFormat(platformFormat)
	if err != nil {
		return err
	}
	session, err := loadPlatformSession(ctx, platformHost)
	if err != nil {
		return err
	}
	sourceOrg, sourceProject, err := resolvePlatformProject(ctx, session, platformResourceOrg, platformResourceProject)
	if err != nil {
		return err
	}
	session.Host.DefaultOrgID = sourceOrg.ID
	session.Host.DefaultOrgName = sourceOrg.Name
	session.Host.DefaultProjectID = sourceProject.ID
	session.Host.DefaultProjectName = sourceProject.Name
	session.Client.SetWorkspaceContext(sourceOrg.ID, sourceProject.ID)
	bundle, err := buildPlatformProjectBundle(ctx, session, sourceProject)
	if err != nil {
		return err
	}
	targetOrg, targetProject, err := resolvePlatformProject(ctx, session, platformBundleToOrg, platformBundleToProject)
	if err != nil {
		return err
	}
	session.Host.DefaultOrgID = targetOrg.ID
	session.Host.DefaultOrgName = targetOrg.Name
	session.Host.DefaultProjectID = targetProject.ID
	session.Host.DefaultProjectName = targetProject.Name
	session.Client.SetWorkspaceContext(targetOrg.ID, targetProject.ID)
	plan, err := planPlatformBundleImport(ctx, session, targetProject, bundle, platform.BundleImportOptions{
		DryRun:             !platformWriteYes,
		Prefix:             platformBundlePrefix,
		RenameConflicts:    platformRenameConflicts,
		OverwriteConflicts: platformOverwriteConflicts,
		Getenv:             os.Getenv,
	})
	if err != nil {
		return err
	}
	if !platformWriteYes {
		if format == output.FormatJSON {
			return writeCommandJSON(cmd, plan)
		}
		return newCommandOutput(cmd, format, platformQuiet).WriteQueryResult(platformBundlePlanTable(plan))
	}
	executePlatformBundlePlanCLI(ctx, session, targetProject.ID, plan)
	if format == output.FormatJSON {
		return writeCommandJSON(cmd, plan)
	}
	return newCommandOutput(cmd, format, platformQuiet).WriteQueryResult(platformBundlePlanTable(plan))
}

func executePlatformBundlePlanCLI(ctx context.Context, session *platformSession, projectID string, plan *platform.BundlePlan) {
	for i := range plan.Actions {
		action := &plan.Actions[i]
		if action.Action != "create" && action.Action != "update" {
			continue
		}
		platform.ApplyBundleDependencyMap(action, bundleDependenciesFromPlan(plan))
		if action.Resource == "file" {
			file, err := uploadBundleFileAction(ctx, session, projectID, action)
			if err != nil {
				action.Action = "failed"
				action.Reason = err.Error()
				continue
			}
			action.TargetID = file.ID
			action.Action = "created"
			continue
		}
		spec, variables, err := buildGenericResourceVariables(projectID, genericResourceWriteInput{Resource: action.Resource, Action: action.Action, ID: action.TargetID}, platform.BundleMutationPayload(action))
		if err != nil {
			action.Action = "failed"
			action.Reason = err.Error()
			continue
		}
		result, err := session.Client.PlatformMutation(ctx, spec.Mutation, variables)
		if err != nil {
			action.Action = "failed"
			action.Reason = err.Error()
			continue
		}
		if id := platformMutationResultID(result); id != "" {
			action.TargetID = id
		}
		if action.Action == "create" {
			action.Action = "created"
		} else {
			action.Action = "updated"
		}
	}
}

func platformBundlePlanTable(plan *platform.BundlePlan) *output.QueryResult {
	rows := make([][]any, len(plan.Actions))
	for i, action := range plan.Actions {
		rows[i] = []any{action.Resource, action.Name, action.Action, action.Reason, strings.Join(action.Impacts, "; ")}
	}
	return tableResult([]string{"resource", "name", "action", "reason", "impacts"}, rows)
}

func platformBundleSummaryTable(bundle *platform.ProjectBundle) *output.QueryResult {
	return tableResult([]string{"resource", "count"}, [][]any{
		{"secrets", len(bundle.Secrets)},
		{"ai_providers", len(bundle.AIProviders)},
		{"datasets", len(bundle.Datasets)},
		{"ontologies", len(bundle.Ontologies)},
		{"transforms", len(bundle.Transforms)},
		{"functions", len(bundle.Functions)},
		{"folders", len(bundle.Folders)},
		{"files", len(bundle.Files)},
	})
}

func importSecretEnvName(name string) string {
	return platform.ImportSecretEnvName(name)
}

func datasetCreatePayloadFromExport(dataset platform.Dataset) map[string]any {
	return platform.DatasetCreatePayloadFromExport(dataset)
}

func functionCreatePayloadFromExport(fn platform.Function, keepProjectReferences bool) map[string]any {
	return platform.FunctionCreatePayloadFromExport(fn, keepProjectReferences)
}

func bundleDependenciesFromPlan(plan *platform.BundlePlan) platform.BundleDependencyMap {
	dependencies := platform.BundleDependencyMap{}
	for _, action := range plan.Actions {
		platform.AddBundleDependencyMapping(dependencies, action.SourceID, action.TargetID)
	}
	return dependencies
}

func platformMutationResultID(result *platform.PlatformMutationResult) string {
	if result == nil || len(result.Data) == 0 {
		return ""
	}
	var payload map[string]any
	if err := json.Unmarshal(result.Data, &payload); err != nil {
		return ""
	}
	if id, _ := payload["id"].(string); id != "" {
		return id
	}
	return ""
}

func uploadBundleFileAction(ctx context.Context, session *platformSession, projectID string, action *platform.BundleAction) (*platform.ProjectFile, error) {
	content, _ := action.Payload["content"].(string)
	if strings.TrimSpace(action.Name) == "" {
		return nil, fmt.Errorf("file name is required")
	}
	tmpDir, err := os.MkdirTemp("", "whodb-bundle-file-*")
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
		return session.Client.UploadProjectFile(ctx, projectID, nil, path)
	}
	return session.Client.UploadProjectFile(ctx, projectID, &folderID, path)
}
