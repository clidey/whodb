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

package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/clidey/whodb/cli/internal/config"
	"github.com/clidey/whodb/cli/internal/platform"
	"github.com/clidey/whodb/cli/pkg/output"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	platformHost        string
	platformFormat      string
	platformQuiet       bool
	platformNoBrowser   bool
	platformLoginYes    bool
	platformLogoutLocal bool
	useOrg              string
	useProject          string
	projectsOrg         string
	sourcesOrg          string
	sourcesProject      string
	sourceName          string
	sourceType          string
	sourceHostname      string
	sourcePort          string
	sourceUsername      string
	sourceDatabase      string
	sourceFields        []string
	sourceAdvanced      []string
	sourcePasswordEnv   string
	sourcePasswordIn    bool
	sourceDeleteYes     bool
	sourceObjectParent  string
	sourceObjectKinds   []string
	sourceObjectLimit   int
	sourceObjectOffset  int
	sourceColumnRef     string
	sourceRowsRef       string
	sourceRowsLimit     int
	sourceRowsOffset    int
	manifestRefresh     bool
)

type platformSession struct {
	Config *config.Config
	Host   config.PlatformHost
	Client *platform.Client
}

type loginOutput struct {
	Host               string `json:"host"`
	ID                 string `json:"id"`
	Email              string `json:"email"`
	DefaultOrgID       string `json:"defaultOrgId,omitempty"`
	DefaultOrgName     string `json:"defaultOrgName,omitempty"`
	DefaultProjectID   string `json:"defaultProjectId,omitempty"`
	DefaultProjectName string `json:"defaultProjectName,omitempty"`
	WorkspaceSelected  bool   `json:"workspaceSelected"`
	OrganizationCount  int    `json:"organizationCount"`
	ProjectCount       int    `json:"projectCount,omitempty"`
}

type whoamiOutput struct {
	Host           string `json:"host"`
	ID             string `json:"id"`
	Email          string `json:"email"`
	DisplayName    string `json:"displayName"`
	OrgID          string `json:"orgId,omitempty"`
	DefaultOrg     string `json:"defaultOrg,omitempty"`
	DefaultProject string `json:"defaultProject,omitempty"`
}

type manifestOutput struct {
	Host                    string                               `json:"host"`
	PlatformVersion         string                               `json:"platformVersion"`
	ManifestProtocolVersion string                               `json:"manifestProtocolVersion"`
	GeneratedAt             string                               `json:"generatedAt"`
	FetchedAt               string                               `json:"fetchedAt,omitempty"`
	Operations              []platform.PlatformManifestOperation `json:"operations"`
	Types                   []platform.PlatformManifestType      `json:"types"`
}

type platformStatusOutput struct {
	Host                      string                     `json:"host"`
	LoggedIn                  bool                       `json:"loggedIn"`
	UserID                    string                     `json:"userId,omitempty"`
	Email                     string                     `json:"email,omitempty"`
	DefaultOrgID              string                     `json:"defaultOrgId,omitempty"`
	DefaultOrgName            string                     `json:"defaultOrgName,omitempty"`
	DefaultProjectID          string                     `json:"defaultProjectId,omitempty"`
	DefaultProjectName        string                     `json:"defaultProjectName,omitempty"`
	WorkspaceSelected         bool                       `json:"workspaceSelected"`
	PlatformVersion           string                     `json:"platformVersion,omitempty"`
	ManifestProtocolVersion   string                     `json:"manifestProtocolVersion,omitempty"`
	ManifestFetchedAt         string                     `json:"manifestFetchedAt,omitempty"`
	ManifestAvailable         bool                       `json:"manifestAvailable"`
	SourceManagementSupported bool                       `json:"sourceManagementSupported"`
	OrganizationCount         int                        `json:"organizationCount"`
	ProjectCount              int                        `json:"projectCount,omitempty"`
	Capabilities              []platformCapabilityStatus `json:"capabilities"`
}

type platformCapabilityStatus struct {
	Name      string `json:"name"`
	Operation string `json:"operation"`
	Supported bool   `json:"supported"`
}

type platformWorkspaceSelection struct {
	Orgs     []platform.Organization
	Projects []platform.Project
	Messages []string
	Changed  bool
}

var loginCmd = &cobra.Command{
	Use:           "login",
	Short:         "Sign in to hosted WhoDB",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		format, err := output.ParseFormat(platformFormat)
		if err != nil {
			return err
		}
		quiet := platformQuiet || format == output.FormatJSON
		out := newCommandOutput(cmd, format, quiet)

		cfg, err := config.LoadConfig()
		if err != nil {
			return fmt.Errorf("cannot load config: %w", err)
		}
		if !cfg.UsesKeyring() {
			return fmt.Errorf("hosted WhoDB login requires an available OS keyring")
		}

		host, err := resolvePlatformHost(cfg, platformHost)
		if err != nil {
			return err
		}
		existingHosts := platformHostsWithLogin(cfg)
		if len(existingHosts) > 0 {
			approved, err := confirmPlatformLoginReplacement(cmd.ErrOrStderr(), existingHosts, platformLoginYes)
			if err != nil {
				return err
			}
			if !approved {
				return fmt.Errorf("login cancelled")
			}
			for _, existing := range existingHosts {
				if _, err := replacePlatformLogin(ctx, cfg, existing.URL, existing.AccountID); err != nil {
					return fmt.Errorf("cannot replace existing hosted WhoDB login for %s: %w\n%s", existing.URL, err, localLogoutHint(existing.URL))
				}
			}
		}
		if !quiet {
			out.Info("Opening browser to sign in to %s", host)
		}
		tokens, err := platform.Login(ctx, platform.LoginOptions{
			Host:        host,
			OpenBrowser: !platformNoBrowser,
			PrintURL: func(loginURL string) {
				if platformNoBrowser || !quiet {
					fmt.Fprintf(cmd.ErrOrStderr(), "If your browser does not open, visit:\n%s\n", loginURL)
				}
			},
		})
		if err != nil {
			return fmt.Errorf("login failed. Use an existing WhoDB account for this host: %w", err)
		}
		if tokens.RefreshToken == "" {
			return fmt.Errorf("login response did not include a refresh token")
		}

		client, err := platform.NewClient(host, tokens.AccessToken)
		if err != nil {
			return err
		}
		manifest := fetchPlatformManifest(ctx, client)
		client.SetPlatformManifest(manifest)
		user, err := client.Me(ctx)
		if err != nil {
			return fmt.Errorf("login failed. Use an existing WhoDB account for this host: %w", err)
		}
		hostEntry := config.PlatformHost{
			URL:       client.Host(),
			AccountID: user.ID,
			Email:     user.Email,
		}
		if manifest != nil {
			if err := storePlatformManifest(&hostEntry, manifest); err != nil {
				return err
			}
		}
		selection, err := autoSelectPlatformWorkspace(ctx, client, &hostEntry)
		if err != nil {
			return err
		}
		cfg.SetOnlyPlatformHost(hostEntry)
		if err := cfg.SavePlatformRefreshToken(client.Host(), user.ID, tokens.RefreshToken); err != nil {
			return err
		}
		if err := cfg.Save(); err != nil {
			return err
		}

		data := loginOutput{
			Host:               client.Host(),
			ID:                 user.ID,
			Email:              user.Email,
			DefaultOrgID:       hostEntry.DefaultOrgID,
			DefaultOrgName:     hostEntry.DefaultOrgName,
			DefaultProjectID:   hostEntry.DefaultProjectID,
			DefaultProjectName: hostEntry.DefaultProjectName,
			WorkspaceSelected:  strings.TrimSpace(hostEntry.DefaultOrgID) != "" && strings.TrimSpace(hostEntry.DefaultProjectID) != "",
			OrganizationCount:  len(selection.Orgs),
			ProjectCount:       len(selection.Projects),
		}
		if format == output.FormatJSON {
			return writeAutomationEnvelope(cmd, "login", data)
		}
		out.Success("Signed in to %s as %s", client.Host(), user.Email)
		for _, message := range selection.Messages {
			out.Info(message)
		}
		if len(selection.Orgs) == 0 {
			out.Info(noOrganizationAccessMessage(client.Host()))
		}
		return nil
	},
}

