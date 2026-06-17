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

package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/clidey/whodb/cli/internal/config"
	platformapi "github.com/clidey/whodb/cli/internal/platform"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const defaultPlatformRowLimit = 50

type platformClient interface {
	Me(context.Context) (*platformapi.User, error)
	PlatformManifest(context.Context) (*platformapi.PlatformManifest, error)
	ProjectSources(context.Context, string, string) ([]platformapi.Source, error)
	SourceObjects(context.Context, string, string, string, *platformapi.SourceObjectRefInput, []platformapi.SourceObjectKind, int, int) ([]platformapi.SourceObject, error)
	SourceColumns(context.Context, string, string, string, platformapi.SourceObjectRefInput) ([]platformapi.Column, error)
	SourceRows(context.Context, string, string, string, platformapi.SourceObjectRefInput, int, int) (*platformapi.RowsResult, error)
}

type platformToolSession struct {
	Host   config.PlatformHost
	Client platformClient
}

var loadPlatformToolSession = loadHostedPlatformToolSession

// PlatformStatusInput is the input for the whodb_platform_status tool.
type PlatformStatusInput struct{}

// PlatformStatusOutput reports hosted WhoDB login and selected workspace state.
type PlatformStatusOutput struct {
	Host                    string `json:"host,omitempty"`
	UserID                  string `json:"user_id,omitempty"`
	Email                   string `json:"email,omitempty"`
	DefaultOrgID            string `json:"default_org_id,omitempty"`
	DefaultOrgName          string `json:"default_org_name,omitempty"`
	DefaultProjectID        string `json:"default_project_id,omitempty"`
	DefaultProjectName      string `json:"default_project_name,omitempty"`
	WorkspaceSelected       bool   `json:"workspace_selected"`
	PlatformVersion         string `json:"platform_version,omitempty"`
	ManifestProtocolVersion string `json:"manifest_protocol_version,omitempty"`
	Error                   string `json:"error,omitempty"`
	RequestID               string `json:"request_id,omitempty"`
}

// PlatformSourcesInput is the input for the whodb_platform_sources tool.
type PlatformSourcesInput struct{}

// PlatformSourcesOutput lists hosted sources in the selected project.
type PlatformSourcesOutput struct {
	Host      string               `json:"host,omitempty"`
	OrgID     string               `json:"org_id,omitempty"`
	ProjectID string               `json:"project_id,omitempty"`
	Sources   []platformapi.Source `json:"sources"`
	Error     string               `json:"error,omitempty"`
	RequestID string               `json:"request_id,omitempty"`
}

// MarshalJSON ensures nil slices are serialized as [] instead of null.
func (o PlatformSourcesOutput) MarshalJSON() ([]byte, error) {
	if o.Sources == nil {
		o.Sources = []platformapi.Source{}
	}
	type Alias PlatformSourcesOutput
	return json.Marshal(Alias(o))
}

// PlatformSourceObjectsInput is the input for the whodb_platform_source_objects tool.
type PlatformSourceObjectsInput struct {
	Source     string   `json:"source" jsonschema:"Hosted source id or name"`
	Parent     string   `json:"parent,omitempty" jsonschema:"Parent object ref as kind:path, for example schema:public"`
	Kinds      []string `json:"kinds,omitempty" jsonschema:"Object kinds to include, for example Table or View"`
	PageSize   int      `json:"page_size,omitempty" jsonschema:"Maximum objects to return"`
	PageOffset int      `json:"page_offset,omitempty" jsonschema:"Object offset"`
}

// PlatformSourceObjectsOutput lists hosted source objects.
type PlatformSourceObjectsOutput struct {
	Objects   []platformapi.SourceObject `json:"objects"`
	Error     string                     `json:"error,omitempty"`
	RequestID string                     `json:"request_id,omitempty"`
}

// MarshalJSON ensures nil slices are serialized as [] instead of null.
func (o PlatformSourceObjectsOutput) MarshalJSON() ([]byte, error) {
	if o.Objects == nil {
		o.Objects = []platformapi.SourceObject{}
	}
	type Alias PlatformSourceObjectsOutput
	return json.Marshal(Alias(o))
}

