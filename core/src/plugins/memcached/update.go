package memcached

import (
	"errors"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/clidey/whodb/core/src/engine"
)

func (p *MemcachedPlugin) UpdateStorageUnit(config *engine.PluginConfig, schema string, storageUnit string, values map[string]string) (bool, error) {
	client, err := DB(config)
	if err != nil {
		return false, err
	}
	defer client.Close()

	if len(values) != 1 {
		return false, errors.New("invalid number of fields for Memcached key")
	}

	value, ok := values["value"]
	if !ok {
		return false, errors.New("missing 'value' for update")
	}

	err = client.Set(&memcache.Item{Key: storageUnit, Value: []byte(value)})
	if err != nil {
		return false, err
	}

	return true, nil
}