var logoutCmd = &cobra.Command{
	Use:           "logout",
	Short:         "Sign out of hosted WhoDB",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		format, err := output.ParseFormat(platformFormat)
		if err != nil {
			return err
		}
		quiet := platformQuiet || format == output.FormatJSON
		out := newCommandOutput(cmd, format, quiet)

		cfg, err := config.LoadConfig()
		if err != nil {
			return fmt.Errorf("cannot load config: %w", err)
		}
		host, err := resolvePlatformHost(cfg, platformHost)
		if err != nil {
			return err
		}
		entry, ok := cfg.GetPlatformHost(host)
		if !ok {
			return fmt.Errorf("not logged in to %s", host)
		}
		var status string
		if platformLogoutLocal {
			if err := clearPlatformLogin(cfg, host, entry.AccountID); err != nil {
				return err
			}
			status = "local_only"
		} else {
			status, err = revokePlatformLogin(ctx, cfg, host, entry.AccountID)
		}
		if err != nil {
			return fmt.Errorf("%w\n%s", err, localLogoutHint(host))
		}
		if format == output.FormatJSON {
			return writeAutomationEnvelope(cmd, "logout", map[string]string{"host": host, "status": status})
		}
		if status == "local_only" {
			out.Success("Removed local credentials for %s; hosted session was not revoked", host)
			return nil
		}
		if status == "already_revoked" {
			out.Success("Signed out of %s; hosted session was already expired or revoked", host)
			return nil
		}
		out.Success("Signed out of %s", host)
		return nil
	},
}

var whoamiCmd = &cobra.Command{
	Use:           "whoami",
	Short:         "Show the current hosted WhoDB user",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		format, err := output.ParseFormat(platformFormat)
		if err != nil {
			return err
		}
		session, err := loadPlatformSession(ctx, platformHost)
		if err != nil {
			return err
		}
		user, err := session.Client.Me(ctx)
		if err != nil {
			return err
		}
		updatePlatformHostUser(session.Config, &session.Host, user)
		if err := session.Config.Save(); err != nil {
			return err
		}

		data := whoamiOutput{
			Host:           session.Host.URL,
			ID:             user.ID,
			Email:          user.Email,
			DisplayName:    user.DisplayName,
			OrgID:          session.Host.DefaultOrgID,
			DefaultOrg:     session.Host.DefaultOrgName,
			DefaultProject: session.Host.DefaultProjectName,
		}
		if format == output.FormatJSON {
			return writeCommandJSON(cmd, data)
		}
		rows := [][]any{
			{"host", data.Host},
			{"email", data.Email},
			{"user_id", data.ID},
			{"active_org_id", data.OrgID},
			{"default_org", data.DefaultOrg},
			{"default_project", data.DefaultProject},
		}
		return newCommandOutput(cmd, format, platformQuiet).WriteQueryResult(&output.QueryResult{
			Columns: []output.Column{{Name: "field", Type: "string"}, {Name: "value", Type: "string"}},
			Rows:    rows,
		})
	},
}

var manifestCmd = &cobra.Command{
	Use:           "manifest",
	Short:         "Show the hosted WhoDB platform manifest",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		format, err := output.ParseFormat(platformFormat)
		if err != nil {
			return err
		}
		session, err := loadPlatformSession(ctx, platformHost)
		if err != nil {
			return err
		}

		manifest := manifestFromCache(session.Host.Manifest)
		if manifestRefresh || manifest == nil {
			manifest, err = refreshPlatformManifest(ctx, session.Config, &session.Host, session.Client)
			if err != nil {
				return err
			}
		}
		data := platformManifestOutput(session.Host, manifest)
		if format == output.FormatJSON {
			return writeCommandJSON(cmd, data)
		}
		rows := [][]any{
			{"host", data.Host},
			{"platform_version", data.PlatformVersion},
			{"manifest_protocol_version", data.ManifestProtocolVersion},
			{"generated_at", data.GeneratedAt},
			{"fetched_at", data.FetchedAt},
			{"operations", len(data.Operations)},
			{"types", len(data.Types)},
		}
		return newCommandOutput(cmd, format, platformQuiet).WriteQueryResult(&output.QueryResult{
			Columns: []output.Column{{Name: "field", Type: "string"}, {Name: "value", Type: "string"}},
			Rows:    rows,
		})
	},
}

var statusCmd = &cobra.Command{
	Use:           "status",
	Short:         "Show hosted WhoDB CLI status",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		format, err := output.ParseFormat(platformFormat)
		if err != nil {
			return err
		}
		session, err := loadPlatformSession(ctx, platformHost)
		if err != nil {
			return err
		}
		user, err := session.Client.Me(ctx)
		if err != nil {
			return err
		}
		updatePlatformHostUser(session.Config, &session.Host, user)

		orgs, err := session.Client.Organizations(ctx)
		if err != nil {
			return err
		}
		selection, err := autoSelectPlatformWorkspaceWithOrgs(ctx, session.Client, &session.Host, orgs)
		if err != nil {
			return err
		}
		projects := selection.Projects
		if projects == nil {
			projects, err = statusProjects(ctx, session, selection.Orgs)
			if err != nil {
				return err
			}
		}
		manifest := manifestFromCache(session.Host.Manifest)
		data := platformStatusFor(session.Host, user, selection.Orgs, projects, manifest)
		session.Config.UpsertPlatformHost(session.Host)
		if err := session.Config.Save(); err != nil {
			return err
		}
		if format == output.FormatJSON {
			return writeCommandJSON(cmd, data)
		}
		rows := [][]any{
			{"host", data.Host},
			{"logged_in", data.LoggedIn},
			{"email", data.Email},
			{"workspace_selected", data.WorkspaceSelected},
			{"default_org", data.DefaultOrgName},
			{"default_project", data.DefaultProjectName},
			{"platform_version", data.PlatformVersion},
			{"manifest_protocol_version", data.ManifestProtocolVersion},
			{"manifest_fetched_at", data.ManifestFetchedAt},
			{"source_management_supported", data.SourceManagementSupported},
			{"organizations", data.OrganizationCount},
			{"projects", data.ProjectCount},
		}
		out := newCommandOutput(cmd, format, platformQuiet)
		for _, message := range selection.Messages {
			out.Info(message)
		}
		return out.WriteQueryResult(&output.QueryResult{
			Columns: []output.Column{{Name: "field", Type: "string"}, {Name: "value", Type: "string"}},
			Rows:    rows,
		})
	},
}

var orgsCmd = &cobra.Command{
	Use:   "orgs",
	Short: "Manage hosted WhoDB organizations",
}

var orgsListCmd = &cobra.Command{
	Use:           "list",
	Short:         "List hosted WhoDB organizations",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		format, err := output.ParseFormat(platformFormat)
		if err != nil {
			return err
		}
		session, err := loadPlatformSession(ctx, platformHost)
		if err != nil {
			return err
		}
		orgs, err := session.Client.Organizations(ctx)
		if err != nil {
			return err
		}
		out := newCommandOutput(cmd, format, platformQuiet)
		if format == output.FormatJSON {
			return writeCommandJSON(cmd, orgs)
		}
		if len(orgs) == 0 {
			out.Info(noOrganizationAccessMessage(session.Host.URL))
		}
		rows := make([][]any, len(orgs))
		for i, org := range orgs {
			current := ""
			if org.ID == session.Host.DefaultOrgID {
				current = "yes"
			}
			rows[i] = []any{org.ID, org.Name, org.Slug, current}
		}
		return out.WriteQueryResult(&output.QueryResult{
			Columns: []output.Column{
				{Name: "id", Type: "string"},
				{Name: "name", Type: "string"},
				{Name: "slug", Type: "string"},
				{Name: "default", Type: "string"},
			},
			Rows: rows,
		})
	},
}

