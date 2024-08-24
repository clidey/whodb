package engine

import "github.com/clidey/whodb/core/graph/model"

type DatabaseType string

const (
	DatabaseType_Postgres      = "Postgres"
	DatabaseType_MySQL         = "MySQL"
	DatabaseType_MariaDB       = "MariaDB"
	DatabaseType_Sqlite3       = "Sqlite3"
	DatabaseType_MongoDB       = "MongoDB"
	DatabaseType_Redis         = "Redis"
	DatabaseType_ElasticSearch = "ElasticSearch"
	DatabaseType_Memcached     = "Memcached"
)

type Engine struct {
	Plugins []*Plugin
}

func (e *Engine) RegistryPlugin(plugin *Plugin) {
	if e.Plugins == nil {
		e.Plugins = []*Plugin{}
	}
	e.Plugins = append(e.Plugins, plugin)
}

func (e *Engine) Choose(databaseType DatabaseType) *Plugin {
	for _, plugin := range e.Plugins {
		if plugin.Type == databaseType {
			return plugin
		}
	}
	return nil
}

func GetStorageUnitModel(unit StorageUnit) *model.StorageUnit {
	attributes := []*model.Record{}
	for _, attribute := range unit.Attributes {
		attributes = append(attributes, &model.Record{
			Key:   attribute.Key,
			Value: attribute.Value,
		})
	}
	return &model.StorageUnit{
		Name:       unit.Name,
		Attributes: attributes,
	}
}
