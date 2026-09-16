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
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/clidey/whodb/cli/internal/platform"
	"github.com/clidey/whodb/cli/pkg/output"
	"github.com/spf13/cobra"
)

type platformApp struct {
	ID                  string   `json:"id"`
	Name                string   `json:"name"`
	Description         string   `json:"description"`
	OntologyIDs         []string `json:"ontologyIds"`
	ReadOnlyOntologyIDs []string `json:"readOnlyOntologyIds"`
	FunctionIDs         []string `json:"functionIds"`
	UpdatedAt           string   `json:"updatedAt"`
}

type platformAppFile struct {
	Path      string `json:"path"`
	Content   string `json:"content"`
	UpdatedAt string `json:"updatedAt"`
}

type appCloneResult struct {
	SourceAppID string   `json:"sourceAppId"`
	TargetAppID string   `json:"targetAppId,omitempty"`
	Name        string   `json:"name"`
	Action      string   `json:"action"`
	Files       int      `json:"files"`
	Warnings    []string `json:"warnings,omitempty"`
}

var (
	appsCmd            = &cobra.Command{Use: "apps", Short: "Manage hosted WhoDB applications"}
	appCloneToOrg      string
	appCloneToProject  string
	appCloneBrandDir   string
	appCloneNamePrefix string
	appCloneOverwrite  bool
)

var appsListCmd = platformProjectListCommand("list", "List hosted WhoDB applications", func(ctx context.Context, session *platformSession, project *platform.Project) (any, *output.QueryResult, error) {
	apps, err := readPlatformApps(ctx, session, project.ID)
	if err != nil {
		return nil, nil, err
	}
	rows := make([][]any, len(apps))
	for index, app := range apps {
		rows[index] = []any{app.ID, app.Name, app.Description, len(app.OntologyIDs), len(app.FunctionIDs), app.UpdatedAt}
	}
	return apps, tableResult([]string{"id", "name", "description", "ontologies", "functions", "updated_at"}, rows), nil
})

var appsCloneCmd = &cobra.Command{
	Use:           "clone <app>",
	Short:         "Clone an app into another project and remap its resource references",
	Args:          cobra.ExactArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runPlatformAppClone,
}

func registerPlatformAppCommands() {
	appsCmd.PersistentFlags().StringVar(&platformResourceOrg, "org", "", "source organization id, slug, or name (defaults to selected organization)")
	appsCmd.PersistentFlags().StringVar(&platformResourceProject, "project", "", "source project id, slug, or name (defaults to selected project)")
	appsCloneCmd.Flags().StringVar(&appCloneToOrg, "to-org", "", "target organization id, slug, or name")
	appsCloneCmd.Flags().StringVar(&appCloneToProject, "to-project", "", "target project id, slug, or name")
	appsCloneCmd.Flags().StringVar(&appCloneBrandDir, "brand-dir", "", "directory of app files to merge over the cloned app")
	appsCloneCmd.Flags().StringVar(&appCloneNamePrefix, "name-prefix", "", "prefix added to the cloned app name")
	appsCloneCmd.Flags().BoolVar(&appCloneOverwrite, "overwrite", false, "update an existing target app with the same name")
	appsCloneCmd.Flags().BoolVarP(&platformWriteYes, "yes", "y", false, "clone without first printing the plan")
	appsCmd.AddCommand(appsListCmd, appsCloneCmd)
}

func readPlatformApps(ctx context.Context, session *platformSession, projectID string) ([]platformApp, error) {
	data, err := session.Client.PlatformQuery(ctx, "ProjectApps", map[string]any{"projectId": projectID})
	if err != nil {
		return nil, err
	}
	var apps []platformApp
	if err := remarshalPlatformData(data, &apps); err != nil {
		return nil, fmt.Errorf("decode project apps: %w", err)
	}
	return apps, nil
}

func readPlatformAppFiles(ctx context.Context, session *platformSession, projectID, appID string) ([]platformAppFile, error) {
	data, err := session.Client.PlatformQuery(ctx, "AppFiles", map[string]any{"projectId": projectID, "appId": appID})
	if err != nil {
		return nil, err
	}
	var files []platformAppFile
	if err := remarshalPlatformData(data, &files); err != nil {
		return nil, fmt.Errorf("decode app files: %w", err)
	}
	return files, nil
}

