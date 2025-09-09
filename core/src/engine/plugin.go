/*
 * Copyright 2025 Clidey, Inc.
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

import "github.com/clidey/whodb/core/graph/model"

type Credentials struct {
	Id          *string
	Type        string
	Hostname    string
	Username    string
	Password    string
	Database    string
	Advanced    []Record
	AccessToken *string
	IsProfile   bool
}

type ExternalModel struct {
	Type  string
	Token string
}

type PluginConfig struct {
	Credentials   *Credentials
	ExternalModel *ExternalModel
}

type Record struct {
	Key   string
	Value string
	Extra map[string]string
}

type StorageUnit struct {
	Name       string
	Attributes []Record
}

type DatabaseFunction struct {
	Name        string
	ReturnType  string
	Parameters  []Record
	Definition  string
	Language    string
	IsAggregate bool
}

type DatabaseProcedure struct {
	Name       string
	Parameters []Record
	Definition string
	Language   string
}

type DatabaseTrigger struct {
	Name       string
	TableName  string
	Event      string // INSERT, UPDATE, DELETE
	Timing     string // BEFORE, AFTER
	Definition string
}

type DatabaseIndex struct {
	Name        string
	TableName   string
	Columns     []string
	Type        string // BTREE, HASH, etc.
	IsUnique    bool
	IsPrimary   bool
	Size        string
}

type DatabaseSequence struct {
	Name        string
	DataType    string
	StartValue  int64
	Increment   int64
	MinValue    int64
	MaxValue    int64
	CacheSize   int64
	IsCycle     bool
}

type DatabaseType struct {
	Name       string
	Schema     string
	Type       string // ENUM, COMPOSITE, etc.
	Definition string
}

type Column struct {
	Type string
	Name string
}

type GetRowsResult struct {
	Columns       []Column
	Rows          [][]string
	DisableUpdate bool
}

type GraphUnitRelationshipType string

const (
	GraphUnitRelationshipType_OneToOne   = "OneToOne"
	GraphUnitRelationshipType_OneToMany  = "OneToMany"
	GraphUnitRelationshipType_ManyToOne  = "ManyToOne"
	GraphUnitRelationshipType_ManyToMany = "ManyToMany"
	GraphUnitRelationshipType_Unknown    = "Unknown"
)

type GraphUnitRelationship struct {
	Name             string
	RelationshipType GraphUnitRelationshipType
}

type GraphUnit struct {
	Unit      StorageUnit
	Relations []GraphUnitRelationship
}

type ChatMessage struct {
	Type   string
	Result *GetRowsResult
	Text   string
}

type PluginFunctions interface {
	GetDatabases(config *PluginConfig) ([]string, error)
	IsAvailable(config *PluginConfig) bool
	GetAllSchemas(config *PluginConfig) ([]string, error)
	GetStorageUnits(config *PluginConfig, schema string) ([]StorageUnit, error)
	AddStorageUnit(config *PluginConfig, schema string, storageUnit string, fields []Record) (bool, error)
	UpdateStorageUnit(config *PluginConfig, schema string, storageUnit string, values map[string]string, updatedColumns []string) (bool, error)
	AddRow(config *PluginConfig, schema string, storageUnit string, values []Record) (bool, error)
	DeleteRow(config *PluginConfig, schema string, storageUnit string, values map[string]string) (bool, error)
	GetRows(config *PluginConfig, schema string, storageUnit string, where *model.WhereCondition, sort []*model.SortCondition, pageSize int, pageOffset int) (*GetRowsResult, error)
	GetGraph(config *PluginConfig, schema string) ([]GraphUnit, error)
	RawExecute(config *PluginConfig, query string) (*GetRowsResult, error)
	Chat(config *PluginConfig, schema string, model string, previousConversation string, query string) ([]*ChatMessage, error)
	ExportData(config *PluginConfig, schema string, storageUnit string, writer func([]string) error, selectedRows []map[string]any) error
	FormatValue(val any) string
	GetColumnsForTable(config *PluginConfig, schema string, storageUnit string) ([]Column, error)

	// Mock data generation methods
	GetColumnConstraints(config *PluginConfig, schema string, storageUnit string) (map[string]map[string]any, error)
	ClearTableData(config *PluginConfig, schema string, storageUnit string) (bool, error)
	
	// Transaction support
	WithTransaction(config *PluginConfig, operation func(tx any) error) error
	
	// Additional database entities
	GetFunctions(config *PluginConfig, schema string) ([]DatabaseFunction, error)
	GetProcedures(config *PluginConfig, schema string) ([]DatabaseProcedure, error)
	GetTriggers(config *PluginConfig, schema string) ([]DatabaseTrigger, error)
	GetIndexes(config *PluginConfig, schema string) ([]DatabaseIndex, error)
	GetSequences(config *PluginConfig, schema string) ([]DatabaseSequence, error)
	GetTypes(config *PluginConfig, schema string) ([]DatabaseType, error)
}

type Plugin struct {
	PluginFunctions
	Type DatabaseType
}

func NewPluginConfig(credentials *Credentials) *PluginConfig {
	return &PluginConfig{
		Credentials: credentials,
	}
}
