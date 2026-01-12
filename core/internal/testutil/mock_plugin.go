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

package testutil

import (
	"fmt"

	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/src/engine"
)

// PluginMock implements engine.PluginFunctions with configurable function hooks.
// It allows tests to override only the behaviors they care about without
// spinning up real database connections.
type PluginMock struct {
	Type engine.DatabaseType

	GetDatabasesFunc         func(*engine.PluginConfig) ([]string, error)
	IsAvailableFunc          func(*engine.PluginConfig) bool
	GetAllSchemasFunc        func(*engine.PluginConfig) ([]string, error)
	GetStorageUnitsFunc      func(*engine.PluginConfig, string) ([]engine.StorageUnit, error)
	StorageUnitExistsFunc    func(*engine.PluginConfig, string, string) (bool, error)
	AddStorageUnitFunc       func(*engine.PluginConfig, string, string, []engine.Record) (bool, error)
	UpdateStorageUnitFunc    func(*engine.PluginConfig, string, string, map[string]string, []string) (bool, error)
	AddRowFunc               func(*engine.PluginConfig, string, string, []engine.Record) (bool, error)
	DeleteRowFunc            func(*engine.PluginConfig, string, string, map[string]string) (bool, error)
	GetRowsFunc              func(*engine.PluginConfig, string, string, *model.WhereCondition, []*model.SortCondition, int, int) (*engine.GetRowsResult, error)
	GetRowCountFunc          func(*engine.PluginConfig, string, string, *model.WhereCondition) (int64, error)
	GetGraphFunc             func(*engine.PluginConfig, string) ([]engine.GraphUnit, error)
	RawExecuteFunc           func(*engine.PluginConfig, string) (*engine.GetRowsResult, error)
	ChatFunc                 func(*engine.PluginConfig, string, string, string, string) ([]*engine.ChatMessage, error)
	ExportDataFunc           func(*engine.PluginConfig, string, string, func([]string) error, []map[string]any) error
	FormatValueFunc          func(any) string
	GetColumnsForTableFunc   func(*engine.PluginConfig, string, string) ([]engine.Column, error)
	GetColumnConstraintsFunc func(*engine.PluginConfig, string, string) (map[string]map[string]any, error)
	ClearTableDataFunc       func(*engine.PluginConfig, string, string) (bool, error)
	GetForeignKeysFunc       func(*engine.PluginConfig, string, string) (map[string]*engine.ForeignKeyRelationship, error)
	WithTransactionFunc      func(*engine.PluginConfig, func(tx any) error) error
	GetDatabaseMetadataFunc  func() *engine.DatabaseMetadata
	GetSSLStatusFunc         func(*engine.PluginConfig) (*engine.SSLStatus, error)
}

// NewPluginMock creates a PluginMock with the provided database type.
func NewPluginMock(dbType engine.DatabaseType) *PluginMock {
	return &PluginMock{Type: dbType}
}

// AsPlugin wraps the mock in an engine.Plugin for code paths that expect it.
func (m *PluginMock) AsPlugin() *engine.Plugin {
	return &engine.Plugin{
		PluginFunctions: m,
		Type:            m.Type,
	}
}

func (m *PluginMock) GetDatabases(config *engine.PluginConfig) ([]string, error) {
	if m.GetDatabasesFunc != nil {
		return m.GetDatabasesFunc(config)
	}
	return nil, nil
}

func (m *PluginMock) IsAvailable(config *engine.PluginConfig) bool {
	if m.IsAvailableFunc != nil {
		return m.IsAvailableFunc(config)
	}
	return true
}

func (m *PluginMock) GetAllSchemas(config *engine.PluginConfig) ([]string, error) {
	if m.GetAllSchemasFunc != nil {
		return m.GetAllSchemasFunc(config)
	}
	return nil, nil
}

func (m *PluginMock) GetStorageUnits(config *engine.PluginConfig, schema string) ([]engine.StorageUnit, error) {
	if m.GetStorageUnitsFunc != nil {
		return m.GetStorageUnitsFunc(config, schema)
	}
	return nil, nil
}

func (m *PluginMock) StorageUnitExists(config *engine.PluginConfig, schema string, storageUnit string) (bool, error) {
	if m.StorageUnitExistsFunc != nil {
		return m.StorageUnitExistsFunc(config, schema, storageUnit)
	}
	return false, nil
}

func (m *PluginMock) AddStorageUnit(config *engine.PluginConfig, schema string, storageUnit string, fields []engine.Record) (bool, error) {
	if m.AddStorageUnitFunc != nil {
		return m.AddStorageUnitFunc(config, schema, storageUnit, fields)
	}
	return false, nil
}

