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
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/clidey/whodb/cli/internal/config"
	platformapi "github.com/clidey/whodb/cli/internal/platform"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const defaultPlatformRowLimit = 50

var (
	pendingPlatformActions = map[string]*PendingPlatformAction{}
	platformPendingMutex   sync.RWMutex
)

type platformClient interface {
	Me(context.Context) (*platformapi.User, error)
	PlatformManifest(context.Context) (*platformapi.PlatformManifest, error)
	Organizations(context.Context) ([]platformapi.Organization, error)
	Projects(context.Context, string) ([]platformapi.Project, error)
	ProjectSources(context.Context, string, string) ([]platformapi.Source, error)
	SourceTypes(context.Context) ([]platformapi.SourceType, error)
	SourceConfig(context.Context, string, string, string) (*platformapi.SourceConfig, error)
	CreateSource(context.Context, platformapi.CreateSourceInput) (*platformapi.Source, error)
	UpdateSource(context.Context, platformapi.UpdateSourceInput) (*platformapi.Source, error)
	TestSourceConnection(context.Context, platformapi.CreateSourceInput) error
	DeleteSource(context.Context, string, string, string) error
	PlatformMutation(context.Context, string, map[string]any) (*platformapi.PlatformMutationResult, error)
	UploadProjectFile(context.Context, string, *string, string) (*platformapi.ProjectFile, error)
	SourceObjects(context.Context, string, string, string, *platformapi.SourceObjectRefInput, []platformapi.SourceObjectKind, int, int) ([]platformapi.SourceObject, error)
	SourceColumns(context.Context, string, string, string, platformapi.SourceObjectRefInput) ([]platformapi.Column, error)
	SourceRows(context.Context, string, string, string, platformapi.SourceObjectRefInput, int, int) (*platformapi.RowsResult, error)
	SourceFieldConstraints(context.Context, string, string, platformapi.SourceObjectRefInput) ([]platformapi.SourceFieldConstraints, error)
	SourceContent(context.Context, string, string, platformapi.SourceObjectRefInput, []string) (*platformapi.SourceContent, error)
	ProjectSecrets(context.Context, string) ([]platformapi.ProjectSecret, error)
	AIProviders(context.Context, string) ([]platformapi.AIProvider, error)
	AIProviderModels(context.Context, string, string) ([]string, error)
	Ontologies(context.Context, string) ([]platformapi.Ontology, error)
	Ontology(context.Context, string, string) (*platformapi.Ontology, error)
	OntologyFastLookups(context.Context, string, string) ([]platformapi.OntologyFastLookup, error)
	OntologyFastLookupSuggestions(context.Context, string, string) ([]platformapi.OntologyFastLookupSuggestion, error)
	OntologyRows(context.Context, string, string, int, int) (*platformapi.DatasetQueryResult, error)
	OntologyFollowLink(context.Context, string, string, string, string, int, int) (*platformapi.DatasetQueryResult, error)
	Datasets(context.Context, string) ([]platformapi.Dataset, error)
	Dataset(context.Context, string, string) (*platformapi.Dataset, error)
	DatasetRows(context.Context, string, string, int, int) (*platformapi.DatasetQueryResult, error)
	Lineage(context.Context, string, string, string, string, int) (*platformapi.LineageGraph, error)
	LineageNeighbors(context.Context, string, string, string) (*platformapi.LineageGraph, error)
	ProjectLineage(context.Context, string) (*platformapi.LineageGraph, error)
	Transforms(context.Context, string) ([]platformapi.Transform, error)
	TransformRuns(context.Context, string, string, int) ([]platformapi.TransformRun, error)
	Functions(context.Context, string, []string) ([]platformapi.Function, error)
	Function(context.Context, string, string, []string) (*platformapi.Function, error)
	FolderContents(context.Context, string, string, []string) (*platformapi.FolderContents, error)
	FilePreview(context.Context, string, string, *int, []string) (*platformapi.FilePreviewResult, error)
	SearchProjectFiles(context.Context, string, string) ([]platformapi.ProjectFile, error)
	ProjectTabularFiles(context.Context, string) ([]platformapi.ProjectFile, error)
	ProjectStorageUsage(context.Context, string) (int, error)
}

type platformToolSession struct {
	Host         config.PlatformHost
	Client       platformClient
	AutoSelected []string
}

var loadPlatformToolSession = loadHostedPlatformToolSession

// PlatformStatusInput is the input for the whodb_platform_status tool.
type PlatformStatusInput struct{}

// PlatformStatusOutput reports hosted WhoDB login and selected workspace state.
type PlatformStatusOutput struct {
	Host                    string   `json:"host,omitempty"`
	UserID                  string   `json:"user_id,omitempty"`
	Email                   string   `json:"email,omitempty"`
	DefaultOrgID            string   `json:"default_org_id,omitempty"`
	DefaultOrgName          string   `json:"default_org_name,omitempty"`
	DefaultProjectID        string   `json:"default_project_id,omitempty"`
	DefaultProjectName      string   `json:"default_project_name,omitempty"`
	WorkspaceSelected       bool     `json:"workspace_selected"`
	PlatformVersion         string   `json:"platform_version,omitempty"`
	ManifestProtocolVersion string   `json:"manifest_protocol_version,omitempty"`
	AutoSelected            []string `json:"auto_selected,omitempty"`
	Error                   string   `json:"error,omitempty"`
	RequestID               string   `json:"request_id,omitempty"`
}

// PlatformSourcesInput is the input for the whodb_platform_sources tool.
type PlatformSourcesInput struct {
	Fields []string `json:"fields,omitempty" jsonschema:"Optional top-level output fields to include in items"`
}

// PlatformOrgsInput is the input for the whodb_platform_orgs tool.
type PlatformOrgsInput struct {
	Fields []string `json:"fields,omitempty" jsonschema:"Optional top-level output fields to include in items"`
}

// PlatformOrgInfo describes an organization visible to the hosted user.
type PlatformOrgInfo struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Slug     string `json:"slug"`
	Selected bool   `json:"selected"`
}

// PlatformOrgsOutput lists organizations visible to the hosted user.
type PlatformOrgsOutput struct {
	Host      string               `json:"host,omitempty"`
	Orgs      []PlatformOrgInfo    `json:"orgs"`
	Items     []map[string]any     `json:"items,omitempty"`
	Count     int                  `json:"count"`
	Scope     *PlatformOutputScope `json:"scope,omitempty"`
	Fields    []string             `json:"fields,omitempty"`
	Warnings  []string             `json:"warnings,omitempty"`
	Error     string               `json:"error,omitempty"`
	RequestID string               `json:"request_id,omitempty"`
}

// MarshalJSON ensures nil slices are serialized as [] instead of null.
func (o PlatformOrgsOutput) MarshalJSON() ([]byte, error) {
	if o.Orgs == nil {
		o.Orgs = []PlatformOrgInfo{}
	}
	type Alias PlatformOrgsOutput
	return json.Marshal(Alias(o))
}

// PlatformProjectsInput is the input for the whodb_platform_projects tool.
type PlatformProjectsInput struct {
	Org    string   `json:"org,omitempty" jsonschema:"Organization id, slug, or name. Defaults to the selected organization when available."`
	Fields []string `json:"fields,omitempty" jsonschema:"Optional top-level output fields to include in items"`
}

// PlatformProjectInfo describes a project visible to the hosted user.
type PlatformProjectInfo struct {
	ID          string `json:"id"`
	OrgID       string `json:"org_id"`
	OrgName     string `json:"org_name,omitempty"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description,omitempty"`
	Selected    bool   `json:"selected"`
}

// PlatformProjectsOutput lists projects visible in one hosted organization.
type PlatformProjectsOutput struct {
	Host      string                `json:"host,omitempty"`
	OrgID     string                `json:"org_id,omitempty"`
	OrgName   string                `json:"org_name,omitempty"`
	Projects  []PlatformProjectInfo `json:"projects"`
	Items     []map[string]any      `json:"items,omitempty"`
	Count     int                   `json:"count"`
	Scope     *PlatformOutputScope  `json:"scope,omitempty"`
	Fields    []string              `json:"fields,omitempty"`
	Warnings  []string              `json:"warnings,omitempty"`
	Error     string                `json:"error,omitempty"`
	RequestID string                `json:"request_id,omitempty"`
}

// MarshalJSON ensures nil slices are serialized as [] instead of null.
func (o PlatformProjectsOutput) MarshalJSON() ([]byte, error) {
	if o.Projects == nil {
		o.Projects = []PlatformProjectInfo{}
	}
	type Alias PlatformProjectsOutput
	return json.Marshal(Alias(o))
}

// PlatformSourceTypesInput is the input for the whodb_platform_source_types tool.
type PlatformSourceTypesInput struct {
	Fields []string `json:"fields,omitempty" jsonschema:"Optional top-level output fields to include in items"`
}

// PlatformSourceTypesOutput lists hosted source types available for creation.
type PlatformSourceTypesOutput struct {
	SourceTypes []platformapi.SourceType `json:"source_types"`
	Items       []map[string]any         `json:"items,omitempty"`
	Count       int                      `json:"count"`
	Scope       *PlatformOutputScope     `json:"scope,omitempty"`
	Fields      []string                 `json:"fields,omitempty"`
	Warnings    []string                 `json:"warnings,omitempty"`
	Error       string                   `json:"error,omitempty"`
	RequestID   string                   `json:"request_id,omitempty"`
}

