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
	"errors"
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
	host              string
	accessToken       string
	httpClient        *http.Client
	manifest          *PlatformManifest
	manifestRefresher ManifestRefresher
}

// ManifestRefresher reloads a hosted platform manifest after schema validation drift.
type ManifestRefresher func(context.Context, *Client) (*PlatformManifest, error)

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

// SetPlatformManifest attaches a cached hosted platform manifest to the client.
func (c *Client) SetPlatformManifest(manifest *PlatformManifest) {
	c.manifest = manifest
}

// SetManifestRefresher sets the callback used for one-shot schema refresh retries.
func (c *Client) SetManifestRefresher(refresher ManifestRefresher) {
	c.manifestRefresher = refresher
}

// RequireOperation returns an error when the cached manifest does not publish an operation.
func (c *Client) RequireOperation(kind, name, feature string) error {
	if c.manifest == nil {
		return nil
	}
	return c.manifest.RequireOperation(kind, name, feature)
}

// PlatformManifest returns the hosted platform contract advertised to the CLI.
func (c *Client) PlatformManifest(ctx context.Context) (*PlatformManifest, error) {
	var resp struct {
		PlatformManifest *PlatformManifest `json:"PlatformManifest"`
	}
	if err := c.graphQLOnce(ctx, operationPlatformManifest, nil, &resp); err != nil {
		return nil, err
	}
	if resp.PlatformManifest == nil {
		return nil, fmt.Errorf("platform returned no manifest")
	}
	c.manifest = resp.PlatformManifest
	return resp.PlatformManifest, nil
}

// PlatformVersion returns the hosted platform version string.
func (c *Client) PlatformVersion(ctx context.Context) (string, error) {
	var resp struct {
		Version string `json:"Version"`
	}
	if err := c.graphQLOnce(ctx, operationPlatformVersion, nil, &resp); err != nil {
		return "", err
	}
	return strings.TrimSpace(resp.Version), nil
}

// Me returns the authenticated platform user.
func (c *Client) Me(ctx context.Context) (*User, error) {
	var resp struct {
		Me *User `json:"Me"`
	}
	query := func() string {
		fields := []string{"id", "email", "displayName"}
		if c.manifest != nil {
			fields = c.manifest.SelectFields("PlatformUser", []string{"id", "email", "displayName", "orgId"})
		}
		if len(fields) == 0 {
			fields = []string{"id", "email", "displayName"}
		}
		return operationMeForFields(fields)
	}
	if err := c.graphQLWithRetry(ctx, query, nil, &resp); err != nil {
		return nil, err
	}
	if resp.Me == nil {
		return nil, fmt.Errorf("platform returned no user")
	}
	return resp.Me, nil
}

// Organizations returns organizations visible to the authenticated user.
func (c *Client) Organizations(ctx context.Context) ([]Organization, error) {
	if err := c.RequireOperation("Query", "MyOrganizations", "organization listing"); err != nil {
		return nil, err
	}
	var resp struct {
		MyOrganizations []Organization `json:"MyOrganizations"`
	}
	err := c.graphQL(ctx, operationOrganizations, nil, &resp)
	return resp.MyOrganizations, err
}

// Projects returns projects visible in one organization.
func (c *Client) Projects(ctx context.Context, orgID string) ([]Project, error) {
	if err := c.RequireOperation("Query", "Projects", "project listing"); err != nil {
		return nil, err
	}
	var resp struct {
		Projects []Project `json:"Projects"`
	}
	variables := map[string]any{"orgId": orgID}
	err := c.graphQL(ctx, operationProjects, variables, &resp)
	return resp.Projects, err
}

// ProjectSources returns sources visible to the user in one project.
func (c *Client) ProjectSources(ctx context.Context, orgID, projectID string) ([]Source, error) {
	if err := c.RequireOperation("Query", "ProjectSources", "source listing"); err != nil {
		return nil, err
	}
	var resp struct {
		ProjectSources []Source `json:"ProjectSources"`
	}
	variables := map[string]any{"orgId": orgID, "projectId": projectID}
	err := c.graphQL(ctx, operationProjectSources, variables, &resp)
	return resp.ProjectSources, err
}

// SourceTypes returns source types available on the hosted platform.
func (c *Client) SourceTypes(ctx context.Context) ([]SourceType, error) {
	if err := c.RequireOperation("Query", "SourceTypes", "source type discovery"); err != nil {
		return nil, err
	}
	var resp struct {
		SourceTypes []SourceType `json:"SourceTypes"`
	}
	err := c.graphQL(ctx, operationSourceTypes, nil, &resp)
	return resp.SourceTypes, err
}

