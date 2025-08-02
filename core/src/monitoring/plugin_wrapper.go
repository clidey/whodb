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

package monitoring

import (
	"context"
	"strings"
	"time"
	
	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/settings"
)

// MonitoredPluginWrapper wraps a plugin to add monitoring capabilities
type MonitoredPluginWrapper struct {
	engine.Plugin
	monitor DatabaseMonitor
}

// NewMonitoredPlugin wraps a plugin with monitoring capabilities
func NewMonitoredPlugin(plugin engine.Plugin) engine.Plugin {
	return &MonitoredPluginWrapper{
		Plugin:  plugin,
		monitor: GetGlobalMonitor(),
	}
}

// recordOperation is a helper to record metrics for any operation
func (w *MonitoredPluginWrapper) recordOperation(ctx context.Context, config *engine.PluginConfig, operationType string, operation func() error) error {
	// Check if monitoring is enabled
	if !settings.Get().PerformanceMonitoringEnabled || w.monitor == nil {
		return operation()
	}
	
	database := config.Credentials.Database
	if database == "" {
		database = config.Credentials.Type
	}
	
	start := time.Now()
	err := operation()
	duration := time.Since(start)
	
	// Determine query type based on operation
	queryType := QueryTypeOther
	switch operationType {
	case "GetRows":
		queryType = QueryTypeSelect
	case "AddRow":
		queryType = QueryTypeInsert
	case "UpdateStorageUnit":
		queryType = QueryTypeUpdate
	case "DeleteRow":
		queryType = QueryTypeDelete
	case "AddStorageUnit":
		queryType = QueryTypeDDL
	}
	
	// Record the metric
	if collector := w.monitor.GetCollector(); collector != nil {
		collector.RecordQuery(ctx, database, "", queryType, duration, err)
	}
	
	return err
}

// GetDatabases implements engine.PluginFunctions
func (w *MonitoredPluginWrapper) GetDatabases(config *engine.PluginConfig) ([]string, error) {
	var result []string
	err := w.recordOperation(context.Background(), config, "GetDatabases", func() error {
		var err error
		result, err = w.Plugin.GetDatabases(config)
		return err
	})
	return result, err
}

// GetStorageUnits implements engine.PluginFunctions
func (w *MonitoredPluginWrapper) GetStorageUnits(config *engine.PluginConfig, schema string) ([]engine.StorageUnit, error) {
	var result []engine.StorageUnit
	ctx := context.Background()
	err := w.recordOperation(ctx, config, "GetStorageUnits", func() error {
		var err error
		result, err = w.Plugin.GetStorageUnits(config, schema)
		return err
	})
	
	// Update schema in context for metrics
	if collector := w.monitor.GetCollector(); collector != nil && err == nil {
		database := config.Credentials.Database
		if database == "" {
			database = config.Credentials.Type
		}
		collector.RecordQuery(ctx, database, schema, QueryTypeSelect, 0, nil)
	}
	
	return result, err
}

// GetRows implements engine.PluginFunctions
func (w *MonitoredPluginWrapper) GetRows(config *engine.PluginConfig, schema string, storageUnit string, where *model.WhereCondition, pageSize int, pageOffset int) (*engine.GetRowsResult, error) {
	var result *engine.GetRowsResult
	ctx := context.Background()
	
	start := time.Now()
	result, err := w.Plugin.GetRows(config, schema, storageUnit, where, pageSize, pageOffset)
	duration := time.Since(start)
	
	// Record with schema information
	if settings.Get().PerformanceMonitoringEnabled && w.monitor != nil {
		if collector := w.monitor.GetCollector(); collector != nil {
			database := config.Credentials.Database
			if database == "" {
				database = config.Credentials.Type
			}
			collector.RecordQuery(ctx, database, schema, QueryTypeSelect, duration, err)
		}
	}
	
	return result, err
}

