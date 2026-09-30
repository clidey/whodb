/*
 * Copyright 2026 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 */

package appcapture

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// QueryClient is the hosted platform read API used to discover apps and pages.
type QueryClient interface {
	PlatformQuery(context.Context, string, map[string]any) (any, error)
}

// ListApps returns the apps in one authorized project.
func ListApps(ctx context.Context, client QueryClient, projectID string) ([]App, error) {
	data, err := client.PlatformQuery(ctx, "ProjectApps", map[string]any{"projectId": projectID})
	if err != nil {
		return nil, err
	}
	var apps []App
	if err := decode(data, &apps); err != nil {
		return nil, fmt.Errorf("decode apps: %w", err)
	}
	return apps, nil
}

// FindApp selects a project app by ID or case-insensitive exact name.
func FindApp(apps []App, ref string) (*App, error) {
	ref = strings.TrimSpace(ref)
	for i := range apps {
		if apps[i].ID == ref {
			return &apps[i], nil
		}
	}
	for i := range apps {
		if strings.EqualFold(apps[i].Name, ref) {
			return &apps[i], nil
		}
	}
	return nil, fmt.Errorf("app %q was not found in the selected project", ref)
}

// ListViews returns the pages in the actual hosted app view.
func ListViews(ctx context.Context, client QueryClient, options Options) ([]View, error) {
	variables := map[string]any{"projectId": options.ProjectID, "id": options.AppID}
	if options.Env == "dev" {
		variables["env"] = "dev"
	}
	data, err := client.PlatformQuery(ctx, "AppView", variables)
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, fmt.Errorf("app is not available in %s environment", options.Env)
	}
	var appView struct {
		Pages []string `json:"pages"`
	}
	if err := decode(data, &appView); err != nil {
		return nil, fmt.Errorf("decode app view: %w", err)
	}
	if len(appView.Pages) == 0 {
		appView.Pages = []string{"main"}
	}
	views := make([]View, 0, len(appView.Pages))
	for _, name := range appView.Pages {
		viewOptions := options
		viewOptions.Page = name
		viewOptions.Tab = ""
		viewURL, err := viewOptions.URL()
		if err != nil {
			return nil, err
		}
		views = append(views, View{Page: name, URL: viewURL})
	}
	return views, nil
}

func decode(value any, target any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, target)
}

// ValidateHost rejects non-HTTP origins before a browser is started.
func ValidateHost(host string) error {
	parsed, err := url.Parse(host)
	if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Host == "" {
		return fmt.Errorf("invalid hosted WhoDB URL")
	}
	return nil
}