// MarshalJSON ensures nil slices are serialized as [] instead of null.
func (o PlatformSourceTypesOutput) MarshalJSON() ([]byte, error) {
	if o.SourceTypes == nil {
		o.SourceTypes = []platformapi.SourceType{}
	}
	type Alias PlatformSourceTypesOutput
	return json.Marshal(Alias(o))
}

// PlatformSourceFieldsInput is the input for the whodb_platform_source_fields tool.
type PlatformSourceFieldsInput struct {
	SourceType string   `json:"source_type" jsonschema:"Hosted source type id"`
	Fields     []string `json:"fields,omitempty" jsonschema:"Optional top-level output fields to include in items"`
}

// PlatformSourceFieldsOutput lists connection fields for one hosted source type.
type PlatformSourceFieldsOutput struct {
	SourceType     string                              `json:"source_type,omitempty"`
	Fields         []platformapi.SourceConnectionField `json:"fields"`
	Items          []map[string]any                    `json:"items,omitempty"`
	Count          int                                 `json:"count"`
	Scope          *PlatformOutputScope                `json:"scope,omitempty"`
	SelectedFields []string                            `json:"selected_fields,omitempty"`
	Warnings       []string                            `json:"warnings,omitempty"`
	Error          string                              `json:"error,omitempty"`
	RequestID      string                              `json:"request_id,omitempty"`
}

// MarshalJSON ensures nil slices are serialized as [] instead of null.
func (o PlatformSourceFieldsOutput) MarshalJSON() ([]byte, error) {
	if o.Fields == nil {
		o.Fields = []platformapi.SourceConnectionField{}
	}
	type Alias PlatformSourceFieldsOutput
	return json.Marshal(Alias(o))
}

// PlatformSourcesOutput lists hosted sources in the selected project.
type PlatformSourcesOutput struct {
	Host      string               `json:"host,omitempty"`
	OrgID     string               `json:"org_id,omitempty"`
	ProjectID string               `json:"project_id,omitempty"`
	Sources   []platformapi.Source `json:"sources"`
	Items     []map[string]any     `json:"items,omitempty"`
	Count     int                  `json:"count"`
	Scope     *PlatformOutputScope `json:"scope,omitempty"`
	Fields    []string             `json:"fields,omitempty"`
	Warnings  []string             `json:"warnings,omitempty"`
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
	Fields     []string `json:"fields,omitempty" jsonschema:"Optional top-level output fields to include in items"`
}

// PlatformSourceObjectsOutput lists hosted source objects.
type PlatformSourceObjectsOutput struct {
	Objects   []platformapi.SourceObject `json:"objects"`
	Items     []map[string]any           `json:"items,omitempty"`
	Count     int                        `json:"count"`
	Scope     *PlatformOutputScope       `json:"scope,omitempty"`
	Fields    []string                   `json:"fields,omitempty"`
	Warnings  []string                   `json:"warnings,omitempty"`
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
	Source string   `json:"source" jsonschema:"Hosted source id or name"`
	Ref    string   `json:"ref" jsonschema:"Object ref as kind:path, for example table:public.users"`
	Fields []string `json:"fields,omitempty" jsonschema:"Optional top-level output fields to include in items"`
}

// PlatformSourceColumnsOutput lists columns for one hosted source object.
type PlatformSourceColumnsOutput struct {
	Columns   []platformapi.Column `json:"columns"`
	Items     []map[string]any     `json:"items,omitempty"`
	Count     int                  `json:"count"`
	Scope     *PlatformOutputScope `json:"scope,omitempty"`
	Fields    []string             `json:"fields,omitempty"`
	Warnings  []string             `json:"warnings,omitempty"`
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

// PlatformSourceConfigInput is the input for the whodb_platform_source_config tool.
type PlatformSourceConfigInput struct {
	Source string `json:"source" jsonschema:"Hosted source id or name"`
}

// PlatformSourceConfigOutput returns redacted hosted source connection config.
type PlatformSourceConfigOutput struct {
	Source    *platformapi.Source              `json:"source,omitempty"`
	Config    platformapi.RedactedSourceConfig `json:"config"`
	Error     string                           `json:"error,omitempty"`
	RequestID string                           `json:"request_id,omitempty"`
}

// PlatformSourceTestInput is the input for the whodb_platform_source_test tool.
type PlatformSourceTestInput struct {
	Source     string            `json:"source,omitempty" jsonschema:"Saved hosted source id or name. If omitted, source_type and config fields test a draft source."`
	SourceType string            `json:"source_type,omitempty" jsonschema:"Hosted source type id for draft connection tests"`
	Hostname   string            `json:"hostname,omitempty"`
	Port       string            `json:"port,omitempty"`
	Username   string            `json:"username,omitempty"`
	Password   string            `json:"password,omitempty"`
	Database   string            `json:"database,omitempty"`
	Advanced   map[string]string `json:"advanced,omitempty"`
}

// PlatformSourceTestOutput reports hosted source connection test status.
type PlatformSourceTestOutput struct {
	Status     string              `json:"status,omitempty"`
	Source     *platformapi.Source `json:"source,omitempty"`
	SourceType string              `json:"source_type,omitempty"`
	Error      string              `json:"error,omitempty"`
	RequestID  string              `json:"request_id,omitempty"`
}

// PlatformSourceCreateInput is the input for the whodb_platform_source_create tool.
type PlatformSourceCreateInput struct {
	SourceType string            `json:"source_type" jsonschema:"Hosted source type id"`
	Name       string            `json:"name" jsonschema:"Source display name"`
	Hostname   string            `json:"hostname,omitempty"`
	Port       string            `json:"port,omitempty"`
	Username   string            `json:"username,omitempty"`
	Password   string            `json:"password,omitempty"`
	Database   string            `json:"database,omitempty"`
	Advanced   map[string]string `json:"advanced,omitempty"`
}

// PlatformSourceUpdateInput is the input for the whodb_platform_source_update tool.
type PlatformSourceUpdateInput struct {
	Source   string            `json:"source" jsonschema:"Hosted source id or name"`
	Name     string            `json:"name,omitempty" jsonschema:"New source display name"`
	Hostname string            `json:"hostname,omitempty"`
	Port     string            `json:"port,omitempty"`
	Username string            `json:"username,omitempty"`
	Password string            `json:"password,omitempty"`
	Database string            `json:"database,omitempty"`
	Advanced map[string]string `json:"advanced,omitempty"`
}

// PlatformSourceDeleteInput is the input for the whodb_platform_source_delete tool.
type PlatformSourceDeleteInput struct {
	Source string `json:"source" jsonschema:"Hosted source id or name"`
}

// PlatformSourceWriteOutput reports a hosted platform write prepared for confirmation.
type PlatformSourceWriteOutput struct {
	ConfirmationRequired bool                   `json:"confirmation_required,omitempty"`
	ConfirmationToken    string                 `json:"confirmation_token,omitempty"`
	ConfirmationAction   string                 `json:"confirmation_action,omitempty"`
	ConfirmationPreview  *PlatformActionPreview `json:"confirmation_preview,omitempty"`
	ConfirmationExpiry   string                 `json:"confirmation_expiry,omitempty"`
	Warning              string                 `json:"warning,omitempty"`
	Source               *platformapi.Source    `json:"source,omitempty"`
	Status               string                 `json:"status,omitempty"`
	Error                string                 `json:"error,omitempty"`
	RequestID            string                 `json:"request_id,omitempty"`
}

// PlatformPendingInput is the input for the whodb_platform_pending tool.
type PlatformPendingInput struct{}

// PlatformPendingInfo represents a pending hosted platform confirmation.
type PlatformPendingInfo struct {
	Token     string                `json:"token"`
	Action    PlatformActionPreview `json:"action"`
	ExpiresAt string                `json:"expires_at"`
}

// PlatformPendingOutput lists pending hosted platform confirmations.
type PlatformPendingOutput struct {
	Pending   []PlatformPendingInfo `json:"pending"`
	Error     string                `json:"error,omitempty"`
	RequestID string                `json:"request_id,omitempty"`
}

// MarshalJSON ensures nil slices are serialized as [] instead of null.
func (o PlatformPendingOutput) MarshalJSON() ([]byte, error) {
	if o.Pending == nil {
		o.Pending = []PlatformPendingInfo{}
	}
	type Alias PlatformPendingOutput
	return json.Marshal(Alias(o))
}

// PlatformActionPreview describes a pending hosted source write without secrets.
type PlatformActionPreview struct {
	Operation   string   `json:"operation"`
	Resource    string   `json:"resource,omitempty"`
	Action      string   `json:"action,omitempty"`
	Host        string   `json:"host"`
	OrgID       string   `json:"org_id"`
	ProjectID   string   `json:"project_id"`
	ProjectName string   `json:"project_name,omitempty"`
	SourceID    string   `json:"source_id,omitempty"`
	SourceName  string   `json:"source_name,omitempty"`
	SourceType  string   `json:"source_type,omitempty"`
	Changes     []string `json:"changes,omitempty"`
}

// PendingPlatformAction stores a hosted platform write awaiting confirmation.
type PendingPlatformAction struct {
	Token       string
	Operation   string
	Resource    string
	Action      string
	Host        string
	OrgID       string
	ProjectID   string
	ProjectName string
	SourceID    string
	SourceName  string
	SourceType  string
	Changes     []string
	CreateInput platformapi.CreateSourceInput
	UpdateInput platformapi.UpdateSourceInput
	Mutation    string
	Variables   map[string]any
	ExpiresAt   time.Time
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
		if !secOpts.ReadOnly && registerPlatformGenericWriteTool(server, tool, secOpts) {
			continue
		}
		switch tool.Name {
		case "whodb_platform_status":
			mcp.AddTool(server, tool, createPlatformStatusHandler())
		case "whodb_platform_sources":
			mcp.AddTool(server, tool, createPlatformSourcesHandler())
		case "whodb_platform_orgs":
			mcp.AddTool(server, tool, createPlatformOrgsHandler())
		case "whodb_platform_projects":
			mcp.AddTool(server, tool, createPlatformProjectsHandler())
		case "whodb_platform_source_types":
			mcp.AddTool(server, tool, createPlatformSourceTypesHandler())
		case "whodb_platform_source_fields":
			mcp.AddTool(server, tool, createPlatformSourceFieldsHandler())
		case "whodb_platform_source_objects":
			mcp.AddTool(server, tool, createPlatformSourceObjectsHandler())
		case "whodb_platform_source_columns":
			mcp.AddTool(server, tool, createPlatformSourceColumnsHandler())
		case "whodb_platform_source_rows":
			mcp.AddTool(server, tool, createPlatformSourceRowsHandler(secOpts))
		case "whodb_platform_source_config":
			mcp.AddTool(server, tool, createPlatformSourceConfigHandler())
		case "whodb_platform_source_test":
			mcp.AddTool(server, tool, createPlatformSourceTestHandler())
		case "whodb_platform_source_create":
			if !secOpts.ReadOnly {
				mcp.AddTool(server, tool, createPlatformSourceCreateHandler(secOpts))
			}
		case "whodb_platform_source_update":
			if !secOpts.ReadOnly {
				mcp.AddTool(server, tool, createPlatformSourceUpdateHandler(secOpts))
			}
		case "whodb_platform_source_delete":
			if !secOpts.ReadOnly {
				mcp.AddTool(server, tool, createPlatformSourceDeleteHandler(secOpts))
			}
		case "whodb_platform_pending":
			if secOpts.ConfirmWrites {
				mcp.AddTool(server, tool, createPlatformPendingHandler())
			}
		case "whodb_platform_confirm":
			if secOpts.ConfirmWrites {
				mcp.AddTool(server, tool, createPlatformConfirmHandler())
			}
		default:
			registerPlatformReadTool(server, tool, secOpts)
		}
	}
}