var projectsCmd = &cobra.Command{
	Use:   "projects",
	Short: "Manage hosted WhoDB projects",
}

var projectsListCmd = &cobra.Command{
	Use:           "list",
	Short:         "List hosted WhoDB projects",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		format, err := output.ParseFormat(platformFormat)
		if err != nil {
			return err
		}
		session, err := loadPlatformSession(ctx, platformHost)
		if err != nil {
			return err
		}
		org, err := resolveOrganization(ctx, session.Client, session.Host, projectsOrg)
		if err != nil {
			return err
		}
		projects, err := session.Client.Projects(ctx, org.ID)
		if err != nil {
			return err
		}
		out := newCommandOutput(cmd, format, platformQuiet)
		if format == output.FormatJSON {
			return writeCommandJSON(cmd, projects)
		}
		if len(projects) == 0 {
			out.Info(noProjectsMessage(org.Name))
		}
		rows := make([][]any, len(projects))
		for i, project := range projects {
			current := ""
			if project.ID == session.Host.DefaultProjectID {
				current = "yes"
			}
			rows[i] = []any{project.ID, project.Name, project.Slug, org.Name, current}
		}
		return out.WriteQueryResult(&output.QueryResult{
			Columns: []output.Column{
				{Name: "id", Type: "string"},
				{Name: "name", Type: "string"},
				{Name: "slug", Type: "string"},
				{Name: "org", Type: "string"},
				{Name: "default", Type: "string"},
			},
			Rows: rows,
		})
	},
}

var sourcesCmd = &cobra.Command{
	Use:   "sources",
	Short: "Manage hosted WhoDB project sources",
}

var sourcesTypesCmd = &cobra.Command{
	Use:           "types",
	Short:         "List hosted WhoDB source types",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		format, err := output.ParseFormat(platformFormat)
		if err != nil {
			return err
		}
		session, err := loadPlatformSession(ctx, platformHost)
		if err != nil {
			return err
		}
		types, err := session.Client.SourceTypes(ctx)
		if err != nil {
			return err
		}
		sortSourceTypes(types)
		if format == output.FormatJSON {
			return writeCommandJSON(cmd, types)
		}
		rows := make([][]any, len(types))
		for i, sourceType := range types {
			rows[i] = []any{sourceType.ID, sourceType.Label, sourceType.Category, sourceType.Connector, len(sourceType.ConnectionFields)}
		}
		return newCommandOutput(cmd, format, platformQuiet).WriteQueryResult(&output.QueryResult{
			Columns: []output.Column{
				{Name: "id", Type: "string"},
				{Name: "label", Type: "string"},
				{Name: "category", Type: "string"},
				{Name: "connector", Type: "string"},
				{Name: "fields", Type: "int"},
			},
			Rows: rows,
		})
	},
}

var sourcesFieldsCmd = &cobra.Command{
	Use:           "fields <source-type>",
	Short:         "Show hosted WhoDB source type fields",
	Args:          cobra.ExactArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		format, err := output.ParseFormat(platformFormat)
		if err != nil {
			return err
		}
		session, err := loadPlatformSession(ctx, platformHost)
		if err != nil {
			return err
		}
		types, err := session.Client.SourceTypes(ctx)
		if err != nil {
			return err
		}
		sourceType, err := resolveSourceType(types, args[0])
		if err != nil {
			return err
		}
		if format == output.FormatJSON {
			return writeCommandJSON(cmd, sourceType.ConnectionFields)
		}
		rows := make([][]any, len(sourceType.ConnectionFields))
		for i, field := range sourceType.ConnectionFields {
			rows[i] = []any{
				field.Key,
				field.Kind,
				field.Section,
				field.Required,
				sourceFieldDefault(field),
				sourceFieldSecret(field),
				field.SupportsOptions,
			}
		}
		return newCommandOutput(cmd, format, platformQuiet).WriteQueryResult(&output.QueryResult{
			Columns: []output.Column{
				{Name: "key", Type: "string"},
				{Name: "kind", Type: "string"},
				{Name: "section", Type: "string"},
				{Name: "required", Type: "bool"},
				{Name: "default", Type: "string"},
				{Name: "secret", Type: "bool"},
				{Name: "supports_options", Type: "bool"},
			},
			Rows: rows,
		})
	},
}

var sourcesListCmd = &cobra.Command{
	Use:           "list",
	Short:         "List hosted WhoDB project sources",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		format, err := output.ParseFormat(platformFormat)
		if err != nil {
			return err
		}
		session, err := loadPlatformSession(ctx, platformHost)
		if err != nil {
			return err
		}
		org, project, err := resolvePlatformProject(ctx, session, sourcesOrg, sourcesProject)
		if err != nil {
			return err
		}
		sources, err := session.Client.ProjectSources(ctx, org.ID, project.ID)
		if err != nil {
			return err
		}
		out := newCommandOutput(cmd, format, platformQuiet)
		if format == output.FormatJSON {
			return writeCommandJSON(cmd, sources)
		}
		if len(sources) == 0 {
			out.Info("No sources found in project %q.", project.Name)
		}
		rows := make([][]any, len(sources))
		for i, source := range sources {
			rows[i] = []any{source.ID, source.Name, source.DatabaseType, project.Name, source.CreatedBy, source.CreatedAt}
		}
		return out.WriteQueryResult(&output.QueryResult{
			Columns: []output.Column{
				{Name: "id", Type: "string"},
				{Name: "name", Type: "string"},
				{Name: "type", Type: "string"},
				{Name: "project", Type: "string"},
				{Name: "created_by", Type: "string"},
				{Name: "created_at", Type: "string"},
			},
			Rows: rows,
		})
	},
}

var sourcesGetCmd = &cobra.Command{
	Use:           "get <source>",
	Short:         "Show hosted WhoDB source metadata",
	Args:          cobra.ExactArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		format, err := output.ParseFormat(platformFormat)
		if err != nil {
			return err
		}
		session, err := loadPlatformSession(ctx, platformHost)
		if err != nil {
			return err
		}
		org, project, err := resolvePlatformProject(ctx, session, sourcesOrg, sourcesProject)
		if err != nil {
			return err
		}
		sources, err := session.Client.ProjectSources(ctx, org.ID, project.ID)
		if err != nil {
			return err
		}
		source, err := resolveSource(sources, args[0])
		if err != nil {
			return err
		}
		if format == output.FormatJSON {
			return writeCommandJSON(cmd, source)
		}
		rows := [][]any{
			{"id", source.ID},
			{"name", source.Name},
			{"type", source.DatabaseType},
			{"project", project.Name},
			{"project_id", source.ProjectID},
			{"created_by", source.CreatedBy},
			{"created_at", source.CreatedAt},
		}
		return newCommandOutput(cmd, format, platformQuiet).WriteQueryResult(&output.QueryResult{
			Columns: []output.Column{{Name: "field", Type: "string"}, {Name: "value", Type: "string"}},
			Rows:    rows,
		})
	},
}

