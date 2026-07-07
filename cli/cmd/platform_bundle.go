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
	"regexp"
	"strings"
	"time"

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

type platformProjectBundle struct {
	BundleVersion int                      `json:"bundleVersion"`
	ExportedAt    string                   `json:"exportedAt"`
	Host          string                   `json:"host"`
	OrgID         string                   `json:"orgId"`
	OrgName       string                   `json:"orgName"`
	ProjectID     string                   `json:"projectId"`
	ProjectName   string                   `json:"projectName"`
	Secrets       []platform.ProjectSecret `json:"secrets"`
	Datasets      []platform.Dataset       `json:"datasets"`
	Ontologies    []platform.Ontology      `json:"ontologies"`
	Transforms    []platform.Transform     `json:"transforms"`
	Functions     []platform.Function      `json:"functions"`
	Folders       []platform.ProjectFolder `json:"folders"`
	Files         []platform.ProjectFile   `json:"files"`
	Notes         []string                 `json:"notes,omitempty"`
}

type platformBundleAction struct {
	Resource string         `json:"resource"`
	Name     string         `json:"name"`
	Action   string         `json:"action"`
	Reason   string         `json:"reason,omitempty"`
	Payload  map[string]any `json:"payload,omitempty"`
}

type platformBundlePlan struct {
	Host        string                 `json:"host"`
	ProjectID   string                 `json:"projectId"`
	ProjectName string                 `json:"projectName"`
	DryRun      bool                   `json:"dryRun"`
	Actions     []platformBundleAction `json:"actions"`
}

func buildPlatformProjectBundle(ctx context.Context, session *platformSession, project *platform.Project) (*platformProjectBundle, error) {
	secrets, err := session.Client.ProjectSecrets(ctx, project.ID)
	if err != nil {
		return nil, err
	}
	datasets, err := session.Client.Datasets(ctx, project.ID)
	if err != nil {
		return nil, err
	}
	ontologies, err := session.Client.Ontologies(ctx, project.ID)
	if err != nil {
		return nil, err
	}
	transforms, err := session.Client.Transforms(ctx, project.ID)
	if err != nil {
		return nil, err
	}
	functions, err := session.Client.Functions(ctx, project.ID, nil)
	if err != nil {
		return nil, err
	}
	tree, err := loadProjectFolderTree(ctx, session, project.ID)
	if err != nil {
		return nil, err
	}
	folders := make([]platform.ProjectFolder, 0)
	files := make([]platform.ProjectFile, 0)
	for _, entry := range tree {
		switch entry.Kind {
		case "folder":
			folders = append(folders, entry.Folder)
		case "file":
			files = append(files, entry.File)
		}
	}
	return &platformProjectBundle{
		BundleVersion: 1,
		ExportedAt:    time.Now().UTC().Format(time.RFC3339),
		Host:          session.Host.URL,
		OrgID:         session.Host.DefaultOrgID,
		OrgName:       session.Host.DefaultOrgName,
		ProjectID:     project.ID,
		ProjectName:   project.Name,
		Secrets:       secrets,
		Datasets:      datasets,
		Ontologies:    ontologies,
		Transforms:    transforms,
		Functions:     functions,
		Folders:       folders,
		Files:         files,
		Notes: []string{
			"Secret values are not exported. Import reads secret values from WHODB_IMPORT_SECRET_<SECRET_NAME> environment variables.",
			"Uploaded file bytes are not exported. File metadata is included for planning only.",
		},
	}, nil
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

func readPlatformProjectBundle(path string) (*platformProjectBundle, error) {
	raw, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, fmt.Errorf("read bundle: %w", err)
	}
	var bundle platformProjectBundle
	if err := json.Unmarshal(raw, &bundle); err != nil {
		return nil, fmt.Errorf("decode bundle: %w", err)
	}
	if bundle.BundleVersion != 1 {
		return nil, fmt.Errorf("unsupported bundle version %d", bundle.BundleVersion)
	}
	return &bundle, nil
}

