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

package gorm_plugin

import (
	"fmt"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
	"github.com/clidey/whodb/core/src/plugins"
	"gorm.io/gorm"
)

// BatchConfig holds configuration for batch operations
type BatchConfig struct {
	BatchSize     int  // Number of records per batch
	UseBulkInsert bool // Use database-specific bulk insert when available
	FailOnError   bool // Stop on first error or continue
	LogProgress   bool // Log progress during batch operations
}

// DefaultBatchConfig returns default batch configuration
func DefaultBatchConfig() *BatchConfig {
	return &BatchConfig{
		BatchSize:     1000,
		UseBulkInsert: true,
		FailOnError:   true,
		LogProgress:   false,
	}
}

// BatchProcessor handles batch operations for the plugin
type BatchProcessor struct {
	plugin GormPluginFunctions
	config *BatchConfig
	dbType engine.DatabaseType
}

// NewBatchProcessor creates a new batch processor
func NewBatchProcessor(plugin GormPluginFunctions, dbType engine.DatabaseType, config *BatchConfig) *BatchProcessor {
	if config == nil {
		config = DefaultBatchConfig()
	}
	return &BatchProcessor{
		plugin: plugin,
		config: config,
		dbType: dbType,
	}
}

// getMaxParametersForDB returns the maximum number of parameters supported by the database.
func (b *BatchProcessor) getMaxParametersForDB() int {
	return b.plugin.GetMaxBulkInsertParameters()
}

// calculateBatchSize returns an appropriate batch size based on column count and DB limits.
func (b *BatchProcessor) calculateBatchSize(columnCount int) int {
	if columnCount == 0 {
		return b.config.BatchSize
	}

	maxParams := b.getMaxParametersForDB()
	// Leave some margin (10%) for safety
	safeMaxParams := int(float64(maxParams) * 0.9)
	maxRowsForDB := safeMaxParams / columnCount

	if maxRowsForDB < 1 {
		maxRowsForDB = 1
	}

	// Use the smaller of configured batch size and calculated max
	if maxRowsForDB < b.config.BatchSize {
		log.WithFields(map[string]any{
			"configuredBatchSize": b.config.BatchSize,
			"calculatedMaxRows":   maxRowsForDB,
			"columnCount":         columnCount,
			"maxParameters":       maxParams,
			"dbType":              b.dbType,
		}).Debug("Reducing batch size due to database parameter limit")
		return maxRowsForDB
	}

	return b.config.BatchSize
}

// InsertBatch inserts multiple rows in batches
func (b *BatchProcessor) InsertBatch(db *gorm.DB, schema, tableName string, records []map[string]any) error {
	if len(records) == 0 {
		return nil
	}

	columnCount := 0
	if len(records) > 0 {
		columnCount = len(records[0])
	}

	if columnCount == 0 {
		log.WithFields(map[string]any{
			"table":       tableName,
			"recordCount": len(records),
		}).Error("All records are empty - no columns to insert")
		return fmt.Errorf("cannot insert empty records into %s: all columns were skipped (check for incorrect auto-increment/computed detection)", tableName)
	}

	effectiveBatchSize := b.calculateBatchSize(columnCount)

	var fullTableName string
	if schema != "" && b.dbType != engine.DatabaseType_Sqlite3 {
		fullTableName = schema + "." + tableName
	} else {
		fullTableName = tableName
	}

	// Use GORM's CreateInBatches for efficient bulk insert
	if b.config.UseBulkInsert {
		result := db.Table(fullTableName).CreateInBatches(records, effectiveBatchSize)
		if result.Error != nil {
			log.WithError(result.Error).
				WithField("table", fullTableName).
				WithField("recordCount", len(records)).
				Error("Batch insert failed")
			return result.Error
		}

		if b.config.LogProgress {
			log.WithField("table", fullTableName).
				WithField("recordsInserted", result.RowsAffected).
				Info("Batch insert completed")
		}

		return nil
	}

	// Fall back to individual inserts if bulk insert is disabled
	totalRecords := len(records)
	for i := 0; i < totalRecords; i += effectiveBatchSize {
		end := i + effectiveBatchSize
		if end > totalRecords {
			end = totalRecords
		}

		batch := records[i:end]

		// Process batch
		err := db.Transaction(func(tx *gorm.DB) error {
			for _, record := range batch {
				if err := tx.Table(fullTableName).Create(record).Error; err != nil {
					if b.config.FailOnError {
						return err
					}
					// Log error but continue
					log.WithError(err).
						WithField("table", fullTableName).
						Warn("Failed to insert record, continuing")
				}
			}
			return nil
		})

		if err != nil && b.config.FailOnError {
			return fmt.Errorf("batch insert failed at batch %d: %w", i/b.config.BatchSize, err)
		}

		if b.config.LogProgress {
			log.WithField("progress", fmt.Sprintf("%d/%d", end, totalRecords)).
				WithField("table", fullTableName).
				Debug("Batch progress")
		}
	}

	return nil
}

// ExportInBatches exports data in batches to avoid memory issues
func (b *BatchProcessor) ExportInBatches(db *gorm.DB, schema, tableName string, columns []string, writer func([]map[string]any) error) error {
	var fullTableName string
	if schema != "" && b.dbType != engine.DatabaseType_Sqlite3 {
		fullTableName = schema + "." + tableName
	} else {
		fullTableName = tableName
	}

	query := db.Table(fullTableName)
	if len(columns) > 0 {
		query = query.Select(columns)
	}

	offset := 0
	totalExported := 0

	for {
		var batch []map[string]any

		// Fetch batch
		result := query.Limit(b.config.BatchSize).Offset(offset).Find(&batch)
		if result.Error != nil {
			return fmt.Errorf("failed to fetch batch at offset %d: %w", offset, result.Error)
		}

		// No more records
		if len(batch) == 0 {
			break
		}

		// Write batch
		if err := writer(batch); err != nil {
			return fmt.Errorf("failed to write batch at offset %d: %w", offset, err)
		}

		totalExported += len(batch)
		offset += b.config.BatchSize

		if b.config.LogProgress {
			log.WithField("recordsExported", totalExported).
				WithField("table", fullTableName).
				Debug("Export progress")
		}

		// If we got less than BatchSize, we're done
		if len(batch) < b.config.BatchSize {
			break
		}
	}

	if b.config.LogProgress {
		log.WithField("table", fullTableName).
			WithField("totalRecordsExported", totalExported).
			Info("Export completed")
	}

	return nil
}

// BulkAddRows adds multiple rows using batch processing
func (p *GormPlugin) BulkAddRows(config *engine.PluginConfig, schema string, storageUnit string, rows [][]engine.Record) (bool, error) {
	return plugins.WithConnection(config, p.DB, func(db *gorm.DB) (bool, error) {
		// Convert Records to maps
		var records []map[string]any
		for _, row := range rows {
			record, err := p.ConvertRecordValuesToMap(row)
			if err != nil {
				return false, fmt.Errorf("failed to convert row: %w", err)
			}
			records = append(records, record)
		}

		processor := NewBatchProcessor(p.GormPluginFunctions, p.Type, &BatchConfig{
			BatchSize:     1000,
			UseBulkInsert: true,
			FailOnError:   true,
			LogProgress:   len(records) > 10000, // Log progress for large datasets
		})

		err := processor.InsertBatch(db, schema, storageUnit, records)
		return err == nil, err
	})
}
