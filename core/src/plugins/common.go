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
	"gorm.io/gorm"
)

type DBOperation[T any] func(*gorm.DB) (T, error)
type DBCreationFunc func(pluginConfig *engine.PluginConfig) (*gorm.DB, error)

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