// PlatformSourceColumnsInput is the input for the whodb_platform_source_columns tool.
type PlatformSourceColumnsInput struct {
	Source string `json:"source" jsonschema:"Hosted source id or name"`
	Ref    string `json:"ref" jsonschema:"Object ref as kind:path, for example table:public.users"`
}

// PlatformSourceColumnsOutput lists columns for one hosted source object.
type PlatformSourceColumnsOutput struct {
	Columns   []platformapi.Column `json:"columns"`
	Error     string               `json:"error,omitempty"`
	RequestID string               `json:"request_id,omitempty"`
}

// MarshalJSON ensures nil slices are serialized as [] instead of null.
func (o PlatformSourceColumnsOutput) MarshalJSON() ([]byte, error) {
	if o.Columns == nil {
		o.Columns = []platformapi.Column{}
	}
	type Alias PlatformSourceColumnsOutput
	return json.Marshal(Alias(o))
}

// PlatformSourceRowsInput is the input for the whodb_platform_source_rows tool.
type PlatformSourceRowsInput struct {
	Source string `json:"source" jsonschema:"Hosted source id or name"`
	Ref    string `json:"ref" jsonschema:"Object ref as kind:path, for example table:public.users"`
	Limit  int    `json:"limit,omitempty" jsonschema:"Maximum rows to return"`
	Offset int    `json:"offset,omitempty" jsonschema:"Row offset"`
}

// PlatformSourceRowsOutput previews rows for one hosted source object.
type PlatformSourceRowsOutput struct {
	Columns   []platformapi.Column `json:"columns"`
	Rows      [][]string           `json:"rows"`
	Total     int                  `json:"total"`
	Truncated bool                 `json:"truncated"`
	Error     string               `json:"error,omitempty"`
	RequestID string               `json:"request_id,omitempty"`
}

// MarshalJSON ensures nil slices are serialized as [] instead of null.
func (o PlatformSourceRowsOutput) MarshalJSON() ([]byte, error) {
	if o.Columns == nil {
		o.Columns = []platformapi.Column{}
	}
	if o.Rows == nil {
		o.Rows = [][]string{}
	}
	type Alias PlatformSourceRowsOutput
	return json.Marshal(Alias(o))
}

func registerPlatformTools(server *mcp.Server, secOpts *SecurityOptions) {
	for _, tool := range platformToolDefinitions() {
		switch tool.Name {
		case "whodb_platform_status":
			mcp.AddTool(server, tool, createPlatformStatusHandler())
		case "whodb_platform_sources":
			mcp.AddTool(server, tool, createPlatformSourcesHandler())
		case "whodb_platform_source_objects":
			mcp.AddTool(server, tool, createPlatformSourceObjectsHandler())
		case "whodb_platform_source_columns":
			mcp.AddTool(server, tool, createPlatformSourceColumnsHandler())
		case "whodb_platform_source_rows":
			mcp.AddTool(server, tool, createPlatformSourceRowsHandler(secOpts))
		}
	}
}

func platformToolDefinitions() []*mcp.Tool {
	return []*mcp.Tool{
		{
			Name:        "whodb_platform_status",
			Description: descPlatformStatus,
			Annotations: platformReadOnlyAnnotations("Hosted WhoDB Status"),
		},
		{
			Name:        "whodb_platform_sources",
			Description: descPlatformSources,
			Annotations: platformReadOnlyAnnotations("List Hosted Sources"),
		},
		{
			Name:        "whodb_platform_source_objects",
			Description: descPlatformSourceObjects,
			Annotations: platformReadOnlyAnnotations("Browse Hosted Source Objects"),
		},
		{
			Name:        "whodb_platform_source_columns",
			Description: descPlatformSourceColumns,
			Annotations: platformReadOnlyAnnotations("Inspect Hosted Source Columns"),
		},
		{
			Name:        "whodb_platform_source_rows",
			Description: descPlatformSourceRows,
			Annotations: platformReadOnlyAnnotations("Preview Hosted Source Rows"),
		},
	}
}

func platformReadOnlyAnnotations(title string) *mcp.ToolAnnotations {
	return &mcp.ToolAnnotations{
		Title:          title,
		ReadOnlyHint:   true,
		IdempotentHint: true,
		OpenWorldHint:  boolPtr(true),
	}
}

