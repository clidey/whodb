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
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Query Scopes - Reusable query patterns for GORM
// These can be used with db.Scopes() to chain common query patterns
//
// NOTE: These helper functions are currently unused but are kept for future use.
// They provide clean, reusable patterns for common database operations.
//
// Example usage:
//   db.Scopes(Paginate(1, 10), OrderBy("created_at", true)).Find(&results)
//   db.Scopes(WhereMap(conditions), SelectColumns(cols)).Find(&results)

// Paginate creates a scope for pagination
func Paginate(page, pageSize int) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if page <= 0 {
			page = 1
		}
		if pageSize <= 0 {
			pageSize = 10
		}
		offset := (page - 1) * pageSize
		return db.Offset(offset).Limit(pageSize)
	}
}

// OrderBy creates a scope for dynamic ordering
func OrderBy(column string, desc bool) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Order(clause.OrderByColumn{
			Column: clause.Column{Name: column},
			Desc:   desc,
		})
	}
}

// SelectColumns creates a scope for selecting specific columns
func SelectColumns(columns []string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if len(columns) > 0 {
			return db.Select(columns)
		}
		return db
	}
}

// WhereMap creates a scope for WHERE conditions from a map
func WhereMap(conditions map[string]any) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if len(conditions) > 0 {
			return db.Where(conditions)
		}
		return db
	}
}

// WithLimit creates a scope that only applies limit if > 0
func WithLimit(limit int) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if limit > 0 {
			return db.Limit(limit)
		}
		return db
	}
}

// InTransaction wraps operations in a transaction with automatic rollback on error
func InTransaction(db *gorm.DB, fn func(*gorm.DB) error) error {
	return db.Transaction(func(tx *gorm.DB) error {
		return fn(tx)
	}, nil)
}

// SafeTransaction wraps operations with panic recovery
func SafeTransaction(db *gorm.DB, fn func(*gorm.DB) error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = db.Error
			if err == nil {
				if e, ok := r.(error); ok {
					err = e
				}
			}
		}
	}()

	return db.Transaction(func(tx *gorm.DB) error {
		return fn(tx)
	}, nil)
}

// BatchProcess processes records in batches to avoid memory issues
func BatchProcess(db *gorm.DB, batchSize int, process func(tx *gorm.DB, batch []map[string]any) error) error {
	var results []map[string]any
	offset := 0

	for {
		// Clear previous results
		results = results[:0]

		// Fetch next batch
		err := db.Limit(batchSize).Offset(offset).Find(&results).Error
		if err != nil {
			return err
		}

		// No more records
		if len(results) == 0 {
			break
		}

		// Process batch
		if err := process(db, results); err != nil {
			return err
		}

		// Move to next batch
		offset += batchSize

		// If we got less than batchSize, we're done
		if len(results) < batchSize {
			break
		}
	}

	return nil
}

// CombineScopes combines multiple scopes into one
func CombineScopes(scopes ...func(*gorm.DB) *gorm.DB) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		for _, scope := range scopes {
			db = scope(db)
		}
		return db
	}
}
