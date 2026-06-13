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

package source

import "io"

// ExternalModel contains the user-selected AI model configuration for chat and
// assistant features.
type ExternalModel struct {
	Type          string
	Token         string
	Model         string
	Endpoint      string
	ContextWindow int64 // optional: model's context window in tokens (0 = unknown)
}

// Record represents a key-value pair with optional extra metadata.
type Record struct {
	Key   string            `json:"Key"`
	Value string            `json:"Value"`
	Extra map[string]string `json:"Extra,omitempty"`
}

// StorageUnit describes a tabular or collection-like storage object.
type StorageUnit struct {
	Name       string
	Attributes []Record
}

// Column describes a source column or field including relationship metadata.
type Column struct {
	Type             string
	Name             string
	MetadataFidelity MetadataFidelity
	IsNullable       bool
	IsPrimary        bool
	IsAutoIncrement  bool
	IsComputed       bool
	IsForeignKey     bool
	ReferencedTable  *string
	ReferencedColumn *string
	Length           *int
	Precision        *int
	Scale            *int
}

// FieldConstraints describes normalized source-level constraints for one
// field, column, or property.
type FieldConstraints struct {
	Name             string
	Type             string
	MetadataFidelity MetadataFidelity
	Nullable         *bool
	Primary          bool
	Unique           bool
	Identity         bool
	DefaultValue     *string
	AllowedValues    []string
	CheckMin         *float64
	CheckMax         *float64
	ForeignKey       *ForeignKeyDefinition
	Length           *int
	Precision        *int
	Scale            *int
}

// ObjectFieldConstraints contains normalized field constraints for one source
// object.
type ObjectFieldConstraints struct {
	Ref    ObjectRef
	Fields []FieldConstraints
}

// RowsResult contains the result of a row query including columns and rows.
type RowsResult struct {
	Columns       []Column
	Rows          [][]string
	DisableUpdate bool
	TotalCount    int64
}

// ContentResult contains a structured content payload for a source object.
type ContentResult struct {
	Text       *string
	MIMEType   string
	IsBinary   bool
	SizeBytes  int64
	Truncated  bool
	FileName   string
	ModifiedAt *string
}

// ContentDownload contains a streaming content payload for a source object.
type ContentDownload struct {
	Reader     io.ReadCloser
	MIMEType   string
	FileName   string
	SizeBytes  int64
	ModifiedAt *string
}

// GraphRelationshipType defines the cardinality of a graph relationship.
type GraphRelationshipType string

const (
	// GraphRelationshipTypeOneToOne identifies a one-to-one relationship.
	GraphRelationshipTypeOneToOne GraphRelationshipType = "OneToOne"
	// GraphRelationshipTypeOneToMany identifies a one-to-many relationship.
	GraphRelationshipTypeOneToMany GraphRelationshipType = "OneToMany"
	// GraphRelationshipTypeManyToOne identifies a many-to-one relationship.
	GraphRelationshipTypeManyToOne GraphRelationshipType = "ManyToOne"
	// GraphRelationshipTypeManyToMany identifies a many-to-many relationship.
	GraphRelationshipTypeManyToMany GraphRelationshipType = "ManyToMany"
	// GraphRelationshipTypeUnknown identifies an unknown relationship.
	GraphRelationshipTypeUnknown GraphRelationshipType = "Unknown"
)

// GraphRelationship describes a relationship between two graph units.
type GraphRelationship struct {
	Name             string
	RelationshipType GraphRelationshipType
	MetadataFidelity MetadataFidelity
	SourceColumn     *string
	TargetColumn     *string
}

// GraphUnit represents one browseable object and its relationships.
type GraphUnit struct {
	Unit      StorageUnit
	Relations []GraphRelationship
}

// ChatMessage represents one assistant/chat response message.
type ChatMessage struct {
	Type                 string
	Result               *RowsResult
	Text                 string
	RequiresConfirmation bool
}

// ForeignKeyRelationship describes a foreign key constraint on a column.
type ForeignKeyRelationship struct {
	ColumnName       string
	ReferencedTable  string
	ReferencedColumn string
}

// SSLStatus contains verified SSL/TLS connection status.
type SSLStatus struct {
	IsEnabled bool
	Mode      string
}

