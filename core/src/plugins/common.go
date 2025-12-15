// Copyright 2025 Clidey, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package plugins

import (
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/env"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// SortDirection indicates ascending or descending sort order.
type SortDirection int

const (
	Up   SortDirection = iota // ASC
	Down                      // DESC
)

// Sort represents a column sort condition with direction.
type Sort struct {
	Column    string
	Direction SortDirection
}

// SearchCondition represents a WHERE clause condition that can be atomic, AND, or OR.
type SearchCondition struct {
	And    *AndCondition
	Or     *OrCondition
	Atomic *AtomicCondition
}

// AndCondition represents multiple conditions joined with AND.
type AndCondition struct {
	Conditions []SearchCondition
}

// OrCondition represents multiple conditions joined with OR.
type OrCondition struct {
	Conditions []SearchCondition
}

// AtomicCondition represents a single comparison condition (e.g., column = value).
type AtomicCondition struct {
	Key        string
	Operator   string
	Value      any
	ColumnType string
}

// AtomicOperator represents SQL comparison operators.
type AtomicOperator string

const (
	Equal              AtomicOperator = "="
	NotEqual           AtomicOperator = "!="
	GreaterThan        AtomicOperator = ">"
	GreaterThanOrEqual AtomicOperator = ">="
	LessThan           AtomicOperator = "<"
	LessThanOrEqual    AtomicOperator = "<="
	Like               AtomicOperator = "LIKE"
	NotLike            AtomicOperator = "NOT LIKE"
	In                 AtomicOperator = "IN"
	NotIn              AtomicOperator = "NOT IN"
	IsNull             AtomicOperator = "IS NULL"
	IsNotNull          AtomicOperator = "IS NOT NULL"
)

// DBOperation is a function that performs database operations with a GORM connection.
type DBOperation[T any] func(*gorm.DB) (T, error)

// DBCreationFunc is a function that creates a new GORM database connection.
type DBCreationFunc func(pluginConfig *engine.PluginConfig) (*gorm.DB, error)

// GetGormLogConfig returns the GORM logger level based on the environment log level setting.
func GetGormLogConfig() logger.LogLevel {
	switch env.LogLevel {
	case "warning":
		return logger.Warn
	case "error":
		return logger.Error
	default:
		return logger.Silent
	}
}

// WithConnection manages the database connection lifecycle for an operation.
// It creates a connection, executes the operation, and ensures the connection is closed.
func WithConnection[T any](config *engine.PluginConfig, DB DBCreationFunc, operation DBOperation[T]) (T, error) {
	db, err := DB(config)
	if err != nil {
		var zero T
		return zero, err
	}
	sqlDb, err := db.DB()
	if err != nil {
		var zero T
		return zero, err
	}
	defer sqlDb.Close()
	return operation(db)
}