var sourcesCreateCmd = &cobra.Command{
	Use:           "create [source-type]",
	Short:         "Create a hosted WhoDB project source",
	Args:          cobra.MaximumNArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		format, err := output.ParseFormat(platformFormat)
		if err != nil {
			return err
		}
		quiet := platformQuiet || format == output.FormatJSON
		out := newCommandOutput(cmd, format, quiet)
		sourceTypeValue, err := sourceTypeFromCreateArgs(args, sourceType)
		if err != nil {
			return err
		}
		session, err := loadPlatformSession(ctx, platformHost)
		if err != nil {
			return err
		}
		types, err := session.Client.SourceTypes(ctx)
		if err != nil {
			return err
		}
		selectedType, err := resolveSourceType(types, sourceTypeValue)
		if err != nil {
			return err
		}
		input, err := collectSourceCreateInput(cmd, selectedType)
		if err != nil {
			return err
		}
		org, project, err := resolvePlatformProject(ctx, session, sourcesOrg, sourcesProject)
		if err != nil {
			return err
		}
		input.OrgID = org.ID
		input.ProjectID = project.ID
		created, err := session.Client.CreateSource(ctx, input)
		if err != nil {
			return err
		}
		if format == output.FormatJSON {
			return writeAutomationEnvelope(cmd, "sources.create", created)
		}
		out.Success("Created source %s in project %s", created.Name, project.Name)
		return nil
	},
}

var sourcesDeleteCmd = &cobra.Command{
	Use:           "delete <source>",
	Short:         "Delete a hosted WhoDB project source",
	Args:          cobra.ExactArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		format, err := output.ParseFormat(platformFormat)
		if err != nil {
			return err
		}
		quiet := platformQuiet || format == output.FormatJSON
		out := newCommandOutput(cmd, format, quiet)
		session, err := loadPlatformSession(ctx, platformHost)
		if err != nil {
			return err
		}
		org, project, err := resolvePlatformProject(ctx, session, sourcesOrg, sourcesProject)
		if err != nil {
			return err
		}
		sources, err := session.Client.ProjectSources(ctx, org.ID, project.ID)
		if err != nil {
			return err
		}
		source, err := resolveSource(sources, args[0])
		if err != nil {
			return err
		}
		approved, err := confirmSourceDelete(cmd.ErrOrStderr(), source, sourceDeleteYes)
		if err != nil {
			return err
		}
		if !approved {
			return fmt.Errorf("delete cancelled")
		}
		if err := session.Client.DeleteSource(ctx, org.ID, project.ID, source.ID); err != nil {
			return err
		}
		if format == output.FormatJSON {
			return writeAutomationEnvelope(cmd, "sources.delete", map[string]string{"id": source.ID, "projectId": project.ID})
		}
		out.Success("Deleted source %s from project %s", source.Name, project.Name)
		return nil
	},
}

var sourcesObjectsCmd = &cobra.Command{
	Use:           "objects <source>",
	Short:         "Browse hosted WhoDB source objects",
	Args:          cobra.ExactArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		format, err := output.ParseFormat(platformFormat)
		if err != nil {
			return err
		}
		if err := validatePlatformPage(sourceObjectLimit, sourceObjectOffset); err != nil {
			return err
		}
		parent, err := parseOptionalSourceObjectRef(sourceObjectParent)
		if err != nil {
			return err
		}
		kinds, err := parseSourceObjectKinds(sourceObjectKinds)
		if err != nil {
			return err
		}
		session, err := loadPlatformSession(ctx, platformHost)
		if err != nil {
			return err
		}
		org, project, source, err := resolvePlatformSource(ctx, session, sourcesOrg, sourcesProject, args[0])
		if err != nil {
			return err
		}
		objects, err := session.Client.SourceObjects(ctx, org.ID, project.ID, source.ID, parent, kinds, sourceObjectLimit, sourceObjectOffset)
		if err != nil {
			return err
		}
		if format == output.FormatJSON {
			return writeCommandJSON(cmd, objects)
		}
		rows := make([][]any, len(objects))
		for i, object := range objects {
			rows[i] = []any{
				formatSourceObjectRef(object.Kind, object.Path),
				object.Kind,
				object.Name,
				object.HasChildren,
				strings.Join(object.Actions, ","),
				formatSourceMetadata(object.Metadata),
			}
		}
		return newCommandOutput(cmd, format, platformQuiet).WriteQueryResult(&output.QueryResult{
			Columns: []output.Column{
				{Name: "ref", Type: "string"},
				{Name: "kind", Type: "string"},
				{Name: "name", Type: "string"},
				{Name: "has_children", Type: "bool"},
				{Name: "actions", Type: "string"},
				{Name: "metadata", Type: "string"},
			},
			Rows: rows,
		})
	},
}

var sourcesColumnsCmd = &cobra.Command{
	Use:           "columns <source>",
	Short:         "Show hosted WhoDB source object columns",
	Args:          cobra.ExactArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		format, err := output.ParseFormat(platformFormat)
		if err != nil {
			return err
		}
		ref, err := parseRequiredSourceObjectRef(sourceColumnRef)
		if err != nil {
			return err
		}
		session, err := loadPlatformSession(ctx, platformHost)
		if err != nil {
			return err
		}
		org, project, source, err := resolvePlatformSource(ctx, session, sourcesOrg, sourcesProject, args[0])
		if err != nil {
			return err
		}
		columns, err := session.Client.SourceColumns(ctx, org.ID, project.ID, source.ID, ref)
		if err != nil {
			return err
		}
		if format == output.FormatJSON {
			return writeCommandJSON(cmd, columns)
		}
		rows := make([][]any, len(columns))
		for i, column := range columns {
			rows[i] = []any{
				column.Name,
				column.Type,
				column.IsPrimary,
				column.IsForeignKey,
				formatColumnReference(column),
				formatOptionalInt(column.Length),
				formatOptionalInt(column.Precision),
				formatOptionalInt(column.Scale),
			}
		}
		return newCommandOutput(cmd, format, platformQuiet).WriteQueryResult(&output.QueryResult{
			Columns: []output.Column{
				{Name: "name", Type: "string"},
				{Name: "type", Type: "string"},
				{Name: "primary", Type: "bool"},
				{Name: "foreign_key", Type: "bool"},
				{Name: "references", Type: "string"},
				{Name: "length", Type: "int"},
				{Name: "precision", Type: "int"},
				{Name: "scale", Type: "int"},
			},
			Rows: rows,
		})
	},
}

var sourcesRowsCmd = &cobra.Command{
	Use:           "rows <source>",
	Short:         "Preview hosted WhoDB source object rows",
	Args:          cobra.ExactArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		format, err := output.ParseFormat(platformFormat)
		if err != nil {
			return err
		}
		if err := validatePlatformPage(sourceRowsLimit, sourceRowsOffset); err != nil {
			return err
		}
		ref, err := parseRequiredSourceObjectRef(sourceRowsRef)
		if err != nil {
			return err
		}
		session, err := loadPlatformSession(ctx, platformHost)
		if err != nil {
			return err
		}
		org, project, source, err := resolvePlatformSource(ctx, session, sourcesOrg, sourcesProject, args[0])
		if err != nil {
			return err
		}
		result, err := session.Client.SourceRows(ctx, org.ID, project.ID, source.ID, ref, sourceRowsLimit, sourceRowsOffset)
		if err != nil {
			return err
		}
		if format == output.FormatJSON {
			return writeCommandJSON(cmd, result)
		}
		return newCommandOutput(cmd, format, platformQuiet).WriteStringQueryResult(platformRowsToOutput(result))
	},
}

