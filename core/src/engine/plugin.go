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

package engine

import (
	"errors"

	"github.com/clidey/whodb/core/graph/model"
)

// ErrMultiStatementUnsupported is returned by plugins that do not support
// executing multiple SQL statements in a single request.
var ErrMultiStatementUnsupported = errors.New("multi-statement SQL is not supported by this database")

// Credentials holds authentication and connection details for a database.
type Credentials struct {
	Id          *string  `json:"Id,omitempty"`
	Type        string   `json:"Type"`
	Hostname    string   `json:"Hostname"`
	Username    string   `json:"Username"`
	Password    string   `json:"Password"`
	Database    string   `json:"Database"`
	Advanced    []Record `json:"Advanced,omitempty"`
	AccessToken *string  `json:"AccessToken,omitempty"`
	IsProfile   bool     `json:"IsProfile,omitempty"`
}

// ExternalModel represents an external AI model configuration for chat functionality.
type ExternalModel struct {
	Type     string // Provider type: "OpenAI", "Anthropic", "Ollama", etc.
	Token    string // API key
	Model    string // User-selected model: "gpt-4o", "claude-sonnet-4", etc.
	Endpoint string // Base URL (for Ollama/generic providers)
}

// PluginConfig contains all configuration needed to connect to and operate on a database.
type PluginConfig struct {
	Credentials    *Credentials
	ExternalModel  *ExternalModel
	Transaction    any  // Optional transaction for transactional operations (e.g., *gorm.DB for SQL plugins)
	MultiStatement bool // Hint for plugins that need special handling for multi-statement scripts (e.g., MySQL)
}

// Record represents a key-value pair with optional extra metadata,
// used for column attributes, configuration, and data transfer.
type Record struct {
	Key   string            `json:"Key"`
	Value string            `json:"Value"`
	Extra map[string]string `json:"Extra,omitempty"`
}

// StorageUnit represents a database table, collection, or equivalent storage structure.
type StorageUnit struct {
	Name       string
	Attributes []Record
}

// Column describes a database column including its type, name, and relationship metadata.
type Column struct {
	Type             string
	Name             string
	IsPrimary        bool
	IsAutoIncrement  bool
	IsComputed       bool // Database-managed, generated, etc
	IsForeignKey     bool
	ReferencedTable  *string
	ReferencedColumn *string
	Length           *int // For VARCHAR(n), CHAR(n) types
	Precision        *int // For DECIMAL(p,s) types
	Scale            *int // For DECIMAL(p,s) types
}

// GetRowsResult contains the result of a row query including columns, data, and pagination info.
type GetRowsResult struct {
	Columns       []Column
	Rows          [][]string
	DisableUpdate bool
	TotalCount    int64
}

// GraphUnitRelationshipType defines the cardinality of a relationship between tables.
type GraphUnitRelationshipType string

// GraphUnitRelationship describes a foreign key relationship between two tables.
type GraphUnitRelationship struct {
	Name             string
	RelationshipType GraphUnitRelationshipType
	SourceColumn     *string
	TargetColumn     *string
}

// GraphUnit represents a table and its relationships for graph visualization.
type GraphUnit struct {
	Unit      StorageUnit
	Relations []GraphUnitRelationship
}

// ChatMessage represents a message in an AI chat conversation with optional query results.
type ChatMessage struct {
	Type                 string
	Result               *GetRowsResult
	Text                 string
	RequiresConfirmation bool
}

// ForeignKeyRelationship describes a foreign key constraint on a column.
type ForeignKeyRelationship struct {
	ColumnName       string
	ReferencedTable  string
	ReferencedColumn string
}

// SSLStatus contains verified SSL/TLS connection status from the database.
type SSLStatus struct {
	IsEnabled bool   // Whether SSL/TLS is active on the current connection
	Mode      string // SSL mode: disabled, required, verify-ca, verify-identity, etc.
}

// PluginFunctions defines the interface that all database plugins must implement.
// Each method provides a specific database operation capability.
type PluginFunctions interface {
	GetDatabases(config *PluginConfig) ([]string, error)
	IsAvailable(config *PluginConfig) bool
	GetAllSchemas(config *PluginConfig) ([]string, error)
	GetStorageUnits(config *PluginConfig, schema string) ([]StorageUnit, error)
	StorageUnitExists(config *PluginConfig, schema string, storageUnit string) (bool, error)
	AddStorageUnit(config *PluginConfig, schema string, storageUnit string, fields []Record) (bool, error)
	UpdateStorageUnit(config *PluginConfig, schema string, storageUnit string, values map[string]string, updatedColumns []string) (bool, error)
	AddRow(config *PluginConfig, schema string, storageUnit string, values []Record) (bool, error)
	AddRowReturningID(config *PluginConfig, schema string, storageUnit string, values []Record) (int64, error)
	BulkAddRows(config *PluginConfig, schema string, storageUnit string, rows [][]Record) (bool, error)
	DeleteRow(config *PluginConfig, schema string, storageUnit string, values map[string]string) (bool, error)
	GetRows(config *PluginConfig, schema string, storageUnit string, where *model.WhereCondition, sort []*model.SortCondition, pageSize int, pageOffset int) (*GetRowsResult, error)
	GetRowCount(config *PluginConfig, schema string, storageUnit string, where *model.WhereCondition) (int64, error)
	GetGraph(config *PluginConfig, schema string) ([]GraphUnit, error)
	RawExecute(config *PluginConfig, query string, params ...any) (*GetRowsResult, error)
	Chat(config *PluginConfig, schema string, previousConversation string, query string) ([]*ChatMessage, error)
	ExportData(config *PluginConfig, schema string, storageUnit string, writer func([]string) error, selectedRows []map[string]any) error
	FormatValue(val any) string
	GetColumnsForTable(config *PluginConfig, schema string, storageUnit string) ([]Column, error)

	// Mock data generation methods
	GetColumnConstraints(config *PluginConfig, schema string, storageUnit string) (map[string]map[string]any, error)
	ClearTableData(config *PluginConfig, schema string, storageUnit string) (bool, error)

	// Foreign key detection
	GetForeignKeyRelationships(config *PluginConfig, schema string, storageUnit string) (map[string]*ForeignKeyRelationship, error)

	// Transaction support
	WithTransaction(config *PluginConfig, operation func(tx any) error) error

	// Database metadata for frontend type/operator configuration
	GetDatabaseMetadata() *DatabaseMetadata

	// GetSSLStatus returns the verified SSL/TLS status of the current connection.
	// Returns nil if SSL status cannot be determined (e.g., SQLite) or is not applicable.
	GetSSLStatus(config *PluginConfig) (*SSLStatus, error)
}

// Plugin wraps PluginFunctions with a database type identifier.
type Plugin struct {
	PluginFunctions
	Type DatabaseType
}

// NewPluginConfig creates a new PluginConfig with the given credentials.
func NewPluginConfig(credentials *Credentials) *PluginConfig {
	return &PluginConfig{
		Credentials: credentials,
	}
}