func createPlatformStatusHandler() func(context.Context, *mcp.CallToolRequest, PlatformStatusInput) (*mcp.CallToolResult, PlatformStatusOutput, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input PlatformStatusInput) (*mcp.CallToolResult, PlatformStatusOutput, error) {
		return HandlePlatformStatus(ctx, req, input)
	}
}

func createPlatformSourcesHandler() func(context.Context, *mcp.CallToolRequest, PlatformSourcesInput) (*mcp.CallToolResult, PlatformSourcesOutput, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input PlatformSourcesInput) (*mcp.CallToolResult, PlatformSourcesOutput, error) {
		return HandlePlatformSources(ctx, req, input)
	}
}

func createPlatformSourceObjectsHandler() func(context.Context, *mcp.CallToolRequest, PlatformSourceObjectsInput) (*mcp.CallToolResult, PlatformSourceObjectsOutput, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input PlatformSourceObjectsInput) (*mcp.CallToolResult, PlatformSourceObjectsOutput, error) {
		return HandlePlatformSourceObjects(ctx, req, input)
	}
}

func createPlatformSourceColumnsHandler() func(context.Context, *mcp.CallToolRequest, PlatformSourceColumnsInput) (*mcp.CallToolResult, PlatformSourceColumnsOutput, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input PlatformSourceColumnsInput) (*mcp.CallToolResult, PlatformSourceColumnsOutput, error) {
		return HandlePlatformSourceColumns(ctx, req, input)
	}
}

func createPlatformSourceRowsHandler(secOpts *SecurityOptions) func(context.Context, *mcp.CallToolRequest, PlatformSourceRowsInput) (*mcp.CallToolResult, PlatformSourceRowsOutput, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input PlatformSourceRowsInput) (*mcp.CallToolResult, PlatformSourceRowsOutput, error) {
		return HandlePlatformSourceRows(ctx, req, input, secOpts)
	}
}

// HandlePlatformStatus reports hosted WhoDB login and workspace state.
func HandlePlatformStatus(ctx context.Context, req *mcp.CallToolRequest, input PlatformStatusInput) (*mcp.CallToolResult, PlatformStatusOutput, error) {
	requestID := generateRequestID("platform_status")
	startTime := time.Now()

	session, err := loadPlatformToolSession(ctx)
	if err != nil {
		TrackToolCall(ctx, "platform_status", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "platform_session"})
		return nil, PlatformStatusOutput{Error: err.Error(), RequestID: requestID}, nil
	}
	user, err := session.Client.Me(ctx)
	if err != nil {
		TrackToolCall(ctx, "platform_status", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "platform_user"})
		return nil, PlatformStatusOutput{Error: err.Error(), RequestID: requestID}, nil
	}
	manifest, err := session.Client.PlatformManifest(ctx)
	if err != nil {
		TrackToolCall(ctx, "platform_status", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "platform_manifest"})
		return nil, PlatformStatusOutput{Error: err.Error(), RequestID: requestID}, nil
	}

	output := PlatformStatusOutput{
		Host:                    session.Host.URL,
		UserID:                  user.ID,
		Email:                   user.Email,
		DefaultOrgID:            session.Host.DefaultOrgID,
		DefaultOrgName:          session.Host.DefaultOrgName,
		DefaultProjectID:        session.Host.DefaultProjectID,
		DefaultProjectName:      session.Host.DefaultProjectName,
		WorkspaceSelected:       hasPlatformWorkspace(session),
		PlatformVersion:         manifest.PlatformVersion,
		ManifestProtocolVersion: manifest.ManifestProtocolVersion,
		RequestID:               requestID,
	}
	TrackToolCall(ctx, "platform_status", requestID, true, time.Since(startTime).Milliseconds(), map[string]any{"workspace_selected": output.WorkspaceSelected})
	return nil, output, nil
}

