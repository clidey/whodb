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
	platformBundlePath   string
	platformImportDryRun bool
)

var resourcesExportCmd = &cobra.Command{
	Use:           "export",
	Short:         "Export hosted WhoDB project metadata as a portable bundle",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
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

func buildPlatformProjectBundle(ctx context.Context, session *platformSession, project *platform.Project) (*platform.ProjectBundle, error) {
	return platform.BuildProjectBundle(ctx, session.Client, session.Host.URL, session.Host.DefaultOrgID, session.Host.DefaultOrgName, project)
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
	plan, err := planPlatformBundleImport(ctx, session, project, bundle, dryRun)
	if err != nil {
		return err
	}
	if !dryRun && !platformWriteYes {
		return fmt.Errorf("import requires --yes; run resources diff --file %s first to review the plan", platformBundlePath)
	}
	if !dryRun {
		for i := range plan.Actions {
			action := &plan.Actions[i]
			if action.Action != "create" {
				continue
			}
			spec, variables, err := buildGenericResourceVariables(project.ID, genericResourceWriteInput{Resource: action.Resource, Action: "create"}, action.Payload)
			if err != nil {
				action.Action = "failed"
				action.Reason = err.Error()
				continue
			}
			if _, err := session.Client.PlatformMutation(ctx, spec.Mutation, variables); err != nil {
				action.Action = "failed"
				action.Reason = err.Error()
				continue
			}
			action.Action = "created"
		}
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

func planPlatformBundleImport(ctx context.Context, session *platformSession, project *platform.Project, bundle *platform.ProjectBundle, dryRun bool) (*platform.BundlePlan, error) {
	return platform.PlanBundleImport(ctx, session.Client, session.Host.URL, project, bundle, dryRun, os.Getenv)
}

func platformBundlePlanTable(plan *platform.BundlePlan) *output.QueryResult {
	rows := make([][]any, len(plan.Actions))
	for i, action := range plan.Actions {
		rows[i] = []any{action.Resource, action.Name, action.Action, action.Reason}
	}
	return tableResult([]string{"resource", "name", "action", "reason"}, rows)
}

func platformBundleSummaryTable(bundle *platform.ProjectBundle) *output.QueryResult {
	return tableResult([]string{"resource", "count"}, [][]any{
		{"secrets", len(bundle.Secrets)},
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

func ontologyCreatePayloadFromExport(ontology platform.Ontology) map[string]any {
	return platform.OntologyCreatePayloadFromExport(ontology)
}

func transformCreatePayloadFromExport(transform platform.Transform) map[string]any {
	return platform.TransformCreatePayloadFromExport(transform)
}

func functionCreatePayloadFromExport(fn platform.Function, keepProjectReferences bool) map[string]any {
	return platform.FunctionCreatePayloadFromExport(fn, keepProjectReferences)
}
