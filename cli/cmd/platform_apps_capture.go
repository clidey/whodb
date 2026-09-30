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
	"strings"

	"github.com/clidey/whodb/cli/internal/appcapture"
	"github.com/clidey/whodb/cli/internal/platform"
	"github.com/clidey/whodb/cli/pkg/output"
	"github.com/spf13/cobra"
)

var (
	appCaptureEnv         string
	appCapturePage        string
	appCaptureTab         string
	appCaptureOutput      string
	appCaptureWidth       int
	appCaptureHeight      int
	appCaptureInstall     bool
	appCaptureClicks      []string
	appCaptureActionsJSON string
	appCaptureJS          string
	appCaptureScriptFile  string
)

var appsSetupCaptureCmd = &cobra.Command{
	Use:           "setup-capture",
	Short:         "Install the optional app screenshot browser in your user cache",
	Args:          cobra.NoArgs,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, _ []string) error {
		packageFile, err := appcapture.SetupRuntime(cmd.Context())
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(cmd.OutOrStdout(), "App capture is ready (%s)\n", packageFile)
		return err
	},
}

var appsViewsCmd = &cobra.Command{
	Use:           "views <app>",
	Short:         "List the pages available in a hosted app",
	Args:          cobra.ExactArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runPlatformAppViews,
}

var appsScreenshotCmd = &cobra.Command{
	Use:           "screenshot <app>",
	Short:         "Capture the rendered hosted app as a WebP image",
	Args:          cobra.ExactArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runPlatformAppScreenshot,
}

func registerPlatformAppCaptureCommands() {
	appsViewsCmd.Flags().StringVar(&appCaptureEnv, "env", "published", "app environment: published or dev")
	appsScreenshotCmd.Flags().StringVar(&appCaptureEnv, "env", "published", "app environment: published or dev")
	appsScreenshotCmd.Flags().StringVar(&appCapturePage, "page", "", "app page from the views command")
	appsScreenshotCmd.Flags().StringVar(&appCaptureTab, "tab", "", "visible app tab or button label to open before capture")
	appsScreenshotCmd.Flags().StringVar(&appCaptureOutput, "output", "", "output WebP file (defaults to the app name and page)")
	appsScreenshotCmd.Flags().IntVar(&appCaptureWidth, "width", 1440, "capture viewport width in pixels")
	appsScreenshotCmd.Flags().IntVar(&appCaptureHeight, "height", 900, "capture viewport height in pixels")
	appsScreenshotCmd.Flags().BoolVar(&appCaptureInstall, "install", false, "install the optional capture browser before screenshotting")
	appsScreenshotCmd.Flags().StringArrayVar(&appCaptureClicks, "click", nil, "click a Playwright selector in the app; repeat for sequential clicks")
	appsScreenshotCmd.Flags().StringVar(&appCaptureActionsJSON, "actions", "", "ordered JSON array of click, fill, press, wait_for, or wait actions")
	appsScreenshotCmd.Flags().StringVar(&appCaptureJS, "js", "", "JavaScript expression or IIFE to evaluate in the app frame")
	appsScreenshotCmd.Flags().StringVar(&appCaptureScriptFile, "script-file", "", "read JavaScript to evaluate from a local file")
	appsCmd.AddCommand(appsViewsCmd, appsScreenshotCmd, appsSetupCaptureCmd)
}

func resolveAppCapture(ctx context.Context, appRef string) (*platformSession, appcapture.Options, string, error) {
	session, err := loadPlatformSession(ctx, platformHost)
	if err != nil {
		return nil, appcapture.Options{}, "", err
	}
	org, project, err := resolvePlatformProject(ctx, session, platformResourceOrg, platformResourceProject)
	if err != nil {
		return nil, appcapture.Options{}, "", err
	}
	session.Client.SetWorkspaceContext(org.ID, project.ID)
	apps, err := appcapture.ListApps(ctx, session.Client, project.ID)
	if err != nil {
		return nil, appcapture.Options{}, "", err
	}
	app, err := appcapture.FindApp(apps, appRef)
	if err != nil {
		return nil, appcapture.Options{}, "", err
	}
	return session, appcapture.Options{
		Host: session.Host.URL, OrgSlug: org.Slug, ProjectSlug: project.Slug,
		OrgID: org.ID, ProjectID: project.ID, AppID: app.ID, Env: appCaptureEnv,
	}, app.Name, nil
}