func platformToolDefinitions() []*mcp.Tool {
	tools := []*mcp.Tool{
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
			Name:        "whodb_platform_orgs",
			Description: descPlatformOrgs,
			Annotations: platformReadOnlyAnnotations("List Hosted Organizations"),
		},
		{
			Name:        "whodb_platform_projects",
			Description: descPlatformProjects,
			Annotations: platformReadOnlyAnnotations("List Hosted Projects"),
		},
		{
			Name:        "whodb_platform_source_types",
			Description: descPlatformSourceTypes,
			Annotations: platformReadOnlyAnnotations("List Hosted Source Types"),
		},
		{
			Name:        "whodb_platform_source_fields",
			Description: descPlatformSourceFields,
			Annotations: platformReadOnlyAnnotations("List Hosted Source Fields"),
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
		{
			Name:        "whodb_platform_source_config",
			Description: descPlatformSourceConfig,
			Annotations: platformReadOnlyAnnotations("Inspect Hosted Source Config"),
		},
		{
			Name:        "whodb_platform_source_test",
			Description: descPlatformSourceTest,
			Annotations: platformReadOnlyAnnotations("Test Hosted Source Connection"),
		},
		{
			Name:        "whodb_platform_source_create",
			Description: descPlatformSourceCreate,
			Annotations: platformDestructiveAnnotations("Create Hosted Source"),
		},
		{
			Name:        "whodb_platform_source_update",
			Description: descPlatformSourceUpdate,
			Annotations: platformDestructiveAnnotations("Update Hosted Source"),
		},
		{
			Name:        "whodb_platform_source_delete",
			Description: descPlatformSourceDelete,
			Annotations: platformDestructiveAnnotations("Delete Hosted Source"),
		},
		{
			Name:        "whodb_platform_pending",
			Description: descPlatformPending,
			Annotations: platformReadOnlyAnnotations("List Pending Hosted Platform Writes"),
		},
		{
			Name:        "whodb_platform_confirm",
			Description: descPlatformConfirm,
			Annotations: platformDestructiveAnnotations("Confirm Hosted Platform Write"),
		},
	}
	tools = append(tools, platformGenericWriteToolDefinitions()...)
	return append(tools, platformReadToolDefinitions()...)
}

func platformReadOnlyAnnotations(title string) *mcp.ToolAnnotations {
	return &mcp.ToolAnnotations{
		Title:          title,
		ReadOnlyHint:   true,
		IdempotentHint: true,
		OpenWorldHint:  boolPtr(true),
	}
}

func platformDestructiveAnnotations(title string) *mcp.ToolAnnotations {
	return &mcp.ToolAnnotations{
		Title:           title,
		ReadOnlyHint:    false,
		DestructiveHint: boolPtr(true),
		IdempotentHint:  false,
		OpenWorldHint:   boolPtr(true),
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

func createPlatformOrgsHandler() func(context.Context, *mcp.CallToolRequest, PlatformOrgsInput) (*mcp.CallToolResult, PlatformOrgsOutput, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input PlatformOrgsInput) (*mcp.CallToolResult, PlatformOrgsOutput, error) {
		return HandlePlatformOrgs(ctx, req, input)
	}
}

func createPlatformProjectsHandler() func(context.Context, *mcp.CallToolRequest, PlatformProjectsInput) (*mcp.CallToolResult, PlatformProjectsOutput, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input PlatformProjectsInput) (*mcp.CallToolResult, PlatformProjectsOutput, error) {
		return HandlePlatformProjects(ctx, req, input)
	}
}

func createPlatformSourceTypesHandler() func(context.Context, *mcp.CallToolRequest, PlatformSourceTypesInput) (*mcp.CallToolResult, PlatformSourceTypesOutput, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input PlatformSourceTypesInput) (*mcp.CallToolResult, PlatformSourceTypesOutput, error) {
		return HandlePlatformSourceTypes(ctx, req, input)
	}
}

func createPlatformSourceFieldsHandler() func(context.Context, *mcp.CallToolRequest, PlatformSourceFieldsInput) (*mcp.CallToolResult, PlatformSourceFieldsOutput, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input PlatformSourceFieldsInput) (*mcp.CallToolResult, PlatformSourceFieldsOutput, error) {
		return HandlePlatformSourceFields(ctx, req, input)
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

func createPlatformSourceConfigHandler() func(context.Context, *mcp.CallToolRequest, PlatformSourceConfigInput) (*mcp.CallToolResult, PlatformSourceConfigOutput, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input PlatformSourceConfigInput) (*mcp.CallToolResult, PlatformSourceConfigOutput, error) {
		return HandlePlatformSourceConfig(ctx, req, input)
	}
}

func createPlatformSourceTestHandler() func(context.Context, *mcp.CallToolRequest, PlatformSourceTestInput) (*mcp.CallToolResult, PlatformSourceTestOutput, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input PlatformSourceTestInput) (*mcp.CallToolResult, PlatformSourceTestOutput, error) {
		return HandlePlatformSourceTest(ctx, req, input)
	}
}

func createPlatformSourceCreateHandler(secOpts *SecurityOptions) func(context.Context, *mcp.CallToolRequest, PlatformSourceCreateInput) (*mcp.CallToolResult, PlatformSourceWriteOutput, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input PlatformSourceCreateInput) (*mcp.CallToolResult, PlatformSourceWriteOutput, error) {
		return handlePlatformSourceCreate(ctx, req, input, secOpts.ConfirmWrites)
	}
}

func createPlatformSourceUpdateHandler(secOpts *SecurityOptions) func(context.Context, *mcp.CallToolRequest, PlatformSourceUpdateInput) (*mcp.CallToolResult, PlatformSourceWriteOutput, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input PlatformSourceUpdateInput) (*mcp.CallToolResult, PlatformSourceWriteOutput, error) {
		return handlePlatformSourceUpdate(ctx, req, input, secOpts.ConfirmWrites)
	}
}

func createPlatformSourceDeleteHandler(secOpts *SecurityOptions) func(context.Context, *mcp.CallToolRequest, PlatformSourceDeleteInput) (*mcp.CallToolResult, PlatformSourceWriteOutput, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input PlatformSourceDeleteInput) (*mcp.CallToolResult, PlatformSourceWriteOutput, error) {
		return handlePlatformSourceDelete(ctx, req, input, secOpts.ConfirmWrites)
	}
}

