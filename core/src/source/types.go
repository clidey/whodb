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

// Package source defines the source-first public contract that sits above the
// existing database plugin layer.
package source

import (
	"strings"
)

// Category identifies the broad family a source belongs to.
type Category string

const (
	// CategoryDatabase is used for database sources.
	CategoryDatabase Category = "Database"
	// CategoryCache is used for cache sources.
	CategoryCache Category = "Cache"
	// CategorySearch is used for search/index sources.
	CategorySearch Category = "Search"
	// CategoryObjectStore is used for object storage sources.
	CategoryObjectStore Category = "ObjectStore"
	// CategoryFileStore is used for filesystem-like sources.
	CategoryFileStore Category = "FileStore"
)

// Model identifies the primary data model of a source.
type Model string

const (
	// ModelRelational is used for relational sources.
	ModelRelational Model = "Relational"
	// ModelDocument is used for document sources.
	ModelDocument Model = "Document"
	// ModelKeyValue is used for key-value sources.
	ModelKeyValue Model = "KeyValue"
	// ModelSearch is used for search/index sources.
	ModelSearch Model = "Search"
	// ModelGraph is used for graph sources.
	ModelGraph Model = "Graph"
	// ModelObject is used for object storage sources.
	ModelObject Model = "Object"
)

// Surface identifies a top-level application surface exposed for a source.
type Surface string

const (
	// SurfaceBrowser enables the object browser.
	SurfaceBrowser Surface = "Browser"
	// SurfaceQuery enables query/scratchpad execution.
	SurfaceQuery Surface = "Query"
	// SurfaceGraph enables graph visualisation.
	SurfaceGraph Surface = "Graph"
	// SurfaceChat enables AI chat.
	SurfaceChat Surface = "Chat"
)

// ObjectKind identifies a browseable object inside a source.
type ObjectKind string

const (
	// ObjectKindDatabase identifies a database/container root.
	ObjectKindDatabase ObjectKind = "Database"
	// ObjectKindSchema identifies a schema/namespace.
	ObjectKindSchema ObjectKind = "Schema"
	// ObjectKindTable identifies a table.
	ObjectKindTable ObjectKind = "Table"
	// ObjectKindView identifies a view.
	ObjectKindView ObjectKind = "View"
	// ObjectKindCollection identifies a document collection.
	ObjectKindCollection ObjectKind = "Collection"
	// ObjectKindIndex identifies an index.
	ObjectKindIndex ObjectKind = "Index"
	// ObjectKindKey identifies a key.
	ObjectKindKey ObjectKind = "Key"
	// ObjectKindItem identifies an item-like entry.
	ObjectKindItem ObjectKind = "Item"
	// ObjectKindFunction identifies a function.
	ObjectKindFunction ObjectKind = "Function"
	// ObjectKindProcedure identifies a procedure.
	ObjectKindProcedure ObjectKind = "Procedure"
	// ObjectKindTrigger identifies a trigger.
	ObjectKindTrigger ObjectKind = "Trigger"
	// ObjectKindSequence identifies a sequence.
	ObjectKindSequence ObjectKind = "Sequence"
)

// Action identifies an operation supported for an object kind.
type Action string

const (
	// ActionBrowse indicates the object can be navigated into or listed.
	ActionBrowse Action = "Browse"
	// ActionInspect indicates metadata can be inspected.
	ActionInspect Action = "Inspect"
	// ActionViewRows indicates tabular rows can be viewed.
	ActionViewRows Action = "ViewRows"
	// ActionViewContent indicates blob/text content can be viewed.
	ActionViewContent Action = "ViewContent"
	// ActionViewDefinition indicates a definition/source view is available.
	ActionViewDefinition Action = "ViewDefinition"
	// ActionCreateChild indicates child objects can be created.
	ActionCreateChild Action = "CreateChild"
	// ActionDelete indicates the object can be removed.
	ActionDelete Action = "Delete"
	// ActionInsertData indicates rows/documents can be inserted.
	ActionInsertData Action = "InsertData"
	// ActionUpdateData indicates rows/documents can be updated.
	ActionUpdateData Action = "UpdateData"
	// ActionDeleteData indicates rows/documents can be deleted.
	ActionDeleteData Action = "DeleteData"
	// ActionImportData indicates import is supported.
	ActionImportData Action = "ImportData"
	// ActionGenerateMockData indicates mock data generation is supported.
	ActionGenerateMockData Action = "GenerateMockData"
	// ActionExecute indicates an executable object or query surface is available.
	ActionExecute Action = "Execute"
	// ActionViewGraph indicates the object can be visualised in graph form.
	ActionViewGraph Action = "ViewGraph"
)

