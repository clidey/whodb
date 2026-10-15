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

// appCloneServerResult is the CloneApp mutation result.
type appCloneServerResult struct {
	Action        string   `json:"action"`
	Name          string   `json:"name"`
	TargetAppID   *string  `json:"targetAppId"`
	FileCount     int      `json:"fileCount"`
	Warnings      []string `json:"warnings"`
	BackupVersion *int     `json:"backupVersion"`
	DryRun        bool     `json:"dryRun"`
}

type appCloneResult struct {
	SourceAppID   string   `json:"sourceAppId"`
	TargetAppID   string   `json:"targetAppId,omitempty"`
	Name          string   `json:"name"`
	Action        string   `json:"action"`
	Files         int      `json:"files"`
	BrandFiles    int      `json:"brandFiles"`
	BackupVersion *int     `json:"backupVersion,omitempty"`
	DryRun        bool     `json:"dryRun"`
	Warnings      []string `json:"warnings,omitempty"`
	Undo          string   `json:"undo,omitempty"`
}

// appCloneOptions are the apps clone flags.
type appCloneOptions struct {
	SourceOrg     string
	SourceProject string
	TargetOrg     string
	TargetProject string
	BrandDir      string
	NamePrefix    string
	Overwrite     bool
	Execute       bool
}

type appUndoCloneResult struct {
	AppID         string `json:"appId"`
	Name          string `json:"name"`
	Action        string `json:"action"`
	BackupVersion int    `json:"backupVersion,omitempty"`
	DryRun        bool   `json:"dryRun"`
}

var (
	appsCmd             = &cobra.Command{Use: "apps", Short: "Manage hosted WhoDB applications"}
	appCloneToOrg       string
	appCloneToProject   string
	appCloneBrandDir    string
	appCloneNamePrefix  string
	appCloneOverwrite   bool
	appUndoCloneVersion int
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
	Use:   "clone <app>",
	Short: "Clone an app into another project and remap its resource references",
	Long: `Clone an app into another project. The server copies every file and the
app's metadata in one transaction, so the clone either fully succeeds or leaves
the target unchanged. Without --yes the server checks everything and prints the
plan without writing. Overwriting an existing app first saves its draft as a
backup version; the result prints the apps undo-clone command that reverses it.`,
	Args:          cobra.ExactArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runPlatformAppClone,
}

var appsUndoCloneCmd = &cobra.Command{
	Use:   "undo-clone <app>",
	Short: "Reverse an apps clone in the selected project",
	Long: `Reverse an apps clone. With --backup-version, restore the app's draft from
the backup the overwrite saved. Without it, delete the app the clone created;
deleted apps stay restorable from the trash. Without --yes, print the plan.`,
	Args:          cobra.ExactArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runPlatformAppUndoClone,
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
	appsUndoCloneCmd.Flags().IntVar(&appUndoCloneVersion, "backup-version", 0, "backup version printed by an overwriting clone; omit to delete a newly created clone")
	appsUndoCloneCmd.Flags().BoolVarP(&platformWriteYes, "yes", "y", false, "undo without first printing the plan")
	appsCmd.AddCommand(appsListCmd, appsCloneCmd, appsUndoCloneCmd)
	registerPlatformAppCaptureCommands()
	registerPlatformLocalAppCommands()
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
	result, err := clonePlatformApp(ctx, session, args[0], appCloneOptions{
		SourceOrg:     platformResourceOrg,
		SourceProject: platformResourceProject,
		TargetOrg:     appCloneToOrg,
		TargetProject: appCloneToProject,
		BrandDir:      appCloneBrandDir,
		NamePrefix:    appCloneNamePrefix,
		Overwrite:     appCloneOverwrite,
		Execute:       platformWriteYes,
	})
	if err != nil {
		return err
	}
	if format == output.FormatJSON {
		return writeCommandJSON(cmd, result)
	}
	backup := ""
	if result.BackupVersion != nil {
		backup = fmt.Sprint(*result.BackupVersion)
	}
	return newCommandOutput(cmd, format, platformQuiet).WriteQueryResult(tableResult(
		[]string{"source_app_id", "target_app_id", "name", "action", "files", "brand_files", "backup_version", "dry_run", "warnings", "undo"},
		[][]any{{result.SourceAppID, result.TargetAppID, result.Name, result.Action, result.Files, result.BrandFiles, backup, result.DryRun, strings.Join(result.Warnings, "; "), result.Undo}},
	))
}