// SSLModeInfo describes one selectable SSL/TLS mode for a source type.
type SSLModeInfo struct {
	Value       string
	Label       string
	Description string
	Aliases     []string
}

// TypeSessionMetadata describes editor/query-builder metadata for one source type.
type TypeSessionMetadata struct {
	TypeDefinitions []TypeDefinition
	Operators       []string
	AliasMap        map[string]string
}

// DiscoveryPrefill describes how cloud-discovered resources should prefill a
// connection form for a source type.
type DiscoveryPrefill struct {
	AdvancedDefaults []DiscoveryAdvancedDefault
}

// DiscoveryAdvancedDefault describes one discovered-metadata rule for an
// advanced connection field.
type DiscoveryAdvancedDefault struct {
	Key           string
	Value         string
	MetadataKey   string
	DefaultValue  string
	ProviderTypes []string
	Conditions    []DiscoveryMetadataCondition
}

// DiscoveryMetadataCondition restricts one discovery prefill rule to a
// specific discovered metadata value.
type DiscoveryMetadataCondition struct {
	Key   string
	Value string
}

// TypeCategory groups type definitions for UI consumers.
type TypeCategory string

const (
	// TypeCategoryNumeric groups numeric types.
	TypeCategoryNumeric TypeCategory = "numeric"
	// TypeCategoryText groups text types.
	TypeCategoryText TypeCategory = "text"
	// TypeCategoryBinary groups binary types.
	TypeCategoryBinary TypeCategory = "binary"
	// TypeCategoryDatetime groups date/time types.
	TypeCategoryDatetime TypeCategory = "datetime"
	// TypeCategoryBoolean groups boolean types.
	TypeCategoryBoolean TypeCategory = "boolean"
	// TypeCategoryJSON groups JSON/document types.
	TypeCategoryJSON TypeCategory = "json"
	// TypeCategoryOther groups uncategorised types.
	TypeCategoryOther TypeCategory = "other"
)

// TypeDefinition describes one source column type and its UI metadata.
type TypeDefinition struct {
	ID               string
	Label            string
	HasLength        bool
	HasPrecision     bool
	DefaultLength    *int
	DefaultPrecision *int
	Category         TypeCategory
	InsertFunc       string
	TableModel       string
	DDLSuffix        string
}

// ObjectCreationMetadata describes the create-object form and capabilities for
// a source type.
type ObjectCreationMetadata struct {
	Supported          bool
	ObjectKind         ObjectKind
	RequiresColumns    bool
	TypeDefinitions    []TypeDefinition
	ColumnCapabilities ColumnCreationCapabilities
	ColumnLabels       ColumnCreationLabels
	TableCapabilities  TableCreationCapabilities
	TableOptions       []CreationOptionDefinition
}

// ColumnCreationCapabilities describes supported column modifiers for object
// creation.
type ColumnCreationCapabilities struct {
	Types               bool
	Nullable            bool
	PrimaryKey          bool
	CompositePrimaryKey bool
	Unique              bool
	Identity            bool
	DefaultValue        bool
	CheckValues         bool
	CheckMinMax         bool
	ForeignKey          bool
}

// ColumnCreationLabels describes source-native labels for column creation
// capabilities.
type ColumnCreationLabels struct {
	Nullable     string
	PrimaryKey   string
	Unique       string
	Identity     string
	DefaultValue string
	CheckValues  string
	CheckMin     string
	CheckMax     string
	ForeignKey   string
}

// DefaultColumnCreationLabels returns generic labels for normalized column
// creation capabilities.
func DefaultColumnCreationLabels() ColumnCreationLabels {
	return ColumnCreationLabels{
		Nullable:     "Nullable",
		PrimaryKey:   "Primary key",
		Unique:       "Unique",
		Identity:     "Identity",
		DefaultValue: "Default value",
		CheckValues:  "Allowed values",
		CheckMin:     "Minimum",
		CheckMax:     "Maximum",
		ForeignKey:   "Foreign key",
	}
}