func planPlatformBundleImport(ctx context.Context, session *platformSession, project *platform.Project, bundle *platformProjectBundle, dryRun bool) (*platformBundlePlan, error) {
	currentSecrets, err := session.Client.ProjectSecrets(ctx, project.ID)
	if err != nil {
		return nil, err
	}
	currentDatasets, err := session.Client.Datasets(ctx, project.ID)
	if err != nil {
		return nil, err
	}
	currentOntologies, err := session.Client.Ontologies(ctx, project.ID)
	if err != nil {
		return nil, err
	}
	currentTransforms, err := session.Client.Transforms(ctx, project.ID)
	if err != nil {
		return nil, err
	}
	currentFunctions, err := session.Client.Functions(ctx, project.ID, []string{"id", "name"})
	if err != nil {
		return nil, err
	}

	plan := &platformBundlePlan{Host: session.Host.URL, ProjectID: project.ID, ProjectName: project.Name, DryRun: dryRun}
	secretNames := namesBy(currentSecrets, func(secret platform.ProjectSecret) string { return secret.Name })
	datasetNames := namesBy(currentDatasets, func(dataset platform.Dataset) string { return dataset.Name })
	ontologyNames := namesBy(currentOntologies, func(ontology platform.Ontology) string { return ontology.APIName })
	transformNames := namesBy(currentTransforms, func(transform platform.Transform) string { return transform.Name })
	functionNames := namesBy(currentFunctions, func(fn platform.Function) string { return fn.Name })

	for _, secret := range bundle.Secrets {
		action := platformBundleAction{Resource: "secret", Name: secret.Name}
		envName := importSecretEnvName(secret.Name)
		if secretNames[secret.Name] {
			action.Action = "skip"
			action.Reason = "secret already exists"
		} else if os.Getenv(envName) == "" {
			action.Action = "skip"
			action.Reason = "missing " + envName
		} else {
			action.Action = "create"
			action.Payload = map[string]any{"name": secret.Name, "description": secret.Description, "value": os.Getenv(envName)}
		}
		plan.Actions = append(plan.Actions, action)
	}
	for _, dataset := range bundle.Datasets {
		action := platformBundleAction{Resource: "dataset", Name: dataset.Name}
		if datasetNames[dataset.Name] {
			action.Action = "skip"
			action.Reason = "dataset already exists"
		} else {
			action.Action = "create"
			action.Payload = datasetCreatePayloadFromExport(dataset)
		}
		plan.Actions = append(plan.Actions, action)
	}
	for _, ontology := range bundle.Ontologies {
		action := platformBundleAction{Resource: "ontology", Name: ontology.APIName}
		if ontologyNames[ontology.APIName] {
			action.Action = "skip"
			action.Reason = "ontology already exists"
		} else {
			action.Action = "create"
			action.Payload = ontologyCreatePayloadFromExport(ontology)
		}
		plan.Actions = append(plan.Actions, action)
	}
	for _, transform := range bundle.Transforms {
		action := platformBundleAction{Resource: "transform", Name: transform.Name}
		if transformNames[transform.Name] {
			action.Action = "skip"
			action.Reason = "transform already exists"
		} else {
			action.Action = "create"
			action.Payload = transformCreatePayloadFromExport(transform)
		}
		plan.Actions = append(plan.Actions, action)
	}
	for _, fn := range bundle.Functions {
		action := platformBundleAction{Resource: "function", Name: fn.Name}
		if functionNames[fn.Name] {
			action.Action = "skip"
			action.Reason = "function already exists"
		} else {
			action.Action = "create"
			action.Payload = functionCreatePayloadFromExport(fn, false)
		}
		plan.Actions = append(plan.Actions, action)
	}
	for _, folder := range bundle.Folders {
		plan.Actions = append(plan.Actions, platformBundleAction{Resource: "folder", Name: folder.Name, Action: "skip", Reason: "folder metadata import is not implemented in v1"})
	}
	for _, file := range bundle.Files {
		plan.Actions = append(plan.Actions, platformBundleAction{Resource: "file", Name: file.Name, Action: "skip", Reason: "file bytes are not included in bundles"})
	}
	return plan, nil
}

func platformBundlePlanTable(plan *platformBundlePlan) *output.QueryResult {
	rows := make([][]any, len(plan.Actions))
	for i, action := range plan.Actions {
		rows[i] = []any{action.Resource, action.Name, action.Action, action.Reason}
	}
	return tableResult([]string{"resource", "name", "action", "reason"}, rows)
}

