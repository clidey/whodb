package src

import (
	"fmt"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/env"
	keeper_integration "github.com/clidey/whodb/core/src/integrations/keeper"
	"github.com/clidey/whodb/core/src/log"
	"github.com/clidey/whodb/core/src/plugins/clickhouse"
	"github.com/clidey/whodb/core/src/plugins/elasticsearch"
	"github.com/clidey/whodb/core/src/plugins/mongodb"
	"github.com/clidey/whodb/core/src/plugins/mysql"
	"github.com/clidey/whodb/core/src/plugins/postgres"
	"github.com/clidey/whodb/core/src/plugins/redis"
	"github.com/clidey/whodb/core/src/plugins/sqlite3"
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

var profiles []env.DatabaseCredentials

func GetLoginProfiles() []env.DatabaseCredentials {
	if profiles != nil {
		return profiles
	}

	keeperLoginProfiles, err := keeper_integration.GetLoginProfiles()
	if err != nil {
		log.Logger.Warn("keeper integration failed with: ", err)
		keeperLoginProfiles = []env.DatabaseCredentials{}
	}

	profiles = append(profiles, keeperLoginProfiles...)

	for _, plugin := range MainEngine.Plugins {
		databaseProfiles := env.GetDefaultDatabaseCredentials(string(plugin.Type))
		for _, databaseProfile := range databaseProfiles {
			databaseProfile.Type = string(plugin.Type)
			profiles = append(profiles, databaseProfile)
		}
	}

	return profiles
}

func GetLoginProfileId(index int, profile env.DatabaseCredentials) string {
	if len(profile.CustomId) > 0 {
		return profile.CustomId
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
		Type:     profile.Type,
		Hostname: profile.Hostname,
		Username: profile.Username,
		Password: profile.Password,
		Database: profile.Database,
		Advanced: advanced,
	}
}
