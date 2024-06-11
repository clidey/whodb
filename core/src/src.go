package src

import (
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/plugins/postgres"
)

var MainEngine *engine.Engine

func InitializeEngine() *engine.Engine {
	MainEngine = &engine.Engine{}
	MainEngine.RegistryPlugin(postgres.NewPostgresPlugin())
	return MainEngine
}
