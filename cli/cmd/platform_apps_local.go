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
	"path/filepath"
	"sort"
	"strings"

	"github.com/clidey/whodb/cli/internal/appcapture"
	"github.com/clidey/whodb/cli/internal/appfolder"
	"github.com/clidey/whodb/cli/internal/platform"
	"github.com/clidey/whodb/cli/pkg/output"
	"github.com/spf13/cobra"
)

var (
	appLocalRef          string
	appDryRun            bool
	appDraftOnly         bool
	appPublishNote       string
	appScenarioPath      string
	appVerifyOutput      string
	appVerifyPage        string
	appVerifyWidth       int
	appVerifyHeight      int
	appVerifyMobile      bool
	appVerifyTablet      bool
	appVerifyOrientation string
	appAllowWrites       bool
	appInitName          string
	appInitFrom          string
	appKernelDir         string
)

type appPublishPlan struct {
	AppID               string   `json:"app_id,omitempty"`
	Name                string   `json:"name"`
	Action              string   `json:"action"`
	UpsertPaths         []string `json:"upsert_paths"`
	DeletePaths         []string `json:"delete_paths"`
	OntologyIDs         []string `json:"ontology_ids"`
	ReadOnlyOntologyIDs []string `json:"read_only_ontology_ids"`
	FunctionIDs         []string `json:"function_ids"`
	BindingsChanged     bool     `json:"bindings_changed"`
	Version             int      `json:"version,omitempty"`
	DraftOnly           bool     `json:"draft_only"`
	DryRun              bool     `json:"dry_run"`
}

var appsCheckCmd = &cobra.Command{
	Use:           "check <folder>",
	Short:         "Validate a local app folder and its event references",
	Args:          cobra.ExactArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		workspace, err := appfolder.Read(args[0])
		if err != nil {
			return err
		}
		format, err := output.ParseFormat(platformFormat)
		if err != nil {
			return err
		}
		if format == output.FormatJSON {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{
				"valid": true, "name": workspace.Manifest.Name, "pages": workspace.Pages, "files": len(workspace.Files),
			})
		}
		_, err = fmt.Fprintf(cmd.OutOrStdout(), "%s: valid (%d pages, %d files)\n", workspace.Manifest.Name, len(workspace.Pages), len(workspace.Files))
		return err
	},
}

var appsInitCmd = &cobra.Command{
	Use:           "init <folder>",
	Short:         "Create a local app folder or import an existing hosted app",
	Args:          cobra.ExactArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if appInitFrom != "" {
			if cmd.Flags().Changed("name") {
				return fmt.Errorf("--name cannot be used with --from; edit app.json after import to rename the app")
			}
			session, err := loadPlatformSession(cmd.Context(), platformHost)
			if err != nil {
				return err
			}
			org, project, err := resolvePlatformProject(cmd.Context(), session, platformResourceOrg, platformResourceProject)
			if err != nil {
				return err
			}
			session.Client.SetWorkspaceContext(org.ID, project.ID)
			app, count, err := initAppFromHosted(cmd.Context(), session, project.ID, appInitFrom, args[0])
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Imported %q (%s) to %s (%d files)\n", app.Name, app.ID, args[0], count)
			return err
		}
		name := strings.TrimSpace(appInitName)
		if name == "" {
			name = filepath.Base(filepath.Clean(args[0]))
		}
		if name == "." || name == string(filepath.Separator) || name == "" {
			return fmt.Errorf("app name is required")
		}
		if err := os.Mkdir(args[0], 0o755); err != nil {
			return fmt.Errorf("create app folder: %w", err)
		}
		manifest, err := json.MarshalIndent(appfolder.Manifest{Name: name, DefaultPage: "main", Ontologies: []string{}, ReadOnlyOntologies: []string{}, Functions: []string{}}, "", "  ")
		if err != nil {
			return err
		}
		files := []appfolder.File{
			{Path: "app.json", Content: string(manifest) + "\n"},
			{Path: "pages/main/tree.json", Content: `{"version":1,"root":{"id":"root","type":"Col","props":{"gap":4,"p":6,"w":"full"},"children":[{"id":"increment-button","type":"Button","props":{},"on":{"click":"increment"},"children":[{"id":"increment-label","type":"Text","props":{"content":"Increment"}}]},{"id":"count-text","type":"Text","props":{"content":""},"bind":{"content":"state.count"}}]}}` + "\n"},
			{Path: "pages/main/state.json", Content: "{\"count\":0}\n"},
			{Path: "pages/main/scripts/increment.js", Content: "(event, ctx) => { ctx.set('count', Number(ctx.get('count') || 0) + 1); }\n"},
			{Path: "shared/state.json", Content: "{}\n"},
			{Path: "scenarios/smoke.json", Content: `{"actions":[{"kind":"click","selector":"[data-node-id='increment-button']"},{"kind":"expect_text","value":"1"}]}` + "\n"},
		}
		for _, file := range files {
			path := filepath.Join(args[0], filepath.FromSlash(file.Path))
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return err
			}
			if err := os.WriteFile(path, []byte(file.Content), 0o644); err != nil {
				return err
			}
		}
		_, err = fmt.Fprintf(cmd.OutOrStdout(), "Created %s\n", args[0])
		return err
	},
}