// clonePlatformApp runs CloneApp as a dry run, or for real when options.Execute
// is set. Brand files travel in the same request, so they commit or roll back
// with the rest of the clone.
func clonePlatformApp(ctx context.Context, session *platformSession, appRef string, options appCloneOptions) (*appCloneResult, error) {
	sourceOrg, sourceProject, err := resolvePlatformProject(ctx, session, options.SourceOrg, options.SourceProject)
	if err != nil {
		return nil, err
	}
	session.Client.SetWorkspaceContext(sourceOrg.ID, sourceProject.ID)
	sourceApps, err := readPlatformApps(ctx, session, sourceProject.ID)
	if err != nil {
		return nil, err
	}
	sourceApp := findPlatformApp(sourceApps, appRef)
	if sourceApp == nil {
		return nil, fmt.Errorf("app %q not found", appRef)
	}
	brandFiles, err := readAppBrandFiles(options.BrandDir, func() (string, error) {
		return readPlatformAppJSON(ctx, session, sourceProject.ID, sourceApp.ID)
	})
	if err != nil {
		return nil, err
	}

	targetOrgValue := options.TargetOrg
	if strings.TrimSpace(targetOrgValue) == "" {
		targetOrgValue = sourceOrg.ID
	}
	targetOrg, targetProject, err := resolvePlatformProject(ctx, session, targetOrgValue, options.TargetProject)
	if err != nil {
		return nil, err
	}

	files := make([]map[string]any, len(brandFiles))
	for i, file := range brandFiles {
		files[i] = map[string]any{"path": file.Path, "content": file.Content}
	}
	variables := map[string]any{
		"projectId":       sourceProject.ID,
		"appId":           sourceApp.ID,
		"targetProjectId": targetProject.ID,
		"name":            options.NamePrefix + sourceApp.Name,
		"overwrite":       options.Overwrite,
		"files":           files,
		"dryRun":          !options.Execute,
	}
	if targetOrg.ID != sourceOrg.ID {
		variables["targetOrgId"] = targetOrg.ID
	}
	// The request runs in the source workspace; the server opens the target.
	session.Client.SetWorkspaceContext(sourceOrg.ID, sourceProject.ID)
	mutation, err := session.Client.PlatformMutation(ctx, "CloneApp", variables)
	if err != nil {
		return nil, err
	}
	var server appCloneServerResult
	if err := json.Unmarshal(mutation.Data, &server); err != nil {
		return nil, fmt.Errorf("decode clone result: %w", err)
	}
	result := &appCloneResult{
		SourceAppID:   sourceApp.ID,
		Name:          server.Name,
		Action:        server.Action,
		Files:         server.FileCount,
		BrandFiles:    len(brandFiles),
		BackupVersion: server.BackupVersion,
		DryRun:        server.DryRun,
		Warnings:      server.Warnings,
	}
	if server.TargetAppID != nil {
		result.TargetAppID = *server.TargetAppID
	}
	if !server.DryRun {
		result.Action += "d"
		result.Undo = fmt.Sprintf("whodb apps undo-clone %s --org %s --project %s", result.TargetAppID, targetOrg.ID, targetProject.ID)
		if server.BackupVersion != nil {
			result.Undo += fmt.Sprintf(" --backup-version %d", *server.BackupVersion)
		}
	}
	return result, nil
}

