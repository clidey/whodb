/*
 * Copyright 2026 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 */

package platform

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// AppFile is one draft file of a hosted app.
type AppFile struct {
	Path      string `json:"path"`
	Content   string `json:"content"`
	UpdatedAt string `json:"updatedAt"`
}

// AppFileQuerier is the hosted platform read API used to page through app files.
type AppFileQuerier interface {
	PlatformQuery(context.Context, string, map[string]any) (any, error)
}

// ReadAppFilesByPage returns an app's draft files sorted by path without
// fetching them in one response, which large apps overflow. It reads app.json
// and shared/ files from the draft app view, then each page's files with one
// AppPageFiles request per page. Files outside app.json, shared/, and
// pages/<page>/ are not served by those queries and are not returned. Apps whose
// package hides their source return no files, as AppFiles does.
func ReadAppFilesByPage(ctx context.Context, client AppFileQuerier, projectID, appID string) ([]AppFile, error) {
	detail, err := client.PlatformQuery(ctx, "AppDetail", map[string]any{"projectId": projectID, "id": appID})
	if err != nil {
		return nil, err
	}
	var app struct {
		SourceHidden bool `json:"sourceHidden"`
	}
	if err := remarshalAppData(detail, &app); err != nil {
		return nil, fmt.Errorf("decode app detail: %w", err)
	}
	if app.SourceHidden {
		return []AppFile{}, nil
	}

	data, err := client.PlatformQuery(ctx, "AppView", map[string]any{"projectId": projectID, "id": appID, "env": "dev"})
	if err != nil {
		return nil, err
	}
	var view struct {
		Pages []string  `json:"pages"`
		Files []AppFile `json:"files"`
	}
	if err := remarshalAppData(data, &view); err != nil {
		return nil, fmt.Errorf("decode app view: %w", err)
	}
	byPath := make(map[string]AppFile, len(view.Files))
	for _, file := range view.Files {
		if !strings.HasPrefix(strings.TrimPrefix(file.Path, "/"), "pages/") {
			byPath[file.Path] = file
		}
	}
	for _, page := range view.Pages {
		data, err := client.PlatformQuery(ctx, "AppPageFiles", map[string]any{"projectId": projectID, "id": appID, "env": "dev", "page": page})
		if err != nil {
			return nil, fmt.Errorf("read app page %q: %w", page, err)
		}
		var files []AppFile
		if err := remarshalAppData(data, &files); err != nil {
			return nil, fmt.Errorf("decode app page %q: %w", page, err)
		}
		for _, file := range files {
			byPath[file.Path] = file
		}
	}
	files := make([]AppFile, 0, len(byPath))
	for _, file := range byPath {
		files = append(files, file)
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}

func remarshalAppData(data any, target any) error {
	raw, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, target)
}
