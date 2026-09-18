package whodb

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"sync"
	"time"
)

// CredentialProvider yields a bearer credential for platform requests.
// Refresh is invoked once after a 401 before the request is retried.
type CredentialProvider interface {
	Token(ctx context.Context) (string, error)
	Refresh()
}

// staticCredentials is an API key or raw token.
type staticCredentials struct{ value string }

// Token returns the configured credential.
func (c staticCredentials) Token(context.Context) (string, error) { return c.value, nil }

// Refresh is a no-op: static credentials cannot rotate.
func (staticCredentials) Refresh() {}

// TokenFunc adapts a caller-managed token callback into a CredentialProvider.
type TokenFunc func(ctx context.Context) (string, error)

// Token invokes the callback.
func (f TokenFunc) Token(ctx context.Context) (string, error) { return f(ctx) }

// Refresh is a no-op: the callback is consulted every call.
func (TokenFunc) Refresh() {}

const cliRefreshSkew = 60 * time.Second

// printTokenOutput mirrors `whodb auth print-token --format json`.
type printTokenOutput struct {
	AccessToken string `json:"access_token"`
	ExpiresAt   string `json:"expires_at"`
	Host        string `json:"host"`
	OrgID       string `json:"org_id"`
	ProjectID   string `json:"project_id"`
}

// cliCredentials execs `whodb auth print-token` and caches the token until
// shortly before expiry — the gcloud-ADC pattern for local development.
// Requires the whodb CLI on PATH and a prior `whodb login`.
type cliCredentials struct {
	command string

	mu     sync.Mutex
	cached *printTokenOutput
}

func newCliCredentials() *cliCredentials {
	return &cliCredentials{command: "whodb"}
}

func (c *cliCredentials) exec(ctx context.Context) (*printTokenOutput, error) {
	execCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	output, err := exec.CommandContext(execCtx, c.command, "auth", "print-token", "--format", "json").Output()
	if err != nil {
		if execErr, ok := err.(*exec.Error); ok && execErr.Err == exec.ErrNotFound {
			return nil, fmt.Errorf("%w: whodb CLI not found — install it or set WHODB_API_KEY", ErrCliCredentials)
		}
		detail := ""
		if exitErr, ok := err.(*exec.ExitError); ok {
			detail = string(exitErr.Stderr)
		}
		return nil, fmt.Errorf("%w: whodb auth print-token failed: %s", ErrCliCredentials, detail)
	}
	var parsed printTokenOutput
	if err := json.Unmarshal(output, &parsed); err != nil {
		return nil, fmt.Errorf("%w: whodb auth print-token returned invalid JSON", ErrCliCredentials)
	}
	return &parsed, nil
}

func (c *cliCredentials) fresh(entry *printTokenOutput) bool {
	if entry == nil || entry.ExpiresAt == "" {
		return false // no expiry info — re-exec every call
	}
	expiry, err := time.Parse(time.RFC3339, entry.ExpiresAt)
	if err != nil {
		return false
	}
	return time.Until(expiry) > cliRefreshSkew
}

// Token returns a fresh access token, re-execing the CLI near expiry.
func (c *cliCredentials) Token(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.fresh(c.cached) {
		entry, err := c.exec(ctx)
		if err != nil {
			return "", err
		}
		c.cached = entry
	}
	if c.cached.AccessToken == "" {
		return "", fmt.Errorf("%w: whodb CLI returned an empty access token", ErrAuth)
	}
	return c.cached.AccessToken, nil
}

// Refresh drops the cached token so the next call re-execs the CLI.
func (c *cliCredentials) Refresh() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cached = nil
}

// defaults returns the CLI's saved host/org/project defaults.
func (c *cliCredentials) defaults(ctx context.Context) (*printTokenOutput, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cached == nil {
		entry, err := c.exec(ctx)
		if err != nil {
			return nil, err
		}
		c.cached = entry
	}
	return c.cached, nil
}