// HandlePlatformSources lists hosted sources in the selected workspace.
func HandlePlatformSources(ctx context.Context, req *mcp.CallToolRequest, input PlatformSourcesInput) (*mcp.CallToolResult, PlatformSourcesOutput, error) {
	requestID := generateRequestID("platform_sources")
	startTime := time.Now()

	session, err := loadPlatformWorkspace(ctx)
	if err != nil {
		TrackToolCall(ctx, "platform_sources", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "platform_session"})
		return nil, PlatformSourcesOutput{Error: err.Error(), RequestID: requestID}, nil
	}
	sources, err := session.Client.ProjectSources(ctx, session.Host.DefaultOrgID, session.Host.DefaultProjectID)
	if err != nil {
		TrackToolCall(ctx, "platform_sources", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "platform_query"})
		return nil, PlatformSourcesOutput{Error: err.Error(), RequestID: requestID}, nil
	}

	TrackToolCall(ctx, "platform_sources", requestID, true, time.Since(startTime).Milliseconds(), map[string]any{"source_count": len(sources)})
	return nil, PlatformSourcesOutput{
		Host:      session.Host.URL,
		OrgID:     session.Host.DefaultOrgID,
		ProjectID: session.Host.DefaultProjectID,
		Sources:   sources,
		RequestID: requestID,
	}, nil
}

// HandlePlatformSourceObjects lists objects in one hosted source.
func HandlePlatformSourceObjects(ctx context.Context, req *mcp.CallToolRequest, input PlatformSourceObjectsInput) (*mcp.CallToolResult, PlatformSourceObjectsOutput, error) {
	requestID := generateRequestID("platform_source_objects")
	startTime := time.Now()

	session, source, err := loadPlatformSource(ctx, input.Source)
	if err != nil {
		TrackToolCall(ctx, "platform_source_objects", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "platform_source"})
		return nil, PlatformSourceObjectsOutput{Error: err.Error(), RequestID: requestID}, nil
	}
	parent, err := parsePlatformOptionalRef(input.Parent)
	if err != nil {
		TrackToolCall(ctx, "platform_source_objects", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "validation"})
		return nil, PlatformSourceObjectsOutput{Error: err.Error(), RequestID: requestID}, nil
	}
	kinds := make([]platformapi.SourceObjectKind, 0, len(input.Kinds))
	for _, kind := range input.Kinds {
		if strings.TrimSpace(kind) != "" {
			kinds = append(kinds, platformapi.SourceObjectKind(strings.TrimSpace(kind)))
		}
	}
	pageSize := input.PageSize
	if pageSize <= 0 {
		pageSize = defaultPlatformRowLimit
	}
	if input.PageOffset < 0 {
		return nil, PlatformSourceObjectsOutput{Error: "page_offset must be non-negative", RequestID: requestID}, nil
	}
	objects, err := session.Client.SourceObjects(ctx, session.Host.DefaultOrgID, session.Host.DefaultProjectID, source.ID, parent, kinds, pageSize, input.PageOffset)
	if err != nil {
		TrackToolCall(ctx, "platform_source_objects", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "platform_query"})
		return nil, PlatformSourceObjectsOutput{Error: err.Error(), RequestID: requestID}, nil
	}

	TrackToolCall(ctx, "platform_source_objects", requestID, true, time.Since(startTime).Milliseconds(), map[string]any{"object_count": len(objects)})
	return nil, PlatformSourceObjectsOutput{Objects: objects, RequestID: requestID}, nil
}

// HandlePlatformSourceColumns returns columns for one hosted source object.
func HandlePlatformSourceColumns(ctx context.Context, req *mcp.CallToolRequest, input PlatformSourceColumnsInput) (*mcp.CallToolResult, PlatformSourceColumnsOutput, error) {
	requestID := generateRequestID("platform_source_columns")
	startTime := time.Now()

	session, source, err := loadPlatformSource(ctx, input.Source)
	if err != nil {
		TrackToolCall(ctx, "platform_source_columns", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "platform_source"})
		return nil, PlatformSourceColumnsOutput{Error: err.Error(), RequestID: requestID}, nil
	}
	ref, err := parsePlatformRequiredRef(input.Ref)
	if err != nil {
		TrackToolCall(ctx, "platform_source_columns", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "validation"})
		return nil, PlatformSourceColumnsOutput{Error: err.Error(), RequestID: requestID}, nil
	}
	columns, err := session.Client.SourceColumns(ctx, session.Host.DefaultOrgID, session.Host.DefaultProjectID, source.ID, ref)
	if err != nil {
		TrackToolCall(ctx, "platform_source_columns", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "platform_query"})
		return nil, PlatformSourceColumnsOutput{Error: err.Error(), RequestID: requestID}, nil
	}

	TrackToolCall(ctx, "platform_source_columns", requestID, true, time.Since(startTime).Milliseconds(), map[string]any{"column_count": len(columns)})
	return nil, PlatformSourceColumnsOutput{Columns: columns, RequestID: requestID}, nil
}

