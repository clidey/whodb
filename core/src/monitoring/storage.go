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
	"encoding/json"
	"fmt"
	"sync"
	"time"
	
	_ "github.com/mattn/go-sqlite3"
)

// MetricsStorage handles persistence of metrics to SQLite
type MetricsStorage struct {
	db           *sql.DB
	mu           sync.RWMutex
	flushTicker  *time.Ticker
	stopCh       chan struct{}
	batchSize    int
	metricBuffer []MetricPoint
	bufferMu     sync.Mutex
}

// NewMetricsStorage creates a new metrics storage instance
func NewMetricsStorage(dbPath string) (*MetricsStorage, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open metrics database: %w", err)
	}
	
	storage := &MetricsStorage{
		db:           db,
		batchSize:    100,
		metricBuffer: make([]MetricPoint, 0, 100),
		stopCh:       make(chan struct{}),
	}
	
	if err := storage.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}
	
	return storage, nil
}

// initSchema creates the necessary tables for storing metrics
func (s *MetricsStorage) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS metrics (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp DATETIME NOT NULL,
		metric_type TEXT NOT NULL,
		value REAL NOT NULL,
		labels TEXT,
		database TEXT NOT NULL,
		schema TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	
	CREATE INDEX IF NOT EXISTS idx_metrics_timestamp ON metrics(timestamp);
	CREATE INDEX IF NOT EXISTS idx_metrics_type_db ON metrics(metric_type, database);
	CREATE INDEX IF NOT EXISTS idx_metrics_timestamp_type ON metrics(timestamp, metric_type);
	
	-- Table for metric metadata
	CREATE TABLE IF NOT EXISTS metric_metadata (
		metric_type TEXT PRIMARY KEY,
		description TEXT,
		unit TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	
	-- Insert default metadata
	INSERT OR IGNORE INTO metric_metadata (metric_type, description, unit) VALUES
		('query_latency', 'Query execution latency', 'milliseconds'),
		('query_count', 'Number of queries executed', 'count'),
		('connection_count', 'Number of active connections', 'count'),
		('error_count', 'Number of errors', 'count'),
		('cpu_usage', 'CPU usage percentage', 'percent'),
		('memory_usage', 'Memory usage', 'bytes'),
		('disk_io', 'Disk I/O operations', 'bytes/second'),
		('cache_hit_ratio', 'Cache hit ratio', 'ratio'),
		('transaction_count', 'Number of transactions', 'count'),
		('lock_wait_time', 'Lock wait time', 'milliseconds');
	`
	
	_, err := s.db.Exec(schema)
	return err
}

// Start begins the background flush process
func (s *MetricsStorage) Start(ctx context.Context) {
	s.flushTicker = time.NewTicker(10 * time.Second)
	go func() {
		for {
			select {
			case <-ctx.Done():
				s.Stop()
				return
			case <-s.stopCh:
				return
			case <-s.flushTicker.C:
				s.flush()
			}
		}
	}()
}

// Stop stops the storage and flushes remaining metrics
func (s *MetricsStorage) Stop() {
	if s.flushTicker != nil {
		s.flushTicker.Stop()
	}
	close(s.stopCh)
	s.flush()
	s.db.Close()
}

// AddMetric adds a metric to the buffer
func (s *MetricsStorage) AddMetric(metric MetricPoint) {
	s.bufferMu.Lock()
	defer s.bufferMu.Unlock()
	
	s.metricBuffer = append(s.metricBuffer, metric)
	
	if len(s.metricBuffer) >= s.batchSize {
		go s.flush()
	}
}

// flush writes buffered metrics to the database
func (s *MetricsStorage) flush() {
	s.bufferMu.Lock()
	if len(s.metricBuffer) == 0 {
		s.bufferMu.Unlock()
		return
	}
	
	metrics := make([]MetricPoint, len(s.metricBuffer))
	copy(metrics, s.metricBuffer)
	s.metricBuffer = s.metricBuffer[:0]
	s.bufferMu.Unlock()
	
	s.mu.Lock()
	defer s.mu.Unlock()
	
	tx, err := s.db.Begin()
	if err != nil {
		return
	}
	defer tx.Rollback()
	
	stmt, err := tx.Prepare(`
		INSERT INTO metrics (timestamp, metric_type, value, labels, database, schema)
		VALUES (?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return
	}
	defer stmt.Close()
	
	for _, metric := range metrics {
		labelsJSON, _ := json.Marshal(metric.Labels)
		_, err = stmt.Exec(
			metric.Timestamp,
			metric.MetricType,
			metric.Value,
			string(labelsJSON),
			metric.Database,
			metric.Schema,
		)
		if err != nil {
			continue
		}
	}
	
	tx.Commit()
}

// QueryMetrics retrieves metrics based on the query parameters
func (s *MetricsStorage) QueryMetrics(ctx context.Context, query MetricsQuery) ([]AggregatedMetric, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	// Build the SQL query
	baseQuery := `
		SELECT 
			metric_type,
			database,
			schema,
			labels,
			timestamp,
			value
		FROM metrics
		WHERE timestamp >= ? AND timestamp <= ?
	`
	
	args := []interface{}{query.StartTime, query.EndTime}
	
	if len(query.MetricTypes) > 0 {
		placeholders := make([]string, len(query.MetricTypes))
		for i, mt := range query.MetricTypes {
			placeholders[i] = "?"
			args = append(args, mt)
		}
		baseQuery += fmt.Sprintf(" AND metric_type IN (%s)", placeholders[0])
	}
	
	if query.Database != "" {
		baseQuery += " AND database = ?"
		args = append(args, query.Database)
	}
	
	if query.Schema != "" {
		baseQuery += " AND schema = ?"
		args = append(args, query.Schema)
	}
	
	baseQuery += " ORDER BY timestamp ASC"
	
	rows, err := s.db.QueryContext(ctx, baseQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	// Process results
	metricsMap := make(map[string]*AggregatedMetric)
	
	for rows.Next() {
		var (
			metricType string
			database   string
			schema     sql.NullString
			labelsJSON sql.NullString
			timestamp  time.Time
			value      float64
		)
		
		if err := rows.Scan(&metricType, &database, &schema, &labelsJSON, &timestamp, &value); err != nil {
			continue
		}
		
		key := fmt.Sprintf("%s|%s|%s", metricType, database, schema.String)
		
		if _, exists := metricsMap[key]; !exists {
			labels := make(map[string]string)
			if labelsJSON.Valid {
				json.Unmarshal([]byte(labelsJSON.String), &labels)
			}
			
			metricsMap[key] = &AggregatedMetric{
				MetricType: MetricType(metricType),
				Database:   database,
				Schema:     schema.String,
				Labels:     labels,
				Values:     []TimeSeriesPoint{},
			}
		}
		
		metricsMap[key].Values = append(metricsMap[key].Values, TimeSeriesPoint{
			Timestamp: timestamp,
			Value:     value,
		})
	}
	
	// Convert map to slice
	result := make([]AggregatedMetric, 0, len(metricsMap))
	for _, metric := range metricsMap {
		result = append(result, *metric)
	}
	
	return result, nil
}

// CleanupOldMetrics removes metrics older than the retention period
func (s *MetricsStorage) CleanupOldMetrics(retentionDays int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	cutoffTime := time.Now().AddDate(0, 0, -retentionDays)
	
	_, err := s.db.Exec("DELETE FROM metrics WHERE timestamp < ?", cutoffTime)
	return err
}