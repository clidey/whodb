package memcached

import (
	"fmt"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
)

func DB(config *engine.PluginConfig) (*memcache.Client, error) {
	client := memcache.New(getAddress(config))
	if client == nil {
		return nil, fmt.Errorf("failed to create Memcached client")
	}
	return client, nil
}

func getAddress(config *engine.PluginConfig) string {
	port := common.GetRecordValueOrDefault(config.Credentials.Advanced, "Port", "11211")
	return fmt.Sprintf("%s:%s", config.Credentials.Hostname, port)
}
