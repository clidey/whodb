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

package graph

import (
	"context"
	"fmt"

	"github.com/clidey/whodb/core/src"
	"github.com/clidey/whodb/core/src/auth"
	"github.com/clidey/whodb/core/src/engine"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct{}

// GetPluginForContext returns the appropriate database plugin and config for the current session.
func GetPluginForContext(ctx context.Context) (*engine.Plugin, *engine.PluginConfig) {
	config := engine.NewPluginConfig(auth.GetCredentials(ctx))
	plugin := src.MainEngine.Choose(engine.DatabaseType(config.Credentials.Type))
	return plugin, config
}

// ValidateStorageUnit checks that a storage unit exists in the given schema.
// This prevents SQL injection by ensuring only existing table names are used.
func ValidateStorageUnit(plugin engine.PluginFunctions, config *engine.PluginConfig, schema string, storageUnit string) error {
	exists, err := plugin.StorageUnitExists(config, schema, storageUnit)
	if err != nil {
		return fmt.Errorf("failed to validate storage unit: %w", err)
	}
	if !exists {
		return fmt.Errorf("storage unit %q not found in schema %q", storageUnit, schema)
	}
	return nil
}
