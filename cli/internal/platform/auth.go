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

package platform

import (
	"cmp"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

const deviceCodeGrantType = "urn:ietf:params:oauth:grant-type:device_code"

// TokenResponse is returned by the OIDC token endpoint.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	IDToken      string `json:"id_token,omitempty"`
	ExpiresIn    int    `json:"expires_in,omitempty"`
}

type oidcDiscovery struct {
	Issuer                      string `json:"issuer"`
	AuthorizationEndpoint       string `json:"authorization_endpoint"`
	TokenEndpoint               string `json:"token_endpoint"`
	DeviceAuthorizationEndpoint string `json:"device_authorization_endpoint"`
	RevocationEndpoint          string `json:"revocation_endpoint"`
}

type deviceAuthorizationResponse struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
}

// DeviceAuthorizationPrompt contains the user-facing device verification details.
type DeviceAuthorizationPrompt struct{ VerificationURI, VerificationURIComplete, UserCode string }

// AuthHTTPError describes a failed OIDC endpoint response.
type AuthHTTPError struct {
	StatusCode            int
	Status, Code, Message string
}

func (e *AuthHTTPError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("auth request failed: %s (%s)", e.Code, e.Status)
	}
	if e.Message != "" {
		return fmt.Sprintf("auth request failed: %s (%s)", e.Message, e.Status)
	}
	return fmt.Sprintf("auth request failed: %s", e.Status)
}

// IsInvalidGrant reports whether a refresh token is no longer usable.
func IsInvalidGrant(err error) bool {
	var e *AuthHTTPError
	return errors.As(err, &e) && e.Code == "invalid_grant"
}

// LoginOptions configures interactive CLI authentication.
type LoginOptions struct {
	Host                     string
	OpenBrowser              bool
	UseDeviceAuthorization   bool
	PrintURL                 func(string)
	PrintDeviceAuthorization func(DeviceAuthorizationPrompt)
	Timeout                  time.Duration
}

func mcpConnectedURL(host string) string {
	parsed, err := url.Parse(host)
	if err == nil && parsed.Port() == "8080" {
		switch parsed.Hostname() {
		case "localhost", "127.0.0.1", "::1":
			parsed.Host = net.JoinHostPort(parsed.Hostname(), "3000")
			return strings.TrimRight(parsed.String(), "/") + "/mcp-connected"
		}
	}
	return strings.TrimRight(host, "/") + "/mcp-connected"
}

// Login performs authorization-code PKCE or RFC 8628 device authorization.
func Login(ctx context.Context, opts LoginOptions) (*TokenResponse, error) {
	host, err := NormalizeHost(opts.Host)
	if err != nil {
		return nil, err
	}
	cfg, err := FetchAuthConfig(ctx, host)
	if err != nil {
		return nil, err
	}
	discovery, err := discoverOIDC(ctx, cfg.Issuer)
	if err != nil {
		return nil, err
	}
	if opts.UseDeviceAuthorization {
		if !cfg.Flows.DeviceAuthorization {
			return nil, errors.New("host does not enable device authorization")
		}
		return loginWithDeviceAuthorization(ctx, cfg, discovery, opts)
	}
	if !cfg.Flows.AuthorizationCodePKCE {
		return nil, errors.New("host does not enable authorization-code PKCE")
	}
	return loginWithAuthorizationCode(ctx, host, cfg, discovery, opts)
}