func remarshalPlatformData(data any, target any) error {
	raw, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, target)
}

func findPlatformApp(apps []platformApp, ref string) *platformApp {
	for index := range apps {
		if apps[index].ID == ref || strings.EqualFold(apps[index].Name, ref) {
			return &apps[index]
		}
	}
	return nil
}

func runPlatformAppClone(cmd *cobra.Command, args []string) error {
	if strings.TrimSpace(appCloneToProject) == "" {
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
	session.Client.SetWorkspaceContext(sourceOrg.ID, sourceProject.ID)
	sourceApps, err := readPlatformApps(ctx, session, sourceProject.ID)
	if err != nil {
		return err
	}
	sourceApp := findPlatformApp(sourceApps, args[0])
	if sourceApp == nil {
		return fmt.Errorf("app %q not found", args[0])
	}
	sourceFiles, err := readPlatformAppFiles(ctx, session, sourceProject.ID, sourceApp.ID)
	if err != nil {
		return err
	}
	sourceOntologies, err := session.Client.Ontologies(ctx, sourceProject.ID)
	if err != nil {
		return err
	}
	sourceFunctions, err := session.Client.Functions(ctx, sourceProject.ID, nil)
	if err != nil {
		return err
	}

	targetOrgValue := appCloneToOrg
	if strings.TrimSpace(targetOrgValue) == "" {
		targetOrgValue = sourceOrg.ID
	}
	targetOrg, targetProject, err := resolvePlatformProject(ctx, session, targetOrgValue, appCloneToProject)
	if err != nil {
		return err
	}
	session.Client.SetWorkspaceContext(targetOrg.ID, targetProject.ID)
	targetOntologies, err := session.Client.Ontologies(ctx, targetProject.ID)
	if err != nil {
		return err
	}
	targetFunctions, err := session.Client.Functions(ctx, targetProject.ID, nil)
	if err != nil {
		return err
	}
	targetApps, err := readPlatformApps(ctx, session, targetProject.ID)
	if err != nil {
		return err
	}

	warnings := make([]string, 0)
	ontologyIDs := remapAppOntologyIDs(sourceApp.OntologyIDs, sourceOntologies, targetOntologies, &warnings)
	readOnlyIDs := remapAppOntologyIDs(sourceApp.ReadOnlyOntologyIDs, sourceOntologies, targetOntologies, &warnings)
	functionIDs := remapAppFunctionIDs(sourceApp.FunctionIDs, sourceFunctions, targetFunctions, &warnings)
	targetName := appCloneNamePrefix + sourceApp.Name
	existing := findPlatformApp(targetApps, targetName)
	action := "create"
	if existing != nil {
		if !appCloneOverwrite {
			return fmt.Errorf("target app %q already exists; pass --overwrite to update it", targetName)
		}
		action = "update"
	}
	files, err := mergeAppBrandFiles(sourceFiles, appCloneBrandDir)
	if err != nil {
		return err
	}
	result := appCloneResult{SourceAppID: sourceApp.ID, Name: targetName, Action: action, Files: len(files), Warnings: warnings}
	if !platformWriteYes {
		return writeAppCloneResult(cmd, format, result)
	}

	payload := map[string]any{
		"projectId":           targetProject.ID,
		"name":                targetName,
		"description":         sourceApp.Description,
		"ontologyIds":         ontologyIDs,
		"readOnlyOntologyIds": readOnlyIDs,
		"functionIds":         functionIDs,
	}
	mutationName := "CreateApp"
	if existing != nil {
		mutationName = "UpdateApp"
		payload["id"] = existing.ID
	}
	mutation, err := session.Client.PlatformMutation(ctx, mutationName, map[string]any{"input": payload})
	if err != nil {
		return err
	}
	var targetApp platformApp
	if err := json.Unmarshal(mutation.Data, &targetApp); err != nil {
		return fmt.Errorf("decode cloned app: %w", err)
	}
	for _, file := range files {
		_, err := session.Client.PlatformMutation(ctx, "UpsertAppFile", map[string]any{
			"projectId": targetProject.ID,
			"appId":     targetApp.ID,
			"path":      file.Path,
			"content":   file.Content,
		})
		if err != nil {
			return fmt.Errorf("upsert app file %s: %w", file.Path, err)
		}
	}
	result.TargetAppID = targetApp.ID
	result.Action += "d"
	return writeAppCloneResult(cmd, format, result)
}

func remapAppOntologyIDs(ids []string, source, target []platform.Ontology, warnings *[]string) []string {
	sourceNames := make(map[string]string, len(source))
	for _, ontology := range source {
		sourceNames[ontology.ID] = ontology.APIName
	}
	targetExact := make(map[string]string, len(target))
	targetFolded := make(map[string]string, len(target))
	for _, ontology := range target {
		targetExact[ontology.APIName] = ontology.ID
		if _, exists := targetFolded[strings.ToLower(ontology.APIName)]; !exists {
			targetFolded[strings.ToLower(ontology.APIName)] = ontology.ID
		}
	}
	result := make([]string, 0, len(ids))
	for _, id := range ids {
		name, ok := sourceNames[id]
		if !ok {
			*warnings = append(*warnings, "source ontology reference not found: "+id)
			continue
		}
		targetID, ok := targetExact[name]
		if !ok {
			targetID, ok = targetFolded[strings.ToLower(name)]
		}
		if !ok {
			*warnings = append(*warnings, "target ontology not found: "+name)
			continue
		}
		result = append(result, targetID)
	}
	return result
}

func remapAppFunctionIDs(ids []string, source, target []platform.Function, warnings *[]string) []string {
	sourceNames := make(map[string]string, len(source))
	for _, function := range source {
		sourceNames[function.ID] = function.Name
	}
	targetIDs := make(map[string]string, len(target))
	for _, function := range target {
		targetIDs[strings.ToLower(function.Name)] = function.ID
	}
	result := make([]string, 0, len(ids))
	for _, id := range ids {
		name, ok := sourceNames[id]
		if !ok {
			*warnings = append(*warnings, "source function reference not found: "+id)
			continue
		}
		targetID, ok := targetIDs[strings.ToLower(name)]
		if !ok {
			*warnings = append(*warnings, "target function not found: "+name)
			continue
		}
		result = append(result, targetID)
	}
	return result
}

func mergeAppBrandFiles(source []platformAppFile, brandDir string) ([]platformAppFile, error) {
	files := make(map[string]platformAppFile, len(source))
	for _, file := range source {
		files[file.Path] = file
	}
	if strings.TrimSpace(brandDir) != "" {
		root, err := filepath.Abs(brandDir)
		if err != nil {
			return nil, err
		}
		err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil || entry.IsDir() {
				return walkErr
			}
			relative, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			targetPath := filepath.ToSlash(relative)
			if targetPath == "app.json" {
				content, err = mergeAppJSON(files[targetPath].Content, content)
				if err != nil {
					return err
				}
			}
			files[targetPath] = platformAppFile{Path: targetPath, Content: string(content)}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("read brand directory: %w", err)
		}
	}
	paths := make([]string, 0, len(files))
	for path := range files {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	result := make([]platformAppFile, 0, len(paths))
	for _, path := range paths {
		result = append(result, files[path])
	}
	return result, nil
}

func mergeAppJSON(baseContent string, overlayContent []byte) ([]byte, error) {
	base := map[string]any{}
	if strings.TrimSpace(baseContent) != "" {
		if err := json.Unmarshal([]byte(baseContent), &base); err != nil {
			return nil, fmt.Errorf("decode source app.json: %w", err)
		}
	}
	var overlay map[string]any
	if err := json.Unmarshal(overlayContent, &overlay); err != nil {
		return nil, fmt.Errorf("decode brand app.json: %w", err)
	}
	for key, value := range overlay {
		base[key] = value
	}
	return json.MarshalIndent(base, "", "  ")
}

func writeAppCloneResult(cmd *cobra.Command, format output.Format, result appCloneResult) error {
	if format == output.FormatJSON {
		return writeCommandJSON(cmd, result)
	}
	return newCommandOutput(cmd, format, platformQuiet).WriteQueryResult(tableResult(
		[]string{"source_app_id", "target_app_id", "name", "action", "files", "warnings"},
		[][]any{{result.SourceAppID, result.TargetAppID, result.Name, result.Action, result.Files, strings.Join(result.Warnings, "; ")}},
	))
}
