package memcached

import (
	"github.com/bradfitz/gomemcache/memcache"
	"github.com/clidey/whodb/core/src/engine"
)

func (p *MemcachedPlugin) DeleteRow(config *engine.PluginConfig, schema string, storageUnit string, values map[string]string) (bool, error) {
	client, err := DB(config)
	if err != nil {
		return false, err
	}
	defer client.Close()

	err = client.Delete(storageUnit)
	if err != nil && err != memcache.ErrCacheMiss {
		return false, err
	}

	return true, nil
}
