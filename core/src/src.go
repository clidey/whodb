package src

import (
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/plugins/mongodb"
	"github.com/clidey/whodb/core/src/plugins/mysql"
	"github.com/clidey/whodb/core/src/plugins/postgres"
	"github.com/clidey/whodb/core/src/plugins/sqlite3"
)

var MainEngine *engine.Engine

func InitializeEngine() *engine.Engine {
	MainEngine = &engine.Engine{}
	MainEngine.RegistryPlugin(postgres.NewPostgresPlugin())
	MainEngine.RegistryPlugin(mysql.NewMySQLPlugin())
	MainEngine.RegistryPlugin(sqlite3.NewSqlite3Plugin())
	MainEngine.RegistryPlugin(mongodb.NewMongoDBPlugin())
	return MainEngine
}
