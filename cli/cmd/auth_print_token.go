package cmd

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/clidey/whodb/cli/internal/config"
	platformapi "github.com/clidey/whodb/cli/internal/platform"
)

// authCmd groups machine-facing credential commands. Interactive commands
// (login, logout, whoami) stay top-level.
var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Credential helpers for SDKs and automation",
}

// printTokenOutput is the JSON contract consumed by the WhoDB SDKs'
// CLI-credentials providers (gcloud-ADC-style local development auth).
type printTokenOutput struct {
	AccessToken string `json:"access_token"`
	ExpiresAt   string `json:"expires_at,omitempty"`
	Host        string `json:"host"`
	OrgID       string `json:"org_id,omitempty"`
	OrgName     string `json:"org_name,omitempty"`
	ProjectID   string `json:"project_id,omitempty"`
	ProjectName string `json:"project_name,omitempty"`
}

var printTokenCmd = &cobra.Command{
	Use:   "print-token",
	Short: "Print a fresh access token and saved workspace as JSON",
	Long: `Print a fresh access token for the hosted WhoDB platform, refreshed from
the stored login when needed, plus the default workspace saved by "whodb use".

Intended for programmatic consumers (the WhoDB SDKs use it for local
development credentials); the token is written to stdout.`,
	Args:          cobra.NoArgs,
	SilenceUsage:  true,
	SilenceErrors: false,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadConfig()
		if err != nil {
			return fmt.Errorf("cannot load hosted WhoDB config: %w", err)
		}
		hostURL := strings.TrimSpace(platformHost)
		if hostURL == "" {
			hostURL = strings.TrimSpace(cfg.Platform.DefaultHost)
		}
		if hostURL == "" {
			hostURL = platformapi.DefaultHost
		}
		hostURL, err = platformapi.NormalizeHost(hostURL)
		if err != nil {
			return err
		}
		host, ok := cfg.GetPlatformHost(hostURL)
		if !ok || strings.TrimSpace(host.AccountID) == "" {
			return fmt.Errorf("not logged in to %s — run: whodb login --host %s", hostURL, hostURL)
		}
		tokenSource := platformapi.NewOIDCTokenSource(hostURL, host.AccountID, cfg)
		token, err := tokenSource.Token(cmd.Context())
		if err != nil {
			return fmt.Errorf("cannot obtain access token: %w — run: whodb login --host %s", err, hostURL)
		}
		out := printTokenOutput{
			AccessToken: token,
			Host:        hostURL,
			OrgID:       host.DefaultOrgID,
			OrgName:     host.DefaultOrgName,
			ProjectID:   host.DefaultProjectID,
			ProjectName: host.DefaultProjectName,
		}
		if exp := jwtExpiry(token); !exp.IsZero() {
			out.ExpiresAt = exp.UTC().Format(time.RFC3339)
		}
		return writeCommandJSON(cmd, out)
	},
}

// jwtExpiry decodes the exp claim of a JWT without verifying the signature.
// The value is informational (SDKs use it to schedule a re-exec); the server
// remains the authority on token validity.
func jwtExpiry(token string) time.Time {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return time.Time{}
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return time.Time{}
	}
	var claims struct {
		Exp int64 `json:"exp"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil || claims.Exp == 0 {
		return time.Time{}
	}
	return time.Unix(claims.Exp, 0)
}

func init() {
	printTokenCmd.Flags().StringVar(&platformHost, "host", "", "hosted WhoDB URL (default app.whodb.com)")
	authCmd.AddCommand(printTokenCmd)
	rootCmd.AddCommand(authCmd)
}