var appsPublishCmd = &cobra.Command{
	Use:   "publish <folder>",
	Short: "Sync a local app folder and promote it as a hosted app version",
	Long: `Validate a local app folder, resolve its ontology and function names in the
selected project, sync changed files to the app draft, and promote a version.
Use --draft to stop after syncing. An existing published version remains active
if a file write fails; rerun the command to complete the draft sync.`,
	Args:          cobra.ExactArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runPlatformAppPublish,
}

var appsDevCmd = &cobra.Command{
	Use:           "dev <folder>",
	Short:         "Preview local app files in the hosted runtime and reload on edits",
	Args:          cobra.ExactArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runPlatformAppDev,
}

var appsVerifyCmd = &cobra.Command{
	Use:           "verify <folder>",
	Short:         "Replay a local app scenario and save a screenshot and flow trace",
	Args:          cobra.ExactArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runPlatformAppVerify,
}

func registerPlatformLocalAppCommands() {
	appsInitCmd.Flags().StringVar(&appInitName, "name", "", "app name; defaults to the folder name")
	appsInitCmd.Flags().StringVar(&appInitFrom, "from", "", "import an existing hosted app by ID or name")
	appsPublishCmd.Flags().StringVar(&appLocalRef, "app", "", "existing app ID or name; defaults to the name in app.json")
	appsPublishCmd.Flags().BoolVar(&appDryRun, "dry-run", false, "show the files and bindings that would change")
	appsPublishCmd.Flags().BoolVar(&appDraftOnly, "draft", false, "sync the draft without promoting a version")
	appsPublishCmd.Flags().StringVar(&appPublishNote, "message", "", "message attached to the promoted version")
	appsDevCmd.Flags().StringVar(&appLocalRef, "app", "", "existing hosted app ID or name; defaults to app.json name")
	appsDevCmd.Flags().StringVar(&appVerifyPage, "page", "", "page to open first")
	appsDevCmd.Flags().BoolVar(&appAllowWrites, "allow-writes", true, "allow app actions to mutate hosted development data")
	appsDevCmd.Flags().BoolVar(&appCaptureInstall, "install", false, "install the optional capture browser")
	appsDevCmd.Flags().StringVar(&appKernelDir, "kernel-dir", os.Getenv("WHODB_APP_KERNEL_DIR"), "local built sandbox assets, for testing runtime changes before server deployment")
	appsVerifyCmd.Flags().StringVar(&appLocalRef, "app", "", "existing hosted app ID or name; defaults to app.json name")
	appsVerifyCmd.Flags().StringVar(&appScenarioPath, "scenario", "", "JSON file with an ordered actions array")
	appsVerifyCmd.Flags().StringVar(&appVerifyOutput, "output", "", "WebP screenshot path; defaults to <app-name>-verify.webp")
	appsVerifyCmd.Flags().StringVar(&appVerifyPage, "page", "", "page to open first")
	appsVerifyCmd.Flags().IntVar(&appVerifyWidth, "width", 1440, "screenshot viewport width in pixels")
	appsVerifyCmd.Flags().IntVar(&appVerifyHeight, "height", 900, "screenshot viewport height in pixels")
	appsVerifyCmd.Flags().BoolVar(&appVerifyMobile, "mobile", false, "emulate a touch-enabled mobile browser (default viewport 390x844)")
	appsVerifyCmd.Flags().BoolVar(&appVerifyTablet, "tablet", false, "emulate a touch-enabled tablet browser (default viewport 820x1180)")
	appsVerifyCmd.Flags().StringVar(&appVerifyOrientation, "orientation", "", "viewport orientation: portrait or landscape (defaults to the device preset)")
	appsVerifyCmd.Flags().BoolVar(&appAllowWrites, "allow-writes", true, "allow scenario actions to mutate hosted development data")
	appsVerifyCmd.Flags().BoolVar(&appCaptureInstall, "install", false, "install the optional capture browser")
	appsVerifyCmd.Flags().StringVar(&appKernelDir, "kernel-dir", os.Getenv("WHODB_APP_KERNEL_DIR"), "local built sandbox assets, for testing runtime changes before server deployment")
	appsCmd.AddCommand(appsInitCmd, appsCheckCmd, appsPublishCmd, appsDevCmd, appsVerifyCmd)
}

