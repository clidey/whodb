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

// InsertBatch inserts multiple rows in batches
func (b *BatchProcessor) InsertBatch(db *gorm.DB, schema, tableName string, records []map[string]any) error {
	if len(records) == 0 {
		return nil
	}

	// Build full table name
	var fullTableName string
	if schema != "" && b.dbType != engine.DatabaseType_Sqlite3 {
		fullTableName = schema + "." + tableName
	} else {
		fullTableName = tableName
	}

	// Use GORM's CreateInBatches for efficient bulk insert
	if b.config.UseBulkInsert {
		// For bulk insert, we need to use Table() to specify the table
		result := db.Table(fullTableName).CreateInBatches(records, b.config.BatchSize)
		if result.Error != nil {
			log.Logger.WithError(result.Error).
				WithField("table", fullTableName).
				WithField("recordCount", len(records)).
				Error("Batch insert failed")
			return result.Error
		}

		if b.config.LogProgress {
			log.Logger.WithField("table", fullTableName).
				WithField("recordsInserted", result.RowsAffected).
				Info("Batch insert completed")
		}

		return nil
	}

	// Fall back to individual inserts if bulk insert is disabled
	totalRecords := len(records)
	for i := 0; i < totalRecords; i += b.config.BatchSize {
		end := i + b.config.BatchSize
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
					log.Logger.WithError(err).
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
			log.Logger.WithField("progress", fmt.Sprintf("%d/%d", end, totalRecords)).
				WithField("table", fullTableName).
				Debug("Batch progress")
		}
	}

	return nil
}

// UpdateBatch updates multiple rows in batches
func (b *BatchProcessor) UpdateBatch(db *gorm.DB, schema, tableName string, updates []map[string]any, keyColumn string) error {
	if len(updates) == 0 {
		return nil
	}

	// Build full table name
	var fullTableName string
	if schema != "" && b.dbType != engine.DatabaseType_Sqlite3 {
		fullTableName = schema + "." + tableName
	} else {
		fullTableName = tableName
	}

	totalRecords := len(updates)
	successCount := 0

	// Process in batches
	for i := 0; i < totalRecords; i += b.config.BatchSize {
		end := i + b.config.BatchSize
		if end > totalRecords {
			end = totalRecords
		}

		batch := updates[i:end]

		// Use transaction for batch
		err := db.Transaction(func(tx *gorm.DB) error {
			for _, record := range batch {
				// Extract key value for WHERE clause
				keyValue, exists := record[keyColumn]
				if !exists {
					if b.config.FailOnError {
						return fmt.Errorf("key column %s not found in update record", keyColumn)
					}
					continue
				}

				// Remove key from updates to avoid updating it
				updateData := make(map[string]any)
				for k, v := range record {
					if k != keyColumn {
						updateData[k] = v
					}
				}

				// Perform update
				result := tx.Table(fullTableName).Where(keyColumn+" = ?", keyValue).Updates(updateData)
				if result.Error != nil {
					if b.config.FailOnError {
						return result.Error
					}
					log.Logger.WithError(result.Error).
						WithField("table", fullTableName).
						WithField(keyColumn, keyValue).
						Warn("Failed to update record, continuing")
				} else {
					successCount += int(result.RowsAffected)
				}
			}
			return nil
		})

		if err != nil && b.config.FailOnError {
			return fmt.Errorf("batch update failed at batch %d: %w", i/b.config.BatchSize, err)
		}

		if b.config.LogProgress {
			log.Logger.WithField("progress", fmt.Sprintf("%d/%d", end, totalRecords)).
				WithField("table", fullTableName).
				Debug("Update batch progress")
		}
	}

	if b.config.LogProgress {
		log.Logger.WithField("table", fullTableName).
			WithField("recordsUpdated", successCount).
			Info("Batch update completed")
	}

	return nil
}