var useCmd = &cobra.Command{
	Use:           "use",
	Short:         "Select the default hosted WhoDB organization and project",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if strings.TrimSpace(useOrg) == "" || strings.TrimSpace(useProject) == "" {
			return fmt.Errorf("--org and --project are required")
		}
		ctx := context.Background()
		format, err := output.ParseFormat(platformFormat)
		if err != nil {
			return err
		}
		quiet := platformQuiet || format == output.FormatJSON
		out := newCommandOutput(cmd, format, quiet)
		session, err := loadPlatformSession(ctx, platformHost)
		if err != nil {
			return err
		}
		org, err := resolveOrganization(ctx, session.Client, session.Host, useOrg)
		if err != nil {
			return err
		}
		projects, err := session.Client.Projects(ctx, org.ID)
		if err != nil {
			return err
		}
		project, err := resolveProject(projects, useProject, org.Name)
		if err != nil {
			return err
		}
		session.Host.DefaultOrgID = org.ID
		session.Host.DefaultOrgName = org.Name
		session.Host.DefaultProjectID = project.ID
		session.Host.DefaultProjectName = project.Name
		session.Config.UpsertPlatformHost(session.Host)
		session.Config.SetDefaultPlatformHost(session.Host.URL)
		if err := session.Config.Save(); err != nil {
			return err
		}
		data := map[string]string{
			"host":      session.Host.URL,
			"org":       org.Name,
			"orgId":     org.ID,
			"project":   project.Name,
			"projectId": project.ID,
		}
		if format == output.FormatJSON {
			return writeAutomationEnvelope(cmd, "use", data)
		}
		out.Success("Using %s / %s on %s", org.Name, project.Name, session.Host.URL)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(logoutCmd)
	rootCmd.AddCommand(whoamiCmd)
	rootCmd.AddCommand(manifestCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(orgsCmd)
	rootCmd.AddCommand(projectsCmd)
	rootCmd.AddCommand(sourcesCmd)
	rootCmd.AddCommand(useCmd)

	for _, command := range []*cobra.Command{loginCmd, logoutCmd, whoamiCmd, manifestCmd, statusCmd, orgsCmd, projectsCmd, sourcesCmd, useCmd} {
		command.PersistentFlags().StringVar(&platformHost, "host", "", "hosted WhoDB URL (default app.whodb.com)")
		command.PersistentFlags().StringVarP(&platformFormat, "format", "f", "auto", "output format: auto, table, plain, json, ndjson, csv")
		command.PersistentFlags().BoolVarP(&platformQuiet, "quiet", "q", false, "suppress informational messages")
		command.RegisterFlagCompletionFunc("format", completeOutputFormats)
	}

	loginCmd.Flags().BoolVar(&platformNoBrowser, "no-browser", false, "print login URL without opening a browser")
	loginCmd.Flags().BoolVarP(&platformLoginYes, "yes", "y", false, "replace an existing hosted WhoDB login without prompting")
	logoutCmd.Flags().BoolVar(&platformLogoutLocal, "local", false, "remove local hosted WhoDB credentials without revoking the hosted session")
	manifestCmd.Flags().BoolVar(&manifestRefresh, "refresh", false, "fetch and cache the current hosted platform manifest")
	orgsCmd.AddCommand(orgsListCmd)
	projectsCmd.AddCommand(projectsListCmd)
	projectsListCmd.Flags().StringVar(&projectsOrg, "org", "", "organization id, slug, or name (defaults to selected organization)")
	sourcesCmd.AddCommand(sourcesTypesCmd)
	sourcesCmd.AddCommand(sourcesFieldsCmd)
	sourcesCmd.AddCommand(sourcesListCmd)
	sourcesCmd.AddCommand(sourcesGetCmd)
	sourcesCmd.AddCommand(sourcesCreateCmd)
	sourcesCmd.AddCommand(sourcesConfigCmd)
	sourcesCmd.AddCommand(sourcesUpdateCmd)
	sourcesCmd.AddCommand(sourcesTestCmd)
	sourcesCmd.AddCommand(sourcesDeleteCmd)
	sourcesCmd.AddCommand(sourcesObjectsCmd)
	sourcesCmd.AddCommand(sourcesColumnsCmd)
	sourcesCmd.AddCommand(sourcesRowsCmd)
	sourcesCmd.PersistentFlags().StringVar(&sourcesOrg, "org", "", "organization id, slug, or name (defaults to selected organization)")
	sourcesCmd.PersistentFlags().StringVar(&sourcesProject, "project", "", "project id, slug, or name (defaults to selected project)")
	registerSourceInputFlags(sourcesCreateCmd, true, true)
	registerSourceInputFlags(sourcesUpdateCmd, true, false)
	registerSourceInputFlags(sourcesTestCmd, false, true)
	sourcesDeleteCmd.Flags().BoolVarP(&sourceDeleteYes, "yes", "y", false, "delete the source without prompting")
	sourcesObjectsCmd.Flags().StringVar(&sourceObjectParent, "parent", "", "parent object ref as kind:path, for example schema:public")
	sourcesObjectsCmd.Flags().StringArrayVar(&sourceObjectKinds, "kind", nil, "object kind to include; repeatable")
	sourcesObjectsCmd.Flags().IntVar(&sourceObjectLimit, "limit", 50, "maximum objects to return")
	sourcesObjectsCmd.Flags().IntVar(&sourceObjectOffset, "offset", 0, "object offset")
	sourcesColumnsCmd.Flags().StringVar(&sourceColumnRef, "ref", "", "object ref as kind:path, for example table:public.users")
	sourcesRowsCmd.Flags().StringVar(&sourceRowsRef, "ref", "", "object ref as kind:path, for example table:public.users")
	sourcesRowsCmd.Flags().IntVar(&sourceRowsLimit, "limit", 50, "maximum rows to return")
	sourcesRowsCmd.Flags().IntVar(&sourceRowsOffset, "offset", 0, "row offset")
	useCmd.Flags().StringVar(&useOrg, "org", "", "organization id, slug, or name")
	useCmd.Flags().StringVar(&useProject, "project", "", "project id, slug, or name")
}

func registerSourceInputFlags(command *cobra.Command, includeName bool, includeType bool) {
	if includeName {
		command.Flags().StringVar(&sourceName, "name", "", "source display name")
	}
	if includeType {
		command.Flags().StringVar(&sourceType, "type", "", "source type, for example Postgres")
	}
	command.Flags().StringVar(&sourceHostname, "hostname", "", "source hostname")
	command.Flags().StringVar(&sourcePort, "port", "", "source port")
	command.Flags().StringVar(&sourceUsername, "username", "", "source username")
	command.Flags().StringVar(&sourceDatabase, "database", "", "source database")
	command.Flags().StringArrayVar(&sourceFields, "field", nil, "source connection field as key=value; repeatable")
	command.Flags().StringArrayVar(&sourceAdvanced, "advanced", nil, "advanced connection option as key=value; repeatable")
	command.Flags().StringVar(&sourcePasswordEnv, "password-env", "", "environment variable containing the source password")
	command.Flags().BoolVar(&sourcePasswordIn, "password-stdin", false, "read the source password from stdin")
}

func loadPlatformSession(ctx context.Context, hostFlag string) (*platformSession, error) {
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("cannot load config: %w", err)
	}
	host, err := resolvePlatformHost(cfg, hostFlag)
	if err != nil {
		return nil, err
	}
	entry, ok := cfg.GetPlatformHost(host)
	if !ok || entry.AccountID == "" {
		return nil, fmt.Errorf("not logged in to %s; run whodb-cli login --host %s", host, host)
	}
	refreshToken, err := cfg.GetPlatformRefreshToken(host, entry.AccountID)
	if err != nil {
		return nil, fmt.Errorf("cannot load hosted WhoDB refresh token: %w", err)
	}
	tokens, err := platform.RefreshToken(ctx, host, refreshToken)
	if err != nil {
		return nil, err
	}
	if tokens.RefreshToken != "" && tokens.RefreshToken != refreshToken {
		if err := cfg.SavePlatformRefreshToken(host, entry.AccountID, tokens.RefreshToken); err != nil {
			return nil, err
		}
	}
	client, err := platform.NewClient(host, tokens.AccessToken)
	if err != nil {
		return nil, err
	}
	attachPlatformManifestRefresher(cfg, entry, client)
	if manifest := resolveCachedPlatformManifest(ctx, client, entry.Manifest); manifest != nil {
		client.SetPlatformManifest(manifest)
	} else if manifest := fetchPlatformManifest(ctx, client); manifest != nil {
		client.SetPlatformManifest(manifest)
		if err := storePlatformManifest(entry, manifest); err != nil {
			return nil, err
		}
		cfg.UpsertPlatformHost(*entry)
		if err := cfg.Save(); err != nil {
			return nil, err
		}
	}
	return &platformSession{
		Config: cfg,
		Host:   *entry,
		Client: client,
	}, nil
}

