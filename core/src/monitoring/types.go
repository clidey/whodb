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
	"time"
)

// MetricType represents the type of metric being collected
type MetricType string

const (
	MetricTypeQueryLatency      MetricType = "query_latency"
	MetricTypeQueryCount        MetricType = "query_count"
	MetricTypeConnectionCount   MetricType = "connection_count"
	MetricTypeErrorCount        MetricType = "error_count"
	MetricTypeCPUUsage          MetricType = "cpu_usage"
	MetricTypeMemoryUsage       MetricType = "memory_usage"
	MetricTypeDiskIO            MetricType = "disk_io"
	MetricTypeCacheHitRatio     MetricType = "cache_hit_ratio"
	MetricTypeTransactionCount  MetricType = "transaction_count"
	MetricTypeLockWaitTime      MetricType = "lock_wait_time"
)

// QueryType represents the type of database query
type QueryType string

const (
	QueryTypeSelect QueryType = "SELECT"
	QueryTypeInsert QueryType = "INSERT"
	QueryTypeUpdate QueryType = "UPDATE"
	QueryTypeDelete QueryType = "DELETE"
	QueryTypeDDL    QueryType = "DDL"
	QueryTypeOther  QueryType = "OTHER"
)

// MetricPoint represents a single metric data point
type MetricPoint struct {
	Timestamp  time.Time              `json:"timestamp"`
	MetricType MetricType             `json:"metric_type"`
	Value      float64                `json:"value"`
	Labels     map[string]string      `json:"labels"`
	Database   string                 `json:"database"`
	Schema     string                 `json:"schema,omitempty"`
}

// MetricConfig represents which metrics are enabled for collection
type MetricConfig struct {
	QueryLatency      bool `json:"query_latency"`
	QueryCount        bool `json:"query_count"`
	ConnectionCount   bool `json:"connection_count"`
	ErrorCount        bool `json:"error_count"`
	CPUUsage          bool `json:"cpu_usage"`
	MemoryUsage       bool `json:"memory_usage"`
	DiskIO            bool `json:"disk_io"`
	CacheHitRatio     bool `json:"cache_hit_ratio"`
	TransactionCount  bool `json:"transaction_count"`
	LockWaitTime      bool `json:"lock_wait_time"`
}

// DefaultMetricConfig returns the default metric configuration
func DefaultMetricConfig() MetricConfig {
	return MetricConfig{
		QueryLatency:    true,
		QueryCount:      true,
		ConnectionCount: true,
		ErrorCount:      true,
		CPUUsage:        false,
		MemoryUsage:     false,
		DiskIO:          false,
		CacheHitRatio:   false,
		TransactionCount: false,
		LockWaitTime:    false,
	}
}

// MetricsQuery represents a query for retrieving metrics
type MetricsQuery struct {
	StartTime   time.Time   `json:"start_time"`
	EndTime     time.Time   `json:"end_time"`
	MetricTypes []MetricType `json:"metric_types,omitempty"`
	Database    string      `json:"database,omitempty"`
	Schema      string      `json:"schema,omitempty"`
	Interval    string      `json:"interval,omitempty"` // e.g., "1m", "5m", "1h"
}

// AggregatedMetric represents aggregated metric data
type AggregatedMetric struct {
	MetricType MetricType             `json:"metric_type"`
	Database   string                 `json:"database"`
	Schema     string                 `json:"schema,omitempty"`
	Labels     map[string]string      `json:"labels,omitempty"`
	Values     []TimeSeriesPoint      `json:"values"`
}

// TimeSeriesPoint represents a point in a time series
type TimeSeriesPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
	Count     int64     `json:"count,omitempty"`
	Min       float64   `json:"min,omitempty"`
	Max       float64   `json:"max,omitempty"`
	Avg       float64   `json:"avg,omitempty"`
	P50       float64   `json:"p50,omitempty"`
	P95       float64   `json:"p95,omitempty"`
	P99       float64   `json:"p99,omitempty"`
}