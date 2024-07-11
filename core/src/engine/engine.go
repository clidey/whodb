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
	DatabaseType_ClickHouse    = "ClickHouse"
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
