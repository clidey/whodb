// Copyright 2025 Clidey, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package redis

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
)

func (p *RedisPlugin) DeleteRow(config *engine.PluginConfig, schema string, storageUnit string, values map[string]string) (bool, error) {
	client, err := DB(config)
	if err != nil {
		log.Logger.WithError(err).WithField("storageUnit", storageUnit).Error("Failed to connect to Redis for row deletion")
		return false, err
	}
	defer client.Close()

	ctx := context.Background()

	keyType, err := client.Type(ctx, storageUnit).Result()
	if err != nil {
		log.Logger.WithError(err).WithField("storageUnit", storageUnit).Error("Failed to get Redis key type for row deletion")
		return false, err
	}

	switch keyType {
	case "string":
		// Deleting the entire string key
		err := client.Del(ctx, storageUnit).Err()
		if err != nil {
			log.Logger.WithError(err).WithField("storageUnit", storageUnit).Error("Failed to delete Redis string key")
			return false, err
		}
	case "hash":
		// Deleting a specific field from a hash
		field, ok := values["field"]
		if !ok {
			err := errors.New("missing 'field' for hash deletion")
			log.Logger.WithError(err).WithField("storageUnit", storageUnit).Error("Missing field parameter for Redis hash deletion")
			return false, err
		}
		err := client.HDel(ctx, storageUnit, field).Err()
		if err != nil {
			log.Logger.WithError(err).WithField("storageUnit", storageUnit).WithField("field", field).Error("Failed to delete Redis hash field")
			return false, err
		}
	case "list":
		// Removing an element from a list
		indexStr, ok := values["index"]
		if !ok {
			err := errors.New("missing 'index' for list deletion")
			log.Logger.WithError(err).WithField("storageUnit", storageUnit).Error("Missing index parameter for Redis list deletion")
			return false, err
		}
		index, err := strconv.ParseInt(indexStr, 10, 64)
		if err != nil {
			log.Logger.WithError(err).WithField("storageUnit", storageUnit).WithField("index", indexStr).Error("Failed to parse list index for Redis deletion")
			return false, errors.New("unable to convert index to int")
		}
		value := client.LIndex(ctx, storageUnit, index).Val()
		if err := client.LRem(ctx, storageUnit, 1, value).Err(); err != nil {
			log.Logger.WithError(err).WithField("storageUnit", storageUnit).WithField("index", index).WithField("value", value).Error("Failed to remove Redis list item")
			return false, errors.New("unable to remove the list item")
		}
	case "set":
		// Removing a specific member from a set
		member, ok := values["member"]
		if !ok {
			err := errors.New("missing 'member' for set deletion")
			log.Logger.WithError(err).WithField("storageUnit", storageUnit).Error("Missing member parameter for Redis set deletion")
			return false, err
		}
		err := client.SRem(ctx, storageUnit, member).Err()
		if err != nil {
			log.Logger.WithError(err).WithField("storageUnit", storageUnit).WithField("member", member).Error("Failed to remove Redis set member")
			return false, err
		}
	case "zset":
		// Removing a specific member from a sorted set
		member, ok := values["member"]
		if !ok {
			err := errors.New("missing 'member' for sorted set deletion")
			log.Logger.WithError(err).WithField("storageUnit", storageUnit).Error("Missing member parameter for Redis sorted set deletion")
			return false, err
		}
		err := client.ZRem(ctx, storageUnit, member).Err()
		if err != nil {
			log.Logger.WithError(err).WithField("storageUnit", storageUnit).WithField("member", member).Error("Failed to remove Redis sorted set member")
			return false, err
		}
	default:
		err := fmt.Errorf("unsupported Redis data type: %s", keyType)
		return false, err
	}

	return true, nil
}
