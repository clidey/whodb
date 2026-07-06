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

// ForeignKeyDefinition describes a source field foreign key target.
type ForeignKeyDefinition struct {
	Table  string `json:"table"`
	Column string `json:"column"`
}

// SourceFieldConstraints describes editable field constraints for a source object.
type SourceFieldConstraints struct {
	Name             string                `json:"name"`
	Type             string                `json:"type"`
	MetadataFidelity string                `json:"metadataFidelity"`
	Nullable         *bool                 `json:"nullable,omitempty"`
	Primary          bool                  `json:"primary"`
	Unique           bool                  `json:"unique"`
	Identity         bool                  `json:"identity"`
	DefaultValue     *string               `json:"defaultValue,omitempty"`
	AllowedValues    []string              `json:"allowedValues"`
	CheckMin         *float64              `json:"checkMin,omitempty"`
	CheckMax         *float64              `json:"checkMax,omitempty"`
	ForeignKey       *ForeignKeyDefinition `json:"foreignKey,omitempty"`
	Length           *int                  `json:"length,omitempty"`
	Precision        *int                  `json:"precision,omitempty"`
	Scale            *int                  `json:"scale,omitempty"`
}

// SourceContent describes readable content returned from a hosted source object.
type SourceContent struct {
	Text       *string `json:"text,omitempty"`
	MIMEType   string  `json:"mimeType"`
	IsBinary   bool    `json:"isBinary"`
	SizeBytes  string  `json:"sizeBytes"`
	Truncated  bool    `json:"truncated"`
	FileName   string  `json:"fileName"`
	ModifiedAt *string `json:"modifiedAt,omitempty"`
}

// ProjectSecret describes secret metadata and usage without secret values.
type ProjectSecret struct {
	ID          string                `json:"id"`
	ProjectID   string                `json:"projectId"`
	Name        string                `json:"name"`
	Description string                `json:"description"`
	CreatedBy   string                `json:"createdBy"`
	UpdatedBy   string                `json:"updatedBy"`
	CreatedAt   string                `json:"createdAt"`
	UpdatedAt   string                `json:"updatedAt"`
	LastUsedAt  *string               `json:"lastUsedAt,omitempty"`
	UsedBy      []PlatformSecretUsage `json:"usedBy"`
}

// PlatformSecretUsage describes where a hosted secret is bound.
type PlatformSecretUsage struct {
	ConsumerType string `json:"consumerType"`
	ConsumerID   string `json:"consumerId"`
	ConsumerName string `json:"consumerName"`
	BindingName  string `json:"bindingName"`
	Mode         string `json:"mode"`
}

// AIProvider describes a hosted AI provider without API key material.
type AIProvider struct {
	ID           string `json:"id"`
	ProjectID    string `json:"projectId"`
	Name         string `json:"name"`
	ProviderType string `json:"providerType"`
	Endpoint     string `json:"endpoint"`
	CreatedBy    string `json:"createdBy"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt"`
}

// OntologyProperty describes one ontology field.
type OntologyProperty struct {
	ID               string `json:"id"`
	APIName          string `json:"apiName"`
	DisplayName      string `json:"displayName"`
	Description      string `json:"description"`
	ColumnName       string `json:"columnName"`
	DataType         string `json:"dataType"`
	ArrayElementType string `json:"arrayElementType"`
	IsRequired       bool   `json:"isRequired"`
	Visibility       string `json:"visibility"`
	IsSearchable     bool   `json:"isSearchable"`
	IsSortable       bool   `json:"isSortable"`
	IsEditOnly       bool   `json:"isEditOnly"`
	SortOrder        int    `json:"sortOrder"`
}

// OntologyLink describes one ontology relationship.
type OntologyLink struct {
	ID                       string `json:"id"`
	APIName                  string `json:"apiName"`
	TargetOntologyAPIName    string `json:"targetOntologyApiName"`
	Cardinality              string `json:"cardinality"`
	ForeignKeyProperty       string `json:"foreignKeyProperty"`
	TargetForeignKeyProperty string `json:"targetForeignKeyProperty"`
	JoinTable                string `json:"joinTable"`
	SourceColumnInJoinTable  string `json:"sourceColumnInJoinTable"`
	TargetColumnInJoinTable  string `json:"targetColumnInJoinTable"`
	DisplayName              string `json:"displayName"`
	PluralDisplayName        string `json:"pluralDisplayName"`
	ReverseDisplayName       string `json:"reverseDisplayName"`
}