func attachPlatformManifestRefresher(cfg *config.Config, host *config.PlatformHost, client *platform.Client) {
	client.SetManifestRefresher(func(ctx context.Context, refreshedClient *platform.Client) (*platform.PlatformManifest, error) {
		return refreshPlatformManifest(ctx, cfg, host, refreshedClient)
	})
}

func refreshPlatformManifest(ctx context.Context, cfg *config.Config, host *config.PlatformHost, client *platform.Client) (*platform.PlatformManifest, error) {
	manifest, err := client.PlatformManifest(ctx)
	if err != nil {
		return nil, err
	}
	if err := storePlatformManifest(host, manifest); err != nil {
		return nil, err
	}
	cfg.UpsertPlatformHost(*host)
	if err := cfg.Save(); err != nil {
		return nil, err
	}
	return manifest, nil
}

func resolvePlatformHost(cfg *config.Config, hostFlag string) (string, error) {
	if strings.TrimSpace(hostFlag) != "" {
		return platform.NormalizeHost(hostFlag)
	}
	if cfg != nil && strings.TrimSpace(cfg.Platform.DefaultHost) != "" {
		return platform.NormalizeHost(cfg.Platform.DefaultHost)
	}
	return platform.NormalizeHost(platform.DefaultHost)
}

func updatePlatformHostUser(cfg *config.Config, host *config.PlatformHost, user *platform.User) {
	host.AccountID = user.ID
	host.Email = user.Email
	cfg.UpsertPlatformHost(*host)
}

func fetchPlatformManifest(ctx context.Context, client *platform.Client) *platform.PlatformManifest {
	manifest, err := client.PlatformManifest(ctx)
	if err != nil {
		return nil
	}
	return manifest
}

func resolveCachedPlatformManifest(ctx context.Context, client *platform.Client, cache *config.PlatformManifestCache) *platform.PlatformManifest {
	manifest := manifestFromCache(cache)
	if manifest == nil {
		return nil
	}
	version, err := client.PlatformVersion(ctx)
	if err != nil || version == "" {
		return manifest
	}
	if version == cache.PlatformVersion {
		return manifest
	}
	return nil
}

func storePlatformManifest(host *config.PlatformHost, manifest *platform.PlatformManifest) error {
	raw, err := json.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("cache platform manifest: %w", err)
	}
	host.Manifest = &config.PlatformManifestCache{
		PlatformVersion:         manifest.PlatformVersion,
		ManifestProtocolVersion: manifest.ManifestProtocolVersion,
		FetchedAt:               time.Now().UTC().Format(time.RFC3339),
		Raw:                     raw,
	}
	return nil
}

func manifestFromCache(cache *config.PlatformManifestCache) *platform.PlatformManifest {
	if cache == nil || len(cache.Raw) == 0 {
		return nil
	}
	var manifest platform.PlatformManifest
	if err := json.Unmarshal(cache.Raw, &manifest); err != nil {
		return nil
	}
	return &manifest
}

func platformManifestOutput(host config.PlatformHost, manifest *platform.PlatformManifest) manifestOutput {
	if manifest == nil {
		return manifestOutput{Host: host.URL}
	}
	out := manifestOutput{
		Host:                    host.URL,
		PlatformVersion:         manifest.PlatformVersion,
		ManifestProtocolVersion: manifest.ManifestProtocolVersion,
		GeneratedAt:             manifest.GeneratedAt,
		Operations:              manifest.Operations,
		Types:                   manifest.Types,
	}
	if host.Manifest != nil {
		out.FetchedAt = host.Manifest.FetchedAt
	}
	return out
}

func statusProjects(ctx context.Context, session *platformSession, orgs []platform.Organization) ([]platform.Project, error) {
	orgID := strings.TrimSpace(session.Host.DefaultOrgID)
	if orgID == "" && len(orgs) == 1 {
		orgID = orgs[0].ID
	}
	if orgID == "" {
		return nil, nil
	}
	return session.Client.Projects(ctx, orgID)
}

func autoSelectPlatformWorkspace(ctx context.Context, client *platform.Client, host *config.PlatformHost) (platformWorkspaceSelection, error) {
	orgs, err := client.Organizations(ctx)
	if err != nil {
		return platformWorkspaceSelection{}, err
	}
	return autoSelectPlatformWorkspaceWithOrgs(ctx, client, host, orgs)
}

func autoSelectPlatformWorkspaceWithOrgs(ctx context.Context, client *platform.Client, host *config.PlatformHost, orgs []platform.Organization) (platformWorkspaceSelection, error) {
	selection := platformWorkspaceSelection{Orgs: orgs}
	if host == nil {
		return selection, nil
	}

	if strings.TrimSpace(host.DefaultOrgID) == "" && len(orgs) == 1 {
		org := orgs[0]
		host.DefaultOrgID = org.ID
		host.DefaultOrgName = org.Name
		selection.Messages = append(selection.Messages, fmt.Sprintf("Selected the only available organization: %s", org.Name))
		selection.Changed = true
	} else if strings.TrimSpace(host.DefaultOrgID) != "" {
		if org := findPlatformOrganization(orgs, host.DefaultOrgID); org != nil && host.DefaultOrgName != org.Name {
			host.DefaultOrgName = org.Name
			selection.Changed = true
		}
	}

	if strings.TrimSpace(host.DefaultOrgID) == "" {
		return selection, nil
	}
	projects, err := client.Projects(ctx, host.DefaultOrgID)
	if err != nil {
		return selection, err
	}
	selection.Projects = projects

	if strings.TrimSpace(host.DefaultProjectID) == "" && len(projects) == 1 {
		project := projects[0]
		host.DefaultProjectID = project.ID
		host.DefaultProjectName = project.Name
		selection.Messages = append(selection.Messages, fmt.Sprintf("Selected the only available project: %s", project.Name))
		selection.Changed = true
	} else if strings.TrimSpace(host.DefaultProjectID) != "" {
		if project := findPlatformProject(projects, host.DefaultProjectID); project != nil && host.DefaultProjectName != project.Name {
			host.DefaultProjectName = project.Name
			selection.Changed = true
		}
	}

	return selection, nil
}

func findPlatformOrganization(orgs []platform.Organization, id string) *platform.Organization {
	for i := range orgs {
		if orgs[i].ID == id {
			return &orgs[i]
		}
	}
	return nil
}

func findPlatformProject(projects []platform.Project, id string) *platform.Project {
	for i := range projects {
		if projects[i].ID == id {
			return &projects[i]
		}
	}
	return nil
}

func platformStatusFor(host config.PlatformHost, user *platform.User, orgs []platform.Organization, projects []platform.Project, manifest *platform.PlatformManifest) platformStatusOutput {
	capabilities := platformStatusCapabilities(manifest)
	sourceSupported := true
	for _, capability := range capabilities {
		if !capability.Supported {
			sourceSupported = false
			break
		}
	}
	workspaceSelected := strings.TrimSpace(host.DefaultOrgID) != "" && strings.TrimSpace(host.DefaultProjectID) != ""
	status := platformStatusOutput{
		Host:                      host.URL,
		LoggedIn:                  user != nil && user.ID != "",
		DefaultOrgID:              host.DefaultOrgID,
		DefaultOrgName:            host.DefaultOrgName,
		DefaultProjectID:          host.DefaultProjectID,
		DefaultProjectName:        host.DefaultProjectName,
		WorkspaceSelected:         workspaceSelected,
		ManifestAvailable:         manifest != nil,
		SourceManagementSupported: manifest != nil && sourceSupported,
		OrganizationCount:         len(orgs),
		ProjectCount:              len(projects),
		Capabilities:              capabilities,
	}
	if user != nil {
		status.UserID = user.ID
		status.Email = user.Email
	}
	if manifest != nil {
		status.PlatformVersion = manifest.PlatformVersion
		status.ManifestProtocolVersion = manifest.ManifestProtocolVersion
	}
	if host.Manifest != nil {
		status.ManifestFetchedAt = host.Manifest.FetchedAt
	}
	return status
}