func loginWithAuthorizationCode(ctx context.Context, host string, cfg *AuthConfig, discovery *oidcDiscovery, opts LoginOptions) (*TokenResponse, error) {
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("start login callback server: %w", err)
	}
	defer listener.Close()
	state, verifier := rand.Text(), oauth2.GenerateVerifier()
	redirectURI := "http://" + listener.Addr().String()
	loginURL := discovery.AuthorizationEndpoint + "?" + url.Values{
		"response_type": {"code"}, "scope": {"openid profile email offline_access"}, "redirect_uri": {redirectURI},
		"code_challenge": {oauth2.S256ChallengeFromVerifier(verifier)}, "code_challenge_method": {"S256"}, "client_id": {cfg.ClientID}, "state": {state},
	}.Encode()
	codeCh, errCh := make(chan string, 1), make(chan error, 1)
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		if r.URL.Query().Get("state") != state {
			http.Error(w, "state mismatch", 400)
			sendLoginError(errCh, errors.New("login state mismatch"))
			return
		}
		if oauthErr := r.URL.Query().Get("error"); oauthErr != "" {
			http.Error(w, oauthErr, 400)
			sendLoginError(errCh, fmt.Errorf("login failed: %s", oauthErr))
			return
		}
		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "missing authorization code", 400)
			sendLoginError(errCh, errors.New("missing authorization code"))
			return
		}
		http.Redirect(w, r, mcpConnectedURL(host), http.StatusFound)
		select {
		case codeCh <- code:
		default:
		}
	}), ReadHeaderTimeout: 10 * time.Second}
	defer server.Shutdown(context.Background())
	go func() {
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			sendLoginError(errCh, err)
		}
	}()
	if opts.PrintURL != nil {
		opts.PrintURL(loginURL)
	}
	if opts.OpenBrowser {
		if err := openBrowser(loginURL); err != nil && opts.PrintURL == nil {
			return nil, err
		}
	}
	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	select {
	case code := <-codeCh:
		return requestTokens(waitCtx, discovery.TokenEndpoint, url.Values{"grant_type": {"authorization_code"}, "code": {code}, "redirect_uri": {redirectURI}, "code_verifier": {verifier}, "client_id": {cfg.ClientID}})
	case err := <-errCh:
		return nil, err
	case <-waitCtx.Done():
		return nil, errors.New("login timed out waiting for browser callback")
	}
}

func sendLoginError(ch chan<- error, err error) {
	select {
	case ch <- err:
	default:
	}
}

func loginWithDeviceAuthorization(ctx context.Context, cfg *AuthConfig, discovery *oidcDiscovery, opts LoginOptions) (*TokenResponse, error) {
	if discovery.DeviceAuthorizationEndpoint == "" {
		return nil, errors.New("OIDC discovery did not include device_authorization_endpoint")
	}
	verifier := oauth2.GenerateVerifier()
	raw, err := postOIDCForm(ctx, discovery.DeviceAuthorizationEndpoint, url.Values{
		"client_id":             {cfg.ClientID},
		"scope":                 {"openid profile email offline_access"},
		"code_challenge":        {oauth2.S256ChallengeFromVerifier(verifier)},
		"code_challenge_method": {"S256"},
	})
	if err != nil {
		return nil, err
	}
	var device deviceAuthorizationResponse
	if err := json.Unmarshal(raw, &device); err != nil {
		return nil, fmt.Errorf("decode device authorization response: %w", err)
	}
	if device.DeviceCode == "" || device.UserCode == "" || device.VerificationURI == "" || device.ExpiresIn <= 0 {
		return nil, errors.New("device authorization response is incomplete")
	}
	if opts.PrintDeviceAuthorization != nil {
		opts.PrintDeviceAuthorization(DeviceAuthorizationPrompt{device.VerificationURI, device.VerificationURIComplete, device.UserCode})
	}
	interval := time.Duration(device.Interval) * time.Second
	if interval <= 0 {
		interval = 5 * time.Second
	}
	deadline := time.Duration(device.ExpiresIn) * time.Second
	if opts.Timeout > 0 && opts.Timeout < deadline {
		deadline = opts.Timeout
	}
	waitCtx, cancel := context.WithTimeout(ctx, deadline)
	defer cancel()
	for {
		timer := time.NewTimer(interval)
		select {
		case <-timer.C:
		case <-waitCtx.Done():
			timer.Stop()
			return nil, errors.New("device authorization expired or was cancelled")
		}
		tokens, err := requestTokens(waitCtx, discovery.TokenEndpoint, url.Values{"grant_type": {deviceCodeGrantType}, "device_code": {device.DeviceCode}, "client_id": {cfg.ClientID}, "code_verifier": {verifier}})
		if err == nil {
			return tokens, nil
		}
		var authErr *AuthHTTPError
		if !errors.As(err, &authErr) {
			return nil, err
		}
		switch authErr.Code {
		case "authorization_pending":
			continue
		case "slow_down":
			interval += 5 * time.Second
			continue
		case "access_denied", "expired_token":
			return nil, fmt.Errorf("device authorization failed: %s", authErr.Code)
		default:
			return nil, err
		}
	}
}