func localAppCapture(ctx context.Context, folder string) (*appfolder.Workspace, *platformSession, appcapture.Options, error) {
	workspace, err := appfolder.Read(folder)
	if err != nil {
		return nil, nil, appcapture.Options{}, err
	}
	ref := appLocalRef
	if ref == "" {
		ref = workspace.Manifest.Name
	}
	session, options, _, err := resolveAppCapture(ctx, ref)
	if err != nil {
		return nil, nil, appcapture.Options{}, err
	}
	options.Env = "dev"
	options.Page = appVerifyPage
	options.LocalDir = folder
	options.KernelDir = appKernelDir
	options.AllowWrites = appAllowWrites
	writeIDs, readOnlyIDs, functionIDs, err := resolveAppFolderBindings(ctx, session.Client, options.ProjectID, workspace.Manifest)
	if err != nil {
		return nil, nil, appcapture.Options{}, err
	}
	apps, err := readPlatformApps(ctx, session, options.ProjectID)
	if err != nil {
		return nil, nil, appcapture.Options{}, err
	}
	remote := findPlatformApp(apps, options.AppID)
	if remote == nil || !sameStrings(remote.OntologyIDs, writeIDs) || !sameStrings(remote.ReadOnlyOntologyIDs, readOnlyIDs) || !sameStrings(remote.FunctionIDs, functionIDs) {
		return nil, nil, appcapture.Options{}, fmt.Errorf("hosted app bindings differ from app.json; run apps publish %s --draft to sync them before preview", folder)
	}
	return workspace, session, options, nil
}

func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	seen := make(map[string]int, len(left))
	for _, value := range left {
		seen[value]++
	}
	for _, value := range right {
		seen[value]--
		if seen[value] < 0 {
			return false
		}
	}
	return true
}

func runPlatformAppDev(cmd *cobra.Command, args []string) error {
	_, session, options, err := localAppCapture(cmd.Context(), args[0])
	if err != nil {
		return err
	}
	if appCaptureInstall {
		if _, err := appcapture.SetupRuntime(cmd.Context()); err != nil {
			return err
		}
	}
	tokenSource := platform.NewOIDCTokenSource(session.Host.URL, session.Host.AccountID, session.Config)
	options.Token, err = tokenSource.Token(cmd.Context())
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Previewing %s from %s; close the browser to stop\n", options.AppID, args[0])
	return appcapture.Preview(cmd.Context(), options)
}