// AddRow implements engine.PluginFunctions
func (w *MonitoredPluginWrapper) AddRow(config *engine.PluginConfig, schema string, storageUnit string, values []engine.Record) (bool, error) {
	var result bool
	ctx := context.Background()
	
	start := time.Now()
	result, err := w.Plugin.AddRow(config, schema, storageUnit, values)
	duration := time.Since(start)
	
	if settings.Get().PerformanceMonitoringEnabled && w.monitor != nil {
		if collector := w.monitor.GetCollector(); collector != nil {
			database := config.Credentials.Database
			if database == "" {
				database = config.Credentials.Type
			}
			collector.RecordQuery(ctx, database, schema, QueryTypeInsert, duration, err)
		}
	}
	
	return result, err
}

// DeleteRow implements engine.PluginFunctions
func (w *MonitoredPluginWrapper) DeleteRow(config *engine.PluginConfig, schema string, storageUnit string, values map[string]string) (bool, error) {
	var result bool
	ctx := context.Background()
	
	start := time.Now()
	result, err := w.Plugin.DeleteRow(config, schema, storageUnit, values)
	duration := time.Since(start)
	
	if settings.Get().PerformanceMonitoringEnabled && w.monitor != nil {
		if collector := w.monitor.GetCollector(); collector != nil {
			database := config.Credentials.Database
			if database == "" {
				database = config.Credentials.Type
			}
			collector.RecordQuery(ctx, database, schema, QueryTypeDelete, duration, err)
		}
	}
	
	return result, err
}

// UpdateStorageUnit implements engine.PluginFunctions
func (w *MonitoredPluginWrapper) UpdateStorageUnit(config *engine.PluginConfig, schema string, storageUnit string, values map[string]string, updatedColumns []string) (bool, error) {
	var result bool
	ctx := context.Background()
	
	start := time.Now()
	result, err := w.Plugin.UpdateStorageUnit(config, schema, storageUnit, values, updatedColumns)
	duration := time.Since(start)
	
	if settings.Get().PerformanceMonitoringEnabled && w.monitor != nil {
		if collector := w.monitor.GetCollector(); collector != nil {
			database := config.Credentials.Database
			if database == "" {
				database = config.Credentials.Type
			}
			collector.RecordQuery(ctx, database, schema, QueryTypeUpdate, duration, err)
		}
	}
	
	return result, err
}

// RawExecute implements engine.PluginFunctions
func (w *MonitoredPluginWrapper) RawExecute(config *engine.PluginConfig, query string) (*engine.GetRowsResult, error) {
	var result *engine.GetRowsResult
	ctx := context.Background()
	
	// Determine query type from the SQL
	queryType := getQueryTypeFromSQL(query)
	
	start := time.Now()
	result, err := w.Plugin.RawExecute(config, query)
	duration := time.Since(start)
	
	if settings.Get().PerformanceMonitoringEnabled && w.monitor != nil {
		if collector := w.monitor.GetCollector(); collector != nil {
			database := config.Credentials.Database
			if database == "" {
				database = config.Credentials.Type
			}
			collector.RecordQuery(ctx, database, "", queryType, duration, err)
		}
	}
	
	return result, err
}

// Helper function to determine query type from SQL
func getQueryTypeFromSQL(query string) QueryType {
	trimmed := strings.TrimSpace(strings.ToUpper(query))
	switch {
	case strings.HasPrefix(trimmed, "SELECT"):
		return QueryTypeSelect
	case strings.HasPrefix(trimmed, "INSERT"):
		return QueryTypeInsert
	case strings.HasPrefix(trimmed, "UPDATE"):
		return QueryTypeUpdate
	case strings.HasPrefix(trimmed, "DELETE"):
		return QueryTypeDelete
	case strings.HasPrefix(trimmed, "CREATE") || strings.HasPrefix(trimmed, "ALTER") || strings.HasPrefix(trimmed, "DROP"):
		return QueryTypeDDL
	default:
		return QueryTypeOther
	}
}