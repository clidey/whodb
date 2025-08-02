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
	"database/sql/driver"
	"fmt"
	"path/filepath"
	"sync"
	
	"github.com/XSAM/otelsql"
	"github.com/clidey/whodb/core/src/settings"
)

var (
	globalMonitor     DatabaseMonitor
	globalMonitorOnce sync.Once
	globalMonitorMu   sync.RWMutex
)

// GetGlobalMonitor returns the global database monitor instance
func GetGlobalMonitor() DatabaseMonitor {
	globalMonitorMu.RLock()
	monitor := globalMonitor
	globalMonitorMu.RUnlock()
	
	if monitor != nil {
		return monitor
	}
	
	// Initialize if not already done
	globalMonitorOnce.Do(func() {
		dbPath := filepath.Join(".", "whodb_metrics.db")
		storage, err := NewMetricsStorage(dbPath)
		if err != nil {
			// Log error but don't fail
			fmt.Printf("Failed to initialize metrics storage: %v\n", err)
			return
		}
		
		collector, err := NewOTelCollector(storage)
		if err != nil {
			// Log error but don't fail
			fmt.Printf("Failed to initialize metrics collector: %v\n", err)
			storage.Stop()
			return
		}
		
		monitor := &databaseMonitor{
			collector: collector,
			enabled:   false,
		}
		
		// Start the collector
		ctx := context.Background()
		if err := collector.Start(ctx); err != nil {
			fmt.Printf("Failed to start metrics collector: %v\n", err)
			return
		}
		
		globalMonitorMu.Lock()
		globalMonitor = monitor
		globalMonitorMu.Unlock()
	})
	
	globalMonitorMu.RLock()
	defer globalMonitorMu.RUnlock()
	return globalMonitor
}

// databaseMonitor implements DatabaseMonitor
type databaseMonitor struct {
	collector MetricsCollector
	enabled   bool
	mu        sync.RWMutex
}

// WrapDB implements DatabaseMonitor
func (m *databaseMonitor) WrapDB(db *sql.DB, database string) (*sql.DB, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// Check if monitoring is enabled in settings
	currentSettings := settings.Get()
	if !currentSettings.PerformanceMonitoringEnabled {
		return db, nil
	}
	
	// Update collector config with current settings
	config := MetricConfig{
		QueryLatency:     currentSettings.PerformanceMetricsConfig["query_latency"],
		QueryCount:       currentSettings.PerformanceMetricsConfig["query_count"],
		ConnectionCount:  currentSettings.PerformanceMetricsConfig["connection_count"],
		ErrorCount:       currentSettings.PerformanceMetricsConfig["error_count"],
		CPUUsage:         currentSettings.PerformanceMetricsConfig["cpu_usage"],
		MemoryUsage:      currentSettings.PerformanceMetricsConfig["memory_usage"],
		DiskIO:           currentSettings.PerformanceMetricsConfig["disk_io"],
		CacheHitRatio:    currentSettings.PerformanceMetricsConfig["cache_hit_ratio"],
		TransactionCount: currentSettings.PerformanceMetricsConfig["transaction_count"],
		LockWaitTime:     currentSettings.PerformanceMetricsConfig["lock_wait_time"],
	}
	m.collector.UpdateConfig(config)
	
	// Get the driver name
	driverName := db.Driver().(*otelsql.otDriver).DriverName()
	
	// Register the otelsql wrapper
	dsnParser := otelsql.NewDSNParser(nil)
	wrappedDriverName, err := otelsql.Register(driverName,
		otelsql.WithDatabaseName(database),
		otelsql.WithDSN(dsnParser),
		otelsql.WithSQLCommenter(true),
		otelsql.WithAttributes(
			otelsql.DBName(database),
		),
	)
	if err != nil {
		return db, fmt.Errorf("failed to register otelsql driver: %w", err)
	}
	
	// Create a custom driver that records metrics
	monitoredDriver := &monitoredDriver{
		driver:    db.Driver(),
		collector: m.collector,
		database:  database,
	}
	
	sql.Register(fmt.Sprintf("monitored_%s_%s", driverName, database), monitoredDriver)
	
	// Open a new connection with the wrapped driver
	// Note: We need to extract the DSN from the original connection
	// This is a simplified approach - in production, you'd want to properly handle this
	wrappedDB, err := sql.Open(wrappedDriverName, "")
	if err != nil {
		return db, fmt.Errorf("failed to open wrapped database: %w", err)
	}
	
	// Copy connection pool settings
	wrappedDB.SetMaxOpenConns(db.Stats().MaxOpenConnections)
	wrappedDB.SetMaxIdleConns(2)
	wrappedDB.SetConnMaxLifetime(0)
	
	return wrappedDB, nil
}

// GetCollector implements DatabaseMonitor
func (m *databaseMonitor) GetCollector() MetricsCollector {
	return m.collector
}

// monitoredDriver wraps a database driver to collect metrics
type monitoredDriver struct {
	driver    driver.Driver
	collector MetricsCollector
	database  string
}

// Open implements driver.Driver
func (d *monitoredDriver) Open(name string) (driver.Conn, error) {
	conn, err := d.driver.Open(name)
	if err != nil {
		return nil, err
	}
	
	return &monitoredConn{
		Conn:      conn,
		collector: d.collector,
		database:  d.database,
	}, nil
}

// monitoredConn wraps a database connection to collect metrics
type monitoredConn struct {
	driver.Conn
	collector MetricsCollector
	database  string
}

// additional methods would be implemented here to track query execution

// Helper function to determine query type from SQL
func getQueryType(query string) QueryType {
	// Simple query type detection - can be improved
	trimmed := query[:min(10, len(query))]
	switch {
	case len(trimmed) >= 6 && trimmed[:6] == "SELECT":
		return QueryTypeSelect
	case len(trimmed) >= 6 && trimmed[:6] == "INSERT":
		return QueryTypeInsert
	case len(trimmed) >= 6 && trimmed[:6] == "UPDATE":
		return QueryTypeUpdate
	case len(trimmed) >= 6 && trimmed[:6] == "DELETE":
		return QueryTypeDelete
	case len(trimmed) >= 6 && (trimmed[:6] == "CREATE" || trimmed[:5] == "ALTER" || trimmed[:4] == "DROP"):
		return QueryTypeDDL
	default:
		return QueryTypeOther
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}