func createPlatformPendingHandler() func(context.Context, *mcp.CallToolRequest, PlatformPendingInput) (*mcp.CallToolResult, PlatformPendingOutput, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input PlatformPendingInput) (*mcp.CallToolResult, PlatformPendingOutput, error) {
		return HandlePlatformPending(ctx, req, input)
	}
}

func createPlatformConfirmHandler() func(context.Context, *mcp.CallToolRequest, ConfirmInput) (*mcp.CallToolResult, ConfirmOutput, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input ConfirmInput) (*mcp.CallToolResult, ConfirmOutput, error) {
		return HandlePlatformConfirm(ctx, req, input)
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
		AutoSelected:            session.AutoSelected,
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

	readOutput := platformReadOutput(session, "platform_sources", sources, len(sources), false, requestID, input.Fields)
	TrackToolCall(ctx, "platform_sources", requestID, true, time.Since(startTime).Milliseconds(), map[string]any{"source_count": len(sources)})
	return nil, PlatformSourcesOutput{
		Host:      session.Host.URL,
		OrgID:     session.Host.DefaultOrgID,
		ProjectID: session.Host.DefaultProjectID,
		Sources:   sources,
		Items:     readOutput.Items,
		Count:     readOutput.Count,
		Scope:     readOutput.Scope,
		Fields:    readOutput.Fields,
		Warnings:  readOutput.Warnings,
		RequestID: requestID,
	}, nil
}

// HandlePlatformOrgs lists hosted organizations visible to the authenticated user.
func HandlePlatformOrgs(ctx context.Context, req *mcp.CallToolRequest, input PlatformOrgsInput) (*mcp.CallToolResult, PlatformOrgsOutput, error) {
	requestID := generateRequestID("platform_orgs")
	startTime := time.Now()

	session, err := loadPlatformToolSession(ctx)
	if err != nil {
		TrackToolCall(ctx, "platform_orgs", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "platform_session"})
		return nil, PlatformOrgsOutput{Error: err.Error(), RequestID: requestID}, nil
	}
	orgs, err := session.Client.Organizations(ctx)
	if err != nil {
		TrackToolCall(ctx, "platform_orgs", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "platform_query"})
		return nil, PlatformOrgsOutput{Error: err.Error(), RequestID: requestID}, nil
	}

	output := PlatformOrgsOutput{
		Host:      session.Host.URL,
		Orgs:      platformOrgInfos(orgs, session.Host.DefaultOrgID),
		RequestID: requestID,
	}
	readOutput := platformReadOutput(session, "platform_orgs", output.Orgs, len(output.Orgs), false, requestID, input.Fields)
	output.Items = readOutput.Items
	output.Count = readOutput.Count
	output.Scope = readOutput.Scope
	output.Fields = readOutput.Fields
	output.Warnings = readOutput.Warnings
	TrackToolCall(ctx, "platform_orgs", requestID, true, time.Since(startTime).Milliseconds(), map[string]any{"org_count": len(output.Orgs)})
	return nil, output, nil
}

// HandlePlatformProjects lists hosted projects in one organization.
func HandlePlatformProjects(ctx context.Context, req *mcp.CallToolRequest, input PlatformProjectsInput) (*mcp.CallToolResult, PlatformProjectsOutput, error) {
	requestID := generateRequestID("platform_projects")
	startTime := time.Now()

	session, err := loadPlatformToolSession(ctx)
	if err != nil {
		TrackToolCall(ctx, "platform_projects", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "platform_session"})
		return nil, PlatformProjectsOutput{Error: err.Error(), RequestID: requestID}, nil
	}
	org, err := resolvePlatformToolOrg(ctx, session, input.Org)
	if err != nil {
		TrackToolCall(ctx, "platform_projects", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "platform_org"})
		return nil, PlatformProjectsOutput{Error: err.Error(), RequestID: requestID}, nil
	}
	projects, err := session.Client.Projects(ctx, org.ID)
	if err != nil {
		TrackToolCall(ctx, "platform_projects", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "platform_query"})
		return nil, PlatformProjectsOutput{Error: err.Error(), RequestID: requestID}, nil
	}

	output := PlatformProjectsOutput{
		Host:      session.Host.URL,
		OrgID:     org.ID,
		OrgName:   org.Name,
		Projects:  platformProjectInfos(projects, org.Name, session.Host.DefaultProjectID),
		RequestID: requestID,
	}
	readOutput := platformReadOutput(session, "platform_projects", output.Projects, len(output.Projects), false, requestID, input.Fields)
	output.Items = readOutput.Items
	output.Count = readOutput.Count
	output.Scope = readOutput.Scope
	output.Fields = readOutput.Fields
	output.Warnings = readOutput.Warnings
	TrackToolCall(ctx, "platform_projects", requestID, true, time.Since(startTime).Milliseconds(), map[string]any{"project_count": len(output.Projects)})
	return nil, output, nil
}

// HandlePlatformSourceTypes lists hosted source types available for creation.
func HandlePlatformSourceTypes(ctx context.Context, req *mcp.CallToolRequest, input PlatformSourceTypesInput) (*mcp.CallToolResult, PlatformSourceTypesOutput, error) {
	requestID := generateRequestID("platform_source_types")
	startTime := time.Now()

	session, err := loadPlatformWorkspace(ctx)
	if err != nil {
		TrackToolCall(ctx, "platform_source_types", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "platform_session"})
		return nil, PlatformSourceTypesOutput{Error: err.Error(), RequestID: requestID}, nil
	}
	types, err := session.Client.SourceTypes(ctx)
	if err != nil {
		TrackToolCall(ctx, "platform_source_types", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "platform_query"})
		return nil, PlatformSourceTypesOutput{Error: err.Error(), RequestID: requestID}, nil
	}

	readOutput := platformReadOutput(session, "platform_source_types", types, len(types), false, requestID, input.Fields)
	TrackToolCall(ctx, "platform_source_types", requestID, true, time.Since(startTime).Milliseconds(), map[string]any{"source_type_count": len(types)})
	return nil, PlatformSourceTypesOutput{
		SourceTypes: types,
		Items:       readOutput.Items,
		Count:       readOutput.Count,
		Scope:       readOutput.Scope,
		Fields:      readOutput.Fields,
		Warnings:    readOutput.Warnings,
		RequestID:   requestID,
	}, nil
}

// HandlePlatformSourceFields lists connection fields for one hosted source type.
func HandlePlatformSourceFields(ctx context.Context, req *mcp.CallToolRequest, input PlatformSourceFieldsInput) (*mcp.CallToolResult, PlatformSourceFieldsOutput, error) {
	requestID := generateRequestID("platform_source_fields")
	startTime := time.Now()

	session, err := loadPlatformWorkspace(ctx)
	if err != nil {
		TrackToolCall(ctx, "platform_source_fields", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "platform_session"})
		return nil, PlatformSourceFieldsOutput{Error: err.Error(), RequestID: requestID}, nil
	}
	sourceType, err := loadPlatformSourceType(ctx, session, input.SourceType)
	if err != nil {
		TrackToolCall(ctx, "platform_source_fields", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "source_type"})
		return nil, PlatformSourceFieldsOutput{Error: err.Error(), RequestID: requestID}, nil
	}

	readOutput := platformReadOutput(session, "platform_source_fields", sourceType.ConnectionFields, len(sourceType.ConnectionFields), false, requestID, input.Fields)
	TrackToolCall(ctx, "platform_source_fields", requestID, true, time.Since(startTime).Milliseconds(), map[string]any{"field_count": len(sourceType.ConnectionFields)})
	return nil, PlatformSourceFieldsOutput{
		SourceType:     sourceType.ID,
		Fields:         sourceType.ConnectionFields,
		Items:          readOutput.Items,
		Count:          readOutput.Count,
		Scope:          readOutput.Scope,
		SelectedFields: readOutput.Fields,
		Warnings:       readOutput.Warnings,
		RequestID:      requestID,
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

	readOutput := platformReadOutput(session, "platform_source_objects", objects, len(objects), false, requestID, input.Fields)
	TrackToolCall(ctx, "platform_source_objects", requestID, true, time.Since(startTime).Milliseconds(), map[string]any{"object_count": len(objects)})
	return nil, PlatformSourceObjectsOutput{
		Objects:   objects,
		Items:     readOutput.Items,
		Count:     readOutput.Count,
		Scope:     readOutput.Scope,
		Fields:    readOutput.Fields,
		Warnings:  readOutput.Warnings,
		RequestID: requestID,
	}, nil
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

	readOutput := platformReadOutput(session, "platform_source_columns", columns, len(columns), false, requestID, input.Fields)
	TrackToolCall(ctx, "platform_source_columns", requestID, true, time.Since(startTime).Milliseconds(), map[string]any{"column_count": len(columns)})
	return nil, PlatformSourceColumnsOutput{
		Columns:   columns,
		Items:     readOutput.Items,
		Count:     readOutput.Count,
		Scope:     readOutput.Scope,
		Fields:    readOutput.Fields,
		Warnings:  readOutput.Warnings,
		RequestID: requestID,
	}, nil
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

// HandlePlatformSourceConfig returns redacted config for one hosted source.
func HandlePlatformSourceConfig(ctx context.Context, req *mcp.CallToolRequest, input PlatformSourceConfigInput) (*mcp.CallToolResult, PlatformSourceConfigOutput, error) {
	requestID := generateRequestID("platform_source_config")
	startTime := time.Now()

	session, source, err := loadPlatformSource(ctx, input.Source)
	if err != nil {
		TrackToolCall(ctx, "platform_source_config", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "platform_source"})
		return nil, PlatformSourceConfigOutput{Error: err.Error(), RequestID: requestID}, nil
	}
	sourceType, err := loadPlatformSourceType(ctx, session, source.DatabaseType)
	if err != nil {
		TrackToolCall(ctx, "platform_source_config", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "source_type"})
		return nil, PlatformSourceConfigOutput{Error: err.Error(), RequestID: requestID}, nil
	}
	config, err := session.Client.SourceConfig(ctx, session.Host.DefaultOrgID, session.Host.DefaultProjectID, source.ID)
	if err != nil {
		TrackToolCall(ctx, "platform_source_config", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "platform_query"})
		return nil, PlatformSourceConfigOutput{Error: err.Error(), RequestID: requestID}, nil
	}

	TrackToolCall(ctx, "platform_source_config", requestID, true, time.Since(startTime).Milliseconds(), nil)
	return nil, PlatformSourceConfigOutput{
		Source:    source,
		Config:    platformapi.RedactSourceConfig(config, sourceType),
		RequestID: requestID,
	}, nil
}

