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

package engine

import (
	"strings"

	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/src/types"
)

// DatabaseType identifies a supported database system.
type DatabaseType string

const (
	DatabaseType_Postgres      = "Postgres"
	DatabaseType_MySQL         = "MySQL"
	DatabaseType_MariaDB       = "MariaDB"
	DatabaseType_Sqlite3       = "Sqlite3"
	DatabaseType_MongoDB       = "MongoDB"
	DatabaseType_Redis         = "Redis"
	DatabaseType_ElasticSearch = "ElasticSearch"
	DatabaseType_ClickHouse    = "ClickHouse"
)

// LoginProfileRetriever is a function that retrieves stored database credentials.
type LoginProfileRetriever func() ([]types.DatabaseCredentials, error)

// Engine manages database plugins and login profiles.
type Engine struct {
	Plugins           []*Plugin
	LoginProfiles     []types.DatabaseCredentials
	ProfileRetrievers []LoginProfileRetriever
}

// RegistryPlugin adds a database plugin to the engine.
func (e *Engine) RegistryPlugin(plugin *Plugin) {
	if e.Plugins == nil {
		e.Plugins = []*Plugin{}
	}
	e.Plugins = append(e.Plugins, plugin)
}

// Choose returns the plugin that matches the given database type, or nil if not found.
func (e *Engine) Choose(databaseType DatabaseType) *Plugin {
	for _, plugin := range e.Plugins {
		if strings.EqualFold(string(plugin.Type), string(databaseType)) {
			return plugin
		}
	}
	return nil
}

// AddLoginProfile adds database credentials to the engine's profile list.
func (e *Engine) AddLoginProfile(profile types.DatabaseCredentials) {
	if e.LoginProfiles == nil {
		e.LoginProfiles = []types.DatabaseCredentials{}
	}
	e.LoginProfiles = append(e.LoginProfiles, profile)
}

// RegisterProfileRetriever adds a function that retrieves database credentials on demand.
func (e *Engine) RegisterProfileRetriever(retriever LoginProfileRetriever) {
	if e.ProfileRetrievers == nil {
		e.ProfileRetrievers = []LoginProfileRetriever{}
	}
	e.ProfileRetrievers = append(e.ProfileRetrievers, retriever)
}

// GetStorageUnitModel converts an engine StorageUnit to a GraphQL model StorageUnit.
func GetStorageUnitModel(unit StorageUnit) *model.StorageUnit {
	attributes := []*model.Record{}
	for _, attribute := range unit.Attributes {
		attributes = append(attributes, &model.Record{
			Key:   attribute.Key,
			Value: attribute.Value,
		})
	}
	return &model.StorageUnit{
		Name:                        unit.Name,
		Attributes:                  attributes,
		IsMockDataGenerationAllowed: false, // Will be set in resolver
	}
}
