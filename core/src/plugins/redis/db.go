package redis

import (
	"context"
	"fmt"

	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/go-redis/redis/v8"
)

func DB(config *engine.PluginConfig) (*redis.Client, error) {
	ctx := context.Background()
	port := common.GetRecordValueOrDefault(config.Credentials.Advanced, "Port", "6379")
	addr := fmt.Sprintf("%s:%s", config.Credentials.Hostname, port)
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: config.Credentials.Password,
		DB:       0,
	})
	if _, err := client.Ping(ctx).Result(); err != nil {
		return nil, err
	}
	return client, nil
}
