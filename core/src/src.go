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
	"github.com/clidey/whodb/core/src/plugins/postgres"
)

var MainEngine *engine.Engine

func InitializeEngine() *engine.Engine {
	MainEngine = &engine.Engine{}
	MainEngine.RegistryPlugin(postgres.NewPostgresPlugin())
	MainEngine.RegistryPlugin(mysql.NewMySQLPlugin())
	MainEngine.RegistryPlugin(mysql.NewMyMariaDBPlugin())
	MainEngine.RegistryPlugin(sqlite3.NewSqlite3Plugin())
	MainEngine.RegistryPlugin(mongodb.NewMongoDBPlugin())
	MainEngine.RegistryPlugin(redis.NewRedisPlugin())
	MainEngine.RegistryPlugin(elasticsearch.NewElasticSearchPlugin())
	MainEngine.RegistryPlugin(clickhouse.NewClickHousePlugin())
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