func platformStatusCapabilities(manifest *platform.PlatformManifest) []platformCapabilityStatus {
	required := []struct {
		name      string
		kind      string
		operation string
	}{
		{name: "source_types", kind: "Query", operation: "SourceTypes"},
		{name: "source_list", kind: "Query", operation: "ProjectSources"},
		{name: "source_create", kind: "Mutation", operation: "CreateSource"},
		{name: "source_config", kind: "Query", operation: "SourceConfig"},
		{name: "source_update", kind: "Mutation", operation: "UpdateSource"},
		{name: "source_delete", kind: "Mutation", operation: "DeleteSource"},
		{name: "source_test", kind: "Mutation", operation: "TestSourceConnection"},
		{name: "source_browse", kind: "Query", operation: "PlatformSourceObjects"},
		{name: "source_columns", kind: "Query", operation: "PlatformSourceColumns"},
		{name: "source_rows", kind: "Query", operation: "PlatformSourceRows"},
	}
	capabilities := make([]platformCapabilityStatus, 0, len(required))
	for _, item := range required {
		capabilities = append(capabilities, platformCapabilityStatus{
			Name:      item.name,
			Operation: item.kind + "." + item.operation,
			Supported: manifest != nil && manifest.HasOperation(item.kind, item.operation),
		})
	}
	return capabilities
}

func platformHostsWithLogin(cfg *config.Config) []config.PlatformHost {
	if cfg == nil {
		return nil
	}
	hosts := make([]config.PlatformHost, 0, len(cfg.Platform.Hosts))
	for _, host := range cfg.Platform.Hosts {
		if strings.TrimSpace(host.URL) != "" && strings.TrimSpace(host.AccountID) != "" {
			hosts = append(hosts, host)
		}
	}
	return hosts
}

func confirmPlatformLoginReplacement(out io.Writer, existingHosts []config.PlatformHost, skipPrompt bool) (bool, error) {
	if len(existingHosts) == 0 || skipPrompt {
		return true, nil
	}
	if !isInteractiveInput() {
		return false, fmt.Errorf("login would replace an existing hosted WhoDB login; rerun with --yes to confirm")
	}

	if len(existingHosts) == 1 {
		host := existingHosts[0]
		account := host.Email
		if account == "" {
			account = host.AccountID
		}
		fmt.Fprintf(out, "You are already signed in to %s as %s.\n", host.URL, account)
	} else {
		fmt.Fprintln(out, "You have existing hosted WhoDB logins:")
		for _, host := range existingHosts {
			account := host.Email
			if account == "" {
				account = host.AccountID
			}
			fmt.Fprintf(out, "  - %s as %s\n", host.URL, account)
		}
	}
	fmt.Fprint(out, "Continuing will log out the existing session before signing in. Proceed? [y/N]: ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("reading confirmation: %w", err)
	}

	return isAffirmativeConfirmation(response), nil
}

func revokePlatformLogin(ctx context.Context, cfg *config.Config, host, accountID string) (string, error) {
	refreshToken, err := cfg.GetPlatformRefreshToken(host, accountID)
	if err != nil {
		return "", fmt.Errorf("cannot load hosted WhoDB refresh token; hosted session was not revoked: %w", err)
	}
	tokens, err := platform.RefreshToken(ctx, host, refreshToken)
	if err != nil {
		if platform.IsInvalidGrant(err) {
			if err := clearPlatformLogin(cfg, host, accountID); err != nil {
				return "", err
			}
			return "already_revoked", nil
		}
		return "", fmt.Errorf("logout failed before local credentials were removed; hosted session was not revoked: %w", err)
	}
	if tokens.RefreshToken != "" && tokens.RefreshToken != refreshToken {
		if err := cfg.SavePlatformRefreshToken(host, accountID, tokens.RefreshToken); err != nil {
			return "", fmt.Errorf("cannot update hosted WhoDB refresh token before revocation retry support: %w", err)
		}
	}
	if err := platform.Logout(ctx, host, tokens.AccessToken); err != nil {
		return "", fmt.Errorf("logout failed before local credentials were removed; hosted session was not revoked: %w", err)
	}
	if err := clearPlatformLogin(cfg, host, accountID); err != nil {
		return "", err
	}
	return "revoked", nil
}

func replacePlatformLogin(ctx context.Context, cfg *config.Config, host, accountID string) (string, error) {
	status, err := revokePlatformLogin(ctx, cfg, host, accountID)
	if err == nil {
		return status, nil
	}
	if !config.IsKeyringNotFound(err) {
		return "", err
	}
	if err := clearPlatformLogin(cfg, host, accountID); err != nil {
		return "", err
	}
	return "local_only", nil
}

func clearPlatformLogin(cfg *config.Config, host, accountID string) error {
	if err := cfg.DeletePlatformRefreshToken(host, accountID); err != nil {
		return err
	}
	cfg.RemovePlatformHost(host)
	return cfg.Save()
}

func localLogoutHint(host string) string {
	return fmt.Sprintf("If this host is no longer reachable, remove only the local CLI credentials with:\n  whodb-cli logout --host %s --local", host)
}

func resolvePlatformProject(ctx context.Context, session *platformSession, orgValue, projectValue string) (*platform.Organization, *platform.Project, error) {
	org, err := resolveOrganization(ctx, session.Client, session.Host, orgValue)
	if err != nil {
		return nil, nil, err
	}
	projects, err := session.Client.Projects(ctx, org.ID)
	if err != nil {
		return nil, nil, err
	}
	needle := strings.TrimSpace(projectValue)
	if needle == "" {
		needle = session.Host.DefaultProjectID
	}
	if needle == "" {
		return nil, nil, fmt.Errorf("no project selected; run use --org <org> --project <project> or pass --project")
	}
	project, err := resolveProject(projects, needle, org.Name)
	if err != nil {
		return nil, nil, err
	}
	return org, project, nil
}

func resolveSource(sources []platform.Source, value string) (*platform.Source, error) {
	needle := strings.TrimSpace(value)
	if needle == "" {
		return nil, fmt.Errorf("source is required")
	}
	for _, source := range sources {
		if matchesPlatformIdentifier(needle, source.ID, "", source.Name) {
			return &source, nil
		}
	}
	return nil, fmt.Errorf("source %q not found", needle)
}

func resolvePlatformSource(ctx context.Context, session *platformSession, orgValue, projectValue, sourceValue string) (*platform.Organization, *platform.Project, *platform.Source, error) {
	org, project, err := resolvePlatformProject(ctx, session, orgValue, projectValue)
	if err != nil {
		return nil, nil, nil, err
	}
	sources, err := session.Client.ProjectSources(ctx, org.ID, project.ID)
	if err != nil {
		return nil, nil, nil, err
	}
	source, err := resolveSource(sources, sourceValue)
	if err != nil {
		return nil, nil, nil, err
	}
	return org, project, source, nil
}

func parseOptionalSourceObjectRef(value string) (*platform.SourceObjectRefInput, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	ref, err := parseRequiredSourceObjectRef(value)
	if err != nil {
		return nil, err
	}
	return &ref, nil
}