// View identifies a UI view that can render an object.
type View string

const (
	// ViewGrid renders a tabular grid.
	ViewGrid View = "Grid"
	// ViewJSON renders JSON content.
	ViewJSON View = "JSON"
	// ViewText renders plain text content.
	ViewText View = "Text"
	// ViewSQL renders SQL/DDL definitions.
	ViewSQL View = "SQL"
	// ViewBinary renders binary/file metadata.
	ViewBinary View = "Binary"
	// ViewMetadata renders object metadata.
	ViewMetadata View = "Metadata"
	// ViewGraph renders a graph.
	ViewGraph View = "Graph"
)

// DataShape identifies the primary data shape exposed by an object kind.
type DataShape string

const (
	// DataShapeTabular is used for row/column data.
	DataShapeTabular DataShape = "Tabular"
	// DataShapeDocument is used for JSON/document data.
	DataShapeDocument DataShape = "Document"
	// DataShapeContent is used for blob/text content.
	DataShapeContent DataShape = "Content"
	// DataShapeGraph is used for graph data.
	DataShapeGraph DataShape = "Graph"
	// DataShapeMetadata is used for metadata-only objects.
	DataShapeMetadata DataShape = "Metadata"
)

// ConnectionFieldKind identifies how a connection field is rendered.
type ConnectionFieldKind string

const (
	// ConnectionFieldKindText is a plain text input.
	ConnectionFieldKindText ConnectionFieldKind = "Text"
	// ConnectionFieldKindPassword is a secret/password input.
	ConnectionFieldKindPassword ConnectionFieldKind = "Password"
	// ConnectionFieldKindBoolean is a boolean toggle input.
	ConnectionFieldKindBoolean ConnectionFieldKind = "Boolean"
	// ConnectionFieldKindFilePath is a file path input.
	ConnectionFieldKindFilePath ConnectionFieldKind = "FilePath"
)

// ConnectionFieldSection controls where a connection field appears in the UI.
type ConnectionFieldSection string

const (
	// ConnectionFieldSectionPrimary is shown in the main form.
	ConnectionFieldSectionPrimary ConnectionFieldSection = "Primary"
	// ConnectionFieldSectionAdvanced is shown in the advanced section.
	ConnectionFieldSectionAdvanced ConnectionFieldSection = "Advanced"
)

// CredentialField identifies how a connection field maps into engine credentials.
type CredentialField string

const (
	// CredentialFieldHostname maps to engine credentials Hostname.
	CredentialFieldHostname CredentialField = "Hostname"
	// CredentialFieldUsername maps to engine credentials Username.
	CredentialFieldUsername CredentialField = "Username"
	// CredentialFieldPassword maps to engine credentials Password.
	CredentialFieldPassword CredentialField = "Password"
	// CredentialFieldDatabase maps to engine credentials Database.
	CredentialFieldDatabase CredentialField = "Database"
	// CredentialFieldAdvanced maps to engine credentials Advanced records.
	CredentialFieldAdvanced CredentialField = "Advanced"
)

// ConnectionField describes one source connection field.
type ConnectionField struct {
	Key             string
	Kind            ConnectionFieldKind
	Section         ConnectionFieldSection
	Required        bool
	LabelKey        string
	PlaceholderKey  string
	DefaultValue    string
	SupportsOptions bool
	CredentialField CredentialField
	AdvancedKey     string
}

// ConnectionExtraField describes one advanced connection field and its default
// presentation metadata in the source catalog.
type ConnectionExtraField struct {
	DefaultValue   string
	Kind           ConnectionFieldKind
	Required       bool
	LabelKey       string
	PlaceholderKey string
}

