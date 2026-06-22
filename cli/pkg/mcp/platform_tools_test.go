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
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/clidey/whodb/cli/internal/config"
	platformapi "github.com/clidey/whodb/cli/internal/platform"
)

type fakePlatformClient struct {
	projectSourcesOrgID     string
	projectSourcesProjectID string
	sourceRowsOrgID         string
	sourceRowsProjectID     string
	rowsLimit               int
	rowsOffset              int
	createdSourceName       string
	updatedSourceName       string
	deletedSourceID         string
}

func (f *fakePlatformClient) Me(context.Context) (*platformapi.User, error) {
	return &platformapi.User{ID: "user-1", Email: "ada@example.com"}, nil
}

func (f *fakePlatformClient) PlatformManifest(context.Context) (*platformapi.PlatformManifest, error) {
	return &platformapi.PlatformManifest{PlatformVersion: "1.2.3", ManifestProtocolVersion: "1"}, nil
}

func (f *fakePlatformClient) Organizations(context.Context) ([]platformapi.Organization, error) {
	return []platformapi.Organization{
		{ID: "org-1", Name: "Clidey", Slug: "clidey"},
		{ID: "org-2", Name: "Acme", Slug: "acme"},
	}, nil
}

func (f *fakePlatformClient) Projects(ctx context.Context, orgID string) ([]platformapi.Project, error) {
	return []platformapi.Project{
		{ID: "proj-1", OrgID: orgID, Name: "Customer", Slug: "customer", Description: "Customer data"},
		{ID: "proj-2", OrgID: orgID, Name: "Internal", Slug: "internal"},
	}, nil
}