func runPlatformAppViews(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	session, options, appName, err := resolveAppCapture(ctx, args[0])
	if err != nil {
		return err
	}
	views, err := appcapture.ListViews(ctx, session.Client, options)
	if err != nil {
		return err
	}
	format, err := output.ParseFormat(platformFormat)
	if err != nil {
		return err
	}
	if format == output.FormatJSON {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{"app": appName, "views": views})
	}
	for _, view := range views {
		fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\n", view.Page, view.URL)
	}
	return nil
}

func runPlatformAppScreenshot(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	if appCaptureInstall {
		if _, err := appcapture.SetupRuntime(ctx); err != nil {
			return err
		}
	}
	session, options, appName, err := resolveAppCapture(ctx, args[0])
	if err != nil {
		return err
	}
	options.Page = appCapturePage
	options.Tab = appCaptureTab
	options.Width = appCaptureWidth
	options.Height = appCaptureHeight
	if appCaptureJS != "" && appCaptureScriptFile != "" {
		return fmt.Errorf("use either --js or --script-file")
	}
	for _, selector := range appCaptureClicks {
		options.Actions = append(options.Actions, appcapture.Action{Kind: "click", Selector: selector})
	}
	if appCaptureActionsJSON != "" {
		var actions []appcapture.Action
		if err := json.Unmarshal([]byte(appCaptureActionsJSON), &actions); err != nil {
			return fmt.Errorf("decode --actions JSON: %w", err)
		}
		options.Actions = append(options.Actions, actions...)
	}
	options.Script = appCaptureJS
	if appCaptureScriptFile != "" {
		contents, err := os.ReadFile(appCaptureScriptFile)
		if err != nil {
			return fmt.Errorf("read JavaScript file: %w", err)
		}
		options.Script = string(contents)
	}
	tokenSource := platform.NewOIDCTokenSource(session.Host.URL, session.Host.AccountID, session.Config)
	options.Token, err = tokenSource.Token(ctx)
	if err != nil {
		return fmt.Errorf("read hosted login token: %w", err)
	}
	capture, err := appcapture.Capture(ctx, options)
	if err != nil {
		return err
	}
	filename := appCaptureOutput
	if filename == "" {
		filename = strings.ToLower(strings.ReplaceAll(appName, " ", "-"))
		if options.Page != "" {
			filename += "-" + options.Page
		}
		filename += ".webp"
	}
	if strings.ToLower(filepath.Ext(filename)) != ".webp" {
		return fmt.Errorf("output must have a .webp extension")
	}
	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return fmt.Errorf("create screenshot: %w", err)
	}
	defer file.Close()
	if _, err := file.Write(capture.Image); err != nil {
		return fmt.Errorf("write screenshot: %w", err)
	}
	format, err := output.ParseFormat(platformFormat)
	if err != nil {
		return err
	}
	checksPassed := len(capture.Diagnostics.ActionErrors) == 0 && len(capture.Diagnostics.ScriptErrors) == 0
	if format == output.FormatJSON {
		if err := json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{
			"app": appName, "path": filename, "width": capture.Width,
			"height": capture.Height, "page": capture.Page, "tab": capture.Tab,
			"visible_text": capture.Text, "diagnostics": capture.Diagnostics,
			"script_result_json": capture.ScriptResultJSON, "checks_passed": checksPassed,
		}); err != nil {
			return err
		}
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "Saved %s (%d×%d)\n", filename, capture.Width, capture.Height)
		fmt.Fprintf(cmd.OutOrStdout(), "Browser errors: %d console, %d page, %d requests, %d HTTP responses; checks: %d action, %d script errors\n",
			len(capture.Diagnostics.ConsoleErrors), len(capture.Diagnostics.PageErrors),
			len(capture.Diagnostics.FailedRequests), len(capture.Diagnostics.HTTPResponses),
			len(capture.Diagnostics.ActionErrors), len(capture.Diagnostics.ScriptErrors))
	}
	if !checksPassed {
		return fmt.Errorf("capture checks failed; screenshot saved to %s, inspect diagnostics", filename)
	}
	return nil
}
