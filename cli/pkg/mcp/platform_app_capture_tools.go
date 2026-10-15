/*
 * Copyright 2026 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 */

package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/clidey/whodb/cli/internal/appcapture"
	"github.com/clidey/whodb/cli/internal/config"
	platformapi "github.com/clidey/whodb/cli/internal/platform"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func platformCaptureTarget(ctx context.Context, input PlatformAppCaptureInput) (*platformToolSession, appcapture.Options, string, error) {
	if input.Mobile && input.Tablet {
		return nil, appcapture.Options{}, "", fmt.Errorf("mobile and tablet modes cannot be combined")
	}
	if strings.TrimSpace(input.App) == "" {
		return nil, appcapture.Options{}, "", fmt.Errorf("app is required")
	}
	session, err := loadPlatformToolSession(ctx)
	if err != nil {
		return nil, appcapture.Options{}, "", err
	}
	orgID, projectID := session.Host.DefaultOrgID, session.Host.DefaultProjectID
	if orgID == "" || projectID == "" {
		return nil, appcapture.Options{}, "", fmt.Errorf("select a hosted organization and project first")
	}
	org, err := cachedOrganizationByID(ctx, session.Client, session.Host.URL, session.Host.AccountID, orgID)
	if err != nil {
		return nil, appcapture.Options{}, "", err
	}
	if org == nil || org.Slug == "" {
		return nil, appcapture.Options{}, "", fmt.Errorf("selected organization is not available")
	}
	orgSlug := org.Slug
	project, err := cachedProjectByID(ctx, session.Client, session.Host.URL, session.Host.AccountID, orgID, projectID)
	if err != nil {
		return nil, appcapture.Options{}, "", err
	}
	if project == nil || project.Slug == "" {
		return nil, appcapture.Options{}, "", fmt.Errorf("selected project is not available")
	}
	projectSlug := project.Slug
	apps, err := appcapture.ListApps(ctx, session.Client, projectID)
	if err != nil {
		return nil, appcapture.Options{}, "", err
	}
	app, err := appcapture.FindApp(apps, input.App)
	if err != nil {
		return nil, appcapture.Options{}, "", err
	}
	env := strings.TrimSpace(input.Env)
	if env == "" {
		env = "published"
	}
	options := appcapture.Options{
		Host: session.Host.URL, OrgSlug: orgSlug, ProjectSlug: projectSlug,
		OrgID: orgID, ProjectID: projectID, AppID: app.ID, Env: env,
		Page: input.Page, Tab: input.Tab, Actions: input.Actions, Script: input.Script, Width: input.Width, Height: input.Height, Mobile: input.Mobile, Tablet: input.Tablet, Orientation: input.Orientation,
	}
	options.Width, options.Height, err = appcapture.ResolveViewport(options.Width, options.Height, options.Mobile, options.Tablet, options.Orientation)
	if err != nil {
		return nil, appcapture.Options{}, "", err
	}
	if _, err := options.URL(); err != nil {
		return nil, appcapture.Options{}, "", err
	}
	return session, options, app.Name, nil
}

func handlePlatformAppViews(ctx context.Context, input PlatformAppCaptureInput) (*mcp.CallToolResult, PlatformAppViewsOutput, error) {
	requestID := generateRequestID("whodb_platform_app_views")
	session, options, appName, err := platformCaptureTarget(ctx, input)
	if err != nil {
		return nil, PlatformAppViewsOutput{Error: err.Error(), RequestID: requestID}, nil
	}
	views, err := appcapture.ListViews(ctx, session.Client, options)
	if err != nil {
		return nil, PlatformAppViewsOutput{Error: err.Error(), RequestID: requestID}, nil
	}
	return nil, PlatformAppViewsOutput{App: appName, AppID: options.AppID, Env: options.Env, Views: views, RequestID: requestID}, nil
}

func handlePlatformAppScreenshot(ctx context.Context, input PlatformAppCaptureInput) (*mcp.CallToolResult, PlatformAppScreenshotOutput, error) {
	requestID := generateRequestID("whodb_platform_app_screenshot")
	_, options, appName, err := platformCaptureTarget(ctx, input)
	if err != nil {
		return nil, PlatformAppScreenshotOutput{Error: err.Error(), RequestID: requestID}, nil
	}
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, PlatformAppScreenshotOutput{Error: err.Error(), RequestID: requestID}, nil
	}
	// The selected host's account ID identifies the keyring entry, not its URL.
	sessionHost, ok := cfg.GetPlatformHost(options.Host)
	if !ok {
		return nil, PlatformAppScreenshotOutput{Error: "hosted login was not found", RequestID: requestID}, nil
	}
	tokenSource := platformapi.NewOIDCTokenSource(options.Host, sessionHost.AccountID, cfg)
	options.Token, err = tokenSource.Token(ctx)
	if err != nil {
		return nil, PlatformAppScreenshotOutput{Error: err.Error(), RequestID: requestID}, nil
	}
	captured, err := appcapture.Capture(ctx, options)
	if err != nil {
		return nil, PlatformAppScreenshotOutput{Error: err.Error(), RequestID: requestID}, nil
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.ImageContent{Data: captured.Image, MIMEType: captured.MIMEType}}},
		PlatformAppScreenshotOutput{App: appName, AppID: options.AppID, Env: options.Env, Page: captured.Page, Tab: captured.Tab, Width: captured.Width, Height: captured.Height, Mobile: options.Mobile, Tablet: options.Tablet, Orientation: options.Orientation, MIMEType: captured.MIMEType, VisibleText: captured.Text, Diagnostics: captured.Diagnostics, ScriptResultJSON: captured.ScriptResultJSON, ChecksPassed: len(captured.Diagnostics.ActionErrors) == 0 && len(captured.Diagnostics.ScriptErrors) == 0, RequestID: requestID}, nil
}