// Ontology describes one hosted ontology object type.
type Ontology struct {
	ID                string             `json:"id"`
	ProjectID         string             `json:"projectId"`
	APIName           string             `json:"apiName"`
	DisplayName       string             `json:"displayName"`
	PluralDisplayName string             `json:"pluralDisplayName"`
	Description       string             `json:"description"`
	PrimaryKey        string             `json:"primaryKey"`
	SourceID          *string            `json:"sourceId,omitempty"`
	TableName         string             `json:"tableName"`
	SchemaName        string             `json:"schemaName"`
	Status            string             `json:"status"`
	Icon              string             `json:"icon"`
	Color             string             `json:"color"`
	CreatedAt         string             `json:"createdAt"`
	UpdatedAt         string             `json:"updatedAt"`
	Properties        []OntologyProperty `json:"properties"`
	Links             []OntologyLink     `json:"links"`
}

// OntologyFastLookup describes one saved ontology fast lookup.
type OntologyFastLookup struct {
	ID          string   `json:"id"`
	EntityID    string   `json:"entityId"`
	DisplayName string   `json:"displayName"`
	Fields      []string `json:"fields"`
	Status      string   `json:"status"`
	Reason      string   `json:"reason"`
	CreatedAt   string   `json:"createdAt"`
	UpdatedAt   string   `json:"updatedAt"`
}

// OntologyFastLookupSuggestion describes one suggested ontology fast lookup.
type OntologyFastLookupSuggestion struct {
	EntityID    string   `json:"entityId"`
	DisplayName string   `json:"displayName"`
	Fields      []string `json:"fields"`
	Reason      string   `json:"reason"`
	CanCreate   bool     `json:"canCreate"`
}

// ColumnDef describes one dataset column.
type ColumnDef struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	IsNullable bool   `json:"isNullable"`
	IsPrimary  bool   `json:"isPrimary"`
}

// Dataset describes one hosted dataset.
type Dataset struct {
	ID          string      `json:"id"`
	ProjectID   string      `json:"projectId"`
	SourceID    string      `json:"sourceId"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Schema      []ColumnDef `json:"schema"`
	SchemaMode  string      `json:"schemaMode"`
	OwnerID     string      `json:"ownerId"`
	RowCount    int         `json:"rowCount"`
	SizeBytes   int         `json:"sizeBytes"`
	CreatedAt   string      `json:"createdAt"`
	UpdatedAt   string      `json:"updatedAt"`
}

// DatasetQueryResult contains tabular dataset or ontology rows.
type DatasetQueryResult struct {
	Columns []string   `json:"columns"`
	Rows    [][]string `json:"rows"`
	Total   int        `json:"total"`
}

// LineageNode describes one lineage graph node.
type LineageNode struct {
	ID       string `json:"id"`
	NodeType string `json:"nodeType"`
	Name     string `json:"name"`
}

// LineageEdge describes one lineage graph edge.
type LineageEdge struct {
	SourceID   string `json:"sourceId"`
	SourceType string `json:"sourceType"`
	TargetID   string `json:"targetId"`
	TargetType string `json:"targetType"`
	CreatedAt  string `json:"createdAt"`
}

// LineageGraph describes a hosted lineage graph.
type LineageGraph struct {
	Nodes []LineageNode `json:"nodes"`
	Edges []LineageEdge `json:"edges"`
}

// Transform describes one hosted transform.
type Transform struct {
	ID           string `json:"id"`
	ProjectID    string `json:"projectId"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	GraphJSON    string `json:"graphJson"`
	ScheduleCron string `json:"scheduleCron"`
	TriggerMode  string `json:"triggerMode"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt"`
}

// TransformRun describes one hosted transform run.
type TransformRun struct {
	ID           string `json:"id"`
	TransformID  string `json:"transformId"`
	Status       string `json:"status"`
	ErrorMessage string `json:"errorMessage"`
	TriggeredBy  string `json:"triggeredBy"`
	StartedAt    string `json:"startedAt"`
	CompletedAt  string `json:"completedAt"`
}

// FunctionSecretBinding describes a function secret reference without secret values.
type FunctionSecretBinding struct {
	Name     string `json:"name"`
	SecretID string `json:"secretId"`
	Target   string `json:"target"`
}

// FunctionProviderConfig describes one function AI provider model binding.
type FunctionProviderConfig struct {
	ProviderID string `json:"providerId"`
	Model      string `json:"model"`
}