func runPlatformAppUndoClone(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	format, err := output.ParseFormat(platformFormat)
	if err != nil {
		return err
	}
	session, err := loadPlatformSession(ctx, platformHost)
	if err != nil {
		return err
	}
	result, err := undoPlatformAppClone(ctx, session, args[0], platformResourceOrg, platformResourceProject, appUndoCloneVersion, platformWriteYes)
	if err != nil {
		return err
	}
	if format == output.FormatJSON {
		return writeCommandJSON(cmd, result)
	}
	backup := ""
	if result.BackupVersion > 0 {
		backup = fmt.Sprint(result.BackupVersion)
	}
	return newCommandOutput(cmd, format, platformQuiet).WriteQueryResult(tableResult(
		[]string{"app_id", "name", "action", "backup_version", "dry_run"},
		[][]any{{result.AppID, result.Name, result.Action, backup, result.DryRun}},
	))
}

// undoPlatformAppClone restores an overwritten app from its backup version or,
// when backupVersion is 0, deletes the app a clone created. It only describes
// the action unless execute is set.
func undoPlatformAppClone(ctx context.Context, session *platformSession, appRef, orgRef, projectRef string, backupVersion int, execute bool) (*appUndoCloneResult, error) {
	if backupVersion < 0 {
		return nil, fmt.Errorf("--backup-version must be positive")
	}
	org, project, err := resolvePlatformProject(ctx, session, orgRef, projectRef)
	if err != nil {
		return nil, err
	}
	session.Client.SetWorkspaceContext(org.ID, project.ID)
	apps, err := readPlatformApps(ctx, session, project.ID)
	if err != nil {
		return nil, err
	}
	app := findPlatformApp(apps, appRef)
	if app == nil {
		return nil, fmt.Errorf("app %q not found", appRef)
	}
	result := &appUndoCloneResult{AppID: app.ID, Name: app.Name, Action: "delete", BackupVersion: backupVersion, DryRun: !execute}
	if backupVersion > 0 {
		result.Action = "restore"
	}
	if !execute {
		return result, nil
	}
	if backupVersion > 0 {
		_, err = session.Client.PlatformMutation(ctx, "RestoreAppVersionToDraft", map[string]any{"projectId": project.ID, "appId": app.ID, "version": backupVersion})
	} else {
		_, err = session.Client.PlatformMutation(ctx, "DeleteApp", map[string]any{"projectId": project.ID, "id": app.ID})
	}
	if err != nil {
		return nil, err
	}
	result.Action += "d"
	return result, nil
}

// readPlatformAppJSON returns the app's draft app.json, or "" when it has none.
// The draft app view carries app.json alongside shared files and one page.
func readPlatformAppJSON(ctx context.Context, session *platformSession, projectID, appID string) (string, error) {
	data, err := session.Client.PlatformQuery(ctx, "AppView", map[string]any{"projectId": projectID, "id": appID, "env": "dev"})
	if err != nil {
		return "", err
	}
	var view struct {
		Files []platformAppFile `json:"files"`
	}
	if err := remarshalPlatformData(data, &view); err != nil {
		return "", fmt.Errorf("decode app view: %w", err)
	}
	for _, file := range view.Files {
		if strings.TrimPrefix(file.Path, "/") == "app.json" {
			return file.Content, nil
		}
	}
	return "", nil
}

// readAppBrandFiles returns the regular files under brandDir as app files
// sorted by path. Hidden files and directories are skipped, and symlinks are
// rejected so nothing outside the directory is uploaded. A brand app.json is
// merged over the source app.json, which sourceAppJSON loads only when needed.
func readAppBrandFiles(brandDir string, sourceAppJSON func() (string, error)) ([]platformAppFile, error) {
	if strings.TrimSpace(brandDir) == "" {
		return nil, nil
	}
	root, err := filepath.Abs(brandDir)
	if err != nil {
		return nil, err
	}
	files := make([]platformAppFile, 0)
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path != root && strings.HasPrefix(entry.Name(), ".") {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Type()&fs.ModeSymlink != 0 {
			return fmt.Errorf("%s is a symlink; brand files must be regular files", path)
		}
		if entry.IsDir() {
			return nil
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
			base, err := sourceAppJSON()
			if err != nil {
				return err
			}
			content, err = mergeAppJSON(base, content)
			if err != nil {
				return err
			}
		}
		files = append(files, platformAppFile{Path: targetPath, Content: string(content)})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("read brand directory: %w", err)
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
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