func (f *fakePlatformClient) ProjectSources(ctx context.Context, orgID, projectID string) ([]platformapi.Source, error) {
	f.projectSourcesOrgID = orgID
	f.projectSourcesProjectID = projectID
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
	f.sourceRowsOrgID = orgID
	f.sourceRowsProjectID = projectID
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
	if len(tools) != 42 {
		t.Fatalf("len(platformToolDefinitions()) = %d, want 42", len(tools))
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
	if output.OrgID != "org-1" || output.ProjectID != "proj-1" || client.projectSourcesOrgID != "org-1" || client.projectSourcesProjectID != "proj-1" {
		t.Fatalf("output/client scope = %#v client=%q/%q, want selected workspace", output, client.projectSourcesOrgID, client.projectSourcesProjectID)
	}
}

func TestHandlePlatformSourcesReportsMissingWorkspaceAction(t *testing.T) {
	client := &fakePlatformClient{}
	withPlatformSessionLoader(t, func(context.Context) (*platformToolSession, error) {
		session := testPlatformSession(client)
		session.Host.DefaultOrgID = ""
		session.Host.DefaultOrgName = ""
		session.Host.DefaultProjectID = ""
		session.Host.DefaultProjectName = ""
		return session, nil
	})

	_, output, err := HandlePlatformSources(context.Background(), nil, PlatformSourcesInput{})
	if err != nil {
		t.Fatalf("HandlePlatformSources() error = %v", err)
	}
	if !strings.Contains(output.Error, "whodb-cli use --org <org> --project <project>") {
		t.Fatalf("output.Error = %q, want whodb-cli use action", output.Error)
	}
}

func TestHandlePlatformOrgsMarksSelectedOrg(t *testing.T) {
	client := &fakePlatformClient{}
	withPlatformSessionLoader(t, func(context.Context) (*platformToolSession, error) {
		return testPlatformSession(client), nil
	})

	_, output, err := HandlePlatformOrgs(context.Background(), nil, PlatformOrgsInput{})
	if err != nil {
		t.Fatalf("HandlePlatformOrgs() error = %v", err)
	}
	if output.Error != "" {
		t.Fatalf("HandlePlatformOrgs() output error = %q", output.Error)
	}
	if len(output.Orgs) != 2 {
		t.Fatalf("orgs = %#v, want two orgs", output.Orgs)
	}
	if !output.Orgs[0].Selected || output.Orgs[1].Selected {
		t.Fatalf("org selection = %#v, want org-1 selected", output.Orgs)
	}
}

func TestHandlePlatformProjectsUsesSelectedOrg(t *testing.T) {
	client := &fakePlatformClient{}
	withPlatformSessionLoader(t, func(context.Context) (*platformToolSession, error) {
		return testPlatformSession(client), nil
	})

	_, output, err := HandlePlatformProjects(context.Background(), nil, PlatformProjectsInput{})
	if err != nil {
		t.Fatalf("HandlePlatformProjects() error = %v", err)
	}
	if output.Error != "" {
		t.Fatalf("HandlePlatformProjects() output error = %q", output.Error)
	}
	if output.OrgID != "org-1" || output.OrgName != "Clidey" {
		t.Fatalf("project output org = %#v, want selected org", output)
	}
	if len(output.Projects) != 2 {
		t.Fatalf("projects = %#v, want two projects", output.Projects)
	}
	if !output.Projects[0].Selected || output.Projects[1].Selected {
		t.Fatalf("project selection = %#v, want proj-1 selected", output.Projects)
	}
}

func TestHandlePlatformProjectsAcceptsOrgName(t *testing.T) {
	client := &fakePlatformClient{}
	withPlatformSessionLoader(t, func(context.Context) (*platformToolSession, error) {
		return testPlatformSession(client), nil
	})

	_, output, err := HandlePlatformProjects(context.Background(), nil, PlatformProjectsInput{Org: "Acme"})
	if err != nil {
		t.Fatalf("HandlePlatformProjects() error = %v", err)
	}
	if output.Error != "" {
		t.Fatalf("HandlePlatformProjects() output error = %q", output.Error)
	}
	if output.OrgID != "org-2" || output.OrgName != "Acme" {
		t.Fatalf("project output org = %#v, want Acme", output)
	}
}

func TestAutoSelectPlatformToolWorkspaceSelectsOnlyOrgAndProject(t *testing.T) {
	client := &singleWorkspacePlatformClient{}
	host := config.PlatformHost{URL: "https://app.whodb.com"}

	messages, changed, err := autoSelectPlatformToolWorkspace(context.Background(), client, &host)
	if err != nil {
		t.Fatalf("autoSelectPlatformToolWorkspace() error = %v", err)
	}
	if !changed || len(messages) != 2 {
		t.Fatalf("changed/messages = %v/%#v, want two auto-selection messages", changed, messages)
	}
	if host.DefaultOrgID != "org-only" || host.DefaultOrgName != "Only Org" || host.DefaultProjectID != "project-only" || host.DefaultProjectName != "Only Project" {
		t.Fatalf("host = %#v, want only workspace selected", host)
	}
}

func TestHandlePlatformStatusReportsAutoSelection(t *testing.T) {
	client := &fakePlatformClient{}
	withPlatformSessionLoader(t, func(context.Context) (*platformToolSession, error) {
		session := testPlatformSession(client)
		session.AutoSelected = []string{"Selected the only available organization: Clidey"}
		return session, nil
	})

	_, output, err := HandlePlatformStatus(context.Background(), nil, PlatformStatusInput{})
	if err != nil {
		t.Fatalf("HandlePlatformStatus() error = %v", err)
	}
	if len(output.AutoSelected) != 1 || output.AutoSelected[0] != "Selected the only available organization: Clidey" {
		t.Fatalf("AutoSelected = %#v, want auto-selection message", output.AutoSelected)
	}
}

func TestHandlePlatformSourceTypes(t *testing.T) {
	client := &fakePlatformClient{}
	withPlatformSessionLoader(t, func(context.Context) (*platformToolSession, error) {
		return testPlatformSession(client), nil
	})

	_, output, err := HandlePlatformSourceTypes(context.Background(), nil, PlatformSourceTypesInput{})
	if err != nil {
		t.Fatalf("HandlePlatformSourceTypes() error = %v", err)
	}
	if output.Error != "" {
		t.Fatalf("HandlePlatformSourceTypes() output error = %q", output.Error)
	}
	if len(output.SourceTypes) != 1 || output.SourceTypes[0].ID != "Postgres" {
		t.Fatalf("source types = %#v, want Postgres", output.SourceTypes)
	}
}

func TestHandlePlatformSourceFields(t *testing.T) {
	client := &fakePlatformClient{}
	withPlatformSessionLoader(t, func(context.Context) (*platformToolSession, error) {
		return testPlatformSession(client), nil
	})

	_, output, err := HandlePlatformSourceFields(context.Background(), nil, PlatformSourceFieldsInput{SourceType: "Postgres"})
	if err != nil {
		t.Fatalf("HandlePlatformSourceFields() error = %v", err)
	}
	if output.Error != "" {
		t.Fatalf("HandlePlatformSourceFields() output error = %q", output.Error)
	}
	if output.SourceType != "Postgres" || len(output.Fields) != 3 {
		t.Fatalf("source fields = %#v, want Postgres fields", output)
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
	if client.sourceRowsOrgID != "org-1" || client.sourceRowsProjectID != "proj-1" {
		t.Fatalf("source rows workspace = %q/%q, want selected workspace", client.sourceRowsOrgID, client.sourceRowsProjectID)
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

	pending, err := getPendingPlatformAction(output.ConfirmationToken)
	if err != nil {
		t.Fatalf("getPendingPlatformAction() error = %v", err)
	}
	if pending.CreateInput.Name != "New Warehouse" {
		t.Fatalf("pending action = %#v, want create action", pending)
	}
	if output.ConfirmationPreview == nil || output.ConfirmationPreview.SourceName != "New Warehouse" || output.ConfirmationPreview.SourceType != "Postgres" {
		t.Fatalf("confirmation preview = %#v, want source name/type", output.ConfirmationPreview)
	}
	consumePendingPlatformAction(output.ConfirmationToken)
}

func TestHandlePlatformSourceUpdateRequiresConfirmationPreview(t *testing.T) {
	client := &fakePlatformClient{}
	withPlatformSessionLoader(t, func(context.Context) (*platformToolSession, error) {
		return testPlatformSession(client), nil
	})

	_, output, err := HandlePlatformSourceUpdate(context.Background(), nil, PlatformSourceUpdateInput{
		Source:   "Warehouse",
		Name:     "Warehouse Updated",
		Database: "analytics",
		Password: "new-secret",
		Advanced: map[string]string{"api_token": "new-token"},
	})
	if err != nil {
		t.Fatalf("HandlePlatformSourceUpdate() error = %v", err)
	}
	if output.Error != "" {
		t.Fatalf("HandlePlatformSourceUpdate() output error = %q", output.Error)
	}
	if !output.ConfirmationRequired || output.ConfirmationToken == "" {
		t.Fatalf("confirmation output = %#v, want token", output)
	}
	preview := output.ConfirmationPreview
	if preview == nil {
		t.Fatal("confirmation preview is nil")
	}
	if preview.Operation != "update_source" || preview.SourceID != "src-1" || preview.SourceName != "Warehouse" || preview.SourceType != "Postgres" {
		t.Fatalf("confirmation preview = %#v, want update metadata", preview)
	}
	if !containsAll(strings.Join(preview.Changes, ","), "Database", "Password", "api_token", "name") {
		t.Fatalf("preview changes = %#v, want changed field names", preview.Changes)
	}
	previewJSON, err := json.Marshal(preview)
	if err != nil {
		t.Fatalf("json.Marshal(preview) error = %v", err)
	}
	if containsAll(string(previewJSON), "analytics") || containsAll(string(previewJSON), "new-secret") || containsAll(string(previewJSON), "new-token") {
		t.Fatalf("preview JSON leaked update values: %s", previewJSON)
	}
	consumePendingPlatformAction(output.ConfirmationToken)
}

func TestHandlePlatformSourceDeleteRequiresConfirmationPreview(t *testing.T) {
	client := &fakePlatformClient{}
	withPlatformSessionLoader(t, func(context.Context) (*platformToolSession, error) {
		return testPlatformSession(client), nil
	})

	_, output, err := HandlePlatformSourceDelete(context.Background(), nil, PlatformSourceDeleteInput{Source: "Warehouse"})
	if err != nil {
		t.Fatalf("HandlePlatformSourceDelete() error = %v", err)
	}
	if output.Error != "" {
		t.Fatalf("HandlePlatformSourceDelete() output error = %q", output.Error)
	}
	if !output.ConfirmationRequired || output.ConfirmationToken == "" {
		t.Fatalf("confirmation output = %#v, want token", output)
	}
	preview := output.ConfirmationPreview
	if preview == nil {
		t.Fatal("confirmation preview is nil")
	}
	if preview.Operation != "delete_source" || preview.SourceID != "src-1" || preview.SourceName != "Warehouse" || preview.SourceType != "Postgres" {
		t.Fatalf("confirmation preview = %#v, want delete metadata", preview)
	}
	if len(preview.Changes) != 1 || preview.Changes[0] != "delete source" {
		t.Fatalf("preview changes = %#v, want delete source", preview.Changes)
	}
	consumePendingPlatformAction(output.ConfirmationToken)
}

func TestHandlePlatformPendingListsPendingActions(t *testing.T) {
	clearPendingPlatformActions(t)
	token, expiresAt := storePendingPlatformAction(&PendingPlatformAction{
		Operation:   "delete_source",
		Host:        "https://app.whodb.com",
		OrgID:       "org-1",
		ProjectID:   "proj-1",
		ProjectName: "Customer",
		SourceID:    "src-1",
		SourceName:  "Warehouse",
		SourceType:  "Postgres",
		Changes:     []string{"delete source"},
	})

	_, output, err := HandlePlatformPending(context.Background(), nil, PlatformPendingInput{})
	if err != nil {
		t.Fatalf("HandlePlatformPending() error = %v", err)
	}
	if output.Error != "" {
		t.Fatalf("HandlePlatformPending() output error = %q", output.Error)
	}
	if len(output.Pending) != 1 {
		t.Fatalf("pending = %#v, want one action", output.Pending)
	}
	pending := output.Pending[0]
	if pending.Token != token || pending.ExpiresAt != expiresAt.UTC().Format(time.RFC3339) {
		t.Fatalf("pending = %#v, want token/expiry", pending)
	}
	if pending.Action.Operation != "delete_source" || pending.Action.SourceName != "Warehouse" {
		t.Fatalf("pending action = %#v, want preview metadata", pending.Action)
	}
}

func TestHandleConfirmExecutesPlatformDelete(t *testing.T) {
	client := &fakePlatformClient{}
	withPlatformSessionLoader(t, func(context.Context) (*platformToolSession, error) {
		return testPlatformSession(client), nil
	})
	token, _ := storePendingPlatformAction(&PendingPlatformAction{
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

func clearPendingPlatformActions(t *testing.T) {
	t.Helper()
	platformPendingMutex.Lock()
	pendingPlatformActions = map[string]*PendingPlatformAction{}
	platformPendingMutex.Unlock()
	t.Cleanup(func() {
		platformPendingMutex.Lock()
		pendingPlatformActions = map[string]*PendingPlatformAction{}
		platformPendingMutex.Unlock()
	})
}

type singleWorkspacePlatformClient struct {
	fakePlatformClient
}

func (f *singleWorkspacePlatformClient) Organizations(context.Context) ([]platformapi.Organization, error) {
	return []platformapi.Organization{{ID: "org-only", Name: "Only Org", Slug: "only-org"}}, nil
}

func (f *singleWorkspacePlatformClient) Projects(ctx context.Context, orgID string) ([]platformapi.Project, error) {
	return []platformapi.Project{{ID: "project-only", OrgID: orgID, Name: "Only Project", Slug: "only-project"}}, nil
}

func containsAll(value string, parts ...string) bool {
	for _, part := range parts {
		if !strings.Contains(value, part) {
			return false
		}
	}
	return true
}