// ConnectionTransport identifies how a source is reached.
type ConnectionTransport string

const (
	// ConnectionTransportNetwork is used for networked sources.
	ConnectionTransportNetwork ConnectionTransport = "Network"
	// ConnectionTransportFile is used for local file-backed sources.
	ConnectionTransportFile ConnectionTransport = "File"
	// ConnectionTransportBridge is used for bridge/sidecar-backed sources.
	ConnectionTransportBridge ConnectionTransport = "Bridge"
)

// HostInputMode identifies how the hostname field should be presented.
type HostInputMode string

const (
	// HostInputModeNone indicates the source does not expose a hostname field.
	HostInputModeNone HostInputMode = "None"
	// HostInputModeHostname indicates the source expects a plain hostname.
	HostInputModeHostname HostInputMode = "Hostname"
	// HostInputModeHostnameOrURL indicates the source accepts a hostname or URL.
	HostInputModeHostnameOrURL HostInputMode = "HostnameOrURL"
)

// HostInputURLParser identifies how hostname URLs should be parsed.
type HostInputURLParser string

const (
	// HostInputURLParserNone indicates no URL parsing is available.
	HostInputURLParserNone HostInputURLParser = "None"
	// HostInputURLParserPostgres parses postgres:// and postgresql:// URLs.
	HostInputURLParserPostgres HostInputURLParser = "Postgres"
	// HostInputURLParserMongoSRV parses mongodb+srv:// URLs.
	HostInputURLParserMongoSRV HostInputURLParser = "MongoSRV"
)

// ProfileLabelStrategy identifies how saved profiles should be labeled in the UI.
type ProfileLabelStrategy string

const (
	// ProfileLabelStrategyDefault uses the generic hostname/username/database fallback.
	ProfileLabelStrategyDefault ProfileLabelStrategy = "Default"
	// ProfileLabelStrategyHostname prefers the hostname as the primary label.
	ProfileLabelStrategyHostname ProfileLabelStrategy = "Hostname"
	// ProfileLabelStrategyDatabase prefers the database/file path as the primary label.
	ProfileLabelStrategyDatabase ProfileLabelStrategy = "Database"
)

// SchemaFidelity identifies whether schema information is exact or sampled.
type SchemaFidelity string

const (
	// SchemaFidelityExact indicates metadata is resolved exactly from the source.
	SchemaFidelityExact SchemaFidelity = "Exact"
	// SchemaFidelitySampled indicates metadata is inferred from sampled data.
	SchemaFidelitySampled SchemaFidelity = "Sampled"
)

// QueryExplainMode identifies how query-plan inspection should be invoked.
type QueryExplainMode string

const (
	// QueryExplainModeNone indicates the source does not declare explain support.
	QueryExplainModeNone QueryExplainMode = "None"
	// QueryExplainModeExplain indicates standard EXPLAIN support.
	QueryExplainModeExplain QueryExplainMode = "Explain"
	// QueryExplainModeExplainAnalyze indicates EXPLAIN ANALYZE support.
	QueryExplainModeExplainAnalyze QueryExplainMode = "ExplainAnalyze"
	// QueryExplainModeExplainPipeline indicates EXPLAIN PIPELINE support.
	QueryExplainModeExplainPipeline QueryExplainMode = "ExplainPipeline"
)

// ConnectionTraits describes UI-facing connection behavior for a source type.
type ConnectionTraits struct {
	Transport               ConnectionTransport
	HostInputMode           HostInputMode
	HostInputURLParser      HostInputURLParser
	SupportsCustomCAContent bool
}

// PresentationTraits describes UI-facing presentation behavior for a source type.
type PresentationTraits struct {
	ProfileLabelStrategy ProfileLabelStrategy
	SchemaFidelity       SchemaFidelity
}

// QueryTraits describes query-surface behavior for a source type.
type QueryTraits struct {
	SupportsAnalyze bool
	ExplainMode     QueryExplainMode
}

// MockDataTraits describes mock-data behavior for a source type.
type MockDataTraits struct {
	SupportsRelationalDependencies bool
}