// FunctionFile describes one hosted function source file.
type FunctionFile struct {
	ID      string `json:"id"`
	Path    string `json:"path"`
	Content string `json:"content"`
}

// FunctionDependency describes one hosted function dependency.
type FunctionDependency struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Function describes one hosted ontology function.
type Function struct {
	ID                  string                   `json:"id"`
	ProjectID           string                   `json:"projectId"`
	Name                string                   `json:"name"`
	Description         string                   `json:"description"`
	Language            string                   `json:"language"`
	EntryPoint          string                   `json:"entryPoint"`
	TimeoutSeconds      int                      `json:"timeoutSeconds"`
	Memory              string                   `json:"memory"`
	CPU                 string                   `json:"cpu"`
	CreatedBy           string                   `json:"createdBy"`
	CreatedAt           string                   `json:"createdAt"`
	UpdatedAt           string                   `json:"updatedAt"`
	Files               []FunctionFile           `json:"files"`
	Dependencies        []FunctionDependency     `json:"dependencies"`
	ProviderIDs         []string                 `json:"providerIds"`
	OntologyIDs         []string                 `json:"ontologyIds"`
	ReadOnlyOntologyIDs []string                 `json:"readOnlyOntologyIds"`
	ProviderConfigs     []FunctionProviderConfig `json:"providerConfigs"`
	SecretBindings      []FunctionSecretBinding  `json:"secretBindings"`
	DefaultMaxTokens    int                      `json:"defaultMaxTokens"`
	DefaultTemperature  float64                  `json:"defaultTemperature"`
	IsDeployed          bool                     `json:"isDeployed"`
}

// FunctionExecutionResult describes one hosted function execution result.
type FunctionExecutionResult struct {
	Output     *string `json:"output,omitempty"`
	Logs       string  `json:"logs"`
	DurationMS int     `json:"durationMs"`
	Success    bool    `json:"success"`
	Error      *string `json:"error,omitempty"`
}

// ProjectFolder describes one hosted project folder.
type ProjectFolder struct {
	ID        string  `json:"id"`
	ProjectID string  `json:"projectId"`
	ParentID  *string `json:"parentId,omitempty"`
	Name      string  `json:"name"`
	CreatedBy string  `json:"createdBy"`
	CreatedAt string  `json:"createdAt"`
}

// ProjectFile describes one hosted project file.
type ProjectFile struct {
	ID          string  `json:"id"`
	ProjectID   string  `json:"projectId"`
	FolderID    *string `json:"folderId,omitempty"`
	Name        string  `json:"name"`
	MIMEType    string  `json:"mimeType"`
	SizeBytes   int     `json:"sizeBytes"`
	IsTabular   bool    `json:"isTabular"`
	RowCount    *int    `json:"rowCount,omitempty"`
	ColumnCount *int    `json:"columnCount,omitempty"`
	DatasetID   *string `json:"datasetId,omitempty"`
	UploadedBy  string  `json:"uploadedBy"`
	CreatedAt   string  `json:"createdAt"`
	UpdatedAt   string  `json:"updatedAt"`
}

// FolderContents describes a hosted project folder listing.
type FolderContents struct {
	Folders     []ProjectFolder `json:"folders"`
	Files       []ProjectFile   `json:"files"`
	Breadcrumbs []ProjectFolder `json:"breadcrumbs"`
	StorageUsed int             `json:"storageUsed"`
}

// FilePreviewColumn describes one tabular file preview column.
type FilePreviewColumn struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// TabularPreviewData describes tabular file preview data.
type TabularPreviewData struct {
	Columns    []FilePreviewColumn `json:"columns"`
	Rows       [][]string          `json:"rows"`
	Total      int                 `json:"total"`
	SheetName  *string             `json:"sheetName,omitempty"`
	SheetIndex *int                `json:"sheetIndex,omitempty"`
	SheetCount *int                `json:"sheetCount,omitempty"`
	SheetNames []string            `json:"sheetNames,omitempty"`
}

// FilePreviewResult describes a hosted project file preview.
type FilePreviewResult struct {
	MIMEType    string              `json:"mimeType"`
	SizeBytes   int                 `json:"sizeBytes"`
	IsTabular   bool                `json:"isTabular"`
	Tabular     *TabularPreviewData `json:"tabular,omitempty"`
	TextContent *string             `json:"textContent,omitempty"`
}