// HandlePlatformSourceRows previews rows for one hosted source object.
func HandlePlatformSourceRows(ctx context.Context, req *mcp.CallToolRequest, input PlatformSourceRowsInput, secOpts *SecurityOptions) (*mcp.CallToolResult, PlatformSourceRowsOutput, error) {
	requestID := generateRequestID("platform_source_rows")
	startTime := time.Now()

	session, source, err := loadPlatformSource(ctx, input.Source)
	if err != nil {
		TrackToolCall(ctx, "platform_source_rows", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "platform_source"})
		return nil, PlatformSourceRowsOutput{Error: err.Error(), RequestID: requestID}, nil
	}
	ref, err := parsePlatformRequiredRef(input.Ref)
	if err != nil {
		TrackToolCall(ctx, "platform_source_rows", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "validation"})
		return nil, PlatformSourceRowsOutput{Error: err.Error(), RequestID: requestID}, nil
	}
	limit := platformRowsLimit(input.Limit, secOpts)
	if input.Offset < 0 {
		return nil, PlatformSourceRowsOutput{Error: "offset must be non-negative", RequestID: requestID}, nil
	}
	rows, err := session.Client.SourceRows(ctx, session.Host.DefaultOrgID, session.Host.DefaultProjectID, source.ID, ref, limit, input.Offset)
	if err != nil {
		TrackToolCall(ctx, "platform_source_rows", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "platform_query"})
		return nil, PlatformSourceRowsOutput{Error: err.Error(), RequestID: requestID}, nil
	}

	output := PlatformSourceRowsOutput{
		Columns:   rows.Columns,
		Rows:      rows.Rows,
		Total:     rows.TotalCount,
		Truncated: rows.TotalCount > len(rows.Rows),
		RequestID: requestID,
	}
	TrackToolCall(ctx, "platform_source_rows", requestID, true, time.Since(startTime).Milliseconds(), map[string]any{"row_count": len(output.Rows), "truncated": output.Truncated})
	return nil, output, nil
}

func loadHostedPlatformToolSession(ctx context.Context) (*platformToolSession, error) {
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("cannot load hosted WhoDB config: %w", err)
	}
	hostURL := strings.TrimSpace(cfg.Platform.DefaultHost)
	if hostURL == "" {
		hostURL = platformapi.DefaultHost
	}
	hostURL, err = platformapi.NormalizeHost(hostURL)
	if err != nil {
		return nil, err
	}
	host, ok := cfg.GetPlatformHost(hostURL)
	if !ok || strings.TrimSpace(host.AccountID) == "" {
		return nil, fmt.Errorf("hosted WhoDB is not logged in for %s. Run: whodb-cli login --host %s", hostURL, hostURL)
	}
	refreshToken, err := cfg.GetPlatformRefreshToken(hostURL, host.AccountID)
	if err != nil {
		return nil, fmt.Errorf("cannot load hosted WhoDB refresh token. Run: whodb-cli login --host %s", hostURL)
	}
	tokens, err := platformapi.RefreshToken(ctx, hostURL, refreshToken)
	if err != nil {
		return nil, fmt.Errorf("cannot refresh hosted WhoDB login. Run: whodb-cli login --host %s", hostURL)
	}
	if tokens.RefreshToken != "" && tokens.RefreshToken != refreshToken {
		if err := cfg.SavePlatformRefreshToken(hostURL, host.AccountID, tokens.RefreshToken); err != nil {
			return nil, fmt.Errorf("cannot update hosted WhoDB refresh token: %w", err)
		}
	}
	client, err := platformapi.NewClient(hostURL, tokens.AccessToken)
	if err != nil {
		return nil, err
	}
	manifest, err := client.PlatformManifest(ctx)
	if err != nil {
		return nil, fmt.Errorf("cannot load hosted WhoDB platform manifest: %w", err)
	}
	client.SetPlatformManifest(manifest)
	return &platformToolSession{Host: *host, Client: client}, nil
}