func runPlatformAppVerify(cmd *cobra.Command, args []string) error {
	workspace, session, options, err := localAppCapture(cmd.Context(), args[0])
	if err != nil {
		return err
	}
	options.Mobile = appVerifyMobile
	options.Tablet = appVerifyTablet
	options.Orientation = appVerifyOrientation
	if cmd.Flags().Changed("width") {
		options.Width = appVerifyWidth
	}
	if cmd.Flags().Changed("height") {
		options.Height = appVerifyHeight
	}
	options.Width, options.Height, err = appcapture.ResolveViewport(options.Width, options.Height, options.Mobile, options.Tablet, options.Orientation)
	if err != nil {
		return err
	}
	if appScenarioPath != "" {
		data, err := os.ReadFile(appScenarioPath)
		if err != nil {
			return err
		}
		var scenario struct {
			Actions []appcapture.Action `json:"actions"`
		}
		if err := json.Unmarshal(data, &scenario); err != nil {
			return fmt.Errorf("decode scenario: %w", err)
		}
		options.Actions = scenario.Actions
	}
	if appCaptureInstall {
		if _, err := appcapture.SetupRuntime(cmd.Context()); err != nil {
			return err
		}
	}
	tokenSource := platform.NewOIDCTokenSource(session.Host.URL, session.Host.AccountID, session.Config)
	options.Token, err = tokenSource.Token(cmd.Context())
	if err != nil {
		return err
	}
	capture, err := appcapture.Capture(cmd.Context(), options)
	if err != nil {
		return err
	}
	path := appVerifyOutput
	if path == "" {
		path = strings.ToLower(strings.ReplaceAll(workspace.Manifest.Name, " ", "-")) + "-verify"
		if options.Mobile {
			path += "-mobile"
		} else if options.Tablet {
			path += "-tablet"
		}
		if (options.Mobile || options.Tablet) && options.Width > options.Height {
			path += "-landscape"
		}
		path += ".webp"
	}
	if strings.ToLower(filepath.Ext(path)) != ".webp" {
		return fmt.Errorf("output must be a .webp file")
	}
	if err := os.WriteFile(path, capture.Image, 0o644); err != nil {
		return err
	}
	passed := len(capture.Diagnostics.ActionErrors) == 0 && len(capture.Diagnostics.ScriptErrors) == 0 && len(capture.Diagnostics.PageErrors) == 0 && len(capture.Diagnostics.ConsoleErrors) == 0
	for _, step := range capture.Trace {
		if step.Kind == "error" || step.Kind == "handler-error" || step.Kind == "bridge-error" {
			passed = false
		}
	}
	format, err := output.ParseFormat(platformFormat)
	if err != nil {
		return err
	}
	if format == output.FormatJSON {
		if err := json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{
			"checks_passed": passed, "name": workspace.Manifest.Name, "screenshot": path,
			"width": capture.Width, "height": capture.Height, "mobile": options.Mobile, "tablet": options.Tablet, "orientation": options.Orientation,
			"visible_text": capture.Text, "diagnostics": capture.Diagnostics, "trace": capture.Trace,
		}); err != nil {
			return err
		}
		if !passed {
			return fmt.Errorf("scenario failed")
		}
		return nil
	}
	_, err = fmt.Fprintf(cmd.OutOrStdout(), "checks_passed=%t screenshot=%s viewport=%dx%d trace_steps=%d\n", passed, path, capture.Width, capture.Height, len(capture.Trace))
	if err != nil {
		return err
	}
	if !passed {
		return fmt.Errorf("scenario failed; rerun with --format json for diagnostics")
	}
	return nil
}

