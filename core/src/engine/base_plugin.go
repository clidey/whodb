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
	"fmt"

	"github.com/clidey/whodb/core/graph/model"
)

// BasePlugin provides default implementations for all PluginFunctions methods.
// Non-SQL plugins embed this and override only the methods they support.
// User-facing operations return errors.ErrUnsupported; internal operations return empty results.
type BasePlugin struct{}

// Compile-time check that BasePlugin satisfies PluginFunctions.
var _ PluginFunctions = (*BasePlugin)(nil)

func (b *BasePlugin) GetDatabases(_ *PluginConfig) ([]string, error) {
	return nil, errors.ErrUnsupported
}

func (b *BasePlugin) IsAvailable(_ context.Context, _ *PluginConfig) bool {
	return false
}

func (b *BasePlugin) GetAllSchemas(_ *PluginConfig) ([]string, error) {
	return nil, errors.ErrUnsupported
}

func (b *BasePlugin) GetStorageUnits(_ *PluginConfig, _ string) ([]StorageUnit, error) {
	return nil, errors.ErrUnsupported
}

func (b *BasePlugin) StorageUnitExists(_ *PluginConfig, _ string, _ string) (bool, error) {
	return false, errors.ErrUnsupported
}

func (b *BasePlugin) AddStorageUnit(_ *PluginConfig, _ string, _ string, _ []Record) (bool, error) {
	return false, errors.ErrUnsupported
}

func (b *BasePlugin) UpdateStorageUnit(_ *PluginConfig, _ string, _ string, _ map[string]string, _ []string) (bool, error) {
	return false, errors.ErrUnsupported
}

func (b *BasePlugin) AddRow(_ *PluginConfig, _ string, _ string, _ []Record) (bool, error) {
	return false, errors.ErrUnsupported
}

func (b *BasePlugin) AddRowReturningID(_ *PluginConfig, _ string, _ string, _ []Record) (int64, error) {
	return 0, errors.ErrUnsupported
}

func (b *BasePlugin) BulkAddRows(_ *PluginConfig, _ string, _ string, _ [][]Record) (bool, error) {
	return false, errors.ErrUnsupported
}

func (b *BasePlugin) DeleteRow(_ *PluginConfig, _ string, _ string, _ map[string]string) (bool, error) {
	return false, errors.ErrUnsupported
}

func (b *BasePlugin) GetRows(_ *PluginConfig, _ *GetRowsRequest) (*GetRowsResult, error) {
	return nil, errors.ErrUnsupported
}

func (b *BasePlugin) GetRowCount(_ *PluginConfig, _ string, _ string, _ *model.WhereCondition) (int64, error) {
	return 0, errors.ErrUnsupported
}

func (b *BasePlugin) GetGraph(_ *PluginConfig, _ string) ([]GraphUnit, error) {
	return nil, errors.ErrUnsupported
}

func (b *BasePlugin) RawExecute(_ *PluginConfig, _ string, _ ...any) (*GetRowsResult, error) {
	return nil, errors.ErrUnsupported
}

func (b *BasePlugin) Chat(_ *PluginConfig, _ string, _ string, _ string) ([]*ChatMessage, error) {
	return nil, errors.ErrUnsupported
}

func (b *BasePlugin) ExportData(_ *PluginConfig, _ string, _ string, _ func([]string) error, _ []map[string]any) error {
	return errors.ErrUnsupported
}

func (b *BasePlugin) FormatValue(val any) string {
	return fmt.Sprintf("%v", val)
}

func (b *BasePlugin) GetColumnsForTable(_ *PluginConfig, _ string, _ string) ([]Column, error) {
	return nil, errors.ErrUnsupported
}

func (b *BasePlugin) MarkGeneratedColumns(_ *PluginConfig, _ string, _ string, _ []Column) error {
	return nil
}

func (b *BasePlugin) GetColumnConstraints(_ *PluginConfig, _ string, _ string) (map[string]map[string]any, error) {
	return map[string]map[string]any{}, nil
}

func (b *BasePlugin) ClearTableData(_ *PluginConfig, _ string, _ string) (bool, error) {
	return false, errors.ErrUnsupported
}

func (b *BasePlugin) NullifyFKColumn(_ *PluginConfig, _ string, _ string, _ string) error {
	return nil
}

func (b *BasePlugin) GetForeignKeyRelationships(_ *PluginConfig, _ string, _ string) (map[string]*ForeignKeyRelationship, error) {
	return map[string]*ForeignKeyRelationship{}, nil
}

func (b *BasePlugin) WithTransaction(_ *PluginConfig, operation func(tx any) error) error {
	return operation(nil)
}

func (b *BasePlugin) GetDatabaseMetadata() *DatabaseMetadata {
	return nil
}

func (b *BasePlugin) GetSSLStatus(_ *PluginConfig) (*SSLStatus, error) {
	return nil, nil
}

