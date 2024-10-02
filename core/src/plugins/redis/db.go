package redis

import (
	"context"
	"fmt"
	"strconv"

	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/go-redis/redis/v8"
)

func DB(config *engine.PluginConfig) (*redis.Client, error) {
	ctx := context.Background()
	port := common.GetRecordValueOrDefault(config.Credentials.Advanced, "Port", "6379")
	database := 0
	if config.Credentials.Database != "" {
		var err error
		database, err = strconv.Atoi(config.Credentials.Database)
		if err != nil {
			return nil, err
		}
	}
	addr := fmt.Sprintf("%s:%s", config.Credentials.Hostname, port)
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: config.Credentials.Password,
		DB:       database,
	})
	if _, err := client.Ping(ctx).Result(); err != nil {
		return nil, err
	}
	return client, nil
}