// HandlePlatformSourceTest checks a saved or draft hosted source connection.
func HandlePlatformSourceTest(ctx context.Context, req *mcp.CallToolRequest, input PlatformSourceTestInput) (*mcp.CallToolResult, PlatformSourceTestOutput, error) {
	requestID := generateRequestID("platform_source_test")
	startTime := time.Now()

	if strings.TrimSpace(input.Source) != "" {
		session, source, err := loadPlatformSource(ctx, input.Source)
		if err != nil {
			TrackToolCall(ctx, "platform_source_test", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "platform_source"})
			return nil, PlatformSourceTestOutput{Error: err.Error(), RequestID: requestID}, nil
		}
		if _, err := session.Client.SourceObjects(ctx, session.Host.DefaultOrgID, session.Host.DefaultProjectID, source.ID, nil, nil, 1, 0); err != nil {
			TrackToolCall(ctx, "platform_source_test", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "platform_query"})
			return nil, PlatformSourceTestOutput{Error: fmt.Sprintf("saved source connection failed: %v", err), RequestID: requestID}, nil
		}
		TrackToolCall(ctx, "platform_source_test", requestID, true, time.Since(startTime).Milliseconds(), map[string]any{"mode": "saved"})
		return nil, PlatformSourceTestOutput{Status: "ok", Source: source, RequestID: requestID}, nil
	}

	session, err := loadPlatformWorkspace(ctx)
	if err != nil {
		TrackToolCall(ctx, "platform_source_test", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "platform_session"})
		return nil, PlatformSourceTestOutput{Error: err.Error(), RequestID: requestID}, nil
	}
	sourceType, err := loadPlatformSourceType(ctx, session, input.SourceType)
	if err != nil {
		TrackToolCall(ctx, "platform_source_test", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "source_type"})
		return nil, PlatformSourceTestOutput{Error: err.Error(), RequestID: requestID}, nil
	}
	draft := platformSourceCreateInput(session, sourceType.ID, "connection-test", platformSourceConfigValues(input.Hostname, input.Port, input.Username, input.Password, input.Database), input.Advanced)
	applyPlatformSourceDefaults(sourceType, &draft)
	if err := validatePlatformSourceRequiredFields(sourceType, draft); err != nil {
		TrackToolCall(ctx, "platform_source_test", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "validation"})
		return nil, PlatformSourceTestOutput{Error: err.Error(), RequestID: requestID}, nil
	}
	if err := session.Client.TestSourceConnection(ctx, draft); err != nil {
		TrackToolCall(ctx, "platform_source_test", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "platform_query"})
		return nil, PlatformSourceTestOutput{Error: fmt.Sprintf("draft source configuration failed: %v", err), RequestID: requestID}, nil
	}
	TrackToolCall(ctx, "platform_source_test", requestID, true, time.Since(startTime).Milliseconds(), map[string]any{"mode": "draft"})
	return nil, PlatformSourceTestOutput{Status: "ok", SourceType: sourceType.ID, RequestID: requestID}, nil
}

// HandlePlatformSourceCreate prepares a hosted source creation for confirmation.
func HandlePlatformSourceCreate(ctx context.Context, req *mcp.CallToolRequest, input PlatformSourceCreateInput) (*mcp.CallToolResult, PlatformSourceWriteOutput, error) {
	return handlePlatformSourceCreate(ctx, req, input, true)
}

func handlePlatformSourceCreate(ctx context.Context, req *mcp.CallToolRequest, input PlatformSourceCreateInput, confirmWrites bool) (*mcp.CallToolResult, PlatformSourceWriteOutput, error) {
	requestID := generateRequestID("platform_source_create")
	startTime := time.Now()

	session, err := loadPlatformWorkspace(ctx)
	if err != nil {
		TrackToolCall(ctx, "platform_source_create", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "platform_session"})
		return nil, PlatformSourceWriteOutput{Error: err.Error(), RequestID: requestID}, nil
	}
	sourceType, err := loadPlatformSourceType(ctx, session, input.SourceType)
	if err != nil {
		TrackToolCall(ctx, "platform_source_create", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "source_type"})
		return nil, PlatformSourceWriteOutput{Error: err.Error(), RequestID: requestID}, nil
	}
	createInput := platformSourceCreateInput(session, sourceType.ID, input.Name, platformSourceConfigValues(input.Hostname, input.Port, input.Username, input.Password, input.Database), input.Advanced)
	applyPlatformSourceDefaults(sourceType, &createInput)
	if strings.TrimSpace(createInput.Name) == "" {
		return nil, PlatformSourceWriteOutput{Error: "name is required", RequestID: requestID}, nil
	}
	if err := validatePlatformSourceRequiredFields(sourceType, createInput); err != nil {
		TrackToolCall(ctx, "platform_source_create", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "validation"})
		return nil, PlatformSourceWriteOutput{Error: err.Error(), RequestID: requestID}, nil
	}

	actionLabel := fmt.Sprintf("create hosted source %q (%s) in %s", createInput.Name, sourceType.ID, session.Host.DefaultProjectName)
	action := &PendingPlatformAction{
		Operation:   "create_source",
		Host:        session.Host.URL,
		OrgID:       session.Host.DefaultOrgID,
		ProjectID:   session.Host.DefaultProjectID,
		ProjectName: session.Host.DefaultProjectName,
		SourceName:  createInput.Name,
		SourceType:  sourceType.ID,
		Changes:     []string{"create source"},
		CreateInput: createInput,
	}
	if !confirmWrites {
		source, err := session.Client.CreateSource(ctx, createInput)
		if err != nil {
			TrackToolCall(ctx, "platform_source_create", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "platform_action"})
			return nil, PlatformSourceWriteOutput{Error: err.Error(), RequestID: requestID}, nil
		}
		TrackToolCall(ctx, "platform_source_create", requestID, true, time.Since(startTime).Milliseconds(), map[string]any{"confirmation_required": false})
		return nil, platformSourceWriteCompletedOutput(requestID, "create_source", source, action.Preview()), nil
	}
	token, expiresAt := storePendingPlatformAction(action)
	TrackToolCall(ctx, "platform_source_create", requestID, true, time.Since(startTime).Milliseconds(), map[string]any{"confirmation_required": true})
	return nil, platformSourceConfirmationOutput(requestID, token, expiresAt, actionLabel, action.Preview()), nil
}

// HandlePlatformSourceUpdate prepares a hosted source update for confirmation.
func HandlePlatformSourceUpdate(ctx context.Context, req *mcp.CallToolRequest, input PlatformSourceUpdateInput) (*mcp.CallToolResult, PlatformSourceWriteOutput, error) {
	return handlePlatformSourceUpdate(ctx, req, input, true)
}

