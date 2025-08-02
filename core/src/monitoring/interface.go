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
	"database/sql"
	"time"
)

// MetricsCollector defines the interface for collecting database metrics
type MetricsCollector interface {
	// RecordQuery records metrics for a database query
	RecordQuery(ctx context.Context, database, schema string, queryType QueryType, duration time.Duration, err error)
	
	// RecordConnection records connection metrics
	RecordConnection(ctx context.Context, database string, active, idle int)
	
	// RecordDatabaseMetrics records database-specific metrics
	RecordDatabaseMetrics(ctx context.Context, database string, metrics map[MetricType]float64)
	
	// QueryMetrics retrieves metrics based on the query parameters
	QueryMetrics(ctx context.Context, query MetricsQuery) ([]AggregatedMetric, error)
	
	// GetConfig returns the current metric configuration
	GetConfig() MetricConfig
	
	// UpdateConfig updates which metrics are collected
	UpdateConfig(config MetricConfig) error
	
	// Start starts the metrics collector
	Start(ctx context.Context) error
	
	// Stop stops the metrics collector
	Stop() error
}

// DatabaseMonitor wraps a database connection with monitoring capabilities
type DatabaseMonitor interface {
	// WrapDB wraps a database connection with monitoring
	WrapDB(db *sql.DB, database string) (*sql.DB, error)
	
	// GetCollector returns the metrics collector
	GetCollector() MetricsCollector
}

// MonitoredPlugin extends a plugin with monitoring capabilities
type MonitoredPlugin interface {
	// SetMonitor sets the database monitor for the plugin
	SetMonitor(monitor DatabaseMonitor)
	
	// GetMonitor returns the database monitor
	GetMonitor() DatabaseMonitor
}