// TypeTraits describes non-CRUD source behavior consumed by frontend and CLI.
type TypeTraits struct {
	Connection   ConnectionTraits
	Presentation PresentationTraits
	Query        QueryTraits
	MockData     MockDataTraits
}

// Contract describes the type-level support surface for a source type.
type Contract struct {
	Model             Model
	Surfaces          []Surface
	RootActions       []Action
	BrowsePath        []ObjectKind
	DefaultObjectKind ObjectKind
	GraphScopeKind    *ObjectKind
	ObjectTypes       []ObjectType
}

// SupportsSurface reports whether the contract exposes a given surface.
func (c Contract) SupportsSurface(surface Surface) bool {
	for _, candidate := range c.Surfaces {
		if candidate == surface {
			return true
		}
	}
	return false
}

// SupportsAction reports whether any declared object kind supports the action.
func (c Contract) SupportsAction(action Action) bool {
	for _, objectType := range c.ObjectTypes {
		if objectType.SupportsAction(action) {
			return true
		}
	}
	return false
}

// ObjectTypeForKind looks up the declared object-type contract by kind.
func (c Contract) ObjectTypeForKind(kind ObjectKind) (ObjectType, bool) {
	for _, objectType := range c.ObjectTypes {
		if objectType.Kind == kind {
			return objectType, true
		}
	}
	return ObjectType{}, false
}

// ObjectType describes support for one source object kind.
type ObjectType struct {
	Kind          ObjectKind
	DataShape     DataShape
	Actions       []Action
	Views         []View
	SingularLabel string
	PluralLabel   string
}

// SupportsAction reports whether the object type exposes the action.
func (o ObjectType) SupportsAction(action Action) bool {
	for _, candidate := range o.Actions {
		if candidate == action {
			return true
		}
	}
	return false
}

// TypeSpec describes a connectable source type.
type TypeSpec struct {
	ID               string
	Label            string
	DriverID         string
	Connector        string
	Category         Category
	Traits           TypeTraits
	ConnectionFields []ConnectionField
	Contract         Contract
	DiscoveryPrefill DiscoveryPrefill
	IsAWSManaged     bool
	SSLModes         []SSLModeInfo
}

// ConnectionFieldByKey looks up a connection field by key.
func (s TypeSpec) ConnectionFieldByKey(key string) (ConnectionField, bool) {
	for _, field := range s.ConnectionFields {
		if strings.EqualFold(field.Key, key) {
			return field, true
		}
	}
	return ConnectionField{}, false
}

// Credentials contains the values needed to open a source session.
type Credentials struct {
	ID          *string           `json:"Id,omitempty"`
	SourceType  string            `json:"SourceType"`
	Values      map[string]string `json:"Values,omitempty"`
	AccessToken *string           `json:"AccessToken,omitempty"`
	IsProfile   bool              `json:"IsProfile,omitempty"`
}

// CloneValues returns a copy of the stored credential values.
func (c *Credentials) CloneValues() map[string]string {
	if c == nil || c.Values == nil {
		return map[string]string{}
	}

	values := make(map[string]string, len(c.Values))
	for key, value := range c.Values {
		values[key] = value
	}
	return values
}

// ObjectRef identifies an object within a source.
type ObjectRef struct {
	Kind    ObjectKind
	Locator string
	Path    []string
}

// Object represents one browseable object in a source.
type Object struct {
	Ref         ObjectRef
	Kind        ObjectKind
	Name        string
	Path        []string
	HasChildren bool
	Actions     []Action
	Metadata    []Record
}

// ObjectColumns pairs an object reference with its resolved columns.
type ObjectColumns struct {
	Ref     ObjectRef
	Columns []Column
}

// SessionMetadata contains query-builder/editor metadata for an active session.
type SessionMetadata struct {
	SourceType      string
	QueryLanguages  []string
	TypeDefinitions []TypeDefinition
	Operators       []string
	AliasMap        map[string]string
}

// Profile describes a saved or environment-defined source profile.
type Profile struct {
	ID                   string
	DisplayName          string
	SourceType           string
	Values               map[string]string
	IsEnvironmentDefined bool
	Source               string
	SSLConfigured        bool
}
