package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/clidey/whodb/cli/internal/platform"
	"github.com/clidey/whodb/cli/pkg/output"
)

func oauthAuthorizeCommand(openBrowser func(string) error) *cobra.Command {
	var noBrowser, yes bool
	command := &cobra.Command{
		Use: "authorize <secret-id-or-name>", Short: "Open the browser to authorize or reconnect an OAuth secret",
		Example: "  whodb secrets authorize SOCIAL_LINKEDIN_ACCESS_TOKEN --org clidey --project social-media",
		Args:    cobra.ExactArgs(1), SilenceUsage: true, SilenceErrors: true,
		Annotations: map[string]string{"platform_operation": "AuthorizeOAuthSecret"},
		RunE: func(cmd *cobra.Command, args []string) error {
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
			id, err := resolvePlatformResourceID(ctx, session, project.ID, "secret", args[0])
			if err != nil {
				return err
			}
			if !yes {
				if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "Start OAuth consent for secret %q (%s) in %s/%s on %s? [y/N]: ", strings.TrimSpace(args[0]), id, org.Slug, project.Slug, session.Host.URL); err != nil {
					return err
				}
				var answer string
				if _, err := fmt.Fscan(cmd.InOrStdin(), &answer); err != nil || !isAffirmativeConfirmation(answer) {
					return fmt.Errorf("authorization cancelled")
				}
			}
			result, err := session.Client.PlatformMutation(ctx, "AuthorizeOAuthSecret", map[string]any{"projectId": project.ID, "id": id})
			if err != nil {
				return err
			}
			link, err := platform.OAuthAuthorizationLink(session.Host.URL, result.Data)
			if err != nil {
				return err
			}
			return presentOAuthAuthorizationLink(cmd, format, noBrowser, link, openBrowser)
		},
	}
	command.Flags().BoolVar(&noBrowser, "no-browser", false, "print the one-time authorization link without opening a browser")
	command.Flags().BoolVarP(&yes, "yes", "y", false, "approve starting OAuth consent")
	return command
}

func presentOAuthAuthorizationLink(cmd *cobra.Command, format output.Format, noBrowser bool, link string, openBrowser func(string) error) error {
	if format == output.FormatJSON {
		return writeCommandJSON(cmd, map[string]string{"authorization_url": link})
	}
	if format != output.FormatAuto && format != output.FormatPlain {
		return newCommandOutput(cmd, format, platformQuiet).WriteQueryResult(tableResult([]string{"authorization_url"}, [][]any{{link}}))
	}
	if !noBrowser {
		if err := openBrowser(link); err == nil {
			return nil
		}
		fmt.Fprintln(cmd.ErrOrStderr(), "Could not open the browser. Open the authorization link manually.")
	}
	_, err := fmt.Fprintln(cmd.OutOrStdout(), link)
	return err
}

func init() {
	secretsCmd.AddCommand(oauthAuthorizeCommand(openURL))
}