func handlePlatformSourceUpdate(ctx context.Context, req *mcp.CallToolRequest, input PlatformSourceUpdateInput, confirmWrites bool) (*mcp.CallToolResult, PlatformSourceWriteOutput, error) {
	requestID := generateRequestID("platform_source_update")
	startTime := time.Now()

	session, source, err := loadPlatformSource(ctx, input.Source)
	if err != nil {
		TrackToolCall(ctx, "platform_source_update", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "platform_source"})
		return nil, PlatformSourceWriteOutput{Error: err.Error(), RequestID: requestID}, nil
	}

	updateInput := platformapi.UpdateSourceInput{OrgID: session.Host.DefaultOrgID, ProjectID: session.Host.DefaultProjectID, ID: source.ID}
	if strings.TrimSpace(input.Name) != "" {
		name := strings.TrimSpace(input.Name)
		updateInput.Name = &name
	}
	values := platformSourceConfigValues(input.Hostname, input.Port, input.Username, input.Password, input.Database)
	changes := []string{}
	if len(values) > 0 || len(input.Advanced) > 0 {
		sourceType, err := loadPlatformSourceType(ctx, session, source.DatabaseType)
		if err != nil {
			TrackToolCall(ctx, "platform_source_update", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "source_type"})
			return nil, PlatformSourceWriteOutput{Error: err.Error(), RequestID: requestID}, nil
		}
		existing, err := session.Client.SourceConfig(ctx, session.Host.DefaultOrgID, session.Host.DefaultProjectID, source.ID)
		if err != nil {
			TrackToolCall(ctx, "platform_source_update", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "platform_query"})
			return nil, PlatformSourceWriteOutput{Error: err.Error(), RequestID: requestID}, nil
		}
		advancedValues, remainingAdvanced := splitPlatformSourceAdvanced(sourceType, input.Advanced)
		configValues := canonicalPlatformSourceValues(sourceType, values)
		for key, value := range advancedValues {
			configValues[key] = value
		}
		config := platformapi.MergeSourceConfig(existing, configValues, remainingAdvanced)
		updateInput.Config = &config
		changes = append(changes, platformSourceConfigChangeNames(values, input.Advanced)...)
	}
	if updateInput.Name == nil && updateInput.Config == nil {
		return nil, PlatformSourceWriteOutput{Error: "nothing to update; pass name or a connection config field", RequestID: requestID}, nil
	}
	if updateInput.Name != nil {
		changes = append(changes, "name")
	}

	actionLabel := fmt.Sprintf("update hosted source %q in %s", source.Name, session.Host.DefaultProjectName)
	action := &PendingPlatformAction{
		Operation:   "update_source",
		Host:        session.Host.URL,
		OrgID:       session.Host.DefaultOrgID,
		ProjectID:   session.Host.DefaultProjectID,
		ProjectName: session.Host.DefaultProjectName,
		SourceID:    source.ID,
		SourceName:  source.Name,
		SourceType:  source.DatabaseType,
		Changes:     changes,
		UpdateInput: updateInput,
	}
	if !confirmWrites {
		updated, err := session.Client.UpdateSource(ctx, updateInput)
		if err != nil {
			TrackToolCall(ctx, "platform_source_update", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "platform_action"})
			return nil, PlatformSourceWriteOutput{Error: err.Error(), RequestID: requestID}, nil
		}
		TrackToolCall(ctx, "platform_source_update", requestID, true, time.Since(startTime).Milliseconds(), map[string]any{"confirmation_required": false})
		return nil, platformSourceWriteCompletedOutput(requestID, "update_source", updated, action.Preview()), nil
	}
	token, expiresAt := storePendingPlatformAction(action)
	TrackToolCall(ctx, "platform_source_update", requestID, true, time.Since(startTime).Milliseconds(), map[string]any{"confirmation_required": true})
	return nil, platformSourceConfirmationOutput(requestID, token, expiresAt, actionLabel, action.Preview()), nil
}

// HandlePlatformSourceDelete prepares a hosted source deletion for confirmation.
func HandlePlatformSourceDelete(ctx context.Context, req *mcp.CallToolRequest, input PlatformSourceDeleteInput) (*mcp.CallToolResult, PlatformSourceWriteOutput, error) {
	return handlePlatformSourceDelete(ctx, req, input, true)
}

func handlePlatformSourceDelete(ctx context.Context, req *mcp.CallToolRequest, input PlatformSourceDeleteInput, confirmWrites bool) (*mcp.CallToolResult, PlatformSourceWriteOutput, error) {
	requestID := generateRequestID("platform_source_delete")
	startTime := time.Now()

	session, source, err := loadPlatformSource(ctx, input.Source)
	if err != nil {
		TrackToolCall(ctx, "platform_source_delete", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "platform_source"})
		return nil, PlatformSourceWriteOutput{Error: err.Error(), RequestID: requestID}, nil
	}

	actionLabel := fmt.Sprintf("delete hosted source %q from %s", source.Name, session.Host.DefaultProjectName)
	action := &PendingPlatformAction{
		Operation:   "delete_source",
		Host:        session.Host.URL,
		OrgID:       session.Host.DefaultOrgID,
		ProjectID:   session.Host.DefaultProjectID,
		ProjectName: session.Host.DefaultProjectName,
		SourceID:    source.ID,
		SourceName:  source.Name,
		SourceType:  source.DatabaseType,
		Changes:     []string{"delete source"},
	}
	if !confirmWrites {
		if err := session.Client.DeleteSource(ctx, session.Host.DefaultOrgID, session.Host.DefaultProjectID, source.ID); err != nil {
			TrackToolCall(ctx, "platform_source_delete", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "platform_action"})
			return nil, PlatformSourceWriteOutput{Error: err.Error(), RequestID: requestID}, nil
		}
		TrackToolCall(ctx, "platform_source_delete", requestID, true, time.Since(startTime).Milliseconds(), map[string]any{"confirmation_required": false})
		return nil, PlatformSourceWriteOutput{
			Status:              "ok",
			ConfirmationAction:  "delete_source",
			ConfirmationPreview: action.Preview(),
			RequestID:           requestID,
		}, nil
	}
	token, expiresAt := storePendingPlatformAction(action)
	TrackToolCall(ctx, "platform_source_delete", requestID, true, time.Since(startTime).Milliseconds(), map[string]any{"confirmation_required": true})
	return nil, platformSourceConfirmationOutput(requestID, token, expiresAt, actionLabel, action.Preview()), nil
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
	autoSelected, changed, err := autoSelectPlatformToolWorkspace(ctx, client, host)
	if err != nil {
		return nil, err
	}
	if changed {
		cfg.UpsertPlatformHost(*host)
		cfg.SetDefaultPlatformHost(host.URL)
		if err := cfg.Save(); err != nil {
			return nil, err
		}
	}
	client.SetWorkspaceContext(host.DefaultOrgID, host.DefaultProjectID)
	return &platformToolSession{Host: *host, Client: client, AutoSelected: autoSelected}, nil
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

func loadPlatformSourceType(ctx context.Context, session *platformToolSession, value string) (*platformapi.SourceType, error) {
	needle := strings.TrimSpace(value)
	if needle == "" {
		return nil, fmt.Errorf("source_type is required")
	}
	types, err := session.Client.SourceTypes(ctx)
	if err != nil {
		return nil, err
	}
	for i := range types {
		sourceType := &types[i]
		if matchesPlatformSourceIdentifier(needle, sourceType.ID, sourceType.Connector, sourceType.Label) {
			return sourceType, nil
		}
	}
	return nil, fmt.Errorf("source type %q not found", needle)
}

func autoSelectPlatformToolWorkspace(ctx context.Context, client platformClient, host *config.PlatformHost) ([]string, bool, error) {
	if host == nil {
		return nil, false, nil
	}
	messages := []string{}
	changed := false
	orgs, err := client.Organizations(ctx)
	if err != nil {
		return nil, false, err
	}
	if strings.TrimSpace(host.DefaultOrgID) == "" && len(orgs) == 1 {
		org := orgs[0]
		host.DefaultOrgID = org.ID
		host.DefaultOrgName = org.Name
		messages = append(messages, fmt.Sprintf("Selected the only available organization: %s", org.Name))
		changed = true
	} else if strings.TrimSpace(host.DefaultOrgID) != "" {
		if org := findPlatformToolOrgByID(orgs, host.DefaultOrgID); org != nil && host.DefaultOrgName != org.Name {
			host.DefaultOrgName = org.Name
			changed = true
		}
	}
	if strings.TrimSpace(host.DefaultOrgID) == "" {
		return messages, changed, nil
	}

	projects, err := client.Projects(ctx, host.DefaultOrgID)
	if err != nil {
		return messages, changed, err
	}
	if strings.TrimSpace(host.DefaultProjectID) == "" && len(projects) == 1 {
		project := projects[0]
		host.DefaultProjectID = project.ID
		host.DefaultProjectName = project.Name
		messages = append(messages, fmt.Sprintf("Selected the only available project: %s", project.Name))
		changed = true
	} else if strings.TrimSpace(host.DefaultProjectID) != "" {
		if project := findPlatformToolProjectByID(projects, host.DefaultProjectID); project != nil && host.DefaultProjectName != project.Name {
			host.DefaultProjectName = project.Name
			changed = true
		}
	}
	return messages, changed, nil
}

func findPlatformToolOrgByID(orgs []platformapi.Organization, id string) *platformapi.Organization {
	for i := range orgs {
		if orgs[i].ID == id {
			return &orgs[i]
		}
	}
	return nil
}

func findPlatformToolProjectByID(projects []platformapi.Project, id string) *platformapi.Project {
	for i := range projects {
		if projects[i].ID == id {
			return &projects[i]
		}
	}
	return nil
}

