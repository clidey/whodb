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
	orgs, err := session.Client.Organizations(ctx)
	if err != nil {
		return nil, appcapture.Options{}, "", err
	}
	orgSlug := ""
	for _, org := range orgs {
		if org.ID == orgID {
			orgSlug = org.Slug
			break
		}
	}
	if orgSlug == "" {
		return nil, appcapture.Options{}, "", fmt.Errorf("selected organization is not available")
	}
	projects, err := session.Client.Projects(ctx, orgID)
	if err != nil {
		return nil, appcapture.Options{}, "", err
	}
	projectSlug := ""
	for _, project := range projects {
		if project.ID == projectID {
			projectSlug = project.Slug
			break
		}
	}
	if projectSlug == "" {
		return nil, appcapture.Options{}, "", fmt.Errorf("selected project is not available")
	}
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
		Page: input.Page, Tab: input.Tab, Actions: input.Actions, Script: input.Script, Width: input.Width, Height: input.Height,
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
		PlatformAppScreenshotOutput{App: appName, AppID: options.AppID, Env: options.Env, Page: captured.Page, Tab: captured.Tab, Width: captured.Width, Height: captured.Height, MIMEType: captured.MIMEType, VisibleText: captured.Text, Diagnostics: captured.Diagnostics, ScriptResultJSON: captured.ScriptResultJSON, ChecksPassed: len(captured.Diagnostics.ActionErrors) == 0 && len(captured.Diagnostics.ScriptErrors) == 0, RequestID: requestID}, nil
}
