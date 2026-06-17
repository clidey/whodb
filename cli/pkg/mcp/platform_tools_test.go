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

package mcp

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/clidey/whodb/cli/internal/config"
	platformapi "github.com/clidey/whodb/cli/internal/platform"
)

type fakePlatformClient struct {
	sourcesRowsProjectID string
	rowsLimit            int
	rowsOffset           int
	createdSourceName    string
	updatedSourceName    string
	deletedSourceID      string
}

func (f *fakePlatformClient) Me(context.Context) (*platformapi.User, error) {
	return &platformapi.User{ID: "user-1", Email: "ada@example.com"}, nil
}

func (f *fakePlatformClient) PlatformManifest(context.Context) (*platformapi.PlatformManifest, error) {
	return &platformapi.PlatformManifest{PlatformVersion: "1.2.3", ManifestProtocolVersion: "1"}, nil
}

func (f *fakePlatformClient) ProjectSources(ctx context.Context, orgID, projectID string) ([]platformapi.Source, error) {
	f.sourcesRowsProjectID = projectID
	return []platformapi.Source{
		{ID: "src-1", ProjectID: projectID, Name: "Warehouse", DatabaseType: "Postgres"},
	}, nil
}

func (f *fakePlatformClient) SourceTypes(context.Context) ([]platformapi.SourceType, error) {
	return []platformapi.SourceType{{
		ID:        "Postgres",
		Label:     "Postgres",
		Connector: "Postgres",
		ConnectionFields: []platformapi.SourceConnectionField{
			{Key: "Hostname", Kind: "Text", Required: true},
			{Key: "Password", Kind: "Password", Required: true},
			{Key: "SSL Mode", Kind: "Text"},
		},
	}}, nil
}

func (f *fakePlatformClient) SourceConfig(ctx context.Context, orgID, projectID, sourceID string) (*platformapi.SourceConfig, error) {
	return &platformapi.SourceConfig{
		Hostname: "localhost",
		Password: "secret",
		Database: "postgres",
		Advanced: map[string]string{"SSL Mode": "require", "api_token": "token"},
	}, nil
}

func (f *fakePlatformClient) CreateSource(ctx context.Context, input platformapi.CreateSourceInput) (*platformapi.Source, error) {
	f.createdSourceName = input.Name
	return &platformapi.Source{ID: "src-created", ProjectID: input.ProjectID, Name: input.Name, DatabaseType: input.DatabaseType}, nil
}

func (f *fakePlatformClient) UpdateSource(ctx context.Context, input platformapi.UpdateSourceInput) (*platformapi.Source, error) {
	if input.Name != nil {
		f.updatedSourceName = *input.Name
	}
	return &platformapi.Source{ID: input.ID, ProjectID: input.ProjectID, Name: f.updatedSourceName, DatabaseType: "Postgres"}, nil
}

func (f *fakePlatformClient) TestSourceConnection(ctx context.Context, input platformapi.CreateSourceInput) error {
	return nil
}

func (f *fakePlatformClient) DeleteSource(ctx context.Context, orgID, projectID, sourceID string) error {
	f.deletedSourceID = sourceID
	return nil
}

func (f *fakePlatformClient) SourceObjects(ctx context.Context, orgID, projectID, sourceID string, parent *platformapi.SourceObjectRefInput, kinds []platformapi.SourceObjectKind, pageSize, pageOffset int) ([]platformapi.SourceObject, error) {
	return []platformapi.SourceObject{{Name: "users", Kind: "Table", Path: []string{"public", "users"}}}, nil
}

func (f *fakePlatformClient) SourceColumns(ctx context.Context, orgID, projectID, sourceID string, ref platformapi.SourceObjectRefInput) ([]platformapi.Column, error) {
	return []platformapi.Column{{Name: "id", Type: "integer", IsPrimary: true}}, nil
}

