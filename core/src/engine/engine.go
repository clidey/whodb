package engine

type DatabaseType string

const (
	DatabaseType_Postgres = "Postgres"
)

type Engine struct {
	plugins map[DatabaseType]*Plugin
}

func (e *Engine) RegistryPlugin(plugin *Plugin) {
	if e.plugins == nil {
		e.plugins = map[DatabaseType]*Plugin{}
	}
	e.plugins[plugin.Type] = plugin
}

func (e *Engine) Choose(databaseType DatabaseType) *Plugin {
	return e.plugins[databaseType]
}
