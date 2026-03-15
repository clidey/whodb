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

package graph

import (
	"context"
	"fmt"

	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/src"
	"github.com/clidey/whodb/core/src/auth"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/env"
	"github.com/clidey/whodb/core/src/log"
	"github.com/clidey/whodb/core/src/mockdata"
	"golang.org/x/sync/errgroup"
)

// Row is the resolver for the Row field.
func (r *queryResolver) Row(ctx context.Context, schema string, storageUnit string, where *model.WhereCondition, sort []*model.SortCondition, pageSize int, pageOffset int) (*model.RowsResult, error) {
	if pageSize <= 0 {
		return nil, fmt.Errorf("pageSize must be greater than 0")
	}
	if pageSize > env.MaxPageSize {
		return nil, fmt.Errorf("pageSize must not exceed %d", env.MaxPageSize)
	}
	if pageOffset < 0 {
		return nil, fmt.Errorf("pageOffset must not be negative")
	}

	plugin, config := GetPluginForContext(ctx)
	typeArg := config.Credentials.Type

	if err := ValidateStorageUnit(plugin, config, schema, storageUnit); err != nil {
		return nil, err
	}

	// Run GetRows and GetColumnsForTable in parallel
	var rowsResult *engine.GetRowsResult
	var tableColumns []engine.Column

	g, _ := errgroup.WithContext(ctx)

	g.Go(func() error {
		var err error
		rowsResult, err = plugin.GetRows(config, &engine.GetRowsRequest{
			Schema:      schema,
			StorageUnit: storageUnit,
			Where:       where,
			Sort:        sort,
			PageSize:    pageSize,
			PageOffset:  pageOffset,
		})
		if err != nil {
			log.WithFields(log.Fields{
				"operation":     "GetRows",
				"schema":        schema,
				"storage_unit":  storageUnit,
				"database_type": typeArg,
				"page_size":     pageSize,
				"page_offset":   pageOffset,
			}).WithError(err).Error("Database operation failed")
			return err
		}
		return nil
	})

	g.Go(func() error {
		var err error
		tableColumns, err = plugin.GetColumnsForTable(config, schema, storageUnit)
		if err != nil {
			log.WithFields(log.Fields{
				"operation":    "GetColumnsForTable",
				"schema":       schema,
				"storage_unit": storageUnit,
				"error":        err.Error(),
			}).Warn("Failed to get table columns")
			tableColumns = nil
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	columnInfo := make(map[string]engine.Column, len(tableColumns))
	for _, col := range tableColumns {
		columnInfo[col.Name] = col
	}

	var columns []*model.Column
	for _, column := range rowsResult.Columns {
		col := columnInfo[column.Name]
		columns = append(columns, &model.Column{
			Type:             column.Type,
			Name:             column.Name,
			IsPrimary:        col.IsPrimary,
			IsForeignKey:     col.IsForeignKey,
			ReferencedTable:  col.ReferencedTable,
			ReferencedColumn: col.ReferencedColumn,
			Length:           column.Length,
			Precision:        column.Precision,
			Scale:            column.Scale,
		})
	}
	return &model.RowsResult{
		Columns:       columns,
		Rows:          rowsResult.Rows,
		DisableUpdate: rowsResult.DisableUpdate,
		TotalCount:    int(rowsResult.TotalCount),
	}, nil
}

// Columns is the resolver for the Columns field.
func (r *queryResolver) Columns(ctx context.Context, schema string, storageUnit string) ([]*model.Column, error) {
	plugin, config := GetPluginForContext(ctx)
	columns, err := FetchColumnsForStorageUnit(plugin, config, schema, storageUnit)
	if err != nil {
		log.WithFields(log.Fields{
			"operation":     "Columns",
			"schema":        schema,
			"storage_unit":  storageUnit,
			"database_type": config.Credentials.Type,
			"error":         err.Error(),
		}).Error("Failed to fetch columns")
		return nil, err
	}
	return columns, nil
}

// ColumnsBatch is the resolver for the ColumnsBatch field.
func (r *queryResolver) ColumnsBatch(ctx context.Context, schema string, storageUnits []string) ([]*model.StorageUnitColumns, error) {
	plugin, config := GetPluginForContext(ctx)

	results := make([]*model.StorageUnitColumns, len(storageUnits))
	g, _ := errgroup.WithContext(ctx)
	g.SetLimit(10)

	for i, storageUnit := range storageUnits {
		i, storageUnit := i, storageUnit
		g.Go(func() error {
			columns, err := FetchColumnsForStorageUnit(plugin, config, schema, storageUnit)
			if err != nil {
				log.WithFields(log.Fields{
					"operation":     "ColumnsBatch",
					"schema":        schema,
					"storage_unit":  storageUnit,
					"database_type": config.Credentials.Type,
					"error":         err.Error(),
				}).Error("Failed to fetch columns")
				// Don't fail the entire batch - just skip this table
				results[i] = nil
				return nil
			}
			results[i] = &model.StorageUnitColumns{
				StorageUnit: storageUnit,
				Columns:     columns,
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	// Filter out nil results (tables that failed to load)
	successfulResults := make([]*model.StorageUnitColumns, 0, len(results))
	for _, result := range results {
		if result != nil {
			successfulResults = append(successfulResults, result)
		}
	}

	return successfulResults, nil
}

// RawExecute is the resolver for the RawExecute field.
func (r *queryResolver) RawExecute(ctx context.Context, query string) (*model.RowsResult, error) {
	plugin, config := GetPluginForContext(ctx)
	typeArg := config.Credentials.Type
	rowsResult, err := plugin.RawExecute(config, query)
	if err != nil {
		log.WithFields(log.Fields{
			"operation":     "RawExecute",
			"database_type": typeArg,
			"query":         query,
		}).WithError(err).Error("Database operation failed")
		return nil, err
	}
	var columns []*model.Column
	for _, column := range rowsResult.Columns {
		columns = append(columns, &model.Column{
			Type:         column.Type,
			Name:         column.Name,
			IsPrimary:    column.IsPrimary,
			IsForeignKey: column.IsForeignKey,
		})
	}
	return &model.RowsResult{
		Columns:       columns,
		Rows:          rowsResult.Rows,
		DisableUpdate: rowsResult.DisableUpdate,
		TotalCount:    int(rowsResult.TotalCount),
	}, nil
}

// Schema is the resolver for the Schema field.
func (r *queryResolver) Schema(ctx context.Context) ([]string, error) {
	plugin, config := GetPluginForContext(ctx)
	typeArg := config.Credentials.Type
	schemas, err := plugin.GetAllSchemas(config)
	if err != nil {
		log.WithFields(log.Fields{
			"operation":     "GetAllSchemas",
			"database_type": typeArg,
		}).WithError(err).Error("Database operation failed")
		return nil, err
	}
	return schemas, nil
}

// StorageUnit is the resolver for the StorageUnit field.
func (r *queryResolver) StorageUnit(ctx context.Context, schema string) ([]*model.StorageUnit, error) {
	plugin, config := GetPluginForContext(ctx)
	typeArg := config.Credentials.Type
	units, err := plugin.GetStorageUnits(config, schema)
	if err != nil {
		log.WithFields(log.Fields{
			"operation":     "GetStorageUnits",
			"schema":        schema,
			"database_type": typeArg,
		}).WithError(err).Error("Database operation failed")
		return nil, err
	}
	var storageUnits []*model.StorageUnit
	for _, unit := range units {
		storageUnit := engine.GetStorageUnitModel(unit)
		storageUnit.IsMockDataGenerationAllowed = mockdata.IsMockDataGenerationAllowed(unit.Name)
		storageUnits = append(storageUnits, storageUnit)
	}
	return storageUnits, nil
}

// Database is the resolver for the Database field.
// This resolver is used in two scenarios:
// 1. Login page: to get available databases (e.g., SQLite files) before authentication
// 2. Sidebar: to get switchable databases when already logged in
//
// For the sidebar case, we use session credentials. For login page, we fall back
// to a minimal config (works for SQLite which scans filesystem, not for MySQL which needs connection).
func (r *queryResolver) Database(ctx context.Context, typeArg string) ([]string, error) {
	plugin := src.MainEngine.Choose(engine.DatabaseType(typeArg))
	if plugin == nil {
		return nil, fmt.Errorf("unsupported database type: %s", typeArg)
	}

	var config *engine.PluginConfig

	// Try to get credentials from session (for sidebar when logged in)
	credentials := auth.GetCredentials(ctx)
	if credentials != nil && credentials.Type == typeArg {
		config = engine.NewPluginConfig(credentials)
	} else {
		// No session or type mismatch - use minimal config. works for sqlite
		config = &engine.PluginConfig{
			Credentials: &engine.Credentials{
				Type: typeArg,
			},
		}
	}

	databases, err := plugin.GetDatabases(config)
	if err != nil {
		log.WithFields(log.Fields{
			"operation":     "GetDatabases",
			"database_type": typeArg,
		}).WithError(err).Error("Database operation failed")
		return nil, err
	}
	return databases, nil
}

// Graph is the resolver for the Graph field.
func (r *queryResolver) Graph(ctx context.Context, schema string) ([]*model.GraphUnit, error) {
	plugin, config := GetPluginForContext(ctx)
	typeArg := config.Credentials.Type
	graphUnits, err := plugin.GetGraph(config, schema)
	if err != nil {
		log.WithFields(log.Fields{
			"operation":     "GetGraph",
			"schema":        schema,
			"database_type": typeArg,
		}).WithError(err).Error("Database operation failed")
		return nil, err
	}
	var graphUnitsModel []*model.GraphUnit
	for _, graphUnit := range graphUnits {
		var relations []*model.GraphUnitRelationship
		for _, relation := range graphUnit.Relations {
			relations = append(relations, &model.GraphUnitRelationship{
				Name:         relation.Name,
				Relationship: model.GraphUnitRelationshipType(relation.RelationshipType),
				SourceColumn: relation.SourceColumn,
				TargetColumn: relation.TargetColumn,
			})
		}
		graphUnitsModel = append(graphUnitsModel, &model.GraphUnit{
			Unit:      engine.GetStorageUnitModel(graphUnit.Unit),
			Relations: relations,
		})
	}
	return graphUnitsModel, nil
}