func (f *fakePlatformClient) SourceRows(ctx context.Context, orgID, projectID, sourceID string, ref platformapi.SourceObjectRefInput, pageSize, pageOffset int) (*platformapi.RowsResult, error) {
	f.rowsLimit = pageSize
	f.rowsOffset = pageOffset
	return &platformapi.RowsResult{
		Columns:    []platformapi.Column{{Name: "id", Type: "integer"}},
		Rows:       [][]string{{"1"}},
		TotalCount: 3,
	}, nil
}

func withPlatformSessionLoader(t *testing.T, loader func(context.Context) (*platformToolSession, error)) {
	t.Helper()
	previous := loadPlatformToolSession
	loadPlatformToolSession = loader
	t.Cleanup(func() {
		loadPlatformToolSession = previous
	})
}

func testPlatformSession(client platformClient) *platformToolSession {
	return &platformToolSession{
		Host: config.PlatformHost{
			URL:                "https://app.whodb.com",
			DefaultOrgID:       "org-1",
			DefaultOrgName:     "Clidey",
			DefaultProjectID:   "proj-1",
			DefaultProjectName: "Customer",
		},
		Client: client,
	}
}

func TestPlatformToolDefinitions(t *testing.T) {
	tools := platformToolDefinitions()
	if len(tools) != 11 {
		t.Fatalf("len(platformToolDefinitions()) = %d, want 11", len(tools))
	}
	for _, tool := range tools {
		if tool.Annotations == nil {
			t.Fatalf("tool %s has no annotations", tool.Name)
		}
	}
}

func TestHandlePlatformSourcesUsesSelectedWorkspace(t *testing.T) {
	client := &fakePlatformClient{}
	withPlatformSessionLoader(t, func(context.Context) (*platformToolSession, error) {
		return testPlatformSession(client), nil
	})

	_, output, err := HandlePlatformSources(context.Background(), nil, PlatformSourcesInput{})
	if err != nil {
		t.Fatalf("HandlePlatformSources() error = %v", err)
	}
	if output.Error != "" {
		t.Fatalf("HandlePlatformSources() output error = %q", output.Error)
	}
	if output.OrgID != "org-1" || output.ProjectID != "proj-1" || client.sourcesRowsProjectID != "proj-1" {
		t.Fatalf("output/client scope = %#v project=%q, want selected workspace", output, client.sourcesRowsProjectID)
	}
}

func TestHandlePlatformSourceRowsCapsLimit(t *testing.T) {
	client := &fakePlatformClient{}
	withPlatformSessionLoader(t, func(context.Context) (*platformToolSession, error) {
		return testPlatformSession(client), nil
	})

	_, output, err := HandlePlatformSourceRows(context.Background(), nil, PlatformSourceRowsInput{
		Source: "Warehouse",
		Ref:    "table:public.users",
		Limit:  100,
		Offset: 2,
	}, &SecurityOptions{MaxRows: 10})
	if err != nil {
		t.Fatalf("HandlePlatformSourceRows() error = %v", err)
	}
	if output.Error != "" {
		t.Fatalf("HandlePlatformSourceRows() output error = %q", output.Error)
	}
	if client.rowsLimit != 10 || client.rowsOffset != 2 {
		t.Fatalf("rows limit/offset = %d/%d, want 10/2", client.rowsLimit, client.rowsOffset)
	}
	if !output.Truncated {
		t.Fatalf("output.Truncated = false, want true")
	}
}

func TestHandlePlatformSourcesReportsActionableLoginError(t *testing.T) {
	withPlatformSessionLoader(t, func(context.Context) (*platformToolSession, error) {
		return nil, errors.New("hosted WhoDB is not logged in for https://app.whodb.com. Run: whodb-cli login --host https://app.whodb.com")
	})

	_, output, err := HandlePlatformSources(context.Background(), nil, PlatformSourcesInput{})
	if err != nil {
		t.Fatalf("HandlePlatformSources() error = %v", err)
	}
	if output.Error == "" || !containsAll(output.Error, "whodb-cli login", "app.whodb.com") {
		t.Fatalf("output error = %q, want login guidance", output.Error)
	}
}

