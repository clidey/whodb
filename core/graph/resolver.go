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

package graph

import (
	"context"
	"fmt"

	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/src"
	"github.com/clidey/whodb/core/src/auth"
	"github.com/clidey/whodb/core/src/engine"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct{}

// GetPluginForContext returns the appropriate database plugin and config for the current session.
func GetPluginForContext(ctx context.Context) (*engine.Plugin, *engine.PluginConfig) {
	config := engine.NewPluginConfig(auth.GetCredentials(ctx))
	plugin := src.MainEngine.Choose(engine.DatabaseType(config.Credentials.Type))
	return plugin, config
}

// ValidateStorageUnit checks that a storage unit exists in the given schema.
// This prevents SQL injection by ensuring only existing table names are used.
func ValidateStorageUnit(plugin engine.PluginFunctions, config *engine.PluginConfig, schema string, storageUnit string) error {
	exists, err := plugin.StorageUnitExists(config, schema, storageUnit)
	if err != nil {
		return fmt.Errorf("failed to validate storage unit: %w", err)
	}
	if !exists {
		return fmt.Errorf("storage unit %q not found in schema %q", storageUnit, schema)
	}
	return nil
}

// MapColumnsToModel converts engine columns to GraphQL model columns,
// enriching them with constraint and foreign key information.
func MapColumnsToModel(
	columnsResult []engine.Column,
	constraints map[string]map[string]any,
	foreignKeys map[string]*engine.ForeignKeyRelationship,
) []*model.Column {
	var columns []*model.Column
	for _, column := range columnsResult {
		isPrimary := column.IsPrimary
		if !isPrimary {
			if colConstraints, ok := constraints[column.Name]; ok {
				if primary, exists := colConstraints["primary"]; exists {
					if primaryBool, isBool := primary.(bool); isBool {
						isPrimary = primaryBool
					}
				}
			}
		}

		isForeignKey := column.IsForeignKey
		referencedTable := column.ReferencedTable
		referencedColumn := column.ReferencedColumn
		if !isForeignKey {
			if fk, exists := foreignKeys[column.Name]; exists {
				isForeignKey = true
				referencedTable = &fk.ReferencedTable
				referencedColumn = &fk.ReferencedColumn
			}
		}

		columns = append(columns, &model.Column{
			Type:             column.Type,
			Name:             column.Name,
			IsPrimary:        isPrimary,
			IsForeignKey:     isForeignKey,
			ReferencedTable:  referencedTable,
			ReferencedColumn: referencedColumn,
			Length:           column.Length,
			Precision:        column.Precision,
			Scale:            column.Scale,
		})
	}
	return columns
}

// FetchColumnsForStorageUnit retrieves column information for a single storage unit,
// including constraints and foreign key relationships.
func FetchColumnsForStorageUnit(
	plugin engine.PluginFunctions,
	config *engine.PluginConfig,
	schema string,
	storageUnit string,
) ([]*model.Column, error) {
	typeArg := config.Credentials.Type

	if err := ValidateStorageUnit(plugin, config, schema, storageUnit); err != nil {
		return nil, err
	}

	columnsResult, err := plugin.GetColumnsForTable(config, schema, storageUnit)
	if err != nil {
		return nil, fmt.Errorf("failed to get columns for %s.%s: %w", schema, storageUnit, err)
	}

	constraints, err := plugin.GetColumnConstraints(config, schema, storageUnit)
	if err != nil {
		constraints = make(map[string]map[string]any)
	}

	foreignKeys, err := plugin.GetForeignKeyRelationships(config, schema, storageUnit)
	if err != nil {
		foreignKeys = make(map[string]*engine.ForeignKeyRelationship)
	}

	_ = typeArg // Used for logging in callers if needed
	return MapColumnsToModel(columnsResult, constraints, foreignKeys), nil
}