// RefreshToken exchanges a refresh token at the discovered token endpoint.
func RefreshToken(ctx context.Context, host, refreshToken string) (*TokenResponse, error) {
	cfg, err := FetchAuthConfig(ctx, host)
	if err != nil {
		return nil, err
	}
	d, err := discoverOIDC(ctx, cfg.Issuer)
	if err != nil {
		return nil, err
	}
	return requestTokens(ctx, d.TokenEndpoint, url.Values{"grant_type": {"refresh_token"}, "refresh_token": {refreshToken}, "client_id": {cfg.ClientID}})
}

// Logout revokes a refresh token at the discovered revocation endpoint.
func Logout(ctx context.Context, host, token string) error {
	if strings.TrimSpace(token) == "" {
		return errors.New("token is required")
	}
	cfg, err := FetchAuthConfig(ctx, host)
	if err != nil {
		return err
	}
	d, err := discoverOIDC(ctx, cfg.Issuer)
	if err != nil {
		return err
	}
	if d.RevocationEndpoint == "" {
		return errors.New("OIDC discovery did not include revocation_endpoint")
	}
	_, err = postOIDCForm(ctx, d.RevocationEndpoint, url.Values{"token": {token}, "token_type_hint": {"refresh_token"}, "client_id": {cfg.ClientID}})
	return err
}

func discoverOIDC(ctx context.Context, issuer string) (*oidcDiscovery, error) {
	raw, err := getOIDCJSON(ctx, issuer+"/.well-known/openid-configuration")
	if err != nil {
		return nil, fmt.Errorf("fetch OIDC discovery: %w", err)
	}
	var d oidcDiscovery
	if err := json.Unmarshal(raw, &d); err != nil {
		return nil, fmt.Errorf("decode OIDC discovery: %w", err)
	}
	if strings.TrimRight(d.Issuer, "/") != issuer {
		return nil, errors.New("OIDC discovery issuer mismatch")
	}
	if d.AuthorizationEndpoint == "" || d.TokenEndpoint == "" {
		return nil, errors.New("OIDC discovery is missing required endpoints")
	}
	for _, endpoint := range []string{d.AuthorizationEndpoint, d.TokenEndpoint, d.DeviceAuthorizationEndpoint, d.RevocationEndpoint} {
		if endpoint != "" {
			if err := validateOIDCEndpoint(issuer, endpoint); err != nil {
				return nil, err
			}
		}
	}
	return &d, nil
}
func validateOIDCEndpoint(issuer, endpoint string) error {
	i, _ := url.Parse(issuer)
	e, err := url.Parse(endpoint)
	if err != nil || e.Scheme == "" || e.Host == "" {
		return errors.New("OIDC endpoint is not absolute")
	}
	if e.Scheme != i.Scheme || !strings.EqualFold(e.Host, i.Host) {
		return errors.New("OIDC endpoint origin does not match issuer")
	}
	if e.RawQuery != "" || e.Fragment != "" {
		return errors.New("OIDC endpoint must omit query and fragment")
	}
	return nil
}
func getOIDCJSON(ctx context.Context, endpoint string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	return doOIDCRequest(req)
}
func requestTokens(ctx context.Context, endpoint string, values url.Values) (*TokenResponse, error) {
	raw, err := postOIDCForm(ctx, endpoint, values)
	if err != nil {
		return nil, err
	}
	var tokens TokenResponse
	if err := json.Unmarshal(raw, &tokens); err != nil {
		return nil, err
	}
	if tokens.AccessToken == "" {
		return nil, errors.New("token response did not include an access token")
	}
	return &tokens, nil
}
func postOIDCForm(ctx context.Context, endpoint string, values url.Values) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(values.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	return doOIDCRequest(req)
}
func doOIDCRequest(req *http.Request) ([]byte, error) {
	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, newAuthHTTPError(resp, raw)
	}
	return raw, nil
}
func newAuthHTTPError(resp *http.Response, raw []byte) error {
	e := &AuthHTTPError{StatusCode: resp.StatusCode, Status: resp.Status}
	var p struct {
		Code             string `json:"error"`
		ErrorDescription string `json:"error_description"`
		Message          string `json:"message"`
	}
	if json.Unmarshal(raw, &p) == nil {
		e.Code = strings.TrimSpace(p.Code)
		e.Message = strings.TrimSpace(cmp.Or(p.ErrorDescription, p.Message))
	}
	return e
}
func openBrowser(target string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", target)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", target)
	default:
		cmd = exec.Command("xdg-open", target)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("open browser: %w", err)
	}
	return nil
}
