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

// Package platform contains the hosted WhoDB client used by CLI commands.
package platform

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	// DefaultHost is the hosted WhoDB platform URL used when no host is provided.
	DefaultHost = "https://app.whodb.com"
	defaultPath = "/api/query"
)

// Client sends authenticated requests to a hosted WhoDB platform endpoint.
type Client struct {
	host        string
	accessToken string
	httpClient  *http.Client
}

// AuthConfig is the public auth configuration advertised by a WhoDB platform host.
type AuthConfig struct {
	MothergateURL string `json:"mothergateUrl"`
}

// NormalizeHost canonicalizes hosted WhoDB URLs for config and requests.
func NormalizeHost(raw string) (string, error) {
	host := strings.TrimSpace(raw)
	if host == "" {
		return DefaultHost, nil
	}
	if !strings.Contains(host, "://") {
		host = "https://" + host
	}
	parsed, err := url.Parse(host)
	if err != nil {
		return "", fmt.Errorf("invalid host %q: %w", raw, err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("invalid host %q: scheme must be http or https", raw)
	}
	if parsed.Host == "" {
		return "", fmt.Errorf("invalid host %q: missing hostname", raw)
	}
	if parsed.Scheme == "http" && !isLoopbackHostname(parsed.Hostname()) {
		return "", fmt.Errorf("invalid host %q: http is only allowed for localhost or loopback addresses", raw)
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return strings.TrimRight(parsed.String(), "/"), nil
}

func isLoopbackHostname(hostname string) bool {
	host := strings.TrimSpace(hostname)
	if host == "" {
		return false
	}
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// ResolveAuthHost returns the Mothergate base URL advertised by the WhoDB host.
func ResolveAuthHost(ctx context.Context, host string) (string, error) {
	normalized, err := NormalizeHost(host)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, normalized+"/api/auth-config", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch auth config: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("auth config request failed: %s: %s", resp.Status, strings.TrimSpace(string(raw)))
	}

	var cfg AuthConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return "", fmt.Errorf("decode auth config: %w", err)
	}
	if strings.TrimSpace(cfg.MothergateURL) == "" {
		return "", fmt.Errorf("auth config did not include mothergateUrl")
	}
	authHost, err := NormalizeHost(cfg.MothergateURL)
	if err != nil {
		return "", fmt.Errorf("invalid mothergateUrl in auth config: %w", err)
	}
	return authHost, nil
}

// NewClient creates a hosted WhoDB platform client.
func NewClient(host, accessToken string) (*Client, error) {
	normalized, err := NormalizeHost(host)
	if err != nil {
		return nil, err
	}
	return &Client{
		host:        normalized,
		accessToken: accessToken,
		httpClient:  &http.Client{Timeout: 30 * time.Second},
	}, nil
}

// Host returns the canonical hosted WhoDB platform URL.
func (c *Client) Host() string {
	return c.host
}

// Me returns the authenticated platform user.
func (c *Client) Me(ctx context.Context) (*User, error) {
	var resp struct {
		Me *User `json:"Me"`
	}
	if err := c.graphQL(ctx, operationMe, nil, &resp); err != nil {
		return nil, err
	}
	if resp.Me == nil {
		return nil, fmt.Errorf("platform returned no user")
	}
	return resp.Me, nil
}

// Organizations returns organizations visible to the authenticated user.
func (c *Client) Organizations(ctx context.Context) ([]Organization, error) {
	var resp struct {
		MyOrganizations []Organization `json:"MyOrganizations"`
	}
	err := c.graphQL(ctx, operationOrganizations, nil, &resp)
	return resp.MyOrganizations, err
}

// Projects returns projects visible in one organization.
func (c *Client) Projects(ctx context.Context, orgID string) ([]Project, error) {
	var resp struct {
		Projects []Project `json:"Projects"`
	}
	variables := map[string]any{"orgId": orgID}
	err := c.graphQL(ctx, operationProjects, variables, &resp)
	return resp.Projects, err
}

// SwitchOrganization updates the user's active organization on the hosted platform.
func (c *Client) SwitchOrganization(ctx context.Context, orgID string) (*Organization, error) {
	var resp struct {
		SwitchOrganization *Organization `json:"SwitchOrganization"`
	}
	variables := map[string]any{"orgId": orgID}
	if err := c.graphQL(ctx, operationSwitchOrganization, variables, &resp); err != nil {
		return nil, err
	}
	if resp.SwitchOrganization == nil {
		return nil, fmt.Errorf("platform returned no organization")
	}
	return resp.SwitchOrganization, nil
}

// ProjectSources returns sources visible to the user in one project.
func (c *Client) ProjectSources(ctx context.Context, projectID string) ([]Source, error) {
	var resp struct {
		ProjectSources []Source `json:"ProjectSources"`
	}
	variables := map[string]any{"projectId": projectID}
	err := c.graphQL(ctx, operationProjectSources, variables, &resp)
	return resp.ProjectSources, err
}

// CreateSource creates a hosted source in one project.
func (c *Client) CreateSource(ctx context.Context, input CreateSourceInput) (*Source, error) {
	var resp struct {
		CreateSource *Source `json:"CreateSource"`
	}
	variables := map[string]any{"input": input.graphQLInput()}
	if err := c.graphQL(ctx, operationCreateSource, variables, &resp); err != nil {
		return nil, err
	}
	if resp.CreateSource == nil {
		return nil, fmt.Errorf("platform returned no source")
	}
	return resp.CreateSource, nil
}

// DeleteSource deletes a hosted source from one project.
func (c *Client) DeleteSource(ctx context.Context, projectID, sourceID string) error {
	var resp struct {
		DeleteSource *struct {
			Status bool `json:"Status"`
		} `json:"DeleteSource"`
	}
	variables := map[string]any{"projectId": projectID, "id": sourceID}
	if err := c.graphQL(ctx, operationDeleteSource, variables, &resp); err != nil {
		return err
	}
	if resp.DeleteSource == nil || !resp.DeleteSource.Status {
		return fmt.Errorf("platform did not confirm source deletion")
	}
	return nil
}

func (c *Client) graphQL(ctx context.Context, query string, variables any, target any) error {
	body, err := json.Marshal(map[string]any{
		"query":     query,
		"variables": variables,
	})
	if err != nil {
		return err
	}

	endpoint := c.host + defaultPath
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if c.accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.accessToken)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("platform request failed: %s: %s", resp.Status, strings.TrimSpace(string(raw)))
	}

	var envelope struct {
		Data   json.RawMessage `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return err
	}
	if len(envelope.Errors) > 0 {
		return fmt.Errorf("platform GraphQL error: %s", envelope.Errors[0].Message)
	}
	if len(envelope.Data) == 0 || string(envelope.Data) == "null" {
		return fmt.Errorf("platform returned no data")
	}
	return json.Unmarshal(envelope.Data, target)
}