// ColumnCreationLabelsWithDefaults fills unset labels with generic labels.
func ColumnCreationLabelsWithDefaults(labels ColumnCreationLabels) ColumnCreationLabels {
	defaults := DefaultColumnCreationLabels()
	if labels.Nullable == "" {
		labels.Nullable = defaults.Nullable
	}
	if labels.PrimaryKey == "" {
		labels.PrimaryKey = defaults.PrimaryKey
	}
	if labels.Unique == "" {
		labels.Unique = defaults.Unique
	}
	if labels.Identity == "" {
		labels.Identity = defaults.Identity
	}
	if labels.DefaultValue == "" {
		labels.DefaultValue = defaults.DefaultValue
	}
	if labels.CheckValues == "" {
		labels.CheckValues = defaults.CheckValues
	}
	if labels.CheckMin == "" {
		labels.CheckMin = defaults.CheckMin
	}
	if labels.CheckMax == "" {
		labels.CheckMax = defaults.CheckMax
	}
	if labels.ForeignKey == "" {
		labels.ForeignKey = defaults.ForeignKey
	}
	return labels
}

// TableCreationCapabilities describes supported table-level creation options.
type TableCreationCapabilities struct {
	RequiresPrimaryKey bool
	PartitionKey       bool
	ClusteringKey      bool
	OrderKey           bool
	KeyValueType       bool
}

// CreationOptionDefinition describes one selectable table creation option.
type CreationOptionDefinition struct {
	Key      string
	Label    string
	Required bool
	Values   []string
}

// ObjectDefinition describes a source object to create.
type ObjectDefinition struct {
	Name         string
	Columns      []ColumnDefinition
	TableOptions map[string]string
}

// ColumnDefinition describes one column or field in a create-object request.
type ColumnDefinition struct {
	Name         string
	Type         string
	Nullable     *bool
	Primary      bool
	Unique       bool
	Identity     bool
	DefaultValue *string
	CheckValues  []string
	CheckMin     *float64
	CheckMax     *float64
	ForeignKey   *ForeignKeyDefinition
}

// ForeignKeyDefinition describes one single-column foreign key.
type ForeignKeyDefinition struct {
	Table  string
	Column string
}

// ImportMode controls how imported rows should be applied to the destination
// source object.
type ImportMode string

const (
	// ImportModeAppend inserts rows and skips duplicates when supported.
	ImportModeAppend ImportMode = "APPEND"
	// ImportModeOverwrite clears existing rows before importing.
	ImportModeOverwrite ImportMode = "OVERWRITE"
	// ImportModeUpsert inserts new rows and updates matching rows by key.
	ImportModeUpsert ImportMode = "UPSERT"
)

// ParsedImportFile contains normalized rows parsed from an uploaded file.
type ParsedImportFile struct {
	Columns   []string
	Rows      [][]string
	Truncated bool
	Sheet     string
}

// ImportColumnMapping maps one input column to a destination column.
type ImportColumnMapping struct {
	SourceColumn string
	TargetColumn *string
	Skip         bool
}

// ImportRequest describes a parsed import ready to be applied to a source
// object.
type ImportRequest struct {
	Mode               ImportMode
	Parsed             ParsedImportFile
	Mapping            []ImportColumnMapping
	AllowAutoGenerated bool
	BatchSize          int
}

// ImportResult summarizes a completed import operation.
type ImportResult struct {
	RowsImported int
}

// QuerySuggestion is one source-scoped onboarding suggestion for the query UI.
type QuerySuggestion struct {
	Description string
	Category    string
}

// MockDataTableDetail describes generation output for one table-like object.
type MockDataTableDetail struct {
	Table            string
	RowsGenerated    int
	UsedExistingData bool
}

// MockDataGenerationResult summarizes a completed mock-data generation run.
type MockDataGenerationResult struct {
	TotalGenerated int
	Details        []MockDataTableDetail
	Warnings       []string
}

// MockDataTableDependency describes one object in the mock-data dependency
// graph.
type MockDataTableDependency struct {
	Table            string
	DependsOn        []string
	RowCount         int
	IsBlocked        bool
	UsesExistingData bool
}

// MockDataDependencyAnalysis contains the ordered dependency plan for mock-data
// generation.
type MockDataDependencyAnalysis struct {
	GenerationOrder []string
	Tables          []MockDataTableDependency
	TotalRows       int
	Warnings        []string
	Error           string
}