func platformBundleSummaryTable(bundle *platformProjectBundle) *output.QueryResult {
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

func namesBy[T any](values []T, name func(T) string) map[string]bool {
	names := make(map[string]bool, len(values))
	for _, value := range values {
		if key := strings.TrimSpace(name(value)); key != "" {
			names[key] = true
		}
	}
	return names
}

func importSecretEnvName(name string) string {
	re := regexp.MustCompile(`[^A-Za-z0-9]+`)
	sanitized := strings.Trim(re.ReplaceAllString(strings.ToUpper(name), "_"), "_")
	if sanitized == "" {
		sanitized = "VALUE"
	}
	return "WHODB_IMPORT_SECRET_" + sanitized
}

func datasetCreatePayloadFromExport(dataset platform.Dataset) map[string]any {
	payload := map[string]any{"name": dataset.Name}
	if dataset.Description != "" {
		payload["description"] = dataset.Description
	}
	if dataset.SchemaMode != "" {
		payload["schemaMode"] = dataset.SchemaMode
	}
	if len(dataset.Schema) > 0 {
		columns := make([]map[string]any, len(dataset.Schema))
		for i, column := range dataset.Schema {
			columns[i] = map[string]any{
				"name":       column.Name,
				"type":       column.Type,
				"isNullable": column.IsNullable,
				"isPrimary":  column.IsPrimary,
			}
		}
		payload["columns"] = columns
	}
	return payload
}

func ontologyCreatePayloadFromExport(ontology platform.Ontology) map[string]any {
	properties := make([]map[string]any, len(ontology.Properties))
	for i, property := range ontology.Properties {
		properties[i] = map[string]any{
			"apiName":          property.APIName,
			"displayName":      property.DisplayName,
			"description":      property.Description,
			"columnName":       property.ColumnName,
			"dataType":         property.DataType,
			"arrayElementType": property.ArrayElementType,
			"isRequired":       property.IsRequired,
			"visibility":       property.Visibility,
			"isSearchable":     property.IsSearchable,
			"isSortable":       property.IsSortable,
			"isEditOnly":       property.IsEditOnly,
			"sortOrder":        property.SortOrder,
		}
	}
	links := make([]map[string]any, len(ontology.Links))
	for i, link := range ontology.Links {
		links[i] = map[string]any{
			"apiName":                  link.APIName,
			"targetOntologyApiName":    link.TargetOntologyAPIName,
			"cardinality":              link.Cardinality,
			"foreignKeyProperty":       link.ForeignKeyProperty,
			"targetForeignKeyProperty": link.TargetForeignKeyProperty,
			"joinTable":                link.JoinTable,
			"sourceColumnInJoinTable":  link.SourceColumnInJoinTable,
			"targetColumnInJoinTable":  link.TargetColumnInJoinTable,
			"displayName":              link.DisplayName,
			"pluralDisplayName":        link.PluralDisplayName,
			"reverseDisplayName":       link.ReverseDisplayName,
		}
	}
	return map[string]any{
		"apiName":           ontology.APIName,
		"displayName":       ontology.DisplayName,
		"pluralDisplayName": ontology.PluralDisplayName,
		"description":       ontology.Description,
		"primaryKey":        ontology.PrimaryKey,
		"tableName":         ontology.TableName,
		"schemaName":        ontology.SchemaName,
		"status":            ontology.Status,
		"icon":              defaultString(ontology.Icon, "table"),
		"color":             defaultString(ontology.Color, "#3366ff"),
		"properties":        properties,
		"links":             links,
	}
}

func transformCreatePayloadFromExport(transform platform.Transform) map[string]any {
	return map[string]any{
		"name":         transform.Name,
		"description":  transform.Description,
		"graphJson":    defaultString(transform.GraphJSON, `{"nodes":[],"edges":[]}`),
		"scheduleCron": transform.ScheduleCron,
		"triggerMode":  defaultString(transform.TriggerMode, "manual"),
	}
}

func functionCreatePayloadFromExport(fn platform.Function, keepProjectReferences bool) map[string]any {
	files := make([]map[string]any, len(fn.Files))
	for i, file := range fn.Files {
		files[i] = map[string]any{"path": file.Path, "content": file.Content}
	}
	dependencies := make([]map[string]any, len(fn.Dependencies))
	for i, dep := range fn.Dependencies {
		dependencies[i] = map[string]any{"name": dep.Name, "version": dep.Version}
	}
	payload := map[string]any{
		"name":           fn.Name,
		"description":    fn.Description,
		"language":       fn.Language,
		"entryPoint":     fn.EntryPoint,
		"timeoutSeconds": fn.TimeoutSeconds,
		"memory":         defaultString(fn.Memory, "128Mi"),
		"cpu":            defaultString(fn.CPU, "100m"),
		"files":          files,
		"dependencies":   dependencies,
	}
	if fn.TimeoutSeconds <= 0 {
		payload["timeoutSeconds"] = 30
	}
	if keepProjectReferences {
		payload["providerIds"] = fn.ProviderIDs
		payload["ontologyIds"] = fn.OntologyIDs
		payload["readOnlyOntologyIds"] = fn.ReadOnlyOntologyIDs
		payload["providerConfigs"] = fn.ProviderConfigs
		payload["secretBindings"] = fn.SecretBindings
	}
	if fn.DefaultMaxTokens > 0 {
		payload["defaultMaxTokens"] = fn.DefaultMaxTokens
	}
	if fn.DefaultTemperature > 0 {
		payload["defaultTemperature"] = fn.DefaultTemperature
	}
	return payload
}