// DeleteBatch deletes multiple rows in batches based on conditions
func (b *BatchProcessor) DeleteBatch(db *gorm.DB, schema, tableName string, conditions []map[string]any) error {
	if len(conditions) == 0 {
		return nil
	}

	// Build full table name
	var fullTableName string
	if schema != "" && b.dbType != engine.DatabaseType_Sqlite3 {
		fullTableName = schema + "." + tableName
	} else {
		fullTableName = tableName
	}

	totalConditions := len(conditions)
	deletedCount := int64(0)

	// Process in batches
	for i := 0; i < totalConditions; i += b.config.BatchSize {
		end := i + b.config.BatchSize
		if end > totalConditions {
			end = totalConditions
		}

		batch := conditions[i:end]

		// Use transaction for batch
		err := db.Transaction(func(tx *gorm.DB) error {
			for _, condition := range batch {
				result := tx.Table(fullTableName).Where(condition).Delete(nil)
				if result.Error != nil {
					if b.config.FailOnError {
						return result.Error
					}
					log.Logger.WithError(result.Error).
						WithField("table", fullTableName).
						Warn("Failed to delete records, continuing")
				} else {
					deletedCount += result.RowsAffected
				}
			}
			return nil
		})

		if err != nil && b.config.FailOnError {
			return fmt.Errorf("batch delete failed at batch %d: %w", i/b.config.BatchSize, err)
		}

		if b.config.LogProgress {
			log.Logger.WithField("progress", fmt.Sprintf("%d/%d", end, totalConditions)).
				WithField("table", fullTableName).
				Debug("Delete batch progress")
		}
	}

	if b.config.LogProgress {
		log.Logger.WithField("table", fullTableName).
			WithField("recordsDeleted", deletedCount).
			Info("Batch delete completed")
	}

	return nil
}

// ExportInBatches exports data in batches to avoid memory issues
func (b *BatchProcessor) ExportInBatches(db *gorm.DB, schema, tableName string, columns []string, writer func([]map[string]any) error) error {
	// Build full table name
	var fullTableName string
	if schema != "" && b.dbType != engine.DatabaseType_Sqlite3 {
		fullTableName = schema + "." + tableName
	} else {
		fullTableName = tableName
	}

	// Build query
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
			log.Logger.WithField("recordsExported", totalExported).
				WithField("table", fullTableName).
				Debug("Export progress")
		}

		// If we got less than BatchSize, we're done
		if len(batch) < b.config.BatchSize {
			break
		}
	}

	if b.config.LogProgress {
		log.Logger.WithField("table", fullTableName).
			WithField("totalRecordsExported", totalExported).
			Info("Export completed")
	}

	return nil
}

// ProcessInBatches processes records in batches with a custom function
func (b *BatchProcessor) ProcessInBatches(db *gorm.DB, schema, tableName string, processor func(*gorm.DB, []map[string]any) error) error {
	// Build full table name
	var fullTableName string
	if schema != "" && b.dbType != engine.DatabaseType_Sqlite3 {
		fullTableName = schema + "." + tableName
	} else {
		fullTableName = tableName
	}

	offset := 0
	totalProcessed := 0

	for {
		var batch []map[string]any

		// Fetch batch
		result := db.Table(fullTableName).Limit(b.config.BatchSize).Offset(offset).Find(&batch)
		if result.Error != nil {
			return fmt.Errorf("failed to fetch batch at offset %d: %w", offset, result.Error)
		}

		// No more records
		if len(batch) == 0 {
			break
		}

		// Process batch
		if err := processor(db, batch); err != nil {
			if b.config.FailOnError {
				return fmt.Errorf("failed to process batch at offset %d: %w", offset, err)
			}
			log.Logger.WithError(err).
				WithField("offset", offset).
				Warn("Failed to process batch, continuing")
		}

		totalProcessed += len(batch)
		offset += b.config.BatchSize

		if b.config.LogProgress {
			log.Logger.WithField("recordsProcessed", totalProcessed).
				WithField("table", fullTableName).
				Debug("Process progress")
		}

		// If we got less than BatchSize, we're done
		if len(batch) < b.config.BatchSize {
			break
		}
	}

	if b.config.LogProgress {
		log.Logger.WithField("table", fullTableName).
			WithField("totalRecordsProcessed", totalProcessed).
			Info("Processing completed")
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

		// Use batch processor for bulk insert
		processor := NewBatchProcessor(p, p.Type, &BatchConfig{
			BatchSize:     1000,
			UseBulkInsert: true,
			FailOnError:   true,
			LogProgress:   len(records) > 10000, // Log progress for large datasets
		})

		err := processor.InsertBatch(db, schema, storageUnit, records)
		return err == nil, err
	})
}
