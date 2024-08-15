package redis

import (
	"context"
	"errors"
	"fmt"
	"github.com/clidey/whodb/core/src/engine"
	"strconv"
)

func (p *RedisPlugin) DeleteRow(config *engine.PluginConfig, schema string, storageUnit string, values map[string]string) (bool, error) {
	client, err := DB(config)
	if err != nil {
		return false, err
	}
	defer client.Close()

	ctx := context.Background()

	keyType, err := client.Type(ctx, storageUnit).Result()
	if err != nil {
		return false, err
	}

	switch keyType {
	case "string":
		// Deleting the entire string key
		err := client.Del(ctx, storageUnit).Err()
		if err != nil {
			return false, err
		}
	case "hash":
		// Deleting a specific field from a hash
		field, ok := values["field"]
		if !ok {
			return false, errors.New("missing 'field' for hash deletion")
		}
		err := client.HDel(ctx, storageUnit, field).Err()
		if err != nil {
			return false, err
		}
	case "list":
		// Removing an element from a list
		indexStr, ok := values["index"]
		if !ok {
			return false, errors.New("missing 'index' for list deletion")
		}
		index, err := strconv.ParseInt(indexStr, 10, 64)
		if err != nil {
			return false, errors.New("unable to convert index to int")
		}
		value := client.LIndex(ctx, storageUnit, index).Val()
		if err := client.LRem(ctx, storageUnit, 1, value).Err(); err != nil {
			return false, errors.New("unable to remove the list item")
		}
	case "set":
		// Removing a specific member from a set
		member, ok := values["member"]
		if !ok {
			return false, errors.New("missing 'member' for set deletion")
		}
		err := client.SRem(ctx, storageUnit, member).Err()
		if err != nil {
			return false, err
		}
	case "zset":
		// Removing a specific member from a sorted set
		member, ok := values["member"]
		if !ok {
			return false, errors.New("missing 'member' for sorted set deletion")
		}
		err := client.ZRem(ctx, storageUnit, member).Err()
		if err != nil {
			return false, err
		}
	default:
		return false, fmt.Errorf("unsupported Redis data type: %s", keyType)
	}

	return true, nil
}
