package mcp

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"

	"github.com/clidey/whodb/cli/internal/config"
	platformapi "github.com/clidey/whodb/cli/internal/platform"
)

// loadAutomationPlatformToolSession shares the CLI's process-only credential
// path. Scope checks precede every network operation and never use saved tokens.
func loadAutomationPlatformToolSession(ctx context.Context, requested platformapi.SessionScope) (*platformToolSession, error) {
	client, fixed, err := platformapi.AutomationClient(requested.Host)
	if err != nil {
		return nil, err
	}
	if client == nil {
		return nil, fmt.Errorf("process-scoped platform credential is unavailable")
	}
	if requested.Org != "" && requested.Org != fixed.Org {
		return nil, fmt.Errorf("process-scoped credential requires its configured organization ID")
	}
	if requested.Project != "" && requested.Project != fixed.Project {
		return nil, fmt.Errorf("process-scoped credential requires its configured project ID")
	}
	if err := checkPlatformPolicy(ctx, fixed.Host, fixed.Org, fixed.Project); err != nil {
		return nil, err
	}
	manifest, err := client.PlatformManifest(ctx)
	if err != nil {
		return nil, fmt.Errorf("cannot load platform manifest: %w", err)
	}
	client.SetPlatformManifest(manifest)
	// Bind pending confirmations to this exact credential without storing it.
	digest := sha256.Sum256([]byte(os.Getenv(platformapi.APIKeyEnv)))
	session := &platformToolSession{Host: config.PlatformHost{URL: fixed.Host, AccountID: fmt.Sprintf("automation:%x", digest), DefaultOrgID: fixed.Org, DefaultProjectID: fixed.Project}, Client: client}
	recordPlatformScope(ctx, session)
	return session, nil
}
