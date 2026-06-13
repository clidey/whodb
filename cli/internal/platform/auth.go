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
	"bytes"
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

const clientID = "whodb-cli"

// TokenResponse is returned by Mothergate auth exchange and refresh endpoints.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	IDToken      string `json:"id_token,omitempty"`
	ExpiresIn    int    `json:"expires_in,omitempty"`
}

// AuthHTTPError describes a failed Mothergate auth endpoint response.
type AuthHTTPError struct {
	StatusCode int
	Status     string
	Code       string
	Message    string
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

// IsInvalidGrant reports whether an auth failure means the refresh token is no longer usable.
func IsInvalidGrant(err error) bool {
	var authErr *AuthHTTPError
	return errors.As(err, &authErr) && authErr.Code == "invalid_grant"
}

// LoginOptions configures the browser PKCE login flow.
type LoginOptions struct {
	Host        string
	OpenBrowser bool
	PrintURL    func(string)
	Timeout     time.Duration
}

// Login runs the browser-based PKCE flow and returns platform tokens.
func Login(ctx context.Context, opts LoginOptions) (*TokenResponse, error) {
	host, err := NormalizeHost(opts.Host)
	if err != nil {
		return nil, err
	}
	authHost, err := ResolveAuthHost(ctx, host)
	if err != nil {
		return nil, err
	}
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("start login callback server: %w", err)
	}
	defer listener.Close()

	state := rand.Text()
	verifier := oauth2.GenerateVerifier()
	challenge := oauth2.S256ChallengeFromVerifier(verifier)

	redirectURI := "http://" + listener.Addr().String() + "/callback"
	loginURL := authHost + "/auth/login?" + url.Values{
		"redirect_uri":          {redirectURI},
		"code_challenge":        {challenge},
		"code_challenge_method": {"S256"},
		"client_id":             {clientID},
		"state":                 {state},
	}.Encode()

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)
	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/callback" {
				http.NotFound(w, r)
				return
			}
			if got := r.URL.Query().Get("state"); got != state {
				http.Error(w, "state mismatch", http.StatusBadRequest)
				errCh <- fmt.Errorf("login state mismatch")
				return
			}
			if oauthErr := r.URL.Query().Get("error"); oauthErr != "" {
				http.Error(w, oauthErr, http.StatusBadRequest)
				errCh <- fmt.Errorf("login failed: %s", oauthErr)
				return
			}
			code := r.URL.Query().Get("code")
			if code == "" {
				http.Error(w, "missing authorization code", http.StatusBadRequest)
				errCh <- fmt.Errorf("missing authorization code")
				return
			}
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			_, _ = w.Write([]byte("WhoDB CLI login complete. You can close this window.\n"))
			codeCh <- code
		}),
		ReadHeaderTimeout: 10 * time.Second,
	}
	defer server.Shutdown(context.Background())

	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			errCh <- err
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

	var code string
	select {
	case code = <-codeCh:
	case err := <-errCh:
		return nil, err
	case <-waitCtx.Done():
		return nil, fmt.Errorf("login timed out waiting for browser callback")
	}

	return ExchangeCode(ctx, authHost, code, redirectURI, verifier)
}

// ExchangeCode exchanges a PKCE authorization code for platform tokens.
func ExchangeCode(ctx context.Context, authHost, code, redirectURI, verifier string) (*TokenResponse, error) {
	payload := map[string]string{
		"code":         code,
		"redirectUri":  redirectURI,
		"codeVerifier": verifier,
		"clientId":     clientID,
	}
	return postAuth(ctx, authHost, "/auth/exchange", payload)
}

// RefreshToken exchanges a refresh token for new platform tokens.
func RefreshToken(ctx context.Context, host, refreshToken string) (*TokenResponse, error) {
	authHost, err := ResolveAuthHost(ctx, host)
	if err != nil {
		return nil, err
	}
	payload := map[string]string{
		"refreshToken": refreshToken,
		"clientId":     clientID,
	}
	return postAuth(ctx, authHost, "/auth/refresh", payload)
}

// Logout revokes the current authenticated platform session.
func Logout(ctx context.Context, host, accessToken string) error {
	authHost, err := ResolveAuthHost(ctx, host)
	if err != nil {
		return err
	}
	normalized, err := NormalizeHost(authHost)
	if err != nil {
		return err
	}
	if strings.TrimSpace(accessToken) == "" {
		return fmt.Errorf("access token is required")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, normalized+"/auth/revoke-current-session", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return newAuthHTTPError(resp, raw)
	}
	return nil
}

func postAuth(ctx context.Context, authHost, path string, payload any) (*TokenResponse, error) {
	normalized, err := NormalizeHost(authHost)
	if err != nil {
		return nil, err
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, normalized+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

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

	var tokens TokenResponse
	if err := json.Unmarshal(raw, &tokens); err != nil {
		return nil, err
	}
	if tokens.AccessToken == "" {
		return nil, fmt.Errorf("auth response did not include an access token")
	}
	return &tokens, nil
}

func newAuthHTTPError(resp *http.Response, raw []byte) error {
	authErr := &AuthHTTPError{
		StatusCode: resp.StatusCode,
		Status:     resp.Status,
	}
	var payload struct {
		Code             string `json:"error"`
		ErrorDescription string `json:"error_description"`
		Message          string `json:"message"`
	}
	if err := json.Unmarshal(raw, &payload); err == nil {
		authErr.Code = strings.TrimSpace(payload.Code)
		authErr.Message = strings.TrimSpace(cmp.Or(payload.ErrorDescription, payload.Message))
	}
	return authErr
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
