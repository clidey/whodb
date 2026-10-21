package cmd

import (
	"github.com/clidey/whodb/cli/internal/platform"
	"github.com/spf13/cobra"
)

func newSecurityAuditCommand() *cobra.Command {
	input := platform.SecurityAuditInput{}
	cmd := &cobra.Command{Use: "security-audit", Short: "Search the platform security ledger (organization managers)", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			session, err := loadPlatformSession(cmd.Context(), platformHost)
			if err != nil {
				return err
			}
			input.OrgID = session.Host.DefaultOrgID
			result, err := session.Client.SecurityAudit(cmd.Context(), input)
			if err != nil {
				return err
			}
			return writeCommandJSON(cmd, result)
		}}
	cmd.Flags().StringVar(&platformHost, "host", "", "Hosted WhoDB URL")
	cmd.Flags().StringVar(&input.Mode, "mode", "events", "events, exports, or verify (whole organization)")
	cmd.Flags().StringVar(&input.ProjectID, "project-id", "", "Filter events by canonical project ID")
	cmd.Flags().StringVar(&input.ActorID, "actor-id", "", "Filter by actor ID")
	cmd.Flags().StringVar(&input.Action, "action", "", "Filter by exact action")
	cmd.Flags().StringVar(&input.Outcome, "outcome", "", "Filter by exact outcome: SUCCESS, FAILURE, or DENIED")
	cmd.Flags().StringVar(&input.RequestID, "request-id", "", "Correlate a request")
	cmd.Flags().StringVar(&input.RunID, "run-id", "", "Correlate a function or transform run")
	cmd.Flags().StringVar(&input.Since, "since", "", "Inclusive RFC3339 start time")
	cmd.Flags().StringVar(&input.Until, "until", "", "Inclusive RFC3339 end time")
	cmd.Flags().IntVar(&input.Limit, "limit", 50, "Events per page (1-200)")
	cmd.Flags().StringVar(&input.Cursor, "cursor", "", "Continue using nextCursor")
	return cmd
}

func init() { rootCmd.AddCommand(newSecurityAuditCommand()) }
