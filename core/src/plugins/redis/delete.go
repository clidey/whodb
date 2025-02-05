// Licensed to Clidey Limited under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Clidey Limited licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

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
