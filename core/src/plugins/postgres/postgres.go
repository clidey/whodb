package postgres

import (
	"github.com/clidey/whodb/core/src/engine"
)

type PostgresPlugin struct{}

func (p *PostgresPlugin) GetSchema(config *engine.PluginConfig) []string {
	return nil
}

func (p *PostgresPlugin) GetStorageUnits(config *engine.PluginConfig, schema string) ([]string, error) {
	return nil, nil
}

func (p *PostgresPlugin) GetRows(config *engine.PluginConfig, schema string, storageUnit string) []string {
	return nil
}

func (p *PostgresPlugin) GetColumns(config *engine.PluginConfig, schema string, storageUnit string, row string) map[string][]string {
	return nil
}

func (p *PostgresPlugin) GetConstraints(config *engine.PluginConfig) map[string]string {
	return nil
}

func (p *PostgresPlugin) RawExecute(config *engine.PluginConfig, sql string) error {
	return nil
}

func NewPostgresPlugin() *engine.Plugin {
	return &engine.Plugin{
		Type:            engine.DatabaseType_Postgres,
		PluginFunctions: &PostgresPlugin{},
	}
}