func runPlatformAppPublish(cmd *cobra.Command, args []string) error {
	workspace, err := appfolder.Read(args[0])
	if err != nil {
		return err
	}
	format, err := output.ParseFormat(platformFormat)
	if err != nil {
		return err
	}
	ctx := cmd.Context()
	session, err := loadPlatformSession(ctx, platformHost)
	if err != nil {
		return err
	}
	org, project, err := resolvePlatformProject(ctx, session, platformResourceOrg, platformResourceProject)
	if err != nil {
		return err
	}
	session.Client.SetWorkspaceContext(org.ID, project.ID)
	plan, err := publishAppFolder(ctx, session, project.ID, workspace, appLocalRef, appDryRun, appDraftOnly, appPublishNote)
	if err != nil {
		return err
	}
	if format == output.FormatJSON {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(plan)
	}
	verb := plan.Action
	if plan.DryRun {
		verb = "would " + verb
	} else if plan.DraftOnly {
		verb += " draft"
	} else {
		verb += " and promote"
	}
	_, err = fmt.Fprintf(cmd.OutOrStdout(), "%s %q (%s): %d files to upsert, %d to delete, bindings changed: %t, version: %d\n", verb, plan.Name, plan.AppID, len(plan.UpsertPaths), len(plan.DeletePaths), plan.BindingsChanged, plan.Version)
	return err
}

func publishAppFolder(ctx context.Context, session *platformSession, projectID string, workspace *appfolder.Workspace, appRef string, dryRun, draftOnly bool, message string) (*appPublishPlan, error) {
	apps, err := readPlatformApps(ctx, session, projectID)
	if err != nil {
		return nil, err
	}
	if appRef == "" {
		appRef = workspace.Manifest.Name
	}
	var target *platformApp
	for index := range apps {
		if apps[index].ID == appRef || strings.EqualFold(apps[index].Name, appRef) {
			if target != nil {
				return nil, fmt.Errorf("multiple apps match %q; pass an app ID with --app", appRef)
			}
			target = &apps[index]
		}
	}
	if target == nil && appRef != workspace.Manifest.Name {
		return nil, fmt.Errorf("app %q does not exist", appRef)
	}
	writeIDs, readOnlyIDs, functionIDs, err := resolveAppFolderBindings(ctx, session.Client, projectID, workspace.Manifest)
	if err != nil {
		return nil, err
	}
	plan := &appPublishPlan{
		Name: workspace.Manifest.Name, Action: "create", UpsertPaths: []string{}, DeletePaths: []string{},
		OntologyIDs: writeIDs, ReadOnlyOntologyIDs: readOnlyIDs, FunctionIDs: functionIDs,
		BindingsChanged: true, DraftOnly: draftOnly, DryRun: dryRun,
	}
	remoteFiles := []platformAppFile{}
	if target != nil {
		plan.Action = "update"
		plan.AppID = target.ID
		plan.BindingsChanged = !sameStrings(target.OntologyIDs, writeIDs) || !sameStrings(target.ReadOnlyOntologyIDs, readOnlyIDs) || !sameStrings(target.FunctionIDs, functionIDs)
		data, err := session.Client.PlatformQuery(ctx, "AppFiles", map[string]any{"projectId": projectID, "appId": target.ID})
		if err != nil {
			return nil, err
		}
		if err := remarshalPlatformData(data, &remoteFiles); err != nil {
			return nil, fmt.Errorf("decode app files: %w", err)
		}
	}
	remoteByPath := make(map[string]string, len(remoteFiles))
	for _, file := range remoteFiles {
		remoteByPath[file.Path] = file.Content
	}
	localByPath := make(map[string]string, len(workspace.Files))
	for _, file := range workspace.Files {
		localByPath[file.Path] = file.Content
		if remoteByPath[file.Path] != file.Content {
			plan.UpsertPaths = append(plan.UpsertPaths, file.Path)
		}
	}
	for path := range remoteByPath {
		if _, ok := localByPath[path]; !ok {
			plan.DeletePaths = append(plan.DeletePaths, path)
		}
	}
	if target == nil {
		for _, path := range []string{"pages/main/tree.json", "pages/main/state.json", "shared/state.json"} {
			if _, ok := localByPath[path]; !ok {
				plan.DeletePaths = append(plan.DeletePaths, path)
			}
		}
	}
	sort.Strings(plan.DeletePaths)
	if dryRun {
		return plan, nil
	}
	if target == nil {
		result, err := session.Client.PlatformMutation(ctx, "CreateApp", map[string]any{"input": map[string]any{
			"projectId": projectID, "name": workspace.Manifest.Name, "description": workspace.Manifest.Description,
			"ontologyIds": writeIDs, "readOnlyOntologyIds": readOnlyIDs, "functionIds": functionIDs,
		}})
		if err != nil {
			return nil, err
		}
		var created platformApp
		if err := json.Unmarshal(result.Data, &created); err != nil {
			return nil, fmt.Errorf("decode created app: %w", err)
		}
		plan.AppID = created.ID
	}
	for _, path := range plan.UpsertPaths {
		if _, err := session.Client.PlatformMutation(ctx, "UpsertAppFile", map[string]any{
			"projectId": projectID, "appId": plan.AppID, "path": path, "content": localByPath[path],
		}); err != nil {
			return nil, fmt.Errorf("upsert %s: %w", path, err)
		}
	}
	for _, path := range plan.DeletePaths {
		if _, err := session.Client.PlatformMutation(ctx, "DeleteAppFile", map[string]any{
			"projectId": projectID, "appId": plan.AppID, "path": path,
		}); err != nil {
			return nil, fmt.Errorf("delete %s: %w", path, err)
		}
	}
	if target != nil {
		_, err := session.Client.PlatformMutation(ctx, "UpdateApp", map[string]any{"input": map[string]any{
			"projectId": projectID, "id": target.ID, "name": workspace.Manifest.Name, "description": workspace.Manifest.Description,
			"ontologyIds": writeIDs, "readOnlyOntologyIds": readOnlyIDs, "functionIds": functionIDs,
		}})
		if err != nil {
			return nil, err
		}
	}
	if draftOnly {
		return plan, nil
	}
	version, err := session.Client.PromoteObject(ctx, projectID, plan.AppID, "app", message)
	if err != nil {
		return nil, err
	}
	plan.Version = version.Version
	return plan, nil
}

