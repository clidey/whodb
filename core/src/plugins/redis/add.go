/*
 * Copyright 2026 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package redis

import (
	"context"
	"fmt"
	"strconv"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
	"github.com/go-redis/redis/v8"
)

// AddStorageUnit creates a new Redis key with initial data.
// The storageUnit name becomes the Redis key. The first field's type (from Extra["type"])
// determines the Redis data structure: string, hash, list, set, or zset.
// If no type is specified, defaults to "string".
func (p *RedisPlugin) AddStorageUnit(config *engine.PluginConfig, schema string, storageUnit string, fields []engine.Record) (bool, error) {
	client, err := DB(config)
	if err != nil {
		return false, err
	}
	defer client.Close()

	ctx := context.Background()

	// Determine key type from the first field's Extra metadata
	keyType := "string"
	if len(fields) > 0 {
		if t, ok := fields[0].Extra["type"]; ok && t != "" {
			keyType = t
		}
	}

	switch keyType {
	case "string":
		value := ""
		if len(fields) > 0 {
			value = fields[0].Value
		}
		if err := client.Set(ctx, storageUnit, value, 0).Err(); err != nil {
			log.WithError(err).WithField("key", storageUnit).Error("Failed to create Redis string key")
			return false, err
		}
	case "hash":
		fieldValues := map[string]any{}
		for _, f := range fields {
			fieldValues[f.Key] = f.Value
		}
		if len(fieldValues) == 0 {
			fieldValues["default"] = ""
		}
		if err := client.HSet(ctx, storageUnit, fieldValues).Err(); err != nil {
			log.WithError(err).WithField("key", storageUnit).Error("Failed to create Redis hash key")
			return false, err
		}
	case "list":
		values := make([]any, 0, len(fields))
		for _, f := range fields {
			values = append(values, f.Value)
		}
		if len(values) == 0 {
			values = append(values, "")
		}
		if err := client.RPush(ctx, storageUnit, values...).Err(); err != nil {
			log.WithError(err).WithField("key", storageUnit).Error("Failed to create Redis list key")
			return false, err
		}
	case "set":
		members := make([]any, 0, len(fields))
		for _, f := range fields {
			members = append(members, f.Value)
		}
		if len(members) == 0 {
			members = append(members, "")
		}
		if err := client.SAdd(ctx, storageUnit, members...).Err(); err != nil {
			log.WithError(err).WithField("key", storageUnit).Error("Failed to create Redis set key")
			return false, err
		}
	case "zset":
		// Fields should have Key=member, Value=score
		for _, f := range fields {
			score, parseErr := strconv.ParseFloat(f.Value, 64)
			if parseErr != nil {
				score = 0
			}
			if err := client.ZAdd(ctx, storageUnit, &redis.Z{Score: score, Member: f.Key}).Err(); err != nil {
				log.WithError(err).WithField("key", storageUnit).Error("Failed to create Redis sorted set key")
				return false, err
			}
		}
	default:
		return false, fmt.Errorf("unsupported Redis key type: %s", keyType)
	}

	return true, nil
}

// AddRow adds data to an existing Redis key based on its type.
// The values map column names (from GetColumnsForTable) to data:
//   - string: values[0].Value is the new value
//   - hash: Key=field name, Value=field value
//   - list: Value=element to append
//   - set: Value=member to add
//   - zset: Key=member, Value=score
func (p *RedisPlugin) AddRow(config *engine.PluginConfig, schema string, storageUnit string, values []engine.Record) (bool, error) {
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

	// Build a lookup from the values
	valuesMap := map[string]string{}
	for _, v := range values {
		valuesMap[v.Key] = v.Value
	}

	switch keyType {
	case "string":
		value := valuesMap["value"]
		if err := client.Set(ctx, storageUnit, value, 0).Err(); err != nil {
			return false, err
		}
	case "hash":
		field := valuesMap["field"]
		value := valuesMap["value"]
		if field == "" {
			return false, fmt.Errorf("field name is required for hash")
		}
		if err := client.HSet(ctx, storageUnit, field, value).Err(); err != nil {
			return false, err
		}
	case "list":
		value := valuesMap["value"]
		if err := client.RPush(ctx, storageUnit, value).Err(); err != nil {
			return false, err
		}
	case "set":
		value := valuesMap["value"]
		if err := client.SAdd(ctx, storageUnit, value).Err(); err != nil {
			return false, err
		}
	case "zset":
		member := valuesMap["member"]
		scoreStr := valuesMap["score"]
		score, parseErr := strconv.ParseFloat(scoreStr, 64)
		if parseErr != nil {
			score = 0
		}
		if err := client.ZAdd(ctx, storageUnit, &redis.Z{Score: score, Member: member}).Err(); err != nil {
			return false, err
		}
	default:
		return false, fmt.Errorf("unsupported Redis data type: %s", keyType)
	}

	return true, nil
}
