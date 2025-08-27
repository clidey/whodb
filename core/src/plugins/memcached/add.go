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

package memcached

import (
	"errors"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/plugins"
)

func Add(config *plugins.Config, storageUnit string, values []engine.Record) (bool, error) {
	if storageUnit != "keys" {
		return false, errors.New("can only add entries to 'keys' storage unit in Memcached")
	}
	
	databaseType := engine.DatabaseType(config.Credentials.Type)
	plugin := config.Engine.Choose(databaseType)
	if plugin == nil {
		return false, errors.New("unsupported database type")
	}
	return plugin.AddRow(engine.NewPluginConfig(config.Credentials), "", storageUnit, values)
}