func resolvePlatformToolOrg(ctx context.Context, session *platformToolSession, value string) (*platformapi.Organization, error) {
	orgs, err := session.Client.Organizations(ctx)
	if err != nil {
		return nil, err
	}
	if len(orgs) == 0 {
		return nil, fmt.Errorf("signed in, but this account does not belong to any organization on %s", session.Host.URL)
	}
	needle := strings.TrimSpace(value)
	if needle == "" {
		needle = session.Host.DefaultOrgID
	}
	if needle == "" && len(orgs) == 1 {
		needle = orgs[0].ID
	}
	if needle == "" {
		return nil, fmt.Errorf("org is required because no hosted WhoDB organization is selected. Call whodb_platform_orgs, then run whodb-cli use --org <org> --project <project> or pass org")
	}
	for i := range orgs {
		org := &orgs[i]
		if matchesPlatformSourceIdentifier(needle, org.ID, org.Slug, org.Name) {
			return org, nil
		}
	}
	return nil, fmt.Errorf("organization %q not found", needle)
}

func platformOrgInfos(orgs []platformapi.Organization, selectedOrgID string) []PlatformOrgInfo {
	result := make([]PlatformOrgInfo, 0, len(orgs))
	for _, org := range orgs {
		result = append(result, PlatformOrgInfo{
			ID:       org.ID,
			Name:     org.Name,
			Slug:     org.Slug,
			Selected: org.ID == selectedOrgID,
		})
	}
	return result
}

func platformProjectInfos(projects []platformapi.Project, orgName, selectedProjectID string) []PlatformProjectInfo {
	result := make([]PlatformProjectInfo, 0, len(projects))
	for _, project := range projects {
		result = append(result, PlatformProjectInfo{
			ID:          project.ID,
			OrgID:       project.OrgID,
			OrgName:     orgName,
			Name:        project.Name,
			Slug:        project.Slug,
			Description: project.Description,
			Selected:    project.ID == selectedProjectID,
		})
	}
	return result
}

func matchesPlatformSourceIdentifier(needle string, values ...string) bool {
	needle = strings.TrimSpace(needle)
	for _, value := range values {
		if strings.EqualFold(needle, strings.TrimSpace(value)) {
			return true
		}
	}
	return false
}

func platformSourceCreateInput(session *platformToolSession, sourceType, name string, values map[string]string, advanced map[string]string) platformapi.CreateSourceInput {
	input := platformapi.CreateSourceInput{
		OrgID:        session.Host.DefaultOrgID,
		ProjectID:    session.Host.DefaultProjectID,
		Name:         strings.TrimSpace(name),
		DatabaseType: sourceType,
		Advanced:     map[string]string{},
	}
	for key, value := range advanced {
		input.Advanced[key] = value
	}
	for key, value := range values {
		assignPlatformSourceCreateField(&input, key, value)
	}
	return input
}

func applyPlatformSourceDefaults(sourceType *platformapi.SourceType, input *platformapi.CreateSourceInput) {
	if sourceType == nil || input == nil {
		return
	}
	for _, field := range sourceType.ConnectionFields {
		if field.DefaultValue == nil || *field.DefaultValue == "" {
			continue
		}
		if platformSourceInputFieldValue(input, field.Key) == "" {
			assignPlatformSourceCreateField(input, field.Key, *field.DefaultValue)
		}
	}
}

func platformSourceInputFieldValue(input *platformapi.CreateSourceInput, key string) string {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "hostname":
		return input.Hostname
	case "port":
		return input.Port
	case "username":
		return input.Username
	case "password":
		return input.Password
	case "database":
		return input.Database
	default:
		return input.Advanced[key]
	}
}

func assignPlatformSourceCreateField(input *platformapi.CreateSourceInput, key, value string) {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "hostname":
		input.Hostname = value
	case "port":
		input.Port = value
	case "username":
		input.Username = value
	case "password":
		input.Password = value
	case "database":
		input.Database = value
	default:
		if input.Advanced == nil {
			input.Advanced = map[string]string{}
		}
		input.Advanced[key] = value
	}
}

func platformSourceConfigValues(hostname, port, username, password, database string) map[string]string {
	values := map[string]string{}
	for key, value := range map[string]string{
		"Hostname": hostname,
		"Port":     port,
		"Username": username,
		"Password": password,
		"Database": database,
	} {
		if strings.TrimSpace(value) != "" {
			values[key] = value
		}
	}
	return values
}

func canonicalPlatformSourceValues(sourceType *platformapi.SourceType, values map[string]string) map[string]string {
	if sourceType == nil {
		return values
	}
	known := sourceTypeFieldKeys(sourceType)
	result := map[string]string{}
	for key, value := range values {
		if canonical, ok := known[strings.ToLower(strings.TrimSpace(key))]; ok {
			result[canonical] = value
			continue
		}
		result[key] = value
	}
	return result
}

func splitPlatformSourceAdvanced(sourceType *platformapi.SourceType, advanced map[string]string) (map[string]string, map[string]string) {
	values := map[string]string{}
	remaining := map[string]string{}
	if sourceType == nil {
		return values, advanced
	}
	known := sourceTypeFieldKeys(sourceType)
	for key, value := range advanced {
		if canonical, ok := known[strings.ToLower(strings.TrimSpace(key))]; ok {
			values[canonical] = value
			continue
		}
		remaining[key] = value
	}
	return values, remaining
}

func sourceTypeFieldKeys(sourceType *platformapi.SourceType) map[string]string {
	keys := make(map[string]string, len(sourceType.ConnectionFields))
	for _, field := range sourceType.ConnectionFields {
		keys[strings.ToLower(strings.TrimSpace(field.Key))] = field.Key
	}
	return keys
}

func validatePlatformSourceRequiredFields(sourceType *platformapi.SourceType, input platformapi.CreateSourceInput) error {
	values := map[string]string{}
	for key, value := range input.Advanced {
		values[strings.ToLower(strings.TrimSpace(key))] = value
	}
	for key, value := range map[string]string{
		"hostname": input.Hostname,
		"port":     input.Port,
		"username": input.Username,
		"password": input.Password,
		"database": input.Database,
	} {
		values[key] = value
	}
	for _, field := range sourceType.ConnectionFields {
		if !field.Required {
			continue
		}
		value := strings.TrimSpace(values[strings.ToLower(strings.TrimSpace(field.Key))])
		if value == "" && field.DefaultValue != nil {
			value = strings.TrimSpace(*field.DefaultValue)
		}
		if value == "" {
			return fmt.Errorf("source field %s is required", field.Key)
		}
	}
	return nil
}

func storePendingPlatformAction(action *PendingPlatformAction) (string, time.Time) {
	token := generateConfirmationToken()
	now := time.Now()
	expiresAt := now.Add(5 * time.Minute)

	platformPendingMutex.Lock()
	defer platformPendingMutex.Unlock()
	for key, pending := range pendingPlatformActions {
		if pending.ExpiresAt.Before(now) {
			delete(pendingPlatformActions, key)
		}
	}
	action.Token = token
	action.ExpiresAt = expiresAt
	pendingPlatformActions[token] = action
	return token, expiresAt
}

func getPendingPlatformAction(token string) (*PendingPlatformAction, error) {
	platformPendingMutex.Lock()
	defer platformPendingMutex.Unlock()
	action, ok := pendingPlatformActions[token]
	if !ok {
		return nil, fmt.Errorf("confirmation token not found or expired")
	}
	if action.ExpiresAt.Before(time.Now()) {
		delete(pendingPlatformActions, token)
		return nil, fmt.Errorf("confirmation token has expired")
	}
	return action, nil
}

func consumePendingPlatformAction(token string) {
	platformPendingMutex.Lock()
	defer platformPendingMutex.Unlock()
	delete(pendingPlatformActions, token)
}

func listPendingPlatformActions() []*PendingPlatformAction {
	now := time.Now()
	platformPendingMutex.Lock()
	defer platformPendingMutex.Unlock()

	actions := make([]*PendingPlatformAction, 0, len(pendingPlatformActions))
	for token, action := range pendingPlatformActions {
		if action.ExpiresAt.Before(now) {
			delete(pendingPlatformActions, token)
			continue
		}
		actions = append(actions, action)
	}
	sort.Slice(actions, func(i, j int) bool {
		if actions[i].ExpiresAt.Equal(actions[j].ExpiresAt) {
			return actions[i].Token < actions[j].Token
		}
		return actions[i].ExpiresAt.Before(actions[j].ExpiresAt)
	})
	return actions
}

func (action *PendingPlatformAction) Preview() *PlatformActionPreview {
	if action == nil {
		return nil
	}
	changes := append([]string(nil), action.Changes...)
	return &PlatformActionPreview{
		Operation:   action.Operation,
		Resource:    action.Resource,
		Action:      action.Action,
		Host:        action.Host,
		OrgID:       action.OrgID,
		ProjectID:   action.ProjectID,
		ProjectName: action.ProjectName,
		SourceID:    action.SourceID,
		SourceName:  action.SourceName,
		SourceType:  action.SourceType,
		Changes:     changes,
	}
}

func platformSourceConfigChangeNames(values, advanced map[string]string) []string {
	seen := map[string]struct{}{}
	changes := make([]string, 0, len(values)+len(advanced))
	for key := range values {
		name := strings.TrimSpace(key)
		if name == "" {
			continue
		}
		seen[strings.ToLower(name)] = struct{}{}
		changes = append(changes, name)
	}
	for key := range advanced {
		name := strings.TrimSpace(key)
		if name == "" {
			continue
		}
		if _, ok := seen[strings.ToLower(name)]; ok {
			continue
		}
		changes = append(changes, name)
	}
	return changes
}

