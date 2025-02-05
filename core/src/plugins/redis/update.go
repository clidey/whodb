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
	"strconv"

	"github.com/clidey/whodb/core/src/engine"
)

func (p *RedisPlugin) UpdateStorageUnit(config *engine.PluginConfig, schema string, storageUnit string, values map[string]string) (bool, error) {
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
		if len(values) != 1 {
			return false, errors.New("invalid number of fields for a string key")
		}
		err := client.Set(ctx, storageUnit, values["value"], 0).Err()
		if err != nil {
			return false, err
		}
	case "hash":
		err := client.HSet(ctx, storageUnit, values["field"], values["value"]).Err()
		if err != nil {
			return false, err
		}
	case "list":
		indexInt, err := strconv.ParseInt(values["index"], 10, 64)
		if err != nil {
			return false, errors.New("unable to convert to int")
		}
		if err := client.LSet(ctx, storageUnit, indexInt, values["value"]).Err(); err != nil {
			return false, errors.New("unable to update the list item")
		}
	case "set":
		return false, errors.New("updating specific values in a set is not supported")
	default:
		return false, fmt.Errorf("unsupported Redis data type: %s", keyType)
	}

	return true, nil
}