func parseRequiredSourceObjectRef(value string) (platform.SourceObjectRefInput, error) {
	kindValue, pathValue, ok := strings.Cut(strings.TrimSpace(value), ":")
	if !ok {
		return platform.SourceObjectRefInput{}, fmt.Errorf("object ref %q must use kind:path", value)
	}
	kind, err := parseSourceObjectKind(kindValue)
	if err != nil {
		return platform.SourceObjectRefInput{}, err
	}
	path := splitSourceObjectPath(pathValue)
	if len(path) == 0 {
		return platform.SourceObjectRefInput{}, fmt.Errorf("object ref %q must include a path", value)
	}
	return platform.SourceObjectRefInput{
		Kind: kind,
		Path: path,
	}, nil
}

func parseSourceObjectKinds(values []string) ([]platform.SourceObjectKind, error) {
	if len(values) == 0 {
		return nil, nil
	}
	kinds := make([]platform.SourceObjectKind, 0, len(values))
	for _, value := range values {
		kind, err := parseSourceObjectKind(value)
		if err != nil {
			return nil, err
		}
		kinds = append(kinds, kind)
	}
	return kinds, nil
}

func parseSourceObjectKind(value string) (platform.SourceObjectKind, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	kinds := map[string]platform.SourceObjectKind{
		"database":   "Database",
		"schema":     "Schema",
		"table":      "Table",
		"view":       "View",
		"collection": "Collection",
		"index":      "Index",
		"key":        "Key",
		"item":       "Item",
		"function":   "Function",
		"procedure":  "Procedure",
		"trigger":    "Trigger",
		"sequence":   "Sequence",
	}
	kind, ok := kinds[normalized]
	if !ok {
		return "", fmt.Errorf("unsupported object kind %q", value)
	}
	return kind, nil
}

func splitSourceObjectPath(value string) []string {
	pathValue := strings.TrimSpace(value)
	if pathValue == "" {
		return nil
	}
	separator := "."
	if strings.Contains(pathValue, "/") {
		separator = "/"
	}
	rawParts := strings.Split(pathValue, separator)
	parts := make([]string, 0, len(rawParts))
	for _, part := range rawParts {
		part = strings.TrimSpace(part)
		if part != "" {
			parts = append(parts, part)
		}
	}
	return parts
}

func validatePlatformPage(limit, offset int) error {
	if limit <= 0 {
		return fmt.Errorf("--limit must be greater than 0")
	}
	if limit > 1000 {
		return fmt.Errorf("--limit must be 1000 or less")
	}
	if offset < 0 {
		return fmt.Errorf("--offset must be 0 or greater")
	}
	return nil
}

func formatSourceObjectRef(kind platform.SourceObjectKind, path []string) string {
	if len(path) == 0 {
		return strings.ToLower(string(kind)) + ":"
	}
	separator := "."
	if kind == "Item" || kind == "Key" {
		separator = "/"
	}
	return strings.ToLower(string(kind)) + ":" + strings.Join(path, separator)
}

func formatSourceMetadata(records []platform.Record) string {
	if len(records) == 0 {
		return ""
	}
	parts := make([]string, 0, len(records))
	for _, record := range records {
		parts = append(parts, record.Key+"="+record.Value)
	}
	return strings.Join(parts, ",")
}

func formatColumnReference(column platform.Column) string {
	if column.ReferencedTable == "" && column.ReferencedColumn == "" {
		return ""
	}
	if column.ReferencedColumn == "" {
		return column.ReferencedTable
	}
	return column.ReferencedTable + "." + column.ReferencedColumn
}

func formatOptionalInt(value *int) any {
	if value == nil {
		return ""
	}
	return *value
}

func platformRowsToOutput(result *platform.RowsResult) *output.StringQueryResult {
	columns := make([]output.Column, len(result.Columns))
	for i, column := range result.Columns {
		columns[i] = output.Column{Name: column.Name, Type: column.Type}
	}
	return &output.StringQueryResult{
		Columns: columns,
		Rows:    result.Rows,
	}
}

func parseSourceAdvanced(values []string) (map[string]string, error) {
	advanced := make(map[string]string, len(values))
	for _, value := range values {
		key, raw, ok := strings.Cut(value, "=")
		key = strings.TrimSpace(key)
		if !ok || key == "" {
			return nil, fmt.Errorf("advanced option %q must use key=value", value)
		}
		advanced[key] = strings.TrimSpace(raw)
	}
	return advanced, nil
}

func readSourcePassword(cmd *cobra.Command) (string, error) {
	if strings.TrimSpace(sourcePasswordEnv) != "" {
		value, ok := os.LookupEnv(strings.TrimSpace(sourcePasswordEnv))
		if !ok {
			return "", fmt.Errorf("environment variable %s is not set", sourcePasswordEnv)
		}
		return value, nil
	}
	if sourcePasswordIn {
		raw, err := io.ReadAll(io.LimitReader(cmd.InOrStdin(), 1<<20))
		if err != nil {
			return "", fmt.Errorf("reading password from stdin: %w", err)
		}
		return strings.TrimRight(string(raw), "\r\n"), nil
	}
	if !isInteractiveInput() {
		return "", fmt.Errorf("source password requires --password-stdin or --password-env when stdin is not interactive")
	}
	fmt.Fprint(cmd.ErrOrStderr(), "Password: ")
	password, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(cmd.ErrOrStderr())
	if err != nil {
		return "", fmt.Errorf("reading password: %w", err)
	}
	return string(password), nil
}

func confirmSourceDelete(out io.Writer, source *platform.Source, skipPrompt bool) (bool, error) {
	if skipPrompt {
		return true, nil
	}
	if !isInteractiveInput() {
		return false, fmt.Errorf("sources delete requires --yes when stdin is not interactive")
	}
	fmt.Fprintf(out, "Delete source %s (%s)? [y/N]: ", source.Name, source.ID)

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("reading confirmation: %w", err)
	}
	return isAffirmativeConfirmation(response), nil
}

func resolveOrganization(ctx context.Context, client *platform.Client, host config.PlatformHost, value string) (*platform.Organization, error) {
	orgs, err := client.Organizations(ctx)
	if err != nil {
		return nil, err
	}
	if len(orgs) == 0 {
		return nil, fmt.Errorf("%s", noOrganizationAccessMessage(host.URL))
	}
	needle := strings.TrimSpace(value)
	if needle == "" {
		needle = host.DefaultOrgID
	}
	if needle == "" && len(orgs) == 1 {
		return &orgs[0], nil
	}
	if needle == "" {
		return nil, fmt.Errorf("no organization selected; run whodb-cli use --org <org> --project <project> or pass --org")
	}
	for _, org := range orgs {
		if matchesPlatformIdentifier(needle, org.ID, org.Slug, org.Name) {
			return &org, nil
		}
	}
	return nil, fmt.Errorf("organization %q not found", needle)
}

func resolveProject(projects []platform.Project, value, orgName string) (*platform.Project, error) {
	if len(projects) == 0 {
		return nil, fmt.Errorf("%s", noProjectsMessage(orgName))
	}
	needle := strings.TrimSpace(value)
	if needle == "" && len(projects) == 1 {
		return &projects[0], nil
	}
	if needle == "" {
		return nil, fmt.Errorf("project is required")
	}
	for _, project := range projects {
		if matchesPlatformIdentifier(needle, project.ID, project.Slug, project.Name) {
			return &project, nil
		}
	}
	return nil, fmt.Errorf("project %q not found", needle)
}

func matchesPlatformIdentifier(value, id, slug, name string) bool {
	if value == id || value == slug {
		return true
	}
	return strings.EqualFold(value, name)
}

func noOrganizationAccessMessage(host string) string {
	return fmt.Sprintf("Signed in, but this account does not belong to any organization on %s. Ask an admin for access in WhoDB.", host)
}

func noProjectsMessage(orgName string) string {
	return fmt.Sprintf("No projects found in organization %q. Create one in the hosted UI or ask an admin.", orgName)
}

func isAffirmativeConfirmation(answer string) bool {
	normalized := strings.ToLower(strings.TrimSpace(answer))
	return normalized == "y" || normalized == "yes"
}