func resolveAppFolderBindings(ctx context.Context, client *platform.Client, projectID string, manifest appfolder.Manifest) ([]string, []string, []string, error) {
	writeIDs := []string{}
	readOnlyIDs := []string{}
	functionIDs := []string{}
	if len(manifest.Ontologies)+len(manifest.ReadOnlyOntologies) > 0 {
		entities, err := client.Ontologies(ctx, projectID)
		if err != nil {
			return nil, nil, nil, err
		}
		lookup := map[string]string{}
		for _, entity := range entities {
			lookup[entity.APIName] = entity.ID
			lookup[entity.ID] = entity.ID
		}
		seen := map[string]bool{}
		for _, group := range []struct {
			names []string
			ids   *[]string
		}{{manifest.Ontologies, &writeIDs}, {manifest.ReadOnlyOntologies, &readOnlyIDs}} {
			for _, name := range group.names {
				id := lookup[name]
				if id == "" {
					return nil, nil, nil, fmt.Errorf("app.json: ontology %q was not found in the selected project", name)
				}
				if seen[id] {
					return nil, nil, nil, fmt.Errorf("app.json: ontology %q is bound more than once", name)
				}
				seen[id] = true
				*group.ids = append(*group.ids, id)
			}
		}
	}
	if len(manifest.Functions) > 0 {
		functions, err := client.Functions(ctx, projectID, []string{"id", "name"})
		if err != nil {
			return nil, nil, nil, err
		}
		lookup := map[string]string{}
		for _, function := range functions {
			lookup[function.Name] = function.ID
			lookup[function.ID] = function.ID
		}
		seen := map[string]bool{}
		for _, name := range manifest.Functions {
			id := lookup[name]
			if id == "" {
				return nil, nil, nil, fmt.Errorf("app.json: function %q was not found in the selected project", name)
			}
			if seen[id] {
				return nil, nil, nil, fmt.Errorf("app.json: function %q is bound more than once", name)
			}
			seen[id] = true
			functionIDs = append(functionIDs, id)
		}
	}
	return writeIDs, readOnlyIDs, functionIDs, nil
}
