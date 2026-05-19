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
	"context"
	"errors"
	"time"

	"github.com/clidey/whodb/core/src/query"
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

// PluginConfig contains all configuration needed to connect to and operate on a database.
type PluginConfig struct {
	Credentials           *Credentials
	ExternalModel         *ExternalModel
	Context               context.Context // Optional request context for database operations.
	Transaction           any             // Optional transaction for transactional operations (e.g., *gorm.DB for SQL plugins)
	MultiStatement        bool            // Hint for plugins that need special handling for multi-statement scripts (e.g., MySQL)
	UpsertPKColumns       []string        // PK columns for ON CONFLICT DO UPDATE; non-nil = upsert mode
	SkipConflictPKColumns []string        // PK columns for ON CONFLICT DO NOTHING (append mode — skip duplicate rows)
}

// GetRowsRequest bundles the parameters for a GetRows query.
type GetRowsRequest struct {
	Schema      string
	StorageUnit string
	Where       *query.WhereCondition
	Sort        []*query.SortCondition
	PageSize    int
	PageOffset  int
}

// PluginFunctions defines the interface that all database plugins must implement.
// Each method provides a specific database operation capability.
type PluginFunctions interface {
	GetDatabases(config *PluginConfig) ([]string, error)
	IsAvailable(ctx context.Context, config *PluginConfig) bool
	GetAllSchemas(config *PluginConfig) ([]string, error)
	GetStorageUnits(config *PluginConfig, schema string) ([]StorageUnit, error)
	StorageUnitExists(config *PluginConfig, schema string, storageUnit string) (bool, error)
	AddStorageUnit(config *PluginConfig, schema string, storageUnit string, fields []Record) (bool, error)
	CreateStorageUnit(config *PluginConfig, schema string, definition ObjectDefinition) (bool, error)
	UpdateStorageUnit(config *PluginConfig, schema string, storageUnit string, values map[string]string, updatedColumns []string) (bool, error)
	AddRow(config *PluginConfig, schema string, storageUnit string, values []Record) (bool, error)
	AddRowReturningID(config *PluginConfig, schema string, storageUnit string, values []Record) (int64, error)
	BulkAddRows(config *PluginConfig, schema string, storageUnit string, rows [][]Record) (bool, error)
	DeleteRow(config *PluginConfig, schema string, storageUnit string, values map[string]string) (bool, error)
	GetRows(config *PluginConfig, req *GetRowsRequest) (*GetRowsResult, error)
	GetRowCount(config *PluginConfig, schema string, storageUnit string, where *query.WhereCondition) (int64, error)
	GetGraph(config *PluginConfig, schema string) ([]GraphUnit, error)
	RawExecute(config *PluginConfig, query string, params ...any) (*GetRowsResult, error)
	Chat(config *PluginConfig, schema string, previousConversation string, query string) ([]*ChatMessage, error)
	ExportData(config *PluginConfig, schema string, storageUnit string, writer func([]string) error, selectedRows []map[string]any) error
	FormatValue(val any) string
	GetColumnsForTable(config *PluginConfig, schema string, storageUnit string) ([]Column, error)

	// MarkGeneratedColumns enriches columns with auto-increment and computed column flags
	// by querying database-specific system catalogs
	MarkGeneratedColumns(config *PluginConfig, schema string, storageUnit string, columns []Column) error

	// Mock data generation methods
	GetColumnConstraints(config *PluginConfig, schema string, storageUnit string) (map[string]map[string]any, error)
	ClearTableData(config *PluginConfig, schema string, storageUnit string) (bool, error)
	NullifyFKColumn(config *PluginConfig, schema string, storageUnit string, column string) error

	// Foreign key detection
	GetForeignKeyRelationships(config *PluginConfig, schema string, storageUnit string) (map[string]*ForeignKeyRelationship, error)

	// Transaction support
	WithTransaction(config *PluginConfig, operation func(tx any) error) error

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

// OperationContext returns the configured request context or context.Background().
func (c *PluginConfig) OperationContext() context.Context {
	if c != nil && c.Context != nil {
		return c.Context
	}
	return context.Background()
}

// OperationContextWithTimeout returns an operation context derived from the
// configured request context with the provided timeout applied.
func (c *PluginConfig) OperationContextWithTimeout(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(c.OperationContext(), timeout)
}
