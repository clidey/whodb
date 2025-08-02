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
	"fmt"
	"sync"
	"time"
	
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

// OTelCollector implements MetricsCollector using OpenTelemetry
type OTelCollector struct {
	meter          metric.Meter
	storage        *MetricsStorage
	config         MetricConfig
	configMu       sync.RWMutex
	
	// Metrics instruments
	queryLatency     metric.Float64Histogram
	queryCounter     metric.Int64Counter
	connectionGauge  metric.Int64ObservableGauge
	errorCounter     metric.Int64Counter
	
	// Connection metrics
	connectionMetrics map[string]connectionInfo
	connectionMu      sync.RWMutex
}

type connectionInfo struct {
	active int64
	idle   int64
}

// NewOTelCollector creates a new OpenTelemetry-based metrics collector
func NewOTelCollector(storage *MetricsStorage) (*OTelCollector, error) {
	// Create resource
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("whodb"),
			semconv.ServiceVersion("1.0.0"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}
	
	// Create meter provider with custom exporter
	exporter := &storageExporter{storage: storage}
	meterProvider := metric.NewMeterProvider(
		metric.WithResource(res),
		metric.WithReader(metric.NewPeriodicReader(exporter, metric.WithInterval(10*time.Second))),
	)
	
	// Set as global provider
	otel.SetMeterProvider(meterProvider)
	
	// Create meter
	meter := meterProvider.Meter("whodb.database.metrics")
	
	collector := &OTelCollector{
		meter:             meter,
		storage:           storage,
		config:            DefaultMetricConfig(),
		connectionMetrics: make(map[string]connectionInfo),
	}
	
	// Initialize instruments
	if err := collector.initInstruments(); err != nil {
		return nil, err
	}
	
	return collector, nil
}