func (m *PluginMock) UpdateStorageUnit(config *engine.PluginConfig, schema string, storageUnit string, values map[string]string, updatedColumns []string) (bool, error) {
	if m.UpdateStorageUnitFunc != nil {
		return m.UpdateStorageUnitFunc(config, schema, storageUnit, values, updatedColumns)
	}
	return false, nil
}

func (m *PluginMock) AddRow(config *engine.PluginConfig, schema string, storageUnit string, values []engine.Record) (bool, error) {
	if m.AddRowFunc != nil {
		return m.AddRowFunc(config, schema, storageUnit, values)
	}
	return false, nil
}

func (m *PluginMock) DeleteRow(config *engine.PluginConfig, schema string, storageUnit string, values map[string]string) (bool, error) {
	if m.DeleteRowFunc != nil {
		return m.DeleteRowFunc(config, schema, storageUnit, values)
	}
	return false, nil
}

func (m *PluginMock) GetRows(config *engine.PluginConfig, schema string, storageUnit string, where *model.WhereCondition, sort []*model.SortCondition, pageSize int, pageOffset int) (*engine.GetRowsResult, error) {
	if m.GetRowsFunc != nil {
		return m.GetRowsFunc(config, schema, storageUnit, where, sort, pageSize, pageOffset)
	}
	return nil, nil
}

func (m *PluginMock) GetRowCount(config *engine.PluginConfig, schema string, storageUnit string, where *model.WhereCondition) (int64, error) {
	if m.GetRowCountFunc != nil {
		return m.GetRowCountFunc(config, schema, storageUnit, where)
	}
	return 0, nil
}

func (m *PluginMock) GetGraph(config *engine.PluginConfig, schema string) ([]engine.GraphUnit, error) {
	if m.GetGraphFunc != nil {
		return m.GetGraphFunc(config, schema)
	}
	return nil, nil
}

func (m *PluginMock) RawExecute(config *engine.PluginConfig, query string) (*engine.GetRowsResult, error) {
	if m.RawExecuteFunc != nil {
		return m.RawExecuteFunc(config, query)
	}
	return nil, nil
}

func (m *PluginMock) Chat(config *engine.PluginConfig, schema string, model string, previousConversation string, query string) ([]*engine.ChatMessage, error) {
	if m.ChatFunc != nil {
		return m.ChatFunc(config, schema, model, previousConversation, query)
	}
	return nil, nil
}

func (m *PluginMock) ExportData(config *engine.PluginConfig, schema string, storageUnit string, writer func([]string) error, selectedRows []map[string]any) error {
	if m.ExportDataFunc != nil {
		return m.ExportDataFunc(config, schema, storageUnit, writer, selectedRows)
	}
	return nil
}

func (m *PluginMock) FormatValue(val any) string {
	if m.FormatValueFunc != nil {
		return m.FormatValueFunc(val)
	}
	return fmt.Sprint(val)
}

func (m *PluginMock) GetColumnsForTable(config *engine.PluginConfig, schema string, storageUnit string) ([]engine.Column, error) {
	if m.GetColumnsForTableFunc != nil {
		return m.GetColumnsForTableFunc(config, schema, storageUnit)
	}
	return nil, nil
}

func (m *PluginMock) GetColumnConstraints(config *engine.PluginConfig, schema string, storageUnit string) (map[string]map[string]any, error) {
	if m.GetColumnConstraintsFunc != nil {
		return m.GetColumnConstraintsFunc(config, schema, storageUnit)
	}
	return nil, nil
}

func (m *PluginMock) ClearTableData(config *engine.PluginConfig, schema string, storageUnit string) (bool, error) {
	if m.ClearTableDataFunc != nil {
		return m.ClearTableDataFunc(config, schema, storageUnit)
	}
	return false, nil
}

func (m *PluginMock) GetForeignKeyRelationships(config *engine.PluginConfig, schema string, storageUnit string) (map[string]*engine.ForeignKeyRelationship, error) {
	if m.GetForeignKeysFunc != nil {
		return m.GetForeignKeysFunc(config, schema, storageUnit)
	}
	return nil, nil
}

func (m *PluginMock) WithTransaction(config *engine.PluginConfig, operation func(tx any) error) error {
	if m.WithTransactionFunc != nil {
		return m.WithTransactionFunc(config, operation)
	}
	if operation != nil {
		return operation(nil)
	}
	return nil
}

func (m *PluginMock) GetDatabaseMetadata() *engine.DatabaseMetadata {
	if m.GetDatabaseMetadataFunc != nil {
		return m.GetDatabaseMetadataFunc()
	}
	return nil
}

func (m *PluginMock) GetSSLStatus(config *engine.PluginConfig) (*engine.SSLStatus, error) {
	if m.GetSSLStatusFunc != nil {
		return m.GetSSLStatusFunc(config)
	}
	return nil, nil
}
