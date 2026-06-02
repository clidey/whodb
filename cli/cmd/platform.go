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
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/clidey/whodb/cli/internal/config"
	"github.com/clidey/whodb/cli/internal/platform"
	"github.com/clidey/whodb/cli/pkg/output"
	"github.com/spf13/cobra"
)

var (
	platformHost      string
	platformFormat    string
	platformQuiet     bool
	platformNoBrowser bool
	platformLoginYes  bool
	useOrg            string
	useProject        string
	projectsOrg       string
)

type platformSession struct {
	Config *config.Config
	Host   config.PlatformHost
	Client *platform.Client
}

type loginOutput struct {
	Host              string `json:"host"`
	ID                string `json:"id"`
	Email             string `json:"email"`
	OrganizationCount int    `json:"organizationCount"`
}

type whoamiOutput struct {
	Host           string `json:"host"`
	ID             string `json:"id"`
	Email          string `json:"email"`
	DisplayName    string `json:"displayName"`
	OrgID          string `json:"orgId"`
	DefaultOrg     string `json:"defaultOrg,omitempty"`
	DefaultProject string `json:"defaultProject,omitempty"`
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
				if _, err := revokePlatformLogin(ctx, cfg, existing.URL, existing.AccountID); err != nil {
					return fmt.Errorf("cannot replace existing hosted WhoDB login for %s: %w", existing.URL, err)
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
		user, err := client.Me(ctx)
		if err != nil {
			return fmt.Errorf("login failed. Use an existing WhoDB account for this host: %w", err)
		}
		orgs, err := client.Organizations(ctx)
		if err != nil {
			return err
		}

		hostEntry := config.PlatformHost{
			URL:       client.Host(),
			AccountID: user.ID,
			Email:     user.Email,
		}
		cfg.UpsertPlatformHost(hostEntry)
		cfg.SetDefaultPlatformHost(client.Host())
		if err := cfg.SavePlatformRefreshToken(client.Host(), user.ID, tokens.RefreshToken); err != nil {
			return err
		}
		if err := cfg.Save(); err != nil {
			return err
		}

		data := loginOutput{Host: client.Host(), ID: user.ID, Email: user.Email, OrganizationCount: len(orgs)}
		if format == output.FormatJSON {
			return writeAutomationEnvelope(cmd, "login", data)
		}
		out.Success("Signed in to %s as %s", client.Host(), user.Email)
		if len(orgs) == 0 {
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
		status, err := revokePlatformLogin(ctx, cfg, host, entry.AccountID)
		if err != nil {
			return err
		}
		if format == output.FormatJSON {
			return writeAutomationEnvelope(cmd, "logout", map[string]string{"host": host, "status": status})
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
			OrgID:          user.OrgID,
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
		if _, err := session.Client.SwitchOrganization(ctx, org.ID); err != nil {
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
	rootCmd.AddCommand(orgsCmd)
	rootCmd.AddCommand(projectsCmd)
	rootCmd.AddCommand(useCmd)

	for _, command := range []*cobra.Command{loginCmd, logoutCmd, whoamiCmd, orgsCmd, projectsCmd, useCmd} {
		command.PersistentFlags().StringVar(&platformHost, "host", "", "hosted WhoDB URL (default app.whodb.com)")
		command.PersistentFlags().StringVarP(&platformFormat, "format", "f", "auto", "output format: auto, table, plain, json, ndjson, csv")
		command.PersistentFlags().BoolVarP(&platformQuiet, "quiet", "q", false, "suppress informational messages")
		command.RegisterFlagCompletionFunc("format", completeOutputFormats)
	}

	loginCmd.Flags().BoolVar(&platformNoBrowser, "no-browser", false, "print login URL without opening a browser")
	loginCmd.Flags().BoolVarP(&platformLoginYes, "yes", "y", false, "replace an existing hosted WhoDB login without prompting")
	orgsCmd.AddCommand(orgsListCmd)
	projectsCmd.AddCommand(projectsListCmd)
	projectsListCmd.Flags().StringVar(&projectsOrg, "org", "", "organization id, slug, or name (defaults to selected organization)")
	useCmd.Flags().StringVar(&useOrg, "org", "", "organization id, slug, or name")
	useCmd.Flags().StringVar(&useProject, "project", "", "project id, slug, or name")
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
	return &platformSession{
		Config: cfg,
		Host:   *entry,
		Client: client,
	}, nil
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

func clearPlatformLogin(cfg *config.Config, host, accountID string) error {
	if err := cfg.DeletePlatformRefreshToken(host, accountID); err != nil {
		return err
	}
	cfg.RemovePlatformHost(host)
	return cfg.Save()
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