func platformSourceConfirmationOutput(requestID, token string, expiresAt time.Time, action string, preview *PlatformActionPreview) PlatformSourceWriteOutput {
	return PlatformSourceWriteOutput{
		ConfirmationRequired: true,
		ConfirmationToken:    token,
		ConfirmationAction:   action,
		ConfirmationPreview:  preview,
		ConfirmationExpiry:   expiresAt.UTC().Format(time.RFC3339),
		Warning:              "This hosted WhoDB source operation requires approval before it runs. Call whodb_platform_confirm with the confirmation_token to continue.",
		RequestID:            requestID,
	}
}

func platformSourceWriteCompletedOutput(requestID, action string, source *platformapi.Source, preview *PlatformActionPreview) PlatformSourceWriteOutput {
	return PlatformSourceWriteOutput{
		Status:              "ok",
		ConfirmationAction:  action,
		ConfirmationPreview: preview,
		Source:              source,
		RequestID:           requestID,
	}
}

// HandlePlatformPending lists pending hosted platform confirmations.
func HandlePlatformPending(ctx context.Context, req *mcp.CallToolRequest, input PlatformPendingInput) (*mcp.CallToolResult, PlatformPendingOutput, error) {
	requestID := generateRequestID("platform_pending")
	startTime := time.Now()

	actions := listPendingPlatformActions()
	pending := make([]PlatformPendingInfo, 0, len(actions))
	for _, action := range actions {
		preview := action.Preview()
		if preview == nil {
			continue
		}
		pending = append(pending, PlatformPendingInfo{
			Token:     action.Token,
			Action:    *preview,
			ExpiresAt: action.ExpiresAt.UTC().Format(time.RFC3339),
		})
	}

	TrackToolCall(ctx, "platform_pending", requestID, true, time.Since(startTime).Milliseconds(), map[string]any{"pending_count": len(pending)})
	return nil, PlatformPendingOutput{Pending: pending, RequestID: requestID}, nil
}

// HandlePlatformConfirm confirms and executes a pending hosted platform write.
func HandlePlatformConfirm(ctx context.Context, req *mcp.CallToolRequest, input ConfirmInput) (*mcp.CallToolResult, ConfirmOutput, error) {
	requestID := generateRequestID("platform_confirm")
	startTime := time.Now()

	if err := ValidateConfirmInput(&input); err != nil {
		TrackToolCall(ctx, "platform_confirm", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "validation"})
		return nil, ConfirmOutput{Error: err.Error(), RequestID: requestID}, nil
	}
	action, err := getPendingPlatformAction(input.Token)
	if err != nil {
		TrackToolCall(ctx, "platform_confirm", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "token_invalid"})
		return nil, ConfirmOutput{Error: err.Error(), RequestID: requestID}, nil
	}
	output, err := executePendingPlatformAction(ctx, action, requestID)
	if err != nil {
		TrackToolCall(ctx, "platform_confirm", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "platform_action"})
		return nil, ConfirmOutput{Error: err.Error(), RequestID: requestID}, nil
	}
	consumePendingPlatformAction(input.Token)
	TrackToolCall(ctx, "platform_confirm", requestID, true, time.Since(startTime).Milliseconds(), map[string]any{"action": action.Operation})
	return nil, output, nil
}

func executePendingPlatformAction(ctx context.Context, action *PendingPlatformAction, requestID string) (ConfirmOutput, error) {
	session, err := loadPlatformWorkspace(ctx)
	if err != nil {
		return ConfirmOutput{}, err
	}
	if session.Host.URL != action.Host {
		return ConfirmOutput{}, fmt.Errorf("hosted WhoDB login changed from %s to %s before confirmation", action.Host, session.Host.URL)
	}
	if session.Host.DefaultOrgID != action.OrgID || session.Host.DefaultProjectID != action.ProjectID {
		return ConfirmOutput{}, fmt.Errorf("hosted WhoDB workspace changed before confirmation")
	}
	if action.Mutation != "" {
		result, err := executePlatformMutation(ctx, session.Client, action.Mutation, action.ProjectID, action.Variables)
		if err != nil {
			return ConfirmOutput{}, err
		}
		resultJSON := ""
		if result != nil {
			resultJSON = string(result.Data)
		}
		return ConfirmOutput{
			Columns:   []string{"operation", "status", "resource", "action", "project_id", "result_json"},
			Rows:      [][]any{{action.Mutation, "ok", action.Resource, action.Action, action.ProjectID, resultJSON}},
			Message:   fmt.Sprintf("Hosted platform mutation %s completed successfully", action.Mutation),
			RequestID: requestID,
		}, nil
	}

	switch action.Operation {
	case "create_source":
		source, err := session.Client.CreateSource(ctx, action.CreateInput)
		if err != nil {
			return ConfirmOutput{}, err
		}
		return platformSourceConfirmOutput("create_source", source, requestID), nil
	case "update_source":
		source, err := session.Client.UpdateSource(ctx, action.UpdateInput)
		if err != nil {
			return ConfirmOutput{}, err
		}
		return platformSourceConfirmOutput("update_source", source, requestID), nil
	case "delete_source":
		if err := session.Client.DeleteSource(ctx, action.OrgID, action.ProjectID, action.SourceID); err != nil {
			return ConfirmOutput{}, err
		}
		return ConfirmOutput{
			Columns:   []string{"operation", "status", "source_id", "source_name", "project_id"},
			Rows:      [][]any{{"delete_source", "ok", action.SourceID, action.SourceName, action.ProjectID}},
			Message:   fmt.Sprintf("Deleted hosted source %s", action.SourceName),
			RequestID: requestID,
		}, nil
	default:
		return ConfirmOutput{}, fmt.Errorf("unknown platform action %q", action.Operation)
	}
}

func platformSourceConfirmOutput(operation string, source *platformapi.Source, requestID string) ConfirmOutput {
	return ConfirmOutput{
		Columns: []string{"operation", "status", "source_id", "source_name", "project_id"},
		Rows: [][]any{{
			operation,
			"ok",
			source.ID,
			source.Name,
			source.ProjectID,
		}},
		Message:   fmt.Sprintf("Hosted source %s completed successfully", operation),
		RequestID: requestID,
	}
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

const descPlatformOrgs = `List hosted WhoDB organizations visible to the authenticated user.

Use this after whodb-cli login to discover valid organization ids, slugs, and names before selecting a workspace. This tool is read-only.`

const descPlatformProjects = `List hosted WhoDB projects in one organization.

Pass org as an organization id, slug, or name. If omitted, the selected organization is used when available; when the account has exactly one organization, that organization is used. This tool is read-only.`

const descPlatformSourceTypes = `List hosted WhoDB source types available for source creation.

Use this before whodb_platform_source_create or draft whodb_platform_source_test to discover valid source_type values. This tool is read-only.`

const descPlatformSourceFields = `List connection fields for one hosted WhoDB source type.

Use this before source creation or draft connection tests to discover required fields and defaults. This tool is read-only and does not expose credentials.`

const descPlatformSourceObjects = `Browse objects in one hosted WhoDB source.

Use the source name or id from whodb_platform_sources. Parent refs use kind:path, for example schema:public. This tool is read-only.`

const descPlatformSourceColumns = `Describe columns for one hosted WhoDB source object.

Use a source name or id and an object ref such as table:public.users. This tool is read-only.`

const descPlatformSourceRows = `Preview rows for one hosted WhoDB source object.

Use a source name or id and an object ref such as table:public.users. Results are capped by the requested limit and the MCP --max-rows setting when provided. This tool is read-only.`

const descPlatformSourceConfig = `Return redacted connection configuration for one hosted WhoDB source.

Secrets such as passwords, tokens, client secrets, and private keys are masked. Use this to understand source shape without exposing credentials.`

const descPlatformSourceTest = `Test a hosted WhoDB source connection.

Pass source to test an existing saved source. Omit source and pass source_type plus connection fields to test a draft config without saving it.`

const descPlatformSourceCreate = `Prepare creation of a hosted WhoDB source in the selected project.

This stores nothing until the returned confirmation token is approved with whodb_platform_confirm. Pass source_type, name, connection fields, and advanced options.`

const descPlatformSourceUpdate = `Prepare update of a hosted WhoDB source.

This changes nothing until the returned confirmation token is approved with whodb_platform_confirm. Omitted config fields preserve the existing hosted values.`

const descPlatformSourceDelete = `Prepare deletion of a hosted WhoDB source.

This deletes nothing until the returned confirmation token is approved with whodb_platform_confirm. Use whodb_platform_sources first to verify the target source.`

const descPlatformPending = `List pending hosted WhoDB platform writes awaiting confirmation.

Use this to recover confirmation tokens returned by hosted platform write tools. Preview output contains metadata and changed field names only; it never includes credential or secret values.`

const descPlatformConfirm = `Confirm and execute a pending hosted WhoDB platform write.

Use the confirmation_token returned by hosted platform write tools. Tokens expire after 5 minutes.`