func loadPlatformWorkspace(ctx context.Context) (*platformToolSession, error) {
	session, err := loadPlatformToolSession(ctx)
	if err != nil {
		return nil, err
	}
	if !hasPlatformWorkspace(session) {
		return nil, fmt.Errorf("no hosted WhoDB workspace selected. Run: whodb-cli use --org <org> --project <project>")
	}
	return session, nil
}

func loadPlatformSource(ctx context.Context, value string) (*platformToolSession, *platformapi.Source, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil, fmt.Errorf("source is required")
	}
	session, err := loadPlatformWorkspace(ctx)
	if err != nil {
		return nil, nil, err
	}
	sources, err := session.Client.ProjectSources(ctx, session.Host.DefaultOrgID, session.Host.DefaultProjectID)
	if err != nil {
		return nil, nil, err
	}
	source, err := resolvePlatformSource(sources, value)
	if err != nil {
		return nil, nil, err
	}
	return session, source, nil
}

func hasPlatformWorkspace(session *platformToolSession) bool {
	return session != nil &&
		strings.TrimSpace(session.Host.DefaultOrgID) != "" &&
		strings.TrimSpace(session.Host.DefaultProjectID) != ""
}

func resolvePlatformSource(sources []platformapi.Source, value string) (*platformapi.Source, error) {
	needle := strings.TrimSpace(value)
	for i := range sources {
		source := &sources[i]
		if source.ID == needle || strings.EqualFold(source.Name, needle) {
			return source, nil
		}
	}
	return nil, fmt.Errorf("hosted source %q not found", needle)
}

func parsePlatformOptionalRef(value string) (*platformapi.SourceObjectRefInput, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	ref, err := parsePlatformRequiredRef(value)
	if err != nil {
		return nil, err
	}
	return &ref, nil
}

func parsePlatformRequiredRef(value string) (platformapi.SourceObjectRefInput, error) {
	kindValue, pathValue, ok := strings.Cut(strings.TrimSpace(value), ":")
	if !ok || strings.TrimSpace(kindValue) == "" || strings.TrimSpace(pathValue) == "" {
		return platformapi.SourceObjectRefInput{}, fmt.Errorf("object ref must use kind:path, for example table:public.users")
	}
	parts := strings.Split(pathValue, ".")
	path := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			path = append(path, part)
		}
	}
	if len(path) == 0 {
		return platformapi.SourceObjectRefInput{}, fmt.Errorf("object ref path is required")
	}
	return platformapi.SourceObjectRefInput{
		Kind: platformapi.SourceObjectKind(strings.TrimSpace(kindValue)),
		Path: path,
	}, nil
}

func platformRowsLimit(requested int, secOpts *SecurityOptions) int {
	limit := requested
	if limit <= 0 {
		limit = defaultPlatformRowLimit
	}
	if secOpts != nil && secOpts.MaxRows > 0 && limit > secOpts.MaxRows {
		return secOpts.MaxRows
	}
	return limit
}

const descPlatformStatus = `Report the current hosted WhoDB login and selected hosted workspace.

Requires a prior hosted login with whodb-cli login. Use this before other whodb_platform_* tools to confirm which host, organization, and project are active.`

const descPlatformSources = `List hosted WhoDB sources in the selected organization and project.

Requires whodb-cli login and whodb-cli use --org <org> --project <project>. This tool is read-only and never exposes source credentials.`

const descPlatformSourceObjects = `Browse objects in one hosted WhoDB source.

Use the source name or id from whodb_platform_sources. Parent refs use kind:path, for example schema:public. This tool is read-only.`

const descPlatformSourceColumns = `Describe columns for one hosted WhoDB source object.

Use a source name or id and an object ref such as table:public.users. This tool is read-only.`

const descPlatformSourceRows = `Preview rows for one hosted WhoDB source object.

Use a source name or id and an object ref such as table:public.users. Results are capped by the requested limit and the MCP --max-rows setting when provided. This tool is read-only.`
