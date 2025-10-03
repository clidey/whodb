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

// Sort direction constants
type SortDirection int

const (
	Up   SortDirection = iota // ASC
	Down                      // DESC
)

// Sort represents a sort condition
type Sort struct {
	Column    string
	Direction SortDirection
}

// SearchCondition types for WHERE clauses
type SearchCondition struct {
	And    *AndCondition
	Or     *OrCondition
	Atomic *AtomicCondition
}

type AndCondition struct {
	Conditions []SearchCondition
}

type OrCondition struct {
	Conditions []SearchCondition
}

type AtomicCondition struct {
	Key        string
	Operator   string
	Value      any
	ColumnType string
}

// AtomicOperator represents comparison operators
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

type DBOperation[T any] func(*gorm.DB) (T, error)
type DBCreationFunc func(pluginConfig *engine.PluginConfig) (*gorm.DB, error)

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

// WithConnection handles database connection lifecycle and executes the operation
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