// initInstruments initializes the OpenTelemetry instruments
func (c *OTelCollector) initInstruments() error {
	var err error
	
	// Query latency histogram
	c.queryLatency, err = c.meter.Float64Histogram(
		"database.query.duration",
		metric.WithDescription("Database query duration in milliseconds"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return fmt.Errorf("failed to create query latency histogram: %w", err)
	}
	
	// Query counter
	c.queryCounter, err = c.meter.Int64Counter(
		"database.query.count",
		metric.WithDescription("Number of database queries"),
	)
	if err != nil {
		return fmt.Errorf("failed to create query counter: %w", err)
	}
	
	// Connection gauge
	c.connectionGauge, err = c.meter.Int64ObservableGauge(
		"database.connection.count",
		metric.WithDescription("Number of database connections"),
		metric.WithInt64Callback(c.observeConnections),
	)
	if err != nil {
		return fmt.Errorf("failed to create connection gauge: %w", err)
	}
	
	// Error counter
	c.errorCounter, err = c.meter.Int64Counter(
		"database.error.count",
		metric.WithDescription("Number of database errors"),
	)
	if err != nil {
		return fmt.Errorf("failed to create error counter: %w", err)
	}
	
	return nil
}

// observeConnections is the callback for the connection gauge
func (c *OTelCollector) observeConnections(_ context.Context, o metric.Int64Observer) error {
	c.connectionMu.RLock()
	defer c.connectionMu.RUnlock()
	
	for db, info := range c.connectionMetrics {
		o.Observe(info.active, metric.WithAttributes(
			attribute.String("database", db),
			attribute.String("state", "active"),
		))
		o.Observe(info.idle, metric.WithAttributes(
			attribute.String("database", db),
			attribute.String("state", "idle"),
		))
	}
	
	return nil
}

// RecordQuery implements MetricsCollector
func (c *OTelCollector) RecordQuery(ctx context.Context, database, schema string, queryType QueryType, duration time.Duration, err error) {
	c.configMu.RLock()
	defer c.configMu.RUnlock()
	
	attrs := []attribute.KeyValue{
		attribute.String("database", database),
		attribute.String("query_type", string(queryType)),
	}
	
	if schema != "" {
		attrs = append(attrs, attribute.String("schema", schema))
	}
	
	// Record query latency
	if c.config.QueryLatency {
		c.queryLatency.Record(ctx, duration.Seconds()*1000, metric.WithAttributes(attrs...))
		
		// Also store in our storage
		c.storage.AddMetric(MetricPoint{
			Timestamp:  time.Now(),
			MetricType: MetricTypeQueryLatency,
			Value:      duration.Seconds() * 1000,
			Labels: map[string]string{
				"query_type": string(queryType),
			},
			Database: database,
			Schema:   schema,
		})
	}
	
	// Record query count
	if c.config.QueryCount {
		c.queryCounter.Add(ctx, 1, metric.WithAttributes(attrs...))
		
		c.storage.AddMetric(MetricPoint{
			Timestamp:  time.Now(),
			MetricType: MetricTypeQueryCount,
			Value:      1,
			Labels: map[string]string{
				"query_type": string(queryType),
			},
			Database: database,
			Schema:   schema,
		})
	}
	
	// Record errors
	if err != nil && c.config.ErrorCount {
		errorAttrs := append(attrs, attribute.String("error", err.Error()))
		c.errorCounter.Add(ctx, 1, metric.WithAttributes(errorAttrs...))
		
		c.storage.AddMetric(MetricPoint{
			Timestamp:  time.Now(),
			MetricType: MetricTypeErrorCount,
			Value:      1,
			Labels: map[string]string{
				"query_type": string(queryType),
				"error":      err.Error(),
			},
			Database: database,
			Schema:   schema,
		})
	}
}

// RecordConnection implements MetricsCollector
func (c *OTelCollector) RecordConnection(ctx context.Context, database string, active, idle int) {
	c.configMu.RLock()
	defer c.configMu.RUnlock()
	
	if !c.config.ConnectionCount {
		return
	}
	
	c.connectionMu.Lock()
	c.connectionMetrics[database] = connectionInfo{
		active: int64(active),
		idle:   int64(idle),
	}
	c.connectionMu.Unlock()
	
	// Store in our storage
	c.storage.AddMetric(MetricPoint{
		Timestamp:  time.Now(),
		MetricType: MetricTypeConnectionCount,
		Value:      float64(active),
		Labels: map[string]string{
			"state": "active",
		},
		Database: database,
	})
	
	c.storage.AddMetric(MetricPoint{
		Timestamp:  time.Now(),
		MetricType: MetricTypeConnectionCount,
		Value:      float64(idle),
		Labels: map[string]string{
			"state": "idle",
		},
		Database: database,
	})
}

// RecordDatabaseMetrics implements MetricsCollector
func (c *OTelCollector) RecordDatabaseMetrics(ctx context.Context, database string, metrics map[MetricType]float64) {
	c.configMu.RLock()
	defer c.configMu.RUnlock()
	
	for metricType, value := range metrics {
		// Check if this metric type is enabled
		enabled := false
		switch metricType {
		case MetricTypeCPUUsage:
			enabled = c.config.CPUUsage
		case MetricTypeMemoryUsage:
			enabled = c.config.MemoryUsage
		case MetricTypeDiskIO:
			enabled = c.config.DiskIO
		case MetricTypeCacheHitRatio:
			enabled = c.config.CacheHitRatio
		case MetricTypeTransactionCount:
			enabled = c.config.TransactionCount
		case MetricTypeLockWaitTime:
			enabled = c.config.LockWaitTime
		}
		
		if enabled {
			c.storage.AddMetric(MetricPoint{
				Timestamp:  time.Now(),
				MetricType: metricType,
				Value:      value,
				Database:   database,
			})
		}
	}
}

// QueryMetrics implements MetricsCollector
func (c *OTelCollector) QueryMetrics(ctx context.Context, query MetricsQuery) ([]AggregatedMetric, error) {
	return c.storage.QueryMetrics(ctx, query)
}

// GetConfig implements MetricsCollector
func (c *OTelCollector) GetConfig() MetricConfig {
	c.configMu.RLock()
	defer c.configMu.RUnlock()
	return c.config
}

// UpdateConfig implements MetricsCollector
func (c *OTelCollector) UpdateConfig(config MetricConfig) error {
	c.configMu.Lock()
	defer c.configMu.Unlock()
	c.config = config
	return nil
}

// Start implements MetricsCollector
func (c *OTelCollector) Start(ctx context.Context) error {
	c.storage.Start(ctx)
	return nil
}

// Stop implements MetricsCollector
func (c *OTelCollector) Stop() error {
	c.storage.Stop()
	return nil
}

// storageExporter is a custom OpenTelemetry exporter that writes to our storage
type storageExporter struct {
	storage *MetricsStorage
}

// Temporality implements metric.Exporter
func (e *storageExporter) Temporality(kind metric.InstrumentKind) metric.Temporality {
	return metric.CumulativeTemporality
}

// Aggregation implements metric.Exporter
func (e *storageExporter) Aggregation(kind metric.InstrumentKind) metric.Aggregation {
	switch kind {
	case metric.InstrumentKindHistogram:
		return metric.AggregationExplicitBucketHistogram{
			Boundaries: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		}
	default:
		return metric.DefaultAggregationSelector(kind)
	}
}

// Export implements metric.Exporter
func (e *storageExporter) Export(ctx context.Context, data *metric.ResourceMetrics) error {
	// The actual metric recording is done directly in the collector methods
	// This is here to satisfy the interface
	return nil
}

// Shutdown implements metric.Exporter
func (e *storageExporter) Shutdown(ctx context.Context) error {
	return nil
}