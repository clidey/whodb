/*
 * Copyright 2025 Clidey, Inc.
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
	"errors"
	"fmt"
	"strconv"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
)

func (p *RedisPlugin) UpdateStorageUnit(config *engine.PluginConfig, schema string, storageUnit string, values map[string]string, updatedColumns []string) (bool, error) {
	client, err := DB(config)
	if err != nil {
		log.WithError(err).WithField("storageUnit", storageUnit).Error("Failed to connect to Redis for storage unit update")
		return false, err
	}
	defer client.Close()

	ctx := context.Background()

	keyType, err := client.Type(ctx, storageUnit).Result()
	if err != nil {
		log.WithError(err).WithField("storageUnit", storageUnit).Error("Failed to get Redis key type for storage unit update")
		return false, err
	}

	switch keyType {
	case "string":
		if len(values) != 1 {
			err := errors.New("invalid number of fields for a string key")
			log.WithError(err).WithField("storageUnit", storageUnit).WithField("valueCount", len(values)).Error("Invalid number of fields for Redis string key update")
			return false, err
		}
		err := client.Set(ctx, storageUnit, values["value"], 0).Err()
		if err != nil {
			log.WithError(err).WithField("storageUnit", storageUnit).WithField("value", values["value"]).Error("Failed to update Redis string value")
			return false, err
		}
	case "hash":
		err := client.HSet(ctx, storageUnit, values["field"], values["value"]).Err()
		if err != nil {
			log.WithError(err).WithField("storageUnit", storageUnit).WithField("field", values["field"]).WithField("value", values["value"]).Error("Failed to update Redis hash field")
			return false, err
		}
	case "list":
		indexInt, err := strconv.ParseInt(values["index"], 10, 64)
		if err != nil {
			log.WithError(err).WithField("storageUnit", storageUnit).WithField("index", values["index"]).Error("Failed to parse list index for Redis update")
			return false, errors.New("unable to convert to int")
		}
		if err := client.LSet(ctx, storageUnit, indexInt, values["value"]).Err(); err != nil {
			log.WithError(err).WithField("storageUnit", storageUnit).WithField("index", indexInt).WithField("value", values["value"]).Error("Failed to update Redis list item")
			return false, errors.New("unable to update the list item")
		}
	case "set":
		return false, errors.ErrUnsupported
	default:
		err := fmt.Errorf("unsupported Redis data type: %s", keyType)
		return false, err
	}

	return true, nil
}
