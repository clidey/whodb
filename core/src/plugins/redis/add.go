package redis

import (
	"errors"

	"github.com/clidey/whodb/core/src/engine"
)

func (p *RedisPlugin) AddStorageUnit(config *engine.PluginConfig, schema string, storageUnit string, fields map[string]string) (bool, error) {
	return false, errors.ErrUnsupported
}

func (p *RedisPlugin) AddRow(config *engine.PluginConfig, schema string, storageUnit string, values []engine.Record) (bool, error) {
	return false, errors.ErrUnsupported
}