func TestHandlePlatformSourceConfigRedactsSecrets(t *testing.T) {
	client := &fakePlatformClient{}
	withPlatformSessionLoader(t, func(context.Context) (*platformToolSession, error) {
		return testPlatformSession(client), nil
	})

	_, output, err := HandlePlatformSourceConfig(context.Background(), nil, PlatformSourceConfigInput{Source: "Warehouse"})
	if err != nil {
		t.Fatalf("HandlePlatformSourceConfig() error = %v", err)
	}
	if output.Error != "" {
		t.Fatalf("HandlePlatformSourceConfig() output error = %q", output.Error)
	}
	if output.Config.Password != platformapi.RedactedValue() || output.Config.Advanced["api_token"] != platformapi.RedactedValue() {
		t.Fatalf("config = %#v, want redacted secrets", output.Config)
	}
	if output.Config.Advanced["SSL Mode"] != "require" {
		t.Fatalf("SSL Mode = %q, want visible non-secret value", output.Config.Advanced["SSL Mode"])
	}
}

func TestHandlePlatformSourceCreateRequiresConfirmation(t *testing.T) {
	client := &fakePlatformClient{}
	withPlatformSessionLoader(t, func(context.Context) (*platformToolSession, error) {
		return testPlatformSession(client), nil
	})

	_, output, err := HandlePlatformSourceCreate(context.Background(), nil, PlatformSourceCreateInput{
		SourceType: "Postgres",
		Name:       "New Warehouse",
		Hostname:   "localhost",
		Password:   "secret",
	})
	if err != nil {
		t.Fatalf("HandlePlatformSourceCreate() error = %v", err)
	}
	if output.Error != "" {
		t.Fatalf("HandlePlatformSourceCreate() output error = %q", output.Error)
	}
	if !output.ConfirmationRequired || output.ConfirmationToken == "" {
		t.Fatalf("confirmation output = %#v, want token", output)
	}

	pending, err := getPendingConfirmation(output.ConfirmationToken)
	if err != nil {
		t.Fatalf("getPendingConfirmation() error = %v", err)
	}
	if pending.PlatformAction == nil || pending.PlatformAction.CreateInput.Name != "New Warehouse" {
		t.Fatalf("pending action = %#v, want create action", pending.PlatformAction)
	}
	consumePendingConfirmation(output.ConfirmationToken)
}

func TestHandleConfirmExecutesPlatformDelete(t *testing.T) {
	client := &fakePlatformClient{}
	withPlatformSessionLoader(t, func(context.Context) (*platformToolSession, error) {
		return testPlatformSession(client), nil
	})
	token, _ := storePendingPlatformAction("delete hosted source Warehouse", &PendingPlatformAction{
		Operation:   "delete_source",
		Host:        "https://app.whodb.com",
		OrgID:       "org-1",
		ProjectID:   "proj-1",
		ProjectName: "Customer",
		SourceID:    "src-1",
		SourceName:  "Warehouse",
	})

	_, output, err := HandlePlatformConfirm(context.Background(), nil, ConfirmInput{Token: token})
	if err != nil {
		t.Fatalf("HandlePlatformConfirm() error = %v", err)
	}
	if output.Error != "" {
		t.Fatalf("HandlePlatformConfirm() output error = %q", output.Error)
	}
	if client.deletedSourceID != "src-1" {
		t.Fatalf("deleted source = %q, want src-1", client.deletedSourceID)
	}
	if output.Message == "" || len(output.Rows) != 1 {
		t.Fatalf("output = %#v, want confirmation rows", output)
	}
}

func containsAll(value string, parts ...string) bool {
	for _, part := range parts {
		if !strings.Contains(value, part) {
			return false
		}
	}
	return true
}
