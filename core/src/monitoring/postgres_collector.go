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
	"fmt"
	"time"
	
	"github.com/clidey/whodb/core/src/engine"
)

// PostgresMetricsCollector collects PostgreSQL-specific metrics
type PostgresMetricsCollector struct {
	monitor DatabaseMonitor
}

// NewPostgresMetricsCollector creates a new PostgreSQL metrics collector
func NewPostgresMetricsCollector(monitor DatabaseMonitor) *PostgresMetricsCollector {
	return &PostgresMetricsCollector{
		monitor: monitor,
	}
}

// CollectMetrics collects PostgreSQL-specific performance metrics
func (p *PostgresMetricsCollector) CollectMetrics(ctx context.Context, db *sql.DB, config *engine.PluginConfig) error {
	if p.monitor == nil || p.monitor.GetCollector() == nil {
		return nil
	}
	
	collector := p.monitor.GetCollector()
	database := config.Credentials.Database
	if database == "" {
		database = "postgres"
	}
	
	metrics := make(map[MetricType]float64)
	
	// Collect cache hit ratio
	if err := p.collectCacheHitRatio(ctx, db, metrics); err != nil {
		// Log but don't fail
		fmt.Printf("Failed to collect cache hit ratio: %v\n", err)
	}
	
	// Collect connection stats
	if err := p.collectConnectionStats(ctx, db, collector, database); err != nil {
		fmt.Printf("Failed to collect connection stats: %v\n", err)
	}
	
	// Collect transaction stats
	if err := p.collectTransactionStats(ctx, db, metrics); err != nil {
		fmt.Printf("Failed to collect transaction stats: %v\n", err)
	}
	
	// Collect lock wait times
	if err := p.collectLockStats(ctx, db, metrics); err != nil {
		fmt.Printf("Failed to collect lock stats: %v\n", err)
	}
	
	// Record all collected metrics
	if len(metrics) > 0 {
		collector.RecordDatabaseMetrics(ctx, database, metrics)
	}
	
	return nil
}

// collectCacheHitRatio collects buffer cache hit ratio
func (p *PostgresMetricsCollector) collectCacheHitRatio(ctx context.Context, db *sql.DB, metrics map[MetricType]float64) error {
	query := `
		SELECT 
			sum(heap_blks_hit) / NULLIF(sum(heap_blks_hit) + sum(heap_blks_read), 0)::float AS ratio
		FROM pg_statio_user_tables
	`
	
	var ratio sql.NullFloat64
	if err := db.QueryRowContext(ctx, query).Scan(&ratio); err != nil {
		return err
	}
	
	if ratio.Valid {
		metrics[MetricTypeCacheHitRatio] = ratio.Float64
	}
	
	return nil
}

// collectConnectionStats collects connection statistics
func (p *PostgresMetricsCollector) collectConnectionStats(ctx context.Context, db *sql.DB, collector MetricsCollector, database string) error {
	query := `
		SELECT 
			count(*) FILTER (WHERE state = 'active') as active,
			count(*) FILTER (WHERE state = 'idle') as idle
		FROM pg_stat_activity
		WHERE datname = $1
	`
	
	var active, idle int
	if err := db.QueryRowContext(ctx, query, database).Scan(&active, &idle); err != nil {
		return err
	}
	
	collector.RecordConnection(ctx, database, active, idle)
	return nil
}

// collectTransactionStats collects transaction statistics
func (p *PostgresMetricsCollector) collectTransactionStats(ctx context.Context, db *sql.DB, metrics map[MetricType]float64) error {
	query := `
		SELECT 
			xact_commit + xact_rollback as total_transactions
		FROM pg_stat_database
		WHERE datname = current_database()
	`
	
	var totalTransactions sql.NullInt64
	if err := db.QueryRowContext(ctx, query).Scan(&totalTransactions); err != nil {
		return err
	}
	
	if totalTransactions.Valid {
		metrics[MetricTypeTransactionCount] = float64(totalTransactions.Int64)
	}
	
	return nil
}

// collectLockStats collects lock wait time statistics
func (p *PostgresMetricsCollector) collectLockStats(ctx context.Context, db *sql.DB, metrics map[MetricType]float64) error {
	// This query gets the average lock wait time for the current period
	query := `
		SELECT 
			COALESCE(AVG(EXTRACT(EPOCH FROM (now() - query_start)) * 1000), 0) as avg_wait_ms
		FROM pg_stat_activity
		WHERE wait_event_type = 'Lock'
			AND state = 'active'
	`
	
	var avgWaitMs float64
	if err := db.QueryRowContext(ctx, query).Scan(&avgWaitMs); err != nil {
		return err
	}
	
	metrics[MetricTypeLockWaitTime] = avgWaitMs
	return nil
}

// StartPeriodicCollection starts periodic collection of PostgreSQL metrics
func (p *PostgresMetricsCollector) StartPeriodicCollection(ctx context.Context, db *sql.DB, config *engine.PluginConfig, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := p.CollectMetrics(ctx, db, config); err != nil {
					fmt.Printf("Error collecting PostgreSQL metrics: %v\n", err)
				}
			}
		}
	}()
}