// CreateSource creates a hosted source in one project.
func (c *Client) CreateSource(ctx context.Context, input CreateSourceInput) (*Source, error) {
	if err := c.RequireOperation("Mutation", "CreateSource", "source creation"); err != nil {
		return nil, err
	}
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

// SourceConfig returns one hosted source's connection configuration.
func (c *Client) SourceConfig(ctx context.Context, orgID, projectID, sourceID string) (*SourceConfig, error) {
	if err := c.RequireOperation("Query", "SourceConfig", "source config reads"); err != nil {
		return nil, err
	}
	type sourceConfigResponse struct {
		Hostname string   `json:"hostname"`
		Port     string   `json:"port"`
		Username string   `json:"username"`
		Password string   `json:"password"`
		Database string   `json:"database"`
		Advanced []Record `json:"advanced"`
	}
	var resp struct {
		SourceConfig *sourceConfigResponse `json:"SourceConfig"`
	}
	variables := map[string]any{"orgId": orgID, "projectId": projectID, "sourceId": sourceID}
	if err := c.graphQL(ctx, operationSourceConfig, variables, &resp); err != nil {
		return nil, err
	}
	if resp.SourceConfig == nil {
		return nil, fmt.Errorf("platform returned no source config")
	}
	config := &SourceConfig{
		Hostname: resp.SourceConfig.Hostname,
		Port:     resp.SourceConfig.Port,
		Username: resp.SourceConfig.Username,
		Password: resp.SourceConfig.Password,
		Database: resp.SourceConfig.Database,
		Advanced: map[string]string{},
	}
	for _, record := range resp.SourceConfig.Advanced {
		config.Advanced[record.Key] = record.Value
	}
	return config, nil
}

// UpdateSource updates one hosted source's metadata or connection configuration.
func (c *Client) UpdateSource(ctx context.Context, input UpdateSourceInput) (*Source, error) {
	if err := c.RequireOperation("Mutation", "UpdateSource", "source updates"); err != nil {
		return nil, err
	}
	var resp struct {
		UpdateSource *Source `json:"UpdateSource"`
	}
	variables := map[string]any{"input": input.graphQLInput()}
	if err := c.graphQL(ctx, operationUpdateSource, variables, &resp); err != nil {
		return nil, err
	}
	if resp.UpdateSource == nil {
		return nil, fmt.Errorf("platform returned no source")
	}
	return resp.UpdateSource, nil
}

// TestSourceConnection checks whether a draft source configuration can connect.
func (c *Client) TestSourceConnection(ctx context.Context, input CreateSourceInput) error {
	if err := c.RequireOperation("Mutation", "TestSourceConnection", "source connection tests"); err != nil {
		return err
	}
	var resp struct {
		TestSourceConnection *struct {
			Status bool `json:"Status"`
		} `json:"TestSourceConnection"`
	}
	variables := map[string]any{"credentials": input.sourceLoginInput()}
	if err := c.graphQL(ctx, operationTestSourceConnection, variables, &resp); err != nil {
		return err
	}
	if resp.TestSourceConnection == nil || !resp.TestSourceConnection.Status {
		return fmt.Errorf("platform did not confirm source connection")
	}
	return nil
}

// DeleteSource deletes a hosted source from one project.
func (c *Client) DeleteSource(ctx context.Context, orgID, projectID, sourceID string) error {
	if err := c.RequireOperation("Mutation", "DeleteSource", "source deletion"); err != nil {
		return err
	}
	var resp struct {
		DeleteSource *struct {
			Status bool `json:"Status"`
		} `json:"DeleteSource"`
	}
	variables := map[string]any{"orgId": orgID, "projectId": projectID, "id": sourceID}
	if err := c.graphQL(ctx, operationDeleteSource, variables, &resp); err != nil {
		return err
	}
	if resp.DeleteSource == nil || !resp.DeleteSource.Status {
		return fmt.Errorf("platform did not confirm source deletion")
	}
	return nil
}

// SourceObjects returns browseable objects for one hosted source.
func (c *Client) SourceObjects(ctx context.Context, orgID, projectID, sourceID string, parent *SourceObjectRefInput, kinds []SourceObjectKind, pageSize, pageOffset int) ([]SourceObject, error) {
	if err := c.RequireOperation("Query", "PlatformSourceObjects", "source object browsing"); err != nil {
		return nil, err
	}
	var resp struct {
		PlatformSourceObjects []SourceObject `json:"PlatformSourceObjects"`
	}
	var parentVariable any
	if parent != nil {
		parentVariable = parent.graphQLInput()
	}
	variables := map[string]any{
		"orgId":      orgID,
		"projectId":  projectID,
		"sourceId":   sourceID,
		"parent":     parentVariable,
		"kinds":      kinds,
		"pageSize":   pageSize,
		"pageOffset": pageOffset,
	}
	err := c.graphQL(ctx, operationPlatformSourceObjects, variables, &resp)
	return resp.PlatformSourceObjects, err
}

// SourceColumns returns columns for one hosted source object.
func (c *Client) SourceColumns(ctx context.Context, orgID, projectID, sourceID string, ref SourceObjectRefInput) ([]Column, error) {
	if err := c.RequireOperation("Query", "PlatformSourceColumns", "source column inspection"); err != nil {
		return nil, err
	}
	var resp struct {
		PlatformSourceColumns []Column `json:"PlatformSourceColumns"`
	}
	variables := map[string]any{
		"orgId":     orgID,
		"projectId": projectID,
		"sourceId":  sourceID,
		"ref":       ref.graphQLInput(),
	}
	err := c.graphQL(ctx, operationPlatformSourceColumns, variables, &resp)
	return resp.PlatformSourceColumns, err
}

// SourceRows returns rows for one hosted source object.
func (c *Client) SourceRows(ctx context.Context, orgID, projectID, sourceID string, ref SourceObjectRefInput, pageSize, pageOffset int) (*RowsResult, error) {
	if err := c.RequireOperation("Query", "PlatformSourceRows", "source row previews"); err != nil {
		return nil, err
	}
	var resp struct {
		PlatformSourceRows *RowsResult `json:"PlatformSourceRows"`
	}
	variables := map[string]any{
		"orgId":      orgID,
		"projectId":  projectID,
		"sourceId":   sourceID,
		"ref":        ref.graphQLInput(),
		"pageSize":   pageSize,
		"pageOffset": pageOffset,
	}
	if err := c.graphQL(ctx, operationPlatformSourceRows, variables, &resp); err != nil {
		return nil, err
	}
	if resp.PlatformSourceRows == nil {
		return nil, fmt.Errorf("platform returned no rows")
	}
	return resp.PlatformSourceRows, nil
}

func (c *Client) graphQL(ctx context.Context, query string, variables any, target any) error {
	return c.graphQLWithRetry(ctx, func() string { return query }, variables, target)
}

func (c *Client) graphQLWithRetry(ctx context.Context, query func() string, variables any, target any) error {
	err := c.graphQLOnce(ctx, query(), variables, target)
	if !IsGraphQLValidationError(err) || c.manifestRefresher == nil {
		return err
	}
	if _, refreshErr := c.manifestRefresher(ctx, c); refreshErr != nil {
		return err
	}
	return c.graphQLOnce(ctx, query(), variables, target)
}

func (c *Client) graphQLOnce(ctx context.Context, query string, variables any, target any) error {
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
			Message    string `json:"message"`
			Extensions struct {
				Code string `json:"code"`
			} `json:"extensions"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return err
	}
	if len(envelope.Errors) > 0 {
		first := envelope.Errors[0]
		return GraphQLError{Message: first.Message, Code: first.Extensions.Code}
	}
	if len(envelope.Data) == 0 || string(envelope.Data) == "null" {
		return fmt.Errorf("platform returned no data")
	}
	return json.Unmarshal(envelope.Data, target)
}

// GraphQLError is a GraphQL response error from the hosted platform.
type GraphQLError struct {
	Message string
	Code    string
}

func (e GraphQLError) Error() string {
	if e.Code == "" {
		return fmt.Sprintf("platform GraphQL error: %s", e.Message)
	}
	return fmt.Sprintf("platform GraphQL error: %s: %s", e.Code, e.Message)
}

// IsGraphQLValidationError reports whether err is a GraphQL validation error.
func IsGraphQLValidationError(err error) bool {
	var graphErr GraphQLError
	return errors.As(err, &graphErr) && graphErr.Code == "GRAPHQL_VALIDATION_FAILED"
}
