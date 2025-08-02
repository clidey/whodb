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

package src

import (
	"fmt"
	"github.com/clidey/whodb/core/src/plugins/clickhouse"
	"github.com/clidey/whodb/core/src/plugins/elasticsearch"
	"github.com/clidey/whodb/core/src/plugins/mongodb"
	"github.com/clidey/whodb/core/src/plugins/mysql"
	"github.com/clidey/whodb/core/src/plugins/redis"
	"github.com/clidey/whodb/core/src/plugins/sqlite3"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/env"
	"github.com/clidey/whodb/core/src/monitoring"
	"github.com/clidey/whodb/core/src/plugins/postgres"
)

var MainEngine *engine.Engine

// InitEEFunc is a function type for initializing Enterprise Edition features
type InitEEFunc func(*engine.Engine)

// initEE is a variable that will be set by the EE build to initialize EE features
var initEE InitEEFunc

// SetEEInitializer allows external packages to register the EE initialization function
func SetEEInitializer(fn InitEEFunc) {
	initEE = fn
}

func InitializeEngine() *engine.Engine {
	MainEngine = &engine.Engine{}
	
	// Register community edition plugins with monitoring wrapper
	MainEngine.RegistryPlugin(monitoring.NewMonitoredPlugin(postgres.NewPostgresPlugin()))
	MainEngine.RegistryPlugin(monitoring.NewMonitoredPlugin(mysql.NewMySQLPlugin()))
	MainEngine.RegistryPlugin(monitoring.NewMonitoredPlugin(mysql.NewMyMariaDBPlugin()))
	MainEngine.RegistryPlugin(monitoring.NewMonitoredPlugin(sqlite3.NewSqlite3Plugin()))
	MainEngine.RegistryPlugin(monitoring.NewMonitoredPlugin(mongodb.NewMongoDBPlugin()))
	MainEngine.RegistryPlugin(monitoring.NewMonitoredPlugin(redis.NewRedisPlugin()))
	MainEngine.RegistryPlugin(monitoring.NewMonitoredPlugin(elasticsearch.NewElasticSearchPlugin()))
	MainEngine.RegistryPlugin(monitoring.NewMonitoredPlugin(clickhouse.NewClickHousePlugin()))
	
	// Initialize Enterprise Edition plugins if available
	if initEE != nil {
		initEE(MainEngine)
	}
	
	return MainEngine
}

func GetLoginProfiles() []env.DatabaseCredentials {
	profiles := []env.DatabaseCredentials{}
	for _, plugin := range MainEngine.Plugins {
		databaseProfiles := env.GetDefaultDatabaseCredentials(string(plugin.Type))
		for _, databaseProfile := range databaseProfiles {
			databaseProfile.Type = string(plugin.Type)
			databaseProfile.IsProfile = true
			profiles = append(profiles, databaseProfile)
		}
	}
	return profiles
}

func GetLoginProfileId(index int, profile env.DatabaseCredentials) string {
	if len(profile.Alias) > 0 {
		return profile.Alias
	}
	return fmt.Sprintf("#%v - %v@%v [%v]", index+1, profile.Username, profile.Hostname, profile.Database)
}

func GetLoginCredentials(profile env.DatabaseCredentials) *engine.Credentials {
	advanced := []engine.Record{
		{
			Key:   "Port",
			Value: profile.Port,
		},
	}

	for key, value := range profile.Config {
		advanced = append(advanced, engine.Record{
			Key:   key,
			Value: value,
		})
	}

	return &engine.Credentials{
		Type:      profile.Type,
		Hostname:  profile.Hostname,
		Username:  profile.Username,
		Password:  profile.Password,
		Database:  profile.Database,
		Advanced:  advanced,
		IsProfile: profile.IsProfile,
	}
}
