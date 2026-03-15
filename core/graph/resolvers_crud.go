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

	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/src/analytics"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
)

// AddStorageUnit is the resolver for the AddStorageUnit field.
func (r *mutationResolver) AddStorageUnit(ctx context.Context, schema string, storageUnit string, fields []*model.RecordInput) (*model.StatusResponse, error) {
	plugin, config := GetPluginForContext(ctx)
	typeArg := config.Credentials.Type
	var fieldsMap []engine.Record
	for _, field := range fields {
		extraFields := map[string]string{}
		for _, extraField := range field.Extra {
			extraFields[extraField.Key] = extraField.Value
		}
		fieldsMap = append(fieldsMap, engine.Record{
			Key:   field.Key,
			Value: field.Value,
			Extra: extraFields,
		})
	}
	status, err := plugin.AddStorageUnit(config, schema, storageUnit, fieldsMap)
	if err != nil {
		log.WithFields(log.Fields{
			"operation":     "AddStorageUnit",
			"schema":        schema,
			"storage_unit":  storageUnit,
			"database_type": typeArg,
		}).WithError(err).Error("Database operation failed")
		analytics.CaptureError(ctx, "AddStorageUnit", err, map[string]any{
			"database_type": typeArg,
			"schema_hash":   analytics.HashIdentifier(schema),
			"storage_hash":  analytics.HashIdentifier(storageUnit),
		})
		return nil, err
	}

	analytics.TrackMutation(ctx, "AddStorageUnit", map[string]any{
		"database_type": typeArg,
		"schema_hash":   analytics.HashIdentifier(schema),
		"storage_hash":  analytics.HashIdentifier(storageUnit),
		"field_count":   len(fields),
	})

	return &model.StatusResponse{
		Status: status,
	}, nil
}

// UpdateStorageUnit is the resolver for the UpdateStorageUnit field.
func (r *mutationResolver) UpdateStorageUnit(ctx context.Context, schema string, storageUnit string, values []*model.RecordInput, updatedColumns []string) (*model.StatusResponse, error) {
	plugin, config := GetPluginForContext(ctx)
	typeArg := config.Credentials.Type

	if err := ValidateStorageUnit(plugin, config, schema, storageUnit); err != nil {
		return nil, err
	}

	valuesMap := map[string]string{}
	for _, value := range values {
		valuesMap[value.Key] = value.Value
	}
	status, err := plugin.UpdateStorageUnit(config, schema, storageUnit, valuesMap, updatedColumns)
	if err != nil {
		log.WithFields(log.Fields{
			"operation":       "UpdateStorageUnit",
			"schema":          schema,
			"storage_unit":    storageUnit,
			"database_type":   typeArg,
			"updated_columns": len(updatedColumns),
		}).WithError(err).Error("Database operation failed")
		analytics.CaptureError(ctx, "UpdateStorageUnit", err, map[string]any{
			"database_type":   typeArg,
			"schema_hash":     analytics.HashIdentifier(schema),
			"storage_hash":    analytics.HashIdentifier(storageUnit),
			"updated_columns": len(updatedColumns),
			"values_supplied": len(values),
		})
		return nil, err
	}

	analytics.TrackMutation(ctx, "UpdateStorageUnit", map[string]any{
		"database_type":   typeArg,
		"schema_hash":     analytics.HashIdentifier(schema),
		"storage_hash":    analytics.HashIdentifier(storageUnit),
		"updated_columns": len(updatedColumns),
		"values_supplied": len(values),
	})

	return &model.StatusResponse{
		Status: status,
	}, nil
}

// AddRow is the resolver for the AddRow field.
func (r *mutationResolver) AddRow(ctx context.Context, schema string, storageUnit string, values []*model.RecordInput) (*model.StatusResponse, error) {
	plugin, config := GetPluginForContext(ctx)
	typeArg := config.Credentials.Type

	if err := ValidateStorageUnit(plugin, config, schema, storageUnit); err != nil {
		return nil, err
	}

	log.WithFields(log.Fields{
		"operation":     "AddRow-Resolver",
		"schema":        schema,
		"storage_unit":  storageUnit,
		"database_type": typeArg,
		"values_count":  len(values),
	}).Debug("AddRow resolver called")

	valuesRecords := []engine.Record{}
	for _, field := range values {
		extraFields := map[string]string{}
		for _, extraField := range field.Extra {
			extraFields[extraField.Key] = extraField.Value
		}
		valuesRecords = append(valuesRecords, engine.Record{
			Key:   field.Key,
			Value: field.Value,
			Extra: extraFields,
		})
	}

	status, err := plugin.AddRow(config, schema, storageUnit, valuesRecords)
	if err != nil {
		log.WithFields(log.Fields{
			"operation":     "AddRow",
			"schema":        schema,
			"storage_unit":  storageUnit,
			"database_type": typeArg,
		}).WithError(err).Error("Database operation failed")
		analytics.CaptureError(ctx, "AddRow", err, map[string]any{
			"database_type": typeArg,
			"schema_hash":   analytics.HashIdentifier(schema),
			"storage_hash":  analytics.HashIdentifier(storageUnit),
			"value_count":   len(values),
		})
		return nil, err
	}

	analytics.TrackMutation(ctx, "AddRow", map[string]any{
		"database_type": typeArg,
		"schema_hash":   analytics.HashIdentifier(schema),
		"storage_hash":  analytics.HashIdentifier(storageUnit),
		"value_count":   len(values),
	})

	return &model.StatusResponse{
		Status: status,
	}, nil
}

// DeleteRow is the resolver for the DeleteRow field.
func (r *mutationResolver) DeleteRow(ctx context.Context, schema string, storageUnit string, values []*model.RecordInput) (*model.StatusResponse, error) {
	plugin, config := GetPluginForContext(ctx)
	typeArg := config.Credentials.Type

	if err := ValidateStorageUnit(plugin, config, schema, storageUnit); err != nil {
		return nil, err
	}

	valuesMap := map[string]string{}
	for _, value := range values {
		valuesMap[value.Key] = value.Value
	}
	status, err := plugin.DeleteRow(config, schema, storageUnit, valuesMap)
	if err != nil {
		log.WithFields(log.Fields{
			"operation":     "DeleteRow",
			"schema":        schema,
			"storage_unit":  storageUnit,
			"database_type": typeArg,
		}).WithError(err).Error("Database operation failed")
		analytics.CaptureError(ctx, "DeleteRow", err, map[string]any{
			"database_type": typeArg,
			"schema_hash":   analytics.HashIdentifier(schema),
			"storage_hash":  analytics.HashIdentifier(storageUnit),
			"value_count":   len(values),
		})
		return nil, err
	}

	analytics.TrackMutation(ctx, "DeleteRow", map[string]any{
		"database_type": typeArg,
		"schema_hash":   analytics.HashIdentifier(schema),
		"storage_hash":  analytics.HashIdentifier(storageUnit),
		"value_count":   len(values),
	})

	return &model.StatusResponse{
		Status: status,
	}, nil
}
